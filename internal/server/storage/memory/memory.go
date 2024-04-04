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

func (ms *MemStorage) GetValue(metricType metrics.MetricType, name string) (string, error) {
	switch metricType {
	case metrics.TypeCounter:
		if counterValue, ok := ms.counters[name]; ok {
			return strconv.FormatInt(int64(counterValue), 10), nil
		} else {
			return "", fmt.Errorf("counter %s not found", name)
		}

	case metrics.TypeGauge:
		if gaugeValue, ok := ms.gauges[name]; ok {
			return strconv.FormatFloat(float64(gaugeValue), 'f', -1, 64), nil
		} else {
			return "", fmt.Errorf("gauge %s not found", name)
		}

	default:
		return "", fmt.Errorf("invalid metric type: %s", metricType)
	}
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
