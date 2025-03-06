// Package value provides a handler for the /value endpoint.
package value

import (
	"fmt"
	"net/http"

	"github.com/maynagashev/go-metrics/internal/server/storage"

	"github.com/go-chi/chi/v5"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// New хэндлер для получения занчения метрики с сервера /value/{type}/{name}.
func New(storage storage.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		metricType := metrics.MetricType(chi.URLParam(r, "type"))
		metricName := chi.URLParam(r, "name")

		metric, ok := storage.GetMetric(r.Context(), metricType, metricName)
		if !ok {
			http.Error(
				w,
				fmt.Sprintf("%s %s not found", metricType, metricName),
				http.StatusNotFound,
			)
			return
		}

		_, err := w.Write([]byte(metric.ValueString()))
		if err != nil {
			return
		}
	}
}
