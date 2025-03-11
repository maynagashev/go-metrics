package sign_test

import (
	"testing"

	"github.com/maynagashev/go-metrics/pkg/sign"
	"github.com/stretchr/testify/assert"
)

func TestCalculateHash(t *testing.T) {
	// Test cases
	tests := []struct {
		name     string
		data     []byte
		key      string
		expected string
	}{
		{
			name:     "Empty data with empty key",
			data:     []byte{},
			key:      "",
			expected: "b613679a0814d9ec772f95d778c35fc5ff1697c493715653c6c712144292c5ad",
		},
		{
			name:     "Data with empty key",
			data:     []byte("test data"),
			key:      "",
			expected: "ed2abf5673fe90f2f5ce861e9a5c80bf9a419df4dcc392f8f603617e7eaa33be",
		},
		{
			name:     "Data with key",
			data:     []byte("test data"),
			key:      "secret",
			expected: "c66d73e3c4354ac8fa8c95dd1f3f79931d723bbc430030329a4de1fcb0993dc3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sign.ComputeHMACSHA256(tt.data, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVerifyHash(t *testing.T) {
	// Test cases
	tests := []struct {
		name    string
		data    []byte
		hash    string
		key     string
		wantErr bool
	}{
		{
			name:    "Valid hash",
			data:    []byte("test data"),
			hash:    "c66d73e3c4354ac8fa8c95dd1f3f79931d723bbc430030329a4de1fcb0993dc3",
			key:     "secret",
			wantErr: false,
		},
		{
			name:    "Invalid hash",
			data:    []byte("test data"),
			hash:    "invalid hash",
			key:     "secret",
			wantErr: true,
		},
		{
			name:    "Empty data with valid hash",
			data:    []byte{},
			hash:    "b613679a0814d9ec772f95d778c35fc5ff1697c493715653c6c712144292c5ad",
			key:     "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sign.VerifyHMACSHA256(tt.data, tt.key, tt.hash)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestComputeHMACSHA256(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		key  string
	}{
		{
			name: "Empty data and key",
			data: []byte{},
			key:  "",
		},
		{
			name: "Simple data with key",
			data: []byte("test data"),
			key:  "secret key",
		},
		{
			name: "Complex data with key",
			data: []byte(`{"id":"test","type":"counter","delta":10}`),
			key:  "my-secret-key-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sign.ComputeHMACSHA256(tt.data, tt.key)
			if len(result) != 64 { // SHA256 хеш в hex формате имеет длину 64 символа
				t.Errorf(
					"ComputeHMACSHA256() returned hash of incorrect length: got %v, want 64",
					len(result),
				)
			}
		})
	}
}

func TestVerifyHMACSHA256(t *testing.T) {
	// Тест для пустого expectedMAC
	t.Run("Empty expectedMAC", func(t *testing.T) {
		data := []byte("test data")
		key := "secret key"
		expectedMAC := ""

		gotMAC, err := sign.VerifyHMACSHA256(data, key, expectedMAC)

		if err != nil {
			t.Errorf("VerifyHMACSHA256() error = %v, wantErr nil", err)
		}

		if gotMAC != "" {
			t.Errorf("VerifyHMACSHA256() gotMAC = %v, want empty string", gotMAC)
		}
	})

	// Тест для валидного MAC
	t.Run("Valid MAC", func(t *testing.T) {
		data := []byte("test data")
		key := "secret key"

		// Сначала вычисляем правильный MAC
		correctMAC := sign.ComputeHMACSHA256(data, key)

		// Затем проверяем его
		gotMAC, err := sign.VerifyHMACSHA256(data, key, correctMAC)

		if err != nil {
			t.Errorf("VerifyHMACSHA256() error = %v, wantErr nil", err)
		}

		if gotMAC != correctMAC {
			t.Errorf("VerifyHMACSHA256() gotMAC = %v, want %v", gotMAC, correctMAC)
		}
	})

	// Тест для невалидного MAC
	t.Run("Invalid MAC", func(t *testing.T) {
		data := []byte("test data")
		key := "secret key"
		invalidMAC := "invalid_mac"

		// Вычисляем правильный MAC для сравнения
		correctMAC := sign.ComputeHMACSHA256(data, key)

		gotMAC, err := sign.VerifyHMACSHA256(data, key, invalidMAC)

		if err == nil {
			t.Errorf("VerifyHMACSHA256() expected error, got nil")
		}

		if err.Error() != "invalid hash in request header" {
			t.Errorf("VerifyHMACSHA256() error = %v, want 'invalid hash in request header'", err)
		}

		if gotMAC != correctMAC {
			t.Errorf("VerifyHMACSHA256() gotMAC = %v, want %v", gotMAC, correctMAC)
		}
	})
}
