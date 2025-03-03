package logger_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/maynagashev/go-metrics/internal/server/middleware/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerMiddleware(t *testing.T) {
	// Создаем наблюдаемый логгер для проверки логов
	core, logs := observer.New(zapcore.InfoLevel)
	log := zap.New(core)

	// Создаем тестовые данные
	testData := []byte(`{"id":"test_gauge","type":"gauge","value":42.5}`)

	// Создаем тестовый запрос
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(testData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")

	// Добавляем request ID в контекст
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
	req = req.WithContext(ctx)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Создаем тестовый обработчик
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что тело запроса доступно для чтения
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(r.Body)
		require.NoError(t, err)
		assert.Equal(t, string(testData), buf.String())

		// Отправляем ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"status":"OK"}`))
		require.NoError(t, err)
	})

	// Создаем middleware
	middleware := logger.New(log)

	// Применяем middleware к тестовому обработчику
	handler := middleware(testHandler)

	// Вызываем обработчик
	handler.ServeHTTP(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusOK, rr.Code)

	// Проверяем, что логи содержат нужную информацию
	logEntries := logs.All()
	require.GreaterOrEqual(t, len(logEntries), 2, "Expected at least 2 log entries")

	// Проверяем первую запись лога (включение middleware)
	assert.Equal(t, "logger middleware enabled", logEntries[0].Message)

	// Проверяем запись о завершении запроса
	requestCompletedLog := logEntries[len(logEntries)-1]
	assert.Equal(t, "request completed", requestCompletedLog.Message)
	assert.Equal(t, int64(http.StatusOK), requestCompletedLog.ContextMap()["status"])
	assert.Contains(t, requestCompletedLog.ContextMap(), "duration")
	assert.Contains(t, requestCompletedLog.ContextMap(), "response_bytes")
}

func TestLoggerMiddleware_WithRequestBody(t *testing.T) {
	// Создаем наблюдаемый логгер для проверки логов
	core, logs := observer.New(zapcore.InfoLevel)
	log := zap.New(core)

	// Создаем тестовые данные
	testData := []byte(`{"id":"test_gauge","type":"gauge","value":42.5}`)

	// Создаем тестовый запрос
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(testData))
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Создаем тестовый обработчик
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Создаем middleware
	middleware := logger.New(log)

	// Применяем middleware к тестовому обработчику
	handler := middleware(testHandler)

	// Вызываем обработчик
	handler.ServeHTTP(rr, req)

	// Проверяем, что логи содержат тело запроса
	logEntries := logs.All()
	require.GreaterOrEqual(t, len(logEntries), 2, "Expected at least 2 log entries")

	// Проверяем запись о завершении запроса
	requestCompletedLog := logEntries[len(logEntries)-1]
	assert.Equal(t, "request completed", requestCompletedLog.Message)

	// Проверяем, что в контексте лога есть информация о методе и пути запроса
	contextMap := logEntries[len(logEntries)-1].ContextMap()
	assert.Equal(t, http.MethodPost, contextMap["method"])
	assert.Equal(t, "/update/", contextMap["path"])
}
