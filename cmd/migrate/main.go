// Утилита для миграции базы данных, обертка над библиотекой golang-migrate/migrate.
package main

import (
	"flag"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/maynagashev/go-metrics/internal/server/storage/pgsql/migration"
)

func main() {
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

	migration.Up(migrationsPath, dsn)
}
