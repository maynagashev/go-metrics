//nolint:testpackage // использует внутреннее API агента для тестирования
package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// mockClient - реализация интерфейса client.Client для тестирования.
type mockClient struct {
	mock.Mock
}

func (m *mockClient) UpdateMetric(_ context.Context, metric *metrics.Metric) error {
	args := m.Called(metric)
	return args.Error(0)
}

func (m *mockClient) UpdateBatch(_ context.Context, metrics []*metrics.Metric) error {
	args := m.Called(metrics)
	return args.Error(0)
}

func (m *mockClient) Ping(_ context.Context) error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockClient) StreamMetrics(_ context.Context, metrics []*metrics.Metric) error {
	args := m.Called(metrics)
	return args.Error(0)
}

func (m *mockClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// runWorkerTestCase содержит параметры тестового случая для runWorker.
type runWorkerTestCase struct {
	name          string
	setupMock     func(m *mockClient)
	metrics       []*metrics.Metric
	wantError     bool
	errorMessage  string
	sendJob       bool
	closeStopCh   bool
	grpcEnabled   bool
	expectContext bool // Проверять контекст не будем, это сложно для мока
}

// TestRunWorker проверяет функцию runWorker.
//
//nolint:gocognit // тестовая функция с разными сценариями
func TestRunWorker(t *testing.T) {
	tests := []runWorkerTestCase{
		{
			name: "Успешная отправка метрик через HTTP",
			setupMock: func(m *mockClient) {
				m.On("UpdateBatch", mock.MatchedBy(func(_ []*metrics.Metric) bool {
					return true
				})).Return(nil)
			},
			metrics: []*metrics.Metric{
				metrics.NewGauge("test_gauge", 42.0),
			},
			wantError:     false,
			sendJob:       true,
			closeStopCh:   false,
			grpcEnabled:   false,
			expectContext: true,
		},
		{
			name: "Успешная отправка метрик через gRPC",
			setupMock: func(m *mockClient) {
				m.On("StreamMetrics", mock.MatchedBy(func(_ []*metrics.Metric) bool {
					return true
				})).Return(nil)
			},
			metrics: []*metrics.Metric{
				metrics.NewGauge("test_gauge", 42.0),
			},
			wantError:     false,
			sendJob:       true,
			closeStopCh:   false,
			grpcEnabled:   true,
			expectContext: true,
		},
		{
			name: "Ошибка при отправке метрик через HTTP",
			setupMock: func(m *mockClient) {
				m.On("UpdateBatch", mock.MatchedBy(func(_ []*metrics.Metric) bool {
					return true
				})).Return(errors.New("http error"))
			},
			metrics: []*metrics.Metric{
				metrics.NewGauge("test_gauge", 42.0),
			},
			wantError:     true,
			errorMessage:  "http error",
			sendJob:       true,
			closeStopCh:   false,
			grpcEnabled:   false,
			expectContext: true,
		},
		{
			name: "Ошибка при отправке метрик через gRPC",
			setupMock: func(m *mockClient) {
				m.On("StreamMetrics", mock.MatchedBy(func(_ []*metrics.Metric) bool {
					return true
				})).Return(errors.New("grpc error"))
			},
			metrics: []*metrics.Metric{
				metrics.NewGauge("test_gauge", 42.0),
			},
			wantError:     true,
			errorMessage:  "grpc error",
			sendJob:       true,
			closeStopCh:   false,
			grpcEnabled:   true,
			expectContext: true,
		},
		{
			name: "Завершение по stopCh",
			setupMock: func(_ *mockClient) {
				// Не настраиваем мок, так как не ожидаем вызовов
			},
			metrics:       nil,
			wantError:     false,
			sendJob:       false,
			closeStopCh:   true,
			grpcEnabled:   false,
			expectContext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем мок клиента
			mockClient := new(mockClient)
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			// Создаем реальный экземпляр агента с необходимыми каналами
			testAgent := &agent{
				sendQueue:   make(chan Job, 1),
				resultQueue: make(chan Result, 1),
				stopCh:      make(chan struct{}),
				client:      mockClient,
				GRPCEnabled: tt.grpcEnabled,
			}

			// Инициализируем WaitGroup для ожидания завершения worker
			var wg sync.WaitGroup
			wg.Add(1)

			// Запускаем worker
			go func() {
				defer wg.Done()
				worker := testAgent.runWorker(1)
				worker()
			}()

			// Ждем немного чтобы worker успел запуститься
			time.Sleep(50 * time.Millisecond)

			// Отправляем задание, если нужно
			if tt.sendJob && tt.metrics != nil {
				job := Job{Metrics: tt.metrics}
				testAgent.sendQueue <- job

				// Ждем результата
				result := <-testAgent.resultQueue

				// Проверяем результат
				if tt.wantError {
					require.Error(t, result.Error)
					if tt.errorMessage != "" {
						assert.Contains(t, result.Error.Error(), tt.errorMessage)
					}
				} else {
					require.NoError(t, result.Error)
				}

				// Проверяем, что задание было передано корректно
				assert.Equal(t, tt.metrics, result.Job.Metrics)
			}

			// Закрываем стоп-канал, если нужно
			if tt.closeStopCh {
				close(testAgent.stopCh)
			}

			// Даем воркеру немного времени на выполнение
			time.Sleep(50 * time.Millisecond)

			// Если мы не закрыли стоп-канал, но воркер должен продолжать работать,
			// закрываем его сейчас для завершения теста
			if !tt.closeStopCh {
				close(testAgent.stopCh)
			}

			// Ожидаем завершения воркера
			wg.Wait()

			// Проверяем, что все ожидаемые методы были вызваны
			mockClient.AssertExpectations(t)
		})
	}
}
