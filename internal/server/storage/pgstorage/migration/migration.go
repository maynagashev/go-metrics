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

// Up применяет миграции к базе данных.
func Up(path, dsn string) {
	var err error
	slog.Info("Запуск миграций...", "path", path)
	m, err := migrate.New("file://"+path, dsn)
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
