package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInitPprof(t *testing.T) {
	tests := []struct {
		name       string
		flags      Flags
		wantServer bool
	}{
		{
			name: "profiling disabled",
			flags: Flags{
				EnablePprof: false,
				PprofPort:   "9999",
			},
			wantServer: false,
		},
		{
			name: "profiling enabled",
			flags: Flags{
				EnablePprof: true,
				PprofPort:   "9998",
			},
			wantServer: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initPprof(tt.flags)

			// Даем время на запуск сервера если он должен быть запущен
			time.Sleep(100 * time.Millisecond)

			// Проверяем доступность сервера pprof
			_, err := http.Get("http://localhost:" + tt.flags.PprofPort + "/debug/pprof/")

			if tt.wantServer {
				assert.NoError(t, err, "pprof server should be running")
			} else {
				assert.Error(t, err, "pprof server should not be running")
			}
		})
	}
}

func TestStartPProf(t *testing.T) {
	tests := []struct {
		name      string
		pprofAddr string
		wantErr   bool
	}{
		{
			name:      "empty address",
			pprofAddr: "",
			wantErr:   true,
		},
		{
			name:      "valid address",
			pprofAddr: "localhost:9997",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startPProf(tt.pprofAddr)

			// Даем время на запуск сервера
			time.Sleep(100 * time.Millisecond)

			// Проверяем доступность сервера pprof
			_, err := http.Get("http://localhost:9997/debug/pprof/")

			if tt.wantErr {
				assert.Error(t, err, "should get error for empty address")
			} else {
				assert.NoError(t, err, "pprof server should be running")
			}
		})
	}
}

func TestStartPProf_ServerError(t *testing.T) {
	// Сначала запускаем сервер на порту
	addr := "localhost:9996"
	srv := &http.Server{
		Addr: addr,
	}
	go func() {
		_ = srv.ListenAndServe()
	}()
	defer srv.Close()

	// Даем время на запуск первого сервера
	time.Sleep(100 * time.Millisecond)

	// Пытаемся запустить второй сервер на том же порту
	startPProf(addr)

	// Даем время на попытку запуска второго сервера
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что ошибка была залогирована
	// К сожалению, мы не можем напрямую проверить логи,
	// но можем убедиться, что код не паникует при ошибке запуска сервера
}
