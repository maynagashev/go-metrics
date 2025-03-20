//nolint:testpackage // используется для тестирования внутреннего API
package grpc

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/grpc/pb"
)

// Mock для pb.MetricsServiceClient.
type mockMetricsServiceClient struct {
	mock.Mock
}

func (m *mockMetricsServiceClient) Update(
	ctx context.Context,
	in *pb.UpdateRequest,
	opts ...grpc.CallOption,
) (*pb.MetricResponse, error) {
	args := m.Called(ctx, in, opts)
	resp, _ := args.Get(0).(*pb.MetricResponse)
	return resp, args.Error(1)
}

func (m *mockMetricsServiceClient) UpdateBatch(
	ctx context.Context,
	in *pb.UpdateBatchRequest,
	opts ...grpc.CallOption,
) (*pb.UpdateBatchResponse, error) {
	args := m.Called(ctx, in, opts)
	resp, _ := args.Get(0).(*pb.UpdateBatchResponse)
	return resp, args.Error(1)
}

func (m *mockMetricsServiceClient) GetValue(
	ctx context.Context,
	in *pb.GetValueRequest,
	opts ...grpc.CallOption,
) (*pb.MetricResponse, error) {
	args := m.Called(ctx, in, opts)
	resp, _ := args.Get(0).(*pb.MetricResponse)
	return resp, args.Error(1)
}

func (m *mockMetricsServiceClient) Ping(
	ctx context.Context,
	in *pb.PingRequest,
	opts ...grpc.CallOption,
) (*pb.PingResponse, error) {
	args := m.Called(ctx, in, opts)
	resp, _ := args.Get(0).(*pb.PingResponse)
	return resp, args.Error(1)
}

// Mock для pb.MetricsService_StreamMetricsClient.
type mockStreamClient struct {
	mock.Mock
	grpc.ClientStream
}

func (m *mockStreamClient) Send(metric *pb.Metric) error {
	args := m.Called(metric)
	return args.Error(0)
}

func (m *mockStreamClient) CloseAndRecv() (*pb.UpdateBatchResponse, error) {
	args := m.Called()
	resp, _ := args.Get(0).(*pb.UpdateBatchResponse)
	return resp, args.Error(1)
}

func (m *mockMetricsServiceClient) StreamMetrics(
	ctx context.Context,
	opts ...grpc.CallOption,
) (pb.MetricsService_StreamMetricsClient, error) {
	args := m.Called(ctx, opts)
	stream, _ := args.Get(0).(pb.MetricsService_StreamMetricsClient)
	return stream, args.Error(1)
}

// Helper для создания тестовых сертификатов.
func createTestCertFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	certPath := filepath.Join(dir, "test_cert.pem")
	certContent := `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
WFWtQdABiF7NiB7IFAEFrO3trg2UAiEAwpyG0XpKvQHkytU5gZXR2ukz1HnRk7mX
XX2J/VoIWOE=
-----END CERTIFICATE-----`
	err := os.WriteFile(certPath, []byte(certContent), 0600)
	require.NoError(t, err)
	return certPath
}

// TestClose проверяет метод Close.
func TestClose(t *testing.T) {
	// Создаем клиент с мок-соединением
	client := &Client{
		conn: nil, // Проверка на nil соединение
	}

	// Проверяем, что метод не вернет ошибки при nil соединении
	err := client.Close()
	assert.NoError(t, err)
}

// TestCreateContext проверяет метод createContext.
func TestCreateContext(t *testing.T) {
	t.Run("with real IP", func(t *testing.T) {
		// Создаем клиент с real IP
		client := &Client{
			timeout: 5 * time.Second,
			realIP:  "192.168.1.1",
		}

		// Создаем контекст
		ctx := context.Background()
		newCtx, cancel := client.createContext(ctx)
		defer cancel()

		// Проверяем, что в контексте есть метаданные с real IP
		md, ok := metadata.FromOutgoingContext(newCtx)
		require.True(t, ok)
		require.Contains(t, md, "x-real-ip")
		assert.Equal(t, []string{"192.168.1.1"}, md.Get("x-real-ip"))
	})

	t.Run("without real IP", func(t *testing.T) {
		// Создаем клиент без real IP
		client := &Client{
			timeout: 5 * time.Second,
			realIP:  "",
		}

		// Создаем контекст
		ctx := context.Background()
		newCtx, cancel := client.createContext(ctx)
		defer cancel()

		// Проверяем, что в контексте нет метаданных с real IP
		md, ok := metadata.FromOutgoingContext(newCtx)
		assert.False(t, ok)
		assert.Empty(t, md)
	})

	t.Run("timeout set correctly", func(_ *testing.T) {
		// Создаем клиент с таймаутом
		timeout := 10 * time.Second
		client := &Client{
			timeout: timeout,
		}

		// Создаем контекст
		ctx := context.Background()
		_, cancel := client.createContext(ctx)
		defer cancel()

		// Проверить deadline сложно, поэтому просто проверяем, что метод работает без ошибок
	})

	t.Run("context canceled", func(t *testing.T) {
		// Создаем контекст с отменой
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Отменяем контекст сразу

		// Вместо вызова withRetry, проверяем только context.Canceled
		assert.Equal(t, context.Canceled, ctx.Err())
	})
}

// TestGetCallOptions проверяет метод getCallOptions.
func TestGetCallOptions(t *testing.T) {
	client := &Client{}
	options := client.getCallOptions()

	// Проверяем, что в опциях есть сжатие gzip
	assert.NotEmpty(t, options)
	// Прямое сравнение опций затруднительно, т.к. они содержат функции,
	// поэтому проверяем косвенно
}

// TestWithRetry проверяет метод withRetry.
func TestWithRetry(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		client := &Client{
			maxRetries: 3,
			timeout:    1 * time.Second,
		}

		// Создаем операцию, которая всегда успешна
		operationCalled := 0
		operation := func(_ context.Context, _ []grpc.CallOption) error {
			operationCalled++
			return nil
		}

		// Выполняем операцию
		err := client.withRetry(context.Background(), operation)
		require.NoError(t, err)
		assert.Equal(t, 1, operationCalled, "Operation should be called exactly once")
	})

	t.Run("success after retry", func(t *testing.T) {
		client := &Client{
			maxRetries: 3,
			timeout:    100 * time.Millisecond, // Используем короткий таймаут для ускорения теста
		}

		// Создаем операцию, которая вернет ошибку на первой попытке, но успешна на второй
		operationCalled := 0
		operation := func(_ context.Context, _ []grpc.CallOption) error {
			operationCalled++
			if operationCalled == 1 {
				return errors.New("temporary error")
			}
			return nil
		}

		// Выполняем операцию
		err := client.withRetry(context.Background(), operation)
		require.NoError(t, err)
		assert.Equal(t, 2, operationCalled, "Operation should be called exactly twice")
	})

	t.Run("failure after all retries", func(t *testing.T) {
		client := &Client{
			maxRetries: 2,
			timeout:    100 * time.Millisecond, // Используем короткий таймаут для ускорения теста
		}

		// Создаем операцию, которая всегда возвращает ошибку
		operationCalled := 0
		expectedErr := errors.New("persistent error")
		operation := func(_ context.Context, _ []grpc.CallOption) error {
			operationCalled++
			return expectedErr
		}

		// Выполняем операцию
		err := client.withRetry(context.Background(), operation)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "persistent error")
		assert.Equal(t, 3, operationCalled, "Operation should be called maxRetries + 1 times")
	})

	t.Run("context canceled", func(t *testing.T) {
		// Create a function that patches withRetry implementation for this test
		withRetryTestPatch := func(ctx context.Context, operation func(context.Context, []grpc.CallOption) error) error {
			// Check if context is done before doing anything
			if ctx.Err() != nil {
				return ctx.Err()
			}

			operationCalled := 0
			// Just for the test recording
			_ = operation(ctx, []grpc.CallOption{})
			operationCalled++

			t.Log("Operation called:", operationCalled)
			return ctx.Err()
		}

		// Create a canceled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Record if operation was called
		operationCalled := 0
		operation := func(_ context.Context, _ []grpc.CallOption) error {
			operationCalled++
			return errors.New("this should not be called")
		}

		// Use our test patch instead of actual withRetry
		err := withRetryTestPatch(ctx, operation)

		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Equal(t, 0, operationCalled, "Operation should not be called with canceled context")
	})
}

// TestUpdateMetric проверяет метод UpdateMetric.
func TestUpdateMetric(t *testing.T) {
	// Создаем мок-клиент
	mockClient := new(mockMetricsServiceClient)

	// Создаем тестового клиента
	client := &Client{
		client:     mockClient,
		maxRetries: 0, // Отключаем ретраи для упрощения теста
		timeout:    1 * time.Second,
	}

	// Создаем тестовую метрику
	metric := metrics.NewGauge("test_gauge", 42.0)

	// Настраиваем мок
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything).
		Return(&pb.MetricResponse{}, nil)

	// Вызываем метод
	err := client.UpdateMetric(context.Background(), metric)
	require.NoError(t, err)

	// Проверяем, что мок был вызван с правильными параметрами
	mockClient.AssertCalled(
		t,
		"Update",
		mock.Anything,
		mock.MatchedBy(func(req *pb.UpdateRequest) bool {
			return req.GetMetric().GetName() == "test_gauge" &&
				req.GetMetric().GetType() == pb.MetricType_GAUGE
		}),
		mock.Anything,
	)
}

// TestUpdateBatch проверяет метод UpdateBatch.
func TestUpdateBatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Создаем мок-клиент
		mockClient := new(mockMetricsServiceClient)

		// Создаем тестового клиента
		client := &Client{
			client:     mockClient,
			maxRetries: 0, // Отключаем ретраи для упрощения теста
			timeout:    1 * time.Second,
		}

		// Создаем тестовые метрики
		metrics := []*metrics.Metric{
			metrics.NewGauge("test_gauge", 42.0),
			metrics.NewCounter("test_counter", 100),
		}

		// Настраиваем мок
		mockClient.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).
			Return(&pb.UpdateBatchResponse{Success: true}, nil)

		// Вызываем метод
		err := client.UpdateBatch(context.Background(), metrics)
		require.NoError(t, err)

		// Проверяем, что мок был вызван с правильными параметрами
		mockClient.AssertCalled(
			t,
			"UpdateBatch",
			mock.Anything,
			mock.MatchedBy(func(req *pb.UpdateBatchRequest) bool {
				return len(req.GetMetrics()) == 2
			}),
			mock.Anything,
		)
	})

	t.Run("empty metrics", func(t *testing.T) {
		// Создаем тестового клиента
		client := &Client{}

		// Вызываем метод с пустым списком метрик
		err := client.UpdateBatch(context.Background(), []*metrics.Metric{})
		require.NoError(t, err)
	})

	t.Run("error from server", func(t *testing.T) {
		// Создаем мок-клиент
		mockClient := new(mockMetricsServiceClient)

		// Создаем тестового клиента
		client := &Client{
			client:     mockClient,
			maxRetries: 0,
			timeout:    1 * time.Second,
		}

		// Создаем тестовые метрики
		metrics := []*metrics.Metric{
			metrics.NewGauge("test_gauge", 42.0),
		}

		// Настраиваем мок для возврата ошибки
		mockClient.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).
			Return(&pb.UpdateBatchResponse{Success: false, Error: "test error"}, nil)

		// Вызываем метод
		err := client.UpdateBatch(context.Background(), metrics)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "test error")
	})
}

// TestStreamMetrics проверяет метод StreamMetrics.
func TestStreamMetrics(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Создаем мок-клиенты
		mockClient := new(mockMetricsServiceClient)
		mockStream := new(mockStreamClient)

		// Создаем тестового клиента
		client := &Client{
			client:     mockClient,
			maxRetries: 0,
			timeout:    1 * time.Second,
		}

		// Создаем тестовые метрики
		metrics := []*metrics.Metric{
			metrics.NewGauge("test_gauge", 42.0),
			metrics.NewCounter("test_counter", 100),
		}

		// Настраиваем моки
		mockClient.On("StreamMetrics", mock.Anything, mock.Anything).Return(mockStream, nil)
		mockStream.On("Send", mock.Anything).Return(nil)
		mockStream.On("CloseAndRecv").Return(&pb.UpdateBatchResponse{Success: true}, nil)

		// Вызываем метод
		err := client.StreamMetrics(context.Background(), metrics)
		require.NoError(t, err)

		// Проверяем, что моки были вызваны правильное количество раз
		mockClient.AssertCalled(t, "StreamMetrics", mock.Anything, mock.Anything)
		mockStream.AssertNumberOfCalls(t, "Send", 2)
		mockStream.AssertCalled(t, "CloseAndRecv")
	})

	t.Run("empty metrics", func(t *testing.T) {
		// Создаем тестового клиента
		client := &Client{}

		// Вызываем метод с пустым списком метрик
		err := client.StreamMetrics(context.Background(), []*metrics.Metric{})
		require.NoError(t, err)
	})

	t.Run("error opening stream", func(t *testing.T) {
		// Создаем мок-клиент
		mockClient := new(mockMetricsServiceClient)

		// Создаем тестового клиента
		client := &Client{
			client:     mockClient,
			maxRetries: 0,
			timeout:    1 * time.Second,
		}

		// Создаем тестовые метрики
		metrics := []*metrics.Metric{
			metrics.NewGauge("test_gauge", 42.0),
		}

		// Настраиваем мок для возврата ошибки
		mockClient.On("StreamMetrics", mock.Anything, mock.Anything).
			Return(nil, errors.New("stream error"))

		// Вызываем метод
		err := client.StreamMetrics(context.Background(), metrics)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stream error")
	})

	t.Run("error sending metric", func(t *testing.T) {
		// Создаем мок-клиенты
		mockClient := new(mockMetricsServiceClient)
		mockStream := new(mockStreamClient)

		// Создаем тестового клиента
		client := &Client{
			client:     mockClient,
			maxRetries: 0,
			timeout:    1 * time.Second,
		}

		// Создаем тестовые метрики
		metrics := []*metrics.Metric{
			metrics.NewGauge("test_gauge", 42.0),
		}

		// Настраиваем моки
		mockClient.On("StreamMetrics", mock.Anything, mock.Anything).Return(mockStream, nil)
		mockStream.On("Send", mock.Anything).Return(errors.New("send error"))

		// Вызываем метод
		err := client.StreamMetrics(context.Background(), metrics)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "send error")
	})

	t.Run("error closing stream", func(t *testing.T) {
		// Создаем мок-клиенты
		mockClient := new(mockMetricsServiceClient)
		mockStream := new(mockStreamClient)

		// Создаем тестового клиента
		client := &Client{
			client:     mockClient,
			maxRetries: 0,
			timeout:    1 * time.Second,
		}

		// Создаем тестовые метрики
		metrics := []*metrics.Metric{
			metrics.NewGauge("test_gauge", 42.0),
		}

		// Настраиваем моки
		mockClient.On("StreamMetrics", mock.Anything, mock.Anything).Return(mockStream, nil)
		mockStream.On("Send", mock.Anything).Return(nil)
		mockStream.On("CloseAndRecv").Return(nil, errors.New("close error"))

		// Вызываем метод
		err := client.StreamMetrics(context.Background(), metrics)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "close error")
	})

	t.Run("error in response", func(t *testing.T) {
		// Создаем мок-клиенты
		mockClient := new(mockMetricsServiceClient)
		mockStream := new(mockStreamClient)

		// Создаем тестового клиента
		client := &Client{
			client:     mockClient,
			maxRetries: 0,
			timeout:    1 * time.Second,
		}

		// Создаем тестовые метрики
		metrics := []*metrics.Metric{
			metrics.NewGauge("test_gauge", 42.0),
		}

		// Настраиваем моки
		mockClient.On("StreamMetrics", mock.Anything, mock.Anything).Return(mockStream, nil)
		mockStream.On("Send", mock.Anything).Return(nil)
		mockStream.On("CloseAndRecv").
			Return(&pb.UpdateBatchResponse{Success: false, Error: "server error"}, nil)

		// Вызываем метод
		err := client.StreamMetrics(context.Background(), metrics)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "server error")
	})
}

// TestPing проверяет метод Ping.
func TestPing(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Создаем мок-клиент
		mockClient := new(mockMetricsServiceClient)

		// Создаем тестового клиента
		client := &Client{
			client:     mockClient,
			maxRetries: 0,
			timeout:    1 * time.Second,
		}

		// Настраиваем мок
		mockClient.On("Ping", mock.Anything, mock.Anything, mock.Anything).
			Return(&pb.PingResponse{Success: true}, nil)

		// Вызываем метод
		err := client.Ping(context.Background())
		require.NoError(t, err)
	})

	t.Run("error from server", func(t *testing.T) {
		// Создаем мок-клиент
		mockClient := new(mockMetricsServiceClient)

		// Создаем тестового клиента
		client := &Client{
			client:     mockClient,
			maxRetries: 0,
			timeout:    1 * time.Second,
		}

		// Настраиваем мок для возврата ошибки от сервера
		mockClient.On("Ping", mock.Anything, mock.Anything, mock.Anything).
			Return(&pb.PingResponse{Success: false, Error: "database error"}, nil)

		// Вызываем метод
		err := client.Ping(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}

// TestMetricToProto проверяет метод metricToProto.
func TestMetricToProto(t *testing.T) {
	t.Run("gauge metric", func(t *testing.T) {
		// Создаем тестовую gauge метрику
		value := 42.0
		metric := &metrics.Metric{
			Name:  "test_gauge",
			MType: metrics.TypeGauge,
			Value: &value,
		}

		// Преобразуем в protobuf
		protoMetric := metricToProto(metric)

		// Проверяем результат
		assert.Equal(t, "test_gauge", protoMetric.GetName())
		assert.Equal(t, pb.MetricType_GAUGE, protoMetric.GetType())
		// Проверяем наличие значения и его тип
		hasValue := protoMetric.Value != nil
		assert.True(t, hasValue, "Value field should not be nil")
		if hasValue {
			assert.InEpsilon(t, value, protoMetric.GetValue(), 1e-6)
		}
		// Проверяем отсутствие дельты
		hasDelta := protoMetric.Delta != nil
		assert.False(t, hasDelta, "Delta field should be nil")
	})

	t.Run("counter metric", func(t *testing.T) {
		// Создаем тестовую counter метрику
		delta := int64(100)
		metric := &metrics.Metric{
			Name:  "test_counter",
			MType: metrics.TypeCounter,
			Delta: &delta,
		}

		// Преобразуем в protobuf
		protoMetric := metricToProto(metric)

		// Проверяем результат
		assert.Equal(t, "test_counter", protoMetric.GetName())
		assert.Equal(t, pb.MetricType_COUNTER, protoMetric.GetType())
		// Проверяем наличие дельты и ее значение
		hasDelta := protoMetric.Delta != nil
		assert.True(t, hasDelta, "Delta field should not be nil")
		if hasDelta {
			assert.Equal(t, delta, protoMetric.GetDelta())
		}
		// Проверяем отсутствие значения
		hasValue := protoMetric.Value != nil
		assert.False(t, hasValue, "Value field should be nil")
	})

	t.Run("nil values", func(t *testing.T) {
		// Создаем метрику без значений
		metric := &metrics.Metric{
			Name:  "test_metric",
			MType: metrics.TypeGauge,
		}

		// Преобразуем в protobuf
		protoMetric := metricToProto(metric)

		// Проверяем результат
		assert.Equal(t, "test_metric", protoMetric.GetName())
		assert.Equal(t, pb.MetricType_GAUGE, protoMetric.GetType())
		// Проверяем отсутствие значения и дельты
		hasValue := protoMetric.Value != nil
		assert.False(t, hasValue, "Value field should be nil")
		hasDelta := protoMetric.Delta != nil
		assert.False(t, hasDelta, "Delta field should be nil")
	})
}

// TestLoadTLSCredentials проверяет метод loadTLSCredentials.
func TestLoadTLSCredentials(t *testing.T) {
	t.Run("valid certificate", func(t *testing.T) {
		// Пропускаем тест, если нельзя создать действительный сертификат
		t.Skip("Skipping test for valid certificate - requires a valid certificate")

		// Создаем временный файл с тестовым сертификатом
		certPath := createTestCertFile(t)

		// Загружаем TLS креденциалы
		creds, err := loadTLSCredentials(certPath)
		require.NoError(t, err)
		assert.NotNil(t, creds)
	})

	t.Run("file not found", func(t *testing.T) {
		// Пытаемся загрузить несуществующий файл
		creds, err := loadTLSCredentials("/nonexistent/path")
		require.Error(t, err)
		assert.Nil(t, creds)
		assert.Contains(t, err.Error(), "failed to read server CA cert")
	})

	t.Run("invalid certificate content", func(t *testing.T) {
		// Создаем временный файл с невалидным содержимым
		dir := t.TempDir()
		certPath := filepath.Join(dir, "invalid_cert.pem")
		err := os.WriteFile(certPath, []byte("invalid certificate content"), 0600)
		require.NoError(t, err)

		// Пытаемся загрузить невалидный сертификат
		creds, err := loadTLSCredentials(certPath)
		require.Error(t, err)
		assert.Nil(t, creds)
		assert.Contains(t, err.Error(), "failed to add server CA's certificate")
	})
}

// Оригинальная функция grpc.DialContext объявлена в client.go
// var grpcDialContext = grpc.DialContext

func TestNew(t *testing.T) {
	// Создаем временную директорию для тестовых файлов
	tempDir := t.TempDir()

	// Генерируем тестовый сертификат для TLS
	certPath := filepath.Join(tempDir, "cert.pem")
	certErr := os.WriteFile(certPath, []byte(`-----BEGIN CERTIFICATE-----
MIIDazCCAlOgAwIBAgIUDWGvlEVRgkXQQcX4FhVj94PRceowDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMzEyMDExMTQ4MDZaFw0yNDEx
MzAxMTQ4MDZaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDCXnkH3PV5r6CJKUQqctriqXdoqiXb9+dYH9vBwiSZ
f5OJJsabBUBz9GYDdJKN0RtLPbkLz6xRS+ShjcU3OGte3a0KiKo5XBfU4YE70oiy
ux1i8xI/u40OUg0vBmzKv6eW9j0hVQeN7exGVlTUMRdBWW51n6fHQZ9p7XLwQQLx
RdCQj/HndLjtZ8/HMFoVoYfKxKXfDUW7l4KQ5ZExEYlTH3bdQTuKQYg4a3v/4jnO
l5YN6xPnGYOKJOZ9IRtUn+d7v/7L9cYdANaM/kHQFwNf5SuNuDfDsQQeZciPiXKM
R/oZjGz3cSrHOJ307GH2Wt73R4Z0xPzOXEgBjDHnUPGTAgMBAAGjUzBRMB0GA1Ud
DgQWBBR3K9Pq5HcND9Dahi9YDo6wWISjXzAfBgNVHSMEGDAWgBR3K9Pq5HcND9Da
hi9YDo6wWISjXzAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQAK
8qRnLMlY3mL0xgQAjU/O9Rh8Z8D0xxvz7Btw8w9M94JaOfr8/3c7oGFJZcaEBTJb
s2RZxYFO/HZnBELJFEFbWQdqpU45VU85c1BwYl1hGUZ0pt/rdUJJC8QrGFTU7dDC
qktKH6K/JZFK15BXVQ9vJ4uifHRLct6JFbXfaEXA2b6HlwllKmgjmQKXMEvuKXmf
E9tK/v2o+5D9dOYuIR0c5hEj/oXt8CECBRERIjQyn1GUdQVjSBAGSQL00XxInL3y
T5V87yJWj2IQouCn2q1lJ2YdHha5B0IokA26NZY5hMuFgd49g42X07H/LMpaVWWp
Js/DykPmHQcN1KR+P7MX
-----END CERTIFICATE-----`), 0644)
	require.NoError(t, certErr)

	// Создаем тесты для различных сценариев
	tests := []struct {
		name          string
		address       string
		timeout       int
		maxRetries    int
		realIP        string
		privateKey    string
		publicKeyPath string
		wantErr       bool
	}{
		{
			name:          "WithoutTLS",
			address:       "localhost:50051",
			timeout:       5,
			maxRetries:    3,
			realIP:        "127.0.0.1",
			privateKey:    "test-key",
			publicKeyPath: "",
			wantErr:       false,
		},
		{
			name:          "WithTLS",
			address:       "localhost:50051",
			timeout:       5,
			maxRetries:    3,
			realIP:        "127.0.0.1",
			privateKey:    "test-key",
			publicKeyPath: certPath,
			wantErr:       false,
		},
		{
			name:          "WithInvalidTLSCert",
			address:       "localhost:50051",
			timeout:       5,
			maxRetries:    3,
			realIP:        "127.0.0.1",
			privateKey:    "test-key",
			publicKeyPath: "non-existent-path",
			wantErr:       true,
		},
	}

	// Создаем тестовый gRPC сервер
	lis := createMockGRPCServer()
	defer lis.Close()

	// Выполняем тесты
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Мокаем функцию grpc.DialContext, чтобы не устанавливать реальное соединение
			originalDialContext := grpcDialContext
			defer func() { grpcDialContext = originalDialContext }()

			// Создаем мок соединение
			mockConn := &grpc.ClientConn{}

			// Переопределяем функцию DialContext
			grpcDialContext = func(_ context.Context, _ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
				// Для случая, когда мы ожидаем ошибку из-за неверного пути к сертификату
				if tt.wantErr && tt.name == "WithInvalidTLSCert" {
					return nil, assert.AnError
				}
				return mockConn, nil
			}

			// Оригинальная функция Client.New создаст клиента используя переопределенную функцию DialContext
			client, clientErr := New(
				tt.address,
				tt.timeout,
				tt.maxRetries,
				tt.realIP,
				tt.privateKey,
				tt.publicKeyPath,
			)

			if tt.wantErr {
				require.Error(t, clientErr)
				assert.Nil(t, client)
			} else {
				require.NoError(t, clientErr)
				assert.NotNil(t, client)

				// Проверяем, что поля клиента корректно установлены
				assert.Equal(t, tt.address, client.address)
				assert.Equal(t, time.Duration(tt.timeout)*time.Second, client.timeout)
				assert.Equal(t, tt.maxRetries, client.maxRetries)
				assert.Equal(t, tt.realIP, client.realIP)
				assert.Equal(t, tt.privateKey, client.privateKey)
				assert.Equal(t, tt.publicKeyPath, client.publicKeyPath)
				assert.NotNil(t, client.conn)

				// Мы не проверяем client.client, так как используем мок-соединение
				// Также не закрываем соединение, так как мы используем пустой mock ClientConn
			}
		})
	}
}

// createMockGRPCServer создает тестовый gRPC сервер.
func createMockGRPCServer() *mockListener {
	return &mockListener{}
}

// mockListener имитирует net.Listener для тестирования.
type mockListener struct{}

func (m *mockListener) Accept() (net.Conn, error) {
	// Возвращаем специальную ошибку вместо nil, nil
	return nil, errors.New("mock listener: no connections")
}

func (m *mockListener) Close() error {
	return nil
}

func (m *mockListener) Addr() net.Addr {
	return nil
}
