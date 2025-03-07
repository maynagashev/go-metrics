package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintVersion(t *testing.T) {
	// Сохраняем оригинальный stdout
	oldStdout := os.Stdout

	// Создаем буфер для перехвата вывода
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Устанавливаем тестовые значения для версии
	origBuildVersion := BuildVersion
	origBuildDate := BuildDate
	origBuildCommit := BuildCommit

	BuildVersion = "v1.0.0"
	BuildDate = "2023-01-01"
	BuildCommit = "abc123"

	// Вызываем функцию, которую тестируем
	printVersion()

	// Закрываем writer и восстанавливаем stdout
	w.Close()
	os.Stdout = oldStdout

	// Читаем перехваченный вывод
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("Failed to copy output: %v", err)
	}

	output := buf.String()

	// Проверяем, что вывод содержит ожидаемые строки
	expectedLines := []string{
		"Build version: v1.0.0",
		"Build date: 2023-01-01",
		"Build commit: abc123",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("Expected output to contain %q, but got: %q", line, output)
		}
	}

	// Восстанавливаем оригинальные значения
	BuildVersion = origBuildVersion
	BuildDate = origBuildDate
	BuildCommit = origBuildCommit
}

func TestPrintVersionDefaultValues(t *testing.T) {
	// Сохраняем оригинальный stdout
	oldStdout := os.Stdout

	// Создаем буфер для перехвата вывода
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Сохраняем оригинальные значения
	origBuildVersion := BuildVersion
	origBuildDate := BuildDate
	origBuildCommit := BuildCommit

	// Устанавливаем значения по умолчанию
	BuildVersion = "N/A"
	BuildDate = "N/A"
	BuildCommit = "N/A"

	// Вызываем функцию, которую тестируем
	printVersion()

	// Закрываем writer и восстанавливаем stdout
	w.Close()
	os.Stdout = oldStdout

	// Читаем перехваченный вывод
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("Failed to copy output: %v", err)
	}

	output := buf.String()

	// Проверяем, что вывод содержит ожидаемые строки
	expectedLines := []string{
		"Build version: N/A",
		"Build date: N/A",
		"Build commit: N/A",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("Expected output to contain %q, but got: %q", line, output)
		}
	}

	// Восстанавливаем оригинальные значения
	BuildVersion = origBuildVersion
	BuildDate = origBuildDate
	BuildCommit = origBuildCommit
}
