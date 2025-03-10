package agent

import (
	"log/slog"
	"sync"
	"time"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"

	"github.com/maynagashev/go-metrics/pkg/random"

	"github.com/go-resty/resty/v2"
)

// Количество попыток отправки запроса на сервер при возникновении ошибок.
const maxSendRetries = 3

// Agent представляет собой интерфейс для сбора и отправки метрик на сервер.
// Реализует функционал сбора runtime метрик и дополнительных системных метрик,
// а также их отправку на сервер с поддержкой подписи данных.
type Agent interface {
	// Run запускает процесс сбора и отправки метрик.
	Run()

	// IsRequestSigningEnabled возвращает true, если включена подпись запросов.
	IsRequestSigningEnabled() bool

	// ResetMetrics очищает все собранные метрики.
	ResetMetrics()

	// CollectRuntimeMetrics собирает метрики времени выполнения.
	CollectRuntimeMetrics()

	// CollectAdditionalMetrics собирает дополнительные системные метрики,
	// такие как использование памяти и CPU.
	CollectAdditionalMetrics()

	// GetMetrics возвращает список всех собранных метрик.
	GetMetrics() []*metrics.Metric
}

// agent конкретная реализация интерфейса Agent.
type agent struct {
	PollInterval       time.Duration
	ReportInterval     time.Duration
	ServerURL          string
	SendCompressedData bool
	PrivateKey         string
	RateLimit          int

	gauges       map[string]float64
	counters     map[string]int64
	mu           sync.Mutex
	wg           sync.WaitGroup
	client       *resty.Client
	pollTicker   *time.Ticker
	reportTicker *time.Ticker
	// Очередь задач на отправку метрик, с буфером в размере RateLimit.
	sendQueue chan Job
	// Очередь результатов выполнения задач, для обработки ошибок.
	resultQueue chan Result
}

// New создает новый экземпляр агента.
//
//nolint:gochecknoglobals // используется для подмены в тестах
var New = func(
	url string,
	pollInterval time.Duration,
	reportInterval time.Duration,
	privateKey string,
	rateLimit int,
) Agent {
	return &agent{
		ServerURL:          url,
		PollInterval:       pollInterval,
		ReportInterval:     reportInterval,
		SendCompressedData: true, // согласно условиям задачи, отправка сжатых данных включена по умолчанию
		PrivateKey:         privateKey,
		RateLimit:          rateLimit,
		gauges:             make(map[string]float64),
		counters:           make(map[string]int64),
		client:             resty.New().SetHeader("Content-Type", "text/plain"),
		pollTicker:         time.NewTicker(pollInterval),
		reportTicker:       time.NewTicker(reportInterval),
		sendQueue:          make(chan Job, rateLimit),
		resultQueue:        make(chan Result, rateLimit),
	}
}

// IsRequestSigningEnabled возвращает true, если задан приватный ключ и агент должен отправлять хэш на его основе.
func (a *agent) IsRequestSigningEnabled() bool {
	return a.PrivateKey != ""
}

// Run запускает агента и его воркеры.
func (a *agent) Run() {
	// Запускаем воркеры агента.
	slog.Info("starting agent...",
		"server_url", a.ServerURL,
		"poll_interval", a.PollInterval,
		"report_interval", a.ReportInterval,
		"send_compressed_data", a.SendCompressedData,
		"private_key", a.PrivateKey,
		"send_hash", a.IsRequestSigningEnabled(),
		"rate_limit", a.RateLimit,
	)
	// Горутина для сбора метрик (с интервалом PollInterval).
	go a.runPolls()
	// Горутина для добавления задач в очередь на отправку, с интервалом ReportInterval.
	go a.runReports()

	// Запуск worker pool для отправки метрик.
	for i := range a.RateLimit {
		a.wg.Add(1)
		go a.worker(i)
	}

	// Запуск коллектора результатов
	a.wg.Add(1)
	go a.collector()

	a.wg.Wait()
}

// runPolls собирает сведения из системы в отдельной горутине.
func (a *agent) runPolls() {
	a.wg.Add(1)
	defer a.wg.Done()
	for range a.pollTicker.C {
		a.mu.Lock()
		// Перезаписываем метрики свежими показаниями
		a.ResetMetrics()
		a.CollectRuntimeMetrics()
		a.CollectAdditionalMetrics()

		// Увеличиваем счетчик PollCount на 1.
		a.counters["PollCount"]++
		// Добавляем обновляемое рандомное значение по условию.
		a.gauges["RandomValue"] = random.GenerateRandomFloat64()

		// Логируем текущее значение счетчика PollCount в консоль для наглядности работы.
		slog.Info("collected metrics", "poll_count", a.counters["PollCount"])
		a.mu.Unlock()
	}
}

// Создает задачи по отправке метрик в очереди задач на отправку.
func (a *agent) runReports() {
	a.wg.Add(1)
	defer a.wg.Done()
	for range a.reportTicker.C {
		a.sendQueue <- Job{Metrics: a.GetMetrics()}
	}
}

// GetMetrics считывает текущие метрики из агента.
func (a *agent) GetMetrics() []*metrics.Metric {
	items := make([]*metrics.Metric, 0, len(a.gauges)+len(a.counters))

	// Делаем копию метрик, чтобы данные не изменились во время отправки.
	a.mu.Lock()
	slog.Info("read metrics for job", "poll_count", a.counters["PollCount"])
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
	return items
}
