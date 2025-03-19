package app_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/maynagashev/go-metrics/internal/server/app"
)

// Пример использования JSON-конфигурации для сервера.
func Example_jsonConfig() {
	// Создаем временный файл с конфигурацией
	tempDir := os.TempDir()
	configPath := filepath.Join(tempDir, "server-config.json")

	// Пример конфигурации сервера
	configContent := `{
		"address": "localhost:9090",
		"restore": false,
		"store_interval": "5s",
		"store_file": "/tmp/metrics.json",
		"database_dsn": "postgres://user:pass@localhost:5432/metrics",
		"crypto_key": "/path/to/key.pem",
		"enable_pprof": true
	}`

	// Записываем конфигурацию в файл
	if writeErr := os.WriteFile(configPath, []byte(configContent), 0o600); writeErr != nil {
		fmt.Printf("Ошибка при записи файла конфигурации: %v\n", writeErr)
		return
	}
	defer os.Remove(configPath)

	// Загружаем конфигурацию из файла
	jsonConfig, loadErr := app.LoadJSONConfig(configPath)
	if loadErr != nil {
		fmt.Printf("Ошибка при загрузке конфигурации: %v\n", loadErr)
		return
	}

	// Выводим загруженную конфигурацию
	fmt.Printf("Адрес сервера: %s\n", jsonConfig.Address)
	fmt.Printf("Восстанавливать метрики: %v\n", jsonConfig.Restore)
	fmt.Printf("Интервал сохранения: %s\n", jsonConfig.StoreInterval)
	fmt.Printf("Путь к файлу хранения: %s\n", jsonConfig.StoreFile)
	fmt.Printf("DSN базы данных: %s\n", jsonConfig.DatabaseDSN)
	fmt.Printf("Путь к ключу: %s\n", jsonConfig.CryptoKey)
	fmt.Printf("Профилирование включено: %v\n", jsonConfig.EnablePprof)

	// Применяем конфигурацию к флагам
	flags := &app.Flags{}
	flags.Server.Addr = "localhost:8080"
	flags.Server.StoreInterval = 300
	flags.Server.FileStoragePath = "/tmp/metrics-db.json"
	flags.Server.Restore = true
	flags.Database.DSN = ""
	flags.CryptoKey = ""
	flags.Server.EnablePprof = false

	applyErr := app.ApplyJSONConfig(flags, jsonConfig)
	if applyErr != nil {
		fmt.Printf("Ошибка при применении конфигурации: %v\n", applyErr)
		return
	}

	// Выводим результат применения конфигурации
	fmt.Printf("Адрес сервера после применения: %s\n", flags.Server.Addr)
	fmt.Printf("Интервал сохранения после применения: %d секунд\n", flags.Server.StoreInterval)
	fmt.Printf("Путь к файлу хранения после применения: %s\n", flags.Server.FileStoragePath)
	fmt.Printf("Восстанавливать метрики после применения: %v\n", flags.Server.Restore)
	fmt.Printf("DSN базы данных после применения: %s\n", flags.Database.DSN)
	fmt.Printf("Путь к ключу после применения: %s\n", flags.CryptoKey)
	fmt.Printf("Профилирование включено после применения: %v\n", flags.Server.EnablePprof)

	// Output:
	// Адрес сервера: localhost:9090
	// Восстанавливать метрики: false
	// Интервал сохранения: 5s
	// Путь к файлу хранения: /tmp/metrics.json
	// DSN базы данных: postgres://user:pass@localhost:5432/metrics
	// Путь к ключу: /path/to/key.pem
	// Профилирование включено: true
	// Адрес сервера после применения: localhost:9090
	// Интервал сохранения после применения: 5 секунд
	// Путь к файлу хранения после применения: /tmp/metrics.json
	// Восстанавливать метрики после применения: false
	// DSN базы данных после применения: postgres://user:pass@localhost:5432/metrics
	// Путь к ключу после применения: /path/to/key.pem
	// Профилирование включено после применения: true
}
