// Package memory provides an in-memory storage for metrics.
package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"

	"github.com/maynagashev/go-metrics/internal/server/storage"
)

type MemStorage struct {
	gauges   storage.Gauges
	counters storage.Counters
	server   *app.Server
	log      *zap.Logger
}

// New создает новый экземпляр хранилища метрик в памяти, на вход
// можно передать набор gauges или counters для инициализации в тестах.
func New(server *app.Server, log *zap.Logger, options ...interface{}) *MemStorage {
	memStorage := &MemStorage{
		gauges:   make(storage.Gauges),
		counters: make(storage.Counters),
		server:   server,
		log:      log,
	}
	log.Debug("memory storage created", zap.Any("storage", memStorage))

	// Если включено восстановление метрик из файла, то пытаемся прочитать метрики из файла.
	if server.IsRestoreEnabled() {
		err := memStorage.RestoreMetricsFromFile()
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

	// Запускаем сохранение метрик в файл c указанным интервалом.
	if server.IsStoreEnabled() && !server.IsSyncStore() {
		interval := time.Duration(server.GetStoreInterval()) * time.Second
		go func() {
			for {
				time.Sleep(interval)
				log.Info(fmt.Sprintf("store %d metrics to file %s", memStorage.Count(), server.GetStorePath()))
				err := memStorage.StoreMetricsToFile()
				if err != nil {
					log.Error("failed to store metrics to file", zap.Error(err))
				}
			}
		}()
	}

	return memStorage
}

// UpdateGauge перезаписывает значение gauge.
func (ms *MemStorage) UpdateGauge(metricName string, metricValue storage.Gauge) {
	ms.gauges[metricName] = metricValue
}

// IncrementCounter увеличивает значение счетчика на заданное значение.
func (ms *MemStorage) IncrementCounter(metricName string, metricValue storage.Counter) {
	ms.counters[metricName] += metricValue
}

func (ms *MemStorage) UpdateMetric(metric metrics.Metric) error {
	switch metric.MType {
	case metrics.TypeGauge:
		if metric.Value == nil {
			return errors.New("gauge value is nil")
		}
		ms.UpdateGauge(metric.ID, storage.Gauge(*metric.Value))
	case metrics.TypeCounter:
		if metric.Delta == nil {
			return errors.New("counter delta is nil")
		}
		ms.IncrementCounter(metric.ID, storage.Counter(*metric.Delta))
	default:
		return fmt.Errorf("unsupported metric type: %s", metric.MType)
	}
	return nil
}

func (ms *MemStorage) GetGauges() storage.Gauges {
	return ms.gauges
}

func (ms *MemStorage) GetCounters() storage.Counters {
	return ms.counters
}

func (ms *MemStorage) GetGauge(name string) (storage.Gauge, bool) {
	value, ok := ms.gauges[name]
	return value, ok
}

func (ms *MemStorage) GetCounter(name string) (storage.Counter, bool) {
	value, ok := ms.counters[name]
	return value, ok
}

func (ms *MemStorage) Count() int {
	return len(ms.gauges) + len(ms.counters)
}

func (ms *MemStorage) GetMetric(mType metrics.MetricType, id string) (metrics.Metric, bool) {
	switch mType {
	case metrics.TypeCounter:
		v, ok := ms.GetCounter(id)
		return metrics.Metric{
			ID:    id,
			MType: mType,
			Delta: (*int64)(&v),
		}, ok
	case metrics.TypeGauge:
		v, ok := ms.GetGauge(id)
		return metrics.Metric{
			ID:    id,
			MType: mType,
			Value: (*float64)(&v),
		}, ok
	}
	return metrics.Metric{}, false
}

// GetValue возвращает значение метрики по типу и имени.
func (ms *MemStorage) GetValue(mType metrics.MetricType, name string) (fmt.Stringer, bool) {
	switch mType {
	case metrics.TypeCounter:
		v, ok := ms.GetCounter(name)
		return v, ok
	case metrics.TypeGauge:
		v, ok := ms.GetGauge(name)
		return v, ok
	}
	return nil, false
}

// GetMetrics возвращает отсортированный список метрик в формате слайса структур.
func (ms *MemStorage) GetMetrics() []metrics.Metric {
	items := make([]metrics.Metric, 0, ms.Count())
	for id, value := range ms.GetGauges() {
		//nolint:gosec // в Go 1.22, значение в цикле копируется (G601: Implicit memory aliasing in for loop.)
		items = append(items, metrics.Metric{ID: id, MType: metrics.TypeGauge, Value: (*float64)(&value)})
	}
	for id, value := range ms.GetCounters() {
		//nolint:gosec // в Go 1.22, значение в цикле копируется (G601: Implicit memory aliasing in for loop.)
		items = append(items, metrics.Metric{ID: id, MType: metrics.TypeCounter, Delta: (*int64)(&value)})
	}
	// slices.Sort(items)
	return items
}

// StoreMetricsToFile сохраняет метрики в файл.
func (ms *MemStorage) StoreMetricsToFile() error {
	path := ms.server.GetStorePath()
	ms.log.Debug("store metrics to file",
		zap.String("path", path),
		zap.Any("gauges", ms.GetGauges()),
		zap.Any("counters", ms.GetCounters()))

	// открытие файла для записи
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		err = f.Close()
		if err != nil {
			ms.log.Error(fmt.Sprintf("error closing file: %s", err))
		}
	}()

	// сериализация метрик metrics.Metric в json и запись сразу в файл
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(ms.GetMetrics())
	if err != nil {
		return err
	}

	return nil
}

// RestoreMetricsFromFile загружает метрики из файла.
func (ms *MemStorage) RestoreMetricsFromFile() error {
	path := ms.server.GetStorePath()
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
		err = ms.UpdateMetric(parsed[m])
		if err != nil {
			return err
		}
	}

	return nil
}
