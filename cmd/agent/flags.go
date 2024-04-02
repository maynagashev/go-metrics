package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Неэкспортированная переменная flagServerAddr содержит адрес и порт для запуска сервера.
var flagServerAddr string

var flagReportInterval int
var flagPollInterval int

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных.
func parseFlags() error {
	var err error

	flag.StringVar(&flagServerAddr, "a", "localhost:8080", "address and port of the server send metrics to")
	flag.IntVar(&flagReportInterval, "r", 10, "report interval in seconds")
	flag.IntVar(&flagPollInterval, "p", 2, "poll interval in seconds")

	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

	if envServerAddr := os.Getenv("ADDRESS"); envServerAddr != "" {
		flagServerAddr = envServerAddr
	}

	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		flagReportInterval, err = strconv.Atoi(envReportInterval)
		if err != nil {
			return fmt.Errorf("error parsing env REPORT_INTERVAL %w", err)
		}
	}
	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		flagPollInterval, err = strconv.Atoi(envPollInterval)
		if err != nil {
			return fmt.Errorf("error parsing env POLL_INTERVAL %w", err)
		}
	}

	return nil
}
