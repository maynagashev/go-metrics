package agent_test

import (
	"testing"
	"time"

	"github.com/maynagashev/go-metrics/internal/agent"
)

func TestAgent_collectRuntimeMetrics(t *testing.T) {
	a := agent.New("http://localhost:8080/metrics", 2*time.Second, 10*time.Second)
	tests := []struct {
		name string
		want int
	}{
		{
			name: "collect runtime metrics",
			want: 27,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := a.CollectRuntimeMetrics(); len(got) != tt.want {
				t.Errorf("CollectRuntimeMetrics() = %v, want %v", len(got), tt.want)
			}
		})
	}
}
