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

type Metric struct {
	ID    string     `json:"id"`              // Имя метрики
	MType MetricType `json:"type"`            // Параметр, принимающий значение gauge или counter
	Delta *int64     `json:"delta,omitempty"` // Значение метрики в случае передачи counter
	Value *float64   `json:"value,omitempty"` // Значение метрики в случае передачи gauge
}

func (m *Metric) String() string {
	if m == nil {
		return "<nil>"
	}
	if m.Delta != nil {
		return fmt.Sprintf("Metric{ID: %s, Type: %s, Delta: %d}", m.ID, m.MType, *m.Delta)
	}
	if m.Value != nil {
		return fmt.Sprintf("Metric{ID: %s, Type: %s, Value: %f}", m.ID, m.MType, *m.Value)
	}

	return fmt.Sprintf("Metric{ID: %s, Type: %s}", m.ID, m.MType)
}

func (m *Metric) ValueString() string {
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
