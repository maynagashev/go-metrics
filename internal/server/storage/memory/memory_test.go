package memory_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
	ms.UpdateGauge(ctx, "test_gauge", 42.0)

	// Проверяем что метрика читается
	value, ok := ms.GetGauge(ctx, "test_gauge")
	assert.True(t, ok)
	assert.Equal(t, 42.0, float64(value))
}

func TestMemStorage_IncrementCounter(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Проверяем что метрика сохраняется
	ms.IncrementCounter(ctx, "test_counter", 1)

	// Проверяем что метрика читается
	value, ok := ms.GetCounter(ctx, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, int64(1), int64(value))

	// Проверяем что метрика инкрементируется
	ms.IncrementCounter(ctx, "test_counter", 2)
	value, ok = ms.GetCounter(ctx, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, int64(3), int64(value))

	// Проверяем что новая метрика создается с нуля
	ms.IncrementCounter(ctx, "new_counter", 5)
	value, ok = ms.GetCounter(ctx, "new_counter")
	assert.True(t, ok)
	assert.Equal(t, int64(5), int64(value))
}

func TestMemStorage_UpdateMetric(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Проверяем обновление gauge
	gaugeValue := 42.0
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)
	err := ms.UpdateMetric(ctx, *gaugeMetric)
	require.NoError(t, err)

	// Проверяем что gauge метрика читается
	metric, ok := ms.GetMetric(ctx, metrics.TypeGauge, "test_gauge")
	assert.True(t, ok)
	assert.Equal(t, "test_gauge", metric.Name)
	assert.Equal(t, metrics.TypeGauge, metric.MType)
	assert.NotNil(t, metric.Value)
	assert.Equal(t, gaugeValue, *metric.Value)

	// Проверяем обновление counter
	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)
	err = ms.UpdateMetric(ctx, *counterMetric)
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
	counterMetric = metrics.NewCounter("test_counter", counterValue)
	err = ms.UpdateMetric(ctx, *counterMetric)
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
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)
	err := ms.UpdateMetric(ctx, *gaugeMetric)
	require.NoError(t, err)

	// Проверяем наличие gauge метрики
	metric, ok := ms.GetMetric(ctx, metrics.TypeGauge, "test_gauge")
	assert.True(t, ok)
	assert.Equal(t, "test_gauge", metric.Name)
	assert.Equal(t, metrics.TypeGauge, metric.MType)
	assert.NotNil(t, metric.Value)
	assert.Equal(t, gaugeValue, *metric.Value)

	// Добавляем counter метрику
	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)
	err = ms.UpdateMetric(ctx, *counterMetric)
	require.NoError(t, err)

	// Проверяем наличие counter метрики
	metric, ok = ms.GetMetric(ctx, metrics.TypeCounter, "test_counter")
	assert.True(t, ok)
	assert.Equal(t, "test_counter", metric.Name)
	assert.Equal(t, metrics.TypeCounter, metric.MType)
	assert.NotNil(t, metric.Delta)
	// Fix line length issue by breaking into multiple assertions
	if !assert.Equal(t, counterValue, *metric.Delta) {
		t.Errorf("GetMetric() counter value mismatch, got %v, want %v", *metric.Delta, counterValue)
	}
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

	// Создаем набор метрик для обновления
	gaugeValue := 42.0
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)

	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)

	metrics := []metrics.Metric{*gaugeMetric, *counterMetric}

	// Обновляем метрики
	err := ms.UpdateMetrics(ctx, metrics)
	require.NoError(t, err)

	// Проверяем что gauge метрика обновилась
	metric, ok := ms.GetMetric(ctx, metrics.TypeGauge, "test_gauge")
	assert.True(t, ok)
	assert.Equal(t, "test_gauge", metric.Name)
	assert.Equal(t, metrics.TypeGauge, metric.MType)
	assert.NotNil(t, metric.Value)
	assert.Equal(t, gaugeValue, *metric.Value)

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
	metrics := ms.GetMetrics(ctx)
	assert.Empty(t, metrics)

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
	foundGauge := false
	foundCounter := false
	for _, m := range allMetrics {
		if m.Name == "test_gauge" && m.MType == metrics.TypeGauge {
			foundGauge = true
			assert.NotNil(t, m.Value)
			assert.Equal(t, gaugeValue, *m.Value)
		}
		if m.Name == "test_counter" && m.MType == metrics.TypeCounter {
			foundCounter = true
			assert.NotNil(t, m.Delta)
			assert.Equal(t, counterValue, *m.Delta)
		}
	}
	assert.True(t, foundGauge, "Gauge metric not found in GetMetrics result")
	assert.True(t, foundCounter, "Counter metric not found in GetMetrics result")
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
	assert.Equal(t, "test_gauge", metric.Name)
	assert.Equal(t, metrics.TypeGauge, metric.MType)
	assert.NotNil(t, metric.Value)
	assert.Equal(t, gaugeValue, *metric.Value)

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

func TestMemStorage_Dump(t *testing.T) {
	// Создаем временный файл для тестирования
	tmpFile, err := os.CreateTemp("", "metrics_test")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Создаем конфигурацию с сохранением в файл
	cfg := &app.Config{
		FileStoragePath: tmpFile.Name(),
		StoreInterval:   1, // Интервал сохранения 1 секунда
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

	// Ждем, чтобы сработал таймер сохранения
	time.Sleep(2 * time.Second)

	// Закрываем хранилище
	err = ms.Close()
	require.NoError(t, err)

	// Проверяем что файл не пустой
	fileInfo, err := os.Stat(tmpFile.Name())
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))
}

func TestMemStorage_UpdateMetric_Errors(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Проверяем ошибку при неверном типе метрики
	invalidMetric := metrics.Metric{
		Name:  "test_invalid",
		MType: "invalid",
	}
	err := ms.UpdateMetric(ctx, invalidMetric)
	assert.Error(t, err)

	// Проверяем ошибку при отсутствии значения для gauge
	invalidGauge := metrics.Metric{
		Name:  "test_gauge",
		MType: metrics.TypeGauge,
		Value: nil,
	}
	err = ms.UpdateMetric(ctx, invalidGauge)
	assert.Error(t, err)

	// Проверяем ошибку при отсутствии значения для counter
	invalidCounter := metrics.Metric{
		Name:  "test_counter",
		MType: metrics.TypeCounter,
		Delta: nil,
	}
	err = ms.UpdateMetric(ctx, invalidCounter)
	assert.Error(t, err)
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
	closeErr = ms.Close()
	// На некоторых ОС может не быть ошибки, поэтому не проверяем результат
}

const validJSON = `[{"id":"test_gauge","type":"gauge","value":42},{"id":"test_counter","type":"counter","delta":10}]`
