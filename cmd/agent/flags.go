package main

import (
	"flag"
)

// неэкспортированная переменная flagServerAddr содержит адрес и порт для запуска сервера
var flagServerAddr string

var flagReportInterval int
var flagPollInterval int

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {

	flag.StringVar(&flagServerAddr, "a", "localhost:8080", "address and port of the server send metrics to")
	flag.IntVar(&flagReportInterval, "r", 10, "report interval in seconds")
	flag.IntVar(&flagPollInterval, "p", 2, "poll interval in seconds")

	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()
}
