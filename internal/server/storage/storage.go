package storage

import (
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

type Gauge float64
type Counter int64
type Gauges map[string]Gauge
type Counters map[string]Counter

// Repository provides an interface for working with metrics storage.
type Repository interface {
	UpdateGauge(metricName string, metricValue Gauge)
	UpdateCounter(metricName string, metricValue Counter)
	GetValue(metricType metrics.MetricType, name string) (string, error)
	GetMetrics() map[string]map[string]string
	GetCounters() Counters
	GetGauges() Gauges
	GetCounter(name string) (Counter, bool)
	GetGauge(name string) (Gauge, bool)
}
