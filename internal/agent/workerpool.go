package agent

import (
	"fmt"
	"log/slog"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// Job структура для задания воркерам.
type Job struct {
	Metrics []*metrics.Metric
}

// Result структура для результата выполнения задания.
type Result struct {
	Job   Job
	Error error
}

// Worker – один из воркеров пула для отправки метрик (обрабатывает задачи из очереди в отдельной горутине).
func (a *Agent) worker(id int) {
	defer a.wg.Done()
	slog.Debug(fmt.Sprintf("worker %d started", id))
	// По мере поступления задач в очередь отправляем их на сервер (читаем из канала очередную запись текущим воркером).
	for job := range a.sendQueue {
		slog.Debug(fmt.Sprintf("worker %d received job, calling sendMetrics()...", id), "workerID", id)
		err := a.sendMetrics(job.Metrics, id)
		// Отправляем результат выполнения задачи (ошибку, если была) в очередь результатов,
		// которые потом разбирает коллектор.
		a.resultQueue <- Result{Job: job, Error: err}
	}
}

// Общий коллектор обрабатывает результаты выполнения задач.
func (a *Agent) collector() {
	defer a.wg.Done()
	for result := range a.resultQueue {
		if result.Error != nil {
			wrappedError := fmt.Errorf("collector: %w", result.Error)
			slog.Error(wrappedError.Error(), "error", wrappedError)
		} else {
			slog.Info("collector: metrics sent successfully")
		}
	}
}
