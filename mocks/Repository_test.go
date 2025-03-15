package mocks_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"github.com/maynagashev/go-metrics/mocks"
)

func TestRepository_Close(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		mockErr error
	}{
		{
			name:    "success",
			wantErr: false,
			mockErr: nil,
		},
		{
			name:    "error",
			wantErr: true,
			mockErr: errors.New("test error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new mock repository
			r := mocks.NewRepository(t)

			// Setup expectations
			r.On("Close").Return(tt.mockErr)

			// Call the method
			err := r.Close()

			// Assert expectations
			r.AssertExpectations(t)

			// Check the result
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.mockErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRepository_Count(t *testing.T) {
	tests := []struct {
		name      string
		mockCount int
	}{
		{
			name:      "empty repository",
			mockCount: 0,
		},
		{
			name:      "non-empty repository",
			mockCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new mock repository
			r := mocks.NewRepository(t)

			// Setup expectations
			r.On("Count").Return(tt.mockCount)

			// Call the method
			count := r.Count()

			// Assert expectations
			r.AssertExpectations(t)

			// Check the result
			assert.Equal(t, tt.mockCount, count)
		})
	}
}

func TestRepository_GetCounter(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		mockValue  storage.Counter
		mockExists bool
	}{
		{
			name:       "counter exists",
			metricName: "test_counter",
			mockValue:  storage.Counter(42),
			mockExists: true,
		},
		{
			name:       "counter does not exist",
			metricName: "non_existent_counter",
			mockValue:  storage.Counter(0),
			mockExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewRepository(t)
			repo.On("GetCounter", tt.metricName).Return(tt.mockValue, tt.mockExists)

			value, exists := repo.GetCounter(tt.metricName)

			assert.Equal(t, tt.mockValue, value)
			assert.Equal(t, tt.mockExists, exists)
			repo.AssertExpectations(t)
		})
	}
}

func TestRepository_GetGauge(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		mockValue  storage.Gauge
		mockExists bool
	}{
		{
			name:       "gauge exists",
			metricName: "test_gauge",
			mockValue:  storage.Gauge(3.14),
			mockExists: true,
		},
		{
			name:       "gauge does not exist",
			metricName: "non_existent_gauge",
			mockValue:  storage.Gauge(0),
			mockExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewRepository(t)
			repo.On("GetGauge", tt.metricName).Return(tt.mockValue, tt.mockExists)

			value, exists := repo.GetGauge(tt.metricName)

			assert.InDelta(t, float64(tt.mockValue), float64(value), 0.0001)
			assert.Equal(t, tt.mockExists, exists)
			repo.AssertExpectations(t)
		})
	}
}

func TestRepository_GetMetric(t *testing.T) {
	tests := []struct {
		name       string
		metricType metrics.MetricType
		metricName string
		mockMetric metrics.Metric
		mockExists bool
	}{
		{
			name:       "gauge metric exists",
			metricType: metrics.TypeGauge,
			metricName: "test_gauge",
			mockMetric: metrics.Metric{
				Name:  "test_gauge",
				MType: metrics.TypeGauge,
				Value: func() *float64 { v := 3.14; return &v }(),
			},
			mockExists: true,
		},
		{
			name:       "counter metric exists",
			metricType: metrics.TypeCounter,
			metricName: "test_counter",
			mockMetric: metrics.Metric{
				Name:  "test_counter",
				MType: metrics.TypeCounter,
				Delta: func() *int64 { v := int64(42); return &v }(),
			},
			mockExists: true,
		},
		{
			name:       "metric does not exist",
			metricType: metrics.TypeGauge,
			metricName: "non_existent_metric",
			mockMetric: metrics.Metric{},
			mockExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewRepository(t)
			repo.On("GetMetric", tt.metricType, tt.metricName).Return(tt.mockMetric, tt.mockExists)

			metric, exists := repo.GetMetric(tt.metricType, tt.metricName)

			assert.Equal(t, tt.mockMetric, metric)
			assert.Equal(t, tt.mockExists, exists)
			repo.AssertExpectations(t)
		})
	}
}

func TestRepository_GetMetrics(t *testing.T) {
	tests := []struct {
		name        string
		mockMetrics []metrics.Metric
	}{
		{
			name:        "empty repository",
			mockMetrics: []metrics.Metric{},
		},
		{
			name: "repository with metrics",
			mockMetrics: []metrics.Metric{
				{
					Name:  "gauge1",
					MType: metrics.TypeGauge,
					Value: func() *float64 { v := 1.1; return &v }(),
				},
				{
					Name:  "counter1",
					MType: metrics.TypeCounter,
					Delta: func() *int64 { v := int64(10); return &v }(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewRepository(t)
			repo.On("GetMetrics").Return(tt.mockMetrics)

			metrics := repo.GetMetrics()

			assert.Equal(t, tt.mockMetrics, metrics)
			repo.AssertExpectations(t)
		})
	}
}

func TestRepository_UpdateMetric(t *testing.T) {
	tests := []struct {
		name       string
		metric     metrics.Metric
		mockErr    error
		shouldFail bool
	}{
		{
			name: "update gauge success",
			metric: metrics.Metric{
				Name:  "gauge1",
				MType: metrics.TypeGauge,
				Value: func() *float64 { v := 1.1; return &v }(),
			},
			mockErr:    nil,
			shouldFail: false,
		},
		{
			name: "update counter success",
			metric: metrics.Metric{
				Name:  "counter1",
				MType: metrics.TypeCounter,
				Delta: func() *int64 { v := int64(10); return &v }(),
			},
			mockErr:    nil,
			shouldFail: false,
		},
		{
			name: "update metric failure",
			metric: metrics.Metric{
				Name:  "invalid",
				MType: "invalid",
			},
			mockErr:    errors.New("invalid metric type"),
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewRepository(t)
			repo.On("UpdateMetric", tt.metric).Return(tt.mockErr)

			err := repo.UpdateMetric(tt.metric)

			if tt.shouldFail {
				require.Error(t, err)
				assert.Equal(t, tt.mockErr, err)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestRepository_UpdateMetrics(t *testing.T) {
	tests := []struct {
		name       string
		metrics    []metrics.Metric
		mockErr    error
		shouldFail bool
	}{
		{
			name:       "update empty metrics",
			metrics:    []metrics.Metric{},
			mockErr:    nil,
			shouldFail: false,
		},
		{
			name: "update multiple metrics success",
			metrics: []metrics.Metric{
				{
					Name:  "gauge1",
					MType: metrics.TypeGauge,
					Value: func() *float64 { v := 1.1; return &v }(),
				},
				{
					Name:  "counter1",
					MType: metrics.TypeCounter,
					Delta: func() *int64 { v := int64(10); return &v }(),
				},
			},
			mockErr:    nil,
			shouldFail: false,
		},
		{
			name: "update metrics failure",
			metrics: []metrics.Metric{
				{
					Name:  "invalid",
					MType: "invalid",
				},
			},
			mockErr:    errors.New("invalid metric type"),
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewRepository(t)
			repo.On("UpdateMetrics", tt.metrics).Return(tt.mockErr)

			err := repo.UpdateMetrics(tt.metrics)

			if tt.shouldFail {
				require.Error(t, err)
				assert.Equal(t, tt.mockErr, err)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestNewRepository(t *testing.T) {
	repo := mocks.NewRepository(t)
	assert.NotNil(t, repo)
	assert.IsType(t, &mocks.Repository{}, repo)
}
