package main

import (
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/router"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
	"go.uber.org/zap"
)

func main() {
	log := initLogger()
	defer func() {
		_ = log.Sync()
	}()

	flags, err := parseFlags()
	if err != nil {
		// Если не удалось распарсить флаги запуска, завершаем программу.
		panic(err)
	}

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
