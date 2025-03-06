package update

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/maynagashev/go-metrics/pkg/sign"

	"github.com/maynagashev/go-metrics/pkg/response"

	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

type ResponseWithMessage struct {
	Message string `json:"message"`
}

type Metric struct {
	Name  string             `json:"id"`              // Имя метрики
	MType metrics.MetricType `json:"type"`            // Параметр, принимающий значение gauge или counter
	Delta *int64             `json:"delta,omitempty"` // Значение метрики в случае передачи counter
	Value *float64           `json:"value,omitempty"` // Значение метрики в случае передачи gauge
}

// New возвращает http.HandlerFunc, который обновляет значение метрики в хранилище.
func New(cfg *app.Config, strg storage.Repository, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestedMetric, err := parseMetricFromRequest(r, log, cfg)
		if err != nil {
			log.Error("error while parsing metric", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Debug("parsed metric", zap.Any("metric", requestedMetric))

		// Конвертируем локальную структуру в структуру из контракта
		metric := metrics.Metric(requestedMetric)
		err = strg.UpdateMetric(r.Context(), metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var resMessage string

		// Получаем значение метрики из хранилища
		m, ok := strg.GetMetric(r.Context(), metric.MType, metric.Name)
		if ok {
			resMessage = fmt.Sprintf("metric %s updated, result: %s", metric.String(), m.String())
		} else {
			resMessage = fmt.Sprintf("metric %s not found", metric.String())
		}

		// Логируем ответ для отладки
		log.Info(resMessage)

		// Отправляем успешный ответ
		response.OK(w, resMessage)
	}
}

// Читаем метрику из json запроса.
func parseMetricFromRequest(r *http.Request, log *zap.Logger, cfg *app.Config) (Metric, error) {
	m := Metric{}
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.Body)

	if err != nil {
		return m, err
	}

	log.Debug("request body", zap.String("body", buf.String()))

	body := buf.Bytes()

	if cfg.IsRequestSigningEnabled() {
		// Проверяем подпись запроса
		expectedHash := r.Header.Get(sign.HeaderKey)
		requestHash, vErr := sign.VerifyHMACSHA256(body, cfg.PrivateKey, expectedHash)
		if vErr != nil {
			log.Error("error while verifying request signature", zap.Error(vErr),
				zap.String("expected_hash", expectedHash), zap.String("request_hash", requestHash))
			return m, vErr
		}
	}

	err = json.Unmarshal(body, &m)
	if err != nil {
		return m, err
	}

	return m, nil
}
