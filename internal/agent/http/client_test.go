//nolint:testpackage // использует внутреннее API для тестирования
package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// mockNetError представляет моковую реализацию net.Error для тестирования.
type mockNetError struct {
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return "mock net error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

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

// mockDialer подменяет net.Dial для тестирования функции getOutboundIP.
type mockDialer struct {
	conn net.Conn
	err  error
}

func (m *mockDialer) Dial(_, _ string) (net.Conn, error) {
	return m.conn, m.err
}

// mockConn реализует net.Conn для тестирования.
type mockConn struct {
	localAddr net.Addr
}

func (c *mockConn) Read(_ []byte) (int, error)         { return 0, nil }
func (c *mockConn) Write(_ []byte) (int, error)        { return 0, nil }
func (c *mockConn) Close() error                       { return nil }
func (c *mockConn) LocalAddr() net.Addr                { return c.localAddr }
func (c *mockConn) RemoteAddr() net.Addr               { return nil }
func (c *mockConn) SetDeadline(_ time.Time) error      { return nil }
func (c *mockConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *mockConn) SetWriteDeadline(_ time.Time) error { return nil }

func TestGetOutboundIP(t *testing.T) {
	// Сохраняем оригинальную функцию, чтобы восстановить после теста
	originalNetDial := netDial
	defer func() { netDial = originalNetDial }()

	t.Run("Success", func(t *testing.T) {
		// Arrange
		// Создаем ожидаемый IP-адрес
		expectedIP := net.ParseIP("192.168.1.100")

		// Используем именно *net.UDPAddr
		udpAddr := &net.UDPAddr{
			IP:   expectedIP,
			Port: 12345,
		}

		// Устанавливаем нашу моковую функцию вместо net.Dial
		netDial = (&mockDialer{
			conn: &mockConn{
				localAddr: udpAddr,
			},
		}).Dial

		// Act
		ip, err := getOutboundIP()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedIP.String(), ip.String())
	})

	t.Run("Dial Error", func(t *testing.T) {
		// Arrange
		// Устанавливаем моковую функцию, которая возвращает ошибку
		netDial = (&mockDialer{
			err: errors.New("dial error"),
		}).Dial

		// Act
		ip, err := getOutboundIP()

		// Assert
		require.Error(t, err)
		assert.Nil(t, ip)
		assert.Contains(t, err.Error(), "dial error")
	})

	t.Run("Wrong Address Type", func(t *testing.T) {
		// Arrange
		// Создаем мок адреса, который не является *net.UDPAddr
		wrongAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}

		// Устанавливаем нашу моковую функцию
		netDial = (&mockDialer{
			conn: &mockConn{
				localAddr: wrongAddr,
			},
		}).Dial

		// Act
		ip, err := getOutboundIP()

		// Assert
		require.Error(t, err)
		assert.Nil(t, ip)
		assert.Contains(t, err.Error(), "unexpected address type")
	})
}

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
