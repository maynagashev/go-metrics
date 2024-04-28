package logger

import (
	"bytes"
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
			)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Сохраняем тело ответа для записи в лог
			body := bytes.NewBuffer(nil)
			ww.Tee(body)

			t1 := time.Now()
			defer func() {
				entry.Info("request completed",
					zap.Int("status", ww.Status()),
					zap.Int("bytes", ww.BytesWritten()),
					zap.String("duration", time.Since(t1).String()),
					zap.String("response_body", body.String()), // Логирование тела ответа
				)
			}()

			next.ServeHTTP(ww, r)
		}

		// приводим к нужному типу
		return http.HandlerFunc(fn)
	}
}
