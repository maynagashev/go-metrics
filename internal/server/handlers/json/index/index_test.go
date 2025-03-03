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
	// Create a logger and config
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	cfg := &app.Config{}

	// Create a storage with some test metrics
	storage := memory.New(cfg, logger)
	ctx := context.Background()

	// Add a gauge metric
	gaugeValue := 42.5
	gaugeMetric := metrics.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
		Value: &gaugeValue,
	}
	err = storage.UpdateMetric(ctx, gaugeMetric)
	require.NoError(t, err)

	// Add a counter metric
	counterValue := int64(100)
	counterMetric := metrics.Metric{
		Name:  "test_counter",
		MType: metrics.TypeCounter,
		Delta: &counterValue,
	}
	err = storage.UpdateMetric(ctx, counterMetric)
	require.NoError(t, err)

	// Create the handler
	handler := index.New(storage)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check content type
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Parse response
	var responseMetrics []metrics.Metric
	err = json.Unmarshal(rr.Body.Bytes(), &responseMetrics)
	require.NoError(t, err)

	// Check that we have at least our two metrics
	assert.GreaterOrEqual(t, len(responseMetrics), 2)

	// Check that our metrics are in the response
	foundGauge := false
	foundCounter := false
	for _, m := range responseMetrics {
		if m.Name == gaugeMetric.Name && m.MType == gaugeMetric.MType {
			foundGauge = true
			assert.Equal(t, *gaugeMetric.Value, *m.Value)
		}
		if m.Name == counterMetric.Name && m.MType == counterMetric.MType {
			foundCounter = true
			assert.Equal(t, *counterMetric.Delta, *m.Delta)
		}
	}
	assert.True(t, foundGauge, "Gauge metric not found in response")
	assert.True(t, foundCounter, "Counter metric not found in response")
}

func TestJSONIndexHandler_EmptyStorage(t *testing.T) {
	// Create a logger and config
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	cfg := &app.Config{}

	// Create an empty storage
	storage := memory.New(cfg, logger)

	// Create the handler
	handler := index.New(storage)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check content type
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Parse response
	var responseMetrics []metrics.Metric
	err = json.Unmarshal(rr.Body.Bytes(), &responseMetrics)
	require.NoError(t, err)

	// Check that we have an empty array
	assert.Empty(t, responseMetrics)
}
