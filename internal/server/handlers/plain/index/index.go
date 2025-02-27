package index

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/storage"
)

// New возвращает http.HandlerFunc, который отдает список метрик на сервере.
func New(st storage.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		// Возвращаем метрики в виде списка строк (первоначальный вариант)
		ms := st.GetMetrics(r.Context())
		items := make([]string, 0, st.Count(r.Context()))
		for _, metric := range ms {
			switch metric.MType {
			case metrics.TypeGauge:
				valF := strconv.FormatFloat(*metric.Value, 'f', -1, 64)
				items = append(items, fmt.Sprintf("gauge/%s: %s", metric.Name, valF))
			case metrics.TypeCounter:
				items = append(items, fmt.Sprintf("counter/%s: %d", metric.Name, *metric.Delta))
			default:
				items = append(items, fmt.Sprintf("unknown/%s", metric.Name))
			}
		}

		for _, metric := range items {
			_, err := w.Write([]byte(metric + "\n"))
			if err != nil {
				return
			}
		}
	}
}
