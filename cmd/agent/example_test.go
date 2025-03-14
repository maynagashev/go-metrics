package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// Пример использования JSON-конфигурации для агента.
func Example_jsonConfig() {
	// Создаем временный файл с конфигурацией
	tempDir := os.TempDir()
	configPath := filepath.Join(tempDir, "agent-config.json")

	// Пример конфигурации агента
	configContent := `{
		"address": "localhost:9090",
		"report_interval": "5s",
		"poll_interval": "2s",
		"crypto_key": "/path/to/key.pem",
		"rate_limit": 5,
		"enable_pprof": true,
		"pprof_port": "7070"
	}`

	// Записываем конфигурацию в файл
	if writeErr := os.WriteFile(configPath, []byte(configContent), 0o600); writeErr != nil {
		fmt.Printf("Ошибка при записи файла конфигурации: %v\n", writeErr)
		return
	}
	defer os.Remove(configPath)

	// Загружаем конфигурацию из файла
	jsonConfig, loadErr := LoadJSONConfig(configPath)
	if loadErr != nil {
		fmt.Printf("Ошибка при загрузке конфигурации: %v\n", loadErr)
		return
	}

	// Выводим загруженную конфигурацию
	fmt.Printf("Адрес сервера: %s\n", jsonConfig.Address)
	fmt.Printf("Интервал отправки: %s\n", jsonConfig.ReportInterval)
	fmt.Printf("Интервал сбора: %s\n", jsonConfig.PollInterval)
	fmt.Printf("Путь к ключу: %s\n", jsonConfig.CryptoKey)
	fmt.Printf("Лимит запросов: %d\n", jsonConfig.RateLimit)
	fmt.Printf("Профилирование включено: %v\n", jsonConfig.EnablePprof)
	fmt.Printf("Порт профилирования: %s\n", jsonConfig.PprofPort)

	// Применяем конфигурацию к флагам
	flags := &Flags{}
	flags.Server.Addr = "localhost:8080"
	flags.Server.ReportInterval = 10.0
	flags.Server.PollInterval = 2.0
	flags.CryptoKey = ""
	flags.RateLimit = 3
	flags.EnablePprof = false
	flags.PprofPort = "6060"

	ApplyJSONConfig(flags, jsonConfig)

	// Выводим результат применения конфигурации
	fmt.Printf("Адрес сервера после применения: %s\n", flags.Server.Addr)
	fmt.Printf("Интервал отправки после применения: %.1f секунд\n", flags.Server.ReportInterval)
	fmt.Printf("Интервал сбора после применения: %.1f секунд\n", flags.Server.PollInterval)
	fmt.Printf("Путь к ключу после применения: %s\n", flags.CryptoKey)
	fmt.Printf("Лимит запросов после применения: %d\n", flags.RateLimit)
	fmt.Printf("Профилирование включено после применения: %v\n", flags.EnablePprof)
	fmt.Printf("Порт профилирования после применения: %s\n", flags.PprofPort)

	// Output:
	// Адрес сервера: localhost:9090
	// Интервал отправки: 5s
	// Интервал сбора: 2s
	// Путь к ключу: /path/to/key.pem
	// Лимит запросов: 5
	// Профилирование включено: true
	// Порт профилирования: 7070
	// Адрес сервера после применения: localhost:9090
	// Интервал отправки после применения: 5.0 секунд
	// Интервал сбора после применения: 2.0 секунд
	// Путь к ключу после применения: /path/to/key.pem
	// Лимит запросов после применения: 5
	// Профилирование включено после применения: true
	// Порт профилирования после применения: 7070
}
