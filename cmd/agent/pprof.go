package main

import (
	"fmt"
	"log/slog"
	"net/http"

	//nolint:gosec // G108: pprof is used intentionally for debugging and profiling
	_ "net/http/pprof"

	"github.com/maynagashev/go-metrics/internal/config"
)

// initPprof запускает pprof сервер если включено профилирование.
func initPprof(flags Flags) {
	if flags.EnablePprof {
		pprofAddr := fmt.Sprintf("localhost:%s", flags.PprofPort)
		startPProf(pprofAddr)
	}
}

func startPProf(pprofAddr string) {
	if pprofAddr != "" {
		go func() {
			srv := &http.Server{
				Addr:              pprofAddr,
				Handler:           nil, // использует DefaultServeMux
				ReadTimeout:       config.DefaultReadTimeout,
				WriteTimeout:      config.DefaultWriteTimeout,
				ReadHeaderTimeout: config.DefaultHeaderTimeout,
				IdleTimeout:       config.DefaultIdleTimeout,
			}
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("Failed to start pprof server", "error", err)
			}
		}()
	}
}
