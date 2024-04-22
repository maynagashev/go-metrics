package storage_test

import (
	"reflect"
	"testing"

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
			want: memory.New(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := memory.New(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MemoryStorage() = %v, want %v", got, tt.want)
			}
		})
	}
}