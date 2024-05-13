package update

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

type ResponseWithMessage struct {
	Message string `json:"message"`
}

type Metric struct {
	Name  string             `json:"id"`              // Имя метрики
	MType metrics.MetricType `json:"type"`            // Параметр, принимающий значение gauge или counter
	Delta *int64             `json:"delta,omitempty"` // Значение метрики в случае передачи counter
	Value *float64           `json:"value,omitempty"` // Значение метрики в случае передачи gauge
}

// New возвращает http.HandlerFunc, который обновляет значение метрики в хранилище.
func New(_ *app.Config, strg storage.Repository, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		requestedMetric, err := parseMetricFromRequest(r, log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Debug("parsed metric", zap.Any("metric", requestedMetric))

		// Конвертируем локальную структуру в структуру из контракта
		metric := metrics.Metric(requestedMetric)
		err = strg.UpdateMetric(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var resMessage string

		// Получаем значение метрики из хранилища
		m, ok := strg.GetMetric(metric.MType, metric.Name)
		if ok {
			resMessage = fmt.Sprintf("metric %s updated, result: %s", metric.String(), m.String())
		} else {
			resMessage = fmt.Sprintf("metric %s not found", metric.String())
		}

		// Отправляем успешный ответ
		w.WriteHeader(http.StatusOK)

		// Логируем ответ для отладки
		log.Info(resMessage)

		// Выводим в тело ответа сообщение о результате
		encoded, err := json.Marshal(ResponseWithMessage{Message: resMessage})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = fmt.Fprint(w, string(encoded))
		if err != nil {
			log.Error(fmt.Sprintf("error writing response: %s", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// Читаем метрику из json запроса.
func parseMetricFromRequest(r *http.Request, log *zap.Logger) (Metric, error) {
	m := Metric{}
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.Body)

	if err != nil {
		return m, err
	}

	log.Debug("request body", zap.String("body", buf.String()))

	err = json.Unmarshal(buf.Bytes(), &m)
	if err != nil {
		return m, err
	}

	return m, nil
}
