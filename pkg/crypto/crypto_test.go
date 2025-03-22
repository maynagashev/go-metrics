package crypto_test

import (
	"os"
	"testing"

	"github.com/maynagashev/go-metrics/pkg/crypto"
)

func TestGenerateAndLoadKeys(t *testing.T) {
	// Создаем временные файлы для ключей
	privateKeyPath := "test_server.key"
	publicKeyPath := "test_server.crt"

	// Очищаем после теста
	defer os.Remove(privateKeyPath)
	defer os.Remove(publicKeyPath)

	// Генерируем пару ключей
	err := crypto.GenerateKeyPair(privateKeyPath, publicKeyPath, 2048)
	if err != nil {
		t.Fatalf("Не удалось сгенерировать пару ключей: %v", err)
	}

	// Загружаем закрытый ключ
	privateKey, err := crypto.LoadPrivateKey(privateKeyPath)
	if err != nil {
		t.Fatalf("Не удалось загрузить закрытый ключ: %v", err)
	}

	// Загружаем открытый ключ
	publicKey, err := crypto.LoadPublicKey(publicKeyPath)
	if err != nil {
		t.Fatalf("Не удалось загрузить открытый ключ: %v", err)
	}

	// Тестируем шифрование и расшифровку
	testData := []byte("Это тестовое сообщение для шифрования и расшифровки")

	// Шифруем с помощью открытого ключа
	encrypted, err := crypto.Encrypt(publicKey, testData)
	if err != nil {
		t.Fatalf("Не удалось зашифровать данные: %v", err)
	}

	// Расшифровываем с помощью закрытого ключа
	decrypted, err := crypto.Decrypt(privateKey, encrypted)
	if err != nil {
		t.Fatalf("Не удалось расшифровать данные: %v", err)
	}

	// Проверяем, что расшифрованные данные совпадают с исходными
	if string(decrypted) != string(testData) {
		t.Errorf(
			"Расшифрованные данные не совпадают с исходными. Получено: %s, Ожидалось: %s",
			decrypted,
			testData,
		)
	}
}

func TestEncryptLargeData(t *testing.T) {
	// Создаем временные файлы для ключей
	privateKeyPath := "test_server.key"
	publicKeyPath := "test_server.crt"

	// Очищаем после теста
	defer os.Remove(privateKeyPath)
	defer os.Remove(publicKeyPath)

	// Генерируем пару ключей
	err := crypto.GenerateKeyPair(privateKeyPath, publicKeyPath, 2048)
	if err != nil {
		t.Fatalf("Не удалось сгенерировать пару ключей: %v", err)
	}

	// Загружаем закрытый ключ
	privateKey, err := crypto.LoadPrivateKey(privateKeyPath)
	if err != nil {
		t.Fatalf("Не удалось загрузить закрытый ключ: %v", err)
	}

	// Загружаем открытый ключ
	publicKey, err := crypto.LoadPublicKey(publicKeyPath)
	if err != nil {
		t.Fatalf("Не удалось загрузить открытый ключ: %v", err)
	}

	// Создаем большие данные (больше, чем можно зашифровать за один раз с помощью RSA)
	// Для ключа 2048 бит максимальный размер данных около 190 байт
	largeData := make([]byte, 1000) // 1000 байт
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// Шифруем большие данные
	encrypted, err := crypto.EncryptLargeData(publicKey, largeData)
	if err != nil {
		t.Fatalf("Не удалось зашифровать большие данные: %v", err)
	}

	// Расшифровываем большие данные
	decrypted, err := crypto.DecryptLargeData(privateKey, encrypted)
	if err != nil {
		t.Fatalf("Не удалось расшифровать большие данные: %v", err)
	}

	// Проверяем, что расшифрованные данные имеют правильную длину
	if len(decrypted) != len(largeData) {
		t.Errorf(
			"Длина расшифрованных данных не совпадает с исходной. "+
				"Получено: %d, Ожидалось: %d",
			len(decrypted),
			len(largeData),
		)
	}

	// Проверяем, что расшифрованные данные совпадают с исходными
	for i := range largeData {
		if decrypted[i] != largeData[i] {
			t.Errorf(
				"Расшифрованные данные не совпадают с исходными в позиции %d. "+
					"Получено: %d, Ожидалось: %d",
				i,
				decrypted[i],
				largeData[i],
			)
			break
		}
	}
}
