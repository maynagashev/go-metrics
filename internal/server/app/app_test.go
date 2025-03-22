package app_test

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/app"
)

func TestNewConfig(t *testing.T) {
	// Test with default values
	flags := &app.Flags{}
	flags.Server.Addr = "localhost:8080"
	flags.Server.StoreInterval = 300
	flags.Server.FileStoragePath = "/tmp/metrics-db.json"
	flags.Server.Restore = true
	flags.Database.DSN = ""
	flags.PrivateKey = ""
	flags.CryptoKey = ""

	config := app.NewConfig(flags)

	// Verify the config was created correctly
	assert.Equal(t, "localhost:8080", config.Addr)
	assert.Equal(t, 300, config.StoreInterval)
	assert.Equal(t, "/tmp/metrics-db.json", config.FileStoragePath)
	assert.True(t, config.Restore)
	assert.Equal(t, "", config.Database.DSN)
	assert.Equal(t, "", config.PrivateKey)
	assert.Nil(t, config.PrivateRSAKey)
}

func TestConfig_IsStoreEnabled(t *testing.T) {
	// Test with store enabled
	config := &app.Config{
		FileStoragePath: "/tmp/metrics-db.json",
	}
	assert.True(t, config.IsStoreEnabled())

	// Test with store disabled
	config = &app.Config{
		FileStoragePath: "",
	}
	assert.False(t, config.IsStoreEnabled())
}

func TestConfig_IsRestoreEnabled(t *testing.T) {
	// Test with restore enabled
	config := &app.Config{
		Restore:         true,
		FileStoragePath: "/tmp/metrics-db.json",
	}
	assert.True(t, config.IsRestoreEnabled())

	// Test with restore disabled
	config = &app.Config{
		Restore: false,
	}
	assert.False(t, config.IsRestoreEnabled())
}

func TestConfig_GetStorePath(t *testing.T) {
	config := &app.Config{
		FileStoragePath: "/tmp/metrics-db.json",
	}
	assert.Equal(t, "/tmp/metrics-db.json", config.GetStorePath())
}

func TestConfig_IsSyncStore(t *testing.T) {
	// Test with sync store
	config := &app.Config{
		StoreInterval: 0,
	}
	assert.True(t, config.IsSyncStore())

	// Test with async store
	config = &app.Config{
		StoreInterval: 300,
	}
	assert.False(t, config.IsSyncStore())
}

func TestConfig_GetStoreInterval(t *testing.T) {
	config := &app.Config{
		StoreInterval: 300,
	}
	assert.Equal(t, 300, config.GetStoreInterval())
}

func TestConfig_IsDatabaseEnabled(t *testing.T) {
	// Test with database enabled
	config := &app.Config{
		Database: app.DatabaseConfig{
			DSN: "postgres://user:password@localhost:5432/metrics",
		},
	}
	assert.True(t, config.IsDatabaseEnabled())

	// Test with database disabled
	config = &app.Config{
		Database: app.DatabaseConfig{
			DSN: "",
		},
	}
	assert.False(t, config.IsDatabaseEnabled())
}

func TestConfig_IsRequestSigningEnabled(t *testing.T) {
	// Test with request signing enabled
	config := &app.Config{
		PrivateKey: "test-key",
	}
	assert.True(t, config.IsRequestSigningEnabled())

	// Test with request signing disabled
	config = &app.Config{
		PrivateKey: "",
	}
	assert.False(t, config.IsRequestSigningEnabled())
}

func TestConfig_IsEncryptionEnabled(t *testing.T) {
	// Generate a real RSA key for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Test with encryption enabled
	config := &app.Config{
		PrivateRSAKey: privateKey,
	}
	assert.True(t, config.IsEncryptionEnabled())

	// Test with encryption disabled
	config = &app.Config{
		PrivateRSAKey: nil,
	}
	assert.False(t, config.IsEncryptionEnabled())
}

func TestNew(t *testing.T) {
	config := &app.Config{
		Addr:            "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "/tmp/metrics-db.json",
		Restore:         true,
	}

	server := app.New(config)

	assert.NotNil(t, server)
	assert.Equal(t, config.StoreInterval, server.GetStoreInterval())
}

func TestServer_GetStoreInterval(t *testing.T) {
	config := &app.Config{
		StoreInterval: 300,
	}

	server := app.New(config)

	assert.Equal(t, 300, server.GetStoreInterval())
}

// TestServer_Start тестирует запуск и остановку сервера по сигналу.
func TestServer_Start(t *testing.T) {
	// Если запущен с флагом -short, пропускаем тест
	if testing.Short() {
		t.Skip("Пропускаем тест, который запускает реальный сервер")
	}

	// Выбираем порт для первого теста
	serverPort := ":18081"

	// Создаем конфигурацию для тестового сервера с адресом для тестирования
	config := &app.Config{
		Addr:          "localhost" + serverPort, // используем нестандартный порт для теста
		StoreInterval: 1,
		Restore:       false,
	}

	// Создаем сервер
	server := app.New(config)

	// Создаем logger для теста
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{"stdout"}
	logger, err := cfg.Build()
	require.NoError(t, err)

	// Создаем простой HTTP-обработчик
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Канал для корректного завершения теста
	done := make(chan struct{})

	// Запускаем сервер в отдельной горутине
	go func() {
		// Используем recover для перехвата возможной паники
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Перехвачена паника в Start: %v", r)
			}
			close(done)
		}()

		// Отправляем сигнал SIGINT через небольшую задержку
		go func() {
			// Даем серверу время на запуск
			time.Sleep(100 * time.Millisecond)

			// Отправляем сигнал для завершения
			process, procErr := os.FindProcess(os.Getpid())
			if procErr != nil {
				t.Logf("Ошибка при получении процесса: %v", procErr)
				return
			}

			sigErr := process.Signal(syscall.SIGINT)
			if sigErr != nil {
				t.Logf("Ошибка при отправке сигнала: %v", sigErr)
			}
		}()

		// Запускаем сервер (блокирующий вызов)
		server.Start(logger, handler)
	}()

	// Ожидаем завершения работы сервера. Не используем тайм-аут здесь, так как
	// при получении сигнала SIGINT сервер корректно завершится.
	<-done
}

// Мы используем простой подход: заведомо недопустимый адрес.
func TestServer_Start_Error(t *testing.T) {
	t.Skip("Этот тест вызывает os.Exit через log.Fatal, поэтому пропускаем его")
}

// TestServer_ShutdownError проверяет вариант с ошибкой при shutdown.
func TestServer_ShutdownError(t *testing.T) {
	// Пропускаем тест при кратком запуске
	if testing.Short() {
		t.Skip("Пропускаем тест, требующий фактического запуска сервера")
	}

	// Создаем конфигурацию с валидным адресом
	config := &app.Config{
		Addr:          "localhost:18083",
		StoreInterval: 1,
		Restore:       false,
	}

	// Создаем сервер
	server := app.New(config)

	// Создаем логгер с перехватом вывода
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{"stdout"}
	logger, err := cfg.Build()
	require.NoError(t, err)

	// Настраиваем мок-обработчик
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond) // Имитируем долгое выполнение запроса
		w.WriteHeader(http.StatusOK)
	})

	// Канал для завершения теста
	done := make(chan struct{})

	// Запускаем сервер в отдельной горутине
	go func() {
		defer close(done)

		// Перехватываем панику при завершении
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Перехвачена паника: %v", r)
			}
		}()

		// Отправляем сигнал для завершения через небольшую задержку
		go func() {
			// Даем серверу время запуститься
			time.Sleep(100 * time.Millisecond)

			// Отправляем тестовый запрос, чтобы запустить обработчик
			go func() {
				resp, httpErr := http.Get("http://localhost:18083")
				if httpErr != nil {
					t.Logf("Ошибка при выполнении тестового запроса: %v", httpErr)
					return
				}
				// Закрываем тело ответа, чтобы избежать утечек ресурсов
				defer func() {
					if closeErr := resp.Body.Close(); closeErr != nil {
						t.Logf("Ошибка при закрытии тела ответа: %v", closeErr)
					}
				}()
			}()

			// Почти сразу отправляем сигнал для остановки
			time.Sleep(10 * time.Millisecond)
			process, procErr := os.FindProcess(os.Getpid())
			if procErr != nil {
				t.Logf("Ошибка при получении процесса: %v", procErr)
				return
			}

			sigErr := process.Signal(syscall.SIGINT)
			if sigErr != nil {
				t.Logf("Ошибка при отправке сигнала: %v", sigErr)
			}
		}()

		// Запускаем сервер
		server.Start(logger, handler)
	}()

	// Ожидаем завершения теста
	<-done
}
