package gzip

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestCompress(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "Empty data",
			data:    []byte{},
			wantErr: false,
		},
		{
			name:    "Simple text",
			data:    []byte("Hello, world!"),
			wantErr: false,
		},
		{
			name:    "Large text",
			data:    bytes.Repeat([]byte("Lorem ipsum dolor sit amet "), 100),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сжимаем данные
			compressed, err := Compress(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Проверяем, что данные действительно сжаты
			if len(compressed) > 0 && len(tt.data) > 0 {
				// Для больших данных сжатие должно уменьшить размер
				if tt.name == "Large text" && len(compressed) >= len(tt.data) {
					t.Errorf("Compress() did not reduce data size: original %d, compressed %d", len(tt.data), len(compressed))
				}
			}

			// Проверяем, что данные можно разжать обратно
			reader, err := gzip.NewReader(bytes.NewReader(compressed))
			if err != nil {
				t.Errorf("Failed to create gzip reader: %v", err)
				return
			}
			defer reader.Close()

			decompressed, err := io.ReadAll(reader)
			if err != nil {
				t.Errorf("Failed to decompress data: %v", err)
				return
			}

			// Проверяем, что разжатые данные совпадают с исходными
			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("Decompressed data does not match original: got %v, want %v", decompressed, tt.data)
			}
		})
	}
}
