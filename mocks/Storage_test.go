package mocks

import (
	"context"
	"testing"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStorage_GetMetrics(t *testing.T) {
	tests := []struct {
		name        string
		mockMetrics []metrics.Metric
	}{
		{
			name:        "empty storage",
			mockMetrics: []metrics.Metric{},
		},
		{
			name: "storage with metrics",
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
			storage := NewStorage(t)
			storage.On("GetMetrics", mock.Anything).Return(tt.mockMetrics)

			metrics := storage.GetMetrics(context.Background())

			assert.Equal(t, tt.mockMetrics, metrics)
			storage.AssertExpectations(t)
		})
	}
}

func TestNewStorage(t *testing.T) {
	storage := NewStorage(t)
	assert.NotNil(t, storage)
	assert.IsType(t, &Storage{}, storage)
}
