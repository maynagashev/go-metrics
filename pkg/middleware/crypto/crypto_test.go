package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/pkg/sign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMiddleware_Handler(t *testing.T) {
	// Setup
	logger, _ := zap.NewDevelopment()
	config := &app.Config{
		PrivateKey: "test-key",
	}
	middleware := New(config, logger)

	// Create a simple test handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the body to verify it's accessible
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	// Test cases
	tests := []struct {
		name           string
		requestBody    []byte
		setupRequest   func(*http.Request)
		expectedStatus int
	}{
		{
			name:           "No body",
			requestBody:    nil,
			setupRequest:   func(r *http.Request) {},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "With body",
			requestBody: []byte("test body"),
			setupRequest: func(r *http.Request) {
				// Add signature
				hash := sign.ComputeHMACSHA256([]byte("test body"), config.PrivateKey)
				r.Header.Set(sign.HeaderKey, hash)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "With invalid signature",
			requestBody: []byte("test body"),
			setupRequest: func(r *http.Request) {
				r.Header.Set(sign.HeaderKey, "invalid-hash")
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			var req *http.Request
			if tc.requestBody != nil {
				req = httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(tc.requestBody))
			} else {
				req = httptest.NewRequest(http.MethodPost, "/test", nil)
			}
			tc.setupRequest(req)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the middleware
			handler := middleware(nextHandler)
			handler.ServeHTTP(rr, req)

			// Check status code
			assert.Equal(t, tc.expectedStatus, rr.Code)
		})
	}
}

func TestMiddleware_ProcessRequestBody(t *testing.T) {
	// Setup
	logger, _ := zap.NewDevelopment()
	config := &app.Config{
		PrivateKey: "test-key",
	}
	m := &Middleware{
		log:              logger,
		config:           config,
		processedBodyKey: ContextKey{"processed_body"},
	}

	// Test cases
	tests := []struct {
		name           string
		requestBody    []byte
		setupRequest   func(*http.Request)
		expectedResult bool
	}{
		{
			name:        "Valid request",
			requestBody: []byte("test body"),
			setupRequest: func(r *http.Request) {
				// No special setup
			},
			expectedResult: true,
		},
		{
			name:        "With signature",
			requestBody: []byte("test body"),
			setupRequest: func(r *http.Request) {
				hash := sign.ComputeHMACSHA256([]byte("test body"), config.PrivateKey)
				r.Header.Set(sign.HeaderKey, hash)
			},
			expectedResult: true,
		},
		{
			name:        "With invalid signature",
			requestBody: []byte("test body"),
			setupRequest: func(r *http.Request) {
				r.Header.Set(sign.HeaderKey, "invalid-hash")
			},
			expectedResult: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create request and response
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			tc.setupRequest(req)
			rr := httptest.NewRecorder()

			// Call the function
			body, ok := m.processRequestBody(rr, req, tc.requestBody)

			// Check result
			assert.Equal(t, tc.expectedResult, ok)
			if ok {
				assert.Equal(t, tc.requestBody, body)
			}
		})
	}
}

func TestMiddleware_HandleEncryption(t *testing.T) {
	// Setup
	logger, _ := zap.NewDevelopment()

	// Создаем конфигурацию
	config := &app.Config{
		PrivateKey: "test-key",
	}

	m := &Middleware{
		log:              logger,
		config:           config,
		processedBodyKey: ContextKey{"processed_body"},
	}

	// Test data
	plaintext := []byte("test encryption data")

	// Создаем тестовую пару ключей для шифрования/дешифрования
	privateKey, publicKey, err := generateTestRSAKeyPair()
	require.NoError(t, err, "Failed to generate test RSA key pair")

	// Шифруем тестовые данные используя формат из pkg/crypto
	encryptedData, err := encryptLargeDataForTest(publicKey.(*rsa.PublicKey), plaintext)
	require.NoError(t, err, "Failed to encrypt test data")

	// Test cases
	tests := []struct {
		name           string
		requestBody    []byte
		setupRequest   func(*http.Request)
		setupConfig    func(*app.Config)
		expectedResult bool
		expectedBody   []byte
		expectedStatus int
	}{
		{
			name:        "No encryption",
			requestBody: plaintext,
			setupRequest: func(r *http.Request) {
				// No Content-Encrypted header
			},
			setupConfig: func(c *app.Config) {
				// No special setup
			},
			expectedResult: true,
			expectedBody:   plaintext,
			expectedStatus: http.StatusOK,
		},
		{
			name:        "With encryption but no private key",
			requestBody: plaintext,
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Encrypted", "true")
			},
			setupConfig: func(c *app.Config) {
				// No private key setup
			},
			expectedResult: false,
			expectedBody:   nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "With encryption and valid private key",
			requestBody: encryptedData,
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Encrypted", "true")
			},
			setupConfig: func(c *app.Config) {
				c.PrivateRSAKey = privateKey.(*rsa.PrivateKey)
			},
			expectedResult: true,
			expectedBody:   plaintext,
			expectedStatus: http.StatusOK,
		},
		{
			name:        "With encryption but invalid encrypted data",
			requestBody: []byte("invalid-encrypted-data"),
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Encrypted", "true")
			},
			setupConfig: func(c *app.Config) {
				c.PrivateRSAKey = privateKey.(*rsa.PrivateKey)
			},
			expectedResult: false,
			expectedBody:   nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "With encryption header but empty body",
			requestBody: []byte{},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Content-Encrypted", "true")
			},
			setupConfig: func(c *app.Config) {
				c.PrivateRSAKey = privateKey.(*rsa.PrivateKey)
			},
			expectedResult: false,
			expectedBody:   nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create request and response
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			tc.setupRequest(req)
			rr := httptest.NewRecorder()

			// Setup config
			tc.setupConfig(config)

			// Call the function
			body, ok := m.handleEncryption(rr, req, tc.requestBody)

			// Check result
			assert.Equal(t, tc.expectedResult, ok)
			if ok {
				if tc.name == "With encryption and valid private key" {
					// Для зашифрованных данных проверяем, что расшифрованные данные соответствуют ожидаемым
					assert.Equal(t, tc.expectedBody, body)
				} else {
					assert.Equal(t, tc.expectedBody, body)
				}
			} else {
				assert.Equal(t, tc.expectedStatus, rr.Code)
			}
		})
	}
}

// Вспомогательная функция для генерации тестовой пары ключей RSA.
func generateTestRSAKeyPair() (privateKey, publicKey interface{}, err error) {
	// Генерируем приватный ключ
	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Получаем публичный ключ
	rsaPublicKey := &rsaPrivateKey.PublicKey

	return rsaPrivateKey, rsaPublicKey, nil
}

// Константа для RSA OAEP padding.
const RSAOAEPPadding = 2

// Вспомогательная функция для шифрования тестовых данных в формате, совместимом с pkg/crypto.
func encryptLargeDataForTest(publicKey *rsa.PublicKey, data []byte) ([]byte, error) {
	// Определяем максимальный размер данных, которые можно зашифровать за один раз
	maxChunkSize := (publicKey.Size() - 2*sha256.Size - RSAOAEPPadding)

	// Разбиваем данные на части
	var chunks [][]byte
	for i := 0; i < len(data); i += maxChunkSize {
		end := i + maxChunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}

	// Шифруем каждую часть
	var encryptedChunks [][]byte
	for _, chunk := range chunks {
		encryptedChunk, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, chunk, nil)
		if err != nil {
			return nil, err
		}
		encryptedChunks = append(encryptedChunks, encryptedChunk)
	}

	// Собираем зашифрованные данные в формате [количество частей (4 байта)][размер части 1 (4 байта)][часть 1]...
	var result bytes.Buffer

	// Записываем количество частей
	numChunks := uint32(len(encryptedChunks))
	if err := binary.Write(&result, binary.BigEndian, numChunks); err != nil {
		return nil, err
	}

	// Записываем каждую часть с её размером
	for _, chunk := range encryptedChunks {
		// Записываем размер части
		chunkSize := uint32(len(chunk))
		if err := binary.Write(&result, binary.BigEndian, chunkSize); err != nil {
			return nil, err
		}
		// Записываем саму часть
		if _, err := result.Write(chunk); err != nil {
			return nil, err
		}
	}

	return result.Bytes(), nil
}

func TestMiddleware_VerifyRequestSignature(t *testing.T) {
	// Setup
	logger, _ := zap.NewDevelopment()
	config := &app.Config{
		PrivateKey: "test-key",
	}
	m := &Middleware{
		log:              logger,
		config:           config,
		processedBodyKey: ContextKey{"processed_body"},
	}

	// Test data
	testBody := []byte("test signature data")
	validHash := sign.ComputeHMACSHA256(testBody, config.PrivateKey)

	// Test cases
	tests := []struct {
		name           string
		body           []byte
		setupRequest   func(*http.Request)
		expectedResult bool
	}{
		{
			name: "No signature",
			body: testBody,
			setupRequest: func(r *http.Request) {
				// No signature header
			},
			expectedResult: true,
		},
		{
			name: "Valid signature",
			body: testBody,
			setupRequest: func(r *http.Request) {
				r.Header.Set(sign.HeaderKey, validHash)
			},
			expectedResult: true,
		},
		{
			name: "Invalid signature",
			body: testBody,
			setupRequest: func(r *http.Request) {
				r.Header.Set(sign.HeaderKey, "invalid-hash")
			},
			expectedResult: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create request and response
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			tc.setupRequest(req)
			rr := httptest.NewRecorder()

			// Call the function
			result := m.verifyRequestSignature(rr, req, tc.body)

			// Check result
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestSignedResponseWriter_Write(t *testing.T) {
	// Setup
	privateKey := "test-key"
	rr := httptest.NewRecorder()
	writer := &signedResponseWriter{
		ResponseWriter: rr,
		privateKey:     privateKey,
	}

	// Test data
	testData := []byte("test response data")

	// Call the Write method
	n, err := writer.Write(testData)

	// Check results
	require.NoError(t, err)
	assert.Equal(t, len(testData), n)

	// Verify the hash was set in the header
	expectedHash := sign.ComputeHMACSHA256(testData, privateKey)
	assert.Equal(t, expectedHash, rr.Header().Get(sign.HeaderKey))

	// Verify the data was written to the underlying ResponseWriter
	assert.Equal(t, testData, rr.Body.Bytes())
}

func TestContextKey_String(t *testing.T) {
	key := ContextKey{"test_key"}
	expected := "crypto middleware context key: test_key"
	assert.Equal(t, expected, key.String())
}
