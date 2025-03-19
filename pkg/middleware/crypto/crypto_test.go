package crypto_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/pkg/middleware/crypto"
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
			middleware := crypto.New(tc.config, logger)

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
