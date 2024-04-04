package index

import (
	"net/http"

	"github.com/maynagashev/go-metrics/internal/server/storage"
)

// New возвращает http.HandlerFunc, который отдает список метрик на сервере.
func New(st storage.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		for _, metric := range st.GetMetrics() {
			_, err := w.Write([]byte(metric + "\n"))
			if err != nil {
				return
			}
		}
	}
}
