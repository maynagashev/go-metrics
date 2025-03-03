package metrics_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/stretchr/testify/assert"
)

// Helper functions for tests.
func MetricToString(m *metrics.Metric) string {
	if m == nil {
		return "<nil>"
	}
	return m.String()
}

func FloatPtr(v float64) *float64 {
	return &v
}

func Int64Ptr(v int64) *int64 {
	return &v
}

func TestNewMetric(t *testing.T) {
	// Тест для создания метрики типа counter
	t.Run("Create counter metric", func(t *testing.T) {
		name := "test_counter"
		mType := metrics.TypeCounter
		delta := int64(10)
		var value *float64 = nil

		metric := metrics.NewMetric(name, mType, &delta, value)

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
		mType := metrics.TypeGauge
		var delta *int64 = nil
		value := 10.5

		metric := metrics.NewMetric(name, mType, delta, &value)

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
	// Test with a valid delta
	delta := int64(42)
	metric := metrics.NewCounter("test_counter", delta)
	assert.Equal(t, "test_counter", metric.Name)
	assert.Equal(t, metrics.TypeCounter, metric.MType)
	assert.NotNil(t, metric.Delta)
	assert.Equal(t, delta, *metric.Delta)
	assert.Nil(t, metric.Value)

	// Test with zero delta
	metric = metrics.NewCounter("test_counter", 0)
	assert.Equal(t, "test_counter", metric.Name)
	assert.Equal(t, metrics.TypeCounter, metric.MType)
	assert.NotNil(t, metric.Delta)
	assert.Equal(t, int64(0), *metric.Delta)
	assert.Nil(t, metric.Value)
}

func TestNewGauge(t *testing.T) {
	// Test with a valid value
	value := 42.0
	metric := metrics.NewGauge("test_gauge", value)
	assert.Equal(t, "test_gauge", metric.Name)
	assert.Equal(t, metrics.TypeGauge, metric.MType)
	assert.NotNil(t, metric.Value)
	assert.Equal(t, value, *metric.Value)
	assert.Nil(t, metric.Delta)

	// Test with zero value
	metric = metrics.NewGauge("test_gauge", 0.0)
	assert.Equal(t, "test_gauge", metric.Name)
	assert.Equal(t, metrics.TypeGauge, metric.MType)
	assert.NotNil(t, metric.Value)
	assert.Equal(t, 0.0, *metric.Value)
	assert.Nil(t, metric.Delta)
}

func TestMetric_String(t *testing.T) {
	// Test with a gauge metric
	value := 42.0
	gaugeMetric := metrics.NewGauge("test_gauge", value)
	assert.Contains(t, gaugeMetric.String(), "test_gauge")
	assert.Contains(t, gaugeMetric.String(), "gauge")
	assert.Contains(t, gaugeMetric.String(), "42")

	// Test with a counter metric
	delta := int64(42)
	counterMetric := metrics.NewCounter("test_counter", delta)
	assert.Contains(t, counterMetric.String(), "test_counter")
	assert.Contains(t, counterMetric.String(), "counter")
	assert.Contains(t, counterMetric.String(), "42")

	// Test with a nil metric
	var metric *metrics.Metric
	assert.Equal(t, "<nil>", MetricToString(metric))

	// Test with a metric with nil values
	emptyMetric := &metrics.Metric{
		Name:  "empty",
		MType: "unknown",
	}
	assert.Contains(t, emptyMetric.String(), "empty")
	assert.Contains(t, emptyMetric.String(), "unknown")
}

func TestMetric_ValueString(t *testing.T) {
	// Тест для nil метрики
	t.Run("Nil metric", func(t *testing.T) {
		var metric *metrics.Metric = nil
		expected := "<nil>"
		if result := metric.ValueString(); result != expected {
			t.Errorf("Metric.ValueString() = %v, want %v", result, expected)
		}
	})

	// Тест для метрики типа counter
	t.Run("Counter metric", func(t *testing.T) {
		delta := int64(10)
		metric := metrics.NewCounter("test_counter", delta)
		expected := "10"
		if result := metric.ValueString(); result != expected {
			t.Errorf("Metric.ValueString() = %v, want %v", result, expected)
		}
	})

	// Тест для метрики типа gauge
	t.Run("Gauge metric", func(t *testing.T) {
		value := 10.5
		metric := metrics.NewGauge("test_gauge", value)
		expected := "10.5"
		if result := metric.ValueString(); result != expected {
			t.Errorf("Metric.ValueString() = %v, want %v", result, expected)
		}
	})

	// Тест для метрики с неизвестным типом
	t.Run("Unknown metric type", func(t *testing.T) {
		metric := &metrics.Metric{
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
		metric := metrics.NewCounter("test_counter", delta)

		jsonBytes := metric.ToJSON()

		var decoded metrics.Metric
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
		metric := metrics.NewGauge("test_gauge", value)

		jsonBytes := metric.ToJSON()

		var decoded metrics.Metric
		err := json.Unmarshal(jsonBytes, &decoded)
		if err != nil {
			t.Errorf("Failed to unmarshal JSON: %v", err)
		}

		if !reflect.DeepEqual(metric, &decoded) {
			t.Errorf("Metric.ToJSON() = %v, want %v", decoded, *metric)
		}
	})
}

func TestMetricToString(t *testing.T) {
	// Test with nil metric
	var metric *metrics.Metric
	assert.Equal(t, "<nil>", MetricToString(metric))

	// Test with a valid metric
	validMetric := &metrics.Metric{
		Name:  "test_metric",
		MType: "gauge",
		Value: FloatPtr(42.0),
	}
	assert.Contains(t, MetricToString(validMetric), "test_metric")
	assert.Contains(t, MetricToString(validMetric), "gauge")
	assert.Contains(t, MetricToString(validMetric), "42")
}

func TestFloatPtr(t *testing.T) {
	value := 42.0
	ptr := FloatPtr(value)
	assert.NotNil(t, ptr)
	assert.Equal(t, value, *ptr)

	ptr = FloatPtr(0.0)
	assert.NotNil(t, ptr)
	assert.Equal(t, 0.0, *ptr)
}

func TestInt64Ptr(t *testing.T) {
	value := int64(42)
	ptr := Int64Ptr(value)
	assert.NotNil(t, ptr)
	assert.Equal(t, value, *ptr)

	ptr = Int64Ptr(0)
	assert.NotNil(t, ptr)
	assert.Equal(t, int64(0), *ptr)
}
