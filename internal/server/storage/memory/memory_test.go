package memory_test

import (
	"context"
	"testing"

	"github.com/maynagashev/go-metrics/internal/server/storage/memory"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/storage"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

func TestMemStorage_GetMetric(t *testing.T) {
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
			ms := memory.New(&app.Config{}, zap.NewNop(), tt.fields.gauges, tt.fields.counters)
			ctx := context.Background()
			if got, _ := ms.GetMetric(ctx, tt.args.metricType, tt.args.name); got.ValueString() != tt.want {
				t.Errorf("GetMetric() = %v, want %v", got.ValueString(), tt.want)
			}
		})
	}
}

func TestMemStorage_UpdateCounter(t *testing.T) {
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
			storage: memory.New(&app.Config{}, zap.NewNop()),
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
			ctx := context.Background()
			for range tt.times {
				m := metrics.NewCounter(tt.args.metricName, int64(tt.args.metricValue))
				err := tt.storage.UpdateMetric(ctx, *m)
				if err != nil {
					t.Errorf("UpdateCounter() error = %v", err)
					return
				}
			}
			if got, _ := tt.storage.GetCounter(ctx, tt.args.metricName); got != tt.want {
				t.Errorf("IncrementCounter() = %v, want %v", got, tt.want)
			}
		})
	}
}
