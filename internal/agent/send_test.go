package agent_test

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maynagashev/go-metrics/internal/agent"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// mockNetError реализует интерфейс net.Error для тестирования.
type mockNetError struct {
	timeout   bool
	temporary bool
	msg       string
}

func (e *mockNetError) Error() string   { return e.msg }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

// createNetOpError создает net.OpError для тестирования.
func createNetOpError() *net.OpError {
	return &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New("connection refused"),
	}
}

// TestIsRetriableSendError тестирует функцию isRetriableSendError
// Нам нужно экспортировать эту функцию для тестирования
//
//nolint:gochecknoglobals // используется только для тестирования
var isRetriableSendError = agent.IsRetriableSendError

func TestIsRetriableSendError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "timeout error",
			err:      &mockNetError{timeout: true, msg: "timeout error"},
			expected: true,
		},
		{
			name:     "temporary error",
			err:      &mockNetError{temporary: true, msg: "temporary error"},
			expected: false, // В функции проверяется только Timeout()
		},
		{
			name:     "net.OpError",
			err:      createNetOpError(),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isRetriableSendError(tc.err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMakeUpdatesRequest(t *testing.T) {
	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаем агента для тестирования
	a := agent.New(
		server.URL,
		time.Second,
		time.Second,
		"",
		5,
		nil,
		"",
		false,
		"localhost:9090",
		5,
		3,
	)

	// Создаем тестовые метрики
	value := 42.0
	metrics := []*metrics.Metric{
		metrics.NewGauge("test_gauge", value),
		metrics.NewCounter("test_counter", 1),
	}

	// Вызываем метод отправки метрик
	err := agent.SendMetrics(a, metrics, 1)
	require.NoError(t, err)
}

func TestMakeUpdatesRequest_Error(t *testing.T) {
	// Создаем тестовый сервер, который возвращает ошибку
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Создаем агента для тестирования
	a := agent.New(
		server.URL,
		time.Second,
		time.Second,
		"",
		5,
		nil,
		"",
		false,
		"localhost:9090",
		5,
		3,
	)

	// Создаем тестовые метрики
	value := 42.0
	metrics := []*metrics.Metric{
		metrics.NewGauge("test_gauge", value),
		metrics.NewCounter("test_counter", 1),
	}

	// Вызываем метод отправки метрик
	err := agent.SendMetrics(a, metrics, 1)
	require.Error(t, err)
}

func TestMakeUpdatesRequest_WithCompression(t *testing.T) {
	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что запрос содержит заголовок Content-Encoding: gzip
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаем агента для тестирования с включенным сжатием
	a := agent.New(
		server.URL,
		time.Second,
		time.Second,
		"",
		5,
		nil,
		"",
		false,
		"localhost:9090",
		5,
		3,
	)

	// Создаем тестовые метрики
	value := 42.0
	metrics := []*metrics.Metric{
		metrics.NewGauge("test_gauge", value),
		metrics.NewCounter("test_counter", 1),
	}

	// Вызываем метод отправки метрик
	err := agent.SendMetrics(a, metrics, 1)
	require.NoError(t, err)
}

func TestMakeUpdatesRequest_WithRealIP(t *testing.T) {
	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что запрос содержит заголовок X-Real-IP
		assert.NotEmpty(t, r.Header.Get("X-Real-IP"))
		// Проверяем, что IP-адрес в заголовке X-Real-IP является валидным
		ip := net.ParseIP(r.Header.Get("X-Real-IP"))
		assert.NotNil(t, ip)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаем агента для тестирования
	a := agent.New(
		server.URL,
		time.Second,
		time.Second,
		"",
		5,
		nil,
		"",
		false,
		"localhost:9090",
		5,
		3,
	)

	// Создаем тестовые метрики
	value := 42.0
	metrics := []*metrics.Metric{
		metrics.NewGauge("test_gauge", value),
		metrics.NewCounter("test_counter", 1),
	}

	// Вызываем метод отправки метрик
	err := agent.SendMetrics(a, metrics, 1)
	require.NoError(t, err)
}

func TestMakeUpdatesRequest_WithSigning(t *testing.T) {
	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что запрос содержит заголовок Hashsha256
		assert.NotEmpty(t, r.Header.Get("Hashsha256"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаем агента для тестирования с включенной подписью
	a := agent.New(
		server.URL,
		time.Second,
		time.Second,
		"test-key",
		5,
		nil,
		"",
		false,
		"localhost:9090",
		5,
		3,
	)

	// Создаем тестовые метрики
	value := 42.0
	metrics := []*metrics.Metric{
		metrics.NewGauge("test_gauge", value),
		metrics.NewCounter("test_counter", 1),
	}

	// Вызываем метод отправки метрик
	err := agent.SendMetrics(a, metrics, 1)
	require.NoError(t, err)
}

func TestMakeUpdatesRequest_WithExplicitRealIP(t *testing.T) {
	// Создаем тестовый сервер
	expectedIP := "192.168.1.100"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что запрос содержит заголовок X-Real-IP с ожидаемым значением
		assert.Equal(t, expectedIP, r.Header.Get("X-Real-IP"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаем агента для тестирования с явно указанным IP-адресом
	a := agent.New(
		server.URL,
		time.Second,
		time.Second,
		"",
		5,
		nil,
		expectedIP,
		false,
		"localhost:9090",
		5,
		3,
	)

	// Создаем тестовые метрики
	value := 42.0
	metrics := []*metrics.Metric{
		metrics.NewGauge("test_gauge", value),
		metrics.NewCounter("test_counter", 1),
	}

	// Вызываем метод отправки метрик
	err := agent.SendMetrics(a, metrics, 1)
	require.NoError(t, err)
}
