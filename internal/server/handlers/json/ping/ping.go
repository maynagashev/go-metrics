// Package ping реализует обработчик для проверки соединения с базой данных.
// Предоставляет эндпоинт для проверки работоспособности системы.
package ping

import (
	"context"
	"errors"
	"net/http"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/storage/pgstorage"

	"github.com/maynagashev/go-metrics/pkg/response"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"go.uber.org/zap"
)

type Response struct {
	response.Response
}

type Storage interface {
	GetMetrics(ctx context.Context) []metrics.Metric
}

// Handle логика обработчика ping с указанной базой данных, чтобы можно было провести тестирование моком.
func Handle(storage Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Проверяем подключение сделав запрос к базе данных.
		_ = storage.GetMetrics(r.Context())

		response.OK(w, "pong")
	}
}

// New создает подключение к базе данных из конфига и возвращает обработчик запроса.
func New(config *app.Config, log *zap.Logger) http.HandlerFunc {
	// Если не используется PostgreSQL, то возвращаем обработчик, который возвращает ошибку.
	if !config.IsDatabaseEnabled() {
		return func(w http.ResponseWriter, _ *http.Request) {
			response.Error(w, errors.New("не указана база данных"), http.StatusInternalServerError)
		}
	}

	// Создаем экземпляр хранилища на основе PostgreSQL, здесь создается подключение и накатываются миграции.
	db, err := pgstorage.New(context.Background(), config, log)
	// Если не удалось создать хранилище, то возвращаем обработчик, который возвращает ошибку.
	if err != nil {
		return func(w http.ResponseWriter, _ *http.Request) {
			response.Error(w, err, http.StatusInternalServerError)
		}
	}

	// Запускаем обработчик запроса с созданным хранилищем.
	return Handle(db)
}
