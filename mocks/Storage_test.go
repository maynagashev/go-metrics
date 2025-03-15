package mocks_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/mocks"
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
				*metrics.NewGauge("gauge1", 1.0),
				*metrics.NewCounter("counter1", 1),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new mock storage
			s := mocks.NewStorage(t)

			// Setup expectations
			s.On("GetMetrics", mock.Anything).Return(tt.mockMetrics)

			// Call the method
			result := s.GetMetrics(context.Background())

			// Assert expectations
			s.AssertExpectations(t)

			// Check the result
			assert.Equal(t, tt.mockMetrics, result)
		})
	}
}

func TestNewStorage(t *testing.T) {
	s := mocks.NewStorage(t)
	assert.NotNil(t, s)
	assert.IsType(t, &mocks.Storage{}, s)
}
