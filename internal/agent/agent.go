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

	gauges       map[string]interface{}
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
		gauges:         make(map[string]interface{}),
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

func (a *Agent) runReports() {
	defer a.wg.Done()
	for range a.reportTicker.C {
		a.sendAllMetrics()
	}
}

func (a *Agent) CollectRuntimeMetrics() map[string]interface{} {
	mm := make(map[string]interface{})

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mm["Alloc"] = m.Alloc
	mm["BuckHashSys"] = m.BuckHashSys
	mm["Frees"] = m.Frees
	mm["GCCPUFraction"] = m.GCCPUFraction
	mm["GCSys"] = m.GCSys
	mm["HeapAlloc"] = m.HeapAlloc
	mm["HeapIdle"] = m.HeapIdle
	mm["HeapInuse"] = m.HeapInuse
	mm["HeapObjects"] = m.HeapObjects
	mm["HeapReleased"] = m.HeapReleased
	mm["HeapSys"] = m.HeapSys
	mm["LastGC"] = m.LastGC
	mm["Lookups"] = m.Lookups
	mm["MCacheInuse"] = m.MCacheInuse
	mm["MCacheSys"] = m.MCacheSys
	mm["MSpanInuse"] = m.MSpanInuse
	mm["MSpanSys"] = m.MSpanSys
	mm["Mallocs"] = m.Mallocs
	mm["NextGC"] = m.NextGC
	mm["NumForcedGC"] = m.NumForcedGC
	mm["NumGC"] = m.NumGC
	mm["OtherSys"] = m.OtherSys
	mm["PauseTotalNs"] = m.PauseTotalNs
	mm["StackInuse"] = m.StackInuse
	mm["StackSys"] = m.StackSys
	mm["Sys"] = m.Sys
	mm["TotalAlloc"] = m.TotalAlloc

	return mm
}

func (a *Agent) sendAllMetrics() {
	gauges := make(map[string]interface{})
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
		err := a.sendMetric(metrics.TypeGauge, name, value)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to send gauge %s: %v", name, err))
			return
		}
	}
	// Отправляем counters.
	for name, value := range counters {
		err := a.sendMetric(metrics.TypeCounter, name, value)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to send counter %s: %v", name, err))
			return
		}
	}
}

func (a *Agent) sendMetric(metricType metrics.MetricType, name string, value interface{}) error {
	url := fmt.Sprintf("%s/update/%s/%s/%v", a.ServerURL, metricType, name, value)
	slog.Info("sending metrics", "url", url)

	res, err := a.client.R().Post(url)
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
