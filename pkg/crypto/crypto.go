// Package crypto предоставляет функции для асимметричного шифрования и расшифровки.
// Он использует RSA шифрование для безопасной связи между агентом и сервером.
package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

const (
	// RSAOAEPPadding - количество байт, используемых для паддинга в RSA-OAEP.
	RSAOAEPPadding = 2

	// CertValidityYears - срок действия сертификата в годах.
	CertValidityYears = 10

	// LoopbackIPv4First - первый октет для 127.0.0.1.
	LoopbackIPv4First = 127

	// LoopbackIPv4Second - второй октет для 127.0.0.1.
	LoopbackIPv4Second = 0

	// LoopbackIPv4Third - третий октет для 127.0.0.1.
	LoopbackIPv4Third = 0

	// LoopbackIPv4Fourth - четвертый октет для 127.0.0.1.
	LoopbackIPv4Fourth = 1
)

// LoadPublicKey загружает открытый ключ RSA из файла.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл открытого ключа: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("не удалось декодировать PEM блок")
	}

	// Обрабатываем оба формата: сертификат и открытый ключ
	switch block.Type {
	case "CERTIFICATE":
		cert, certErr := x509.ParseCertificate(block.Bytes)
		if certErr != nil {
			return nil, fmt.Errorf("не удалось разобрать сертификат: %w", certErr)
		}

		rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("сертификат не содержит открытый ключ RSA")
		}

		return rsaPub, nil

	case "PUBLIC KEY":
		pub, pubErr := x509.ParsePKIXPublicKey(block.Bytes)
		if pubErr != nil {
			return nil, fmt.Errorf("не удалось разобрать открытый ключ: %w", pubErr)
		}

		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("ключ не является открытым ключом RSA")
		}

		return rsaPub, nil

	default:
		return nil, fmt.Errorf("неподдерживаемый тип ключа: %s", block.Type)
	}
}

// LoadPrivateKey загружает закрытый ключ RSA из файла.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл закрытого ключа: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("не удалось декодировать PEM блок")
	}

	// Обрабатываем различные форматы закрытого ключа
	switch block.Type {
	case "RSA PRIVATE KEY":
		// Формат PKCS#1
		return x509.ParsePKCS1PrivateKey(block.Bytes)

	case "PRIVATE KEY":
		// Формат PKCS#8
		priv, privErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if privErr != nil {
			return nil, fmt.Errorf("не удалось разобрать закрытый ключ PKCS#8: %w", privErr)
		}

		rsaPriv, ok := priv.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("ключ не является закрытым ключом RSA")
		}

		return rsaPriv, nil

	default:
		return nil, fmt.Errorf("неподдерживаемый тип ключа: %s", block.Type)
	}
}

// Encrypt шифрует данные с использованием RSA-OAEP с SHA-256.
// Примечание: RSA может шифровать только небольшие объемы данных.
// Для ключа 2048 бит максимальный размер данных составляет около 190 байт.
func Encrypt(publicKey *rsa.PublicKey, data []byte) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, data, nil)
}

// Decrypt расшифровывает данные с использованием RSA-OAEP с SHA-256.
func Decrypt(privateKey *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, ciphertext, nil)
}

// Формат: [количество частей (4 байта)][размер части 1 (4 байта)][часть 1]...[размер части N (4 байта)][часть N].
func EncryptLargeData(publicKey *rsa.PublicKey, data []byte) ([]byte, error) {
	// Определяем максимальный размер данных, которые можно зашифровать за один раз
	// Для RSA-OAEP с SHA-256 это (размер ключа в байтах) - 2 * (размер хеша в байтах) - 2
	maxChunkSize := (publicKey.Size() - 2*sha256.Size - RSAOAEPPadding)

	// Разбиваем данные на части
	var chunks [][]byte
	for i := 0; i < len(data); i += maxChunkSize {
		end := i + maxChunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}

	// Шифруем каждую часть
	var encryptedChunks [][]byte
	for _, chunk := range chunks {
		encryptedChunk, err := Encrypt(publicKey, chunk)
		if err != nil {
			return nil, fmt.Errorf("ошибка при шифровании части данных: %w", err)
		}
		encryptedChunks = append(encryptedChunks, encryptedChunk)
	}

	// Формируем результат
	var result bytes.Buffer

	// Проверяем, что количество частей не превышает максимальное значение uint32
	if len(encryptedChunks) > int(^uint32(0)) {
		return nil, fmt.Errorf(
			"слишком много частей данных для шифрования: %d",
			len(encryptedChunks),
		)
	}

	// Записываем количество частей (4 байта)
	// #nosec G115 - мы проверили выше, что len(encryptedChunks) не превышает максимальное значение uint32
	numChunks := uint32(len(encryptedChunks))
	if err := binary.Write(&result, binary.BigEndian, numChunks); err != nil {
		return nil, fmt.Errorf("ошибка при записи количества частей: %w", err)
	}

	// Записываем каждую зашифрованную часть с её размером
	for _, chunk := range encryptedChunks {
		// Проверяем, что размер части не превышает максимальное значение uint32
		if len(chunk) > int(^uint32(0)) {
			return nil, fmt.Errorf("размер части данных слишком велик: %d", len(chunk))
		}

		// Записываем размер части (4 байта)
		// #nosec G115 - мы проверили выше, что len(chunk) не превышает максимальное значение uint32
		chunkSize := uint32(len(chunk))
		if err := binary.Write(&result, binary.BigEndian, chunkSize); err != nil {
			return nil, fmt.Errorf("ошибка при записи размера части: %w", err)
		}

		// Записываем саму часть
		if _, err := result.Write(chunk); err != nil {
			return nil, fmt.Errorf("ошибка при записи части: %w", err)
		}
	}

	return result.Bytes(), nil
}

// DecryptLargeData расшифровывает данные, которые были зашифрованы с помощью EncryptLargeData.
func DecryptLargeData(privateKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	buffer := bytes.NewReader(data)

	// Читаем количество частей
	var numChunks uint32
	if err := binary.Read(buffer, binary.BigEndian, &numChunks); err != nil {
		return nil, fmt.Errorf("ошибка при чтении количества частей: %w", err)
	}

	// Читаем и расшифровываем каждую часть
	var result bytes.Buffer
	for i := range numChunks {
		// Читаем размер части
		var chunkSize uint32
		if err := binary.Read(buffer, binary.BigEndian, &chunkSize); err != nil {
			return nil, fmt.Errorf("ошибка при чтении размера части %d: %w", i, err)
		}

		// Читаем саму часть
		chunk := make([]byte, chunkSize)
		if _, err := buffer.Read(chunk); err != nil {
			return nil, fmt.Errorf("ошибка при чтении части %d: %w", i, err)
		}

		// Расшифровываем часть
		decryptedChunk, err := Decrypt(privateKey, chunk)
		if err != nil {
			return nil, fmt.Errorf("ошибка при расшифровке части %d: %w", i, err)
		}

		// Добавляем расшифрованную часть к результату
		if _, writeErr := result.Write(decryptedChunk); writeErr != nil {
			return nil, fmt.Errorf("ошибка при записи расшифрованной части %d: %w", i, writeErr)
		}
	}

	return result.Bytes(), nil
}

// GenerateKeyPair генерирует новую пару ключей RSA и сохраняет их в файлы.
// Также генерирует сертификат X.509 для открытого ключа.
func GenerateKeyPair(privateKeyPath, publicKeyPath string, bits int) error {
	// Создаем шаблон сертификата
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"go-metrics"},
			Country:      []string{"RU"},
		},
		IPAddresses: []net.IP{
			net.IPv4(LoopbackIPv4First, LoopbackIPv4Second, LoopbackIPv4Third, LoopbackIPv4Fourth),
			net.IPv6loopback,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(CertValidityYears, 0, 0), // Действителен 10 лет
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// Генерируем новую пару ключей RSA
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return fmt.Errorf("не удалось сгенерировать пару ключей RSA: %w", err)
	}

	// Создаем сертификат
	certBytes, err := x509.CreateCertificate(
		rand.Reader,
		cert,
		cert,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		return fmt.Errorf("не удалось создать сертификат: %w", err)
	}

	// Сохраняем закрытый ключ в файл (формат PKCS#1)
	privateKeyFile, err := os.Create(privateKeyPath)
	if err != nil {
		return fmt.Errorf("не удалось создать файл закрытого ключа: %w", err)
	}
	defer privateKeyFile.Close()

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	if encodeErr := pem.Encode(privateKeyFile, privateKeyPEM); encodeErr != nil {
		return fmt.Errorf("не удалось записать закрытый ключ в файл: %w", encodeErr)
	}

	// Сохраняем открытый ключ в файл
	publicKeyFile, err := os.Create(publicKeyPath)
	if err != nil {
		return fmt.Errorf("не удалось создать файл открытого ключа: %w", err)
	}
	defer publicKeyFile.Close()

	// Используем сертификат как открытый ключ
	publicKeyPEM := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}

	if encodeErr := pem.Encode(publicKeyFile, publicKeyPEM); encodeErr != nil {
		return fmt.Errorf("не удалось записать открытый ключ в файл: %w", encodeErr)
	}

	return nil
}
