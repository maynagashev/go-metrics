//nolint:testpackage // использует внутреннее API агента для тестирования
package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// TestSendMetrics проверяет функцию sendMetrics.
func TestSendMetrics(t *testing.T) {
	tests := []struct {
		name         string
		setupAgent   func() *agent
		metrics      []*metrics.Metric
		expectError  bool
		errorMessage string
	}{
		{
			name: "Успешная отправка через HTTP",
			setupAgent: func() *agent {
				mockClient := new(mockClient)
				mockClient.On("UpdateBatch", mock.MatchedBy(func(metrics []*metrics.Metric) bool {
					return len(metrics) == 1 && metrics[0].Name == "test_gauge"
				})).Return(nil)

				return &agent{
					client:      mockClient,
					GRPCEnabled: false,
				}
			},
			metrics: []*metrics.Metric{
				metrics.NewGauge("test_gauge", 42.0),
			},
			expectError:  false,
			errorMessage: "",
		},
		{
			name: "Успешная отправка через gRPC",
			setupAgent: func() *agent {
				mockClient := new(mockClient)
				mockClient.On("StreamMetrics", mock.MatchedBy(func(metrics []*metrics.Metric) bool {
					return len(metrics) == 1 && metrics[0].Name == "test_gauge"
				})).Return(nil)

				return &agent{
					client:      mockClient,
					GRPCEnabled: true,
				}
			},
			metrics: []*metrics.Metric{
				metrics.NewGauge("test_gauge", 42.0),
			},
			expectError:  false,
			errorMessage: "",
		},
		{
			name: "Ошибка при отправке через HTTP",
			setupAgent: func() *agent {
				mockClient := new(mockClient)
				mockClient.On("UpdateBatch", mock.Anything).Return(errors.New("HTTP error"))

				return &agent{
					client:      mockClient,
					GRPCEnabled: false,
				}
			},
			metrics: []*metrics.Metric{
				metrics.NewGauge("test_gauge", 42.0),
			},
			expectError:  true,
			errorMessage: "HTTP error",
		},
		{
			name: "Ошибка при отправке через gRPC",
			setupAgent: func() *agent {
				mockClient := new(mockClient)
				mockClient.On("StreamMetrics", mock.Anything).Return(errors.New("gRPC error"))

				return &agent{
					client:      mockClient,
					GRPCEnabled: true,
				}
			},
			metrics: []*metrics.Metric{
				metrics.NewGauge("test_gauge", 42.0),
			},
			expectError:  true,
			errorMessage: "gRPC error",
		},
		{
			name: "Ошибка при nil клиенте",
			setupAgent: func() *agent {
				return &agent{
					client:      nil,
					GRPCEnabled: false,
				}
			},
			metrics:      []*metrics.Metric{},
			expectError:  true,
			errorMessage: "client is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем агента по настройке из теста
			a := tt.setupAgent()

			// Вызываем тестируемый метод
			err := a.sendMetrics(context.Background(), tt.metrics, 1)

			// Проверяем результат
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				require.NoError(t, err)
			}

			// Проверяем ожидания мока, если клиент не nil
			if mockClient, ok := a.client.(*mockClient); ok {
				mockClient.AssertExpectations(t)
			}
		})
	}
}
