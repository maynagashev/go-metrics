package logger_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maynagashev/go-metrics/internal/server/middleware/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerMiddleware(t *testing.T) {
	// Create a logger that records logs for testing
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)

	// Create test data
	testData := map[string]string{"test": "data"}
	requestBody, jsonErr := json.Marshal(testData)
	if jsonErr != nil {
		t.Fatalf("Failed to marshal test data: %v", jsonErr)
	}

	// Create a test request
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	})

	// Create and use the middleware
	middleware := logger.New(observedLogger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())

	// Check that the logs contain the expected entries
	logs := observedLogs.All()
	assert.GreaterOrEqual(t, len(logs), 2, "Expected at least 2 log entries")

	// Check for the middleware enabled log
	foundEnabledLog := false
	for _, log := range logs {
		if log.Message == "logger middleware enabled" {
			foundEnabledLog = true
			break
		}
	}
	assert.True(t, foundEnabledLog, "Expected 'logger middleware enabled' log entry")

	// Check for the request completed log
	foundCompletedLog := false
	for _, log := range logs {
		if log.Message == "request completed" {
			foundCompletedLog = true
			assert.Equal(t, int64(http.StatusOK), log.ContextMap()["status"])
			break
		}
	}
	assert.True(t, foundCompletedLog, "Expected 'request completed' log entry")
}

func TestLoggerMiddleware_WithRequestBody(t *testing.T) {
	// Create a logger that records logs for testing
	testLogger := zaptest.NewLogger(t)

	// Create test data
	testData := map[string]string{"test": "data"}
	requestBody, jsonErr := json.Marshal(testData)
	if jsonErr != nil {
		t.Fatalf("Failed to marshal test data: %v", jsonErr)
	}

	// Create a test request
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create and use the middleware
	middleware := logger.New(testLogger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusOK, rr.Code)
}
