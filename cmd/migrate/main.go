// Утилита для миграции базы данных, обертка над библиотекой golang-migrate/migrate.
package main

import (
	"errors"
	"flag"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var err error
	var dsn, migrationsPath string
	flag.StringVar(&dsn, "d", "",
		"Параметры подключения к базе данных Postgres, формат: postgres://user:password@localhost:5432/database")
	flag.StringVar(&migrationsPath, "migrations-path", "", "Путь к директории с миграциями")
	flag.Parse()

	if dsn == "" {
		panic("Не указаны параметры подключения к БД: -d postgres://user:password@localhost:5432/database")
	}
	if migrationsPath == "" {
		panic("Не указан путь к директории с миграциями: -migrations-path ../../migrations")
	}

	slog.Info("Запуск миграций...")
	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		panic(err)
	}

	if err = m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("Нет новых миграций для применения.")
			return
		}
		panic(err)
	}

	slog.Info("Миграции применены.")
}
