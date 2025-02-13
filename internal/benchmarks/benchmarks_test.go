package benchmarks

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"testing"

	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
)

// newTestStorage создает тестовое хранилище в памяти с пустым конфигом
// для измерения производительности операций с метриками без накладных расходов на работу с файлами
func newTestStorage() *memory.MemStorage {
	logger := zap.NewNop()
	cfg := &app.Config{}
	return memory.New(cfg, logger)
}

// BenchmarkStorageOperations измеряет производительность базовых операций с хранилищем:
// - Запись метрики (UpdateMetric)
// - Чтение метрики (GetMetric)
// Это помогает оценить скорость работы in-memory хранилища и выявить потенциальные узкие места
func BenchmarkStorageOperations(b *testing.B) {
	store := newTestStorage()

	metric := metrics.Metric{
		Name:  "TestGauge",
		MType: "gauge",
		Value: ptr(1.23),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := store.UpdateMetric(metric)
		if err != nil {
			b.Fatal(err)
		}
		_, found := store.GetMetric(metric.MType, metric.Name)
		if !found {
			b.Fatal("metric not found")
		}
	}
}

// BenchmarkMetricsSerialization измеряет производительность сериализации метрик в JSON
// Это важно для оценки производительности API сервера при подготовке ответов
// и для агента при подготовке данных для отправки
func BenchmarkMetricsSerialization(b *testing.B) {
	testMetrics := []metrics.Metric{
		{
			Name:  "TestGauge1",
			MType: "gauge",
			Value: ptr(1.23),
		},
		{
			Name:  "TestCounter1",
			MType: "counter",
			Delta: ptr(int64(42)),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(testMetrics)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMetricsCompression измеряет производительность сжатия данных алгоритмом gzip
// Это критично для оценки накладных расходов на сжатие при отправке метрик по сети,
// особенно при больших объемах данных
func BenchmarkMetricsCompression(b *testing.B) {
	data := []byte(`{"metrics":[{"name":"TestGauge1","value":1.23},{"name":"TestCounter1","delta":42}]}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, err := gz.Write(data)
		if err != nil {
			b.Fatal(err)
		}
		err = gz.Close()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHashCalculation измеряет производительность вычисления HMAC-SHA256 хеша
// Это важно для оценки накладных расходов на подпись данных, которая используется
// для обеспечения целостности метрик при передаче между агентом и сервером
func BenchmarkHashCalculation(b *testing.B) {
	data := []byte(`{"name":"TestGauge","type":"gauge","value":1.23}`)
	key := []byte("test_key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := hmac.New(sha256.New, key)
		h.Write(data)
		h.Sum(nil)
	}
}

// ptr - вспомогательная функция для создания указателей на значения
// используется для заполнения полей Value и Delta в структуре Metric
func ptr[T any](v T) *T {
	return &v
}
