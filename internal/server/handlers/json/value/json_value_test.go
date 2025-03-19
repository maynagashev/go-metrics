package value_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/handlers/json/value"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
)

// Helper functions.
func FloatPtr(v float64) *float64 {
	return &v
}

func Int64Ptr(v int64) *int64 {
	return &v
}

func TestJSONValueHandler(t *testing.T) {
	tests := []struct {
		name           string
		requestMetric  metrics.Metric
		expectedMetric *metrics.Metric
		expectedStatus int
		setupMetric    *metrics.Metric
	}{
		{
			name: "Get gauge metric",
			requestMetric: metrics.Metric{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
			},
			expectedMetric: &metrics.Metric{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
				Value: FloatPtr(42.0),
			},
			expectedStatus: http.StatusOK,
			setupMetric: &metrics.Metric{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
				Value: FloatPtr(42.0),
			},
		},
		{
			name: "Get counter metric",
			requestMetric: metrics.Metric{
				Name:  "test_counter",
				MType: metrics.TypeCounter,
			},
			expectedMetric: &metrics.Metric{
				Name:  "test_counter",
				MType: metrics.TypeCounter,
				Delta: Int64Ptr(10),
			},
			expectedStatus: http.StatusOK,
			setupMetric: &metrics.Metric{
				Name:  "test_counter",
				MType: metrics.TypeCounter,
				Delta: Int64Ptr(10),
			},
		},
		{
			name: "Metric not found",
			requestMetric: metrics.Metric{
				Name:  "non_existent_metric",
				MType: metrics.TypeGauge,
			},
			expectedMetric: nil,
			expectedStatus: http.StatusNotFound,
			setupMetric:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			logger, err := zap.NewDevelopment()
			require.NoError(t, err)

			// Create a test storage
			cfg := &app.Config{}
			storage := memory.New(cfg, logger)

			// Add the test metric to the storage if needed
			if tt.setupMetric != nil {
				err = storage.UpdateMetric(context.Background(), *tt.setupMetric)
				require.NoError(t, err)
			}

			// Create the handler
			handler := value.New(cfg, storage)

			// Create a request
			requestJSON, marshalErr := json.Marshal(tt.requestMetric)
			require.NoError(t, marshalErr)

			req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(requestJSON))
			req.Header.Set("Content-Type", "application/json")

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// If we expect a successful response, check the response body
			if tt.expectedStatus == http.StatusOK {
				var responseMetric metrics.Metric
				decodeErr := json.NewDecoder(rr.Body).Decode(&responseMetric)
				require.NoError(t, decodeErr)

				assert.Equal(t, tt.expectedMetric.Name, responseMetric.Name)
				assert.Equal(t, tt.expectedMetric.MType, responseMetric.MType)

				if tt.expectedMetric.Value != nil {
					require.NotNil(t, responseMetric.Value)
					assert.InDelta(t, *tt.expectedMetric.Value, *responseMetric.Value, 0.0001)
				}

				if tt.expectedMetric.Delta != nil {
					require.NotNil(t, responseMetric.Delta)
					assert.Equal(t, *tt.expectedMetric.Delta, *responseMetric.Delta)
				}
			}
		})
	}
}

func TestJSONValueHandler_InvalidJSON(t *testing.T) {
	// Setup
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Create a test storage
	cfg := &app.Config{}
	storage := memory.New(cfg, logger)

	// Create the handler
	handler := value.New(cfg, storage)

	// Create a request with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader([]byte(`{invalid json}`)))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestJSONValueHandler_GaugeMetric(t *testing.T) {
	// Create a test storage with a gauge metric
	storage := setupTestStorage(t)

	// Create a gauge metric request
	gaugeValue := 42.0
	gaugeMetric := metrics.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
	}

	// Create the handler
	cfg := &app.Config{}
	handler := value.New(cfg, storage)

	// Create a request
	requestJSON, marshalErr := json.Marshal(gaugeMetric)
	require.NoError(t, marshalErr)

	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(requestJSON))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body
	var responseMetric metrics.Metric
	decodeErr := json.NewDecoder(rr.Body).Decode(&responseMetric)
	require.NoError(t, decodeErr)

	assert.Equal(t, gaugeMetric.Name, responseMetric.Name)
	assert.Equal(t, gaugeMetric.MType, responseMetric.MType)
	require.NotNil(t, responseMetric.Value)
	assert.InDelta(t, gaugeValue, *responseMetric.Value, 0.0001)
}

// setupTestStorage creates a test storage with some test metrics.
func setupTestStorage(t *testing.T) *memory.MemStorage {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	cfg := &app.Config{}
	storage := memory.New(cfg, logger)
	ctx := context.Background()

	// Add a gauge metric
	gaugeValue := 42.0
	gaugeMetric := &metrics.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
		Value: FloatPtr(gaugeValue),
	}
	err = storage.UpdateMetric(ctx, *gaugeMetric)
	require.NoError(t, err)

	// Add a counter metric
	counterValue := int64(10)
	counterMetric := &metrics.Metric{
		Name:  "test_counter",
		MType: metrics.TypeCounter,
		Delta: Int64Ptr(counterValue),
	}
	err = storage.UpdateMetric(ctx, *counterMetric)
	require.NoError(t, err)

	return storage
}

func TestJSONValueHandler_MetricNotFound(t *testing.T) {
	// Setup
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Create a test storage
	cfg := &app.Config{}
	storage := memory.New(cfg, logger)

	// Create the handler
	handler := value.New(cfg, storage)

	// Create a request
	requestMetric := metrics.Metric{
		Name:  "non_existent_metric",
		MType: metrics.TypeGauge,
	}
	requestJSON, marshalErr := json.Marshal(requestMetric)
	require.NoError(t, marshalErr)

	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(requestJSON))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestJSONValueHandler_WithSignature(t *testing.T) {
	// Setup
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Create a test storage with a key for signature verification
	cfg := &app.Config{
		PrivateKey: "test-key",
	}
	storage := memory.New(cfg, logger)

	// Add a test metric to the storage
	gaugeValue := 42.0
	gaugeMetric := metrics.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
		Value: FloatPtr(gaugeValue),
	}
	err = storage.UpdateMetric(context.Background(), gaugeMetric)
	require.NoError(t, err)

	// Create the handler
	handler := value.New(cfg, storage)

	// Create a request
	requestMetric := metrics.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
	}
	requestJSON, marshalErr := json.Marshal(requestMetric)
	require.NoError(t, marshalErr)

	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(requestJSON))
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)
}
