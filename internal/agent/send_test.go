package agent

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/stretchr/testify/assert"
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

// Создаем реальную ошибку net.OpError для тестирования.
func createNetOpError() *net.OpError {
	return &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New("connection refused"),
	}
}

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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что запрос пришел на правильный URL
		assert.Equal(t, "/updates", r.URL.Path)

		// Проверяем заголовок Content-Type
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Возвращаем успешный ответ
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаем агента для тестирования
	a := &agent{
		ServerURL:          server.URL,
		client:             resty.New(),
		SendCompressedData: false,
		PrivateKey:         "",
		PublicKey:          nil,
	}

	// Создаем тестовые метрики
	value := 42.0
	delta := int64(10)
	items := []*metrics.Metric{
		{
			Name:  "test_gauge",
			MType: "gauge",
			Value: &value,
		},
		{
			Name:  "test_counter",
			MType: "counter",
			Delta: &delta,
		},
	}

	// Вызываем тестируемую функцию
	err := a.makeUpdatesRequest(items, 0, 1)

	// Проверяем результат
	assert.NoError(t, err)
}

func TestMakeUpdatesRequest_Error(t *testing.T) {
	// Создаем тестовый сервер, который возвращает ошибку
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Создаем агента для тестирования
	a := &agent{
		ServerURL:          server.URL,
		client:             resty.New(),
		SendCompressedData: false,
		PrivateKey:         "",
		PublicKey:          nil,
	}

	// Создаем тестовые метрики
	value := 42.0
	items := []*metrics.Metric{
		{
			Name:  "test_gauge",
			MType: "gauge",
			Value: &value,
		},
	}

	// Вызываем тестируемую функцию
	err := a.makeUpdatesRequest(items, 0, 1)

	// Проверяем результат
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 500")
}

func TestMakeUpdatesRequest_WithCompression(t *testing.T) {
	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем заголовок Content-Encoding
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))

		// Возвращаем успешный ответ
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаем агента для тестирования с включенным сжатием
	a := &agent{
		ServerURL:          server.URL,
		client:             resty.New(),
		SendCompressedData: true,
		PrivateKey:         "",
		PublicKey:          nil,
	}

	// Создаем тестовые метрики
	value := 42.0
	items := []*metrics.Metric{
		{
			Name:  "test_gauge",
			MType: "gauge",
			Value: &value,
		},
	}

	// Вызываем тестируемую функцию
	err := a.makeUpdatesRequest(items, 0, 1)

	// Проверяем результат
	assert.NoError(t, err)
}

func TestMakeUpdatesRequest_WithSigning(t *testing.T) {
	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем наличие заголовка с подписью
		assert.NotEmpty(t, r.Header.Get("HashSHA256"))

		// Возвращаем успешный ответ
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Создаем агента для тестирования с включенной подписью
	a := &agent{
		ServerURL:          server.URL,
		client:             resty.New(),
		SendCompressedData: false,
		PrivateKey:         "test-key",
		PublicKey:          nil,
	}

	// Создаем тестовые метрики
	value := 42.0
	items := []*metrics.Metric{
		{
			Name:  "test_gauge",
			MType: "gauge",
			Value: &value,
		},
	}

	// Вызываем тестируемую функцию
	err := a.makeUpdatesRequest(items, 0, 1)

	// Проверяем результат
	assert.NoError(t, err)
}
