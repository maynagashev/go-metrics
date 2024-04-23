package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maynagashev/go-metrics/internal/server/handlers/index"
	"github.com/maynagashev/go-metrics/internal/server/handlers/update"
	"github.com/maynagashev/go-metrics/internal/server/handlers/value"
	logger "github.com/maynagashev/go-metrics/internal/server/middleware"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"go.uber.org/zap"
)

// New инстанцирует новый роутер.
func New(st storage.Repository, log *zap.Logger) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(logger.New(log)) // используем единый логгер для запросов, вместо встроенного логгера chi
	r.Use(middleware.Recoverer)

	r.Get("/", index.New(st))
	r.Post("/update/*", update.New(st, log))
	r.Get("/value/{type}/{name}", value.New(st))

	return r
}
