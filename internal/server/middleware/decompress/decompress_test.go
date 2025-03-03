package decompress_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maynagashev/go-metrics/internal/server/middleware/decompress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// compressData compresses the input data using gzip
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil, err
	}

	if err = gzipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func TestDecompressMiddleware_WithGzip(t *testing.T) {
	// Create a logger for testing
	logger := zaptest.NewLogger(t)

	// Prepare test data
	testData := []byte(`{"id":"test","type":"gauge","value":123.45}`)

	// Compress the data
	compressedData, compressErr := compressData(testData)
	require.NoError(t, compressErr)

	// Create a test request with compressed data
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(compressedData))
	req.Header.Set("Content-Encoding", "gzip")

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a test handler that checks if the body was decompressed
	var decompressedBody []byte
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the decompressed body
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusInternalServerError)
			return
		}
		decompressedBody = body

		w.WriteHeader(http.StatusOK)
	})

	// Create and use the middleware
	middleware := decompress.New(logger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check that the body was correctly decompressed
	assert.Equal(t, testData, decompressedBody)
}

func TestDecompressMiddleware_WithoutGzip(t *testing.T) {
	// Create a logger for testing
	logger := zaptest.NewLogger(t)

	// Prepare test data
	testData := []byte(`{"id":"test","type":"gauge","value":123.45}`)

	// Create a test request with uncompressed data
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(testData))

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a test handler that checks if the body is unchanged
	var receivedBody []byte
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the body
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusInternalServerError)
			return
		}
		receivedBody = body

		w.WriteHeader(http.StatusOK)
	})

	// Create and use the middleware
	middleware := decompress.New(logger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check that the body is unchanged
	assert.Equal(t, testData, receivedBody)
}

func TestDecompressMiddleware_InvalidGzip(t *testing.T) {
	// Create a logger for testing
	logger := zaptest.NewLogger(t)

	// Prepare invalid gzip data
	invalidData := []byte(`not a valid gzip data`)

	// Create a test request with invalid compressed data
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(invalidData))
	req.Header.Set("Content-Encoding", "gzip")

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a test handler that should not be called
	handlerCalled := false
	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
	})

	// Create and use the middleware
	middleware := decompress.New(logger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Check that the handler was not called
	assert.False(t, handlerCalled)
}
