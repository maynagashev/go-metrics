// Package migration предоставляет функционал для управления миграциями базы данных.
// Обеспечивает корректное обновление схемы базы данных при изменениях.
package migration

import (
	"errors"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	// Подключение драйвера для работы с PostgreSQL.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// Подключение драйвера файловой системы, для чтения миграций из файлов.
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Up выполняет миграции базы данных.
func Up(migrationsPath string, dsn string) error {
	var err error
	slog.Info("Запуск миграций...", "path", migrationsPath)
	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		panic(err)
	}

	if err = m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("Нет новых миграций для применения.")
			return nil
		}
		panic(err)
	}

	slog.Info("Миграции применены.")
	return nil
}
