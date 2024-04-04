// Package value provides a handler for the /value endpoint.
package value

import (
	"net/http"

	"github.com/maynagashev/go-metrics/internal/server/storage"

	"github.com/go-chi/chi/v5"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

func New(storage storage.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		metricType := metrics.MetricType(chi.URLParam(r, "type"))
		metricName := chi.URLParam(r, "name")

		value, err := storage.GetValue(metricType, metricName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
		}

		_, err = w.Write([]byte(value))
		if err != nil {
			return
		}
	}
}
