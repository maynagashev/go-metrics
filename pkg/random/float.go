// Package random предоставляет функции для генерации случайных чисел.
// Использует криптографически стойкий генератор случайных чисел.
package random

import (
	"crypto/rand"
	"math"
	"math/big"
)

// GenerateRandomFloat64 генерирует случайное число типа float64 в диапазоне от 0 до 1.
func GenerateRandomFloat64() float64 {
	// Генерация случайного int64.
	randomInt, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return 0
	}

	// Преобразование int64 в float64 в диапазоне от 0 до 1.
	randomFloat := float64(randomInt.Int64()) / float64(math.MaxInt64)

	return randomFloat
}
