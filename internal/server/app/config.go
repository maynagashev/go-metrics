package app

// Config содержит конфигурацию сервера метрик.
type Config struct {
	// Addr адрес и порт для запуска сервера.
	Addr string
	// Интервал сохранения метрик на сервере в секундах.
	StoreInterval int
	// Полное имя файла, в который будут сохранены метрики.
	FileStoragePath string
	// Загружать или нет ранее сохраненные метрики из файла.
	Restore bool
	// Параметры базы данных
	Database DatabaseConfig
	// Приватный ключ для подписи метрик.
	PrivateKey string
	// Включить профилирование через pprof
	EnablePprof bool
}

// DatabaseConfig содержит настройки подключения к базе данных.
type DatabaseConfig struct {
	// DSN строка подключения к базе данных.
	DSN string
	// MigrationsPath путь к директории с миграциями.
	MigrationsPath string
}

func NewConfig(flags *Flags) *Config {
	return &Config{
		Addr:            flags.Server.Addr,
		StoreInterval:   flags.Server.StoreInterval,
		FileStoragePath: flags.Server.FileStoragePath,
		Restore:         flags.Server.Restore,
		Database: DatabaseConfig{
			DSN:            flags.Database.DSN,
			MigrationsPath: flags.Database.MigrationsPath,
		},
		PrivateKey:  flags.PrivateKey,
		EnablePprof: flags.Server.EnablePprof,
	}
}

// IsStoreEnabled возвращает true, если включено сохранение метрик на сервере.
func (cfg *Config) IsStoreEnabled() bool {
	return cfg.FileStoragePath != ""
}

// IsRestoreEnabled надо ли восстанавливать метрики из файла при старте.
func (cfg *Config) IsRestoreEnabled() bool {
	return cfg.Restore
}

// GetStorePath возвращает путь к файлу для сохранения метрик.
func (cfg *Config) GetStorePath() string {
	return cfg.FileStoragePath
}

// IsSyncStore сохранение метрик на сервере синхронно (сразу после изменения, если нулевой интервал).
func (cfg *Config) IsSyncStore() bool {
	return cfg.StoreInterval == 0
}

// GetStoreInterval возвращает интервал сохранения метрик на сервере в секундах.
func (cfg *Config) GetStoreInterval() int {
	return cfg.StoreInterval
}

// IsDatabaseEnabled возвращает true, если переданы параметры подключения к БД.
func (cfg *Config) IsDatabaseEnabled() bool {
	return cfg.Database.DSN != ""
}

// IsRequestSigningEnabled включена ли проверка подписи метрик.
func (cfg *Config) IsRequestSigningEnabled() bool {
	return cfg.PrivateKey != ""
}
