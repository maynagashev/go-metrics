package agent

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/json"
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
	PollInterval       time.Duration
	ReportInterval     time.Duration
	ServerURL          string
	SendCompressedData bool

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
		ServerURL:          url,
		PollInterval:       pollInterval,
		ReportInterval:     reportInterval,
		SendCompressedData: true, // согласно условиям задачи, отправка сжатых данных включена по умолчанию
		gauges:             make(map[string]float64),
		counters:           make(map[string]int64),
		client:             resty.New().SetHeader("Content-Type", "text/plain"),
		pollTicker:         time.NewTicker(pollInterval),
		reportTicker:       time.NewTicker(reportInterval),
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
	items := make([]*metrics.Metric, 0, len(a.gauges)+len(a.counters))

	// Делаем копию метрик, чтобы данные не изменились во время отправки.
	a.mu.Lock()
	slog.Info("sending metrics", "poll_count", a.counters["PollCount"])
	for name, value := range a.gauges {
		items = append(items, metrics.NewGauge(name, value))
	}
	for name, value := range a.counters {
		items = append(items, metrics.NewCounter(name, value))
	}
	// Обнуляем счетчик PollCount сразу как только подготовили его к отправке.
	// Из минусов: счетчик PollCount будет обнулен, даже если отправка метрик не удалась.
	// Другой вариант: обнулять счетчик PollCount только после успешной отправки метрик.
	a.counters["PollCount"] = 0
	slog.Info("reset poll count", "poll_count", 0)

	a.mu.Unlock()

	// Отправляем все метрики пачкой на новый маршрут /updates
	err := a.sendMetricsBatch(items)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to send metrics: %s", err), "metrics", items)
		return
	}
}

// Отправка пачки метрик на сервер.
func (a *Agent) sendMetricsBatch(items []*metrics.Metric) error {
	var err error
	url := fmt.Sprintf("%s/updates", a.ServerURL)
	slog.Info("sending metrics batch", "url", url, "metrics", items)

	// Создаем новый запрос.
	req := a.client.R()
	req.Debug = true // Включаем отладочный режим, чтобы видеть все детали запроса, в частности, использование сжатия.
	req.SetHeader("Content-Type", "application/json")

	// Преобразуем метрики в JSON.
	bytesBody, err := json.Marshal(items)
	if err != nil {
		return err
	}

	// Если включена сразу отправка сжатых данных, добавляем соответствующий заголовок.
	// Go клиент автоматом также добавляет заголовок "Accept-Encoding: gzip".
	if a.SendCompressedData {
		req.SetHeader("Content-Encoding", "gzip")
		bytesBody, err = compress(bytesBody)
		if err != nil {
			return err
		}
	}

	req.SetBody(bytesBody)

	slog.Debug("sendMetricsBatch", "url", url, "req", req)

	res, err := req.Post(url)
	if err != nil {
		return err
	}

	// Обрабатываем ответ сервера.
	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode())
	}

	return nil
}

// Сompress сжимает данные методом gzip (перед отправкой на сервер).
func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	// создаём переменную w — в неё будут записываться входящие данные,
	// которые будут сжиматься и сохраняться в bytes.Buffer
	w, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed init compress writer: %w", err)
	}

	// запись данных
	_, err = w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to compress temporary buffer: %w", err)
	}

	// обязательно нужно вызвать метод Close() — в противном случае часть данных
	// может не записаться в буфер b; если нужно выгрузить все упакованные данные
	// в какой-то момент сжатия, используйте метод Flush()
	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed compress data: %w", err)
	}
	// буфер b содержит сжатые данные
	return b.Bytes(), nil
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
