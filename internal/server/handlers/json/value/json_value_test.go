package value_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/handlers/json/value"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
	"github.com/maynagashev/go-metrics/pkg/sign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestJSONValueHandler(t *testing.T) {
	// Setup
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	tests := []struct {
		name           string
		requestMetric  metrics.Metric
		expectedMetric *metrics.Metric
		expectedStatus int
	}{
		{
			name: "Valid gauge metric",
			requestMetric: metrics.Metric{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
			},
			expectedMetric: &metrics.Metric{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
				Value: metrics.FloatPtr(42.0),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Valid counter metric",
			requestMetric: metrics.Metric{
				Name:  "test_counter",
				MType: metrics.TypeCounter,
			},
			expectedMetric: &metrics.Metric{
				Name:  "test_counter",
				MType: metrics.TypeCounter,
				Delta: metrics.Int64Ptr(10),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Non-existent metric",
			requestMetric: metrics.Metric{
				Name:  "non_existent",
				MType: metrics.TypeGauge,
			},
			expectedMetric: nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Invalid metric type",
			requestMetric: metrics.Metric{
				Name:  "test_gauge",
				MType: "invalid",
			},
			expectedMetric: nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing metric name",
			requestMetric: metrics.Metric{
				MType: metrics.TypeGauge,
			},
			expectedMetric: nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing metric type",
			requestMetric: metrics.Metric{
				Name: "test_gauge",
			},
			expectedMetric: nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test storage with test metrics
			storage := setupTestStorage(t)
			ctx := context.Background()

			// Create the handler
			handler := value.New(storage, logger)

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
	storage := setupTestStorage(t)

	// Create the handler
	handler := value.New(storage, logger)

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
	// Setup
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Create a test storage with a gauge metric
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create a gauge metric request
	gaugeValue := 42.0
	gaugeMetric := metrics.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
	}

	// Create the handler
	handler := value.New(storage, logger)

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

// setupTestStorage creates a test storage with some test metrics
func setupTestStorage(t *testing.T) *memory.MemStorage {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := memory.New(nil, logger)
	ctx := context.Background()

	// Add a gauge metric
	gaugeMetric := metrics.NewGauge("test_gauge", 42.0)
	err = storage.UpdateMetric(ctx, *gaugeMetric)
	require.NoError(t, err)

	// Add a counter metric
	counterMetric := metrics.NewCounter("test_counter", 10)
	err = storage.UpdateMetric(ctx, *counterMetric)
	require.NoError(t, err)

	return storage
}

func TestJSONValueHandler_WithSigning(t *testing.T) {
	// Create a logger and config
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Create config with signing
	privateKey := "test-key"
	cfg := &app.Config{
		PrivateKey: privateKey,
	}

	// Create a storage with a test metric
	storage := memory.New(cfg, logger)
	ctx := context.Background()

	// Add a gauge metric
	gaugeValue := 42.5
	gaugeName := "test_gauge"
	gaugeMetric := metrics.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
		Value: &gaugeValue,
	}
	err = storage.UpdateMetric(ctx, gaugeMetric)
	require.NoError(t, err)

	// Create the handler
	handler := value.New(cfg, storage)

	// Create request body
	requestMetric := metrics.Metric{
		Name:  gaugeName,
		MType: metrics.TypeGauge,
	}
	requestBody, err := json.Marshal(requestMetric)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Parse response
	var responseMetric metrics.Metric
	err = json.Unmarshal(rr.Body.Bytes(), &responseMetric)
	require.NoError(t, err)

	// Check response
	assert.Equal(t, gaugeMetric.Name, responseMetric.Name)
	assert.Equal(t, gaugeMetric.MType, responseMetric.MType)
	assert.Equal(t, *gaugeMetric.Value, *responseMetric.Value)

	// Check signature
	signature := rr.Header().Get(sign.HeaderKey)
	assert.NotEmpty(t, signature)

	// Verify signature
	expectedSignature := sign.ComputeHMACSHA256(rr.Body.Bytes(), privateKey)
	assert.Equal(t, expectedSignature, signature)
}
