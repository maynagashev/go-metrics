package updates

import (
	"encoding/json"
	"net/http"

	"github.com/maynagashev/go-metrics/internal/lib/api/response"

	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/storage"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// NewBulkUpdate возвращает http.HandlerFunc, который обновляет множество метрик в хранилище.
// Метрики передаются в теле запроса в формате JSON.
func NewBulkUpdate(st storage.Repository, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Парсим тело запроса в слайс метрик
		var metricsToUpdate []metrics.Metric
		err := json.NewDecoder(r.Body).Decode(&metricsToUpdate)
		if err != nil {
			response.Error(w, err, http.StatusBadRequest)
			return
		}

		// Обновляем каждую метрику в хранилище
		// TODO: Метрики следует обновлять одним запросом или транзакцией, чтобы уменьшить нагрузку на сервер.
		for _, m := range metricsToUpdate {
			err = st.UpdateMetric(m)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		// Отправляем успешный ответ
		w.WriteHeader(http.StatusOK)

		// Логируем ответ для отладки
		log.Info("Metrics updated successfully")

		// Выводим в тело ответа сообщение о результате
		response.OK(w, "Metrics updated successfully")
	}
}
