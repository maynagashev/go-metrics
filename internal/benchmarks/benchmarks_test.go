package benchmarks_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"compress/gzip"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/middleware/decompress"
	"github.com/maynagashev/go-metrics/internal/server/middleware/decompresspool"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
)

// для измерения производительности операций с метриками без накладных расходов на работу с файлами.
func newTestStorage() *memory.MemStorage {
	logger := zap.NewNop()
	cfg := &app.Config{}
	return memory.New(cfg, logger)
}

// Это помогает оценить скорость работы in-memory хранилища и выявить потенциальные узкие места.
func BenchmarkStorageOperations(b *testing.B) {
	store := newTestStorage()
	ctx := context.Background()

	metric := metrics.Metric{
		Name:  "TestGauge",
		MType: "gauge",
		Value: ptr(1.23),
	}

	b.ResetTimer()
	for range b.N {
		err := store.UpdateMetric(ctx, metric)
		if err != nil {
			b.Fatal(err)
		}
		_, found := store.GetMetric(ctx, metric.MType, metric.Name)
		if !found {
			b.Fatal("metric not found")
		}
	}
}

// и для агента при подготовке данных для отправки.
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
	for range b.N {
		_, err := json.Marshal(testMetrics)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// особенно при больших объемах данных.
func BenchmarkMetricsCompression(b *testing.B) {
	data := []byte(
		`{"metrics":[{"name":"TestGauge1","value":1.23},{"name":"TestCounter1","delta":42}]}`,
	)

	var buf bytes.Buffer
	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		buf.Reset()
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

// для обеспечения целостности метрик при передаче между агентом и сервером.
func BenchmarkHashCalculation(b *testing.B) {
	data := []byte(`{"name":"TestGauge","type":"gauge","value":1.23}`)
	key := []byte("test_key")

	b.ResetTimer()
	for range b.N {
		h := hmac.New(sha256.New, key)
		h.Write(data)
		h.Sum(nil)
	}
}

// используется для заполнения полей Value и Delta в структуре Metric.
func ptr[T any](v T) *T {
	return &v
}

// BenchmarkDecompressMiddleware сравнивает производительность middleware с пулом и без.
func BenchmarkDecompressMiddleware(b *testing.B) {
	testCases := []struct {
		name string
		size int
	}{
		{"small", 100},
		{"medium", 10000},
		{"large", 1000000},
	}

	logger := zap.NewNop()

	// Создаем тестовый обработчик
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			b.Error(err)
		}
	})

	for _, tc := range testCases {
		// Создаем тестовые данные
		testData := make([]byte, tc.size)
		for i := range testData {
			testData[i] = 'a'
		}

		// Сжимаем данные
		var compressedData bytes.Buffer
		gz := gzip.NewWriter(&compressedData)
		_, err := gz.Write(testData)
		if err != nil {
			b.Fatal(err)
		}
		err = gz.Close()
		if err != nil {
			b.Fatal(err)
		}

		b.Run(tc.name+"/standard", func(b *testing.B) {
			middleware := decompress.New(logger)
			h := middleware(handler)

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				req := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewReader(compressedData.Bytes()),
				)
				req.Header.Set("Content-Encoding", "gzip")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, req)
			}
		})

		b.Run(tc.name+"/pool", func(b *testing.B) {
			middleware := decompresspool.New(logger)
			h := middleware(handler)

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				req := httptest.NewRequest(
					http.MethodPost,
					"/",
					bytes.NewReader(compressedData.Bytes()),
				)
				req.Header.Set("Content-Encoding", "gzip")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, req)
			}
		})
	}
}
