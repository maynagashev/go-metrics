//nolint:testpackage // использует внутреннее API для тестирования
package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

func TestNew(t *testing.T) {
	t.Run("No Crypto Key", func(t *testing.T) {
		// Act
		c := New("http://localhost:8080", "", "", "127.0.0.1")

		// Assert
		assert.NotNil(t, c)
	})

	t.Run("Invalid Crypto Key Path", func(t *testing.T) {
		// Act
		c := New("http://localhost:8080", "", "/nonexistent/path.pem", "127.0.0.1")

		// Assert
		assert.NotNil(t, c)
	})
}

func TestClient_Close(t *testing.T) {
	// Arrange
	c := New("http://localhost:8080", "", "", "127.0.0.1")

	// Act
	err := c.Close()

	// Assert
	require.NoError(t, err)
}

func TestClient_UpdateMetric(t *testing.T) {
	// Arrange
	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(server.URL, "", "", "127.0.0.1")
	metric := metrics.NewCounter("test_metric", 10)

	// Act
	err := c.UpdateMetric(context.Background(), metric)

	// Assert
	require.NoError(t, err)
}

func TestClient_Ping(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		// Создаем тестовый сервер для успешного пинга
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/ping", r.URL.Path)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := New(server.URL, "", "", "127.0.0.1")

		// Act
		err := c.Ping(context.Background())

		// Assert
		require.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		// Arrange
		// Создаем тестовый сервер для неуспешного пинга
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/ping", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := New(server.URL, "", "", "127.0.0.1")

		// Act
		err := c.Ping(context.Background())

		// Assert
		require.Error(t, err)
	})

	t.Run("Network Error", func(t *testing.T) {
		// Arrange
		// Используем несуществующий адрес, чтобы вызвать ошибку сети
		c := New("http://nonexistent.server", "", "", "127.0.0.1")

		// Создаем кастомный транспорт, который всегда возвращает ошибку
		customTransport := &mockTransport{
			err: errors.New("custom dial error"),
		}

		// Подменяем транспорт в нижележащем клиенте resty через отражение
		c.client.SetTransport(customTransport)

		// Act
		err := c.Ping(context.Background())

		// Assert
		require.Error(t, err)
	})
}

// mockTransport реализует http.RoundTripper для тестирования.
type mockTransport struct {
	err error
}

func (t *mockTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	if t.err != nil {
		return nil, t.err
	}
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func TestClient_StreamMetrics(t *testing.T) {
	// Arrange
	c := New("http://localhost:8080", "", "", "127.0.0.1")
	metrics := []*metrics.Metric{
		metrics.NewCounter("test_metric", 10),
	}

	// Act
	err := c.StreamMetrics(context.Background(), metrics)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

// mockNetError представляет моковую реализацию net.Error для тестирования.
type mockNetError struct {
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return "mock net error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

func TestIsRetriableSendError(t *testing.T) {
	// Тестовые случаи
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
			name:     "net.Error timeout",
			err:      &mockNetError{timeout: true},
			expected: true,
		},
		{
			name:     "net.Error temporary",
			err:      &mockNetError{temporary: true},
			expected: false,
		},
		{
			name:     "net.Error neither timeout nor temporary",
			err:      &mockNetError{},
			expected: false,
		},
		{
			name:     "connection refused error",
			err:      errors.New("connect: connection refused"),
			expected: true,
		},
		{
			name:     "connection reset error",
			err:      errors.New("read: connection reset by peer"),
			expected: true,
		},
		{
			name:     "EOF error",
			err:      errors.New("unexpected EOF"),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isRetriableSendError(tc.err)
			assert.Equal(t, tc.expected, result)
		})
	}
}
