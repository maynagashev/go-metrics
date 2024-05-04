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
	cfg Config
}

type Config struct {
	Addr string
	// Интервал сохранения метрик на сервере в секундах.
	StoreInterval int
	// Полное имя файла, в который будут сохранены метрики.
	FileStoragePath string
	// Загружать или нет ранее сохраненные метрики из файла.
	Restore bool
}

func New(cfg Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

// IsStoreEnabled возвращает true, если включено сохранение метрик на сервере.
func (s *Server) IsStoreEnabled() bool {
	return s.cfg.FileStoragePath != ""
}

// IsRestoreEnabled надо ли восстанавливать метрики из файла при старте.
func (s *Server) IsRestoreEnabled() bool {
	return s.cfg.Restore
}

func (s *Server) GetStorePath() string {
	return s.cfg.FileStoragePath
}

// IsSyncStore сохранение метрик на сервере синхронно (сразу после изменения, если нулевой интервал).
func (s *Server) IsSyncStore() bool {
	return s.cfg.StoreInterval == 0
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
