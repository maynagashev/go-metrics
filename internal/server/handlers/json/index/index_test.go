package index_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/handlers/json/index"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestJSONIndexHandler(t *testing.T) {
	// Create a test storage with some test metrics
	storage := setupTestStorage(t)

	// Create the handler
	handler := index.New(storage)

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the content type
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Parse the response
	var metricsResponse []metrics.Metric
	err := json.Unmarshal(rr.Body.Bytes(), &metricsResponse)
	require.NoError(t, err)

	// Check that we got the expected metrics
	assert.Len(t, metricsResponse, 2)

	// Find and check the gauge metric
	gaugeValue := 42.0
	var gaugeMetric = &metrics.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
		Value: &gaugeValue,
	}

	foundGauge := false
	for _, m := range metricsResponse {
		if m.Name == "test_gauge" && m.MType == metrics.TypeGauge {
			foundGauge = true
			assert.NotNil(t, m.Value)
			assert.InDelta(t, *gaugeMetric.Value, *m.Value, 0.0001)
		}
	}
	assert.True(t, foundGauge, "Gauge metric not found in response")

	// Find and check the counter metric
	counterValue := int64(10)
	var counterMetric = &metrics.Metric{
		Name:  "test_counter",
		MType: metrics.TypeCounter,
		Delta: &counterValue,
	}

	foundCounter := false
	for _, m := range metricsResponse {
		if m.Name == "test_counter" && m.MType == metrics.TypeCounter {
			foundCounter = true
			assert.NotNil(t, m.Delta)
			assert.Equal(t, *counterMetric.Delta, *m.Delta)
		}
	}
	assert.True(t, foundCounter, "Counter metric not found in response")
}

func TestJSONIndexHandler_EmptyStorage(t *testing.T) {
	// Setup
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Create an empty storage
	storage := memory.New(&app.Config{}, logger)

	// Create the handler
	handler := index.New(storage)

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the content type
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Parse the response
	var metricsResponse []metrics.Metric
	err = json.Unmarshal(rr.Body.Bytes(), &metricsResponse)
	require.NoError(t, err)

	// Check that we got an empty array
	assert.Empty(t, metricsResponse)
}

// setupTestStorage creates a test storage with some test metrics
func setupTestStorage(t *testing.T) *memory.MemStorage {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := memory.New(&app.Config{}, logger)
	ctx := context.Background()

	// Add a gauge metric
	gaugeValue := 42.0
	gaugeMetric := &metrics.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
		Value: &gaugeValue,
	}
	err = storage.UpdateMetric(ctx, *gaugeMetric)
	require.NoError(t, err)

	// Add a counter metric
	counterValue := int64(10)
	counterMetric := &metrics.Metric{
		Name:  "test_counter",
		MType: metrics.TypeCounter,
		Delta: &counterValue,
	}
	err = storage.UpdateMetric(ctx, *counterMetric)
	require.NoError(t, err)

	return storage
}
