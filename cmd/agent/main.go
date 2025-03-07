// Агент (HTTP-клиент) для сбора рантайм-метрик и их последующей отправки на сервер по протоколу HTTP
package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/maynagashev/go-metrics/internal/agent"
)

// Глобальные переменные для информации о сборке.
//
//nolint:gochecknoglobals // Эти переменные необходимы для информации о версии и задаются при сборке
var (
	BuildVersion = "N/A"
	BuildDate    = "N/A"
	BuildCommit  = "N/A"
)

// printVersion выводит информацию о версии сборки.
//
//nolint:forbidigo // Используем fmt.Println для вывода в stdout согласно требованиям задания
func printVersion() {
	fmt.Println("Build version:", BuildVersion)
	fmt.Println("Build date:", BuildDate)
	fmt.Println("Build commit:", BuildCommit)
}

func main() {
	initLogger()
	printVersion()

	flags := mustParseFlags()
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
