package random

import (
	"testing"
)

func TestGenerateRandomFloat64(t *testing.T) {
	// Тест на генерацию случайного числа в диапазоне от 0 до 1
	for i := 0; i < 100; i++ {
		randomFloat := GenerateRandomFloat64()

		// Проверяем, что число находится в диапазоне от 0 до 1
		if randomFloat < 0 || randomFloat > 1 {
			t.Errorf("GenerateRandomFloat64() = %v, want value in range [0, 1]", randomFloat)
		}
	}

	// Тест на уникальность генерируемых чисел
	// Генерируем 10 случайных чисел и проверяем, что они не все одинаковые
	var values []float64
	for i := 0; i < 10; i++ {
		values = append(values, GenerateRandomFloat64())
	}

	allSame := true
	for i := 1; i < len(values); i++ {
		if values[i] != values[0] {
			allSame = false
			break
		}
	}

	if allSame {
		t.Errorf("GenerateRandomFloat64() generated identical values: %v", values[0])
	}
}
