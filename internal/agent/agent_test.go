package agent

import (
	"testing"
	"time"
)

//func TestAgent_sendMetric(t *testing.T) {
//
//	a := New("http://localhost:8080/metrics", 2*time.Second, 10*time.Second)
//
//	type args struct {
//		metricType string
//		name       string
//		value      interface{}
//		pollCount  int64
//	}
//	tests := []struct {
//		name    string
//		args    args
//		wantErr bool
//	}{
//		{
//			name: "send gauge",
//			args: args{
//				metricType: "gauge",
//				name:       "test_gauge",
//				value:      1.1,
//				pollCount:  1,
//			},
//			wantErr: false,
//		},
//		{
//			name: "send counter",
//			args: args{
//				metricType: "counter",
//				name:       "test_counter",
//				value:      1,
//				pollCount:  3,
//			},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if err := a.sendMetric(tt.args.metricType, tt.args.name, tt.args.value, tt.args.pollCount); (err != nil) != tt.wantErr {
//				t.Errorf("sendMetric() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}

func TestAgent_collectRuntimeMetrics(t *testing.T) {
	a := New("http://localhost:8080/metrics", 2*time.Second, 10*time.Second)
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
			if got := a.collectRuntimeMetrics(); len(got) != tt.want {
				t.Errorf("collectRuntimeMetrics() = %v, want %v", len(got), tt.want)
			}
		})
	}
}
