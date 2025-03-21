package decompresspool_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/maynagashev/go-metrics/internal/server/middleware/decompresspool"
)

func TestDecompressPoolMiddleware_WithGzip(t *testing.T) {
	// Create a logger for testing
	logger := zaptest.NewLogger(t)

	// Prepare test data
	testData := []byte(`{"id":"test","type":"gauge","value":123.45}`)

	// Compress the data
	compressedData := bytes.NewBuffer(nil)
	compressErr := func() error {
		gzipWriter := gzip.NewWriter(compressedData)
		defer gzipWriter.Close()
		_, err := gzipWriter.Write(testData)
		if err != nil {
			return err
		}
		return gzipWriter.Close()
	}()
	require.NoError(t, compressErr)

	// Create a test request with compressed data
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(compressedData.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a test handler that checks if the body was decompressed
	var decompressedBody []byte
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the decompressed body
		var readErr error
		decompressedBody, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	// Create and use the middleware
	middleware := decompresspool.New(logger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check that the body was correctly decompressed
	assert.Equal(t, testData, decompressedBody)
}

func TestDecompressPoolMiddleware_WithoutGzip(t *testing.T) {
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
		var readErr error
		receivedBody, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	// Create and use the middleware
	middleware := decompresspool.New(logger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check that the body is unchanged
	assert.Equal(t, testData, receivedBody)
}

func TestDecompressPoolMiddleware_InvalidGzip(t *testing.T) {
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
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create and use the middleware
	middleware := decompresspool.New(logger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Check that the handler was not called
	assert.False(t, handlerCalled)
}

func TestDecompressPoolMiddleware_CloseMethod(t *testing.T) {
	// Create a logger for testing
	logger := zaptest.NewLogger(t)

	// Prepare test data
	testData := []byte(`{"id":"test","type":"gauge","value":123.45}`)

	// Compress the data
	compressedData := bytes.NewBuffer(nil)
	compressErr := func() error {
		gzipWriter := gzip.NewWriter(compressedData)
		defer gzipWriter.Close()
		_, err := gzipWriter.Write(testData)
		if err != nil {
			return err
		}
		return gzipWriter.Close()
	}()
	require.NoError(t, compressErr)

	// Create a test request with compressed data
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(compressedData.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Variable to track if body was closed
	bodyClosed := false

	// Create a test handler that explicitly closes the body
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read all data to ensure it's working
		data, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		assert.Equal(t, testData, data)

		// Close the body explicitly to test Close method
		err = r.Body.Close()
		assert.NoError(t, err)
		bodyClosed = true

		w.WriteHeader(http.StatusOK)
	})

	// Create and use the middleware
	middleware := decompresspool.New(logger)
	handler := middleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify body was closed
	assert.True(t, bodyClosed, "Body should have been closed")
}

// Создаем тест с имитацией ошибок при закрытии базового Reader.
type errorReadCloser struct {
	reader io.Reader
}

func (e *errorReadCloser) Read(p []byte) (int, error) {
	return e.reader.Read(p)
}

func (e *errorReadCloser) Close() error {
	return assert.AnError // возвращаем ошибку при закрытии
}

func TestDecompressPoolMiddleware_CloseWithError(t *testing.T) {
	// Create a logger for testing
	logger := zaptest.NewLogger(t)

	// Prepare test data
	testData := []byte(`{"id":"test","type":"gauge","value":123.45}`)

	// Compress the data
	compressedData := bytes.NewBuffer(nil)
	compressErr := func() error {
		gzipWriter := gzip.NewWriter(compressedData)
		defer gzipWriter.Close()
		_, err := gzipWriter.Write(testData)
		if err != nil {
			return err
		}
		return gzipWriter.Close()
	}()
	require.NoError(t, compressErr)

	// Create a test request with compressed data wrapped in errorReadCloser
	// This will cause an error when the original body is closed
	mockBody := &errorReadCloser{reader: bytes.NewReader(compressedData.Bytes())}
	req := httptest.NewRequest(http.MethodPost, "/update", mockBody)
	req.Header.Set("Content-Encoding", "gzip")

	// Create a test response recorder
	rr := httptest.NewRecorder()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the data first
		data, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		assert.Equal(t, testData, data)

		// Then try to close
		err = r.Body.Close()
		assert.Error(t, err)

		w.WriteHeader(http.StatusOK)
	})

	// Create and use the middleware
	middleware := decompresspool.New(logger)
	handler := middleware(testHandler)

	// This should not panic even with error in Close()
	handler.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusOK, rr.Code)
}
