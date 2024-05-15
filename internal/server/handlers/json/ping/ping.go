package ping

import (
	"context"
	"net/http"

	"github.com/maynagashev/go-metrics/internal/lib/api/response"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage/pgsql"
	"go.uber.org/zap"
)

type Response struct {
	response.Response
}

// New возвращает http.HandlerFunc.
func New(config *app.Config, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Проверяем подключение к базе данных
		_, err := pgsql.New(context.Background(), config, log)

		if err != nil {
			response.Error(w, err, http.StatusInternalServerError)
			return
		}

		response.OK(w, "pong")
	}
}
