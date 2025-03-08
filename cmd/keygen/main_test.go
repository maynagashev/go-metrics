package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"log/slog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateKeyPair проверяет генерацию пары ключей.
func TestGenerateKeyPair(t *testing.T) {
	// Создаем временную директорию для тестовых файлов
	tempDir, err := os.MkdirTemp("", "keygen-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Пути к файлам ключей
	privateKeyPath := filepath.Join(tempDir, "private.pem")
	publicKeyPath := filepath.Join(tempDir, "public.pem")

	// Генерируем ключи
	err = generateKeys(privateKeyPath, publicKeyPath, DefaultKeySize)
	require.NoError(t, err)

	// Проверяем, что файлы были созданы
	assert.FileExists(t, privateKeyPath)
	assert.FileExists(t, publicKeyPath)

	// Проверяем содержимое файлов
	privateKeyData, err := os.ReadFile(privateKeyPath)
	require.NoError(t, err)
	assert.Contains(t, string(privateKeyData), "PRIVATE KEY")

	publicKeyData, err := os.ReadFile(publicKeyPath)
	require.NoError(t, err)
	// Проверяем, что файл содержит данные ключа, без проверки конкретного формата
	assert.NotEmpty(t, publicKeyData)
}

// TestParseFlags проверяет разбор флагов командной строки.
func TestParseFlags(t *testing.T) {
	// Сохраняем оригинальные аргументы
	oldArgs := os.Args
	oldFlagCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlagCommandLine
	}()

	// Тест 1: Проверка значений по умолчанию
	os.Args = []string{"keygen"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	privateKeyPath, publicKeyPath, keySize, err := parseFlags()
	require.NoError(t, err)
	assert.Equal(t, "private.pem", privateKeyPath)
	assert.Equal(t, "public.pem", publicKeyPath)
	assert.Equal(t, DefaultKeySize, keySize)

	// Тест 2: Проверка пользовательских значений
	os.Args = []string{"keygen", "-private", "custom_private.pem", "-public", "custom_public.pem", "-bits", "4096"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	privateKeyPath, publicKeyPath, keySize, err = parseFlags()
	require.NoError(t, err)
	assert.Equal(t, "custom_private.pem", privateKeyPath)
	assert.Equal(t, "custom_public.pem", publicKeyPath)
	assert.Equal(t, 4096, keySize)

	// Тест 3: Проверка неверного размера ключа
	os.Args = []string{"keygen", "-bits", "3000"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	_, _, _, err = parseFlags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "неверный размер ключа")
}

// TestGetStringOrDefault проверяет функцию getStringOrDefault.
func TestGetStringOrDefault(t *testing.T) {
	assert.Equal(t, "default", getStringOrDefault("", "default"))
	assert.Equal(t, "value", getStringOrDefault("value", "default"))
}

// TestPrintVersion проверяет вывод версии.
func TestPrintVersion(t *testing.T) {
	// Since we're now using slog.Info instead of fmt.Printf, we need to capture the log output
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Save the original logger and restore it after the test
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	// Create a new logger that writes to our buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Устанавливаем тестовые значения
	oldBuildVersion := BuildVersion
	oldBuildDate := BuildDate
	oldBuildCommit := BuildCommit
	defer func() {
		BuildVersion = oldBuildVersion
		BuildDate = oldBuildDate
		BuildCommit = oldBuildCommit
	}()

	// Тест 1: Пустые значения
	BuildVersion = ""
	BuildDate = ""
	BuildCommit = ""
	buf.Reset()
	printVersion()
	output := buf.String()
	assert.Contains(t, output, "version=N/A")
	assert.Contains(t, output, "date=N/A")
	assert.Contains(t, output, "commit=N/A")

	// Тест 2: Заполненные значения
	BuildVersion = "v1.0.0"
	BuildDate = "2023-01-01"
	BuildCommit = "abc123"
	buf.Reset()
	printVersion()
	output = buf.String()
	assert.Contains(t, output, "version=v1.0.0")
	assert.Contains(t, output, "date=2023-01-01")
	assert.Contains(t, output, "commit=abc123")
}

// TestInitLogger проверяет инициализацию логгера.
func TestInitLogger(_ *testing.T) {
	// Вызываем функцию initLogger
	initLogger()
	// Здесь мы просто проверяем, что функция не паникует
}

// TestMain проверяет функцию main.
func TestMain(t *testing.T) {
	// Сохраняем оригинальные аргументы и функцию выхода
	oldArgs := os.Args
	oldExitFunc := ExitFunc
	defer func() {
		os.Args = oldArgs
		ExitFunc = oldExitFunc
	}()

	// Создаем временную директорию для тестовых файлов
	tempDir, err := os.MkdirTemp("", "keygen-test-main")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Пути к файлам ключей
	privateKeyPath := filepath.Join(tempDir, "private.pem")
	publicKeyPath := filepath.Join(tempDir, "public.pem")

	// Устанавливаем тестовые аргументы
	os.Args = []string{"keygen", "-private", privateKeyPath, "-public", publicKeyPath}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Заменяем функцию выхода
	exitCalled := false
	exitCode := 0
	ExitFunc = func(code int) {
		exitCalled = true
		exitCode = code
	}

	// Временно перенаправляем stdout и stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout, _ = os.Open(os.DevNull)
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Вызываем main
	main()

	// Проверяем, что функция выхода не была вызвана с ошибкой
	assert.False(t, exitCalled)
	assert.Equal(t, 0, exitCode)

	// Проверяем, что файлы были созданы
	assert.FileExists(t, privateKeyPath)
	assert.FileExists(t, publicKeyPath)
}
