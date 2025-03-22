package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/grpc/pb"
)

// Client представляет gRPC клиент для отправки метрик.
type Client struct {
	address       string        // адрес gRPC сервера
	timeout       time.Duration // таймаут для запросов
	maxRetries    int           // максимальное количество повторных попыток
	conn          *grpc.ClientConn
	client        pb.MetricsServiceClient
	realIP        string // IP-адрес для заголовка X-Real-IP
	privateKey    string // приватный ключ для подписи запросов
	publicKeyPath string // путь к публичному ключу для TLS
}

// New создает новый gRPC клиент.
func New(
	address string,
	timeout int,
	maxRetries int,
	realIP,
	privateKey string,
	publicKeyPath string,
) (*Client, error) {
	// Создаем соединение с сервером
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Создаем клиентские перехватчики
	unaryInterceptors := []grpc.UnaryClientInterceptor{
		SigningInterceptor(privateKey), // Добавляем перехватчик для подписи запросов
	}

	streamInterceptors := []grpc.StreamClientInterceptor{
		StreamSigningInterceptor(
			privateKey,
		), // Добавляем перехватчик для подписи потоковых запросов
	}

	// Опции для подключения
	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts,
		grpc.WithChainUnaryInterceptor(unaryInterceptors...),
		grpc.WithChainStreamInterceptor(streamInterceptors...),
		grpc.WithBlock(),
	)

	// Если указан путь к публичному ключу, настраиваем TLS
	if publicKeyPath != "" {
		creds, err := loadTLSCredentials(publicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
		slog.Info(
			"TLS encryption enabled for gRPC client, credentials loaded from file",
			"path",
			publicKeyPath,
		)
	} else {
		// Устанавливаем соединение без TLS
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		slog.Warn("TLS encryption disabled for gRPC client, using insecure connection")
	}

	slog.Info("Connecting to gRPC server",
		"address", address,
		"timeout", timeout,
		"maxRetries", maxRetries,
		"realIP", realIP,
		"privateKey", privateKey,
		"publicKeyPath", publicKeyPath,
		"dialOpts", dialOpts,
	)

	// Устанавливаем соединение с сервером
	conn, err := grpcDialContext(ctx, address, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	// Создаем клиент
	client := pb.NewMetricsServiceClient(conn)

	// Логируем, что сжатие gRPC включено по умолчанию
	slog.Info("gRPC compression enabled by default")

	// Логируем состояние подписи запросов
	if privateKey != "" {
		slog.Info("gRPC request signing enabled")
	}

	return &Client{
		address:       address,
		timeout:       time.Duration(timeout) * time.Second,
		maxRetries:    maxRetries,
		conn:          conn,
		client:        client,
		realIP:        realIP,
		privateKey:    privateKey,
		publicKeyPath: publicKeyPath,
	}, nil
}

// Функция для подключения к серверу с контекстом, может быть заменена в тестах.
//
//nolint:gochecknoglobals // Переменная используется для мокинга в тестах
var grpcDialContext = grpc.DialContext

// loadTLSCredentials загружает TLS креды для защищенного соединения.
func loadTLSCredentials(publicKeyPath string) (credentials.TransportCredentials, error) {
	// Загружаем сертификат CA
	pemServerCA, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read server CA cert: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, errors.New("failed to add server CA's certificate")
	}

	// Создаем TLS конфигурацию с проверкой сервера
	config := &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS12,
	}

	return credentials.NewTLS(config), nil
}

// Close закрывает соединение с сервером.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// createContext создает контекст с метаданными и таймаутом.
func (c *Client) createContext(parent context.Context) (context.Context, context.CancelFunc) {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(parent, c.timeout)

	// Добавляем метаданные (X-Real-IP)
	if c.realIP != "" {
		md := metadata.New(map[string]string{
			"X-Real-IP": c.realIP,
		})
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	return ctx, cancel
}

// Возвращает опции вызова, всегда используем сжатие.
func (c *Client) getCallOptions() []grpc.CallOption {
	// Всегда используем сжатие gzip для gRPC запросов
	return []grpc.CallOption{
		grpc.UseCompressor(gzip.Name),
	}
}

// withRetry выполняет операцию с повторными попытками при ошибке.
func (c *Client) withRetry(
	ctx context.Context,
	operation func(context.Context, []grpc.CallOption) error,
) error {
	var err error
	retryIntervals := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	// Проверяем, не отменен ли контекст перед выполнением операции
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Получаем опции вызова
	callOpts := c.getCallOptions()

	// Выполняем операцию с учетом повторных попыток
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Проверяем, не отменен ли контекст перед каждой попыткой
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Создаем контекст для текущей попытки
		opCtx, cancel := c.createContext(ctx)

		// Выполняем операцию с опциями вызова
		err = operation(opCtx, callOpts)
		cancel()

		// Если операция успешна или контекст завершен, выходим из цикла
		if err == nil || ctx.Err() != nil {
			return err
		}

		// Если это последняя попытка, возвращаем ошибку
		if attempt == c.maxRetries {
			break
		}

		// Определяем интервал для следующей попытки
		retryInterval := time.Second
		if attempt < len(retryIntervals) {
			retryInterval = retryIntervals[attempt]
		}

		slog.Warn("gRPC request failed, retrying",
			"error", err,
			"attempt", attempt+1,
			"maxRetries", c.maxRetries,
			"retryIn", retryInterval)

		// Ждем перед следующей попыткой
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
			// Продолжаем выполнение
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", c.maxRetries+1, err)
}

// UpdateMetric отправляет метрику на сервер.
func (c *Client) UpdateMetric(ctx context.Context, metric *metrics.Metric) error {
	// Преобразуем метрику из доменной модели в protobuf
	protoMetric := metricToProto(metric)

	// Создаем запрос
	request := &pb.UpdateRequest{
		Metric: protoMetric,
	}

	// Отправляем запрос с повторными попытками
	return c.withRetry(ctx, func(opCtx context.Context, callOpts []grpc.CallOption) error {
		_, err := c.client.Update(opCtx, request, callOpts...)
		return err
	})
}

// UpdateBatch отправляет пакет метрик на сервер.
func (c *Client) UpdateBatch(ctx context.Context, metrics []*metrics.Metric) error {
	// Если метрик нет, ничего не делаем
	if len(metrics) == 0 {
		return nil
	}

	// Преобразуем метрики из доменной модели в protobuf
	protoMetrics := make([]*pb.Metric, 0, len(metrics))
	for _, m := range metrics {
		protoMetrics = append(protoMetrics, metricToProto(m))
	}

	// Создаем запрос
	request := &pb.UpdateBatchRequest{
		Metrics: protoMetrics,
	}

	// Отправляем запрос с повторными попытками
	return c.withRetry(ctx, func(opCtx context.Context, callOpts []grpc.CallOption) error {
		response, err := c.client.UpdateBatch(opCtx, request, callOpts...)
		if err != nil {
			return err
		}

		// Проверяем успешность операции
		if !response.GetSuccess() {
			return fmt.Errorf("server returned error: %s", response.GetError())
		}

		return nil
	})
}

// StreamMetrics отправляет метрики потоком на сервер.
func (c *Client) StreamMetrics(ctx context.Context, metrics []*metrics.Metric) error {
	// Если метрик нет, ничего не делаем
	if len(metrics) == 0 {
		return nil
	}

	slog.Info("Starting gRPC streaming of metrics", "count", len(metrics))

	// Отправляем запрос с повторными попытками
	return c.withRetry(ctx, func(opCtx context.Context, callOpts []grpc.CallOption) error {
		// Открываем поток с опциями сжатия
		stream, err := c.client.StreamMetrics(opCtx, callOpts...)
		if err != nil {
			slog.Error("Failed to open gRPC stream", "error", err)
			return fmt.Errorf("failed to open stream: %w", err)
		}

		// Отправляем метрики в поток
		sentCount := 0
		for _, m := range metrics {
			protoMetric := metricToProto(m)
			if sendErr := stream.Send(protoMetric); sendErr != nil {
				slog.Error("Failed to send metric in stream",
					"error", sendErr,
					"metric", m.Name,
					"sent", sentCount,
					"total", len(metrics))
				return fmt.Errorf("failed to send metric: %w", sendErr)
			}
			sentCount++
		}

		slog.Info("All metrics sent to gRPC stream", "count", sentCount)

		// Закрываем поток и получаем ответ
		response, err := stream.CloseAndRecv()
		if err != nil {
			slog.Error("Failed to close gRPC stream", "error", err)
			return fmt.Errorf("failed to close stream: %w", err)
		}

		// Проверяем успешность операции
		if !response.GetSuccess() {
			errMsg := fmt.Sprintf("server returned error: %s", response.GetError())
			slog.Error("Stream operation failed", "error", errMsg)
			return errors.New(errMsg)
		}

		slog.Info("Stream successfully completed")
		return nil
	})
}

// Ping проверяет соединение с сервером.
func (c *Client) Ping(ctx context.Context) error {
	return c.withRetry(ctx, func(opCtx context.Context, callOpts []grpc.CallOption) error {
		response, err := c.client.Ping(opCtx, &pb.PingRequest{}, callOpts...)
		if err != nil {
			return err
		}

		// Проверяем успешность операции
		if !response.GetSuccess() {
			return fmt.Errorf("server returned error: %s", response.GetError())
		}

		return nil
	})
}

// metricToProto преобразует метрику из доменной модели в protobuf.
func metricToProto(metric *metrics.Metric) *pb.Metric {
	if metric == nil {
		return nil
	}

	// Создаем базовую метрику только с обязательными полями
	protoMetric := &pb.Metric{
		Name: metric.Name,
	}

	// Устанавливаем специфичные для типа поля
	switch metric.MType {
	case metrics.TypeGauge:
		protoMetric.Type = pb.MetricType_GAUGE
		// Устанавливаем Value только если он не nil
		if metric.Value != nil {
			value := *metric.Value
			protoMetric.Value = &value
		}
		// Delta остается nil для gauge метрик
	case metrics.TypeCounter:
		protoMetric.Type = pb.MetricType_COUNTER
		// Устанавливаем Delta только если он не nil
		if metric.Delta != nil {
			delta := *metric.Delta
			protoMetric.Delta = &delta
		}
		// Value остается nil для counter метрик
	}

	return protoMetric
}
