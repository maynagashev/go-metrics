package app

import (
	"flag"
	"os"
	"strconv"
)

const defaultStoreInterval = 300

// Flags содержит все флаги сервера.
type Flags struct {
	Server struct {
		Addr string
		// Интервал сохранения метрик на сервере в секундах
		StoreInterval int
		// Полное имя файла, в который будут сохранены метрики
		FileStoragePath string
		// Загружать или нет ранее сохраненные метрики из файла
		Restore bool
		// Включить профилирование через pprof
		EnablePprof bool
	}

	Database struct {
		// Параметры подключения к БД, например postgres://username:password@localhost:5432/database_name
		DSN string
		// Путь к директории с миграциями
		MigrationsPath string
	}

	PrivateKey string
}

// ParseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных.
func ParseFlags() (*Flags, error) {
	flags := Flags{}
	var err error

	// Регистрируем переменную flagRunAddr как аргумент -a со значением ":8080" по умолчанию.
	flag.StringVar(
		&flags.Server.Addr,
		"a",
		"localhost:8080",
		"IP  адрес и порт на которых следует запустить сервер",
	)
	// Регистрируем переменную flagStoreInterval как аргумент -i со значением 300 по умолчанию.
	flag.IntVar(
		&flags.Server.StoreInterval,
		"i",
		defaultStoreInterval,
		"Интервал сохранения метрик на диск, в секундах",
	)
	// Регистрируем переменную flagFileStoragePath как аргумент -f со значением metrics.json по умолчанию.
	flag.StringVar(
		&flags.Server.FileStoragePath,
		"f",
		"/tmp/metrics-db.json",
		"Путь к файлу для хранения метрик",
	)
	// Регистрируем переменную flagRestore как аргумент -r со значением false по умолчанию.
	flag.BoolVar(&flags.Server.Restore, "r", true, "Восстанавливать метрики из файла при старте?")

	// Добавляем флаг профилирования
	flag.BoolVar(
		&flags.Server.EnablePprof,
		"pprof",
		false,
		"enable pprof profiling with /debug/pprof routes",
	)

	// Адрес подключения к БД PostgresSQL, по умолчанию пустое значение (не подключаемся к БД).
	flag.StringVar(
		&flags.Database.DSN,
		"d",
		"",
		"Параметры подключения к базе данных Postgres, формат: postgres://user:password@localhost:5432/database",
	)
	// Путь к директории с миграциями относительно корня проекта, по умолчанию "migrations/server".
	flag.StringVar(&flags.Database.MigrationsPath,
		"migrations-path",
		"migrations/server",
		"Путь к директории с миграциями")

	flag.StringVar(&flags.PrivateKey, "k", "", "Приватный ключ для подписи запросов к серверу")

	// Парсим переданные серверу аргументы в зарегистрированные переменные.
	flag.Parse()

	// Для случаев, когда в переменной окружения ADDRESS присутствует непустое значение,
	// переопределим адрес запуска сервера,
	// даже если он был передан через аргумент командной строки.
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flags.Server.Addr = envRunAddr
	}
	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		flags.Server.StoreInterval, err = strconv.Atoi(envStoreInterval)
		if err != nil {
			return nil, err
		}
	}
	// Если переменная окружения FILE_STORAGE_PATH присутствует (даже
	// пустая), переопределим путь к файлу хранения метрик.
	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		flags.Server.FileStoragePath = envFileStoragePath
	}
	// Если переменная окружения RESTORE присутствует (даже пустая), переопределим флаг восстановления метрик из файла.
	if envRestore, ok := os.LookupEnv("RESTORE"); ok {
		flags.Server.Restore, err = strconv.ParseBool(envRestore)
		if err != nil {
			return nil, err
		}
	}

	// Если переданы параметры БД в параметрах окружения, используем их
	if envDatabaseDSN, ok := os.LookupEnv("DATABASE_DSN"); ok {
		flags.Database.DSN = envDatabaseDSN
	}

	// Если передан ключ в параметрах окружения, используем его
	if envPrivateKey, ok := os.LookupEnv("KEY"); ok {
		flags.PrivateKey = envPrivateKey
	}

	return &flags, nil
}
