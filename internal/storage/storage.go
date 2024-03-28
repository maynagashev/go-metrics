package storage

import "strconv"

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
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

func (ms *MemStorage) GetString(metricType string, name string) string {
	switch metricType {
	case "counter":
		return strconv.FormatInt(ms.counters[name], 10)
	case "gauge":
		return strconv.FormatFloat(ms.gauges[name], 'f', -1, 64)
	default:
		return ""
	}
}
