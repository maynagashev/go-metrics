//nolint:testpackage // используется для тестирования внутреннего API
package grpc

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/grpc/pb"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"
)

// MockRepository - мок для интерфейса storage.Repository.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRepository) Count(ctx context.Context) int {
	args := m.Called(ctx)
	return args.Int(0)
}

func (m *MockRepository) GetMetrics(ctx context.Context) []metrics.Metric {
	args := m.Called(ctx)
	result, _ := args.Get(0).([]metrics.Metric)
	return result
}

func (m *MockRepository) GetMetric(
	ctx context.Context,
	mType metrics.MetricType,
	name string,
) (metrics.Metric, bool) {
	args := m.Called(ctx, mType, name)
	result, _ := args.Get(0).(metrics.Metric)
	return result, args.Bool(1)
}

func (m *MockRepository) GetCounter(ctx context.Context, name string) (storage.Counter, bool) {
	args := m.Called(ctx, name)
	val, ok := args.Get(0).(int64)
	if !ok {
		return 0, args.Bool(1)
	}
	return storage.Counter(val), args.Bool(1)
}

func (m *MockRepository) GetGauge(ctx context.Context, name string) (storage.Gauge, bool) {
	args := m.Called(ctx, name)
	val, ok := args.Get(0).(float64)
	if !ok {
		return 0, args.Bool(1)
	}
	return storage.Gauge(val), args.Bool(1)
}

func (m *MockRepository) UpdateMetric(ctx context.Context, metric metrics.Metric) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

func (m *MockRepository) UpdateMetrics(ctx context.Context, metrics []metrics.Metric) error {
	args := m.Called(ctx, metrics)
	return args.Error(0)
}

// MockConfig реализует интерфейс app.Config для тестирования.
type MockConfig struct {
	mock.Mock
}

func (m *MockConfig) IsDatabaseEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfig) IsRestoreEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfig) IsStoreEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfig) IsSyncStore() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfig) GetStoreInterval() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockConfig) GetStorePath() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockConfig) IsRequestSigningEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfig) IsEncryptionEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfig) IsTrustedSubnetEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfig) IsGRPCEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockConfig) GetCryptoKeyPath() string {
	args := m.Called()
	return args.String(0)
}

// TestNewMetricsService проверяет создание нового MetricsService.
func TestNewMetricsService(t *testing.T) {
	// Создаем фиктивные зависимости
	logger := zap.NewNop()
	config := &app.Config{}
	repo := &MockRepository{}

	// Создаем сервис
	service := NewMetricsService(logger, config, repo)

	// Проверяем, что сервис создан и его поля инициализированы
	assert.NotNil(t, service)
	assert.Equal(t, logger, service.log)
	assert.Equal(t, config, service.cfg)
	assert.Equal(t, repo, service.storage)
}

// TestProtoToMetric тестирует преобразование из protobuf в доменную модель.
func TestProtoToMetric(t *testing.T) {
	service := &MetricsService{
		log: zap.NewNop(),
	}

	t.Run("nil metric", func(t *testing.T) {
		// Проверяем обработку nil метрики
		metric, err := service.protoToMetric(nil)
		require.Error(t, err)
		assert.Nil(t, metric)
	})

	t.Run("gauge metric", func(t *testing.T) {
		// Создаем gauge метрику в protobuf формате
		value := 42.5
		protoMetric := &pb.Metric{
			Name:  "test_gauge",
			Type:  pb.MetricType_GAUGE,
			Value: &value,
		}

		// Преобразуем в доменную модель
		metric, err := service.protoToMetric(protoMetric)

		// Проверяем результат
		require.NoError(t, err)
		assert.NotNil(t, metric)
		assert.Equal(t, "test_gauge", metric.Name)
		assert.Equal(t, metrics.TypeGauge, metric.MType)
		assert.NotNil(t, metric.Value)
		assert.InDelta(t, value, *metric.Value, 0.0001)
		assert.Nil(t, metric.Delta)
	})

	t.Run("counter metric", func(t *testing.T) {
		// Создаем counter метрику в protobuf формате
		delta := int64(100)
		protoMetric := &pb.Metric{
			Name:  "test_counter",
			Type:  pb.MetricType_COUNTER,
			Delta: &delta,
		}

		// Преобразуем в доменную модель
		metric, err := service.protoToMetric(protoMetric)

		// Проверяем результат
		require.NoError(t, err)
		assert.NotNil(t, metric)
		assert.Equal(t, "test_counter", metric.Name)
		assert.Equal(t, metrics.TypeCounter, metric.MType)
		assert.NotNil(t, metric.Delta)
		assert.Equal(t, delta, *metric.Delta)
		assert.Nil(t, metric.Value)
	})

	t.Run("gauge without value", func(t *testing.T) {
		// Создаем gauge метрику без значения
		protoMetric := &pb.Metric{
			Name: "invalid_gauge",
			Type: pb.MetricType_GAUGE,
			// Value: nil,
		}

		// Преобразуем в доменную модель
		metric, err := service.protoToMetric(protoMetric)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, metric)
	})

	t.Run("counter without delta", func(t *testing.T) {
		// Создаем counter метрику без дельты
		protoMetric := &pb.Metric{
			Name: "invalid_counter",
			Type: pb.MetricType_COUNTER,
			// Delta: nil,
		}

		// Преобразуем в доменную модель
		metric, err := service.protoToMetric(protoMetric)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, metric)
	})

	t.Run("unknown metric type", func(t *testing.T) {
		// Создаем метрику с неизвестным типом
		protoMetric := &pb.Metric{
			Name: "unknown_type",
			Type: 999, // Неизвестный тип
		}

		// Преобразуем в доменную модель
		metric, err := service.protoToMetric(protoMetric)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, metric)
	})
}

// TestMetricToProto тестирует преобразование из доменной модели в protobuf.
func TestMetricToProto(t *testing.T) {
	service := &MetricsService{
		log: zap.NewNop(),
	}

	t.Run("gauge metric", func(t *testing.T) {
		// Создаем gauge метрику в доменной модели
		value := 42.5
		metric := &metrics.Metric{
			Name:  "test_gauge",
			MType: metrics.TypeGauge,
			Value: &value,
		}

		// Преобразуем в protobuf
		protoMetric := service.metricToProto(metric)

		// Проверяем результат
		assert.NotNil(t, protoMetric)
		assert.Equal(t, "test_gauge", protoMetric.GetName())
		assert.Equal(t, pb.MetricType_GAUGE, protoMetric.GetType())
		assert.NotNil(t, protoMetric.GetValue())
		assert.InDelta(t, value, protoMetric.GetValue(), 0.0001)
		assert.Zero(t, protoMetric.GetDelta())
	})

	t.Run("counter metric", func(t *testing.T) {
		// Создаем counter метрику в доменной модели
		delta := int64(100)
		metric := &metrics.Metric{
			Name:  "test_counter",
			MType: metrics.TypeCounter,
			Delta: &delta,
		}

		// Преобразуем в protobuf
		protoMetric := service.metricToProto(metric)

		// Проверяем результат
		assert.NotNil(t, protoMetric)
		assert.Equal(t, "test_counter", protoMetric.GetName())
		assert.Equal(t, pb.MetricType_COUNTER, protoMetric.GetType())
		assert.NotNil(t, protoMetric.GetDelta())
		assert.Equal(t, delta, protoMetric.GetDelta())
		assert.Zero(t, protoMetric.GetValue())
	})
}

// TestUpdate проверяет метод Update для обновления одной метрики.
func TestUpdate(t *testing.T) {
	// Создаем контекст
	ctx := context.Background()

	t.Run("valid gauge metric", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Настраиваем ожидания для репозитория
		value := 42.5
		expectedMetric := metrics.Metric{
			Name:  "test_gauge",
			MType: metrics.TypeGauge,
			Value: &value,
		}
		repo.On("UpdateMetric", ctx, expectedMetric).Return(nil)

		// Создаем сервис и запрос
		service := NewMetricsService(logger, config, repo)
		req := &pb.UpdateRequest{
			Metric: &pb.Metric{
				Name:  "test_gauge",
				Type:  pb.MetricType_GAUGE,
				Value: &value,
			},
		}

		// Выполняем запрос
		resp, err := service.Update(ctx, req)

		// Проверяем результат
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "test_gauge", resp.GetMetric().GetName())
		assert.Equal(t, pb.MetricType_GAUGE, resp.GetMetric().GetType())
		assert.NotNil(t, resp.GetMetric().GetValue())
		assert.InDelta(t, value, resp.GetMetric().GetValue(), 0.0001)

		// Проверяем, что все ожидания репозитория выполнены
		repo.AssertExpectations(t)
	})

	t.Run("valid counter metric", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Настраиваем ожидания для репозитория
		delta := int64(100)
		expectedMetric := metrics.Metric{
			Name:  "test_counter",
			MType: metrics.TypeCounter,
			Delta: &delta,
		}
		repo.On("UpdateMetric", ctx, expectedMetric).Return(nil)

		// Создаем сервис и запрос
		service := NewMetricsService(logger, config, repo)
		req := &pb.UpdateRequest{
			Metric: &pb.Metric{
				Name:  "test_counter",
				Type:  pb.MetricType_COUNTER,
				Delta: &delta,
			},
		}

		// Выполняем запрос
		resp, err := service.Update(ctx, req)

		// Проверяем результат
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "test_counter", resp.GetMetric().GetName())
		assert.Equal(t, pb.MetricType_COUNTER, resp.GetMetric().GetType())
		assert.NotNil(t, resp.GetMetric().GetDelta())
		assert.Equal(t, delta, resp.GetMetric().GetDelta())

		// Проверяем, что все ожидания репозитория выполнены
		repo.AssertExpectations(t)
	})

	t.Run("nil request", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Выполняем запрос с nil
		resp, err := service.Update(ctx, nil)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
	})

	t.Run("nil metric in request", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем сервис и запрос с пустой метрикой
		service := NewMetricsService(logger, config, repo)
		req := &pb.UpdateRequest{
			Metric: nil,
		}

		// Выполняем запрос
		resp, err := service.Update(ctx, req)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
	})

	t.Run("invalid metric type", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем сервис и запрос с невалидным типом метрики
		service := NewMetricsService(logger, config, repo)
		req := &pb.UpdateRequest{
			Metric: &pb.Metric{
				Name: "invalid_type",
				Type: 999, // Неизвестный тип
			},
		}

		// Выполняем запрос
		resp, err := service.Update(ctx, req)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
	})

	t.Run("repository error", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Настраиваем ожидания для репозитория - возвращаем ошибку
		value := 42.5
		expectedMetric := metrics.Metric{
			Name:  "test_gauge",
			MType: metrics.TypeGauge,
			Value: &value,
		}
		repoError := errors.New("repository error")
		repo.On("UpdateMetric", ctx, expectedMetric).Return(repoError)

		// Создаем сервис и запрос
		service := NewMetricsService(logger, config, repo)
		req := &pb.UpdateRequest{
			Metric: &pb.Metric{
				Name:  "test_gauge",
				Type:  pb.MetricType_GAUGE,
				Value: &value,
			},
		}

		// Выполняем запрос
		resp, err := service.Update(ctx, req)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())

		// Проверяем, что все ожидания репозитория выполнены
		repo.AssertExpectations(t)
	})
}

// TestUpdateBatch проверяет метод UpdateBatch для пакетного обновления метрик.
func TestUpdateBatch(t *testing.T) {
	// Создаем контекст
	ctx := context.Background()

	t.Run("valid metrics batch", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Подготавливаем данные для теста
		gaugeValue := 42.5
		counterDelta := int64(100)

		// Ожидаемые метрики для обновления
		expectedMetrics := []metrics.Metric{
			{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
				Value: &gaugeValue,
			},
			{
				Name:  "test_counter",
				MType: metrics.TypeCounter,
				Delta: &counterDelta,
			},
		}

		// Настраиваем ожидания для репозитория
		repo.On("UpdateMetrics", ctx, expectedMetrics).Return(nil)

		// Создаем сервис и запрос
		service := NewMetricsService(logger, config, repo)
		req := &pb.UpdateBatchRequest{
			Metrics: []*pb.Metric{
				{
					Name:  "test_gauge",
					Type:  pb.MetricType_GAUGE,
					Value: &gaugeValue,
				},
				{
					Name:  "test_counter",
					Type:  pb.MetricType_COUNTER,
					Delta: &counterDelta,
				},
			},
		}

		// Выполняем запрос
		resp, err := service.UpdateBatch(ctx, req)

		// Проверяем результат
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.GetSuccess())

		// Проверяем, что все ожидания репозитория выполнены
		repo.AssertExpectations(t)
	})

	t.Run("nil request", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Выполняем запрос с nil
		resp, err := service.UpdateBatch(ctx, nil)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
	})

	t.Run("empty metrics array", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем сервис и запрос с пустым массивом метрик
		service := NewMetricsService(logger, config, repo)
		req := &pb.UpdateBatchRequest{
			Metrics: []*pb.Metric{},
		}

		// Выполняем запрос
		resp, err := service.UpdateBatch(ctx, req)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
	})

	t.Run("all invalid metrics", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем сервис и запрос с невалидными метриками
		service := NewMetricsService(logger, config, repo)
		req := &pb.UpdateBatchRequest{
			Metrics: []*pb.Metric{
				{
					Name: "invalid_gauge",
					Type: pb.MetricType_GAUGE,
					// Отсутствует значение
				},
				{
					Name: "invalid_counter",
					Type: pb.MetricType_COUNTER,
					// Отсутствует дельта
				},
			},
		}

		// Выполняем запрос
		resp, err := service.UpdateBatch(ctx, req)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
	})

	t.Run("repository error", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Подготавливаем данные для теста
		gaugeValue := 42.5
		counterDelta := int64(100)

		// Ожидаемые метрики для обновления
		expectedMetrics := []metrics.Metric{
			{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
				Value: &gaugeValue,
			},
			{
				Name:  "test_counter",
				MType: metrics.TypeCounter,
				Delta: &counterDelta,
			},
		}

		// Настраиваем ожидания для репозитория - возвращаем ошибку
		repoError := errors.New("repository error")
		repo.On("UpdateMetrics", ctx, expectedMetrics).Return(repoError)

		// Создаем сервис и запрос
		service := NewMetricsService(logger, config, repo)
		req := &pb.UpdateBatchRequest{
			Metrics: []*pb.Metric{
				{
					Name:  "test_gauge",
					Type:  pb.MetricType_GAUGE,
					Value: &gaugeValue,
				},
				{
					Name:  "test_counter",
					Type:  pb.MetricType_COUNTER,
					Delta: &counterDelta,
				},
			},
		}

		// Выполняем запрос
		resp, err := service.UpdateBatch(ctx, req)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())

		// Проверяем, что все ожидания репозитория выполнены
		repo.AssertExpectations(t)
	})
}

// TestGetValue проверяет метод GetValue для получения значения метрики.
func TestGetValue(t *testing.T) {
	// Создаем контекст
	ctx := context.Background()

	t.Run("get existing gauge", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Настраиваем ожидания для репозитория
		value := 42.5
		mockMetric := metrics.Metric{
			Name:  "test_gauge",
			MType: metrics.TypeGauge,
			Value: &value,
		}
		repo.On("GetMetric", ctx, metrics.TypeGauge, "test_gauge").Return(mockMetric, true)

		// Создаем сервис и запрос
		service := NewMetricsService(logger, config, repo)
		req := &pb.GetValueRequest{
			Name: "test_gauge",
			Type: pb.MetricType_GAUGE,
		}

		// Выполняем запрос
		resp, err := service.GetValue(ctx, req)

		// Проверяем результат
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "test_gauge", resp.GetMetric().GetName())
		assert.Equal(t, pb.MetricType_GAUGE, resp.GetMetric().GetType())
		assert.NotNil(t, resp.GetMetric().GetValue())
		assert.InDelta(t, value, resp.GetMetric().GetValue(), 0.0001)

		// Проверяем, что все ожидания репозитория выполнены
		repo.AssertExpectations(t)
	})

	t.Run("get existing counter", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Настраиваем ожидания для репозитория
		delta := int64(100)
		mockMetric := metrics.Metric{
			Name:  "test_counter",
			MType: metrics.TypeCounter,
			Delta: &delta,
		}
		repo.On("GetMetric", ctx, metrics.TypeCounter, "test_counter").Return(mockMetric, true)

		// Создаем сервис и запрос
		service := NewMetricsService(logger, config, repo)
		req := &pb.GetValueRequest{
			Name: "test_counter",
			Type: pb.MetricType_COUNTER,
		}

		// Выполняем запрос
		resp, err := service.GetValue(ctx, req)

		// Проверяем результат
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "test_counter", resp.GetMetric().GetName())
		assert.Equal(t, pb.MetricType_COUNTER, resp.GetMetric().GetType())
		assert.NotNil(t, resp.GetMetric().GetDelta())
		assert.Equal(t, delta, resp.GetMetric().GetDelta())

		// Проверяем, что все ожидания репозитория выполнены
		repo.AssertExpectations(t)
	})

	t.Run("nil request", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Выполняем запрос с nil
		resp, err := service.GetValue(ctx, nil)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
	})

	t.Run("empty metric name", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем сервис и запрос с пустым именем метрики
		service := NewMetricsService(logger, config, repo)
		req := &pb.GetValueRequest{
			Name: "",
			Type: pb.MetricType_GAUGE,
		}

		// Выполняем запрос
		resp, err := service.GetValue(ctx, req)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
	})

	t.Run("invalid metric type", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем сервис и запрос с невалидным типом метрики
		service := NewMetricsService(logger, config, repo)
		req := &pb.GetValueRequest{
			Name: "test_metric",
			Type: 999, // Неизвестный тип
		}

		// Выполняем запрос
		resp, err := service.GetValue(ctx, req)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, statusErr.Code())
	})

	t.Run("metric not found", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Настраиваем ожидания для репозитория - метрика не найдена
		repo.On("GetMetric", ctx, metrics.TypeGauge, "non_existent").Return(metrics.Metric{}, false)

		// Создаем сервис и запрос
		service := NewMetricsService(logger, config, repo)
		req := &pb.GetValueRequest{
			Name: "non_existent",
			Type: pb.MetricType_GAUGE,
		}

		// Выполняем запрос
		resp, err := service.GetValue(ctx, req)

		// Проверяем результат
		require.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, statusErr.Code())

		// Проверяем, что все ожидания репозитория выполнены
		repo.AssertExpectations(t)
	})
}

// TestPing проверяет метод Ping для проверки соединения с БД.
func TestPing(t *testing.T) {
	// Создаем контекст
	ctx := context.Background()

	t.Run("database enabled", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{
			Database: app.DatabaseConfig{
				DSN: "test-dsn", // Устанавливаем DSN, чтобы IsDatabaseEnabled вернул true
			},
		}
		repo := &MockRepository{}

		// Настраиваем ожидания
		repo.On("Count", ctx).Return(5) // Просто какое-то число

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Выполняем запрос
		resp, err := service.Ping(ctx, &pb.PingRequest{})

		// Проверяем результат
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.GetSuccess())
		assert.Empty(t, resp.GetError())

		// Проверяем, что все ожидания выполнены
		repo.AssertExpectations(t)
	})

	t.Run("database disabled", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{
			Database: app.DatabaseConfig{
				DSN: "", // Пустой DSN, чтобы IsDatabaseEnabled вернул false
			},
		}
		repo := &MockRepository{}

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Выполняем запрос
		resp, err := service.Ping(ctx, &pb.PingRequest{})

		// Проверяем результат
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.GetSuccess())
		assert.Equal(t, "database not configured", resp.GetError())

		// Проверяем, что repo не должен вызываться
		repo.AssertNotCalled(t, "Count")
	})
}

// MockStreamServer реализует интерфейс pb.MetricsService_StreamMetricsServer для тестирования.
type MockStreamServer struct {
	mock.Mock
	ctx      context.Context
	recvData []*pb.Metric
	recvIdx  int
	sentResp *pb.UpdateBatchResponse
	grpc.ServerStream
}

func NewMockStreamServer(ctx context.Context, metrics []*pb.Metric) *MockStreamServer {
	return &MockStreamServer{
		ctx:      ctx,
		recvData: metrics,
	}
}

func (m *MockStreamServer) Context() context.Context {
	return m.ctx
}

func (m *MockStreamServer) Recv() (*pb.Metric, error) {
	if m.recvIdx >= len(m.recvData) {
		return nil, io.EOF
	}
	metric := m.recvData[m.recvIdx]
	m.recvIdx++
	return metric, nil
}

func (m *MockStreamServer) SendAndClose(resp *pb.UpdateBatchResponse) error {
	args := m.Called(resp)
	m.sentResp = resp
	return args.Error(0)
}

// TestSaveMetricsBuffer проверяет метод saveMetricsBuffer.
func TestSaveMetricsBuffer(t *testing.T) {
	// Создаем контекст
	ctx := context.Background()

	t.Run("empty buffer", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Вызываем метод с пустым буфером
		err := service.saveMetricsBuffer(ctx, []metrics.Metric{})

		// Проверяем результат
		require.NoError(t, err)

		// Проверяем, что repo не вызывался
		repo.AssertNotCalled(t, "UpdateMetrics")
	})

	t.Run("non-empty buffer", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Подготавливаем данные для теста
		gaugeValue := 42.5
		counterDelta := int64(100)

		// Создаем буфер метрик
		metricsBuffer := []metrics.Metric{
			{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
				Value: &gaugeValue,
			},
			{
				Name:  "test_counter",
				MType: metrics.TypeCounter,
				Delta: &counterDelta,
			},
		}

		// Настраиваем ожидания для репозитория
		repo.On("UpdateMetrics", ctx, metricsBuffer).Return(nil)

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Вызываем метод
		err := service.saveMetricsBuffer(ctx, metricsBuffer)

		// Проверяем результат
		require.NoError(t, err)

		// Проверяем, что все ожидания репозитория выполнены
		repo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Подготавливаем данные для теста
		gaugeValue := 42.5

		// Создаем буфер метрик
		metricsBuffer := []metrics.Metric{
			{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
				Value: &gaugeValue,
			},
		}

		// Настраиваем ожидания для репозитория - возвращаем ошибку
		repoError := errors.New("repository error")
		repo.On("UpdateMetrics", ctx, metricsBuffer).Return(repoError)

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Вызываем метод
		err := service.saveMetricsBuffer(ctx, metricsBuffer)

		// Проверяем результат
		require.Error(t, err)
		assert.Equal(t, repoError, err)

		// Проверяем, что все ожидания репозитория выполнены
		repo.AssertExpectations(t)
	})
}

// MockErrorStreamServer - специальный мок для тестирования ошибок приема данных.
type MockErrorStreamServer struct {
	mock.Mock
	grpc.ServerStream
}

func (m *MockErrorStreamServer) Context() context.Context {
	args := m.Called()
	val, ok := args.Get(0).(context.Context)
	if !ok {
		return context.Background()
	}
	return val
}

func (m *MockErrorStreamServer) Recv() (*pb.Metric, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	val, ok := args.Get(0).(*pb.Metric)
	if !ok {
		return nil, args.Error(1)
	}
	return val, args.Error(1)
}

func (m *MockErrorStreamServer) SendAndClose(resp *pb.UpdateBatchResponse) error {
	args := m.Called(resp)
	return args.Error(0)
}

func (m *MockErrorStreamServer) SendMsg(msg interface{}) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *MockErrorStreamServer) RecvMsg(msg interface{}) error {
	args := m.Called(msg)
	return args.Error(0)
}

// TestStreamMetrics проверяет метод StreamMetrics.
func TestStreamMetrics(t *testing.T) {
	// Создаем контекст
	ctx := context.Background()

	t.Run("empty stream", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Создаем пустой мок-стрим
		stream := NewMockStreamServer(ctx, []*pb.Metric{})

		// Настраиваем ожидания
		stream.On("SendAndClose", &pb.UpdateBatchResponse{Success: true}).Return(nil)

		// Вызываем метод
		err := service.StreamMetrics(stream)

		// Проверяем результат
		require.NoError(t, err)

		// Проверяем, что все ожидания выполнены
		stream.AssertExpectations(t)
		repo.AssertNotCalled(t, "UpdateMetrics")
	})

	t.Run("valid metrics stream", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Подготавливаем данные для теста
		gaugeValue := 42.5
		counterDelta := int64(100)

		// Создаем мок-стрим с метриками
		stream := NewMockStreamServer(ctx, []*pb.Metric{
			{
				Name:  "test_gauge",
				Type:  pb.MetricType_GAUGE,
				Value: &gaugeValue,
			},
			{
				Name:  "test_counter",
				Type:  pb.MetricType_COUNTER,
				Delta: &counterDelta,
			},
		})

		// Ожидаемые метрики для обновления
		expectedMetrics := []metrics.Metric{
			{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
				Value: &gaugeValue,
			},
			{
				Name:  "test_counter",
				MType: metrics.TypeCounter,
				Delta: &counterDelta,
			},
		}

		// Настраиваем ожидания
		repo.On("UpdateMetrics", ctx, expectedMetrics).Return(nil)
		stream.On("SendAndClose", &pb.UpdateBatchResponse{Success: true}).Return(nil)

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Вызываем метод
		err := service.StreamMetrics(stream)

		// Проверяем результат
		require.NoError(t, err)

		// Проверяем, что все ожидания выполнены
		stream.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("stream with invalid metrics", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем мок-стрим с метриками, где одна валидная, а другая нет
		gaugeValue := 42.5
		stream := NewMockStreamServer(ctx, []*pb.Metric{
			{
				Name:  "test_gauge",
				Type:  pb.MetricType_GAUGE,
				Value: &gaugeValue,
			},
			{
				Name: "invalid_counter",
				Type: pb.MetricType_COUNTER,
				// Отсутствует дельта
			},
		})

		// Ожидаемые метрики для обновления (только валидные)
		expectedMetrics := []metrics.Metric{
			{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
				Value: &gaugeValue,
			},
		}

		// Настраиваем ожидания
		repo.On("UpdateMetrics", ctx, expectedMetrics).Return(nil)
		stream.On("SendAndClose", &pb.UpdateBatchResponse{Success: true}).Return(nil)

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Вызываем метод
		err := service.StreamMetrics(stream)

		// Проверяем результат
		require.NoError(t, err)

		// Проверяем, что все ожидания выполнены
		stream.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Подготавливаем данные для теста
		gaugeValue := 42.5

		// Создаем мок-стрим с метриками
		stream := NewMockStreamServer(ctx, []*pb.Metric{
			{
				Name:  "test_gauge",
				Type:  pb.MetricType_GAUGE,
				Value: &gaugeValue,
			},
		})

		// Ожидаемые метрики для обновления
		expectedMetrics := []metrics.Metric{
			{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
				Value: &gaugeValue,
			},
		}

		// Настраиваем ожидания для репозитория - возвращаем ошибку
		repoError := errors.New("repository error")
		repo.On("UpdateMetrics", ctx, expectedMetrics).Return(repoError)

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Вызываем метод
		err := service.StreamMetrics(stream)

		// Проверяем результат
		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())

		// Проверяем, что ожидания репозитория выполнены
		repo.AssertExpectations(t)
		// Закрытие потока не должно произойти из-за ошибки
		stream.AssertNotCalled(t, "SendAndClose")
	})

	t.Run("stream receive error", func(t *testing.T) {
		// Создаем зависимости
		logger := zap.NewNop()
		config := &app.Config{}
		repo := &MockRepository{}

		// Создаем специальный мок для тестирования ошибки
		stream := new(MockErrorStreamServer)
		stream.On("Context").Return(ctx)
		stream.On("Recv").Return(nil, errors.New("stream error"))
		// НЕ настраиваем ожидание для SendAndClose, так как мы ожидаем, что метод вернет ошибку до вызова SendAndClose

		// Создаем сервис
		service := NewMetricsService(logger, config, repo)

		// Вызываем метод
		err := service.StreamMetrics(stream)

		// Проверяем результат
		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, statusErr.Code())

		// Проверяем, что ожидания выполнены
		stream.AssertExpectations(t)
		// Репозиторий не должен вызываться
		repo.AssertNotCalled(t, "UpdateMetrics")
	})
}
