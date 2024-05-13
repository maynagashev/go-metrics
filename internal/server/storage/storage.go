package storage

import (
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
	// Close закрывает хранилище метрик.
	Close() error

	// Count возвращает общее количество метрик в хранилище.
	Count() int

	// GetMetrics возвращает все метрики в виде структур.
	GetMetrics() []metrics.Metric

	// GetMetric получение значения метрики указанного типа в виде универсальной структуры.
	GetMetric(mType metrics.MetricType, name string) (metrics.Metric, bool)

	// GetCounter возвращает счетчик по имени.
	GetCounter(name string) (Counter, bool)

	// GetGauge возвращает измерение по имени.
	GetGauge(name string) (Gauge, bool)

	// GetCounters возвращает все счетчики в виде мапы Counters.
	GetCounters() Counters

	// GetGauges возвращает все измерения в виде мапы Gauges.
	GetGauges() Gauges

	// IncrementCounter увеличивает значение счетчика на указанное значение.
	IncrementCounter(metricName string, metricValue Counter)

	// UpdateGauge перезаписывает значения метрики.
	UpdateGauge(metricName string, metricValue Gauge)

	// UpdateMetric универсальный метод обновления метрики: gauge, counter.
	UpdateMetric(metric metrics.Metric) error
}
