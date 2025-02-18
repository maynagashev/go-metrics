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

// Repository предоставляет интерфейс для работы с хранилищем метрик.
type Repository interface {
	// Close закрывает хранилище метрик.
	Close() error

	// Count возвращает общее количество метрик в хранилище.
	Count() int

	// GetMetrics возвращает все метрики в виде структур.
	GetMetrics() []metrics.Metric

	// GetMetric получает значение метрики указанного типа.
	// Возвращает метрику и флаг, указывающий на её наличие в хранилище.
	GetMetric(mType metrics.MetricType, name string) (metrics.Metric, bool)

	// GetCounter возвращает значение счетчика по имени.
	// Возвращает значение и флаг, указывающий на наличие счетчика.
	GetCounter(name string) (Counter, bool)

	// GetGauge возвращает значение gauge-метрики по имени.
	// Возвращает значение и флаг, указывающий на наличие метрики.
	GetGauge(name string) (Gauge, bool)

	// UpdateMetric обновляет или создает метрику в хранилище.
	// Поддерживает типы gauge и counter.
	UpdateMetric(metric metrics.Metric) error

	// UpdateMetrics пакетно обновляет набор метрик в хранилище.
	UpdateMetrics(metrics []metrics.Metric) error
}
