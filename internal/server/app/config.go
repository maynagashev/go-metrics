package app

type Config struct {
	Addr string
	// Интервал сохранения метрик на сервере в секундах.
	StoreInterval int
	// Полное имя файла, в который будут сохранены метрики.
	FileStoragePath string
	// Загружать или нет ранее сохраненные метрики из файла.
	Restore bool
	// Параметры базы данных
	Database DatabaseConfig
}

type DatabaseConfig struct {
	DSN            string
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
