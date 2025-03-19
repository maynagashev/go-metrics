package app

import (
	"crypto/rsa"
	"log/slog"
	"os"

	"github.com/maynagashev/go-metrics/pkg/crypto"
)

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
	// Параметры gRPC сервера
	GRPC GRPCConfig
	// Приватный ключ для подписи метрик.
	PrivateKey string
	// Включить профилирование через pprof
	EnablePprof bool
	// CryptoKey путь к файлу приватного ключа для шифрования для настройки TLS в gRPC сервере
	CryptoKey string
	// Приватный ключ для расшифровки данных загруженный из CryptoKey, используется в HTTP сервере
	PrivateRSAKey *rsa.PrivateKey
	// Конфигурационный файл
	ConfigFile string
	// TrustedSubnet CIDR доверенной подсети для проверки IP-адресов агентов
	TrustedSubnet string
}

// DatabaseConfig содержит настройки подключения к базе данных.
type DatabaseConfig struct {
	// DSN строка подключения к базе данных.
	DSN string
	// MigrationsPath путь к директории с миграциями.
	MigrationsPath string
}

// GRPCConfig содержит настройки gRPC сервера.
type GRPCConfig struct {
	// Addr адрес и порт для gRPC сервера.
	Addr string
	// Enabled включен ли gRPC сервер.
	Enabled bool
	// MaxConn максимальное количество одновременных соединений.
	MaxConn int
	// Timeout таймаут для gRPC запросов в секундах.
	Timeout int
}

func NewConfig(flags *Flags) *Config {
	cfg := &Config{
		Addr:            flags.Server.Addr,
		StoreInterval:   flags.Server.StoreInterval,
		FileStoragePath: flags.Server.FileStoragePath,
		Restore:         flags.Server.Restore,
		Database: DatabaseConfig{
			DSN:            flags.Database.DSN,
			MigrationsPath: flags.Database.MigrationsPath,
		},
		GRPC: GRPCConfig{
			Addr:    flags.GRPC.Addr,
			Enabled: flags.GRPC.Enabled,
			MaxConn: flags.GRPC.MaxConn,
			Timeout: flags.GRPC.Timeout,
		},
		PrivateKey:    flags.PrivateKey,
		EnablePprof:   flags.Server.EnablePprof,
		ConfigFile:    flags.ConfigFile,
		TrustedSubnet: flags.Server.TrustedSubnet,
		CryptoKey:     flags.CryptoKey,
	}

	// Load private key for decryption if provided
	if flags.CryptoKey != "" {
		var err error
		cfg.PrivateRSAKey, err = crypto.LoadPrivateKey(flags.CryptoKey)
		if err != nil {
			slog.Error("failed to load private key", "error", err, "path", flags.CryptoKey)
			os.Exit(1)
		}
		slog.Info("loaded private key for decryption", "path", flags.CryptoKey)
	}

	return cfg
}

// IsStoreEnabled возвращает true, если включено сохранение метрик на сервере.
func (cfg *Config) IsStoreEnabled() bool {
	return cfg.FileStoragePath != ""
}

// IsRestoreEnabled надо ли восстанавливать метрики из файла при старте.
func (cfg *Config) IsRestoreEnabled() bool {
	return cfg.Restore && cfg.IsStoreEnabled()
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

// IsEncryptionEnabled возвращает true, если включено шифрование.
func (cfg *Config) IsEncryptionEnabled() bool {
	return cfg.PrivateRSAKey != nil
}

// IsTrustedSubnetEnabled возвращает true, если указана доверенная подсеть.
func (cfg *Config) IsTrustedSubnetEnabled() bool {
	return cfg.TrustedSubnet != ""
}

// IsGRPCEnabled возвращает true, если включен gRPC сервер.
func (cfg *Config) IsGRPCEnabled() bool {
	return cfg.GRPC.Enabled
}

// GetCryptoKeyPath возвращает путь к файлу с ключом для шифрования.
func (cfg *Config) GetCryptoKeyPath() string {
	return cfg.CryptoKey
}
