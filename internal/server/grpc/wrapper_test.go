package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/grpc"
	"github.com/maynagashev/go-metrics/internal/server/storage"
)

// mockStorage - мок для storage.Repository.
type mockStorage struct {
	mock.Mock
}

func (m *mockStorage) UpdateMetric(ctx context.Context, metric metrics.Metric) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

func (m *mockStorage) UpdateMetrics(ctx context.Context, metrics []metrics.Metric) error {
	args := m.Called(ctx, metrics)
	return args.Error(0)
}

func (m *mockStorage) GetGauge(ctx context.Context, name string) (storage.Gauge, bool) {
	args := m.Called(ctx, name)
	val, ok := args.Get(0).(storage.Gauge)
	if !ok && args.Get(0) != nil {
		panic("GetGauge: первый аргумент не является storage.Gauge")
	}
	return val, args.Bool(1)
}

func (m *mockStorage) GetCounter(ctx context.Context, name string) (storage.Counter, bool) {
	args := m.Called(ctx, name)
	val, ok := args.Get(0).(storage.Counter)
	if !ok && args.Get(0) != nil {
		panic("GetCounter: первый аргумент не является storage.Counter")
	}
	return val, args.Bool(1)
}

func (m *mockStorage) Count(ctx context.Context) int {
	args := m.Called(ctx)
	return args.Int(0)
}

func (m *mockStorage) GetMetric(
	ctx context.Context,
	mType metrics.MetricType,
	name string,
) (metrics.Metric, bool) {
	args := m.Called(ctx, mType, name)
	var metric metrics.Metric
	if val := args.Get(0); val != nil {
		var ok bool
		metric, ok = val.(metrics.Metric)
		if !ok {
			panic("GetMetric: первый аргумент не является metrics.Metric")
		}
	}
	return metric, args.Bool(1)
}

func (m *mockStorage) GetMetrics(ctx context.Context) []metrics.Metric {
	args := m.Called(ctx)
	var metricsResult []metrics.Metric
	if val := args.Get(0); val != nil {
		var ok bool
		metricsResult, ok = val.([]metrics.Metric)
		if !ok {
			panic("GetMetrics: первый аргумент не является []metrics.Metric")
		}
	}
	return metricsResult
}

func (m *mockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

// TestNewServer проверяет создание нового экземпляра ServerWrapper.
func TestNewServer(t *testing.T) {
	// Создаем зависимости
	log, err := zap.NewDevelopment()
	require.NoError(t, err)

	cfg := &app.Config{}
	repo := &mockStorage{}

	// Вызываем тестируемую функцию
	wrapper := grpc.NewServer(log, cfg, repo)

	// Проверяем результат
	assert.NotNil(t, wrapper)
	assert.Same(t, log, wrapper.GetLogger())
	assert.Same(t, cfg, wrapper.GetConfig())
}

// TestStartDisabled проверяет запуск gRPC сервера когда он отключен.
func TestStartDisabled(t *testing.T) {
	// Создаем конфигурацию, где gRPC отключен
	cfg := &app.Config{
		GRPC: app.GRPCConfig{
			Enabled: false,
		},
	}

	// Создаем логгер
	log, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Создаем wrapper
	wrapper := grpc.NewServer(log, cfg, &mockStorage{})

	// Вызываем тестируемую функцию
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = wrapper.Start(ctx)

	// Проверяем результат
	require.NoError(t, err)
}

// TestStopWithNilServer проверяет корректную обработку nil сервера при Stop.
func TestStopWithNilServer(t *testing.T) {
	// Создаем логгер
	log, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Создаем wrapper с отключенным GRPC
	cfg := &app.Config{
		GRPC: app.GRPCConfig{
			Enabled: false,
		},
	}

	wrapper := grpc.NewServer(log, cfg, &mockStorage{})

	// Проверяем, что вызов Stop не вызывает панику
	assert.NotPanics(t, func() {
		wrapper.Stop()
	})
}

// TestServerWrapperIntegration это интеграционный тест, который проверяет, что
// ServerWrapper правильно создает и обращается к серверу.
func TestServerWrapperIntegration(t *testing.T) {
	// Пропускаем интеграционные тесты при коротком тестировании
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Создаем конфигурацию
	cfg := &app.Config{
		GRPC: app.GRPCConfig{
			Enabled: true,
			Addr:    "localhost:0", // Используем порт 0 для автоматического выбора свободного порта
		},
	}

	// Создаем логгер
	log, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Создаем хранилище
	repo := &mockStorage{}

	// Создаем wrapper
	wrapper := grpc.NewServer(log, cfg, repo)
	assert.NotNil(t, wrapper)

	// Запускаем сервер
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем сервер и проверяем, что ошибок нет
	err = wrapper.Start(ctx)
	require.NoError(t, err)

	// Даем серверу время на запуск
	time.Sleep(50 * time.Millisecond)

	// Останавливаем сервер и проверяем, что нет паники
	assert.NotPanics(t, func() {
		wrapper.Stop()
	})
}
