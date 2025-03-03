package decompress_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maynagashev/go-metrics/internal/server/middleware/decompress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// compressData сжимает данные с помощью gzip
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestDecompressMiddleware_WithGzip(t *testing.T) {
	// Создаем тестовые данные
	testData := []byte(`{"id":"test_gauge","type":"gauge","value":42.5}`)

	// Сжимаем данные
	compressedData, err := compressData(testData)
	require.NoError(t, err)

	// Создаем тестовый запрос со сжатыми данными
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(compressedData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Создаем тестовый обработчик, который будет проверять, что данные были распакованы
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Читаем тело запроса
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		// Проверяем, что данные были распакованы
		assert.Equal(t, testData, body)

		// Отправляем успешный ответ
		w.WriteHeader(http.StatusOK)
	})

	// Создаем middleware
	logger, _ := zap.NewDevelopment()
	middleware := decompress.New(logger)

	// Применяем middleware к тестовому обработчику
	handler := middleware(testHandler)

	// Вызываем обработчик
	handler.ServeHTTP(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDecompressMiddleware_WithoutGzip(t *testing.T) {
	// Создаем тестовые данные
	testData := []byte(`{"id":"test_gauge","type":"gauge","value":42.5}`)

	// Создаем тестовый запрос с несжатыми данными
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(testData))
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Создаем тестовый обработчик, который будет проверять, что данные не изменились
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Читаем тело запроса
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		// Проверяем, что данные не изменились
		assert.Equal(t, testData, body)

		// Отправляем успешный ответ
		w.WriteHeader(http.StatusOK)
	})

	// Создаем middleware
	logger, _ := zap.NewDevelopment()
	middleware := decompress.New(logger)

	// Применяем middleware к тестовому обработчику
	handler := middleware(testHandler)

	// Вызываем обработчик
	handler.ServeHTTP(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDecompressMiddleware_InvalidGzip(t *testing.T) {
	// Создаем тестовые данные с невалидным gzip
	invalidGzip := []byte("invalid gzip data")

	// Создаем тестовый запрос с невалидными сжатыми данными
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(invalidGzip))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Создаем тестовый обработчик, который не должен быть вызван
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid gzip data")
	})

	// Создаем middleware
	logger, _ := zap.NewDevelopment()
	middleware := decompress.New(logger)

	// Применяем middleware к тестовому обработчику
	handler := middleware(testHandler)

	// Вызываем обработчик
	handler.ServeHTTP(rr, req)

	// Проверяем код ответа
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
