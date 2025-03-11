package agent_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тестовые версии типов Job и Result

type Job struct {
	Metrics []*metrics.Metric
}

type Result struct {
	Job   Job
	Error error
}

// mockAgent - это тестовая реализация агента, позволяющая имитировать sendMetrics.
type mockAgent struct {
	sendQueue           chan Job
	resultQueue         chan Result
	stopCh              chan struct{}
	wg                  sync.WaitGroup
	mockSendMetricsFunc func(metrics []*metrics.Metric, workerID int) error
}

// worker - это копия оригинальной функции worker, но использует mockSendMetricsFunc.
func (m *mockAgent) worker(id int) {
	defer m.wg.Done()

	for {
		select {
		case job, ok := <-m.sendQueue:
			if !ok {
				return
			}
			var err error
			if m.mockSendMetricsFunc != nil {
				err = m.mockSendMetricsFunc(job.Metrics, id)
			}
			select {
			case m.resultQueue <- Result{Job: job, Error: err}:
				// Результат успешно отправлен
			case <-m.stopCh:
				return
			}
		case <-m.stopCh:
			return
		}
	}
}

// collector - это копия оригинальной функции collector.
func (m *mockAgent) collector() {
	defer m.wg.Done()

	for {
		select {
		case result, ok := <-m.resultQueue:
			if !ok {
				return
			}
			// В тестах мы не логируем
			_ = result
		case <-m.stopCh:
			return
		}
	}
}

// Параметр t не используется — переименуем его в _.
func setupMockAgent(_ *testing.T, tt struct {
	name            string
	setupFunc       func(m *mockAgent)
	jobToSend       *Job
	expectedError   error
	closeQueue      bool
	closeStopCh     bool
	fullResultQueue bool
}, m *mockAgent) {
	if tt.setupFunc != nil {
		tt.setupFunc(m)
	}

	if tt.fullResultQueue {
		for range [10]int{} {
			m.resultQueue <- Result{}
		}
	}
}

// Вспомогательная функция для отправки задания и проверки результата.
func sendJobAndValidateResult(t *testing.T, tt struct {
	name            string
	setupFunc       func(m *mockAgent)
	jobToSend       *Job
	expectedError   error
	closeQueue      bool
	closeStopCh     bool
	fullResultQueue bool
}, m *mockAgent) {
	if tt.jobToSend == nil {
		return
	}

	m.sendQueue <- *tt.jobToSend

	if tt.fullResultQueue {
		time.Sleep(100 * time.Millisecond)
		return
	}

	result := <-m.resultQueue
	assert.Equal(t, *tt.jobToSend, result.Job, "Результат должен содержать исходное задание")
	if tt.expectedError != nil {
		require.Error(t, result.Error)
		assert.Equal(t, tt.expectedError.Error(), result.Error.Error(), "Ошибки должны совпадать")
	} else {
		assert.NoError(t, result.Error, "Ошибки быть не должно")
	}
}

// TestWorker тестирует функцию worker.
func TestWorker(t *testing.T) {
	tests := []struct {
		name            string
		setupFunc       func(m *mockAgent)
		jobToSend       *Job
		expectedError   error
		closeQueue      bool
		closeStopCh     bool
		fullResultQueue bool
	}{
		{
			name: "Успешная обработка задания",
			setupFunc: func(m *mockAgent) {
				m.mockSendMetricsFunc = func(_ []*metrics.Metric, _ int) error {
					return nil
				}
			},
			jobToSend: &Job{
				Metrics: []*metrics.Metric{
					{
						Name:  "test_metric",
						MType: "gauge",
						Value: func() *float64 { v := 42.0; return &v }(),
					},
				},
			},
			expectedError: nil,
			closeQueue:    false,
			closeStopCh:   true,
		},
		{
			name: "Обработка ошибки из sendMetrics",
			setupFunc: func(m *mockAgent) {
				m.mockSendMetricsFunc = func(_ []*metrics.Metric, _ int) error {
					return errors.New("test error")
				}
			},
			jobToSend: &Job{
				Metrics: []*metrics.Metric{
					{
						Name:  "test_metric",
						MType: "gauge",
						Value: func() *float64 { v := 42.0; return &v }(),
					},
				},
			},
			expectedError: errors.New("test error"),
			closeQueue:    false,
			closeStopCh:   true,
		},
		{
			name: "Обработка закрытой очереди отправки",
			setupFunc: func(_ *mockAgent) {
				// Нет особых настроек
			},
			jobToSend:     nil,
			expectedError: nil,
			closeQueue:    true,
			closeStopCh:   false,
		},
		{
			name: "Обработка сигнала остановки",
			setupFunc: func(_ *mockAgent) {
				// Нет особых настроек
			},
			jobToSend:     nil,
			expectedError: nil,
			closeQueue:    false,
			closeStopCh:   true,
		},
		{
			name: "Обработка сигнала остановки во время отправки результата",
			setupFunc: func(_ *mockAgent) {
				// Нет особых настроек
			},
			jobToSend: &Job{
				Metrics: []*metrics.Metric{},
			},
			expectedError:   nil,
			closeQueue:      false,
			closeStopCh:     true,
			fullResultQueue: true,
		},
	}

	for _, tt := range tests {
		// Локальная копия для параллельных тестов, если захотите вызывать t.Parallel()

		t.Run(tt.name, func(t *testing.T) {
			m := &mockAgent{
				sendQueue:   make(chan Job, 10),
				resultQueue: make(chan Result, 10),
				stopCh:      make(chan struct{}),
			}

			setupMockAgent(t, tt, m)

			m.wg.Add(1)
			go m.worker(1)

			if tt.closeQueue {
				close(m.sendQueue)
			}

			sendJobAndValidateResult(t, tt, m)

			if tt.closeStopCh {
				close(m.stopCh)
			}

			m.wg.Wait()
		})
	}
}

// TestCollector тестирует функцию collector.
func TestCollector(t *testing.T) {
	tests := []struct {
		name        string
		sendResults []Result
		closeQueue  bool
		closeStopCh bool
	}{
		{
			name: "Успешная обработка результатов",
			sendResults: []Result{
				{
					Job: Job{
						Metrics: []*metrics.Metric{
							{
								Name:  "test_metric",
								MType: "gauge",
								Value: func() *float64 { v := 42.0; return &v }(),
							},
						},
					},
					Error: nil,
				},
				{
					Job: Job{
						Metrics: []*metrics.Metric{
							{
								Name:  "test_metric",
								MType: "gauge",
								Value: func() *float64 { v := 42.0; return &v }(),
							},
						},
					},
					Error: errors.New("test error"),
				},
			},
			closeQueue:  false,
			closeStopCh: true,
		},
		{
			name:        "Обработка закрытой очереди результатов",
			sendResults: nil,
			closeQueue:  true,
			closeStopCh: false,
		},
		{
			name:        "Обработка сигнала остановки",
			sendResults: nil,
			closeQueue:  false,
			closeStopCh: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			m := &mockAgent{
				sendQueue:   make(chan Job, 10),
				resultQueue: make(chan Result, 10),
				stopCh:      make(chan struct{}),
			}

			m.wg.Add(1)
			go m.collector()

			// Отправляем результаты, если нужно
			if tt.sendResults != nil {
				for _, result := range tt.sendResults {
					m.resultQueue <- result
				}
				time.Sleep(100 * time.Millisecond)
			}

			// Закрываем очередь, если это предусмотрено тестом
			if tt.closeQueue {
				close(m.resultQueue)
			}

			// Закрываем stopCh, если это предусмотрено тестом
			if tt.closeStopCh {
				close(m.stopCh)
			}

			m.wg.Wait()
		})
	}
}
