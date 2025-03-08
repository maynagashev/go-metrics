package app

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlags(t *testing.T) {
	// Сохраняем оригинальные аргументы командной строки
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Устанавливаем тестовые аргументы
	os.Args = []string{"cmd", "-a", "localhost:9090", "-i", "10", "-f", "/tmp/test.db", "-r=false"}

	// Сбрасываем флаги перед тестом
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Вызываем функцию
	flags, err := ParseFlags()

	// Проверяем результат
	require.NoError(t, err)
	assert.Equal(t, "localhost:9090", flags.Server.Addr)
	assert.Equal(t, 10, flags.Server.StoreInterval)
	assert.Equal(t, "/tmp/test.db", flags.Server.FileStoragePath)
	assert.False(t, flags.Server.Restore)
}

func TestRegisterCommandLineFlags(t *testing.T) {
	// Сохраняем оригинальный FlagSet
	oldFlagCommandLine := flag.CommandLine
	defer func() { flag.CommandLine = oldFlagCommandLine }()

	// Создаем новый FlagSet для теста
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	// Создаем структуру Flags
	flags := &Flags{}

	// Регистрируем флаги
	registerCommandLineFlags(flags)

	// Парсим тестовые аргументы
	err := flag.CommandLine.Parse([]string{"-a", "localhost:9090", "-i", "10", "-f", "/tmp/test.db", "-r=false"})

	// Проверяем результат
	require.NoError(t, err)
	assert.Equal(t, "localhost:9090", flags.Server.Addr)
	assert.Equal(t, 10, flags.Server.StoreInterval)
	assert.Equal(t, "/tmp/test.db", flags.Server.FileStoragePath)
	assert.False(t, flags.Server.Restore)
}

func TestApplyEnvironmentVariables(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	oldEnv := map[string]string{
		"ADDRESS":           os.Getenv("ADDRESS"),
		"STORE_INTERVAL":    os.Getenv("STORE_INTERVAL"),
		"FILE_STORAGE_PATH": os.Getenv("FILE_STORAGE_PATH"),
		"RESTORE":           os.Getenv("RESTORE"),
		"DATABASE_DSN":      os.Getenv("DATABASE_DSN"),
		"KEY":               os.Getenv("KEY"),
		"CRYPTO_KEY":        os.Getenv("CRYPTO_KEY"),
	}

	// Восстанавливаем оригинальные переменные окружения после теста
	defer func() {
		for k, v := range oldEnv {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	// Устанавливаем тестовые переменные окружения
	os.Setenv("ADDRESS", "localhost:9090")
	os.Setenv("STORE_INTERVAL", "10")
	os.Setenv("FILE_STORAGE_PATH", "/tmp/test.db")
	os.Setenv("RESTORE", "false")
	os.Setenv("DATABASE_DSN", "postgres://user:pass@localhost:5432/db")
	os.Setenv("KEY", "test-key")
	os.Setenv("CRYPTO_KEY", "/path/to/key.pem")

	// Создаем структуру Flags с дефолтными значениями
	flags := &Flags{}
	flags.Server.Addr = "default:8080"
	flags.Server.StoreInterval = 300
	flags.Server.FileStoragePath = "/tmp/default.db"
	flags.Server.Restore = true

	// Применяем переменные окружения
	err := applyEnvironmentVariables(flags)

	// Проверяем результат
	require.NoError(t, err)
	assert.Equal(t, "localhost:9090", flags.Server.Addr)
	assert.Equal(t, 10, flags.Server.StoreInterval)
	assert.Equal(t, "/tmp/test.db", flags.Server.FileStoragePath)
	assert.False(t, flags.Server.Restore)
	assert.Equal(t, "postgres://user:pass@localhost:5432/db", flags.Database.DSN)
	assert.Equal(t, "test-key", flags.PrivateKey)
	assert.Equal(t, "/path/to/key.pem", flags.CryptoKey)
}

func TestLoadAndApplyJSONConfig(t *testing.T) {
	// Создаем временный файл конфигурации
	tmpFile, err := os.CreateTemp("", "config_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Записываем тестовую конфигурацию в файл
	configJSON := `{
		"address": "localhost:9090",
		"store_interval": "10s",
		"store_file": "/tmp/test.db",
		"restore": false,
		"database_dsn": "postgres://user:pass@localhost:5432/db",
		"key": "test-key",
		"crypto_key": "/path/to/key.pem"
	}`
	_, err = tmpFile.WriteString(configJSON)
	require.NoError(t, err)
	err = tmpFile.Close()
	require.NoError(t, err)

	// Создаем структуру Flags с дефолтными значениями
	flags := &Flags{}
	flags.Server.Addr = defaultServerAddr
	flags.Server.StoreInterval = defaultStoreInterval
	flags.Server.FileStoragePath = defaultFileStoragePath
	flags.Server.Restore = true
	flags.ConfigFile = tmpFile.Name()

	// Загружаем конфигурацию из JSON
	jsonConfig, err := LoadJSONConfig(flags.ConfigFile)
	require.NoError(t, err)

	// Применяем конфигурацию
	err = ApplyJSONConfig(flags, jsonConfig)
	require.NoError(t, err)

	// Проверяем результат
	assert.Equal(t, "localhost:9090", flags.Server.Addr)
	assert.Equal(t, 10, flags.Server.StoreInterval)
	assert.Equal(t, "/tmp/test.db", flags.Server.FileStoragePath)
	assert.False(t, flags.Server.Restore)
	assert.Equal(t, "postgres://user:pass@localhost:5432/db", flags.Database.DSN)
	assert.Equal(t, "/path/to/key.pem", flags.CryptoKey)
}
