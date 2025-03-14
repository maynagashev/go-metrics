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
func (a *agent) worker(id int) {
	defer func() {
		slog.Info("Worker shutting down", "workerID", id)
		a.wg.Done()
	}()

	slog.Debug(fmt.Sprintf("worker %d started", id))
	// По мере поступления задач в очередь отправляем их на сервер (читаем из канала очередную запись текущим воркером).
	for {
		select {
		case job, ok := <-a.sendQueue:
			if !ok {
				slog.Info("Send queue closed, worker exiting", "workerID", id)
				return
			}
			slog.Debug(
				fmt.Sprintf("worker %d received job, calling sendMetrics()...", id),
				"workerID",
				id,
			)
			err := a.sendMetrics(job.Metrics, id)
			// Отправляем результат выполнения задачи (ошибку, если была) в очередь результатов,
			// которые потом разбирает коллектор.
			select {
			case a.resultQueue <- Result{Job: job, Error: err}:
				// Результат успешно отправлен
			case <-a.stopCh:
				slog.Info(
					"Stop signal received while sending result, worker exiting",
					"workerID",
					id,
				)
				return
			}
		case <-a.stopCh:
			slog.Info("Stop signal received, worker exiting", "workerID", id)
			return
		}
	}
}

// Общий коллектор обрабатывает результаты выполнения задач.
func (a *agent) collector() {
	defer func() {
		slog.Info("Collector shutting down")
		a.wg.Done()
	}()

	slog.Info("Collector started")
	for {
		select {
		case result, ok := <-a.resultQueue:
			if !ok {
				slog.Info("Result queue closed, collector exiting")
				return
			}
			if result.Error != nil {
				wrappedError := fmt.Errorf("collector: %w", result.Error)
				slog.Error(wrappedError.Error(), "error", wrappedError)
			} else {
				slog.Info("Metrics sent successfully", "count", len(result.Job.Metrics))
			}
		case <-a.stopCh:
			slog.Info("Stop signal received, collector exiting")
			return
		}
	}
}
