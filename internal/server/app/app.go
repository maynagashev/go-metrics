// Package app реализует основную логику работы HTTP-сервера.
// Содержит инициализацию и запуск сервера, а также обработку конфигурации.
package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

const (
	DefaultReadTimeout  = 5 * time.Second
	DefaultWriteTimeout = 10 * time.Second
	DefaultIdleTimeout  = 120 * time.Second
	ShutdownTimeout     = 30 * time.Second
)

// Server представляет собой HTTP-сервер для сбора метрик.
// Обрабатывает запросы от агентов и сохраняет метрики в хранилище.
type Server struct {
	cfg *Config
}

// New создает новый экземпляр сервера с указанной конфигурацией.
func New(cfg *Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

// Start запускает HTTP-сервер с указанным обработчиком и логгером.
// Настраивает таймауты и другие параметры сервера.
// Обрабатывает сигналы SIGTERM, SIGINT, SIGQUIT для graceful shutdown.
func (s *Server) Start(log *zap.Logger, handler http.Handler) {
	log.Info("starting server", zap.Any("config", s.cfg))

	httpServer := &http.Server{
		Addr:    s.cfg.Addr,
		Handler: handler,
		// Настройка таймаутов для сервера по рекомендациям линтера gosec
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
		IdleTimeout:  DefaultIdleTimeout,
	}

	// Канал для получения ошибок от запущенного сервера
	serverErrors := make(chan error, 1)

	// Запускаем сервер в отдельной горутине
	go func() {
		log.Info("server is listening", zap.String("addr", s.cfg.Addr))
		serverErrors <- httpServer.ListenAndServe()
	}()

	// Канал для получения сигналов от ОС
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Блокируемся до получения сигнала или ошибки
	select {
	case err := <-serverErrors:
		log.Fatal("server error", zap.Error(err))
	case sig := <-shutdown:
		log.Info("shutdown signal received", zap.String("signal", sig.String()))

		// Создаем контекст с таймаутом для graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		// Сначала останавливаем прием новых запросов
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Error("server shutdown failed", zap.Error(err))
			if closeErr := httpServer.Close(); closeErr != nil {
				log.Error("server close failed", zap.Error(closeErr))
			}
		} else {
			log.Info("server shutdown completed")
		}

		// Даем время на завершение текущих операций и сохранение данных
		select {
		case <-ctx.Done():
			log.Error("shutdown timeout exceeded")
		case <-time.After(5 * time.Second): // Дополнительное время для сохранения данных
			log.Info("all pending operations completed")
		}
	}
}

// GetStoreInterval возвращает интервал сохранения метрик в секундах.
func (s *Server) GetStoreInterval() int {
	return s.cfg.StoreInterval
}
