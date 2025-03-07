package main

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustParseFlags(t *testing.T) {
	// Сохраняем оригинальные аргументы и переменные окружения
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Сохраняем оригинальный набор флагов
	originalFlagCommandLine := flag.CommandLine
	defer func() { flag.CommandLine = originalFlagCommandLine }()

	tests := []struct {
		name      string
		args      []string
		env       map[string]string
		expected  Flags
		wantPanic bool
	}{
		{
			name: "default values",
			args: []string{"app"},
			expected: Flags{
				Server: struct {
					Addr           string
					ReportInterval float64
					PollInterval   float64
				}{
					Addr:           "localhost:8080",
					ReportInterval: 10.0,
					PollInterval:   2.0,
				},
				RateLimit:   3,
				EnablePprof: false,
				PprofPort:   "6060",
			},
		},
		{
			name: "command line arguments",
			args: []string{
				"app",
				"-a", "localhost:9090",
				"-r", "5.0",
				"-p", "1.0",
				"-l", "5",
				"-pprof",
				"-pprof-port", "6061",
			},
			expected: Flags{
				Server: struct {
					Addr           string
					ReportInterval float64
					PollInterval   float64
				}{
					Addr:           "localhost:9090",
					ReportInterval: 5.0,
					PollInterval:   1.0,
				},
				RateLimit:   5,
				EnablePprof: true,
				PprofPort:   "6061",
			},
		},
		{
			name: "environment variables override",
			args: []string{"app"},
			env: map[string]string{
				"ADDRESS":         "localhost:7070",
				"REPORT_INTERVAL": "15.0",
				"POLL_INTERVAL":   "3.0",
				"KEY":             "test-key",
				"RATE_LIMIT":      "10",
			},
			expected: Flags{
				Server: struct {
					Addr           string
					ReportInterval float64
					PollInterval   float64
				}{
					Addr:           "localhost:7070",
					ReportInterval: 15.0,
					PollInterval:   3.0,
				},
				PrivateKey:  "test-key",
				RateLimit:   10,
				EnablePprof: false,
				PprofPort:   "6060",
			},
		},
		{
			name: "invalid report interval env",
			args: []string{"app"},
			env: map[string]string{
				"REPORT_INTERVAL": "invalid",
			},
			wantPanic: true,
		},
		{
			name: "invalid poll interval env",
			args: []string{"app"},
			env: map[string]string{
				"POLL_INTERVAL": "invalid",
			},
			wantPanic: true,
		},
		{
			name: "invalid rate limit env",
			args: []string{"app"},
			env: map[string]string{
				"RATE_LIMIT": "invalid",
			},
			wantPanic: true,
		},
		{
			name:      "rate limit less than 1",
			args:      []string{"app", "-l", "0"},
			wantPanic: true,
		},
		{
			name: "minimum intervals",
			args: []string{"app", "-r", "0.0000001", "-p", "0.0000001"},
			expected: Flags{
				Server: struct {
					Addr           string
					ReportInterval float64
					PollInterval   float64
				}{
					Addr:           "localhost:8080",
					ReportInterval: minInterval,
					PollInterval:   minInterval,
				},
				RateLimit:   3,
				EnablePprof: false,
				PprofPort:   "6060",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сбрасываем флаги перед каждым тестом
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Устанавливаем тестовые аргументы
			os.Args = tt.args

			// Устанавливаем тестовые переменные окружения
			for k, v := range tt.env {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("failed to set environment variable %s: %v", k, err)
				}
			}
			defer func() {
				// Очищаем переменные окружения после теста
				for k := range tt.env {
					if err := os.Unsetenv(k); err != nil {
						t.Errorf("failed to unset environment variable %s: %v", k, err)
					}
				}
			}()

			if tt.wantPanic {
				assert.Panics(t, func() { mustParseFlags() })
				return
			}

			flags := mustParseFlags()
			assert.Equal(t, tt.expected, flags)
		})
	}
}
