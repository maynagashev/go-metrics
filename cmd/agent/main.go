// Агент (HTTP-клиент) для сбора рантайм-метрик и их последующей отправки на сервер по протоколу HTTP
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
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

	// Создаем функцию, которая будет выполнять всю логику
	// и возвращать код ошибки
	if err := run(); err != nil {
		os.Exit(1)
	}
}

// run выполняет основную логику программы и возвращает ошибку.
func run() error {
	flags := mustParseFlags()
	slog.Debug("parsed flags and env variables", "flags", flags)

	initPprof(flags)

	// Путь к ключу для шифрования передаем напрямую в агент
	cryptoKeyPath := flags.CryptoKey
	if cryptoKeyPath != "" {
		slog.Info("using crypto key for encryption", "path", cryptoKeyPath)
	}

	serverURL := "http://" + flags.Server.Addr
	pollInterval := time.Duration(flags.Server.PollInterval * float64(time.Second))
	reportInterval := time.Duration(flags.Server.ReportInterval * float64(time.Second))

	// Создаем контекст с отменой для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Канал для получения сигналов от ОС
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Запускаем агента
	a, err := agent.New(
		serverURL,
		pollInterval,
		reportInterval,
		flags.PrivateKey,
		flags.RateLimit,
		flags.RealIP,
		flags.GRPCEnabled,
		flags.GRPCAddress,
		flags.GRPCTimeout,
		flags.GRPCRetry,
		flags.CryptoKey,
	)
	if err != nil {
		slog.Error("Failed to create agent", "error", err)
		return err
	}

	// Запускаем горутину для обработки сигналов
	go func() {
		sig := <-sigCh
		slog.Info("received signal", "signal", sig)
		cancel() // Отменяем контекст, что приведет к graceful shutdown
	}()

	// Запускаем агента с контекстом
	a.Run(ctx)
	return nil
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
