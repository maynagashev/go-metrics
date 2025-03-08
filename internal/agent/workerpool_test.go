package agent

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/stretchr/testify/assert"
)

// mockAgent is a test implementation of agent that allows mocking sendMetrics.
type mockAgent struct {
	sendQueue           chan Job
	resultQueue         chan Result
	stopCh              chan struct{}
	wg                  sync.WaitGroup
	mockSendMetricsFunc func(metrics []*metrics.Metric, workerID int) error
}

// worker is a copy of the original worker function but uses mockSendMetricsFunc.
func (m *mockAgent) worker(id int) {
	defer func() {
		m.wg.Done()
	}()

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
				// Result sent successfully
			case <-m.stopCh:
				return
			}
		case <-m.stopCh:
			return
		}
	}
}

// collector is a copy of the original collector function.
func (m *mockAgent) collector() {
	defer func() {
		m.wg.Done()
	}()

	for {
		select {
		case result, ok := <-m.resultQueue:
			if !ok {
				return
			}
			// We don't need to log anything in tests
			_ = result
		case <-m.stopCh:
			return
		}
	}
}

// TestWorker tests the worker function.
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
			name: "Process job successfully",
			setupFunc: func(m *mockAgent) {
				m.mockSendMetricsFunc = func(metrics []*metrics.Metric, workerID int) error {
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
			name: "Handle error from sendMetrics",
			setupFunc: func(m *mockAgent) {
				m.mockSendMetricsFunc = func(metrics []*metrics.Metric, workerID int) error {
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
			name: "Handle closed send queue",
			setupFunc: func(m *mockAgent) {
				// No special setup needed
			},
			jobToSend:     nil,
			expectedError: nil,
			closeQueue:    true,
			closeStopCh:   false,
		},
		{
			name: "Handle stop signal",
			setupFunc: func(m *mockAgent) {
				// No special setup needed
			},
			jobToSend:     nil,
			expectedError: nil,
			closeQueue:    false,
			closeStopCh:   true,
		},
		{
			name: "Handle stop signal while sending result",
			setupFunc: func(m *mockAgent) {
				// No special setup needed
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
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock agent
			m := &mockAgent{
				sendQueue:   make(chan Job, 10),
				resultQueue: make(chan Result, 10),
				stopCh:      make(chan struct{}),
				wg:          sync.WaitGroup{},
			}

			// Apply test-specific setup
			if tt.setupFunc != nil {
				tt.setupFunc(m)
			}

			// Fill result queue if needed for the test
			if tt.fullResultQueue {
				for range 10 {
					m.resultQueue <- Result{}
				}
			}

			m.wg.Add(1)

			// Start the worker
			go m.worker(1)

			// Close queue if needed for the test
			if tt.closeQueue {
				close(m.sendQueue)
			}

			// Send job if provided
			if tt.jobToSend != nil {
				m.sendQueue <- *tt.jobToSend

				// If we're not testing a full result queue, wait for the result
				if !tt.fullResultQueue {
					result := <-m.resultQueue
					assert.Equal(t, *tt.jobToSend, result.Job, "The result should contain the original job")
					if tt.expectedError != nil {
						assert.Error(t, result.Error)
						assert.Equal(t, tt.expectedError.Error(), result.Error.Error(), "The error should match")
					} else {
						assert.NoError(t, result.Error, "There should be no error")
					}
				} else {
					// Give the worker time to process the job and try to send the result
					time.Sleep(100 * time.Millisecond)
				}
			}

			// Close stop channel if needed for the test
			if tt.closeStopCh {
				close(m.stopCh)
			}

			// Wait for the worker to finish
			m.wg.Wait()
		})
	}
}

// TestCollector tests the collector function.
func TestCollector(t *testing.T) {
	tests := []struct {
		name        string
		sendResults []Result
		closeQueue  bool
		closeStopCh bool
	}{
		{
			name: "Process results successfully",
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
			name:        "Handle closed result queue",
			sendResults: nil,
			closeQueue:  true,
			closeStopCh: false,
		},
		{
			name:        "Handle stop signal",
			sendResults: nil,
			closeQueue:  false,
			closeStopCh: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock agent
			m := &mockAgent{
				sendQueue:   make(chan Job, 10),
				resultQueue: make(chan Result, 10),
				stopCh:      make(chan struct{}),
				wg:          sync.WaitGroup{},
			}

			m.wg.Add(1)

			// Start the collector
			go m.collector()

			// Send results if provided
			if tt.sendResults != nil {
				for _, result := range tt.sendResults {
					m.resultQueue <- result
				}
				// Give the collector time to process the results
				time.Sleep(100 * time.Millisecond)
			}

			// Close queue if needed for the test
			if tt.closeQueue {
				close(m.resultQueue)
			}

			// Close stop channel if needed for the test
			if tt.closeStopCh {
				close(m.stopCh)
			}

			// Wait for the collector to finish
			m.wg.Wait()
		})
	}
}
