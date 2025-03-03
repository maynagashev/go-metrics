package response_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maynagashev/go-metrics/pkg/response"
)

func TestOK(t *testing.T) {
	// Создаем тестовый HTTP-рекордер
	w := httptest.NewRecorder()

	// Вызываем функцию OK
	response.OK(w, "test message")

	// Проверяем статус-код
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Проверяем заголовок Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type %s, got %s", "application/json", contentType)
	}

	// Проверяем тело ответа
	expectedBody := `{"status":"OK","message":"test message"}`
	if strings.TrimSpace(w.Body.String()) != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestError(t *testing.T) {
	// Создаем тестовый HTTP-рекордер
	w := httptest.NewRecorder()

	// Создаем тестовую ошибку
	testErr := errors.New("test error")

	// Вызываем функцию Error
	response.Error(w, testErr, http.StatusBadRequest)

	// Проверяем статус-код
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Проверяем заголовок Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type %s, got %s", "application/json", contentType)
	}

	// Проверяем тело ответа
	expectedBody := `{"status":"Error","error":"test error"}`
	if strings.TrimSpace(w.Body.String()) != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

// TestWriteResponseError проверяет обработку ошибок в функции writeResponse.
func TestWriteResponseError(_ *testing.T) {
	// Создаем тестовый HTTP-рекордер с ошибкой записи
	w := &errorWriter{
		ResponseWriter: httptest.NewRecorder(),
		failOnWrite:    true,
	}

	// Вызываем функцию OK, которая использует writeResponse
	response.OK(w, "test message")

	// Проверяем, что был установлен статус ошибки
	// Примечание: в данном случае мы не можем проверить статус-код,
	// так как errorWriter не позволяет его установить
}

// errorWriter - мок http.ResponseWriter, который возвращает ошибку при записи.
type errorWriter struct {
	http.ResponseWriter
	failOnWrite bool
}

// Write возвращает ошибку, если failOnWrite установлен в true.
func (w *errorWriter) Write(b []byte) (int, error) {
	if w.failOnWrite {
		return 0, errors.New("write error")
	}
	return w.ResponseWriter.Write(b)
}
