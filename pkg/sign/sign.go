// Package sign предоставляет функции для создания и проверки цифровых подписей.
// Реализует подпись данных с использованием алгоритма HMAC-SHA256.
package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

const HeaderKey = "HashSHA256"

// ComputeHMACSHA256 вычисляет хеш SHA256 от данных с использованием ключа.
func ComputeHMACSHA256(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyHMACSHA256 проверяет, что хеш SHA256 от данных с использованием ключа совпадает с ожидаемым значением.
func VerifyHMACSHA256(data []byte, key string, expectedMAC string) (string, error) {
	// Если хэш не задан, то и не проверяем.
	// Тесты предполагают что с пустым хэшем его не следует проверять, даже если указан приватный ключ -k при старте.
	// См. обсуждение в чате: https://app.pachca.com/chats/8850763?message=245816301
	if expectedMAC == "" {
		return "", nil
	}

	mac := ComputeHMACSHA256(data, key)

	if !hmac.Equal([]byte(mac), []byte(expectedMAC)) {
		return mac, errors.New("invalid hash in request header")
	}

	return mac, nil
}
