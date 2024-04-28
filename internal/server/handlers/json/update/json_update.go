package update

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/storage"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

type ResponseWithMessage struct {
	Message string `json:"message"`
}

// New возвращает http.HandlerFunc, который обновляет значение метрики в хранилище.
func New(st storage.Repository, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Отметаем все кроме POST
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// todo: При попытке передать запрос без имени метрики возвращать http.StatusNotFound.

		metric, err := parseMetricFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Debug("parsed metric", zap.Any("metric", metric))

		err = st.UpdateMetric(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var resMessage string
		metricType, metricName, metricValue := metric.MType, metric.ID, metric.Value
		// Получаем значение метрики из хранилища
		v, ok := st.GetValue(metricType, metricName)
		if ok {
			resMessage = fmt.Sprintf("metric %s/%s updated with value %f, result: %s",
				metricType, metricName, *metricValue, v)
		} else {
			resMessage = fmt.Sprintf("metric %s not found", metric.String())
		}

		// Отправляем успешный ответ
		w.WriteHeader(http.StatusOK)

		// Логируем ответ для отладки
		log.Info(resMessage)

		// Выводим в тело ответа сообщение о результате
		encoded, err := json.MarshalIndent(ResponseWithMessage{Message: resMessage}, "", " ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = fmt.Fprint(w, string(encoded))
		if err != nil {
			log.Error(fmt.Sprintf("error writing response: %s", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// Читаем метрику из json запроса.
func parseMetricFromRequest(r *http.Request) (metrics.Metrics, error) {
	m := metrics.Metrics{}
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
