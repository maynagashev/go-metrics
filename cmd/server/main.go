package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/maynagashev/go-metrics/internal/storage/memory"

	"github.com/maynagashev/go-metrics/internal/server/router"
)

const (
	DefaultReadTimeout  = 5 * time.Second
	DefaultWriteTimeout = 10 * time.Second
	DefaultIdleTimeout  = 120 * time.Second
)

func main() {
	parseFlags()
	fmt.Printf("Starting server on %s\n", flagRunAddr)

	server := &http.Server{
		Addr:    flagRunAddr,
		Handler: router.New(memory.New()),
		// Настройка таймаутов для сервера по рекомендациям линтера gosec
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
		IdleTimeout:  DefaultIdleTimeout,
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
