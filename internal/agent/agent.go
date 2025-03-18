package agent

import (
	"context"
	"crypto/rsa"
	"log/slog"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/pkg/random"
)

// Количество попыток отправки запроса на сервер при возникновении ошибок.
const maxSendRetries = 3

// Agent представляет собой интерфейс для сбора и отправки метрик на сервер.
// Реализует функционал сбора runtime метрик и дополнительных системных метрик,
// а также их отправку на сервер с поддержкой подписи данных.
type Agent interface {
	// Run запускает процесс сбора и отправки метрик.
	Run(ctx context.Context)

	// IsRequestSigningEnabled возвращает true, если включена подпись запросов.
	IsRequestSigningEnabled() bool

	// IsEncryptionEnabled возвращает true, если включено шифрование.
	IsEncryptionEnabled() bool

	// ResetMetrics очищает все собранные метрики.
	ResetMetrics()

	// CollectRuntimeMetrics собирает метрики времени выполнения.
	CollectRuntimeMetrics()

	// CollectAdditionalMetrics собирает дополнительные системные метрики,
	// такие как использование памяти и CPU.
	CollectAdditionalMetrics()

	// GetMetrics возвращает список всех собранных метрик.
	GetMetrics() []*metrics.Metric

	// Shutdown корректно завершает работу агента, дожидаясь завершения всех задач.
	Shutdown()
}

// agent конкретная реализация интерфейса Agent.
type agent struct {
	PollInterval       time.Duration
	ReportInterval     time.Duration
	ServerURL          string
	SendCompressedData bool
	PrivateKey         string
	RateLimit          int
	PublicKey          *rsa.PublicKey // Public key for encryption

	// Конфигурация gRPC
	GRPCEnabled bool   // Использовать gRPC вместо HTTP
	GRPCAddress string // Адрес и порт gRPC сервера
	GRPCTimeout int    // Таймаут для gRPC запросов в секундах
	GRPCRetry   int    // Количество повторных попыток при ошибке gRPC запроса

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
	// Канал для сигнала остановки
	stopCh chan struct{}
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
	publicKey *rsa.PublicKey,
	realIP string,
	grpcEnabled bool, // флаг использования gRPC вместо HTTP
	grpcAddress string, // адрес и порт gRPC сервера
	grpcTimeout int, // таймаут для gRPC запросов в секундах
	grpcRetry int, // количество повторных попыток при ошибке gRPC запроса
) Agent {
	return &agent{
		ServerURL:          url,
		PollInterval:       pollInterval,
		ReportInterval:     reportInterval,
		SendCompressedData: true, // согласно условиям задачи, отправка сжатых данных включена по умолчанию
		PrivateKey:         privateKey,
		RateLimit:          rateLimit,
		PublicKey:          publicKey,
		GRPCEnabled:        grpcEnabled,
		GRPCAddress:        grpcAddress,
		GRPCTimeout:        grpcTimeout,
		GRPCRetry:          grpcRetry,
		gauges:             make(map[string]float64),
		counters:           make(map[string]int64),
		client:             initHTTPClient(realIP),
		pollTicker:         time.NewTicker(pollInterval),
		reportTicker:       time.NewTicker(reportInterval),
		sendQueue:          make(chan Job, rateLimit),
		resultQueue:        make(chan Result, rateLimit),
		stopCh:             make(chan struct{}),
	}
}

// IsRequestSigningEnabled возвращает true, если задан приватный ключ и агент должен отправлять хэш на его основе.
func (a *agent) IsRequestSigningEnabled() bool {
	return a.PrivateKey != ""
}

// IsEncryptionEnabled возвращает true, если задан публичный ключ и агент должен шифровать данные.
func (a *agent) IsEncryptionEnabled() bool {
	return a.PublicKey != nil
}

// Run запускает агента и его воркеры.
func (a *agent) Run(ctx context.Context) {
	// Запускаем воркеры агента.
	slog.Info("======= AGENT STARTING =======",
		"server_url", a.ServerURL,
		"poll_interval", a.PollInterval,
		"report_interval", a.ReportInterval,
		"send_compressed_data", a.SendCompressedData,
		"private_key", a.PrivateKey,
		"send_hash", a.IsRequestSigningEnabled(),
		"encryption_enabled", a.IsEncryptionEnabled(),
		"rate_limit", a.RateLimit,
		"grpc_enabled", a.GRPCEnabled, // использовать ли gRPC
		"grpc_address", a.GRPCAddress, // адрес gRPC сервера
		"grpc_timeout", a.GRPCTimeout, // таймаут gRPC запросов
		"grpc_retry", a.GRPCRetry, // число повторных попыток
	)
	// Горутина для сбора метрик (с интервалом PollInterval).
	go a.runPolls(ctx)
	// Горутина для добавления задач в очередь на отправку, с интервалом ReportInterval.
	go a.runReports(ctx)

	// Запуск worker pool для отправки метрик.
	for i := range a.RateLimit {
		a.wg.Add(1)
		go a.worker(i)
	}

	// Запуск коллектора результатов
	a.wg.Add(1)
	go a.collector()

	slog.Info("======= AGENT STARTED =======")

	// Ожидаем завершения контекста
	<-ctx.Done()
	slog.Info("Context done, initiating graceful shutdown")
	a.Shutdown()
}

// Shutdown корректно завершает работу агента.
func (a *agent) Shutdown() {
	slog.Info("======= GRACEFUL SHUTDOWN STARTED =======")

	// Останавливаем тикеры
	a.pollTicker.Stop()
	a.reportTicker.Stop()
	slog.Info("Tickers stopped")

	// Сигнализируем о завершении работы
	close(a.stopCh)
	slog.Info("Stop channel closed")

	// Отправляем последние собранные метрики
	metrics := a.GetMetrics()
	if len(metrics) > 0 {
		slog.Info("Sending final metrics batch before shutdown", "count", len(metrics))

		// Пытаемся отправить метрики напрямую
		err := a.sendMetrics(metrics, -1) // -1 означает, что это финальная отправка
		if err != nil {
			slog.Error("Failed to send final metrics directly", "error", err)

			// Если не удалось отправить напрямую, пробуем через очередь
			select {
			case a.sendQueue <- Job{Metrics: metrics}:
				slog.Info("Final metrics queued for sending")
			default:
				slog.Error("Failed to queue final metrics, queue might be full")
			}
		} else {
			slog.Info("Final metrics sent successfully")
		}
	} else {
		slog.Info("No metrics to send before shutdown")
	}

	// Закрываем каналы после отправки всех метрик
	slog.Info("Closing send queue")
	close(a.sendQueue)

	// Ожидаем завершения всех горутин
	slog.Info("Waiting for all goroutines to finish")
	a.wg.Wait()
	slog.Info("======= GRACEFUL SHUTDOWN COMPLETED =======")
}

// runPolls собирает сведения из системы в отдельной горутине.
func (a *agent) runPolls(ctx context.Context) {
	a.wg.Add(1)
	defer a.wg.Done()

	slog.Info("Poll routine started",
		"poll_interval", a.PollInterval,
		"metrics_storage_size", len(a.gauges)+len(a.counters))

	for {
		select {
		case <-a.pollTicker.C:
			pollStart := time.Now()
			slog.Debug("Starting metrics collection cycle")

			a.mu.Lock()
			// Перезаписываем метрики свежими показаниями
			a.ResetMetrics()
			a.CollectRuntimeMetrics()
			a.CollectAdditionalMetrics()

			// Увеличиваем счетчик PollCount на 1.
			a.counters["PollCount"]++
			// Добавляем обновляемое рандомное значение по условию.
			a.gauges["RandomValue"] = random.GenerateRandomFloat64()

			metricsCount := len(a.gauges) + len(a.counters)
			pollDuration := time.Since(pollStart)

			// Логируем текущее значение счетчика PollCount в консоль для наглядности работы.
			slog.Info("Metrics collection completed",
				"poll_count", a.counters["PollCount"],
				"metrics_count", metricsCount,
				"duration_ms", pollDuration.Milliseconds())
			a.mu.Unlock()
		case <-ctx.Done():
			slog.Info("Stopping polls due to context cancellation")
			return
		case <-a.stopCh:
			slog.Info("Stopping polls due to agent shutdown")
			return
		}
	}
}

// Создает задачи по отправке метрик в очереди задач на отправку.
func (a *agent) runReports(ctx context.Context) {
	a.wg.Add(1)
	defer a.wg.Done()

	slog.Info("Report routine started",
		"report_interval", a.ReportInterval,
		"server_url", a.ServerURL)

	for {
		select {
		case <-a.reportTicker.C:
			reportStart := time.Now()
			slog.Debug("Starting metrics report cycle")

			metrics := a.GetMetrics()
			metricsCount := len(metrics)

			if metricsCount > 0 {
				slog.Info("Sending metrics to queue",
					"metrics_count", metricsCount,
					"queue_size", len(a.sendQueue))
				a.sendQueue <- Job{Metrics: metrics}
			} else {
				slog.Debug("No metrics to send in this report cycle")
			}

			reportDuration := time.Since(reportStart)
			slog.Debug("Report cycle completed",
				"duration_ms", reportDuration.Milliseconds())
		case <-ctx.Done():
			slog.Info("Stopping reports due to context cancellation")
			return
		case <-a.stopCh:
			slog.Info("Stopping reports due to agent shutdown")
			return
		}
	}
}

// GetMetrics считывает текущие метрики из агента.
func (a *agent) GetMetrics() []*metrics.Metric {
	startTime := time.Now()
	slog.Debug("Starting metrics preparation for sending")

	items := make([]*metrics.Metric, 0, len(a.gauges)+len(a.counters))

	// Делаем копию метрик, чтобы данные не изменились во время отправки.
	a.mu.Lock()

	gaugesCount := len(a.gauges)
	countersCount := len(a.counters)
	pollCount := a.counters["PollCount"]

	// Проверяем, получен ли сигнал завершения
	select {
	case <-a.stopCh:
		slog.Info(
			"Shutdown signal received while preparing metrics - ensuring final metrics are sent",
			"gauges_count",
			gaugesCount,
			"counters_count",
			countersCount,
			"poll_count",
			pollCount,
		)
	default:
		slog.Info("Preparing metrics for sending",
			"gauges_count", gaugesCount,
			"counters_count", countersCount,
			"poll_count", pollCount)
	}

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
	slog.Debug("Reset poll count to zero")

	a.mu.Unlock()

	duration := time.Since(startTime)
	slog.Debug("Metrics preparation completed",
		"metrics_count", len(items),
		"duration_ms", duration.Milliseconds())

	return items
}

// initHTTPClient создает и настраивает HTTP-клиент с перехватчиком для установки заголовка X-Real-IP.
func initHTTPClient(realIP string) *resty.Client {
	client := resty.New().SetHeader("Content-Type", "text/plain")

	// Добавляем перехватчик для установки заголовка X-Real-IP
	client.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
		// Если указан явный IP-адрес, используем его
		if realIP != "" {
			req.SetHeader("X-Real-IP", realIP)
			slog.Debug("set X-Real-IP header (explicit)", "ip", realIP)
			return nil
		}

		// Иначе получаем исходящий IP-адрес автоматически
		hostIP, err := GetOutboundIP()
		if err == nil {
			// Устанавливаем заголовок X-Real-IP
			req.SetHeader("X-Real-IP", hostIP.String())
			slog.Debug("set X-Real-IP header (auto-detected)", "ip", hostIP.String())
		} else {
			slog.Error("failed to set X-Real-IP header", "error", err)
		}
		return nil
	})

	return client
}
