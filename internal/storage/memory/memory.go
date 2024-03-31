// Package memory provides an in-memory storage for metrics.
package memory

import (
	"fmt"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"strconv"
)

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func New() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (ms *MemStorage) UpdateGauge(metricName string, metricValue float64) {
	ms.gauges[metricName] = metricValue
}

func (ms *MemStorage) UpdateCounter(metricName string, metricValue int64) {
	ms.counters[metricName] += metricValue
}

func (ms *MemStorage) GetValue(metricType metrics.MetricType, name string) (string, error) {
	switch metricType {
	case "counter":
		if counterValue, ok := ms.counters[name]; ok {
			return strconv.FormatInt(counterValue, 10), nil
		} else {
			return "", fmt.Errorf("counter %s not found", name)
		}

	case "gauge":
		if gaugeValue, ok := ms.gauges[name]; ok {
			return strconv.FormatFloat(gaugeValue, 'f', -1, 64), nil
		} else {
			return "", fmt.Errorf("gauge %s not found", name)
		}

	default:
		return "", fmt.Errorf("invalid metric type: %s", metricType)
	}
}

func (ms *MemStorage) GetMetrics() map[string]map[string]string {
	items := make(map[string]map[string]string)
	items["gauge"] = make(map[string]string)
	items["counter"] = make(map[string]string)

	for name, value := range ms.counters {
		items["counter"][name] = strconv.FormatInt(value, 10)
	}
	for name, value := range ms.gauges {
		items["gauge"][name] = strconv.FormatFloat(value, 'f', -1, 64)
	}

	return items
}
