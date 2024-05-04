package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maynagashev/go-metrics/internal/server/app"
	jsonUpdate "github.com/maynagashev/go-metrics/internal/server/handlers/json/update"
	jasonValue "github.com/maynagashev/go-metrics/internal/server/handlers/json/value"
	plainIndex "github.com/maynagashev/go-metrics/internal/server/handlers/plain/index"
	plainUpdate "github.com/maynagashev/go-metrics/internal/server/handlers/plain/update"
	plainValue "github.com/maynagashev/go-metrics/internal/server/handlers/plain/value"
	"github.com/maynagashev/go-metrics/internal/server/middleware/decompress"
	"github.com/maynagashev/go-metrics/internal/server/middleware/logger"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"go.uber.org/zap"
)

// New инстанцирует новый роутер.
func New(server *app.Server, storage storage.Repository, log *zap.Logger) chi.Router {
	compressLevel := 5

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	// Используем единый логгер для запросов, вместо встроенного логгера chi
	r.Use(logger.New(log))
	// Добавляем middleware для сжатия ответов
	r.Use(middleware.Compress(compressLevel, "application/json", "text/html"))
	// Обработка сжатых запросов, когда от клиента сразу пришел заголовок Content-Encoding: gzip
	r.Use(decompress.New(log))

	r.Get("/", plainIndex.New(storage))
	r.Post("/update", jsonUpdate.New(server, storage, log))
	r.Post("/value", jasonValue.New(storage))

	// Первые версии обработчиков для работы тестов начальных итераций
	r.Post("/update/*", plainUpdate.New(storage, log))
	r.Get("/value/{type}/{name}", plainValue.New(storage))

	return r
}
