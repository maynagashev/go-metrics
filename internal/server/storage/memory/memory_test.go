package memory

import (
	"context"
	"os"
	"testing"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"go.uber.org/zap"
)

func setupTestStorage(t *testing.T) *MemStorage {
	t.Helper()

	// Создаем тестовый логгер
	logger, _ := zap.NewDevelopment()

	// Создаем тестовый конфиг
	cfg := &app.Config{}

	// Создаем тестовое хранилище
	return New(cfg, logger)
}

// setupTestStorageWithConfig создает тестовое хранилище с заданным конфигом
func setupTestStorageWithConfig(t *testing.T, cfg *app.Config) *MemStorage {
	t.Helper()

	// Создаем тестовый логгер
	logger, _ := zap.NewDevelopment()

	// Создаем тестовое хранилище
	return New(cfg, logger)
}

func TestMemStorage_UpdateGauge(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Обновляем метрику
	metricName := "test_gauge"
	metricValue := storage.Gauge(10.5)
	ms.UpdateGauge(metricName, metricValue)

	// Проверяем, что метрика была обновлена
	value, exists := ms.GetGauge(ctx, metricName)
	if !exists {
		t.Errorf("UpdateGauge() failed to update gauge metric")
	}
	if value != metricValue {
		t.Errorf("UpdateGauge() = %v, want %v", value, metricValue)
	}
}

func TestMemStorage_IncrementCounter(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Инкрементируем метрику
	metricName := "test_counter"
	metricValue := storage.Counter(10)
	ms.IncrementCounter(metricName, metricValue)

	// Проверяем, что метрика была инкрементирована
	value, exists := ms.GetCounter(ctx, metricName)
	if !exists {
		t.Errorf("IncrementCounter() failed to increment counter metric")
	}
	if value != metricValue {
		t.Errorf("IncrementCounter() = %v, want %v", value, metricValue)
	}

	// Инкрементируем еще раз
	ms.IncrementCounter(metricName, metricValue)

	// Проверяем, что значение увеличилось
	value, exists = ms.GetCounter(ctx, metricName)
	if !exists {
		t.Errorf("IncrementCounter() failed to increment counter metric")
	}
	if value != metricValue*2 {
		t.Errorf("IncrementCounter() = %v, want %v", value, metricValue*2)
	}
}

func TestMemStorage_UpdateMetric(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Тест для метрики типа gauge
	t.Run("Update gauge metric", func(t *testing.T) {
		value := 10.5
		metric := metrics.NewGauge("test_gauge", value)

		err := ms.UpdateMetric(ctx, *metric)
		if err != nil {
			t.Errorf("UpdateMetric() error = %v", err)
		}

		// Проверяем, что метрика была обновлена
		storedValue, exists := ms.GetGauge(ctx, metric.Name)
		if !exists {
			t.Errorf("UpdateMetric() failed to update gauge metric")
		}
		if storedValue != storage.Gauge(value) {
			t.Errorf("UpdateMetric() = %v, want %v", storedValue, value)
		}
	})

	// Тест для метрики типа counter
	t.Run("Update counter metric", func(t *testing.T) {
		delta := int64(10)
		metric := metrics.NewCounter("test_counter", delta)

		err := ms.UpdateMetric(ctx, *metric)
		if err != nil {
			t.Errorf("UpdateMetric() error = %v", err)
		}

		// Проверяем, что метрика была обновлена
		storedValue, exists := ms.GetCounter(ctx, metric.Name)
		if !exists {
			t.Errorf("UpdateMetric() failed to update counter metric")
		}
		if storedValue != storage.Counter(delta) {
			t.Errorf("UpdateMetric() = %v, want %v", storedValue, delta)
		}

		// Обновляем еще раз
		err = ms.UpdateMetric(ctx, *metric)
		if err != nil {
			t.Errorf("UpdateMetric() error = %v", err)
		}

		// Проверяем, что значение увеличилось
		storedValue, exists = ms.GetCounter(ctx, metric.Name)
		if !exists {
			t.Errorf("UpdateMetric() failed to update counter metric")
		}
		if storedValue != storage.Counter(delta*2) {
			t.Errorf("UpdateMetric() = %v, want %v", storedValue, delta*2)
		}
	})

	// Тест для метрики с неизвестным типом
	t.Run("Update metric with unknown type", func(t *testing.T) {
		metric := metrics.Metric{
			Name:  "test_unknown",
			MType: "unknown",
		}

		err := ms.UpdateMetric(ctx, metric)
		if err == nil {
			t.Errorf("UpdateMetric() expected error for unknown metric type")
		}
	})
}

func TestMemStorage_GetMetric(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Добавляем тестовые метрики
	gaugeValue := 10.5
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)
	ms.UpdateMetric(ctx, *gaugeMetric)

	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)
	ms.UpdateMetric(ctx, *counterMetric)

	// Тест для получения метрики типа gauge
	t.Run("Get gauge metric", func(t *testing.T) {
		metric, exists := ms.GetMetric(ctx, metrics.TypeGauge, "test_gauge")
		if !exists {
			t.Errorf("GetMetric() failed to get gauge metric")
		}
		if metric.Name != "test_gauge" || metric.MType != metrics.TypeGauge || *metric.Value != gaugeValue {
			t.Errorf("GetMetric() = %v, want name=%s, type=%s, value=%v", metric, "test_gauge", metrics.TypeGauge, gaugeValue)
		}
	})

	// Тест для получения метрики типа counter
	t.Run("Get counter metric", func(t *testing.T) {
		metric, exists := ms.GetMetric(ctx, metrics.TypeCounter, "test_counter")
		if !exists {
			t.Errorf("GetMetric() failed to get counter metric")
		}
		if metric.Name != "test_counter" || metric.MType != metrics.TypeCounter || *metric.Delta != counterValue {
			t.Errorf("GetMetric() = %v, want name=%s, type=%s, delta=%v", metric, "test_counter", metrics.TypeCounter, counterValue)
		}
	})

	// Тест для получения несуществующей метрики
	t.Run("Get non-existent metric", func(t *testing.T) {
		_, exists := ms.GetMetric(ctx, metrics.TypeGauge, "non_existent")
		if exists {
			t.Errorf("GetMetric() expected non-existent metric to not exist")
		}
	})
}

func TestMemStorage_Count(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Проверяем начальное количество метрик
	count := ms.Count(ctx)
	if count != 0 {
		t.Errorf("Count() = %v, want %v", count, 0)
	}

	// Добавляем метрики
	ms.UpdateGauge("gauge1", 1.0)
	ms.UpdateGauge("gauge2", 2.0)
	ms.IncrementCounter("counter1", 1)

	// Проверяем количество метрик
	count = ms.Count(ctx)
	if count != 3 {
		t.Errorf("Count() = %v, want %v", count, 3)
	}
}

func TestMemStorage_UpdateMetrics(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Создаем набор метрик для обновления
	gaugeValue := 10.5
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)

	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)

	// Обновляем метрики пакетно
	err := ms.UpdateMetrics(ctx, []metrics.Metric{*gaugeMetric, *counterMetric})
	if err != nil {
		t.Errorf("UpdateMetrics() error = %v", err)
	}

	// Проверяем, что метрики были обновлены
	storedGauge, existsGauge := ms.GetGauge(ctx, gaugeMetric.Name)
	if !existsGauge {
		t.Errorf("UpdateMetrics() failed to update gauge metric")
	}
	if storedGauge != storage.Gauge(gaugeValue) {
		t.Errorf("UpdateMetrics() gauge = %v, want %v", storedGauge, gaugeValue)
	}

	storedCounter, existsCounter := ms.GetCounter(ctx, counterMetric.Name)
	if !existsCounter {
		t.Errorf("UpdateMetrics() failed to update counter metric")
	}
	if storedCounter != storage.Counter(counterValue) {
		t.Errorf("UpdateMetrics() counter = %v, want %v", storedCounter, counterValue)
	}

	// Тест с ошибкой в одной из метрик
	invalidMetric := metrics.Metric{
		Name:  "invalid",
		MType: "unknown",
	}

	err = ms.UpdateMetrics(ctx, []metrics.Metric{*gaugeMetric, invalidMetric})
	if err == nil {
		t.Errorf("UpdateMetrics() expected error for invalid metric")
	}
}

func TestMemStorage_GetMetrics(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Добавляем тестовые метрики
	gaugeValue := 10.5
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)
	ms.UpdateMetric(ctx, *gaugeMetric)

	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)
	ms.UpdateMetric(ctx, *counterMetric)

	// Получаем все метрики
	allMetrics := ms.GetMetrics(ctx)

	// Проверяем количество метрик
	if len(allMetrics) != 2 {
		t.Errorf("GetMetrics() returned %d metrics, want %d", len(allMetrics), 2)
	}

	// Проверяем, что все метрики присутствуют
	foundGauge := false
	foundCounter := false

	for _, m := range allMetrics {
		if m.Name == "test_gauge" && m.MType == metrics.TypeGauge && *m.Value == gaugeValue {
			foundGauge = true
		}
		if m.Name == "test_counter" && m.MType == metrics.TypeCounter && *m.Delta == counterValue {
			foundCounter = true
		}
	}

	if !foundGauge {
		t.Errorf("GetMetrics() did not return the expected gauge metric")
	}
	if !foundCounter {
		t.Errorf("GetMetrics() did not return the expected counter metric")
	}
}

func TestMemStorage_FileOperations(t *testing.T) {
	// Создаем временный файл для тестирования
	tmpFile, err := os.CreateTemp("", "metrics_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Создаем конфиг с включенным сохранением в файл
	cfg := &app.Config{
		FileStoragePath: tmpFile.Name(),
		StoreInterval:   1,
	}

	// Создаем хранилище
	ms := setupTestStorageWithConfig(t, cfg)
	ctx := context.Background()

	// Добавляем тестовые метрики
	gaugeValue := 10.5
	gaugeMetric := metrics.NewGauge("test_gauge", gaugeValue)
	ms.UpdateMetric(ctx, *gaugeMetric)

	counterValue := int64(10)
	counterMetric := metrics.NewCounter("test_counter", counterValue)
	ms.UpdateMetric(ctx, *counterMetric)

	// Сохраняем метрики в файл
	err = ms.storeMetricsToFile()
	if err != nil {
		t.Errorf("storeMetricsToFile() error = %v", err)
	}

	// Создаем новое хранилище для тестирования восстановления
	newCfg := &app.Config{
		FileStoragePath: tmpFile.Name(),
		StoreInterval:   1,
	}
	newMS := setupTestStorageWithConfig(t, newCfg)

	// Вручную вызываем восстановление метрик из файла
	err = newMS.restoreMetricsFromFile()
	if err != nil {
		t.Errorf("restoreMetricsFromFile() error = %v", err)
	}

	// Проверяем, что метрики были восстановлены
	restoredGauge, existsGauge := newMS.GetGauge(ctx, gaugeMetric.Name)
	if !existsGauge {
		t.Errorf("restoreMetricsFromFile() failed to restore gauge metric")
	}
	if restoredGauge != storage.Gauge(gaugeValue) {
		t.Errorf("restoreMetricsFromFile() gauge = %v, want %v", restoredGauge, gaugeValue)
	}

	// Для counter метрик, значение может быть увеличено, если метрика уже существовала
	// Поэтому проверяем только наличие метрики
	_, existsCounter := newMS.GetCounter(ctx, counterMetric.Name)
	if !existsCounter {
		t.Errorf("restoreMetricsFromFile() failed to restore counter metric")
	}
}

func TestMemStorage_Dump(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Добавляем тестовые метрики
	ms.UpdateGauge("gauge1", 1.0)
	ms.UpdateGauge("gauge2", 2.0)
	ms.IncrementCounter("counter1", 1)

	// Вызываем Dump (просто проверяем, что не возникает паники)
	ms.Dump()

	// Проверяем, что количество метрик не изменилось
	count := ms.Count(ctx)
	if count != 3 {
		t.Errorf("Count() after Dump() = %v, want %v", count, 3)
	}
}

func TestMemStorage_UpdateMetric_Errors(t *testing.T) {
	ms := setupTestStorage(t)
	ctx := context.Background()

	// Тест для метрики типа gauge с nil value
	t.Run("Update gauge metric with nil value", func(t *testing.T) {
		metric := metrics.Metric{
			Name:  "test_gauge_nil",
			MType: metrics.TypeGauge,
			Value: nil,
		}

		err := ms.UpdateMetric(ctx, metric)
		if err == nil {
			t.Errorf("UpdateMetric() expected error for nil gauge value")
		}
	})

	// Тест для метрики типа counter с nil delta
	t.Run("Update counter metric with nil delta", func(t *testing.T) {
		metric := metrics.Metric{
			Name:  "test_counter_nil",
			MType: metrics.TypeCounter,
			Delta: nil,
		}

		err := ms.UpdateMetric(ctx, metric)
		if err == nil {
			t.Errorf("UpdateMetric() expected error for nil counter delta")
		}
	})
}

func TestMemStorage_Close(t *testing.T) {
	// Создаем временный файл для тестирования
	tmpFile, err := os.CreateTemp("", "metrics_close_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Тест с выключенным сохранением
	t.Run("Close with store disabled", func(t *testing.T) {
		cfg := &app.Config{
			FileStoragePath: "",
		}

		ms := setupTestStorageWithConfig(t, cfg)

		err := ms.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	// Тест с включенным сохранением
	t.Run("Close with store enabled", func(t *testing.T) {
		// Создаем файл с валидным JSON для восстановления
		validJSON := `[{"id":"test_gauge","type":"gauge","value":1}]`
		err := os.WriteFile(tmpFile.Name(), []byte(validJSON), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		cfg := &app.Config{
			FileStoragePath: tmpFile.Name(),
			StoreInterval:   0, // Синхронное сохранение
		}

		ms := setupTestStorageWithConfig(t, cfg)

		// Добавляем метрику
		ms.UpdateGauge("test_gauge", 1.0)

		// Закрываем хранилище
		err = ms.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}

		// Проверяем, что файл существует и содержит данные
		fileInfo, err := os.Stat(tmpFile.Name())
		if err != nil {
			t.Errorf("Failed to stat file after Close(): %v", err)
		}
		if fileInfo.Size() == 0 {
			t.Errorf("File is empty after Close()")
		}
	})
}
