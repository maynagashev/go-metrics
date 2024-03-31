package update

import (
	"fmt"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/storage"
	"net/http"
	"strconv"
	"strings"
)

// New возвращает http.HandlerFunc, который обновляет значение метрики в хранилище.
func New(storage storage.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Получаем части пути из URL /update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
		parts := strings.Split(r.URL.Path, "/")

		// При попытке передать запрос без имени метрики возвращать http.StatusNotFound.
		if len(parts) != 5 {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}

		metricType := metrics.MetricType(parts[2])
		metricName := parts[3]
		metricValue := parts[4]

		switch metricType {
		case "counter":
			intValue, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				http.Error(w, "Invalid metrics value, must be convertable to int64", http.StatusBadRequest)
				return
			}
			storage.UpdateCounter(metricName, intValue)
		case "gauge":
			floatValue, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				http.Error(w, "Invalid metrics value, must be convertable to float64", http.StatusBadRequest)
				return
			}
			storage.UpdateGauge(metricName, floatValue)
		default:
			http.Error(w, "Invalid metrics type, must be: counter or gauge", http.StatusBadRequest)
			return
		}

		v, getError := storage.GetValue(metricType, metricName)
		if getError != nil {
			v = getError.Error()
		}
		resMessage := fmt.Sprintf("Metric %s/%s updated with value %s, result: %s",
			metricType, metricName, metricValue, v)

		// Отправляем успешный ответ
		w.WriteHeader(http.StatusOK)

		// Выводим в тело ответа сообщение о результате
		_, err := fmt.Fprint(w, resMessage)
		if err != nil {
			fmt.Printf("error writing response: %s\n", err)
			return
		}
		// Выводим в консоль результат операции
		fmt.Println(resMessage)
	}
}
