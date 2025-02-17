// Утилита для миграции базы данных, обертка над библиотекой golang-migrate/migrate.
package main

import (
	"errors"
	"flag"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/maynagashev/go-metrics/internal/server/storage/pgstorage/migration"
)

func run(dsn, migrationsPath string) error {
	if dsn == "" {
		return errors.New("не указаны параметры подключения к БД: -d postgres://user:password@localhost:5432/database")
	}
	if migrationsPath == "" {
		return errors.New("не указан путь к директории с миграциями: -migrations-path ../../migrations")
	}

	if err := migration.Up(migrationsPath, dsn); err != nil {
		return err
	}
	return nil
}

func main() {
	var dsn, migrationsPath string
	flag.StringVar(&dsn, "d", "",
		"Параметры подключения к базе данных Postgres, формат: postgres://user:password@localhost:5432/database")
	flag.StringVar(&migrationsPath, "migrations-path", "", "Путь к директории с миграциями")
	flag.Parse()

	if err := run(dsn, migrationsPath); err != nil {
		panic(err)
	}
}
