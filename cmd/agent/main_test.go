package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"crypto/rsa"

	"github.com/maynagashev/go-metrics/internal/agent"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAgent - мок для агента.
type MockAgent struct {
	mock.Mock
}

func (m *MockAgent) Run(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockAgent) IsRequestSigningEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockAgent) IsEncryptionEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockAgent) ResetMetrics() {
	m.Called()
}

func (m *MockAgent) CollectRuntimeMetrics() {
	m.Called()
}

func (m *MockAgent) CollectAdditionalMetrics() {
	m.Called()
}

func (m *MockAgent) GetMetrics() []*metrics.Metric {
	args := m.Called()
	result, _ := args.Get(0).([]*metrics.Metric)
	return result
}

func (m *MockAgent) Shutdown() {
	m.Called()
}

func TestMain(t *testing.T) {
	// Сохраняем оригинальные значения
	originalArgs := os.Args
	originalNew := agent.New
	defer func() {
		os.Args = originalArgs
		agent.New = originalNew
	}()

	// Устанавливаем тестовые аргументы
	os.Args = []string{
		"app",
		"-a", "localhost:9090",
		"-r", "5.0",
		"-p", "2.0",
		"-k", "test-key",
		"-l", "5",
	}

	// Создаем мок для агента
	mockAgent := new(MockAgent)
	mockAgent.On("Run", mock.Anything).Return()

	// Подменяем функцию создания агента
	agent.New = func(
		serverURL string,
		pollInterval time.Duration,
		reportInterval time.Duration,
		privateKey string,
		rateLimit int,
		publicKey *rsa.PublicKey,
	) agent.Agent {
		// Проверяем, что параметры переданы правильно
		assert.Equal(t, "http://localhost:9090", serverURL)
		assert.Equal(t, 2*time.Second, pollInterval)
		assert.Equal(t, 5*time.Second, reportInterval)
		assert.Equal(t, "test-key", privateKey)
		assert.Equal(t, 5, rateLimit)
		assert.Nil(t, publicKey)
		return mockAgent
	}

	// Запускаем main()
	main()

	// Проверяем, что метод Run был вызван
	mockAgent.AssertExpectations(t)
}

func TestInitLogger(t *testing.T) {
	// Сохраняем оригинальный логгер
	originalLogger := slog.Default()
	defer func() {
		slog.SetDefault(originalLogger)
	}()

	// Вызываем тестируемую функцию
	initLogger()

	// Проверяем, что логгер был установлен
	logger := slog.Default()
	assert.NotNil(t, logger)

	// Проверяем, что уровень логирования установлен в Debug
	handler := logger.Handler()
	assert.NotNil(t, handler)

	// Пишем тестовое сообщение и проверяем, что оно записывается
	logger.Debug("test message")
	// Здесь мы не можем напрямую проверить содержимое лога,
	// так как он пишется в os.Stderr, но можем убедиться,
	// что операция не вызывает паники
}

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
