// Package value provides a handler for the /value endpoint.
package value

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"github.com/maynagashev/go-metrics/pkg/sign"
)

// New хэндлер для получения значения метрики с сервера в ответ на запрос `POST /value`.
func New(cfg *app.Config, storage storage.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var err error

		requestMetric, err := parseMetricFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		// Получаем значение метрики из хранилища
		metric, ok := storage.GetMetric(r.Context(), requestMetric.MType, requestMetric.Name)
		if !ok {
			http.Error(w, fmt.Sprintf("%s not found", metric.String()), http.StatusNotFound)
			return
		}

		// Отправляем json ответ с метрикой
		encodedBody, err := json.Marshal(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Если задан приватный ключ, то подписываем ответ
		if cfg.IsRequestSigningEnabled() {
			signature := sign.ComputeHMACSHA256(encodedBody, cfg.PrivateKey)
			w.Header().Set(sign.HeaderKey, signature)
		}

		_, err = w.Write(encodedBody)
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
