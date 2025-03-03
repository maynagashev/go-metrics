package update_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	jsonupdate "github.com/maynagashev/go-metrics/internal/server/handlers/json/update"
	plainupdate "github.com/maynagashev/go-metrics/internal/server/handlers/plain/update"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
	"github.com/maynagashev/go-metrics/pkg/sign"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestUpdateHandler is testing the plain update handler, not the JSON update handler
func TestUpdateHandler(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name    string
		target  string
		storage storage.Repository
		want
	}{
		{
			name:    "update gauge",
			target:  "/update/gauge/test_gauge/1.1",
			storage: memory.New(&app.Config{}, zap.NewNop()),
			want: want{
				code:        200,
				contentType: "text/plain",
			},
		},
		{
			name:    "update counter",
			target:  "/update/counter/test_counter/1",
			storage: memory.New(&app.Config{}, zap.NewNop()),
			want: want{
				code:        200,
				contentType: "text/plain",
			},
		},
		{
			name:    "invalid metrics type",
			target:  "/update/invalid/test_counter/1",
			storage: memory.New(&app.Config{}, zap.NewNop()),
			want: want{
				code:        400,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:    "invalid url",
			target:  "/update/gauge/1",
			storage: memory.New(&app.Config{}, zap.NewNop()),
			want: want{
				code:        404,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.target, nil)
			w := httptest.NewRecorder()
			// This is testing the plain update handler, not the JSON update handler
			plainHandler := plainupdate.New(tt.storage, zap.NewNop())
			plainHandler(w, request)

			res := w.Result()
			assert.Equal(t, tt.want.code, res.StatusCode)

			defer res.Body.Close()

			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.NotEmpty(t, string(resBody))
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestJSONUpdateHandler_Gauge(t *testing.T) {
	// Создаем хранилище
	cfg := &app.Config{}
	logger := zap.NewNop()
	repo := memory.New(cfg, logger)

	// Создаем обработчик
	handler := jsonupdate.New(cfg, repo, logger)

	// Создаем тестовый запрос с метрикой типа gauge
	value := 42.5
	metric := jsonupdate.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
		Value: &value,
	}

	body, err := json.Marshal(metric)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Проверяем, что метрика была сохранена в хранилище
	savedMetric, ok := repo.GetMetric(context.Background(), metrics.TypeGauge, "test_gauge")
	assert.True(t, ok)
	assert.Equal(t, value, *savedMetric.Value)
}

func TestJSONUpdateHandler_Counter(t *testing.T) {
	// Создаем хранилище
	cfg := &app.Config{}
	logger := zap.NewNop()
	repo := memory.New(cfg, logger)

	// Создаем обработчик
	handler := jsonupdate.New(cfg, repo, logger)

	// Создаем тестовый запрос с метрикой типа counter
	delta := int64(10)
	metric := jsonupdate.Metric{
		Name:  "test_counter",
		MType: metrics.TypeCounter,
		Delta: &delta,
	}

	body, err := json.Marshal(metric)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Проверяем, что метрика была сохранена в хранилище
	savedMetric, ok := repo.GetMetric(context.Background(), metrics.TypeCounter, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, delta, *savedMetric.Delta)

	// Обновляем метрику еще раз
	req = httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	handler(rr, req)

	// Проверяем, что значение метрики увеличилось
	savedMetric, ok = repo.GetMetric(context.Background(), metrics.TypeCounter, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, delta*2, *savedMetric.Delta)
}

func TestJSONUpdateHandler_InvalidJSON(t *testing.T) {
	// Создаем хранилище
	cfg := &app.Config{}
	logger := zap.NewNop()
	repo := memory.New(cfg, logger)

	// Создаем обработчик
	handler := jsonupdate.New(cfg, repo, logger)

	// Создаем тестовый запрос с невалидным JSON
	invalidJSON := []byte(`{"id": "test", "type": "gauge", "value": invalid}`)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestJSONUpdateHandler_WithSignature(t *testing.T) {
	// Создаем хранилище с включенной подписью запросов
	privateKey := "test-key"
	cfg := &app.Config{
		PrivateKey: privateKey,
	}
	logger := zap.NewNop()
	repo := memory.New(cfg, logger)

	// Создаем обработчик
	handler := jsonupdate.New(cfg, repo, logger)

	// Создаем тестовый запрос с метрикой типа gauge
	value := 42.5
	metric := jsonupdate.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
		Value: &value,
	}

	body, err := json.Marshal(metric)
	require.NoError(t, err)

	// Создаем подпись
	hash := sign.ComputeHMACSHA256(body, privateKey)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(sign.HeaderKey, hash)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Проверяем, что метрика была сохранена в хранилище
	savedMetric, ok := repo.GetMetric(context.Background(), metrics.TypeGauge, "test_gauge")
	assert.True(t, ok)
	assert.Equal(t, value, *savedMetric.Value)
}

func TestJSONUpdateHandler_InvalidSignature(t *testing.T) {
	// Создаем хранилище с включенной подписью запросов
	privateKey := "test-key"
	cfg := &app.Config{
		PrivateKey: privateKey,
	}
	logger := zap.NewNop()
	repo := memory.New(cfg, logger)

	// Создаем обработчик
	handler := jsonupdate.New(cfg, repo, logger)

	// Создаем тестовый запрос с метрикой типа gauge
	value := 42.5
	metric := jsonupdate.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
		Value: &value,
	}

	body, err := json.Marshal(metric)
	require.NoError(t, err)

	// Создаем неверную подпись
	invalidHash := "invalid-hash"

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(sign.HeaderKey, invalidHash)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
