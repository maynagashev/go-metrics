package updates

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/maynagashev/go-metrics/internal/server/app"
	sign "github.com/maynagashev/go-metrics/pkg/sign"

	"github.com/maynagashev/go-metrics/pkg/response"

	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/storage"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// NewBulkUpdate возвращает http.HandlerFunc, который обновляет множество метрик в хранилище.
// Метрики передаются в теле запроса в формате JSON.
func NewBulkUpdate(cfg *app.Config, st storage.Repository, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		w.Header().Set("Content-Type", "application/json")

		// Проверяем запрос на валидность и подпись если требуется.
		body, err := validateRequest(r, log, cfg)
		if err != nil {
			log.Debug("validate request failed", zap.Error(err))
			response.Error(w, err, http.StatusBadRequest)
			return
		}

		// Парсим тело запроса в слайс метрик.
		var metricsToUpdate []metrics.Metric
		err = json.Unmarshal([]byte(body), &metricsToUpdate)
		if err != nil {
			log.Debug("json decode failed", zap.Error(err))
			response.Error(w, err, http.StatusBadRequest)
			return
		}

		// Обновляем метрики в хранилище.
		err = st.UpdateMetrics(r.Context(), metricsToUpdate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Отправляем успешный ответ.
		w.WriteHeader(http.StatusOK)

		// Логируем ответ для отладки
		log.Info("Metrics updated successfully")

		// Выводим в тело ответа сообщение о результате
		response.OK(w, "Metrics updated successfully")
	}
}

// validateRequest проверяет запрос на валидность.
func validateRequest(r *http.Request, log *zap.Logger, cfg *app.Config) (string, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		return "", err
	}

	body := buf.Bytes()

	// Проверяем подпись запроса
	if cfg.IsRequestSigningEnabled() {
		hashFromRequest := r.Header.Get(sign.HeaderKey)
		hash, vErr := sign.VerifyHMACSHA256(body, cfg.PrivateKey, hashFromRequest)

		log.Debug(
			"validateRequest => sign.VerifyHMACSHA256",
			zap.String("hash_from_request", hashFromRequest),
			zap.Error(
				vErr,
			),
			zap.String("calc_hash", hash),
			zap.Any("headers", r.Header),
			zap.String("body", buf.String()),
		)

		if vErr != nil {
			return "", vErr
		}
	}

	return buf.String(), nil
}
