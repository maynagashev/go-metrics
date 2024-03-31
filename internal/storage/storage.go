package storage

import (
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/storage/memory"
)

// Repository provides an interface for working with metrics storage.
type Repository interface {
	UpdateGauge(metricName string, metricValue float64)
	UpdateCounter(metricName string, metricValue int64)
	GetValue(metricType metrics.MetricType, name string) (string, error)
	GetMetrics() map[string]map[string]string
}

func New() Repository {
	// На данном этапе используется in-memory хранилище.
	return memory.New()
}
