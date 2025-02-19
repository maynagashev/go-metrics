// Package update реализует обработчик для обновления метрик в текстовом формате.
// Предоставляет эндпоинт для обновления значений метрик через простой текстовый интерфейс.
package update

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/storage"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// New возвращает http.HandlerFunc, который обновляет значение метрики в хранилище.
func New(st storage.Repository, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		// Получаем части пути из URL /update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
		parts := strings.Split(r.URL.Path, "/")
		expectedPartsLen := 5

		// При попытке передать запрос без имени метрики возвращать http.StatusNotFound.
		if len(parts) != expectedPartsLen {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}

		metricType := metrics.MetricType(parts[2])
		metricName := parts[3]
		metricValue := parts[4]

		var m *metrics.Metric
		switch metricType {
		case metrics.TypeCounter:
			intValue, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				http.Error(
					w,
					"Invalid metrics value, must be convertable to int64",
					http.StatusBadRequest,
				)
				return
			}
			m = metrics.NewCounter(metricName, intValue)
		case metrics.TypeGauge:
			floatValue, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				http.Error(
					w,
					"Invalid metrics value, must be convertable to float64",
					http.StatusBadRequest,
				)
				return
			}
			m = metrics.NewGauge(metricName, floatValue)
		default:
			http.Error(w, "Invalid metrics type, must be: counter or gauge", http.StatusBadRequest)
			return
		}

		// Обновляем метрику в хранилище
		err := st.UpdateMetric(r.Context(), *m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		var resMessage string
		// Получаем значение метрики из хранилища
		v, ok := st.GetMetric(r.Context(), metricType, metricName)
		if ok {
			resMessage = fmt.Sprintf("metric %s/%s updated with value %s, result: %s",
				metricType, metricName, metricValue, v.String())
		} else {
			resMessage = fmt.Sprintf("metric %s/%s not found", metricType, metricName)
		}

		// Отправляем успешный ответ
		w.WriteHeader(http.StatusOK)

		// Логируем ответ для отладки
		log.Info(resMessage)

		// Выводим в тело ответа сообщение о результате
		_, err = fmt.Fprint(w, resMessage)
		if err != nil {
			log.Error(fmt.Sprintf("error writing response: %s", err))
			return
		}
	}
}
