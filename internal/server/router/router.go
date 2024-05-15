package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/handlers/json/ping"
	jsonUpdate "github.com/maynagashev/go-metrics/internal/server/handlers/json/update"
	jsonUpdates "github.com/maynagashev/go-metrics/internal/server/handlers/json/updates"
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
func New(config *app.Config, storage storage.Repository, log *zap.Logger) chi.Router {
	compressLevel := 5

	r := chi.NewRouter()

	// Добавляем middleware для генерации ID запроса
	r.Use(middleware.RequestID)
	// Восстанавливаем панику, если она произошла внутри обработчика
	r.Use(middleware.Recoverer)
	// Удаляем слеши в конце URL
	r.Use(middleware.StripSlashes)
	// Добавляем middleware для сжатия ответов
	r.Use(middleware.Compress(compressLevel, "application/json", "text/html"))
	// Обработка сжатых запросов, когда от клиента сразу пришел заголовок Content-Encoding: gzip
	r.Use(decompress.New(log))
	// Используем единый логгер для запросов, вместо встроенного логгера chi
	r.Use(logger.New(log))

	// Обработчики запросов
	r.Get("/", plainIndex.New(storage))
	r.Post("/update", jsonUpdate.New(config, storage, log))
	r.Post("/updates", jsonUpdates.NewBulkUpdate(storage, log))
	r.Post("/value", jasonValue.New(storage))
	r.Get("/ping", ping.New(config, log))

	// Первые версии обработчиков для работы тестов начальных итераций
	r.Post("/update/*", plainUpdate.New(storage, log))
	r.Get("/value/{type}/{name}", plainValue.New(storage))

	return r
}
