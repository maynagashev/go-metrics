package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
)

const (
	defaultReportInterval  = 10.0
	defaultPollInterval    = 2.0
	defaultRateLimit       = 3
	minInterval            = 0.000001 // Минимально допустимый интервал в секундах.
	defaultPprofPort       = "6060"
	defaultAgentServerAddr = "localhost:8080" // Адрес и порт сервера по умолчанию
	defaultGRPCAddress     = "localhost:9090" // Адрес и порт gRPC сервера по умолчанию
	defaultGRPCTimeout     = 5                // Таймаут для gRPC запросов в секундах по умолчанию
	defaultGRPCRetry       = 3                // Количество повторных попыток при ошибке по умолчанию
)

// Flags содержит флаги агента.
type Flags struct {
	Server struct {
		Addr           string
		ReportInterval float64
		PollInterval   float64
	}
	PrivateKey  string // путь к файлу с приватным ключом для подписи запросов к серверу
	CryptoKey   string // путь к файлу с публичным ключом для шифрования
	RateLimit   int
	EnablePprof bool   // добавляем поле для профилирования
	PprofPort   string // добавляем порт для pprof
	ConfigFile  string // путь к файлу конфигурации в формате JSON
	RealIP      string // явно указанный IP-адрес для заголовка X-Real-IP
	GRPCAddress string // адрес и порт gRPC сервера
	GRPCEnabled bool   // флаг использования gRPC вместо HTTP
	GRPCTimeout int    // таймаут для gRPC запросов в секундах
	GRPCRetry   int    // количество повторных попыток при ошибке
}

// mustParseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных.
func mustParseFlags() Flags {
	flags := Flags{}

	// Регистрируем флаги командной строки
	registerCommandLineFlags(&flags)

	// Парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

	// Применяем переменные окружения
	applyEnvironmentVariables(&flags)

	// Загружаем и применяем JSON-конфигурацию
	applyJSONConfig(&flags)

	// Проверяем и корректируем значения
	validateFlags(&flags)

	return flags
}

// registerCommandLineFlags регистрирует флаги командной строки.
func registerCommandLineFlags(flags *Flags) {
	flag.StringVar(
		&flags.Server.Addr,
		"a",
		defaultAgentServerAddr,
		"address and port of the server send metrics to",
	)
	flag.Float64Var(
		&flags.Server.ReportInterval,
		"r",
		defaultReportInterval,
		"report interval in seconds",
	)
	flag.Float64Var(
		&flags.Server.PollInterval,
		"p",
		defaultPollInterval,
		"poll interval in seconds",
	)
	flag.StringVar(&flags.PrivateKey, "k", "", "приватный ключ для подписи запросов к серверу")
	flag.StringVar(
		&flags.CryptoKey,
		"crypto-key",
		"",
		"путь к файлу с публичным ключом для шифрования",
	)
	flag.IntVar(
		&flags.RateLimit,
		"l",
		defaultRateLimit,
		"макс. количество одновременно исходящих запросов на сервер",
	)
	flag.BoolVar(&flags.EnablePprof, "pprof", false, "enable pprof profiling")
	flag.StringVar(&flags.PprofPort, "pprof-port", defaultPprofPort, "port for pprof server")

	// Добавляем флаг для явного указания IP-адреса для заголовка X-Real-IP
	flag.StringVar(&flags.RealIP, "real-ip", "", "IP address to use in X-Real-IP header")

	// Добавляем флаги для gRPC
	flag.StringVar(
		&flags.GRPCAddress,
		"grpc-address",
		defaultGRPCAddress,
		"адрес и порт gRPC сервера",
	)
	flag.BoolVar(&flags.GRPCEnabled, "grpc-enabled", false, "использовать gRPC вместо HTTP")
	flag.IntVar(
		&flags.GRPCTimeout,
		"grpc-timeout",
		defaultGRPCTimeout,
		"таймаут для gRPC запросов в секундах",
	)
	flag.IntVar(
		&flags.GRPCRetry,
		"grpc-retry",
		defaultGRPCRetry,
		"количество повторных попыток при ошибке",
	)

	// Добавляем флаг для пути к файлу конфигурации
	flag.StringVar(&flags.ConfigFile, "c", "", "путь к файлу конфигурации в формате JSON")
	flag.StringVar(&flags.ConfigFile, "config", "", "путь к файлу конфигурации в формате JSON")
}

// applyEnvironmentVariables применяет переменные окружения к флагам.
func applyEnvironmentVariables(flags *Flags) {
	applyServerEnvVariables(flags)
	applySecurityEnvVariables(flags)
	applyPerformanceEnvVariables(flags)
	applyNetworkEnvVariables(flags)

	// Если передан путь к файлу конфигурации в параметрах окружения, используем его
	if envConfigFile, ok := os.LookupEnv("CONFIG"); ok {
		flags.ConfigFile = envConfigFile
	}
}

// applyServerEnvVariables применяет переменные окружения для настроек сервера.
func applyServerEnvVariables(flags *Flags) {
	// если переданы переменные окружения, то они перезаписывают
	// значения флагов: envServerAddr, envReportInterval, envPollInterval
	if envServerAddr := os.Getenv("ADDRESS"); envServerAddr != "" {
		flags.Server.Addr = envServerAddr
	}
	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		i, err := strconv.ParseFloat(envReportInterval, 64)
		if err != nil {
			panic(fmt.Sprintf("error parsing env REPORT_INTERVAL %s", err))
		}
		flags.Server.ReportInterval = i
	}
	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		i, err := strconv.ParseFloat(envPollInterval, 64)
		if err != nil {
			panic(fmt.Sprintf("error parsing env POLL_INTERVAL %s", err))
		}
		flags.Server.PollInterval = i
	}
}

// applySecurityEnvVariables применяет переменные окружения для настроек безопасности.
func applySecurityEnvVariables(flags *Flags) {
	if envPrivateKey, ok := os.LookupEnv("KEY"); ok {
		flags.PrivateKey = envPrivateKey
	}
	if envCryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		flags.CryptoKey = envCryptoKey
	}
}

// applyPerformanceEnvVariables применяет переменные окружения для настроек производительности.
func applyPerformanceEnvVariables(flags *Flags) {
	if envRateLimit, ok := os.LookupEnv("RATE_LIMIT"); ok {
		l, err := strconv.Atoi(envRateLimit)
		if err != nil {
			panic(fmt.Sprintf("error parsing env RATE_LIMIT %s", err))
		}
		flags.RateLimit = l
	}
}

// applyNetworkEnvVariables применяет переменные окружения для сетевых настроек.
func applyNetworkEnvVariables(flags *Flags) {
	// Добавляем обработку переменной окружения для X-Real-IP
	if envRealIP, ok := os.LookupEnv("REAL_IP"); ok {
		flags.RealIP = envRealIP
	}

	// Добавляем обработку переменных окружения для gRPC
	applyGRPCEnvVariables(flags)
}

// applyGRPCEnvVariables применяет переменные окружения для настроек gRPC.
func applyGRPCEnvVariables(flags *Flags) {
	if envGRPCAddress, ok := os.LookupEnv("GRPC_ADDRESS"); ok {
		flags.GRPCAddress = envGRPCAddress
	}

	if envGRPCEnabled, ok := os.LookupEnv("GRPC_ENABLED"); ok {
		enabled, err := strconv.ParseBool(envGRPCEnabled)
		if err == nil {
			flags.GRPCEnabled = enabled
		}
	}

	if envGRPCTimeout, ok := os.LookupEnv("GRPC_TIMEOUT"); ok {
		timeout, err := strconv.Atoi(envGRPCTimeout)
		if err == nil {
			flags.GRPCTimeout = timeout
		}
	}

	if envGRPCRetry, ok := os.LookupEnv("GRPC_RETRY"); ok {
		retry, err := strconv.Atoi(envGRPCRetry)
		if err == nil {
			flags.GRPCRetry = retry
		}
	}
}

// applyJSONConfig загружает и применяет JSON-конфигурацию.
func applyJSONConfig(flags *Flags) {
	// Загружаем конфигурацию из JSON-файла, если он указан
	jsonConfig, configErr := LoadJSONConfig(flags.ConfigFile)
	if configErr != nil {
		// Если файл конфигурации не указан, это не ошибка
		if errors.Is(configErr, ErrConfigFileNotSpecified) {
			return
		}
		panic(fmt.Sprintf("error loading config file: %s", configErr))
	}

	// Применяем настройки из JSON-конфигурации (с более низким приоритетом)
	ApplyJSONConfig(flags, jsonConfig)
}

// validateFlags проверяет и корректирует значения флагов.
func validateFlags(flags *Flags) {
	if flags.RateLimit < 1 {
		panic("RateLimit should be greater than 0")
	}

	// Устанавливаем минимальные допустимые значения для интервалов
	if flags.Server.ReportInterval < minInterval {
		flags.Server.ReportInterval = minInterval
	}
	if flags.Server.PollInterval < minInterval {
		flags.Server.PollInterval = minInterval
	}

	// Валидация gRPC параметров
	if flags.GRPCTimeout < 1 {
		flags.GRPCTimeout = defaultGRPCTimeout
	}

	if flags.GRPCRetry < 0 {
		flags.GRPCRetry = defaultGRPCRetry
	}
}
