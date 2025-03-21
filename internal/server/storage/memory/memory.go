// Package memory provides an in-memory storage for metrics.
package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"
)

const maxRetries = 3

type MemStorage struct {
	gauges   storage.Gauges
	counters storage.Counters
	cfg      *app.Config
	log      *zap.Logger
}

// New создает новый экземпляр хранилища метрик в памяти, на вход
// можно передать набор gauges или counters для инициализации в тестах.
func New(cfg *app.Config, log *zap.Logger, options ...interface{}) *MemStorage {
	memStorage := &MemStorage{
		gauges:   make(storage.Gauges),
		counters: make(storage.Counters),
		cfg:      cfg,
		log:      log,
	}
	log.Debug("memory storage created", zap.Any("storage", memStorage))

	// Если включено восстановление метрик из файла, то пытаемся прочитать метрики из файла.
	if cfg.IsRestoreEnabled() {
		err := memStorage.restoreMetricsFromFile()
		if err != nil {
			log.Error("failed to read metrics from file", zap.Error(err))
		}
	}

	// Если переданы метрики для инициализации (для тестов хранилища) то обновляем их в хранилище.
	for _, option := range options {
		switch opt := option.(type) {
		case storage.Gauges:
			memStorage.gauges = opt
		case storage.Counters:
			memStorage.counters = opt
		}
	}

	// Запускаем сохранение метрик в файл с указанным интервалом.
	if cfg.IsStoreEnabled() && !cfg.IsSyncStore() {
		interval := time.Duration(cfg.GetStoreInterval()) * time.Second
		go func() {
			for {
				time.Sleep(interval)
				log.Info(
					fmt.Sprintf(
						"store %d metrics to file %s",
						memStorage.Count(context.Background()),
						cfg.GetStorePath(),
					),
				)
				err := memStorage.storeMetricsToFile()
				if err != nil {
					log.Error("failed to store metrics to file", zap.Error(err))
				}
			}
		}()
	}

	return memStorage
}

func (ms *MemStorage) Close() error {
	if ms.cfg.IsStoreEnabled() && !ms.cfg.IsSyncStore() {
		return ms.storeMetricsToFile()
	}
	return nil
}

// UpdateGauge перезаписывает значение gauge.
func (ms *MemStorage) UpdateGauge(metricName string, metricValue storage.Gauge) {
	ms.gauges[metricName] = metricValue
}

// IncrementCounter увеличивает значение счетчика на заданное значение.
func (ms *MemStorage) IncrementCounter(metricName string, metricValue storage.Counter) {
	ms.counters[metricName] += metricValue
}

// UpdateMetric универсальный метод обновления метрики в хранилище: gauge, counter.
func (ms *MemStorage) UpdateMetric(_ context.Context, metric metrics.Metric) error {
	switch metric.MType {
	case metrics.TypeGauge:
		if metric.Value == nil {
			return errors.New("gauge value is nil")
		}
		ms.UpdateGauge(metric.Name, storage.Gauge(*metric.Value))
	case metrics.TypeCounter:
		if metric.Delta == nil {
			return errors.New("counter delta is nil")
		}
		ms.IncrementCounter(metric.Name, storage.Counter(*metric.Delta))
	default:
		return fmt.Errorf("unsupported metric type: %s", metric.MType)
	}

	// Сохраняем метрики в файл сразу после изменения, если включено синхронное сохранение.
	if ms.cfg.IsStoreEnabled() && ms.cfg.IsSyncStore() {
		err := ms.storeMetricsToFile()
		if err != nil {
			// Информация об ошибке синхронной записи для клиента может быть избыточной, поэтому просто логируем ошибку.
			ms.log.Error(
				fmt.Sprintf("error while trying to syncroniously store metrics to file: %s", err),
			)
		}
	}
	return nil
}

func (ms *MemStorage) UpdateMetrics(ctx context.Context, items []metrics.Metric) error {
	for _, item := range items {
		err := ms.UpdateMetric(ctx, item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ms *MemStorage) GetGauges() storage.Gauges {
	return ms.gauges
}

func (ms *MemStorage) GetCounters() storage.Counters {
	return ms.counters
}

func (ms *MemStorage) GetGauge(_ context.Context, name string) (storage.Gauge, bool) {
	value, ok := ms.gauges[name]
	return value, ok
}

func (ms *MemStorage) GetCounter(_ context.Context, name string) (storage.Counter, bool) {
	value, ok := ms.counters[name]
	return value, ok
}

func (ms *MemStorage) Count(_ context.Context) int {
	return len(ms.gauges) + len(ms.counters)
}

func (ms *MemStorage) GetMetric(
	ctx context.Context,
	mType metrics.MetricType,
	id string,
) (metrics.Metric, bool) {
	switch mType {
	case metrics.TypeCounter:
		v, ok := ms.GetCounter(ctx, id)
		return metrics.Metric{
			Name:  id,
			MType: mType,
			Delta: (*int64)(&v),
		}, ok
	case metrics.TypeGauge:
		v, ok := ms.GetGauge(ctx, id)
		return metrics.Metric{
			Name:  id,
			MType: mType,
			Value: (*float64)(&v),
		}, ok
	}
	return metrics.Metric{}, false
}

// GetMetrics возвращает отсортированный список метрик в формате слайса структур.
func (ms *MemStorage) GetMetrics(ctx context.Context) []metrics.Metric {
	items := make([]metrics.Metric, 0, ms.Count(ctx))
	for id, value := range ms.GetGauges() {
		items = append(
			items,
			metrics.Metric{Name: id, MType: metrics.TypeGauge, Value: (*float64)(&value)},
		)
	}
	for id, value := range ms.GetCounters() {
		items = append(
			items,
			metrics.Metric{Name: id, MType: metrics.TypeCounter, Delta: (*int64)(&value)},
		)
	}
	// slices.Sort(items)
	return items
}

// StoreMetricsToFile сохраняет метрики в файл.
func (ms *MemStorage) storeMetricsToFile() error {
	path := ms.cfg.GetStorePath()
	ms.log.Debug("store metrics to file",
		zap.String("path", path),
		zap.Any("gauges", ms.GetGauges()),
		zap.Any("counters", ms.GetCounters()))

	var f *os.File
	var err error

	for i := 0; i <= maxRetries; i++ {
		// попытка открытия файла для записи
		f, err = os.Create(path)
		if err == nil {
			break
		}
		ms.log.Error(fmt.Sprintf("Attempt %d: Error opening file: %v", i+1, err))
		time.Sleep(time.Duration((i+1)*2-1) * time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to open file after %d attempts: %w", maxRetries, err)
	}

	// отложенное закрытие файла
	defer func() {
		err = f.Close()
		if err != nil {
			ms.log.Error(fmt.Sprintf("error closing file: %s", err))
		}
	}()

	// сериализация метрик metrics.Metric в json и запись сразу в файл
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(ms.GetMetrics(context.Background()))
	if err != nil {
		return err
	}

	return nil
}

// RestoreMetricsFromFile загружает метрики из файла.
func (ms *MemStorage) restoreMetricsFromFile() error {
	path := ms.cfg.GetStorePath()
	ms.log.Debug("load metrics from file", zap.String("path", path))

	// открытие файла для чтения и парсинг json метрик metrics.Metric
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		err = f.Close()
		if err != nil {
			ms.log.Error(fmt.Sprintf("error closing file: %s", err), zap.Any("file", f))
		}
	}()

	// парсинг json метрик metrics.Metric
	var parsed []metrics.Metric
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&parsed)
	if err != nil {
		return err
	}

	// обновление метрик в хранилище в памяти
	for m := range parsed {
		err = ms.UpdateMetric(context.Background(), parsed[m])
		if err != nil {
			return err
		}
	}

	// Выводим информацию о восстановленных метриках в лог.
	ms.log.Info(
		"Metrics restored from file",
		zap.String("file", ms.cfg.GetStorePath()),
		zap.Int("metrics", len(ms.GetMetrics(context.Background()))),
	)
	return nil
}
