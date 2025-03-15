// Агент (HTTP-клиент) для сбора рантайм-метрик и их последующей отправки на сервер по протоколу HTTP
package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maynagashev/go-metrics/internal/agent"
	"github.com/maynagashev/go-metrics/pkg/crypto"
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

	// Загружаем публичный ключ для шифрования, если он указан
	var publicKey *rsa.PublicKey
	if flags.CryptoKey != "" {
		var err error
		publicKey, err = crypto.LoadPublicKey(flags.CryptoKey)
		if err != nil {
			slog.Error("failed to load public key", "error", err, "path", flags.CryptoKey)
			os.Exit(1)
		}
		slog.Info("loaded public key for encryption", "path", flags.CryptoKey)
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
	a := agent.New(
		serverURL,
		pollInterval,
		reportInterval,
		flags.PrivateKey,
		flags.RateLimit,
		publicKey,
	)

	// Запускаем горутину для обработки сигналов
	go func() {
		sig := <-sigCh
		slog.Info("received signal", "signal", sig)
		cancel() // Отменяем контекст, что приведет к graceful shutdown
	}()

	// Запускаем агента с контекстом
	a.Run(ctx)
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
