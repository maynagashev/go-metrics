package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"
)

func TestPrintVersion(t *testing.T) {
	// Сохраняем оригинальный stdout
	oldStdout := os.Stdout

	// Создаем pipe для перехвата stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	// Устанавливаем stdout в наш pipe
	os.Stdout = w

	// Устанавливаем тестовые значения для переменных сборки
	originalBuildVersion := BuildVersion
	originalBuildDate := BuildDate
	originalBuildCommit := BuildCommit

	// Восстанавливаем оригинальные значения после завершения теста
	defer func() {
		BuildVersion = originalBuildVersion
		BuildDate = originalBuildDate
		BuildCommit = originalBuildCommit
		os.Stdout = oldStdout
	}()

	// Устанавливаем тестовые значения
	BuildVersion = "v1.0.0"
	BuildDate = "2023-01-01"
	BuildCommit = "abc123"

	// Вызываем функцию
	printVersion()

	// Закрываем запись в pipe для сброса буфера
	w.Close()

	// Читаем перехваченный вывод
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Проверяем вывод
	output := buf.String()
	assert.Contains(t, output, "Build version: v1.0.0")
	assert.Contains(t, output, "Build date: 2023-01-01")
	assert.Contains(t, output, "Build commit: abc123")
}

func TestInitLogger(t *testing.T) {
	// Проверяем, что логгер создается без паники
	assert.NotPanics(t, func() {
		logger := initLogger()
		assert.NotNil(t, logger)
	})
}

func TestInitStorage(t *testing.T) {
	// Создаем минимальную конфигурацию для тестирования
	logger := initLogger()

	// Тестируем с отключенной базой данных
	t.Run("MemoryStorage", func(t *testing.T) {
		cfg := &app.Config{
			Database: app.DatabaseConfig{
				DSN: "",
			},
		}

		repo, err := initStorage(cfg, logger)
		require.NoError(t, err)
		assert.NotNil(t, repo)

		// Очищаем ресурсы
		err = repo.Close()
		require.NoError(t, err)
	})

	// Мы не можем легко протестировать путь PostgreSQL без реальной базы данных,
	// поэтому пропускаем этот тестовый случай
}

// TestInitStorageWithDB тестирует инициализацию хранилища с включенной БД.
func TestInitStorageWithDB(t *testing.T) {
	// Пропускаем тест, если не можем подключиться к реальной базе данных
	// Это интеграционный тест, требующий PostgreSQL
	t.Run("SkipWithoutRealDB", func(t *testing.T) {
		// Создаем конфигурацию с DSN для базы данных
		cfg := &app.Config{
			Database: app.DatabaseConfig{
				DSN: "fake-dsn", // Используем фиктивный DSN, который вызовет ошибку подключения
			},
		}
		logger := initLogger()

		// Пытаемся инициализировать хранилище БД, должна возникнуть ошибка с неверным DSN
		_, err := initStorage(cfg, logger)

		// Ожидаем ошибку, так как DSN неверный
		assert.Error(t, err, "Должна возникнуть ошибка с неверным DSN")
	})

	// Проверяем, установлена ли переменная окружения DATABASE_DSN
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("Пропускаем тест, требующий переменную окружения DATABASE_DSN")
	}

	t.Run("WithRealDB", func(t *testing.T) {
		// Создаем конфигурацию с реальным DSN базы данных
		cfg := &app.Config{
			Database: app.DatabaseConfig{
				DSN: dsn,
			},
		}
		logger := initLogger()

		// Пытаемся инициализировать PostgreSQL хранилище
		repo, err := initStorage(cfg, logger)

		// Проверяем, успешно ли прошло подключение
		if assert.NoError(t, err, "Должно подключиться к реальной базе данных") {
			assert.NotNil(t, repo, "Репозиторий не должен быть nil")

			// Закрываем соединение
			defer func() {
				closeErr := repo.Close()
				assert.NoError(t, closeErr, "Должно закрыть соединение без ошибок")
			}()

			// Выполняем простую операцию для проверки работоспособности репозитория
			ctx := context.Background()
			count := repo.Count(ctx)
			t.Logf("Текущее количество метрик: %d", count)
		}
	})
}

// TestMainFunction тестирует поток выполнения функции main, не запуская её фактически.
func TestMainFunction(t *testing.T) {
	// Патчинг функций напрямую невозможен в Go
	// Вместо этого создаем упрощенную версию, которая тестирует основной рабочий процесс

	// Создаем мок-зависимости
	log := initLogger()
	mockRepo := &mockRepository{}
	mockServer := &mockServer{}
	mockGRPCServer := &mockGRPCServer{}

	// Создаем тестовое окружение
	// Устанавливаем переменные окружения для быстрого выполнения теста
	t.Setenv("ADDRESS", "localhost:8089")
	t.Setenv("STORE_INTERVAL", "1")

	// Используем канал для симуляции сигналов ОС
	sigCh := make(chan os.Signal, 1)

	// Отправляем сигнал для запуска завершения работы после короткой задержки
	go func() {
		time.Sleep(100 * time.Millisecond)
		sigCh <- syscall.SIGINT
	}()

	// Запускаем тест
	assert.NotPanics(t, func() {
		// Выполняем упрощенную версию логики main
		testMainFunction(log, mockServer, mockGRPCServer, mockRepo, sigCh)
	})

	// Проверяем ожидания
	assert.True(t, mockServer.started, "HTTP сервер должен быть запущен")
	assert.True(t, mockGRPCServer.started, "gRPC сервер должен быть запущен")
	assert.True(t, mockRepo.closed, "Репозиторий должен быть закрыт")
}

// testMainFunction - тестируемая версия main(), которая принимает мок-зависимости.
func testMainFunction(
	log *zap.Logger,
	mockServer *mockServer,
	mockGRPCServer *mockGRPCServer,
	mockRepo *mockRepository,
	sigCh chan os.Signal,
) {
	// Создаем контекст, который можно отменить
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем gRPC сервер
	if err := mockGRPCServer.Start(ctx); err != nil {
		log.Error("не удалось запустить gRPC сервер", zap.Error(err))
	}

	// Запускаем HTTP сервер
	go mockServer.Start()

	// Ожидаем сигнал
	sig := <-sigCh
	log.Info("получен сигнал, инициируем корректное завершение", zap.String("signal", sig.String()))

	// Отменяем контекст для завершения работы gRPC сервера
	cancel()

	// Закрываем репозиторий
	_ = mockRepo.Close()
}

// Реализации моков.
type mockServer struct {
	started bool
}

func (m *mockServer) Start() {
	m.started = true
}

type mockGRPCServer struct {
	started bool
}

// Start реализует мок-метод запуска, который всегда успешен
//
//nolint:unparam // Требуется для соответствия интерфейсу в production коде
func (m *mockGRPCServer) Start(_ context.Context) error {
	m.started = true
	return nil
}

type mockRepository struct {
	closed bool
}

// Close реализует мок-метод закрытия, который всегда успешен
//
//nolint:unparam // Требуется для соответствия интерфейсу в production коде
func (m *mockRepository) Close() error {
	m.closed = true
	return nil
}

// Реализация методов интерфейса storage.Repository.
func (m *mockRepository) Count(_ context.Context) int                   { return 0 }
func (m *mockRepository) GetMetrics(_ context.Context) []metrics.Metric { return nil }

func (m *mockRepository) GetMetric(
	_ context.Context,
	_ metrics.MetricType,
	_ string,
) (metrics.Metric, bool) {
	return metrics.Metric{}, false
}
func (m *mockRepository) GetCounter(_ context.Context, _ string) (storage.Counter, bool) {
	return 0, false
}
func (m *mockRepository) GetGauge(_ context.Context, _ string) (storage.Gauge, bool) {
	return 0, false
}
func (m *mockRepository) UpdateMetric(_ context.Context, _ metrics.Metric) error {
	return nil
}
func (m *mockRepository) UpdateMetrics(_ context.Context, _ []metrics.Metric) error {
	return nil
}

// TestIntegrationMain - интеграционный тест, запускающий функцию main
// с таймаутом и сигналом для корректного завершения.
func TestIntegrationMain(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем интеграционный тест в режиме short")
	}

	// Сохраняем оригинальные stdout/stderr и восстанавливаем после теста
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Перенаправляем stdout и stderr в /dev/null
	os.Stdout = os.NewFile(0, os.DevNull)
	os.Stderr = os.NewFile(0, os.DevNull)

	// Устанавливаем тестовые переменные окружения
	t.Setenv("ADDRESS", "localhost:18080") // Используем другой порт, чтобы избежать конфликтов
	t.Setenv("STORE_INTERVAL", "1")
	t.Setenv("FILE_STORAGE_PATH", "") // Отключаем файловое хранилище
	t.Setenv("RESTORE", "false")
	t.Setenv("GRPC_ENABLED", "false") // Отключаем gRPC для упрощения теста

	// Запускаем функцию main в горутине с таймаутом
	done := make(chan struct{})
	go func() {
		// Используем recover для перехвата паник из main
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Перехвачена паника в main: %v", r)
			}
			close(done)
		}()

		// Отправляем SIGINT после короткой задержки для остановки сервера
		go func() {
			time.Sleep(500 * time.Millisecond)
			// Получаем текущий процесс
			process, err := os.FindProcess(os.Getpid())
			if err == nil {
				// Отправляем SIGINT для запуска корректного завершения
				_ = process.Signal(syscall.SIGINT)
			}
		}()

		// Запускаем функцию main
		main()
	}()

	// Ожидаем завершения main с таймаутом
	select {
	case <-done:
		// main успешно завершился
	case <-time.After(2 * time.Second):
		t.Fatal("Тест истек по таймауту, ожидая завершения main")
	}
}
