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
	// Create a logger and config
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	cfg := &app.Config{}

	// Create a storage with some test metrics
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

	// Add a counter metric
	counterValue := int64(100)
	counterName := "test_counter"
	counterMetric := metrics.Metric{
		Name:  "test_counter",
		MType: metrics.TypeCounter,
		Delta: &counterValue,
	}
	err = storage.UpdateMetric(ctx, counterMetric)
	require.NoError(t, err)

	// Create the handler
	handler := value.New(cfg, storage)

	tests := []struct {
		name           string
		requestMetric  metrics.Metric
		expectedStatus int
		expectedMetric metrics.Metric
		expectError    bool
	}{
		{
			name: "Get existing gauge metric",
			requestMetric: metrics.Metric{
				Name:  gaugeName,
				MType: metrics.TypeGauge,
			},
			expectedStatus: http.StatusOK,
			expectedMetric: gaugeMetric,
			expectError:    false,
		},
		{
			name: "Get existing counter metric",
			requestMetric: metrics.Metric{
				Name:  counterName,
				MType: metrics.TypeCounter,
			},
			expectedStatus: http.StatusOK,
			expectedMetric: counterMetric,
			expectError:    false,
		},
		{
			name: "Get non-existent metric",
			requestMetric: metrics.Metric{
				Name:  "non_existent",
				MType: metrics.TypeGauge,
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			requestBody, err := json.Marshal(tt.requestMetric)
			require.NoError(t, err)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler.ServeHTTP(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if !tt.expectError {
				// Parse response
				var responseMetric metrics.Metric
				err = json.Unmarshal(rr.Body.Bytes(), &responseMetric)
				require.NoError(t, err)

				// Check response
				if tt.expectedMetric.MType == metrics.TypeGauge {
					assert.Equal(t, tt.expectedMetric.Name, responseMetric.Name)
					assert.Equal(t, tt.expectedMetric.MType, responseMetric.MType)
					assert.Equal(t, *tt.expectedMetric.Value, *responseMetric.Value)
				} else {
					assert.Equal(t, tt.expectedMetric.Name, responseMetric.Name)
					assert.Equal(t, tt.expectedMetric.MType, responseMetric.MType)
					assert.Equal(t, *tt.expectedMetric.Delta, *responseMetric.Delta)
				}
			}
		})
	}
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

func TestJSONValueHandler_InvalidJSON(t *testing.T) {
	// Create a logger and config
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	cfg := &app.Config{}

	// Create storage
	storage := memory.New(cfg, logger)

	// Create the handler
	handler := value.New(cfg, storage)

	// Create invalid JSON request
	invalidJSON := []byte(`{"id": "test", "type": "gauge"`) // Missing closing brace

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check status code
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
