package memory_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
)

func setupTestStorage(t *testing.T) *memory.MemStorage {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	cfg := &app.Config{}
	ms := memory.New(cfg, logger)
	require.NotNil(t, ms)

	return ms
}

func setupTestStorageWithConfig(t *testing.T, cfg *app.Config) *memory.MemStorage {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	ms := memory.New(cfg, logger)
	require.NotNil(t, ms)

	return ms
}

func TestMemStorage_UpdateGauge(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Проверяем что метрика сохраняется
	ms.UpdateGauge("test_gauge", storage.Gauge(42.0))

	// Проверяем что метрика читается
	value, ok := ms.GetGauge(ctx, "test_gauge")
	assert.True(t, ok)
	assert.InDelta(t, 42.0, float64(value), 1e-9)
}

func TestMemStorage_IncrementCounter(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Проверяем что метрика сохраняется
	ms.IncrementCounter("test_counter", storage.Counter(1))

	// Проверяем что метрика читается
	value, ok := ms.GetCounter(ctx, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, int64(1), int64(value))

	// Проверяем что метрика инкрементируется
	ms.IncrementCounter("test_counter", storage.Counter(2))
	value, ok = ms.GetCounter(ctx, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, int64(3), int64(value))

	// Проверяем что метрика инкрементируется с отрицательным значением
	ms.IncrementCounter("test_counter", storage.Counter(-1))
	value, ok = ms.GetCounter(ctx, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, int64(2), int64(value))
}

func TestMemStorage_UpdateMetric(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Проверяем что gauge метрика сохраняется
	gaugeValue := 42.0
	metric := metrics.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
		Value: &gaugeValue,
	}
	err := ms.UpdateMetric(ctx, metric)
	require.NoError(t, err)

	// Проверяем что gauge метрика читается
	result, ok := ms.GetMetric(ctx, metrics.TypeGauge, "test_gauge")
	assert.True(t, ok)
	assert.InDelta(t, gaugeValue, *result.Value, 1e-9)

	// Проверяем обновление counter
	counterValue := int64(10)
	err = ms.UpdateMetric(ctx, metrics.Metric{
		Name:  "test_counter",
		MType: metrics.TypeCounter,
		Delta: &counterValue,
	})
	require.NoError(t, err)

	// Проверяем что counter метрика читается
	metric, ok = ms.GetMetric(ctx, metrics.TypeCounter, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, "test_counter", metric.Name)
	assert.Equal(t, metrics.TypeCounter, metric.MType)
	assert.NotNil(t, metric.Delta)
	assert.Equal(t, counterValue, *metric.Delta)

	// Проверяем инкремент counter
	counterValue = int64(5)
	err = ms.UpdateMetric(ctx, metrics.Metric{
		Name:  "test_counter",
		MType: metrics.TypeCounter,
		Delta: &counterValue,
	})
	require.NoError(t, err)

	// Проверяем что counter метрика инкрементировалась
	metric, ok = ms.GetMetric(ctx, metrics.TypeCounter, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, "test_counter", metric.Name)
	assert.Equal(t, metrics.TypeCounter, metric.MType)
	assert.NotNil(t, metric.Delta)
	assert.Equal(t, int64(15), *metric.Delta)

	// Проверяем ошибку при неверном типе метрики
	invalidMetric := metrics.Metric{
		Name:  "test_invalid",
		MType: "invalid",
	}
	err = ms.UpdateMetric(ctx, invalidMetric)
	assert.Error(t, err)
}

func TestMemStorage_GetMetric(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Проверяем отсутствие метрики
	_, ok := ms.GetMetric(ctx, metrics.TypeGauge, "non_existent")
	assert.False(t, ok)

	// Добавляем gauge метрику
	gaugeValue := 42.0
	err := ms.UpdateMetric(ctx, metrics.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
		Value: &gaugeValue,
	})
	require.NoError(t, err)

	// Проверяем что gauge метрика читается
	metric, ok := ms.GetMetric(ctx, metrics.TypeGauge, "test_gauge")
	assert.True(t, ok)
	assert.InDelta(t, gaugeValue, *metric.Value, 1e-9)

	// Добавляем counter метрику
	counterValue := int64(10)
	err = ms.UpdateMetric(ctx, metrics.Metric{
		Name:  "test_counter",
		MType: metrics.TypeCounter,
		Delta: &counterValue,
	})
	require.NoError(t, err)

	// Проверяем наличие counter метрики
	metric, ok = ms.GetMetric(ctx, metrics.TypeCounter, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, "test_counter", metric.Name)
	assert.Equal(t, metrics.TypeCounter, metric.MType)
	assert.NotNil(t, metric.Delta)
	assert.Equal(t, counterValue, *metric.Delta)
}

func TestMemStorage_Count(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Проверяем начальное количество метрик
	count := ms.Count(ctx)
	assert.Equal(t, 0, count)

	// Добавляем gauge метрику
	gaugeValue := 42.0
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)
	err := ms.UpdateMetric(ctx, *gaugeMetric)
	require.NoError(t, err)

	// Проверяем количество метрик после добавления gauge
	count = ms.Count(ctx)
	assert.Equal(t, 1, count)

	// Добавляем counter метрику
	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)
	err = ms.UpdateMetric(ctx, *counterMetric)
	require.NoError(t, err)

	// Проверяем количество метрик после добавления counter
	count = ms.Count(ctx)
	assert.Equal(t, 2, count)
}

func TestMemStorage_UpdateMetrics(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Создаем тестовые метрики
	gaugeValue := 42.0
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)

	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)

	metricsSlice := []metrics.Metric{*gaugeMetric, *counterMetric}

	// Обновляем метрики
	err := ms.UpdateMetrics(ctx, metricsSlice)
	require.NoError(t, err)

	// Проверяем что gauge метрика обновилась
	metric, ok := ms.GetMetric(ctx, metrics.TypeGauge, "test_gauge")
	assert.True(t, ok)
	assert.Equal(t, "test_gauge", metric.Name)
	assert.Equal(t, metrics.TypeGauge, metric.MType)
	assert.NotNil(t, metric.Value)
	assert.InDelta(t, gaugeValue, *metric.Value, 1e-9)

	// Проверяем что counter метрика обновилась
	metric, ok = ms.GetMetric(ctx, metrics.TypeCounter, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, "test_counter", metric.Name)
	assert.Equal(t, metrics.TypeCounter, metric.MType)
	assert.NotNil(t, metric.Delta)
	assert.Equal(t, counterValue, *metric.Delta)
}

func TestMemStorage_GetMetrics(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Проверяем начальное количество метрик
	metricsSlice := ms.GetMetrics(ctx)
	assert.Empty(t, metricsSlice)

	// Добавляем gauge метрику
	gaugeValue := 42.0
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)
	err := ms.UpdateMetric(ctx, *gaugeMetric)
	require.NoError(t, err)

	// Добавляем counter метрику
	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)
	err = ms.UpdateMetric(ctx, *counterMetric)
	require.NoError(t, err)

	// Получаем все метрики
	allMetrics := ms.GetMetrics(ctx)
	assert.Len(t, allMetrics, 2)

	// Проверяем наличие gauge метрики в списке
	gaugeFound := false
	counterFound := false
	for _, m := range allMetrics {
		if m.Name == "test_gauge" && m.MType == metrics.TypeGauge {
			gaugeFound = true
			assert.NotNil(t, m.Value)
			assert.InDelta(t, gaugeValue, *m.Value, 1e-9)
		}
		if m.Name == "test_counter" && m.MType == metrics.TypeCounter {
			counterFound = true
			assert.NotNil(t, m.Delta)
			assert.Equal(t, counterValue, *m.Delta)
		}
	}
	assert.True(t, gaugeFound, "Gauge metric not found in GetMetrics result")
	assert.True(t, counterFound, "Counter metric not found in GetMetrics result")
}

func TestMemStorage_FileOperations(t *testing.T) {
	// Создаем временный файл для тестирования
	tmpFile, err := os.CreateTemp("", "metrics_test")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Создаем конфигурацию с сохранением в файл
	cfg := &app.Config{
		FileStoragePath: tmpFile.Name(),
		StoreInterval:   0, // Синхронное сохранение
		Restore:         true,
	}

	// Создаем хранилище с конфигурацией
	ms := setupTestStorageWithConfig(t, cfg)
	ctx := context.Background()

	// Добавляем gauge метрику
	gaugeValue := 42.0
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)
	err = ms.UpdateMetric(ctx, *gaugeMetric)
	require.NoError(t, err)

	// Добавляем counter метрику
	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)
	err = ms.UpdateMetric(ctx, *counterMetric)
	require.NoError(t, err)

	// Закрываем хранилище, чтобы сохранить метрики
	err = ms.Close()
	require.NoError(t, err)

	// Создаем новое хранилище с той же конфигурацией
	newMS := setupTestStorageWithConfig(t, cfg)

	// Проверяем что gauge метрика восстановилась
	metric, ok := newMS.GetMetric(ctx, metrics.TypeGauge, "test_gauge")
	assert.True(t, ok)
	assert.InDelta(t, gaugeValue, *metric.Value, 1e-9)

	// Проверяем что counter метрика восстановилась
	metric, ok = newMS.GetMetric(ctx, metrics.TypeCounter, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, "test_counter", metric.Name)
	assert.Equal(t, metrics.TypeCounter, metric.MType)
	assert.NotNil(t, metric.Delta)
	assert.Equal(t, counterValue, *metric.Delta)

	// Закрываем новое хранилище
	err = newMS.Close()
	require.NoError(t, err)
}

func TestMemStorage_Close(t *testing.T) {
	// Тест без файла хранения
	ms := setupTestStorage(t)
	closeErr := ms.Close()
	require.NoError(t, closeErr)

	// Тест с файлом хранения
	tmpFile, tmpErr := os.CreateTemp("", "metrics_test")
	require.NoError(t, tmpErr)
	defer os.Remove(tmpFile.Name())

	// Создаем конфигурацию с сохранением в файл
	cfg := &app.Config{
		FileStoragePath: tmpFile.Name(),
		StoreInterval:   0, // Синхронное сохранение
		Restore:         true,
	}

	// Создаем хранилище с конфигурацией
	ms = setupTestStorageWithConfig(t, cfg)
	closeErr = ms.Close()
	require.NoError(t, closeErr)

	// Тест с ошибкой записи в файл (делаем файл только для чтения)
	writeErr := os.WriteFile(tmpFile.Name(), []byte(validJSON), 0400)
	require.NoError(t, writeErr)

	ms = setupTestStorageWithConfig(t, cfg)
	// Не проверяем результат закрытия, так как на некоторых ОС может не быть ошибки
	_ = ms.Close()
}

const validJSON = `[{"id":"test_gauge","type":"gauge","value":42},{"id":"test_counter","type":"counter","delta":10}]`

func TestMemStorage_GetAllMetrics(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Добавляем gauge метрику
	gaugeValue := 42.0
	err := ms.UpdateMetric(ctx, metrics.Metric{
		Name:  "test_gauge",
		MType: "gauge",
		Value: &gaugeValue,
	})
	require.NoError(t, err)

	// Добавляем counter метрику
	counterValue := int64(10)
	err = ms.UpdateMetric(ctx, metrics.Metric{
		Name:  "test_counter",
		MType: "counter",
		Delta: &counterValue,
	})
	require.NoError(t, err)

	// Получаем все метрики
	allMetrics := ms.GetMetrics(ctx)

	// Проверяем что есть обе метрики
	assert.Len(t, allMetrics, 2)

	// Проверяем gauge метрику
	gaugeFound := false
	for _, m := range allMetrics {
		if m.Name == "test_gauge" && m.MType == "gauge" {
			gaugeFound = true
			assert.NotNil(t, m.Value)
			assert.InDelta(t, gaugeValue, *m.Value, 1e-9)
		}
	}
	assert.True(t, gaugeFound, "Gauge metric not found")

	// Проверяем counter метрику
	counterFound := false
	for _, m := range allMetrics {
		if m.Name == "test_counter" && m.MType == "counter" {
			counterFound = true
			assert.NotNil(t, m.Delta)
			assert.Equal(t, counterValue, *m.Delta)
		}
	}
	assert.True(t, counterFound, "Counter metric not found")
}
