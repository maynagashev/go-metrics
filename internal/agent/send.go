package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/lib/utils"
)

// Отправка всех накопленных метрик.
func (a *Agent) sendAllMetrics() {
	items := make([]*metrics.Metric, 0, len(a.gauges)+len(a.counters))

	// Делаем копию метрик, чтобы данные не изменились во время отправки.
	a.mu.Lock()
	slog.Info("sending metrics", "poll_count", a.counters["PollCount"])
	for name, value := range a.gauges {
		items = append(items, metrics.NewGauge(name, value))
	}
	for name, value := range a.counters {
		items = append(items, metrics.NewCounter(name, value))
	}
	// Обнуляем счетчик PollCount сразу как только подготовили его к отправке.
	// Из минусов: счетчик PollCount будет обнулен, даже если отправка метрик не удалась.
	// Другой вариант: обнулять счетчик PollCount только после успешной отправки метрик.
	a.counters["PollCount"] = 0
	slog.Info("reset poll count", "poll_count", 0)

	a.mu.Unlock()

	// Отправляем все метрики пачкой на новый маршрут /updates
	// Ошибки подключения при отправке метрик можно повторить, но не более 3-х раз (retriable errors).
	for i := 0; i <= maxSendRetries; i++ {
		// Пауза перед повторной отправкой.
		if i > 0 {
			//nolint:gomnd // количество секунд для паузы зависит от номера попытки
			sleepSeconds := i*2 - 1 // 1, 3, 5, 7, 9, 11, ...
			slog.Info(fmt.Sprintf("retrying to send metrics (try=%d) in %d seconds", i, sleepSeconds))
			time.Sleep(time.Duration(sleepSeconds) * time.Second)
		}

		err := a.makeUpdatesRequest(items, i)
		// Если нет ошибок выходим из цикла и функции.
		if err == nil {
			return
		}

		// Логируем ошибку
		slog.Error(fmt.Sprintf("failed to send metrics (try=%d): %s", i, err), "metrics", items)

		// Если ошибка не retriable, то выходим из цикла и функции, иначе продолжаем попытки.
		if !isRetriableSendError(err) {
			slog.Debug("non-retriable error, stopping retries", "err", err)
			return
		}
	}
}

func isRetriableSendError(err error) bool {
	slog.Debug(fmt.Sprintf("isRetriableSendError: %#v", err))

	// Проверяем, является ли ошибка общей ошибкой сети, временной или таймаутом.
	var netErr net.Error
	if errors.As(err, &netErr) {
		slog.Debug(fmt.Sprintf("isRetriableSendError => AS net.Error: %#v", netErr))
		// Проверяем, является ли ошибка временной
		if netErr.Timeout() {
			return true
		}
	}

	// Проверяем, является ли ошибка ошибкой сети.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		slog.Debug("isRetriableSendError => AS net.OpError", "err", err)
		return true
	}

	// Если ошибка не является временной, возвращаем false.
	return false
}

// Отправка запроса на сервер с пачкой метрик, маршрут: `POST /updates`.
// При ошибках подключения запрос можно повторить, но не более 3-х раз (retriable errors).
func (a *Agent) makeUpdatesRequest(items []*metrics.Metric, try int) error {
	var err error
	url := fmt.Sprintf("%s/updates", a.ServerURL)
	slog.Info(fmt.Sprintf("sending metrics batch (try=%d)", try), "url", url, "metrics", items)

	// Создаем новый запрос.
	req := a.client.R()
	req.Debug = true // Включаем отладочный режим, чтобы видеть все детали запроса, в частности, использование сжатия.
	req.SetHeader("Content-Type", "application/json")

	// Преобразуем метрики в JSON.
	bytesBody, err := json.Marshal(items)
	if err != nil {
		return err
	}

	// Если включена сразу отправка сжатых данных, добавляем соответствующий заголовок.
	// Go клиент автоматом также добавляет заголовок "Accept-Encoding: gzip".
	if a.SendCompressedData {
		req.SetHeader("Content-Encoding", "gzip")
		bytesBody, err = utils.Gzip(bytesBody)
		if err != nil {
			return err
		}
	}

	req.SetBody(bytesBody)

	res, err := req.Post(url)
	if err != nil {
		return err
	}

	// Обрабатываем ответ сервера.
	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode())
	}

	return nil
}
