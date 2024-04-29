package agent

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// Agent (HTTP-клиент) для сбора рантайм-метрик и их последующей отправки на сервер по протоколу HTTP.
type Agent struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerURL      string

	gauges       map[string]float64
	counters     map[string]int64
	mu           sync.Mutex
	wg           sync.WaitGroup
	client       *resty.Client
	pollTicker   *time.Ticker
	reportTicker *time.Ticker
}

// New создает новый экземпляр агента.
func New(url string, pollInterval time.Duration, reportInterval time.Duration) *Agent {
	return &Agent{
		ServerURL:      url,
		PollInterval:   pollInterval,
		ReportInterval: reportInterval,
		gauges:         make(map[string]float64),
		counters:       make(map[string]int64),
		client:         resty.New().SetHeader("Content-Type", "text/plain"),
		pollTicker:     time.NewTicker(pollInterval),
		reportTicker:   time.NewTicker(reportInterval),
	}
}

// Run запускает агента и его воркеры.
func (a *Agent) Run() {
	const goroutinesCount = 2
	a.wg.Add(goroutinesCount)

	// Запускаем воркеры агента.
	slog.Info("starting agent...",
		"server_url", a.ServerURL,
		"poll_interval", a.PollInterval,
		"report_interval", a.ReportInterval,
	)
	go a.runPolls()
	go a.runReports()

	a.wg.Wait()
}

// runPolls собирает сведения из системы в отдельной горутине.
func (a *Agent) runPolls() {
	defer a.wg.Done()
	for range a.pollTicker.C {
		a.mu.Lock()
		// Перезаписываем метрики свежими показаниями runtime.MemStats.
		a.gauges = a.CollectRuntimeMetrics()
		// Увеличиваем счетчик PollCount на 1.
		a.counters["PollCount"]++
		// Добавляем обновляемое рандомное значение по условию.
		a.gauges["RandomValue"] = generateRandomFloat64()

		// Логируем текущее значение счетчика PollCount в консоль для наглядности работы.
		slog.Info("collected metrics", "poll_count", a.counters["PollCount"])
		a.mu.Unlock()
	}
}

// Отправляет отчеты на сервер в отдельной горутине.
func (a *Agent) runReports() {
	defer a.wg.Done()
	for range a.reportTicker.C {
		a.sendAllMetrics()
	}
}

func (a *Agent) CollectRuntimeMetrics() map[string]float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mm := make(map[string]float64)
	mm["Alloc"] = float64(m.Alloc)
	mm["BuckHashSys"] = float64(m.BuckHashSys)
	mm["Frees"] = float64(m.Frees)
	mm["GCCPUFraction"] = m.GCCPUFraction
	mm["GCSys"] = float64(m.GCSys)
	mm["HeapAlloc"] = float64(m.HeapAlloc)
	mm["HeapIdle"] = float64(m.HeapIdle)
	mm["HeapInuse"] = float64(m.HeapInuse)
	mm["HeapObjects"] = float64(m.HeapObjects)
	mm["HeapReleased"] = float64(m.HeapReleased)
	mm["HeapSys"] = float64(m.HeapSys)
	mm["LastGC"] = float64(m.LastGC)
	mm["Lookups"] = float64(m.Lookups)
	mm["MCacheInuse"] = float64(m.MCacheInuse)
	mm["MCacheSys"] = float64(m.MCacheSys)
	mm["MSpanInuse"] = float64(m.MSpanInuse)
	mm["MSpanSys"] = float64(m.MSpanSys)
	mm["Mallocs"] = float64(m.Mallocs)
	mm["NextGC"] = float64(m.NextGC)
	mm["NumForcedGC"] = float64(m.NumForcedGC)
	mm["NumGC"] = float64(m.NumGC)
	mm["OtherSys"] = float64(m.OtherSys)
	mm["PauseTotalNs"] = float64(m.PauseTotalNs)
	mm["StackInuse"] = float64(m.StackInuse)
	mm["StackSys"] = float64(m.StackSys)
	mm["Sys"] = float64(m.Sys)
	mm["TotalAlloc"] = float64(m.TotalAlloc)

	return mm
}

// Отправка всех накопленных метрик.
func (a *Agent) sendAllMetrics() {
	gauges := make(map[string]float64)
	counters := make(map[string]int64)

	// Делаем копию метрик, чтобы данные не изменились во время отправки.
	a.mu.Lock()
	slog.Info("sending metrics", "poll_count", a.counters["PollCount"])
	for name, value := range a.gauges {
		gauges[name] = value
	}
	for name, value := range a.counters {
		counters[name] = value
	}
	// Обнуляем счетчик PollCount сразу как только подготовили его к отправке.
	// Из минусов: счетчик PollCount будет обнулен, даже если отправка метрик не удалась.
	// Другой вариант: обнулять счетчик PollCount только после успешной отправки метрик.
	a.counters["PollCount"] = 0
	slog.Info("reset poll count", "poll_count", 0)

	a.mu.Unlock()

	// Отправляем gauges.
	for name, value := range gauges {
		m := metrics.Metrics{
			ID:    name,
			MType: metrics.TypeGauge,
			//nolint:gosec // в Go 1.22, значение в цикле копируется (G601: Implicit memory aliasing in for loop.)
			Value: &value,
		}
		err := a.sendMetric(m)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to send gauge %s: %v", name, err))
			return
		}
	}
	// Отправляем counters.
	for name, value := range counters {
		m := metrics.Metrics{
			ID:    name,
			MType: metrics.TypeCounter,
			//nolint:gosec // в Go 1.22, значение в цикле копируется (G601: Implicit memory aliasing in for loop.)
			Delta: &value,
		}
		err := a.sendMetric(m)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to send counter %s: %v", name, err))
			return
		}
	}
}

// Отправка отдельной метрики на сервер.
func (a *Agent) sendMetric(metric metrics.Metrics) error {
	url := fmt.Sprintf("%s/update", a.ServerURL)
	slog.Info("sending metric", "url", url, "metric", metric.String())

	res, err := a.client.R().
		SetHeader("Content-Type", "application/json").
		// Go клиент автоматом добавляет заголовок "Accept-Encoding: gzip"
		SetBody(metric). // resty автоматом сериализует в json
		Post(url)

	if err != nil {
		return err
	}

	// Обрабатываем ответ сервера.
	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode())
	}

	return nil
}

func (a *Agent) DecrementCounter(name string, count int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.counters[name] -= count
}

func generateRandomFloat64() float64 {
	// Генерация случайного int64.
	randomInt, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return 0
	}

	// Преобразование int64 в float64 в диапазоне от 0 до 1.
	randomFloat := float64(randomInt.Int64()) / float64(math.MaxInt64)

	return randomFloat
}
