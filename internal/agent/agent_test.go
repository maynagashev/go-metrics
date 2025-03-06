package agent_test

import (
	"testing"
	"time"

	"github.com/maynagashev/go-metrics/internal/agent"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/stretchr/testify/mock"
)

func TestAgent_collectRuntimeMetrics(t *testing.T) {
	a := agent.New("http://localhost:8080/metrics", 2*time.Second, 10*time.Second, "", 0)
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
			a.ResetMetrics()
			a.CollectRuntimeMetrics()
			got := len(a.GetMetrics())
			if got != tt.want {
				t.Errorf("CollectRuntimeMetrics() = %v, want %v", got, tt.want)
			}
		})
	}
}

type MockAgent struct {
	mock.Mock
}

func (m *MockAgent) Run() {
	m.Called()
}

func (m *MockAgent) IsRequestSigningEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockAgent) ResetMetrics() {
	m.Called()
}

func (m *MockAgent) CollectRuntimeMetrics() {
	m.Called()
}

func (m *MockAgent) CollectAdditionalMetrics() {
	m.Called()
}

func (m *MockAgent) GetMetrics() []*metrics.Metric {
	args := m.Called()
	result, _ := args.Get(0).([]*metrics.Metric)
	return result
}
