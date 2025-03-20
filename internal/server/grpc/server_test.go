//nolint:testpackage // Тесты размещены в том же пакете для доступа к неэкспортируемым функциям
package grpc

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTLSCredentials(t *testing.T) {
	// Сохраняем оригинальную функцию и восстанавливаем её после теста
	originalLoadX509KeyPair := tlsLoadX509KeyPair
	defer func() { tlsLoadX509KeyPair = originalLoadX509KeyPair }()

	// Тест на успешную загрузку
	t.Run("SuccessfulLoad", func(t *testing.T) {
		// Подменяем функцию для тестирования
		tlsLoadX509KeyPair = func(_, _ string) (tls.Certificate, error) {
			return tls.Certificate{}, nil
		}

		// Вызываем функцию loadTLSCredentials
		creds, err := loadTLSCredentials("key.pem")

		// Проверяем, что ошибка отсутствует
		require.NoError(t, err)

		// Проверяем, что результат не nil
		assert.NotNil(t, creds, "Credentials should not be nil")
	})

	// Тест на некорректный путь к ключу
	t.Run("InvalidKeyPath", func(t *testing.T) {
		// Подменяем функцию для тестирования с возвратом ошибки
		tlsLoadX509KeyPair = func(_, _ string) (tls.Certificate, error) {
			return tls.Certificate{}, assert.AnError
		}

		// Вызываем функцию loadTLSCredentials
		creds, err := loadTLSCredentials("invalid.pem")

		// Проверяем наличие ошибки
		require.Error(t, err)

		// Проверяем, что creds равно nil
		assert.Nil(t, creds)

		// Проверяем сообщение об ошибке
		assert.Contains(t, err.Error(), "failed to load server certificate and key")
	})
}
