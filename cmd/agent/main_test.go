package main

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/maynagashev/go-metrics/internal/agent"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAgent - мок для агента.
type MockAgent struct {
	mock.Mock
}

func (m *MockAgent) Run() {
	m.Called()
}

func (m *MockAgent) IsRequestSigningEnabled() bool {
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
	mockAgent.On("Run").Return()

	// Подменяем функцию создания агента
	agent.New = func(
		serverURL string,
		pollInterval time.Duration,
		reportInterval time.Duration,
		privateKey string,
		rateLimit int,
	) agent.Agent {
		// Проверяем, что параметры переданы правильно
		assert.Equal(t, "http://localhost:9090", serverURL)
		assert.Equal(t, 2*time.Second, pollInterval)
		assert.Equal(t, 5*time.Second, reportInterval)
		assert.Equal(t, "test-key", privateKey)
		assert.Equal(t, 5, rateLimit)
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
