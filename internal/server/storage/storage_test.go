package storage_test

import (
	"reflect"
	"testing"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/storage"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
)

func TestMemoryStorage(t *testing.T) {
	tests := []struct {
		name string
		want storage.Repository
	}{
		{
			name: "memory storage",
			want: memory.New(&app.Config{}, zap.NewNop()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := memory.New(&app.Config{}, zap.NewNop()); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MemoryStorage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGauge_String(t *testing.T) {
	tests := []struct {
		name  string
		value storage.Gauge
		want  string
	}{
		{
			name:  "positive integer",
			value: 42,
			want:  "42",
		},
		{
			name:  "negative integer",
			value: -10,
			want:  "-10",
		},
		{
			name:  "zero",
			value: 0,
			want:  "0",
		},
		{
			name:  "positive float",
			value: 3.14159,
			want:  "3.14159",
		},
		{
			name:  "negative float",
			value: -2.718,
			want:  "-2.718",
		},
		{
			name:  "very large number",
			value: 1234567890.12345,
			want:  "1234567890.12345",
		},
		{
			name:  "very small number",
			value: 0.0000001,
			want:  "0.0000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.want {
				t.Errorf("Gauge.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCounter_String(t *testing.T) {
	tests := []struct {
		name  string
		value storage.Counter
		want  string
	}{
		{
			name:  "positive integer",
			value: 42,
			want:  "42",
		},
		{
			name:  "negative integer",
			value: -10,
			want:  "-10",
		},
		{
			name:  "zero",
			value: 0,
			want:  "0",
		},
		{
			name:  "max int64",
			value: 9223372036854775807,
			want:  "9223372036854775807",
		},
		{
			name:  "min int64",
			value: -9223372036854775808,
			want:  "-9223372036854775808",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.want {
				t.Errorf("Counter.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
