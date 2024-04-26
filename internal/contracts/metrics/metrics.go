package metrics

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
