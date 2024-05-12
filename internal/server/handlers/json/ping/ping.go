package ping

import (
	"context"
	"net/http"

	"github.com/go-chi/render"
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
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Проверяем подключение к базе данных
		_, err := pgsql.New(context.Background(), config, log)

		if err != nil {
			responseError(w, r, err)
			return
		}

		responseOK(w, r)
	}
}

func responseError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	render.JSON(w, r, response.Error(err.Error()))
}

func responseOK(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, Response{
		Response: response.OK(),
	})
}
