package index

import (
	"fmt"
	"github.com/maynagashev/go-metrics/internal/storage"
	"net/http"
	"sort"
)

// New возвращает http.HandlerFunc, который отдает список метрик на сервере от
func New(st storage.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		metrics := st.GetMetrics()
		// Выбираем и сортируем ключи (типы метрик)
		metricTypes := make([]string, 0, len(metrics))
		for k := range metrics {
			metricTypes = append(metricTypes, k)
		}
		sort.Strings(metricTypes)

		for _, metricType := range metricTypes {
			metricsByType := metrics[metricType]

			// Выбираем и сортируем ключи (названия метрик)
			metricNames := make([]string, 0, len(metricsByType))
			for k := range metricsByType {
				metricNames = append(metricNames, k)
			}
			sort.Strings(metricNames)

			for _, name := range metricNames {
				_, err := w.Write([]byte(fmt.Sprintf("%s/%s: %v\n", metricType, name, metricsByType[name])))
				if err != nil {
					return
				}
			}
		}

	}
}
