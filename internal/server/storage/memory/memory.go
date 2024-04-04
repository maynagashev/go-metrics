// Package memory provides an in-memory storage for metrics.
package memory

import (
	"fmt"
	"strconv"

	"github.com/maynagashev/go-metrics/internal/server/storage"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
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

func (ms *MemStorage) GetMetrics() map[string]map[string]string {
	items := make(map[string]map[string]string)
	items["gauge"] = make(map[string]string)
	items["counter"] = make(map[string]string)

	for name, value := range ms.counters {
		items["counter"][name] = strconv.FormatInt(int64(value), 10)
	}
	for name, value := range ms.gauges {
		items["gauge"][name] = strconv.FormatFloat(float64(value), 'f', -1, 64)
	}

	return items
}
