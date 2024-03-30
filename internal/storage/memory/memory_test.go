package memory

import (
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"reflect"
	"testing"
)

func TestMemStorage_GetValue(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
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
				counters: map[string]int64{
					"test_counter": 1,
				},
				gauges: map[string]float64{
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
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}
			if got := ms.GetValue(tt.args.metricType, tt.args.name); got != tt.want {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemStorage_UpdateCounter(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	type args struct {
		metricName  string
		metricValue int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int64
		times  int
	}{
		{
			name: "update counter",
			fields: fields{
				counters: map[string]int64{},
				gauges:   map[string]float64{},
			},
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
			ms := &MemStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}
			for i := 0; i < tt.times; i++ {
				ms.UpdateCounter(tt.args.metricName, tt.args.metricValue)
			}
			if got := ms.counters[tt.args.metricName]; got != tt.want {
				t.Errorf("UpdateCounter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemStorage_UpdateGauge(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	type args struct {
		metricName  string
		metricValue float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}
			ms.UpdateGauge(tt.args.metricName, tt.args.metricValue)
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		want *MemStorage
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}
