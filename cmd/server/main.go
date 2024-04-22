package main

import (
	"log/slog"
	"net/http"
	"time"

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
	slog.Info("starting server...", "addr", flags.Server.Addr)

	server := &http.Server{
		Addr:    flags.Server.Addr,
		Handler: router.New(memory.New()),
		// Настройка таймаутов для сервера по рекомендациям линтера gosec
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
		IdleTimeout:  DefaultIdleTimeout,
	}

	err := server.ListenAndServe()
	if err != nil {
		slog.Error("server failed to start", "error", err)
	}
}
