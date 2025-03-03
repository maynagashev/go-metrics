package metrics

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNewMetric(t *testing.T) {
	// Тест для создания метрики типа counter
	t.Run("Create counter metric", func(t *testing.T) {
		name := "test_counter"
		mType := TypeCounter
		delta := int64(10)
		var value *float64 = nil

		metric := NewMetric(name, mType, &delta, value)

		if metric.Name != name {
			t.Errorf("NewMetric() name = %v, want %v", metric.Name, name)
		}
		if metric.MType != mType {
			t.Errorf("NewMetric() mType = %v, want %v", metric.MType, mType)
		}
		if *metric.Delta != delta {
			t.Errorf("NewMetric() delta = %v, want %v", *metric.Delta, delta)
		}
		if metric.Value != nil {
			t.Errorf("NewMetric() value = %v, want nil", metric.Value)
		}
	})

	// Тест для создания метрики типа gauge
	t.Run("Create gauge metric", func(t *testing.T) {
		name := "test_gauge"
		mType := TypeGauge
		var delta *int64 = nil
		value := 10.5

		metric := NewMetric(name, mType, delta, &value)

		if metric.Name != name {
			t.Errorf("NewMetric() name = %v, want %v", metric.Name, name)
		}
		if metric.MType != mType {
			t.Errorf("NewMetric() mType = %v, want %v", metric.MType, mType)
		}
		if metric.Delta != nil {
			t.Errorf("NewMetric() delta = %v, want nil", metric.Delta)
		}
		if *metric.Value != value {
			t.Errorf("NewMetric() value = %v, want %v", *metric.Value, value)
		}
	})
}

func TestNewCounter(t *testing.T) {
	id := "test_counter"
	delta := int64(10)

	metric := NewCounter(id, delta)

	if metric.Name != id {
		t.Errorf("NewCounter() name = %v, want %v", metric.Name, id)
	}
	if metric.MType != TypeCounter {
		t.Errorf("NewCounter() mType = %v, want %v", metric.MType, TypeCounter)
	}
	if *metric.Delta != delta {
		t.Errorf("NewCounter() delta = %v, want %v", *metric.Delta, delta)
	}
	if metric.Value != nil {
		t.Errorf("NewCounter() value = %v, want nil", metric.Value)
	}
}

func TestNewGauge(t *testing.T) {
	id := "test_gauge"
	value := 10.5

	metric := NewGauge(id, value)

	if metric.Name != id {
		t.Errorf("NewGauge() name = %v, want %v", metric.Name, id)
	}
	if metric.MType != TypeGauge {
		t.Errorf("NewGauge() mType = %v, want %v", metric.MType, TypeGauge)
	}
	if metric.Delta != nil {
		t.Errorf("NewGauge() delta = %v, want nil", metric.Delta)
	}
	if *metric.Value != value {
		t.Errorf("NewGauge() value = %v, want %v", *metric.Value, value)
	}
}

func TestMetric_String(t *testing.T) {
	// Тест для nil метрики
	t.Run("Nil metric", func(t *testing.T) {
		var metric *Metric = nil
		expected := "<nil>"
		if result := metric.String(); result != expected {
			t.Errorf("Metric.String() = %v, want %v", result, expected)
		}
	})

	// Тест для метрики типа counter
	t.Run("Counter metric", func(t *testing.T) {
		delta := int64(10)
		metric := NewCounter("test_counter", delta)
		expected := "Metric{Name: test_counter, Type: counter, Delta: 10}"
		if result := metric.String(); result != expected {
			t.Errorf("Metric.String() = %v, want %v", result, expected)
		}
	})

	// Тест для метрики типа gauge
	t.Run("Gauge metric", func(t *testing.T) {
		value := 10.5
		metric := NewGauge("test_gauge", value)
		expected := "Metric{Name: test_gauge, Type: gauge, Value: 10.500000}"
		if result := metric.String(); result != expected {
			t.Errorf("Metric.String() = %v, want %v", result, expected)
		}
	})

	// Тест для метрики без значений
	t.Run("Metric without values", func(t *testing.T) {
		metric := &Metric{
			Name:  "test_metric",
			MType: TypeGauge,
		}
		expected := "Metric{Name: test_metric, Type: gauge}"
		if result := metric.String(); result != expected {
			t.Errorf("Metric.String() = %v, want %v", result, expected)
		}
	})
}

func TestMetric_ValueString(t *testing.T) {
	// Тест для nil метрики
	t.Run("Nil metric", func(t *testing.T) {
		var metric *Metric = nil
		expected := "<nil>"
		if result := metric.ValueString(); result != expected {
			t.Errorf("Metric.ValueString() = %v, want %v", result, expected)
		}
	})

	// Тест для метрики типа counter
	t.Run("Counter metric", func(t *testing.T) {
		delta := int64(10)
		metric := NewCounter("test_counter", delta)
		expected := "10"
		if result := metric.ValueString(); result != expected {
			t.Errorf("Metric.ValueString() = %v, want %v", result, expected)
		}
	})

	// Тест для метрики типа gauge
	t.Run("Gauge metric", func(t *testing.T) {
		value := 10.5
		metric := NewGauge("test_gauge", value)
		expected := "10.5"
		if result := metric.ValueString(); result != expected {
			t.Errorf("Metric.ValueString() = %v, want %v", result, expected)
		}
	})

	// Тест для метрики с неизвестным типом
	t.Run("Unknown metric type", func(t *testing.T) {
		metric := &Metric{
			Name:  "test_metric",
			MType: "unknown",
		}
		expected := ""
		if result := metric.ValueString(); result != expected {
			t.Errorf("Metric.ValueString() = %v, want %v", result, expected)
		}
	})
}

func TestMetric_ToJSON(t *testing.T) {
	// Тест для метрики типа counter
	t.Run("Counter metric to JSON", func(t *testing.T) {
		delta := int64(10)
		metric := NewCounter("test_counter", delta)

		jsonBytes := metric.ToJSON()

		var decoded Metric
		err := json.Unmarshal(jsonBytes, &decoded)
		if err != nil {
			t.Errorf("Failed to unmarshal JSON: %v", err)
		}

		if !reflect.DeepEqual(metric, &decoded) {
			t.Errorf("Metric.ToJSON() = %v, want %v", decoded, *metric)
		}
	})

	// Тест для метрики типа gauge
	t.Run("Gauge metric to JSON", func(t *testing.T) {
		value := 10.5
		metric := NewGauge("test_gauge", value)

		jsonBytes := metric.ToJSON()

		var decoded Metric
		err := json.Unmarshal(jsonBytes, &decoded)
		if err != nil {
			t.Errorf("Failed to unmarshal JSON: %v", err)
		}

		if !reflect.DeepEqual(metric, &decoded) {
			t.Errorf("Metric.ToJSON() = %v, want %v", decoded, *metric)
		}
	})
}
