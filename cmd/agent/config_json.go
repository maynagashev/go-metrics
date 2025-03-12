package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// ErrConfigFileNotSpecified возвращается, когда путь к файлу конфигурации не указан.
var ErrConfigFileNotSpecified = errors.New("config file path not specified")

// JSONConfig представляет структуру конфигурационного файла агента в формате JSON.
type JSONConfig struct {
	Address        string `json:"address"`         // Адрес и порт сервера
	ReportInterval string `json:"report_interval"` // Интервал отправки метрик в виде строки (например, "1s")
	PollInterval   string `json:"poll_interval"`   // Интервал сбора метрик в виде строки (например, "1s")
	CryptoKey      string `json:"crypto_key"`      // Путь к файлу с публичным ключом для шифрования
	RateLimit      int    `json:"rate_limit"`      // Максимальное количество одновременно исходящих запросов
	EnablePprof    bool   `json:"enable_pprof"`    // Включить профилирование через pprof
	PprofPort      string `json:"pprof_port"`      // Порт для pprof сервера
	RealIP         string `json:"real_ip"`         // IP-адрес для заголовка X-Real-IP
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
	if flags.Server.Addr == defaultAgentServerAddr && jsonConfig.Address != "" {
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
	if flags.PprofPort == defaultPprofPort && jsonConfig.PprofPort != "" {
		flags.PprofPort = jsonConfig.PprofPort
	}

	// IP-адрес для заголовка X-Real-IP
	if flags.RealIP == "" && jsonConfig.RealIP != "" {
		flags.RealIP = jsonConfig.RealIP
	}

	return nil
}
