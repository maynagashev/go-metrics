// Package logger реализует middleware для логирования HTTP-запросов.
// Обеспечивает логирование всех входящих запросов и их результатов.
package logger

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/go-chi/chi/v5/middleware"
)

func New(log *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		log.Info("logger middleware enabled")

		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := log.With(
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
				zap.String("request_id", middleware.GetReqID(r.Context())),
				// Добавляем логирование заголовков запроса
				zap.Any("headers", r.Header),
				// Добавляем логирование тела запроса
				zap.String("request_body", string(readRequestBody(r, log))),
			)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Сохраняем тело ответа для записи в лог
			body := bytes.NewBuffer(nil)
			ww.Tee(body)

			t1 := time.Now()
			defer func() {
				entry.Info("request completed",
					zap.Int("status", ww.Status()),
					zap.Int("response_bytes", ww.BytesWritten()),
					zap.Any("response_headers", ww.Header()),   // Логирование заголовков ответа
					zap.String("response_body", body.String()), // Логирование тела ответа
					zap.String("duration", time.Since(t1).String()),
				)
			}()

			next.ServeHTTP(ww, r)
		}

		// приводим к нужному типу
		return http.HandlerFunc(fn)
	}
}

func readRequestBody(r *http.Request, log *zap.Logger) []byte {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Ошибка при чтении тела запроса", zap.Error(err))
		return nil
	}
	defer func() {
		_ = r.Body.Close()
	}()

	// Восстановление r.Body для дальнейшего использования
	r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
	return reqBody
}
