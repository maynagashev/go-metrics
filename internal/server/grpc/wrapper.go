package grpc

import (
	"context"

	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"
)

// ServerWrapper представляет обертку для gRPC сервера.
// Эта обертка будет использоваться для запуска gRPC сервера
// в контексте HTTP-сервера.
type ServerWrapper struct {
	server *Server
	log    *zap.Logger
	cfg    *app.Config
}

// NewServer создает новый экземпляр обертки gRPC сервера.
func NewServer(log *zap.Logger, cfg *app.Config, storage storage.Repository) *ServerWrapper {
	return &ServerWrapper{
		server: New(log, cfg, storage),
		log:    log,
		cfg:    cfg,
	}
}

// Start запускает gRPC сервер в отдельной горутине.
func (w *ServerWrapper) Start(ctx context.Context) error {
	// Проверяем, включен ли gRPC сервер
	if !w.cfg.IsGRPCEnabled() {
		w.log.Info("gRPC server is disabled, skipping start")
		return nil
	}

	// Запускаем gRPC сервер
	if err := w.server.Start(); err != nil {
		return err
	}

	// В отдельной горутине ожидаем завершения контекста для graceful shutdown
	go func() {
		<-ctx.Done()
		w.log.Info("context done, stopping gRPC server")
		w.server.Stop()
	}()

	return nil
}

// Stop останавливает gRPC сервер.
func (w *ServerWrapper) Stop() {
	if w.server != nil {
		w.server.Stop()
	}
}

// GetLogger возвращает логгер, используемый оберткой.
// Метод используется в основном для тестирования.
func (w *ServerWrapper) GetLogger() *zap.Logger {
	return w.log
}

// GetConfig возвращает конфигурацию, используемую оберткой.
// Метод используется в основном для тестирования.
func (w *ServerWrapper) GetConfig() *app.Config {
	return w.cfg
}
