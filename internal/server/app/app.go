package app

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	DefaultReadTimeout  = 5 * time.Second
	DefaultWriteTimeout = 10 * time.Second
	DefaultIdleTimeout  = 120 * time.Second
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

	err := httpServer.ListenAndServe()
	if err != nil {
		log.Fatal("server failed to start", zap.Error(err))
	}
}

// GetStoreInterval возвращает интервал сохранения метрик в секундах.
func (s *Server) GetStoreInterval() int {
	return s.cfg.StoreInterval
}
