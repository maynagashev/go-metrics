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
