package decompresspool

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDecompressPoolMiddleware_WithGzip(t *testing.T) {
	// Create a test logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Original data to be compressed
	originalData := []byte(`{"key": "value"}`)

	// Compress the data
	var compressedData bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedData)
	_, err = gzipWriter.Write(originalData)
	require.NoError(t, err)
	require.NoError(t, gzipWriter.Close())

	// Create a test request with compressed data
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(compressedData.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a test handler that reads the request body
	var receivedData []byte
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedData, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	})

	// Create and use the middleware
	middleware := New(logger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check that the decompressed body matches the original data
	assert.Equal(t, originalData, receivedData)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDecompressPoolMiddleware_WithoutGzip(t *testing.T) {
	// Create a test logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Original data
	originalData := []byte(`{"key": "value"}`)

	// Create a test request with uncompressed data
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(originalData))

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a test handler that reads the request body
	var receivedData []byte
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedData, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	})

	// Create and use the middleware
	middleware := New(logger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check that the body matches the original data
	assert.Equal(t, originalData, receivedData)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDecompressPoolMiddleware_InvalidGzip(t *testing.T) {
	// Create a test logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Invalid gzip data
	invalidData := []byte(`not a valid gzip data`)

	// Create a test request with invalid gzip data
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(invalidData))
	req.Header.Set("Content-Encoding", "gzip")

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a test handler that should not be called
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create and use the middleware
	middleware := New(logger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check that we got a bad request response and the handler was not called
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.False(t, handlerCalled)
}

// TestDecompressPoolMiddleware_ReaderPoolError tests the case when getting a reader from the pool fails
func TestDecompressPoolMiddleware_ReaderPoolError(t *testing.T) {
	// Create a test logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Create a middleware with a broken reader pool
	m := &Middleware{
		log: logger,
		readerPool: sync.Pool{
			New: func() interface{} {
				return "not a gzip.Reader" // This will cause a type assertion error
			},
		},
		closerPool: sync.Pool{
			New: func() interface{} {
				return new(gzipReadCloser)
			},
		},
	}

	// Create a test request with gzip header
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("test")))
	req.Header.Set("Content-Encoding", "gzip")

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a test handler that should not be called
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Use the middleware
	handler := m.Handler(testHandler)
	handler.ServeHTTP(rr, req)

	// Check that we got an internal server error and the handler was not called
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.False(t, handlerCalled)
}

// TestDecompressPoolMiddleware_CloserPoolError tests the case when getting a closer from the pool fails
func TestDecompressPoolMiddleware_CloserPoolError(t *testing.T) {
	// Create a test logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Original data to be compressed
	originalData := []byte(`{"key": "value"}`)

	// Compress the data
	var compressedData bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedData)
	_, err = gzipWriter.Write(originalData)
	require.NoError(t, err)
	require.NoError(t, gzipWriter.Close())

	// Create a middleware with a broken closer pool
	m := &Middleware{
		log: logger,
		readerPool: sync.Pool{
			New: func() interface{} {
				return new(gzip.Reader)
			},
		},
		closerPool: sync.Pool{
			New: func() interface{} {
				return "not a gzipReadCloser" // This will cause a type assertion error
			},
		},
	}

	// Create a test request with gzip header
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(compressedData.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a test handler that should not be called
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Use the middleware
	handler := m.Handler(testHandler)
	handler.ServeHTTP(rr, req)

	// Check that we got an internal server error and the handler was not called
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.False(t, handlerCalled)
}
