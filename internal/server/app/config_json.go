// Package app реализует основную логику работы HTTP-сервера.
package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	defaultServerAddr      = "localhost:8080"
	defaultFileStoragePath = "/tmp/metrics-db.json"
)

// ErrConfigFileNotSpecified возвращается, когда путь к файлу конфигурации не указан.
var ErrConfigFileNotSpecified = errors.New("config file path not specified")

// DefaultServerAddr возвращает адрес сервера по умолчанию.
func DefaultServerAddr() string {
	return defaultServerAddr
}

// DefaultStoreInterval возвращает интервал сохранения метрик по умолчанию.
func DefaultStoreInterval() int {
	return defaultStoreInterval
}

// DefaultFileStoragePath возвращает путь к файлу хранения метрик по умолчанию.
func DefaultFileStoragePath() string {
	return defaultFileStoragePath
}

// JSONConfig представляет структуру конфигурационного файла сервера в формате JSON.
type JSONConfig struct {
	Address       string `json:"address"`        // Адрес и порт сервера
	Restore       bool   `json:"restore"`        // Восстанавливать метрики из файла при старте
	StoreInterval string `json:"store_interval"` // Интервал сохранения метрик в виде строки (например, "1s")
	StoreFile     string `json:"store_file"`     // Путь к файлу для хранения метрик
	DatabaseDSN   string `json:"database_dsn"`   // Строка подключения к базе данных
	CryptoKey     string `json:"crypto_key"`     // Путь к файлу с приватным ключом для расшифровки
	EnablePprof   bool   `json:"enable_pprof"`   // Включить профилирование через pprof
	TrustedSubnet string `json:"trusted_subnet"` // CIDR доверенной подсети для проверки IP-адресов агентов
}

// LoadJSONConfig загружает конфигурацию из JSON-файла.
// Возвращает ErrConfigFileNotSpecified, если файл не указан.
func LoadJSONConfig(filePath string) (*JSONConfig, error) {
	if filePath == "" {
		return nil, ErrConfigFileNotSpecified
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config JSONConfig
	jsonErr := json.Unmarshal(data, &config)
	if jsonErr != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", jsonErr)
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
	if flags.Server.Addr == defaultServerAddr && jsonConfig.Address != "" {
		flags.Server.Addr = jsonConfig.Address
	}

	// Интервал сохранения метрик
	if flags.Server.StoreInterval == defaultStoreInterval && jsonConfig.StoreInterval != "" {
		duration, durationErr := time.ParseDuration(jsonConfig.StoreInterval)
		if durationErr != nil {
			return fmt.Errorf("invalid store_interval in config: %w", durationErr)
		}
		flags.Server.StoreInterval = int(duration.Seconds())
	}

	// Путь к файлу для хранения метрик
	if flags.Server.FileStoragePath == defaultFileStoragePath && jsonConfig.StoreFile != "" {
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

	// Доверенная подсеть
	if flags.Server.TrustedSubnet == "" && jsonConfig.TrustedSubnet != "" {
		flags.Server.TrustedSubnet = jsonConfig.TrustedSubnet
	}

	return nil
}
