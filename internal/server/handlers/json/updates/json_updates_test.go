package updates_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/handlers/json/updates"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
	"github.com/maynagashev/go-metrics/pkg/sign"
)

func TestNewBulkUpdate_Success(t *testing.T) {
	// Создаем хранилище
	cfg := &app.Config{}
	logger := zap.NewNop()
	repo := memory.New(cfg, logger)

	// Создаем обработчик
	handler := updates.NewBulkUpdate(cfg, repo, logger)

	// Создаем тестовые метрики
	gaugeValue := 42.5
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)

	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)

	// Создаем массив метрик для обновления
	metricsToUpdate := []metrics.Metric{*gaugeMetric, *counterMetric}

	// Сериализуем метрики в JSON
	body, err := json.Marshal(metricsToUpdate)
	require.NoError(t, err)

	// Создаем HTTP-запрос
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Проверяем, что метрики были сохранены в хранилище
	savedGaugeMetric, ok := repo.GetMetric(context.Background(), metrics.TypeGauge, "test_gauge")
	assert.True(t, ok)
	assert.InDelta(t, gaugeValue, *savedGaugeMetric.Value, 0.0001)

	savedCounterMetric, ok := repo.GetMetric(
		context.Background(),
		metrics.TypeCounter,
		"test_counter",
	)
	assert.True(t, ok)
	assert.Equal(t, counterValue, *savedCounterMetric.Delta)
}

func TestNewBulkUpdate_InvalidJSON(t *testing.T) {
	// Создаем хранилище
	cfg := &app.Config{}
	logger := zap.NewNop()
	repo := memory.New(cfg, logger)

	// Создаем обработчик
	handler := updates.NewBulkUpdate(cfg, repo, logger)

	// Создаем невалидный JSON
	invalidJSON := []byte(`[{"id": "test", "type": "gauge", "value": invalid}]`)

	// Создаем HTTP-запрос
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNewBulkUpdate_WithSignature(t *testing.T) {
	// Создаем хранилище с включенной подписью запросов
	privateKey := "test-key"
	cfg := &app.Config{
		PrivateKey: privateKey,
	}
	logger := zap.NewNop()
	repo := memory.New(cfg, logger)

	// Создаем обработчик
	handler := updates.NewBulkUpdate(cfg, repo, logger)

	// Создаем тестовые метрики
	gaugeValue := 42.5
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)

	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)

	// Создаем массив метрик для обновления
	metricsToUpdate := []metrics.Metric{*gaugeMetric, *counterMetric}

	// Сериализуем метрики в JSON
	body, err := json.Marshal(metricsToUpdate)
	require.NoError(t, err)

	// Создаем подпись
	hash := sign.ComputeHMACSHA256(body, privateKey)

	// Создаем HTTP-запрос
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(sign.HeaderKey, hash)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Проверяем, что метрики были сохранены в хранилище
	savedGaugeMetric, ok := repo.GetMetric(context.Background(), metrics.TypeGauge, "test_gauge")
	assert.True(t, ok)
	assert.InDelta(t, gaugeValue, *savedGaugeMetric.Value, 0.0001)

	savedCounterMetric, ok := repo.GetMetric(
		context.Background(),
		metrics.TypeCounter,
		"test_counter",
	)
	assert.True(t, ok)
	assert.Equal(t, counterValue, *savedCounterMetric.Delta)
}

func TestNewBulkUpdate_InvalidSignature(t *testing.T) {
	// Создаем хранилище с включенной подписью запросов
	privateKey := "test-key"
	cfg := &app.Config{
		PrivateKey: privateKey,
	}
	logger := zap.NewNop()
	repo := memory.New(cfg, logger)

	// Создаем обработчик
	handler := updates.NewBulkUpdate(cfg, repo, logger)

	// Создаем тестовые метрики
	gaugeValue := 42.5
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)

	// Сериализуем метрики в JSON
	body, err := json.Marshal([]metrics.Metric{*gaugeMetric})
	require.NoError(t, err)

	// Создаем неверную подпись
	invalidHash := "invalid-hash"

	// Создаем HTTP-запрос
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(sign.HeaderKey, invalidHash)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
