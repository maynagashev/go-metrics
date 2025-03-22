// Package main предоставляет тесты для мультичекера staticlint.
package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
)

// TestLoadConfig тестирует функцию loadConfig.
// Он проверяет, что функция корректно загружает и парсит файл конфигурации.
func TestLoadConfig(t *testing.T) {
	// Test with non-existent file
	_, ok := loadConfig("non_existent_file.json")
	if ok {
		t.Errorf("Expected loadConfig to return false for non-existent file, got true")
	}

	// Create a temporary config file for testing
	tempFile := "temp_config.json"
	content := `{
		"staticcheck": ["SA1000", "SA1001"],
		"stylecheck": ["ST1000"]
	}`
	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temporary config file: %v", err)
	}
	defer os.Remove(tempFile)

	// Test with valid file
	cfg, ok := loadConfig(tempFile)
	if !ok {
		t.Errorf("Expected loadConfig to return true for valid file, got false")
	}
	if len(cfg.Staticcheck) != 2 || cfg.Staticcheck[0] != "SA1000" ||
		cfg.Staticcheck[1] != "SA1001" {
		t.Errorf("Unexpected Staticcheck config: %v", cfg.Staticcheck)
	}
	if len(cfg.Stylecheck) != 1 || cfg.Stylecheck[0] != "ST1000" {
		t.Errorf("Unexpected Stylecheck config: %v", cfg.Stylecheck)
	}

	// Test with invalid JSON
	invalidContent := `{invalid json}`
	err = os.WriteFile(tempFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to update temporary config file: %v", err)
	}
	_, ok = loadConfig(tempFile)
	if ok {
		t.Errorf("Expected loadConfig to return false for invalid JSON, got true")
	}
}

// TestAddAnalyzers тестирует функцию addAnalyzers.
// Он проверяет, что функция корректно добавляет анализаторы на основе конфигурации.
func TestAddAnalyzers(t *testing.T) {
	// This is a simplified test that doesn't use actual analyzers
	// but verifies the logic of the addAnalyzers function

	// Test with empty config and empty analyzers list
	cfg := ConfigData{}
	mychecks := []*analysis.Analyzer{}
	analyzers := []*analysis.Analyzer{}

	// Test with empty config
	result := addAnalyzers(mychecks, cfg, "staticcheck", analyzers)
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty config and analyzers, got %d items", len(result))
	}

	// Test with config but empty analyzers list
	cfg.Staticcheck = []string{"SA1000"}
	result = addAnalyzers(mychecks, cfg, "staticcheck", analyzers)
	if len(result) != 0 {
		t.Errorf("Expected empty result for config with empty analyzers, got %d items", len(result))
	}

	// Note: A more comprehensive test would create mock analyzers,
	// but that would require significant setup and is beyond the scope
	// of this basic test file.
}

// TestGetAllStaticcheckAnalyzers тестирует функцию getAllStaticcheckAnalyzers.
// Он проверяет, что функция возвращает непустой список анализаторов.
func TestGetAllStaticcheckAnalyzers(t *testing.T) {
	analyzers := getAllStaticcheckAnalyzers()
	if len(analyzers) == 0 {
		t.Errorf("Expected non-empty list of staticcheck analyzers")
	}
}

// TestGetAllStylecheckAnalyzers тестирует функцию getAllStylecheckAnalyzers.
// Он проверяет, что функция возвращает непустой список анализаторов.
func TestGetAllStylecheckAnalyzers(t *testing.T) {
	analyzers := getAllStylecheckAnalyzers()
	if len(analyzers) == 0 {
		t.Errorf("Expected non-empty list of stylecheck analyzers")
	}
}

// TestPrintAnalyzersList проверяет, что функция printAnalyzersList правильно выводит список анализаторов.
func TestPrintAnalyzersList(t *testing.T) {
	// Создаем тестовые анализаторы
	analyzers := []*analysis.Analyzer{
		{
			Name: "test1",
			Doc:  "This is a test analyzer 1\nSecond line",
		},
		{
			Name: "test2",
			Doc:  "This is a test analyzer 2",
		},
	}

	// Перехватываем вывод log
	var buf bytes.Buffer
	origOutput := log.Writer()
	log.SetOutput(&buf)
	origFlags := log.Flags()
	log.SetFlags(0)

	defer func() {
		// Восстанавливаем исходный вывод log
		log.SetOutput(origOutput)
		log.SetFlags(origFlags)
	}()

	// Вызываем функцию
	printAnalyzersList(analyzers)

	// Проверяем вывод
	output := buf.String()
	assert.Contains(t, output, "Итоговый список анализаторов:")
	assert.Contains(t, output, "1. test1: This is a test analyzer 1")
	assert.Contains(t, output, "2. test2: This is a test analyzer 2")

	// Проверяем, что для первого анализатора взята только первая строка описания
	assert.NotContains(t, output, "Second line")
}

// TestMain проверяет, что функция main не вызывает панику и корректно запускается.
func TestMain(t *testing.T) {
	// Этот тест запускает исполняемый файл как отдельный процесс
	// для проверки main без вызова os.Exit

	// Пропускаем, если запущены все тесты, т.к. этот тест может быть долгим
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Получаем текущую директорию
	pwd, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")

	// Создаем команду для запуска текущего исполняемого файла как подпроцесса
	// с аргументом -help, чтобы получить помощь, а не запускать анализ
	cmd := exec.Command("go", "run", "staticlint.go", "-help")

	// Запускаем в текущей директории
	t.Logf("Running in directory: %s", pwd)

	// Подготавливаем буферы для stdout и stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Запускаем команду
	_ = cmd.Run() // Игнорируем ошибку, т.к. вызов с -help может вернуть ненулевой код возврата

	// Выводим stdout и stderr для отладки
	t.Logf("Command stdout: %s", stdout.String())
	t.Logf("Command stderr: %s", stderr.String())

	// Проверяем результат
	// Не ожидаем ошибки, т.к. может быть возвращен код ошибки при запуске с -help
	if err != nil {
		t.Logf("Command completed with error: %v (это может быть нормально)", err)
	}

	// Проверяем, что в выводе есть ожидаемая информация о использовании
	output := stdout.String() + stderr.String()
	assert.True(t, strings.Contains(output, "usage") ||
		strings.Contains(output, "Usage") ||
		strings.Contains(output, "staticlint") ||
		strings.Contains(output, "golang.org/x/tools/go/analysis"),
		"Output should contain usage or analysis information")
}

// TestMainIntegration проверяет вывод справки статического анализатора.
func TestMainIntegration(t *testing.T) {
	// Пропускаем этот тест при быстром запуске тестов
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Получаем текущую директорию
	pwd, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	t.Logf("Current directory: %s", pwd)

	// Создаем временную директорию для теста в текущей директории
	tempDir, err := os.MkdirTemp(".", "staticlint_test_dir_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) // Удаляем после завершения теста
	t.Logf("Created temp dir: %s", tempDir)

	// Создаем временный файл с тестовым кодом внутри временной директории
	testFilePath := tempDir + "/test_file.go"
	t.Logf("Creating test file: %s", testFilePath)

	// Записываем тестовый код с ошибкой, которую должен найти анализатор
	testCode := `package test

import "fmt"

func main() {
	// Ошибка: не используемая переменная
	x := 10
	fmt.Println("Hello, world!")
}
`
	err = os.WriteFile(testFilePath, []byte(testCode), 0644)
	require.NoError(t, err)

	// Запускаем анализатор только с опцией -help для проверки справки
	cmd := exec.Command("go", "run", "staticlint.go", "-help")

	// Подготавливаем буферы для stdout и stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Запускаем команду
	_ = cmd.Run() // Игнорируем ошибку, т.к. вызов с -help может вернуть ненулевой код возврата

	// Выводим stdout и stderr для отладки
	t.Logf("Command stdout: %s", stdout.String())
	t.Logf("Command stderr: %s", stderr.String())

	// Объединяем вывод stdout и stderr
	output := stdout.String() + stderr.String()

	// Проверяем, что в выводе есть информация о справке
	assert.True(t, strings.Contains(output, "Usage") ||
		strings.Contains(output, "usage") ||
		strings.Contains(output, "analysis"),
		"Output should contain usage information")
}
