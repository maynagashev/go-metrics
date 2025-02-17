package router

import (
	"net/http/pprof"

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
	"github.com/maynagashev/go-metrics/internal/server/middleware/decompresspool"
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
	r.Use(decompresspool.New(log))
	// Используем единый логгер для запросов, вместо встроенного логгера chi
	r.Use(logger.New(log))

	// Обработчики запросов
	r.Get("/", plainIndex.New(storage))
	r.Post("/update", jsonUpdate.New(config, storage, log))
	r.Post("/updates", jsonUpdates.NewBulkUpdate(config, storage, log))
	r.Post("/value", jasonValue.New(config, storage))
	r.Get("/ping", ping.New(config, log))

	// Первые версии обработчиков для работы тестов начальных итераций
	r.Post("/update/*", plainUpdate.New(storage, log))
	r.Get("/value/{type}/{name}", plainValue.New(storage))

	// Добавляем pprof хендлеры только если включено профилирование
	if config.EnablePprof {
		log.Info("Registering pprof handlers at /debug/pprof/")
		r.Route("/debug/pprof", func(r chi.Router) {
			r.HandleFunc("/", pprof.Index)
			r.HandleFunc("/cmdline", pprof.Cmdline)
			r.HandleFunc("/profile", pprof.Profile)
			r.HandleFunc("/symbol", pprof.Symbol)
			r.HandleFunc("/trace", pprof.Trace)

			/* Постоянно обновляемые метрики, снимают отчеты в реальном времени */
			r.HandleFunc("/goroutine", pprof.Handler("goroutine").ServeHTTP)
			r.HandleFunc("/heap", pprof.Handler("heap").ServeHTTP)
			r.HandleFunc("/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
			r.HandleFunc("/block", pprof.Handler("block").ServeHTTP)
			r.HandleFunc("/allocs", pprof.Handler("allocs").ServeHTTP)
			r.HandleFunc("/mutex", pprof.Handler("mutex").ServeHTTP)
		})
	}

	return r
}
