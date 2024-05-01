package main

import (
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/router"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
	"go.uber.org/zap"
)

func main() {
	flags := mustParseFlags()

	// Инициализируем сторонний логгер
	log := initLogger()
	defer func() {
		_ = log.Sync()
	}()

	cfg := app.Config{
		Addr:            flags.Server.Addr,
		StoreInterval:   flags.Server.StoreInterval,
		FileStoragePath: flags.Server.FileStoragePath,
		Restore:         flags.Server.Restore,
	}

	server := app.New(cfg)
	storage := memory.New(server, log)

	handlers := router.New(server, storage, log)
	server.Start(log, handlers)
}

func initLogger() *zap.Logger {
	// создаём предустановленный регистратор zap
	logger, err := zap.NewDevelopment()
	if err != nil {
		// вызываем панику, если ошибка
		panic(err)
	}
	return logger
}
