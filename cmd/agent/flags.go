package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

const defaultReportInterval = 10
const defaultPollInterval = 2

// Flags содержит флаги агента.
type Flags struct {
	Server struct {
		Addr           string
		ReportInterval int
		PollInterval   int
	}
	PrivateKey string
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
	flag.IntVar(&flags.Server.ReportInterval, "r", defaultReportInterval, "report interval in seconds")
	flag.IntVar(&flags.Server.PollInterval, "p", defaultPollInterval, "poll interval in seconds")
	flag.StringVar(&flags.PrivateKey, "k", "", "приватный ключ для подписи запросов к серверу")

	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

	// если переданы переменные окружения, то они перезаписывают
	// значения флагов: envServerAddr, envReportInterval, envPollInterval
	if envServerAddr := os.Getenv("ADDRESS"); envServerAddr != "" {
		flags.Server.Addr = envServerAddr
	}
	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		i, err := strconv.Atoi(envReportInterval)
		if err != nil {
			panic(fmt.Sprintf("error parsing env REPORT_INTERVAL %s", err))
		}
		flags.Server.ReportInterval = i
	}
	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		i, err := strconv.Atoi(envPollInterval)
		if err != nil {
			panic(fmt.Sprintf("error parsing env POLL_INTERVAL %s", err))
		}
		flags.Server.PollInterval = i
	}
	// Если передан ключ в параметрах окружения, используем его
	if envPrivateKey, ok := os.LookupEnv("KEY"); ok {
		flags.PrivateKey = envPrivateKey
	}

	return flags
}
