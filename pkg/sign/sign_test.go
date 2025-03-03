package sign

import (
	"testing"
)

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
			result := ComputeHMACSHA256(tt.data, tt.key)
			if len(result) != 64 { // SHA256 хеш в hex формате имеет длину 64 символа
				t.Errorf("ComputeHMACSHA256() returned hash of incorrect length: got %v, want 64", len(result))
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

		gotMAC, err := VerifyHMACSHA256(data, key, expectedMAC)

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
		correctMAC := ComputeHMACSHA256(data, key)

		// Затем проверяем его
		gotMAC, err := VerifyHMACSHA256(data, key, correctMAC)

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
		correctMAC := ComputeHMACSHA256(data, key)

		gotMAC, err := VerifyHMACSHA256(data, key, invalidMAC)

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
