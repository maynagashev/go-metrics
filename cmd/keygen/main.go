// Package main предоставляет инструмент командной строки для генерации пар ключей RSA.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/maynagashev/go-metrics/pkg/crypto"
)

// BuildVersion содержит версию сборки
var BuildVersion string

// BuildDate содержит дату сборки
var BuildDate string

// BuildCommit содержит коммит сборки
var BuildCommit string

// DefaultKeySize - размер ключа RSA по умолчанию.
const DefaultKeySize = 2048

// ExitFunc - функция для выхода из программы, может быть заменена в тестах
var ExitFunc = os.Exit

func main() {
	// Выводим информацию о сборке
	printVersion()

	// Инициализируем логгер
	initLogger()

	// Парсим аргументы командной строки
	privateKeyPath, publicKeyPath, keySize, err := parseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при разборе аргументов: %v\n", err)
		ExitFunc(1)
		return
	}

	// Генерируем ключи
	if err := generateKeys(privateKeyPath, publicKeyPath, keySize); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при генерации пары ключей: %v\n", err)
		ExitFunc(1)
		return
	}

	slog.Info("Пара ключей успешно сгенерирована",
		"private_key", privateKeyPath,
		"public_key", publicKeyPath)

	slog.Warn("ВАЖНО: Храните ваш закрытый ключ в безопасном месте и не передавайте его никому!")
}

// printVersion выводит информацию о версии сборки
func printVersion() {
	fmt.Printf("Build version: %s\n", getStringOrDefault(BuildVersion, "N/A"))
	fmt.Printf("Build date: %s\n", getStringOrDefault(BuildDate, "N/A"))
	fmt.Printf("Build commit: %s\n", getStringOrDefault(BuildCommit, "N/A"))
}

// getStringOrDefault возвращает строку или значение по умолчанию, если строка пуста
func getStringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// parseFlags разбирает флаги командной строки
func parseFlags() (privateKeyPath, publicKeyPath string, keySize int, err error) {
	// Определяем флаги командной строки
	privateKeyPathPtr := flag.String("private", "private.pem", "путь для сохранения закрытого ключа")
	publicKeyPathPtr := flag.String("public", "public.pem", "путь для сохранения открытого ключа")
	keySizePtr := flag.Int("bits", DefaultKeySize, "размер ключа RSA в битах (1024, 2048, 4096)")

	// Разбираем флаги
	flag.Parse()

	// Проверяем размер ключа
	validKeySizes := map[int]bool{1024: true, 2048: true, 4096: true}
	if !validKeySizes[*keySizePtr] {
		return "", "", 0, fmt.Errorf("неверный размер ключа: %d. Допустимые размеры: 1024, 2048, 4096", *keySizePtr)
	}

	return *privateKeyPathPtr, *publicKeyPathPtr, *keySizePtr, nil
}

// generateKeys генерирует пару ключей RSA
func generateKeys(privateKeyPath, publicKeyPath string, keySize int) error {
	slog.Info("Генерация RSA ключей", "bits", keySize)
	return crypto.GenerateKeyPair(privateKeyPath, publicKeyPath, keySize)
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
