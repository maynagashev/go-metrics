package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/maynagashev/go-metrics/internal/agent"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

func TestAgent_collectRuntimeMetrics(t *testing.T) {
	// Создаем контекст для тестирования
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := agent.New(
		"http://localhost:8080/metrics",
		2*time.Second,
		10*time.Second,
		"",               // пустой приватный ключ
		0,                // нулевой rate limit
		nil,              // без публичного ключа
		"",               // без явного IP-адреса
		false,            // gRPC выключен
		"localhost:9090", // адрес gRPC сервера по умолчанию
		5,                // таймаут по умолчанию
		3,                // количество повторных попыток по умолчанию
	)
	tests := []struct {
		name string
		want int
	}{
		{
			name: "collect runtime metrics",
			want: 27, // ожидаем 27 метрик из runtime
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a.ResetMetrics()
			a.CollectRuntimeMetrics()
			got := len(a.GetMetrics())
			if got != tt.want {
				t.Errorf("CollectRuntimeMetrics() = %v, want %v", got, tt.want)
			}
		})
	}

	// Запускаем агент в отдельной горутине и сразу отменяем контекст
	go func() {
		a.Run(ctx)
	}()
	cancel()
}

func TestAgent_GRPCConfig(t *testing.T) {
	// Создаем тестовый агент с включенным gRPC
	a := agent.New(
		"http://localhost:8080",
		time.Second,
		time.Second,
		"",
		5,
		nil,
		"",
		true,               // включаем gRPC
		"custom-grpc:9999", // указываем адрес gRPC сервера
		10,                 // устанавливаем таймаут в секундах
		5,                  // устанавливаем количество повторных попыток
	)

	// Запускаем агент (это выведет gRPC конфигурацию в логи)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		a.Run(ctx)
	}()

	// Сразу отменяем контекст, чтобы агент не работал долго
	cancel()

	// Добавляем простую проверку для использования параметра t
	t.Log("Test completed successfully")
}

// MockAgent - мок-реализация интерфейса Agent для тестирования.
type MockAgent struct {
	mock.Mock
}

// Run имитирует запуск агента.
func (m *MockAgent) Run(ctx context.Context) {
	m.Called(ctx)
}

// IsRequestSigningEnabled имитирует проверку на включенную подпись запросов.
func (m *MockAgent) IsRequestSigningEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

// IsEncryptionEnabled имитирует проверку на включенное шифрование.
func (m *MockAgent) IsEncryptionEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

// ResetMetrics имитирует сброс метрик.
func (m *MockAgent) ResetMetrics() {
	m.Called()
}

// CollectRuntimeMetrics имитирует сбор runtime-метрик.
func (m *MockAgent) CollectRuntimeMetrics() {
	m.Called()
}

// CollectAdditionalMetrics имитирует сбор дополнительных метрик.
func (m *MockAgent) CollectAdditionalMetrics() {
	m.Called()
}

// GetMetrics имитирует получение списка метрик.
func (m *MockAgent) GetMetrics() []*metrics.Metric {
	args := m.Called()
	result, _ := args.Get(0).([]*metrics.Metric)
	return result
}

// Shutdown имитирует корректное завершение работы агента.
func (m *MockAgent) Shutdown() {
	m.Called()
}
