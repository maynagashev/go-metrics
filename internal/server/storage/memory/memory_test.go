package memory_test

import (
	"testing"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/storage"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

func TestMemStorage_GetValue(t *testing.T) {
	server := app.New(app.Config{})

	type fields struct {
		gauges   storage.Gauges
		counters storage.Counters
	}
	type args struct {
		metricType metrics.MetricType
		name       string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "get counter",
			fields: fields{
				counters: storage.Counters{
					"test_counter": 1,
				},
				gauges: storage.Gauges{
					"test_gauge": 1.1,
				},
			},
			args: args{
				name:       "test_counter",
				metricType: metrics.TypeCounter,
			},
			want: "1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			ms := memory.New(server, zap.NewNop(), tt.fields.gauges, tt.fields.counters)
			if got, _ := ms.GetValue(tt.args.metricType, tt.args.name); got.String() != tt.want {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemStorage_UpdateCounter(t *testing.T) {
	server := app.New(app.Config{})

	type args struct {
		metricName  string
		metricValue storage.Counter
	}
	tests := []struct {
		name    string
		storage storage.Repository
		args    args
		want    storage.Counter
		times   int
	}{
		{
			name:    "update counter",
			storage: memory.New(server, zap.NewNop()),
			args: args{
				metricName:  "test_counter",
				metricValue: 3,
			},
			times: 2,
			want:  6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range tt.times {
				tt.storage.IncrementCounter(tt.args.metricName, tt.args.metricValue)
			}
			if got, _ := tt.storage.GetCounter(tt.args.metricName); got != tt.want {
				t.Errorf("IncrementCounter() = %v, want %v", got, tt.want)
			}
		})
	}
}
