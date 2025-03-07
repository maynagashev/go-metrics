package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// JSONConfig представляет структуру конфигурационного файла агента в формате JSON.
type JSONConfig struct {
	Address        string `json:"address"`         // Адрес и порт сервера
	ReportInterval string `json:"report_interval"` // Интервал отправки метрик в виде строки (например, "1s")
	PollInterval   string `json:"poll_interval"`   // Интервал сбора метрик в виде строки (например, "1s")
	CryptoKey      string `json:"crypto_key"`      // Путь к файлу с публичным ключом для шифрования
	RateLimit      int    `json:"rate_limit"`      // Максимальное количество одновременно исходящих запросов
	EnablePprof    bool   `json:"enable_pprof"`    // Включить профилирование через pprof
	PprofPort      string `json:"pprof_port"`      // Порт для pprof сервера
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

	// Интервал отправки метрик
	if flags.Server.ReportInterval == defaultReportInterval && jsonConfig.ReportInterval != "" {
		duration, err := time.ParseDuration(jsonConfig.ReportInterval)
		if err != nil {
			return fmt.Errorf("invalid report_interval in config: %w", err)
		}
		flags.Server.ReportInterval = duration.Seconds()
	}

	// Интервал сбора метрик
	if flags.Server.PollInterval == defaultPollInterval && jsonConfig.PollInterval != "" {
		duration, err := time.ParseDuration(jsonConfig.PollInterval)
		if err != nil {
			return fmt.Errorf("invalid poll_interval in config: %w", err)
		}
		flags.Server.PollInterval = duration.Seconds()
	}

	// Путь к файлу с публичным ключом для шифрования
	if flags.CryptoKey == "" && jsonConfig.CryptoKey != "" {
		flags.CryptoKey = jsonConfig.CryptoKey
	}

	// Максимальное количество одновременно исходящих запросов
	if flags.RateLimit == defaultRateLimit && jsonConfig.RateLimit > 0 {
		flags.RateLimit = jsonConfig.RateLimit
	}

	// Включить профилирование через pprof
	if !flags.EnablePprof && jsonConfig.EnablePprof {
		flags.EnablePprof = jsonConfig.EnablePprof
	}

	// Порт для pprof сервера
	if flags.PprofPort == "6060" && jsonConfig.PprofPort != "" {
		flags.PprofPort = jsonConfig.PprofPort
	}

	return nil
}
