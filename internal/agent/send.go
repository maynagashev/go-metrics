package agent

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

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
	// TODO: добавить повторные попытки отправки при ошибках, если ошибка retriable.
	err := a.makeUpdatesRequest(items)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to send metrics: %s", err), "metrics", items)
		return
	}
}

// Отправка запроса на сервер с пачкой метрик, маршрут: `POST /updates`.
// При ошибках подключения запрос можно повторить, но не более 3-х раз (retriable errors).
func (a *Agent) makeUpdatesRequest(items []*metrics.Metric) error {
	var err error
	url := fmt.Sprintf("%s/updates", a.ServerURL)
	slog.Info("sending metrics batch", "url", url, "metrics", items)

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

	slog.Debug("makeUpdatesRequest", "url", url, "req", req)

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
