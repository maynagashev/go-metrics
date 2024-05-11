// Package value provides a handler for the /value endpoint.
package value

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/maynagashev/go-metrics/internal/server/storage"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// New хэндлер для получения значения метрики с сервера в ответ на запрос `POST /value`.
func New(storage storage.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var err error

		requestMetric, err := parseMetricFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		// Получаем значение метрики из хранилища
		metric, ok := storage.GetMetric(requestMetric.MType, requestMetric.ID)
		if !ok {
			http.Error(w, fmt.Sprintf("%s not found", metric.String()), http.StatusNotFound)
			return
		}

		// Отправляем json ответ с метрикой
		encoded, err := json.Marshal(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(encoded)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// Читаем метрику из json запроса.
func parseMetricFromRequest(r *http.Request) (metrics.Metric, error) {
	m := metrics.Metric{}
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.Body)

	if err != nil {
		return m, err
	}

	err = json.Unmarshal(buf.Bytes(), &m)
	if err != nil {
		return m, err
	}

	return m, nil
}
