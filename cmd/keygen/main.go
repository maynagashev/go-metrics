// Package main предоставляет инструмент командной строки для генерации пар ключей RSA.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/maynagashev/go-metrics/pkg/crypto"
)

// Build information set during compilation
// These are allowed to be global variables as they are set at build time
//
//nolint:gochecknoglobals // These variables are set at build time by the compiler
var (
	// BuildVersion contains the version of the build.
	BuildVersion string
	// BuildDate contains the date of the build.
	BuildDate string
	// BuildCommit contains the commit hash of the build.
	BuildCommit string
	// ExitFunc is the function used to exit the program, can be replaced in tests.
	ExitFunc = os.Exit
)

// DefaultKeySize - размер ключа RSA по умолчанию.
const DefaultKeySize = 2048

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
	genErr := generateKeys(privateKeyPath, publicKeyPath, keySize)
	if genErr != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при генерации пары ключей: %v\n", genErr)
		ExitFunc(1)
		return
	}

	slog.Info("Пара ключей успешно сгенерирована",
		"private_key", privateKeyPath,
		"public_key", publicKeyPath)

	slog.Warn("ВАЖНО: Храните ваш закрытый ключ в безопасном месте и не передавайте его никому!")
}

// printVersion выводит информацию о версии сборки.
func printVersion() {
	slog.Info("Build information",
		"version", getStringOrDefault(BuildVersion, "N/A"),
		"date", getStringOrDefault(BuildDate, "N/A"),
		"commit", getStringOrDefault(BuildCommit, "N/A"))
}

// getStringOrDefault возвращает строку или значение по умолчанию, если строка пуста.
func getStringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// parseFlags разбирает флаги командной строки.
func parseFlags() (string, string, int, error) {
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

// generateKeys генерирует пару ключей RSA.
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
