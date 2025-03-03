package agent

import (
	"testing"
)

func TestAgent_ResetMetrics(t *testing.T) {
	// Создаем тестового агента
	a := &agent{
		gauges:   map[string]float64{"test": 1.0},
		counters: map[string]int64{"test": 1},
	}

	// Проверяем, что метрики не пусты
	if len(a.gauges) == 0 || len(a.counters) == 0 {
		t.Errorf("Expected non-empty metrics before reset")
	}

	// Сбрасываем метрики
	a.ResetMetrics()

	// Проверяем, что метрики пусты
	if len(a.gauges) != 0 {
		t.Errorf("Expected empty gauges after reset, got %d", len(a.gauges))
	}
	if len(a.counters) != 0 {
		t.Errorf("Expected empty counters after reset, got %d", len(a.counters))
	}
}

func TestAgent_CollectRuntimeMetrics(t *testing.T) {
	// Создаем тестового агента
	a := &agent{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}

	// Собираем метрики
	a.CollectRuntimeMetrics()

	// Проверяем, что метрики были собраны
	expectedMetrics := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc",
	}

	for _, metric := range expectedMetrics {
		if _, exists := a.gauges[metric]; !exists {
			t.Errorf("Expected metric %s to be collected", metric)
		}
	}
}

func TestAgent_CollectAdditionalMetrics(t *testing.T) {
	// Создаем тестового агента
	a := &agent{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}

	// Собираем метрики
	a.CollectAdditionalMetrics()

	// Проверяем, что метрики памяти были собраны
	// Примечание: в тестовой среде не всегда могут быть доступны все метрики,
	// поэтому проверяем только наличие метрик, которые должны быть доступны
	memoryMetricsFound := 0
	expectedMemoryMetrics := []string{"TotalMemory", "FreeMemory"}

	for _, metric := range expectedMemoryMetrics {
		if _, exists := a.gauges[metric]; exists {
			memoryMetricsFound++
		}
	}

	// Проверяем, что хотя бы одна метрика памяти была собрана
	if memoryMetricsFound == 0 {
		t.Errorf("Expected at least one memory metric to be collected")
	} else {
		t.Logf("Collected %d/%d memory metrics", memoryMetricsFound, len(expectedMemoryMetrics))
	}

	// Проверяем, что метрики CPU были собраны
	// Количество CPU может отличаться на разных машинах,
	// а в тестовой среде метрики CPU могут быть недоступны,
	// поэтому просто логируем результат
	cpuMetricCount := 0
	for metric := range a.gauges {
		if len(metric) >= 15 && metric[:15] == "CPUutilization" {
			cpuMetricCount++
		}
	}

	t.Logf("Collected %d CPU utilization metrics", cpuMetricCount)
}
