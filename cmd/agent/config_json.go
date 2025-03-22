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
	GRPCAddress    string `json:"grpc_address"`    // Адрес и порт gRPC сервера
	GRPCEnabled    bool   `json:"grpc_enabled"`    // Флаг использования gRPC вместо HTTP
	GRPCTimeout    int    `json:"grpc_timeout"`    // Таймаут для gRPC запросов в секундах
	GRPCRetry      int    `json:"grpc_retry"`      // Количество повторных попыток при ошибке
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
func ApplyJSONConfig(flags *Flags, jsonConfig *JSONConfig) {
	if jsonConfig == nil {
		return
	}

	// Применяем настройки только если соответствующие флаги не были установлены
	// через командную строку или переменные окружения
	applyServerConfig(flags, jsonConfig)
	applySecurityConfig(flags, jsonConfig)
	applyPerformanceConfig(flags, jsonConfig)
	applyNetworkConfig(flags, jsonConfig)
}

// applyServerConfig применяет настройки сервера из JSON-конфигурации.
func applyServerConfig(flags *Flags, jsonConfig *JSONConfig) {
	// Адрес сервера
	if flags.Server.Addr == defaultAgentServerAddr && jsonConfig.Address != "" {
		flags.Server.Addr = jsonConfig.Address
	}

	// Интервал отправки метрик
	if flags.Server.ReportInterval == defaultReportInterval && jsonConfig.ReportInterval != "" {
		duration, err := time.ParseDuration(jsonConfig.ReportInterval)
		if err == nil {
			flags.Server.ReportInterval = duration.Seconds()
		}
	}

	// Интервал сбора метрик
	if flags.Server.PollInterval == defaultPollInterval && jsonConfig.PollInterval != "" {
		duration, err := time.ParseDuration(jsonConfig.PollInterval)
		if err == nil {
			flags.Server.PollInterval = duration.Seconds()
		}
	}
}

// applySecurityConfig применяет настройки безопасности из JSON-конфигурации.
func applySecurityConfig(flags *Flags, jsonConfig *JSONConfig) {
	// Путь к файлу с публичным ключом для шифрования
	if flags.CryptoKey == "" && jsonConfig.CryptoKey != "" {
		flags.CryptoKey = jsonConfig.CryptoKey
	}
}

// applyPerformanceConfig применяет настройки производительности из JSON-конфигурации.
func applyPerformanceConfig(flags *Flags, jsonConfig *JSONConfig) {
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
}

// applyNetworkConfig применяет сетевые настройки из JSON-конфигурации.
func applyNetworkConfig(flags *Flags, jsonConfig *JSONConfig) {
	// IP-адрес для заголовка X-Real-IP
	if flags.RealIP == "" && jsonConfig.RealIP != "" {
		flags.RealIP = jsonConfig.RealIP
	}

	// gRPC настройки
	if flags.GRPCAddress == defaultGRPCAddress && jsonConfig.GRPCAddress != "" {
		flags.GRPCAddress = jsonConfig.GRPCAddress
	}

	if !flags.GRPCEnabled && jsonConfig.GRPCEnabled {
		flags.GRPCEnabled = jsonConfig.GRPCEnabled
	}

	if flags.GRPCTimeout == defaultGRPCTimeout && jsonConfig.GRPCTimeout > 0 {
		flags.GRPCTimeout = jsonConfig.GRPCTimeout
	}

	if flags.GRPCRetry == defaultGRPCRetry && jsonConfig.GRPCRetry > 0 {
		flags.GRPCRetry = jsonConfig.GRPCRetry
	}
}
