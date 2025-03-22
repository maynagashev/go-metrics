package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/maynagashev/go-metrics/internal/agent/grpc"
	"github.com/maynagashev/go-metrics/internal/agent/http"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// Client интерфейс для клиента метрик.
type Client interface {
	// UpdateMetric отправляет метрику на сервер.
	UpdateMetric(ctx context.Context, metric *metrics.Metric) error

	// UpdateBatch отправляет пакет метрик на сервер.
	UpdateBatch(ctx context.Context, metrics []*metrics.Metric) error

	// StreamMetrics отправляет метрики потоком на сервер (используется только для gRPC).
	StreamMetrics(ctx context.Context, metrics []*metrics.Metric) error

	// Ping проверяет соединение с сервером.
	Ping(ctx context.Context) error

	// Close закрывает клиент.
	Close() error
}

// Factory фабрика для создания клиентов.
type Factory struct {
	httpServerAddr string
	grpcServerAddr string
	grpcEnabled    bool
	grpcTimeout    int
	grpcRetry      int
	realIP         string
	privateKey     string // приватный ключ для подписи запросов
	cryptoKeyPath  string // путь к файлу с ключом (одинаковый и для шифрования, и для TLS)
}

// NewFactory создает новую фабрику клиентов.
func NewFactory(
	httpServerAddr, grpcServerAddr string,
	grpcEnabled bool,
	grpcTimeout, grpcRetry int,
	realIP, privateKey string,
	cryptoKeyPath string,
) *Factory {
	return &Factory{
		httpServerAddr: httpServerAddr,
		grpcServerAddr: grpcServerAddr,
		grpcEnabled:    grpcEnabled,
		grpcTimeout:    grpcTimeout,
		grpcRetry:      grpcRetry,
		realIP:         realIP,
		privateKey:     privateKey,
		cryptoKeyPath:  cryptoKeyPath,
	}
}

// CreateClient создает клиент в зависимости от конфигурации.
func (f *Factory) CreateClient() (Client, error) {
	if f.grpcEnabled {
		slog.Info("using gRPC client", "address", f.grpcServerAddr)
		return f.createGRPCClient()
	}

	slog.Info("using HTTP client", "address", f.httpServerAddr)
	return f.createHTTPClient()
}

// createGRPCClient создает gRPC клиент.
func (f *Factory) createGRPCClient() (Client, error) {
	client, err := grpc.New(
		f.grpcServerAddr,
		f.grpcTimeout,
		f.grpcRetry,
		f.realIP,
		f.privateKey,
		f.cryptoKeyPath, // используем тот же путь для TLS
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return client, nil
}

// createHTTPClient создает HTTP клиент.
func (f *Factory) createHTTPClient() (Client, error) {
	client := http.New(
		f.httpServerAddr,
		f.privateKey,
		f.cryptoKeyPath, // передаем путь к файлу ключа вместо самого ключа
		f.realIP,
	)

	return client, nil
}
