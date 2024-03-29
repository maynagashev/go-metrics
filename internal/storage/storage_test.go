package storage

import (
	"reflect"
	"testing"
)

func TestMemoryStorage(t *testing.T) {
	tests := []struct {
		name string
		want Repository
	}{
		{
			name: "memory storage",
			want: MemoryStorage(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MemoryStorage(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MemoryStorage() = %v, want %v", got, tt.want)
			}
		})
	}
}
