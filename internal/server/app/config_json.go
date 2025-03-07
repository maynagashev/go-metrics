// Package app реализует основную логику работы HTTP-сервера.
package app

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// JSONConfig представляет структуру конфигурационного файла сервера в формате JSON.
type JSONConfig struct {
	Address       string `json:"address"`        // Адрес и порт сервера
	Restore       bool   `json:"restore"`        // Восстанавливать метрики из файла при старте
	StoreInterval string `json:"store_interval"` // Интервал сохранения метрик в виде строки (например, "1s")
	StoreFile     string `json:"store_file"`     // Путь к файлу для хранения метрик
	DatabaseDSN   string `json:"database_dsn"`   // Строка подключения к базе данных
	CryptoKey     string `json:"crypto_key"`     // Путь к файлу с приватным ключом для расшифровки
	EnablePprof   bool   `json:"enable_pprof"`   // Включить профилирование через pprof
}

// LoadJSONConfig загружает конфигурацию из JSON-файла.
// Возвращает nil, если файл не найден или не указан.
func LoadJSONConfig(filePath string) (*JSONConfig, error) {
	if filePath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config JSONConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// ApplyJSONConfig применяет настройки из JSON-конфигурации к флагам.
// Настройки из JSON имеют более низкий приоритет, чем флаги командной строки и переменные окружения.
func ApplyJSONConfig(flags *Flags, jsonConfig *JSONConfig) error {
	if jsonConfig == nil {
		return nil
	}

	// Применяем настройки только если соответствующие флаги не были установлены
	// через командную строку или переменные окружения

	// Адрес сервера
	if flags.Server.Addr == "localhost:8080" && jsonConfig.Address != "" {
		flags.Server.Addr = jsonConfig.Address
	}

	// Интервал сохранения метрик
	if flags.Server.StoreInterval == defaultStoreInterval && jsonConfig.StoreInterval != "" {
		duration, err := time.ParseDuration(jsonConfig.StoreInterval)
		if err != nil {
			return fmt.Errorf("invalid store_interval in config: %w", err)
		}
		flags.Server.StoreInterval = int(duration.Seconds())
	}

	// Путь к файлу для хранения метрик
	if flags.Server.FileStoragePath == "/tmp/metrics-db.json" && jsonConfig.StoreFile != "" {
		flags.Server.FileStoragePath = jsonConfig.StoreFile
	}

	// Восстанавливать метрики из файла при старте
	if flags.Server.Restore && !jsonConfig.Restore {
		flags.Server.Restore = jsonConfig.Restore
	}

	// Строка подключения к базе данных
	if flags.Database.DSN == "" && jsonConfig.DatabaseDSN != "" {
		flags.Database.DSN = jsonConfig.DatabaseDSN
	}

	// Путь к файлу с приватным ключом для расшифровки
	if flags.CryptoKey == "" && jsonConfig.CryptoKey != "" {
		flags.CryptoKey = jsonConfig.CryptoKey
	}

	// Включить профилирование через pprof
	if !flags.Server.EnablePprof && jsonConfig.EnablePprof {
		flags.Server.EnablePprof = jsonConfig.EnablePprof
	}

	return nil
}
