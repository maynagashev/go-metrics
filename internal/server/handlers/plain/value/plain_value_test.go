package value_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/handlers/plain/value"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPlainValueHandler(t *testing.T) {
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
	handler := value.New(storage)

	tests := []struct {
		name           string
		metricType     string
		metricName     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Get existing gauge metric",
			metricType:     "gauge",
			metricName:     gaugeName,
			expectedStatus: http.StatusOK,
			expectedBody:   "42.5",
		},
		{
			name:           "Get existing counter metric",
			metricType:     "counter",
			metricName:     counterName,
			expectedStatus: http.StatusOK,
			expectedBody:   "100",
		},
		{
			name:           "Get non-existent metric",
			metricType:     "gauge",
			metricName:     "non_existent",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new router for each test
			r := chi.NewRouter()
			r.Get("/value/{type}/{name}", handler)

			// Create request
			req := httptest.NewRequest(
				http.MethodGet,
				"/value/"+tt.metricType+"/"+tt.metricName,
				nil,
			)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			r.ServeHTTP(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Check response body for successful requests
			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedBody, rr.Body.String())
			}
		})
	}
}
