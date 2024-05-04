package update

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

type ResponseWithMessage struct {
	Message string `json:"message"`
}

// New возвращает http.HandlerFunc, который обновляет значение метрики в хранилище.
func New(server *app.Server, strg storage.Repository, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Отметаем все кроме POST
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// todo: При попытке передать запрос без имени метрики возвращать http.StatusNotFound.

		metric, err := parseMetricFromRequest(r, log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Debug("parsed metric", zap.Any("metric", metric))

		err = strg.UpdateMetric(metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var resMessage string

		// Получаем значение метрики из хранилища
		v, ok := strg.GetValue(metric.MType, metric.ID)
		if ok {
			resMessage = fmt.Sprintf("metric %s updated, result: %s", metric.String(), v)
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

		// Сохраняем метрики в файл сразу после изменения, если включено синхронное сохранение
		if server.IsStoreEnabled() && server.IsSyncStore() {
			err = strg.StoreMetricsToFile()
			if err != nil {
				log.Error(fmt.Sprintf("error storing metrics: %s", err))
			}
		}
	}
}

// Читаем метрику из json запроса.
func parseMetricFromRequest(r *http.Request, log *zap.Logger) (metrics.Metric, error) {
	m := metrics.Metric{}
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.Body)

	if err != nil {
		return m, err
	}

	log.Debug("request body", zap.String("body", buf.String()))

	err = json.Unmarshal(buf.Bytes(), &m)
	if err != nil {
		return m, err
	}

	return m, nil
}
