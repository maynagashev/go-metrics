// Package memory provides an in-memory storage for metrics.
package memory

import (
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

// GetMetrics возвращает отсортированный список метрик в формате "тип/имя: значение".
func (ms *MemStorage) GetMetrics() []string {
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