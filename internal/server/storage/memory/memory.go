// Package memory provides an in-memory storage for metrics.
package memory

import (
	"errors"
	"fmt"
	"slices"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"

	"github.com/maynagashev/go-metrics/internal/server/storage"
)

type MemStorage struct {
	gauges   storage.Gauges
	counters storage.Counters
}

func New(options ...interface{}) *MemStorage {
	memStorage := &MemStorage{
		gauges:   make(storage.Gauges),
		counters: make(storage.Counters),
	}

	for _, option := range options {
		switch opt := option.(type) {
		case storage.Gauges:
			memStorage.gauges = opt
		case storage.Counters:
			memStorage.counters = opt
		}
	}

	return memStorage
}

func (ms *MemStorage) UpdateGauge(metricName string, metricValue storage.Gauge) {
	ms.gauges[metricName] = metricValue
}

func (ms *MemStorage) UpdateCounter(metricName string, metricValue storage.Counter) {
	ms.counters[metricName] += metricValue
}

func (ms *MemStorage) UpdateMetric(metric metrics.Metrics) error {
	switch metric.MType {
	case metrics.TypeGauge:
		if metric.Value == nil {
			return errors.New("gauge value is nil")
		}
		ms.gauges[metric.ID] = storage.Gauge(*metric.Value)
	case metrics.TypeCounter:
		if metric.Delta == nil {
			return errors.New("counter delta is nil")
		}
		ms.counters[metric.ID] += storage.Counter(*metric.Delta)
	default:
		return fmt.Errorf("unsupported metric type: %s", metric.MType)
	}
	return nil
}

func (ms *MemStorage) GetGauges() storage.Gauges {
	return ms.gauges
}

func (ms *MemStorage) GetCounters() storage.Counters {
	return ms.counters
}

func (ms *MemStorage) GetGauge(name string) (storage.Gauge, bool) {
	value, ok := ms.gauges[name]
	return value, ok
}

func (ms *MemStorage) GetCounter(name string) (storage.Counter, bool) {
	value, ok := ms.counters[name]
	return value, ok
}

func (ms *MemStorage) Count() int {
	return len(ms.gauges) + len(ms.counters)
}

func (ms *MemStorage) GetMetric(mType metrics.MetricType, id string) (metrics.Metrics, bool) {
	switch mType {
	case metrics.TypeCounter:
		v, ok := ms.GetCounter(id)
		return metrics.Metrics{
			ID:    id,
			MType: mType,
			Delta: (*int64)(&v),
		}, ok
	case metrics.TypeGauge:
		v, ok := ms.GetGauge(id)
		return metrics.Metrics{
			ID:    id,
			MType: mType,
			Value: (*float64)(&v),
		}, ok
	}
	return metrics.Metrics{}, false
}

// GetValue возвращает значение метрики по типу и имени.
func (ms *MemStorage) GetValue(mType metrics.MetricType, name string) (fmt.Stringer, bool) {
	switch mType {
	case metrics.TypeCounter:
		v, ok := ms.GetCounter(name)
		return v, ok
	case metrics.TypeGauge:
		v, ok := ms.GetGauge(name)
		return v, ok
	}
	return nil, false
}

// GetMetricsPlain возвращает отсортированный список метрик в формате "тип/имя: значение".
func (ms *MemStorage) GetMetricsPlain() []string {
	items := make([]string, 0, ms.Count())
	for name, value := range ms.GetGauges() {
		items = append(items, fmt.Sprintf("counter/%s: %v", name, value))
	}
	for name, value := range ms.GetCounters() {
		items = append(items, fmt.Sprintf("gauge/%s: %v", name, value))
	}
	slices.Sort(items)
	return items
}

// GetMetrics возвращает отсортированный список метрик в формате слайса структур.
func (ms *MemStorage) GetMetrics() []metrics.Metrics {
	items := make([]metrics.Metrics, 0, ms.Count())
	for id, value := range ms.GetGauges() {
		//nolint:gosec // в Go 1.22, значение в цикле копируется (G601: Implicit memory aliasing in for loop.)
		items = append(items, metrics.Metrics{ID: id, MType: metrics.TypeGauge, Value: (*float64)(&value)})
	}
	for id, value := range ms.GetCounters() {
		//nolint:gosec // в Go 1.22, значение в цикле копируется (G601: Implicit memory aliasing in for loop.)
		items = append(items, metrics.Metrics{ID: id, MType: metrics.TypeCounter, Delta: (*int64)(&value)})
	}
	// slices.Sort(items)
	return items
}
