package main

import (
	"flag"
	"os"
)

// Flags содержит все флаги сервера.
type Flags struct {
	Server struct {
		Addr string
	}
}

// mustParseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных.
func mustParseFlags() Flags {
	flags := Flags{}
	// Регистрируем переменную flagRunAddr как аргумент -a со значением :8080 по умолчанию.
	flag.StringVar(&flags.Server.Addr, "a", "localhost:8080", "address and port to run server")
	// Парсим переданные серверу аргументы в зарегистрированные переменные.
	flag.Parse()

	// Для случаев, когда в переменной окружения ADDRESS присутствует непустое значение,
	// переопределим адрес запуска сервера,
	// даже если он был передан через аргумент командной строки.
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flags.Server.Addr = envRunAddr
	}
	return flags
}
