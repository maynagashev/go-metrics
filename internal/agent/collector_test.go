//nolint:testpackage // использует внутреннее API агента для тестирования
package agent

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// TestRunCollector проверяет функцию runCollector.
func TestRunCollector(t *testing.T) {
	tests := []struct {
		name        string
		results     []Result
		closeQueue  bool
		checkErrors bool
	}{
		{
			name: "Успешная обработка результатов без ошибок",
			results: []Result{
				{
					Job: Job{
						Metrics: []*metrics.Metric{
							metrics.NewGauge("gauge1", 1.0),
						},
					},
					Error: nil,
				},
				{
					Job: Job{
						Metrics: []*metrics.Metric{
							metrics.NewGauge("gauge2", 2.0),
						},
					},
					Error: nil,
				},
			},
			closeQueue:  true,
			checkErrors: false,
		},
		{
			name: "Обработка результатов с ошибками",
			results: []Result{
				{
					Job: Job{
						Metrics: []*metrics.Metric{
							metrics.NewGauge("gauge1", 1.0),
						},
					},
					Error: errors.New("test error 1"),
				},
				{
					Job: Job{
						Metrics: []*metrics.Metric{
							metrics.NewGauge("gauge2", 2.0),
						},
					},
					Error: errors.New("test error 2"),
				},
			},
			closeQueue:  true,
			checkErrors: true,
		},
		{
			name:        "Завершение по stopCh",
			results:     []Result{},
			closeQueue:  false,
			checkErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем агента и необходимые каналы
			a := &agent{
				resultQueue: make(chan Result, len(tt.results)+1),
				stopCh:      make(chan struct{}),
			}

			// Добавляем результаты в очередь
			for _, result := range tt.results {
				a.resultQueue <- result
			}

			// Инициализируем WaitGroup для ожидания завершения collector
			var wg sync.WaitGroup
			wg.Add(1)

			// Запускаем collector
			go func() {
				defer wg.Done()
				collector := a.runCollector()
				collector()
			}()

			// Ждем небольшой промежуток времени для начала выполнения
			time.Sleep(50 * time.Millisecond)

			// Закрываем очередь или стоп-канал
			if tt.closeQueue {
				close(a.resultQueue)
			} else {
				close(a.stopCh)
			}

			// Ожидаем завершения collector
			wg.Wait()

			// Проверяем, что все результаты были обработаны
			assert.Empty(t, a.resultQueue, "Результаты должны быть обработаны")
		})
	}
}
