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

// Server (HTTP-клиент) для сбора рантайм-метрик от агентов.
type Server struct {
	cfg *Config
}

func New(cfg *Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

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

func (s *Server) GetStoreInterval() int {
	return s.cfg.StoreInterval
}
