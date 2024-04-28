package metrics

import (
	"fmt"
	"strconv"
)

type MetricType string

const (
	TypeCounter MetricType = "counter"
	TypeGauge   MetricType = "gauge"
)

type Metrics struct {
	ID    string     `json:"id"`              // Имя метрики
	MType MetricType `json:"type"`            // Параметр, принимающий значение gauge или counter
	Delta *int64     `json:"delta,omitempty"` // Значение метрики в случае передачи counter
	Value *float64   `json:"value,omitempty"` // Значение метрики в случае передачи gauge
}

func (m *Metrics) String() string {
	if m == nil {
		return "<nil>"
	}

	return fmt.Sprintf("Metrics{ID: %s, Type: %s, Value: %f, Delta: %d}", m.ID, m.MType, *m.Value, *m.Delta)
}

func (m *Metrics) ValueString() string {
	if m == nil {
		return "<nil>"
	}
	switch m.MType {
	case TypeCounter:
		return strconv.FormatInt(*m.Delta, 10)
	case TypeGauge:
		return strconv.FormatFloat(*m.Value, 'f', -1, 64)
	}
	return ""
}
