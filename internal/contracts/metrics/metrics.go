package metrics

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type MetricType string

const (
	TypeCounter MetricType = "counter"
	TypeGauge   MetricType = "gauge"
)

type Metric struct {
	Name  string     `json:"id"`              // Имя метрики
	MType MetricType `json:"type"`            // Параметр, принимающий значение gauge или counter
	Delta *int64     `json:"delta,omitempty"` // Значение метрики в случае передачи counter
	Value *float64   `json:"value,omitempty"` // Значение метрики в случае передачи gauge
}

func NewMetric(name string, mType MetricType, delta *int64, value *float64) *Metric {
	return &Metric{
		Name:  name,
		MType: mType,
		Delta: delta,
		Value: value,
	}
}

func NewCounter(id string, delta int64) *Metric {
	return NewMetric(id, TypeCounter, &delta, nil)
}

func NewGauge(id string, value float64) *Metric {
	return NewMetric(id, TypeGauge, nil, &value)
}

func (m *Metric) String() string {
	if m == nil {
		return "<nil>"
	}
	if m.Delta != nil {
		return fmt.Sprintf("Metric{Name: %s, Type: %s, Delta: %d}", m.Name, m.MType, *m.Delta)
	}
	if m.Value != nil {
		return fmt.Sprintf("Metric{Name: %s, Type: %s, Value: %f}", m.Name, m.MType, *m.Value)
	}
	// Значение метрики может быть не задано в структуре,
	// т.к. эта же структура используется для парсинга json в запросе получения значения метрики.
	return fmt.Sprintf("Metric{Name: %s, Type: %s}", m.Name, m.MType)
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

func (m *Metric) ToJSON() []byte {
	encoded, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return encoded
}
