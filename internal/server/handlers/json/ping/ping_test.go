package ping_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/handlers/json/ping"
	"github.com/maynagashev/go-metrics/mocks"

	"github.com/stretchr/testify/assert"
)

func TestHandle_Success(t *testing.T) {
	// Создаем новый мок для интерфейса Storage
	mockStorage := new(mocks.Storage)

	// Настраиваем мок, чтобы метод GetMetrics возвращал не пустое значение
	mockStorage.On("GetMetrics", context.Background()).Return([]metrics.Metric{
		*metrics.NewCounter("metric1", 1),
		*metrics.NewCounter("metric1", 2),
	})

	// Создаем HTTP-запрос для теста
	req, err := http.NewRequest(http.MethodGet, "/ping", nil)
	require.NoError(t, err)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Создаем обработчик с использованием мокированного хранилища
	handler := ping.Handle(mockStorage)

	// Вызываем обработчик с записанным запросом и ответом
	handler.ServeHTTP(rr, req)

	// Проверяем, что код ответа равен 200
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"status":"OK","message":"pong"}`, rr.Body.String())

	// Проверяем вызов метода GetMetrics
	mockStorage.AssertCalled(t, "GetMetrics", context.Background())
}
