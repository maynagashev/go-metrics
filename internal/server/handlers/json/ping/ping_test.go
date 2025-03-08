package ping_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/handlers/json/ping"
	"github.com/maynagashev/go-metrics/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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

func TestNew_DatabaseNotEnabled(t *testing.T) {
	// Создаем конфиг без базы данных
	cfg := &app.Config{
		Database: app.DatabaseConfig{
			DSN: "", // Пустой DSN означает, что база данных не включена
		},
	}

	// Создаем логгер
	logger, _ := zap.NewDevelopment()

	// Создаем обработчик
	handler := ping.New(cfg, logger)

	// Создаем HTTP-запрос для теста
	req, err := http.NewRequest(http.MethodGet, "/ping", nil)
	require.NoError(t, err)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик с записанным запросом и ответом
	handler.ServeHTTP(rr, req)

	// Проверяем, что код ответа равен 500 (Internal Server Error)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "не указана база данных")
}

func TestNew_InvalidDatabaseConfig(t *testing.T) {
	// Создаем конфиг с неверными параметрами базы данных
	cfg := &app.Config{
		Database: app.DatabaseConfig{
			DSN: "invalid-dsn", // Неверный DSN вызовет ошибку при подключении
		},
	}

	// Создаем логгер
	logger, _ := zap.NewDevelopment()

	// Создаем обработчик
	handler := ping.New(cfg, logger)

	// Создаем HTTP-запрос для теста
	req, err := http.NewRequest(http.MethodGet, "/ping", nil)
	require.NoError(t, err)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик с записанным запросом и ответом
	handler.ServeHTTP(rr, req)

	// Проверяем, что код ответа равен 500 (Internal Server Error)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	// Проверяем, что в ответе содержится сообщение об ошибке
	assert.Contains(t, rr.Body.String(), "error")
}

// Добавляем тест для случая, когда база данных включена, но возвращает ошибку.
func TestNew_DatabaseError(t *testing.T) {
	// Создаем мок для хранилища, который будет возвращать ошибку
	mockStorage := new(mocks.Storage)
	mockStorage.On("GetMetrics", context.Background()).Return([]metrics.Metric{}).Once()

	// Создаем обработчик, который будет использовать мок вместо реальной базы данных
	handler := ping.Handle(mockStorage)

	// Создаем HTTP-запрос для теста
	req, err := http.NewRequest(http.MethodGet, "/ping", nil)
	require.NoError(t, err)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик с записанным запросом и ответом
	handler.ServeHTTP(rr, req)

	// Проверяем, что код ответа равен 200 (OK)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "pong")

	// Проверяем вызов метода GetMetrics
	mockStorage.AssertExpectations(t)
}

// Добавляем тест для проверки случая с пустым ответом от базы данных.
func TestHandle_EmptyResponse(t *testing.T) {
	// Создаем новый мок для интерфейса Storage
	mockStorage := new(mocks.Storage)

	// Настраиваем мок, чтобы метод GetMetrics возвращал пустой слайс
	mockStorage.On("GetMetrics", context.Background()).Return([]metrics.Metric{})

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

// Добавляем тест для проверки различных HTTP методов.
func TestHandle_DifferentMethods(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			// Создаем новый мок для интерфейса Storage
			mockStorage := new(mocks.Storage)

			// Настраиваем мок, чтобы метод GetMetrics возвращал не пустое значение
			mockStorage.On("GetMetrics", context.Background()).Return([]metrics.Metric{
				*metrics.NewCounter("metric1", 1),
			})

			// Создаем HTTP-запрос с текущим методом
			req, err := http.NewRequest(method, "/ping", nil)
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
		})
	}
}

// Добавляем тест для проверки работы с контекстом запроса.
func TestHandle_WithContext(t *testing.T) {
	// Создаем новый мок для интерфейса Storage
	mockStorage := new(mocks.Storage)

	// Создаем контекст с значением
	ctx := context.WithValue(context.Background(), "test-key", "test-value")

	// Настраиваем мок, чтобы метод GetMetrics ожидал контекст с нашим значением
	mockStorage.On("GetMetrics", ctx).Return([]metrics.Metric{
		*metrics.NewCounter("metric1", 1),
	})

	// Создаем HTTP-запрос с нашим контекстом
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/ping", nil)
	require.NoError(t, err)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Создаем обработчик с использованием мокированного хранилища
	handler := ping.Handle(mockStorage)

	// Вызываем обработчик с записанным запросом и ответом
	handler.ServeHTTP(rr, req)

	// Проверяем, что код ответа равен 200
	assert.Equal(t, http.StatusOK, rr.Code)

	// Проверяем вызов метода GetMetrics с правильным контекстом
	mockStorage.AssertCalled(t, "GetMetrics", ctx)
}
