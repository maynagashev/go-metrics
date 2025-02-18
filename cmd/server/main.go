// Package main реализует HTTP-сервер для сбора и хранения метрик.
//
// Сервер поддерживает хранение метрик в PostgreSQL или в памяти. Выбор хранилища
// определяется наличием параметров подключения к БД (флаг -d или переменная DATABASE_DSN).
//
// # Поддерживаемые типы метрик
//
//   - gauge - число с плавающей точкой
//   - counter - целочисленный счетчик
//
// # API Endpoints
//
//   - POST /update - обновление одиночной метрики
//   - POST /updates/ - пакетное обновление метрик
//   - POST /value - получение значения метрики
//   - GET /ping - проверка подключения к БД
//   - GET / - получение всех метрик (текстовый формат)
//
// # Конфигурация
//
// Сервер поддерживает настройку через флаги командной строки и переменные окружения:
//   - DATABASE_DSN - строка подключения к PostgreSQL
//   - STORE_INTERVAL - интервал сохранения метрик (для in-memory хранилища)
//   - FILE_STORAGE_PATH - путь к файлу для сохранения метрик
//   - RESTORE - восстанавливать ли метрики из файла при старте
//
// # Примеры
//
// Примеры использования API представлены в тестах:
//   - Example - обновление метрики
//   - Example_getValue - получение значения
//   - Example_updateBatch - пакетное обновление
//   - Example_ping - проверка БД
package main

import (
	"context"
	//nolint:gosec // G108: pprof is used intentionally for debugging and profiling
	_ "net/http/pprof"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/router"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
	"github.com/maynagashev/go-metrics/internal/server/storage/pgstorage"
	"go.uber.org/zap"
)

func main() {
	flags, err := app.ParseFlags()
	if err != nil {
		panic(err)
	}

	log := initLogger()
	defer func() {
		_ = log.Sync()
	}()

	cfg := app.NewConfig(flags)
	server := app.New(cfg)

	// Инициализируем хранилище
	repo, err := initStorage(cfg, log)
	if err != nil {
		log.Error("failed to init storage", zap.Error(err))
		panic(err)
	}
	defer func() {
		err = repo.Close()
		if err != nil {
			log.Error("failed to close storage", zap.Error(err))
		}
	}()

	handlers := router.New(cfg, repo, log)

	server.Start(log, handlers)

	log.Debug("server stopped")
}

func initStorage(cfg *app.Config, log *zap.Logger) (storage.Repository, error) {
	// Если указан DATABASE_DSN или флаг -d, то используем PostgreSQL.
	if cfg.IsDatabaseEnabled() {
		pg, err := pgstorage.New(context.Background(), cfg, log)
		if err != nil {
			return nil, err
		}
		return pg, nil
	}

	return memory.New(cfg, log), nil
}

func initLogger() *zap.Logger {
	// Создаем конфигурацию для регистратора в режиме разработки
	cfg := zap.NewDevelopmentConfig()

	// Указываем путь к файлу для записи логов, для записи в файл добавить в список например: "../../run.log"
	cfg.OutputPaths = []string{"stderr"}

	// Создаем регистратор с заданной конфигурацией
	logger, err := cfg.Build()
	if err != nil {
		// вызываем панику, если ошибка
		panic(err)
	}
	return logger
}
