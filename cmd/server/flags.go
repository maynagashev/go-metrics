package main

import (
	"flag"
	"os"
)

// Неэкспортированная переменная flagRunAddr содержит адрес и порт для запуска сервера.
var flagRunAddr string

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных.
func parseFlags() {
	// Регистрируем переменную flagRunAddr как аргумент -a со значением :8080 по умолчанию.
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	// Парсим переданные серверу аргументы в зарегистрированные переменные.
	flag.Parse()

	// Для случаев, когда в переменной окружения ADDRESS присутствует непустое значение,
	// переопределим адрес запуска сервера,
	// даже если он был передан через аргумент командной строки.
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagRunAddr = envRunAddr
	}
}
