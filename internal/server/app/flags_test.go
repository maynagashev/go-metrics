//nolint:testpackage // тестирует внутреннее API app напрямую
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

	// Создаем флаги
	flags := &Flags{}

	// Вызываем функцию напрямую
	registerCommandLineFlags(flags)

	// Проверяем, что флаги были успешно зарегистрированы
	err := flag.CommandLine.Parse(
		[]string{"-a", "localhost:9090", "-i", "10", "-f", "/tmp/test.db", "-r=false"},
	)
	require.NoError(t, err)

	// Проверяем, что значения флагов установлены
	addrFlag := flag.Lookup("a")
	require.NotNil(t, addrFlag)
	assert.Equal(t, "localhost:9090", addrFlag.Value.String())
}

func TestApplyEnvironmentVariables(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	oldEnv := map[string]string{
		"ADDRESS":             os.Getenv("ADDRESS"),
		"STORE_INTERVAL":      os.Getenv("STORE_INTERVAL"),
		"FILE_STORAGE_PATH":   os.Getenv("FILE_STORAGE_PATH"),
		"RESTORE":             os.Getenv("RESTORE"),
		"DATABASE_DSN":        os.Getenv("DATABASE_DSN"),
		"DATABASE_MIGRATIONS": os.Getenv("DATABASE_MIGRATIONS"),
		"KEY":                 os.Getenv("KEY"),
		"CRYPTO_KEY":          os.Getenv("CRYPTO_KEY"),
		"TRUSTED_SUBNET":      os.Getenv("TRUSTED_SUBNET"),
		"ENABLE_PPROF":        os.Getenv("ENABLE_PPROF"),
		"CONFIG":              os.Getenv("CONFIG"),
		"GRPC_ADDRESS":        os.Getenv("GRPC_ADDRESS"),
		"GRPC_ENABLED":        os.Getenv("GRPC_ENABLED"),
		"GRPC_MAX_CONN":       os.Getenv("GRPC_MAX_CONN"),
		"GRPC_TIMEOUT":        os.Getenv("GRPC_TIMEOUT"),
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
	os.Setenv("DATABASE_MIGRATIONS", "/path/to/migrations")
	os.Setenv("KEY", "test-key")
	os.Setenv("CRYPTO_KEY", "/path/to/key.pem")
	os.Setenv("TRUSTED_SUBNET", "192.168.0.0/24")
	os.Setenv("ENABLE_PPROF", "true")
	os.Setenv("CONFIG", "/path/to/config.json")
	os.Setenv("GRPC_ADDRESS", "localhost:9091")
	os.Setenv("GRPC_ENABLED", "true")
	os.Setenv("GRPC_MAX_CONN", "100")
	os.Setenv("GRPC_TIMEOUT", "5")

	// Создаем структуру Flags с дефолтными значениями
	flags := &Flags{}
	flags.Server.Addr = "default:8080"
	flags.Server.StoreInterval = 300
	flags.Server.FileStoragePath = "/tmp/default.db"
	flags.Server.Restore = true

	// Вызываем функцию применения переменных окружения напрямую
	err := applyEnvironmentVariables(flags)
	require.NoError(t, err)

	// Проверяем серверные настройки
	assert.Equal(t, "localhost:9090", flags.Server.Addr)
	assert.Equal(t, 10, flags.Server.StoreInterval)
	assert.Equal(t, "/tmp/test.db", flags.Server.FileStoragePath)
	assert.False(t, flags.Server.Restore)
	assert.Equal(t, "192.168.0.0/24", flags.Server.TrustedSubnet)
	assert.True(t, flags.Server.EnablePprof)

	// Проверяем настройки базы данных
	assert.Equal(t, "postgres://user:pass@localhost:5432/db", flags.Database.DSN)
	assert.Equal(t, "/path/to/migrations", flags.Database.MigrationsPath)

	// Проверяем настройки безопасности
	assert.Equal(t, "test-key", flags.PrivateKey)
	assert.Equal(t, "/path/to/key.pem", flags.CryptoKey)

	// Проверяем настройки конфигурации
	assert.Equal(t, "/path/to/config.json", flags.ConfigFile)

	// Проверяем настройки gRPC
	assert.Equal(t, "localhost:9091", flags.GRPC.Addr)
	assert.True(t, flags.GRPC.Enabled)
	assert.Equal(t, 100, flags.GRPC.MaxConn)
	assert.Equal(t, 5, flags.GRPC.Timeout)
}

func TestApplyServerEnvVars(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	oldEnv := map[string]string{
		"ADDRESS":           os.Getenv("ADDRESS"),
		"STORE_INTERVAL":    os.Getenv("STORE_INTERVAL"),
		"FILE_STORAGE_PATH": os.Getenv("FILE_STORAGE_PATH"),
		"RESTORE":           os.Getenv("RESTORE"),
		"TRUSTED_SUBNET":    os.Getenv("TRUSTED_SUBNET"),
		"ENABLE_PPROF":      os.Getenv("ENABLE_PPROF"),
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
	os.Setenv("TRUSTED_SUBNET", "192.168.0.0/24")
	os.Setenv("ENABLE_PPROF", "true")

	// Создаем структуру Flags с дефолтными значениями
	flags := &Flags{}
	flags.Server.Addr = "default:8080"
	flags.Server.StoreInterval = 300
	flags.Server.FileStoragePath = "/tmp/default.db"
	flags.Server.Restore = true
	flags.Server.EnablePprof = false

	// Вызываем функцию
	err := applyServerEnvVars(flags)
	require.NoError(t, err)

	// Проверяем результат
	assert.Equal(t, "localhost:9090", flags.Server.Addr)
	assert.Equal(t, 10, flags.Server.StoreInterval)
	assert.Equal(t, "/tmp/test.db", flags.Server.FileStoragePath)
	assert.False(t, flags.Server.Restore)
	assert.Equal(t, "192.168.0.0/24", flags.Server.TrustedSubnet)
	assert.True(t, flags.Server.EnablePprof)

	// Тест с некорректными значениями для числовых полей
	os.Setenv("STORE_INTERVAL", "not-a-number")
	os.Setenv("RESTORE", "not-a-bool")
	os.Setenv("ENABLE_PPROF", "not-a-bool")

	flags = &Flags{}
	flags.Server.StoreInterval = 300
	flags.Server.Restore = true
	flags.Server.EnablePprof = false

	// Вызываем функцию
	err = applyServerEnvVars(flags)
	require.Error(t, err)

	// Проверяем, что значения остались без изменений для невалидных значений
	assert.Equal(t, 300, flags.Server.StoreInterval)
	assert.True(t, flags.Server.Restore)
	assert.False(t, flags.Server.EnablePprof)
}

func TestApplyDatabaseEnvVars(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	oldEnv := map[string]string{
		"DATABASE_DSN":        os.Getenv("DATABASE_DSN"),
		"DATABASE_MIGRATIONS": os.Getenv("DATABASE_MIGRATIONS"),
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
	os.Setenv("DATABASE_DSN", "postgres://user:pass@localhost:5432/db")
	os.Setenv("DATABASE_MIGRATIONS", "/path/to/migrations")

	// Создаем структуру Flags
	flags := &Flags{}

	// Вызываем функцию
	applyDatabaseEnvVars(flags)

	// Проверяем результат
	assert.Equal(t, "postgres://user:pass@localhost:5432/db", flags.Database.DSN)
	assert.Equal(t, "/path/to/migrations", flags.Database.MigrationsPath)
}

func TestApplySecurityEnvVars(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	oldEnv := map[string]string{
		"KEY":        os.Getenv("KEY"),
		"CRYPTO_KEY": os.Getenv("CRYPTO_KEY"),
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
	os.Setenv("KEY", "test-key")
	os.Setenv("CRYPTO_KEY", "/path/to/key.pem")

	// Создаем структуру Flags
	flags := &Flags{}

	// Вызываем функцию
	applySecurityEnvVars(flags)

	// Проверяем результат
	assert.Equal(t, "test-key", flags.PrivateKey)
	assert.Equal(t, "/path/to/key.pem", flags.CryptoKey)
}

func TestApplyConfigEnvVars(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	oldEnv := map[string]string{
		"CONFIG": os.Getenv("CONFIG"),
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
	os.Setenv("CONFIG", "/path/to/config.json")

	// Создаем структуру Flags
	flags := &Flags{}

	// Вызываем функцию
	applyConfigEnvVars(flags)

	// Проверяем результат
	assert.Equal(t, "/path/to/config.json", flags.ConfigFile)
}

func TestApplyGRPCEnvVars(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	oldEnv := map[string]string{
		"GRPC_ADDRESS":  os.Getenv("GRPC_ADDRESS"),
		"GRPC_ENABLED":  os.Getenv("GRPC_ENABLED"),
		"GRPC_MAX_CONN": os.Getenv("GRPC_MAX_CONN"),
		"GRPC_TIMEOUT":  os.Getenv("GRPC_TIMEOUT"),
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
	os.Setenv("GRPC_ADDRESS", "localhost:9091")
	os.Setenv("GRPC_ENABLED", "true")
	os.Setenv("GRPC_MAX_CONN", "100")
	os.Setenv("GRPC_TIMEOUT", "5")

	// Создаем структуру Flags
	flags := &Flags{}

	// Вызываем функцию
	err := applyGRPCEnvVars(flags)
	require.NoError(t, err)

	// Проверяем результат
	assert.Equal(t, "localhost:9091", flags.GRPC.Addr)
	assert.True(t, flags.GRPC.Enabled)
	assert.Equal(t, 100, flags.GRPC.MaxConn)
	assert.Equal(t, 5, flags.GRPC.Timeout)

	// Тест с некорректными значениями
	os.Setenv("GRPC_ENABLED", "not-a-bool")
	os.Setenv("GRPC_MAX_CONN", "not-a-number")
	os.Setenv("GRPC_TIMEOUT", "not-a-number")

	flags = &Flags{}
	flags.GRPC.Enabled = false
	flags.GRPC.MaxConn = 10
	flags.GRPC.Timeout = 1

	// Вызываем функцию
	err = applyGRPCEnvVars(flags)
	require.Error(t, err)

	// Проверяем, что значения остались без изменений для невалидных значений
	assert.False(t, flags.GRPC.Enabled)
	assert.Equal(t, 10, flags.GRPC.MaxConn)
	assert.Equal(t, 1, flags.GRPC.Timeout)
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
		"crypto_key": "/path/to/key.pem",
		"trusted_subnet": "192.168.0.0/24",
		"enable_pprof": true,
		"grpc_address": "localhost:9091",
		"grpc_enabled": true,
		"grpc_max_conn": 100,
		"grpc_timeout": 5
	}`
	_, err = tmpFile.WriteString(configJSON)
	require.NoError(t, err)
	err = tmpFile.Close()
	require.NoError(t, err)

	// Создаем структуру Flags с дефолтными значениями
	flags := &Flags{}
	flags.Server.Addr = DefaultServerAddr()
	flags.Server.StoreInterval = DefaultStoreInterval()
	flags.Server.FileStoragePath = DefaultFileStoragePath()
	flags.Server.Restore = true
	flags.GRPC.Addr = defaultGRPCAddr
	flags.GRPC.MaxConn = defaultGRPCMaxConn
	flags.GRPC.Timeout = defaultGRPCTimeout
	flags.ConfigFile = tmpFile.Name()

	// Вызываем функцию
	err = loadAndApplyJSONConfig(flags)
	require.NoError(t, err)

	// Проверяем серверные настройки
	assert.Equal(t, "localhost:9090", flags.Server.Addr)
	assert.Equal(t, 10, flags.Server.StoreInterval)
	assert.Equal(t, "/tmp/test.db", flags.Server.FileStoragePath)
	assert.False(t, flags.Server.Restore)
	assert.Equal(t, "192.168.0.0/24", flags.Server.TrustedSubnet)
	assert.True(t, flags.Server.EnablePprof)

	// Проверяем настройки базы данных
	assert.Equal(t, "postgres://user:pass@localhost:5432/db", flags.Database.DSN)

	// Проверяем настройки безопасности
	assert.Equal(t, "test-key", flags.PrivateKey)
	assert.Equal(t, "/path/to/key.pem", flags.CryptoKey)

	// Проверяем настройки gRPC
	assert.Equal(t, "localhost:9091", flags.GRPC.Addr)
	assert.True(t, flags.GRPC.Enabled)
	assert.Equal(t, 100, flags.GRPC.MaxConn)
	assert.Equal(t, 5, flags.GRPC.Timeout)

	// Тест с несуществующим файлом
	flags = &Flags{}
	flags.ConfigFile = "non-existent-file.json"
	err = loadAndApplyJSONConfig(flags)
	require.Error(t, err)

	// Тест с некорректным JSON
	invalidJSONFile, err := os.CreateTemp("", "invalid_config_*.json")
	require.NoError(t, err)
	defer os.Remove(invalidJSONFile.Name())

	_, err = invalidJSONFile.WriteString(`{invalid json}`)
	require.NoError(t, err)
	err = invalidJSONFile.Close()
	require.NoError(t, err)

	flags = &Flags{}
	flags.ConfigFile = invalidJSONFile.Name()
	err = loadAndApplyJSONConfig(flags)
	require.Error(t, err)
}
