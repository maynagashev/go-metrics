// Package main предоставляет инструмент командной строки для генерации пар ключей RSA.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/maynagashev/go-metrics/pkg/crypto"
)

// DefaultKeySize - размер ключа RSA по умолчанию.
const DefaultKeySize = 2048

func main() {
	// Инициализируем логгер
	initLogger()

	// Определяем флаги командной строки
	privateKeyPath := flag.String("private", "private.pem", "путь для сохранения закрытого ключа")
	publicKeyPath := flag.String("public", "public.pem", "путь для сохранения открытого ключа")
	keySize := flag.Int("bits", DefaultKeySize, "размер ключа RSA в битах (1024, 2048, 4096)")

	// Разбираем флаги
	flag.Parse()

	// Проверяем размер ключа
	validKeySizes := map[int]bool{1024: true, 2048: true, 4096: true}
	if !validKeySizes[*keySize] {
		fmt.Fprintf(os.Stderr, "Неверный размер ключа: %d. Допустимые размеры: 1024, 2048, 4096\n", *keySize)
		os.Exit(1)
	}

	slog.Info("Генерация RSA ключей", "bits", *keySize)

	// Генерируем пару ключей
	err := crypto.GenerateKeyPair(*privateKeyPath, *publicKeyPath, *keySize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при генерации пары ключей: %v\n", err)
		os.Exit(1)
	}

	slog.Info("Пара ключей успешно сгенерирована",
		"private_key", *privateKeyPath,
		"public_key", *publicKeyPath)

	slog.Warn("ВАЖНО: Храните ваш закрытый ключ в безопасном месте и не передавайте его никому!")
}

// initLogger инициализирует логгер.
func initLogger() {
	// Создаем переменную для уровня логирования и устанавливаем ее в Info
	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelInfo)

	// Создаем новый обработчик с настроенным уровнем логирования
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Устанавливаем созданный логгер как логгер по умолчанию
	slog.SetDefault(logger)
}
