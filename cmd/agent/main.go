// Агент (HTTP-клиент) для сбора рантайм-метрик и их последующей отправки на сервер по протоколу HTTP
package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/maynagashev/go-metrics/internal/agent"
)

func main() {
	flags := mustParseFlags()
	initLogger()
	slog.Debug("parsed flags and env variables", "flags", flags)

	initPprof(flags)

	serverURL := "http://" + flags.Server.Addr
	pollInterval := time.Duration(flags.Server.PollInterval * float64(time.Second))
	reportInterval := time.Duration(flags.Server.ReportInterval * float64(time.Second))

	a := agent.New(serverURL, pollInterval, reportInterval, flags.PrivateKey, flags.RateLimit)
	a.Run()
}

func initLogger() {
	// Создаем переменную для уровня логирования и устанавливаем ее в Debug
	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelDebug)

	// Создаем новый обработчик с настроенным уровнем логирования
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Устанавливаем созданный логгер как логгер по умолчанию
	slog.SetDefault(logger)
}
