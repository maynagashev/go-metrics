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
	// Создаем конфигурацию для регистратора в режиме разработки
	cfg := zap.NewDevelopmentConfig()

	// Указываем путь к файлу для записи логов
	cfg.OutputPaths = []string{"./run.log", "stderr"}

	// Создаем регистратор с заданной конфигурацией
	logger, err := cfg.Build()
	if err != nil {
		// вызываем панику, если ошибка
		panic(err)
	}
	return logger
}
