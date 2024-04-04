package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maynagashev/go-metrics/internal/server/handlers/index"
	"github.com/maynagashev/go-metrics/internal/server/handlers/update"
	"github.com/maynagashev/go-metrics/internal/server/handlers/value"
	"github.com/maynagashev/go-metrics/internal/server/storage"
)

// New инстанцирует новый роутер.
func New(st storage.Repository) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", index.New(st))
	r.Post("/update/*", update.New(st))
	r.Get("/value/{type}/{name}", value.New(st))

	return r
}
