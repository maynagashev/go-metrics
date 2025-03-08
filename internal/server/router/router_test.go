package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a config
	config := &app.Config{
		Addr:            "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "/tmp/metrics-db.json",
		Restore:         true,
	}

	// Create a real storage implementation
	storage := memory.New(config, logger)

	// Create the router
	router := New(config, storage, logger)

	// Verify the router was created
	assert.NotNil(t, router)

	// Test some basic routes to ensure they're registered
	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "GET /",
			method:         http.MethodGet,
			path:           "/",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST /update",
			method:         http.MethodPost,
			path:           "/update",
			expectedStatus: http.StatusBadRequest, // Without a valid body, we expect a bad request
		},
		{
			name:           "POST /updates",
			method:         http.MethodPost,
			path:           "/updates",
			expectedStatus: http.StatusBadRequest, // Without a valid body, we expect a bad request
		},
		{
			name:           "POST /value",
			method:         http.MethodPost,
			path:           "/value",
			expectedStatus: http.StatusBadRequest, // Without a valid body, we expect a bad request
		},
		{
			name:           "GET /ping",
			method:         http.MethodGet,
			path:           "/ping",
			expectedStatus: http.StatusInternalServerError, // Without a valid DB connection, we expect an error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a request
			req := httptest.NewRequest(tc.method, tc.path, nil)

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, tc.expectedStatus, rr.Code)
		})
	}
}

func TestNew_WithPprof(t *testing.T) {
	// Create a logger
	logger, _ := zap.NewDevelopment()

	// Create a config with pprof enabled
	config := &app.Config{
		Addr:            "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "/tmp/metrics-db.json",
		Restore:         true,
		EnablePprof:     true,
	}

	// Create a real storage implementation
	storage := memory.New(config, logger)

	// Create the router
	router := New(config, storage, logger)

	// Verify the router was created
	assert.NotNil(t, router)

	// Test the pprof routes
	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "GET /debug/pprof/",
			method:         http.MethodGet,
			path:           "/debug/pprof/",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET /debug/pprof/cmdline",
			method:         http.MethodGet,
			path:           "/debug/pprof/cmdline",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET /debug/pprof/heap",
			method:         http.MethodGet,
			path:           "/debug/pprof/heap",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a request
			req := httptest.NewRequest(tc.method, tc.path, nil)

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, tc.expectedStatus, rr.Code)
		})
	}
}
