package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	jsonUpdate "github.com/maynagashev/go-metrics/internal/server/handlers/json/update"
	jasonValue "github.com/maynagashev/go-metrics/internal/server/handlers/json/value"
	"github.com/maynagashev/go-metrics/internal/server/handlers/plain/index"
	plainUpdate "github.com/maynagashev/go-metrics/internal/server/handlers/plain/update"
	plainValue "github.com/maynagashev/go-metrics/internal/server/handlers/plain/value"
	logger "github.com/maynagashev/go-metrics/internal/server/middleware"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"go.uber.org/zap"
)

// New инстанцирует новый роутер.
func New(st storage.Repository, log *zap.Logger) chi.Router {
	compressLevel := 5

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	// Добавляем middleware для сжатия ответов
	r.Use(middleware.Compress(compressLevel, "application/json", "text/html"))
	r.Use(logger.New(log)) // используем единый логгер для запросов, вместо встроенного логгера chi

	r.Get("/", index.New(st))
	r.Post("/update", jsonUpdate.New(st, log))
	r.Post("/value", jasonValue.New(st))

	r.Post("/update/*", plainUpdate.New(st, log))
	r.Get("/value/{type}/{name}", plainValue.New(st))

	return r
}
