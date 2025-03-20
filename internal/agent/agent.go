package agent

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"

	"github.com/maynagashev/go-metrics/internal/agent/client"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/pkg/random"
)

// Константы для конвертации единиц измерения.
const (
	BytesInKB = 1024
	BytesInMB = BytesInKB * 1024
	BytesInGB = BytesInMB * 1024
)

// Константы для работы агента.
const (
	// Минимальная длина строки для маскирования.
	minMaskLength = 6
	// Таймаут для запросов по умолчанию.
	defaultRequestTimeout = 10 * time.Second
	// Количество видимых символов в начале и конце маскированной строки.
	visibleCharsCount = 2
	// Количество концов строки (начало и конец) для отображения видимых символов.
	stringEndsCount = 2
)

// Job структура для задания воркерам.
type Job struct {
	Metrics []*metrics.Metric
}

// Result структура для результата выполнения задания.
type Result struct {
	Job   Job
	Error error
}

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
	RateLimit          int            // Количество воркеров для сбора и отправки метрик
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
	client       client.Client // Клиент для отправки метрик
	pollTicker   *time.Ticker
	reportTicker *time.Ticker
	// Очередь задач на отправку метрик, с буфером в размере RateLimit.
	sendQueue chan Job
	// Очередь результатов выполнения задач, для обработки ошибок.
	resultQueue chan Result
	// Канал для сигнала остановки
	stopCh chan struct{}
	// Флаг шифрования: true, если путь к ключу задан
	encryptionEnabled bool
}

// New создает новый экземпляр агента.
//
//nolint:gochecknoglobals // используется для подмены в тестах
var New = func(
	url string,
	pollInterval time.Duration,
	reportInterval time.Duration,
	privateKey string, // путь к файлу с приватным ключом для подписи запросов к серверу
	rateLimit int,
	realIP string,
	grpcEnabled bool, // флаг использования gRPC вместо HTTP
	grpcAddress string, // адрес и порт gRPC сервера
	grpcTimeout int, // таймаут для gRPC запросов в секундах
	grpcRetry int, // количество повторных попыток при ошибке gRPC запроса
	cryptoKeyPath string, // путь к файлу с ключом
) (Agent, error) {
	// Создаем фабрику клиентов
	factory := client.NewFactory(
		url,
		grpcAddress,
		grpcEnabled,
		grpcTimeout,
		grpcRetry,
		realIP,
		privateKey,
		cryptoKeyPath,
	)

	// Создаем клиент через фабрику
	client, err := factory.CreateClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create client with factory: %w", err)
	}

	// Флаг шифрования: true, если путь к ключу задан
	encryptionEnabled := cryptoKeyPath != ""

	return &agent{
		ServerURL:          url,
		PollInterval:       pollInterval,
		ReportInterval:     reportInterval,
		SendCompressedData: true, // согласно условиям задачи, отправка сжатых данных включена по умолчанию
		PrivateKey:         privateKey,
		RateLimit:          rateLimit,
		PublicKey:          nil, // больше не используется, используем путь к файлу
		GRPCEnabled:        grpcEnabled,
		GRPCAddress:        grpcAddress,
		GRPCTimeout:        grpcTimeout,
		GRPCRetry:          grpcRetry,
		gauges:             make(map[string]float64),
		counters:           make(map[string]int64),
		client:             client,
		pollTicker:         time.NewTicker(pollInterval),
		reportTicker:       time.NewTicker(reportInterval),
		sendQueue:          make(chan Job, rateLimit),
		resultQueue:        make(chan Result, rateLimit),
		stopCh:             make(chan struct{}),
		encryptionEnabled:  encryptionEnabled,
	}, nil
}

// IsRequestSigningEnabled возвращает true, если задан приватный ключ и агент должен отправлять хэш на его основе.
func (a *agent) IsRequestSigningEnabled() bool {
	return a.PrivateKey != ""
}

// IsEncryptionEnabled возвращает true, если включено шифрование.
func (a *agent) IsEncryptionEnabled() bool {
	return a.encryptionEnabled
}

// runInWaitGroup запускает функцию в отдельной горутине и управляет WaitGroup.
func (a *agent) runInWaitGroup(f func()) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		f()
	}()
}

// Run запускает агента и его воркеры.
func (a *agent) Run(ctx context.Context) {
	slog.Info("Starting agent",
		"server_url", a.ServerURL,
		"poll_interval", a.PollInterval,
		"report_interval", a.ReportInterval,
		"private_key", maskString(a.PrivateKey),
		"rate_limit", a.RateLimit,
		"grpc_enabled", a.GRPCEnabled,
		"grpc_address", a.GRPCAddress)

	// Запускаем горутину для сбора метрик по таймеру pollTicker
	a.runInWaitGroup(func() {
		a.runPolls(ctx)
	})

	// Запускаем горутину для создания задач по отправке метрик по таймеру reportTicker
	a.runInWaitGroup(func() {
		a.runReports(ctx)
	})

	// Запускаем воркеры для отправки метрик из очереди задач на отправку
	for i := range a.RateLimit {
		workerID := i
		a.runInWaitGroup(func() {
			a.runWorker(workerID)()
		})
	}

	// Запускаем горутину для сбора результатов из очереди результатов выполнения задач
	a.runInWaitGroup(func() {
		a.runCollector()()
	})

	// Ждем сигнала завершения из контекста
	<-ctx.Done()

	// После получения сигнала завершения, корректно завершаем работу агента
	a.Shutdown()
}

// maskString маскирует строку, заменяя все символы кроме первых и последних на '*'.
func maskString(s string) string {
	if len(s) < minMaskLength {
		return "<empty>"
	}
	// Оставляем по visibleCharsCount символов в начале и конце, остальное заменяем звездочками
	maskLen := len(s) - (visibleCharsCount * stringEndsCount)
	return s[:visibleCharsCount] + strings.Repeat("*", maskLen) + s[len(s)-visibleCharsCount:]
}

// runWorker запускает обработчик задач на отправку метрик.
func (a *agent) runWorker(id int) func() {
	return func() {
		slog.Info("Starting sender worker", "workerID", id)
		defer slog.Info("Sender worker shutting down", "workerID", id)

		for {
			select {
			case <-a.stopCh:
				return
			case job, ok := <-a.sendQueue:
				if !ok {
					// Канал закрыт, завершаем работу
					return
				}

				// Контекст с таймаутом для отправки метрик
				ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
				err := a.sendMetrics(ctx, job.Metrics, id)
				cancel()

				// Отправляем результат в канал для обработки
				a.resultQueue <- Result{Job: job, Error: err}
			}
		}
	}
}

// sendMetrics отправляет метрики на сервер.
func (a *agent) sendMetrics(ctx context.Context, items []*metrics.Metric, workerID int) error {
	slog.Info("Sending metrics batch", "workerID", workerID, "metrics_count", len(items))

	// Проверяем, что клиент не nil перед использованием
	if a.client == nil {
		return errors.New("client is nil")
	}

	// Используем потоковую передачу для gRPC, если она включена
	if a.GRPCEnabled {
		slog.Debug("Using gRPC streaming for metrics", "workerID", workerID)
		return a.client.StreamMetrics(ctx, items)
	}

	// Используем обычное пакетное обновление для HTTP или если gRPC не поддерживает потоки
	return a.client.UpdateBatch(ctx, items)
}

// runCollector возвращает функцию, которая запускает сборщик результатов.
func (a *agent) runCollector() func() {
	return func() {
		slog.Info("Collector started")
		for {
			select {
			case result, ok := <-a.resultQueue:
				if !ok {
					slog.Info("Result queue closed, collector exiting")
					return
				}
				if result.Error != nil {
					slog.Error("Failed to send metrics", "error", result.Error)
				} else {
					slog.Info("Metrics sent successfully", "count", len(result.Job.Metrics))
				}
			case <-a.stopCh:
				slog.Info("Stop signal received, collector exiting")
				return
			}
		}
	}
}

// GetMetrics возвращает список всех собранных метрик.
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

// Shutdown корректно завершает работу агента, дожидаясь завершения всех задач.
func (a *agent) Shutdown() {
	slog.Info("======= GRACEFUL SHUTDOWN STARTED =======")

	// Останавливаем тикеры
	if a.pollTicker != nil {
		a.pollTicker.Stop()
	}
	if a.reportTicker != nil {
		a.reportTicker.Stop()
	}
	slog.Info("Tickers stopped")

	// Сигнализируем о завершении работы
	close(a.stopCh)
	slog.Info("Stop channel closed")

	// Отправляем последние собранные метрики
	metrics := a.GetMetrics()
	if len(metrics) > 0 && a.client != nil {
		slog.Info("Sending final metrics batch before shutdown", "count", len(metrics))

		// Пытаемся отправить метрики напрямую
		ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
		err := a.client.UpdateBatch(ctx, metrics)
		cancel()

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

	// Закрываем клиент
	if a.client != nil {
		if err := a.client.Close(); err != nil {
			slog.Error("Failed to close client", "error", err)
		}
	}

	// Закрываем каналы после отправки всех метрик
	if a.sendQueue != nil {
		slog.Info("Closing send queue")
		close(a.sendQueue)
	}

	// Ожидаем завершения всех горутин
	slog.Info("Waiting for all goroutines to finish")
	a.wg.Wait()
	slog.Info("======= GRACEFUL SHUTDOWN COMPLETED =======")
}

// runPolls собирает сведения из системы в отдельной горутине по таймеру.
func (a *agent) runPolls(ctx context.Context) {
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

// runReports создает по таймеру задачи по отправке метрик в очереди задач на отправку.
func (a *agent) runReports(ctx context.Context) {
	slog.Info("Report routine started",
		"report_interval", a.ReportInterval,
		"server_url", a.ServerURL,
		"grpc_enabled", a.GRPCEnabled,
		"grpc_address", a.GRPCAddress)

	// Добавляем отладочную информацию о клиенте
	if a.client != nil {
		slog.Info("Client initialized", "client_type", fmt.Sprintf("%T", a.client))
	} else {
		slog.Error("Client is nil")
	}

	for {
		select {
		case <-a.reportTicker.C:
			reportStart := time.Now()
			slog.Info("Starting metrics report cycle")

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
			slog.Info("Report cycle completed",
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

// ResetMetrics очищает все метрики агента, вызываем перед сбором новых метрик.
func (a *agent) ResetMetrics() {
	slog.Debug("Resetting metrics before collection")
	a.gauges = make(map[string]float64)
	a.counters = make(map[string]int64)
}

// CollectRuntimeMetrics собирает метрики времени выполнения.
func (a *agent) CollectRuntimeMetrics() {
	// Проверяем сигнал остановки перед сбором метрик
	select {
	case <-a.stopCh:
		slog.Info("Shutdown signal received, skipping runtime metrics collection")
		return
	default:
		// Продолжаем выполнение
	}

	slog.Debug("Starting runtime metrics collection")
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	a.gauges["Alloc"] = float64(m.Alloc)
	a.gauges["BuckHashSys"] = float64(m.BuckHashSys)
	a.gauges["Frees"] = float64(m.Frees)
	a.gauges["GCCPUFraction"] = m.GCCPUFraction
	a.gauges["GCSys"] = float64(m.GCSys)
	a.gauges["HeapAlloc"] = float64(m.HeapAlloc)
	a.gauges["HeapIdle"] = float64(m.HeapIdle)
	a.gauges["HeapInuse"] = float64(m.HeapInuse)
	a.gauges["HeapObjects"] = float64(m.HeapObjects)
	a.gauges["HeapReleased"] = float64(m.HeapReleased)
	a.gauges["HeapSys"] = float64(m.HeapSys)
	a.gauges["LastGC"] = float64(m.LastGC)
	a.gauges["Lookups"] = float64(m.Lookups)
	a.gauges["MCacheInuse"] = float64(m.MCacheInuse)
	a.gauges["MCacheSys"] = float64(m.MCacheSys)
	a.gauges["MSpanInuse"] = float64(m.MSpanInuse)
	a.gauges["MSpanSys"] = float64(m.MSpanSys)
	a.gauges["Mallocs"] = float64(m.Mallocs)
	a.gauges["NextGC"] = float64(m.NextGC)
	a.gauges["NumForcedGC"] = float64(m.NumForcedGC)
	a.gauges["NumGC"] = float64(m.NumGC)
	a.gauges["OtherSys"] = float64(m.OtherSys)
	a.gauges["PauseTotalNs"] = float64(m.PauseTotalNs)
	a.gauges["StackInuse"] = float64(m.StackInuse)
	a.gauges["StackSys"] = float64(m.StackSys)
	a.gauges["Sys"] = float64(m.Sys)
	a.gauges["TotalAlloc"] = float64(m.TotalAlloc)

	slog.Debug("Runtime metrics collection completed",
		"metrics_count", len(a.gauges),
		"heap_alloc_mb", float64(m.HeapAlloc)/BytesInMB,
		"sys_mb", float64(m.Sys)/BytesInMB)
}

// CollectAdditionalMetrics собирает дополнительные метрики системы.
func (a *agent) CollectAdditionalMetrics() {
	// Проверяем сигнал остановки перед сбором метрик
	select {
	case <-a.stopCh:
		slog.Info("Shutdown signal received, skipping additional metrics collection")
		return
	default:
		// Продолжаем выполнение
	}

	slog.Debug("Starting additional system metrics collection")

	v, err := mem.VirtualMemory()
	if err != nil {
		slog.Error("Failed to collect virtual memory metrics", "error", err)
		return
	}
	a.gauges["TotalMemory"] = float64(v.Total)
	a.gauges["FreeMemory"] = float64(v.Free)

	slog.Debug("Memory metrics collected",
		"total_memory_gb", float64(v.Total)/BytesInGB,
		"free_memory_gb", float64(v.Free)/BytesInGB,
		"used_percent", v.UsedPercent)

	c, err := cpu.Percent(0, true)
	if err != nil {
		slog.Error("Failed to collect CPU metrics", "error", err)
		return
	}

	cpuCount := 0
	for i, percent := range c {
		a.gauges[fmt.Sprintf("CPUutilization%d", i+1)] = percent
		cpuCount++
	}

	slog.Debug("CPU metrics collection completed", "cpu_count", cpuCount)
}
