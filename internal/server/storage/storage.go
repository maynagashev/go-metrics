package storage

import (
	"fmt"
	"strconv"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

type Gauge float64
type Counter int64
type Gauges map[string]Gauge
type Counters map[string]Counter

func (v Gauge) String() string {
	return strconv.FormatFloat(float64(v), 'f', -1, 64)
}
func (v Counter) String() string {
	return strconv.FormatInt(int64(v), 10)
}

// Repository provides an interface for working with metrics storage.
type Repository interface {
	GetMetrics() []string
	GetValue(metricType metrics.MetricType, name string) (fmt.Stringer, bool)
	GetCounter(name string) (Counter, bool)
	GetGauge(name string) (Gauge, bool)
	GetCounters() Counters
	GetGauges() Gauges
	UpdateGauge(metricName string, metricValue Gauge)
	UpdateCounter(metricName string, metricValue Counter)
	UpdateMetric(metric metrics.Metrics) error // универсальный метод обновления метрики: gauge, counter
	Count() int
}
