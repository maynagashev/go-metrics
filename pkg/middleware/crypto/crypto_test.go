// Package crypto содержит тесты для пакета crypto.
//
//nolint:testpackage // Используем тот же пакет для доступа к непубличным полям
package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/pkg/crypto"
	"github.com/maynagashev/go-metrics/pkg/sign"
)

// TestMiddleware_Handler tests the middleware's handler functionality.
func TestMiddleware_Handler(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Test cases
	testCases := []struct {
		name           string
		config         *app.Config
		requestBody    []byte
		setupRequest   func(r *http.Request)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "No encryption or signing",
			config: &app.Config{
				PrivateKey: "test-key",
			},
			requestBody:    []byte(`{"test":"data"}`),
			setupRequest:   func(_ *http.Request) {},
			expectedStatus: http.StatusOK,
			expectedBody:   "handler called",
		},
		{
			name: "With valid signature",
			config: &app.Config{
				PrivateKey: "test-key",
			},
			requestBody: []byte(`{"test":"data"}`),
			setupRequest: func(r *http.Request) {
				hash := sign.ComputeHMACSHA256([]byte(`{"test":"data"}`), "test-key")
				r.Header.Set(sign.HeaderKey, hash)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "handler called",
		},
		{
			name: "With invalid signature",
			config: &app.Config{
				PrivateKey: "test-key",
			},
			requestBody: []byte(`{"test":"data"}`),
			setupRequest: func(r *http.Request) {
				r.Header.Set(sign.HeaderKey, "invalid-signature")
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request signature",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create middleware
			middleware := New(tc.config, logger)

			// Create a simple test handler
			handlerCalled := false
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("handler called"))
				if err != nil {
					t.Fatalf("Failed to write response: %v", err)
				}
			})

			// Create a request with the test body
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Apply any request setup
			tc.setupRequest(req)

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Call the middleware with our test handler
			handler := middleware(testHandler)
			handler.ServeHTTP(rr, req)

			// Check the response
			assert.Equal(t, tc.expectedStatus, rr.Code)

			// Only check for expected body content if we got the expected status
			if rr.Code == tc.expectedStatus {
				assert.Contains(t, rr.Body.String(), tc.expectedBody)
			}

			// Only check if handler was called if we expect a successful response
			if tc.expectedStatus == http.StatusOK {
				assert.True(t, handlerCalled, "Handler should have been called")
			}
		})
	}
}

// TestContextKey_String tests the String method of the ContextKey type.
func TestContextKey_String(t *testing.T) {
	testCases := []struct {
		name     string
		keyName  string
		expected string
	}{
		{
			name:     "Basic key name",
			keyName:  "test_key",
			expected: "crypto middleware context key: test_key",
		},
		{
			name:     "Empty key name",
			keyName:  "",
			expected: "crypto middleware context key: ",
		},
		{
			name:     "Special characters",
			keyName:  "key-with.special_chars",
			expected: "crypto middleware context key: key-with.special_chars",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Теперь можем напрямую создать ContextKey с приватным полем
			key := ContextKey{name: tc.keyName}
			result := key.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

// mockResponseWriter имитирует http.ResponseWriter для тестирования.
type mockResponseWriter struct {
	mock.Mock
	http.ResponseWriter
	headers    http.Header
	statusCode int
	body       bytes.Buffer
}

func (m *mockResponseWriter) Header() http.Header {
	return m.headers
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	args := m.Called(b)
	m.body.Write(b) // Игнорируем ошибку и возвращаемое значение здесь
	return args.Int(0), args.Error(1)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

// TestHandleEncryption проверяет метод handleEncryption.
func TestHandleEncryption(t *testing.T) {
	// Создаем приватный ключ для тестирования
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Общие настройки для всех тестов
	logger, _ := zap.NewDevelopment()

	testCases := []struct {
		name                 string
		encryptionEnabled    bool
		contentEncrypted     bool
		requestBody          []byte
		expectedStatus       int
		expectedSuccess      bool
		expectedBodyContains string
	}{
		{
			name:                 "No encryption header",
			encryptionEnabled:    true,
			contentEncrypted:     false,
			requestBody:          []byte("unencrypted data"),
			expectedStatus:       0,
			expectedSuccess:      true,
			expectedBodyContains: "",
		},
		{
			name:                 "Encryption header but encryption disabled",
			encryptionEnabled:    false,
			contentEncrypted:     true,
			requestBody:          []byte("encrypted data"),
			expectedStatus:       http.StatusBadRequest,
			expectedSuccess:      false,
			expectedBodyContains: "Server is not configured for encryption",
		},
		{
			name:                 "Invalid encrypted data",
			encryptionEnabled:    true,
			contentEncrypted:     true,
			requestBody:          []byte("invalid encrypted data"),
			expectedStatus:       http.StatusBadRequest,
			expectedSuccess:      false,
			expectedBodyContains: "Failed to decrypt request body",
		},
		{
			name:              "Successfully decrypted data",
			encryptionEnabled: true,
			contentEncrypted:  true,
			requestBody: func() []byte {
				// Шифруем тестовые данные с помощью публичного ключа
				originalData := []byte("test data to encrypt")
				encrypted, _ := crypto.EncryptLargeData(&privateKey.PublicKey, originalData)
				return encrypted
			}(),
			expectedStatus:       0,
			expectedSuccess:      true,
			expectedBodyContains: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Настраиваем конфигурацию
			config := &app.Config{}
			if tc.encryptionEnabled {
				config.PrivateRSAKey = privateKey
			}

			// Создаем middleware
			middleware := &Middleware{
				log:              logger,
				config:           config,
				processedBodyKey: ContextKey{name: "processed_body"},
			}

			// Создаем запрос
			req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(tc.requestBody))
			if tc.contentEncrypted {
				req.Header.Set("Content-Encrypted", "true")
			}

			// Создаем response writer
			mockWriter := &mockResponseWriter{
				headers: make(http.Header),
			}
			mockWriter.On("Write", mock.Anything).Return(len(tc.expectedBodyContains), nil)

			// Вызываем тестируемый метод
			processedBody, success := middleware.handleEncryption(mockWriter, req, tc.requestBody)

			// Проверяем результаты
			assert.Equal(t, tc.expectedSuccess, success)

			if tc.expectedStatus != 0 {
				assert.Equal(t, tc.expectedStatus, mockWriter.statusCode)
			}

			if tc.expectedSuccess && tc.contentEncrypted {
				// Проверяем, что данные успешно расшифрованы
				assert.NotEqual(t, tc.requestBody, processedBody)
				assert.NotNil(t, processedBody)
			}

			if tc.expectedBodyContains != "" {
				mockWriter.AssertCalled(t, "Write", mock.Anything)
			}
		})
	}
}
