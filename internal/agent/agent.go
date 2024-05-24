package agent

import (
	"log/slog"
	"sync"
	"time"

	"github.com/maynagashev/go-metrics/pkg/random"

	"github.com/go-resty/resty/v2"
)

// Количество попыток отправки запроса на сервер при возникновении ошибок.
const maxSendRetries = 3

// Agent (HTTP-клиент) для сбора рантайм-метрик и их последующей отправки на сервер по протоколу HTTP.
type Agent struct {
	PollInterval       time.Duration
	ReportInterval     time.Duration
	ServerURL          string
	SendCompressedData bool
	PrivateKey         string

	gauges       map[string]float64
	counters     map[string]int64
	mu           sync.Mutex
	wg           sync.WaitGroup
	client       *resty.Client
	pollTicker   *time.Ticker
	reportTicker *time.Ticker
}

// New создает новый экземпляр агента.
func New(url string, pollInterval time.Duration, reportInterval time.Duration, privateKey string) *Agent {
	return &Agent{
		ServerURL:          url,
		PollInterval:       pollInterval,
		ReportInterval:     reportInterval,
		SendCompressedData: true, // согласно условиям задачи, отправка сжатых данных включена по умолчанию
		PrivateKey:         privateKey,
		gauges:             make(map[string]float64),
		counters:           make(map[string]int64),
		client:             resty.New().SetHeader("Content-Type", "text/plain"),
		pollTicker:         time.NewTicker(pollInterval),
		reportTicker:       time.NewTicker(reportInterval),
	}
}

// IsRequestSigningEnabled возвращает true, если задан приватный ключ и агент должен отправлять хэш на его основе.
func (a *Agent) IsRequestSigningEnabled() bool {
	return a.PrivateKey != ""
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
		"send_compressed_data", a.SendCompressedData,
		"private_key", a.PrivateKey,
		"send_hash", a.IsRequestSigningEnabled(),
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
		a.gauges["RandomValue"] = random.GenerateRandomFloat64()

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
