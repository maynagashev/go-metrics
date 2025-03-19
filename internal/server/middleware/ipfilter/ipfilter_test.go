package ipfilter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/middleware/ipfilter"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestIPFilterMiddleware(t *testing.T) {
	// Создаем логгер для тестов
	logger := zaptest.NewLogger(t)

	// Тестовые случаи
	testCases := []struct {
		name           string
		trustedSubnet  string
		ipHeader       string
		expectedStatus int
	}{
		{
			name:           "No trusted subnet",
			trustedSubnet:  "",
			ipHeader:       "192.168.1.1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IP in trusted subnet",
			trustedSubnet:  "192.168.1.0/24",
			ipHeader:       "192.168.1.1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IP not in trusted subnet",
			trustedSubnet:  "192.168.1.0/24",
			ipHeader:       "192.168.2.1",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Invalid IP address",
			trustedSubnet:  "192.168.1.0/24",
			ipHeader:       "invalid-ip",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "No X-Real-IP header",
			trustedSubnet:  "192.168.1.0/24",
			ipHeader:       "",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Invalid CIDR",
			trustedSubnet:  "invalid-cidr",
			ipHeader:       "192.168.1.1",
			expectedStatus: http.StatusOK, // Middleware должен пропускать запросы при некорректном CIDR
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Создаем конфигурацию
			config := &app.Config{
				TrustedSubnet: tc.trustedSubnet,
			}

			// Создаем тестовый запрос
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.ipHeader != "" {
				req.Header.Set("X-Real-IP", tc.ipHeader)
			}

			// Создаем тестовый ответ
			rr := httptest.NewRecorder()

			// Создаем тестовый обработчик
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Создаем и используем middleware
			middleware := ipfilter.New(config, logger)
			handler := middleware(testHandler)
			handler.ServeHTTP(rr, req)

			// Проверяем статус ответа
			assert.Equal(t, tc.expectedStatus, rr.Code)
		})
	}
}

func TestIPFilterMiddleware_WithRealLogger(t *testing.T) {
	// Создаем реальный логгер
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Создаем конфигурацию с доверенной подсетью
	config := &app.Config{
		TrustedSubnet: "192.168.1.0/24",
	}

	// Создаем тестовый запрос с IP-адресом из доверенной подсети
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.1")

	// Создаем тестовый ответ
	rr := httptest.NewRecorder()

	// Создаем тестовый обработчик
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Создаем и используем middleware
	middleware := ipfilter.New(config, logger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Проверяем статус ответа
	assert.Equal(t, http.StatusOK, rr.Code)
}
