// Package main предоставляет тесты для мультичекера staticlint.
package main

import (
	"os"
	"testing"

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
