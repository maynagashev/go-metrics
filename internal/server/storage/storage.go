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
	GetMetrics() []metrics.Metric
	GetValue(metricType metrics.MetricType, name string) (fmt.Stringer, bool)
	GetCounter(name string) (Counter, bool)
	GetGauge(name string) (Gauge, bool)
	GetCounters() Counters
	GetGauges() Gauges

	// UpdateGauge перезаписывает значение метрики.
	UpdateGauge(metricName string, metricValue Gauge)

	// IncrementCounter увеличивает значение счетчика на указанное значение.
	IncrementCounter(metricName string, metricValue Counter)

	// UpdateMetric универсальный метод обновления метрики: gauge, counter.
	UpdateMetric(metric metrics.Metric) error

	// Count возвращает общее количество метрик в хранилище.
	Count() int

	// GetMetric получение значения метрики в виде структуры.
	GetMetric(mType metrics.MetricType, id string) (metrics.Metric, bool) //

	// StoreMetricsToFile сохраняет метрики в файл.
	StoreMetricsToFile() error

	// RestoreMetricsFromFile восстанавливает метрики из файла.
	RestoreMetricsFromFile() error
}
