package index

import (
	"encoding/json"
	"net/http"

	"github.com/maynagashev/go-metrics/internal/server/storage"
)

// New возвращает http.HandlerFunc, который отдает список метрик на сервере.
func New(st storage.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Возвращаем метрики в формате JSON архива
		metrics := st.GetMetrics()
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
