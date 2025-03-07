package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadJSONConfig(t *testing.T) {
	// Создаем временный файл с конфигурацией
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Тест 1: Корректный JSON
	configContent := `{
		"address": "localhost:9090",
		"restore": false,
		"store_interval": "5s",
		"store_file": "/tmp/test-metrics.json",
		"database_dsn": "postgres://user:pass@localhost:5432/testdb",
		"crypto_key": "/tmp/test-key.pem",
		"enable_pprof": true
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	config, err := LoadJSONConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "localhost:9090", config.Address)
	assert.False(t, config.Restore)
	assert.Equal(t, "5s", config.StoreInterval)
	assert.Equal(t, "/tmp/test-metrics.json", config.StoreFile)
	assert.Equal(t, "postgres://user:pass@localhost:5432/testdb", config.DatabaseDSN)
	assert.Equal(t, "/tmp/test-key.pem", config.CryptoKey)
	assert.True(t, config.EnablePprof)

	// Тест 2: Пустой путь к файлу
	config, err = LoadJSONConfig("")
	require.NoError(t, err)
	assert.Nil(t, config)

	// Тест 3: Некорректный JSON
	invalidConfigPath := filepath.Join(tempDir, "invalid-config.json")
	err = os.WriteFile(invalidConfigPath, []byte(`{invalid json`), 0o600)
	require.NoError(t, err)

	config, err = LoadJSONConfig(invalidConfigPath)
	require.Error(t, err)
	assert.Nil(t, config)

	// Тест 4: Несуществующий файл
	config, err = LoadJSONConfig("/non/existent/path.json")
	require.Error(t, err)
	assert.Nil(t, config)
}

func TestApplyJSONConfig(t *testing.T) {
	// Тест 1: Применение конфигурации к флагам по умолчанию
	flags := &Flags{}
	flags.Server.Addr = "localhost:8080"
	flags.Server.StoreInterval = defaultStoreInterval
	flags.Server.FileStoragePath = "/tmp/metrics-db.json"
	flags.Server.Restore = true
	flags.Database.DSN = ""
	flags.CryptoKey = ""
	flags.Server.EnablePprof = false

	jsonConfig := &JSONConfig{
		Address:       "localhost:9090",
		Restore:       false,
		StoreInterval: "5s",
		StoreFile:     "/tmp/test-metrics.json",
		DatabaseDSN:   "postgres://user:pass@localhost:5432/testdb",
		CryptoKey:     "/tmp/test-key.pem",
		EnablePprof:   true,
	}

	err := ApplyJSONConfig(flags, jsonConfig)
	require.NoError(t, err)

	assert.Equal(t, "localhost:9090", flags.Server.Addr)
	assert.Equal(t, 5, flags.Server.StoreInterval) // 5s -> 5 секунд
	assert.Equal(t, "/tmp/test-metrics.json", flags.Server.FileStoragePath)
	assert.False(t, flags.Server.Restore)
	assert.Equal(t, "postgres://user:pass@localhost:5432/testdb", flags.Database.DSN)
	assert.Equal(t, "/tmp/test-key.pem", flags.CryptoKey)
	assert.True(t, flags.Server.EnablePprof)

	// Тест 2: Применение nil конфигурации
	flags = &Flags{}
	flags.Server.Addr = "localhost:8080"
	err = ApplyJSONConfig(flags, nil)
	require.NoError(t, err)
	assert.Equal(t, "localhost:8080", flags.Server.Addr) // Значение не должно измениться

	// Тест 3: Некорректный формат интервала
	flags = &Flags{}
	flags.Server.StoreInterval = defaultStoreInterval
	jsonConfig = &JSONConfig{
		StoreInterval: "invalid",
	}
	err = ApplyJSONConfig(flags, jsonConfig)
	require.Error(t, err)
}
