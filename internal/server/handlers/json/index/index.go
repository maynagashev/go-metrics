// Package index реализует обработчик для получения списка всех метрик в формате JSON.
// Предоставляет эндпоинт для получения текущего состояния всех метрик в системе.
package index

import (
	"encoding/json"
	"net/http"

	"github.com/maynagashev/go-metrics/internal/server/storage"
)

// New возвращает http.HandlerFunc, который отдает список метрик на сервере.
func New(st storage.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Возвращаем метрики в формате JSON архива
		metrics := st.GetMetrics(r.Context())
		jsonData, err := json.Marshal(metrics)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = w.Write(jsonData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
