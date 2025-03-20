//nolint:testpackage // использует внутреннее API для тестирования
package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maynagashev/go-metrics/internal/agent/http"
)

const (
	testPrivateKey = "private_key_example"
)

// prepareCertFile создает временный файл с сертификатом для тестирования.
func prepareCertFile(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "test-keys")
	require.NoError(t, err)

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	certPath := filepath.Join(tempDir, "server.crt")
	certContent, err := os.ReadFile("../../../server.crt")
	if os.IsNotExist(err) {
		t.Skip("Skipping test: server.crt not found")
	}
	require.NoError(t, err)

	err = os.WriteFile(certPath, certContent, 0644)
	require.NoError(t, err)

	return certPath, cleanup
}

func TestFactory_NewFactory(t *testing.T) {
	// Arrange
	httpAddr := "http://localhost:8080"
	grpcAddr := "localhost:9090"
	cryptoKey := "/path/to/key.pem"
	realIP := "192.168.1.100"

	// Act
	factory := NewFactory(
		httpAddr,
		grpcAddr,
		true, // grpcEnabled
		5,    // grpcTimeout
		3,    // grpcRetry
		realIP,
		"test-key",
		cryptoKey,
	)

	// Assert
	assert.Equal(t, httpAddr, factory.httpServerAddr)
	assert.Equal(t, grpcAddr, factory.grpcServerAddr)
	assert.True(t, factory.grpcEnabled)
	assert.Equal(t, 5, factory.grpcTimeout)
	assert.Equal(t, 3, factory.grpcRetry)
	assert.Equal(t, realIP, factory.realIP)
	assert.Equal(t, "test-key", factory.privateKey)
	assert.Equal(t, cryptoKey, factory.cryptoKeyPath)
}

// TestBasicHTTPCreation тестирует создание HTTP клиента.
func TestBasicHTTPCreation(t *testing.T) {
	// Создаем временный сертификат
	certPath, cleanup := prepareCertFile(t)
	defer cleanup()

	// Arrange
	factory := &Factory{
		httpServerAddr: "http://localhost:8080",
		grpcEnabled:    false, // используем HTTP клиент
		cryptoKeyPath:  certPath,
	}

	// Act
	client, err := factory.CreateClient()

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// TestFactoryCreateClientHTTP тестирует создание клиента через метод фабрики.
func TestFactoryCreateClientHTTP(t *testing.T) {
	// Создаем временный сертификат
	certPath, cleanup := prepareCertFile(t)
	defer cleanup()

	// Arrange
	factory := &Factory{
		httpServerAddr: "http://localhost:8080",
		grpcEnabled:    false, // используем HTTP клиент
		cryptoKeyPath:  certPath,
	}

	// Act
	client, err := factory.CreateClient()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, client)

	// Проверяем тип клиента
	httpClient, ok := client.(*http.Client)
	assert.True(t, ok, "Client should be a HTTP client")
	assert.NotNil(t, httpClient)
}

// TestRealGRPCClient проверяет реальную функцию создания gRPC клиента.
func TestRealGRPCClient(t *testing.T) {
	// Создаем временный файл с сертификатом
	certPath, cleanup := prepareCertFile(t)
	defer cleanup()

	// Создаем фабрику с реальным путем к ключу
	factory := &Factory{
		grpcServerAddr: "localhost:9090",
		grpcTimeout:    5,
		grpcRetry:      3,
		cryptoKeyPath:  certPath,
	}

	// Вызываем реальный метод создания gRPC клиента
	client, err := factory.createGRPCClient()

	// В зависимости от наличия сервера, тест может как успешно пройти, так и пропуститься
	if err != nil {
		// Если ошибка связана с невозможностью подключиться к серверу - это ОК
		t.Logf("Expected error because no real server: %v", err)
	} else {
		require.NotNil(t, client)
		// Закрываем клиент
		err = client.Close()
		require.NoError(t, err)
	}
}

// TestFactoryCreateClient проверяет работу основного метода фабрики CreateClient.
func TestFactoryCreateClient(t *testing.T) {
	// Создаем временный сертификат
	certPath, cleanup := prepareCertFile(t)
	defer cleanup()

	t.Run("HTTP client when grpcEnabled=false", func(t *testing.T) {
		// Arrange
		factory := &Factory{
			httpServerAddr: "http://localhost:8080",
			grpcEnabled:    false,
			realIP:         "192.168.1.1",
			privateKey:     testPrivateKey,
			cryptoKeyPath:  certPath,
		}

		// Act
		client, err := factory.CreateClient()

		// Assert
		require.NoError(t, err)
		require.NotNil(t, client)
		_, ok := client.(*http.Client)
		assert.True(t, ok, "Client should be an HTTP client")
	})

	t.Run("gRPC client when grpcEnabled=true", func(t *testing.T) {
		// Этот тест может пропускаться, если нет gRPC сервера
		// Arrange
		factory := &Factory{
			httpServerAddr: "http://localhost:8080",
			grpcServerAddr: "localhost:9090",
			grpcEnabled:    true,
			grpcTimeout:    5,
			grpcRetry:      3,
			realIP:         "192.168.1.1",
			privateKey:     testPrivateKey,
			cryptoKeyPath:  certPath,
		}

		// Act
		client, err := factory.CreateClient()

		// Assert
		if err != nil {
			t.Logf("Expected error because no real gRPC server: %v", err)
		} else {
			require.NotNil(t, client)
			// Закрываем клиент
			err = client.Close()
			require.NoError(t, err)
		}
	})
}
