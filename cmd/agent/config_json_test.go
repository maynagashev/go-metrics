package main

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
		"report_interval": "5s",
		"poll_interval": "2s",
		"crypto_key": "/tmp/test-key.pem",
		"rate_limit": 5,
		"enable_pprof": true,
		"pprof_port": "7070"
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	config, err := LoadJSONConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "localhost:9090", config.Address)
	assert.Equal(t, "5s", config.ReportInterval)
	assert.Equal(t, "2s", config.PollInterval)
	assert.Equal(t, "/tmp/test-key.pem", config.CryptoKey)
	assert.Equal(t, 5, config.RateLimit)
	assert.True(t, config.EnablePprof)
	assert.Equal(t, "7070", config.PprofPort)

	// Тест 2: Пустой путь к файлу
	config, err = LoadJSONConfig("")
	require.ErrorIs(t, err, ErrConfigFileNotSpecified)
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
	flags.Server.Addr = defaultAgentServerAddr
	flags.Server.ReportInterval = defaultReportInterval
	flags.Server.PollInterval = defaultPollInterval
	flags.CryptoKey = ""
	flags.RateLimit = defaultRateLimit
	flags.EnablePprof = false
	flags.PprofPort = defaultPprofPort

	jsonConfig := &JSONConfig{
		Address:        "localhost:9090",
		ReportInterval: "5s",
		PollInterval:   "2s",
		CryptoKey:      "/tmp/test-key.pem",
		RateLimit:      5,
		EnablePprof:    true,
		PprofPort:      "7070",
	}

	err := ApplyJSONConfig(flags, jsonConfig)
	require.NoError(t, err)

	assert.Equal(t, "localhost:9090", flags.Server.Addr)
	assert.InEpsilon(t, 5.0, flags.Server.ReportInterval, 0.001) // 5s -> 5 секунд
	assert.InEpsilon(t, 2.0, flags.Server.PollInterval, 0.001)   // 2s -> 2 секунды
	assert.Equal(t, "/tmp/test-key.pem", flags.CryptoKey)
	assert.Equal(t, 5, flags.RateLimit)
	assert.True(t, flags.EnablePprof)
	assert.Equal(t, "7070", flags.PprofPort)

	// Тест 2: Применение nil конфигурации
	flags = &Flags{}
	flags.Server.Addr = defaultAgentServerAddr
	err = ApplyJSONConfig(flags, nil)
	require.NoError(t, err)
	assert.Equal(t, defaultAgentServerAddr, flags.Server.Addr) // Значение не должно измениться

	// Тест 3: Некорректный формат интервала
	flags = &Flags{}
	flags.Server.ReportInterval = defaultReportInterval
	jsonConfig = &JSONConfig{
		ReportInterval: "invalid",
	}
	err = ApplyJSONConfig(flags, jsonConfig)
	require.Error(t, err)
}
