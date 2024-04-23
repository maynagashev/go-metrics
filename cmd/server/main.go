package main

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/storage/memory"

	"github.com/maynagashev/go-metrics/internal/server/router"
)

const (
	DefaultReadTimeout  = 5 * time.Second
	DefaultWriteTimeout = 10 * time.Second
	DefaultIdleTimeout  = 120 * time.Second
)

func main() {
	flags := mustParseFlags()

	// Инициализируем сторонний логгер
	log := initLogger()
	defer func() {
		_ = log.Sync()
	}()

	log.Info("starting server", zap.String("addr", flags.Server.Addr))

	server := &http.Server{
		Addr:    flags.Server.Addr,
		Handler: router.New(memory.New(), log),
		// Настройка таймаутов для сервера по рекомендациям линтера gosec
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
		IdleTimeout:  DefaultIdleTimeout,
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Info("server failed to start", zap.Error(err))
	}
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
