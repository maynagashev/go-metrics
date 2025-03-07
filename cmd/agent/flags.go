package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

const (
	defaultReportInterval = 10.0
	defaultPollInterval   = 2.0
	defaultRateLimit      = 3
	minInterval           = 0.000001 // Минимально допустимый интервал в секундах.
)

// Flags содержит флаги агента.
type Flags struct {
	Server struct {
		Addr           string
		ReportInterval float64
		PollInterval   float64
	}
	PrivateKey  string
	CryptoKey   string // путь к файлу с публичным ключом для шифрования
	RateLimit   int
	EnablePprof bool   // добавляем поле для профилирования
	PprofPort   string // добавляем порт для pprof
	ConfigFile  string // путь к файлу конфигурации в формате JSON
}

// mustParseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных.
func mustParseFlags() Flags {
	flags := Flags{}

	flag.StringVar(
		&flags.Server.Addr,
		"a",
		"localhost:8080",
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
	flag.StringVar(&flags.CryptoKey, "crypto-key", "", "путь к файлу с публичным ключом для шифрования")
	flag.IntVar(
		&flags.RateLimit,
		"l",
		defaultRateLimit,
		"макс. количество одновременно исходящих запросов на сервер",
	)
	flag.BoolVar(&flags.EnablePprof, "pprof", false, "enable pprof profiling")
	flag.StringVar(&flags.PprofPort, "pprof-port", "6060", "port for pprof server")

	// Добавляем флаг для пути к файлу конфигурации
	flag.StringVar(&flags.ConfigFile, "c", "", "путь к файлу конфигурации в формате JSON")
	flag.StringVar(&flags.ConfigFile, "config", "", "путь к файлу конфигурации в формате JSON")

	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

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
	if envPrivateKey, ok := os.LookupEnv("KEY"); ok {
		flags.PrivateKey = envPrivateKey
	}
	if envCryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		flags.CryptoKey = envCryptoKey
	}
	if envRateLimit, ok := os.LookupEnv("RATE_LIMIT"); ok {
		l, err := strconv.Atoi(envRateLimit)
		if err != nil {
			panic(fmt.Sprintf("error parsing env RATE_LIMIT %s", err))
		}
		flags.RateLimit = l
	}

	// Если передан путь к файлу конфигурации в параметрах окружения, используем его
	if envConfigFile, ok := os.LookupEnv("CONFIG"); ok {
		flags.ConfigFile = envConfigFile
	}

	// Загружаем конфигурацию из JSON-файла, если он указан
	jsonConfig, err := LoadJSONConfig(flags.ConfigFile)
	if err != nil {
		panic(fmt.Sprintf("error loading config file: %s", err))
	}

	// Применяем настройки из JSON-конфигурации (с более низким приоритетом)
	if err := ApplyJSONConfig(&flags, jsonConfig); err != nil {
		panic(fmt.Sprintf("error applying config: %s", err))
	}

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

	return flags
}
