// Package agent методы агента для сбора метрик.
package agent

import (
	"fmt"
	"log/slog"
	"runtime"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// Константы для конвертации единиц измерения.
const (
	BytesInKB = 1024
	BytesInMB = BytesInKB * 1024
	BytesInGB = BytesInMB * 1024
)

// ResetMetrics очищает все метрики агента, вызываем перед сбором новых метрик.
func (a *agent) ResetMetrics() {
	slog.Debug("Resetting metrics before collection")
	a.gauges = make(map[string]float64)
	a.counters = make(map[string]int64)
}

func (a *agent) CollectRuntimeMetrics() {
	// Проверяем сигнал остановки перед сбором метрик
	select {
	case <-a.stopCh:
		slog.Info("Shutdown signal received, skipping runtime metrics collection")
		return
	default:
		// Продолжаем выполнение
	}

	slog.Debug("Starting runtime metrics collection")
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	a.gauges["Alloc"] = float64(m.Alloc)
	a.gauges["BuckHashSys"] = float64(m.BuckHashSys)
	a.gauges["Frees"] = float64(m.Frees)
	a.gauges["GCCPUFraction"] = m.GCCPUFraction
	a.gauges["GCSys"] = float64(m.GCSys)
	a.gauges["HeapAlloc"] = float64(m.HeapAlloc)
	a.gauges["HeapIdle"] = float64(m.HeapIdle)
	a.gauges["HeapInuse"] = float64(m.HeapInuse)
	a.gauges["HeapObjects"] = float64(m.HeapObjects)
	a.gauges["HeapReleased"] = float64(m.HeapReleased)
	a.gauges["HeapSys"] = float64(m.HeapSys)
	a.gauges["LastGC"] = float64(m.LastGC)
	a.gauges["Lookups"] = float64(m.Lookups)
	a.gauges["MCacheInuse"] = float64(m.MCacheInuse)
	a.gauges["MCacheSys"] = float64(m.MCacheSys)
	a.gauges["MSpanInuse"] = float64(m.MSpanInuse)
	a.gauges["MSpanSys"] = float64(m.MSpanSys)
	a.gauges["Mallocs"] = float64(m.Mallocs)
	a.gauges["NextGC"] = float64(m.NextGC)
	a.gauges["NumForcedGC"] = float64(m.NumForcedGC)
	a.gauges["NumGC"] = float64(m.NumGC)
	a.gauges["OtherSys"] = float64(m.OtherSys)
	a.gauges["PauseTotalNs"] = float64(m.PauseTotalNs)
	a.gauges["StackInuse"] = float64(m.StackInuse)
	a.gauges["StackSys"] = float64(m.StackSys)
	a.gauges["Sys"] = float64(m.Sys)
	a.gauges["TotalAlloc"] = float64(m.TotalAlloc)

	slog.Debug("Runtime metrics collection completed",
		"metrics_count", len(a.gauges),
		"heap_alloc_mb", float64(m.HeapAlloc)/BytesInMB,
		"sys_mb", float64(m.Sys)/BytesInMB)
}

// CollectAdditionalMetrics собирает дополнительные метрики системы.
func (a *agent) CollectAdditionalMetrics() {
	// Проверяем сигнал остановки перед сбором метрик
	select {
	case <-a.stopCh:
		slog.Info("Shutdown signal received, skipping additional metrics collection")
		return
	default:
		// Продолжаем выполнение
	}

	slog.Debug("Starting additional system metrics collection")

	v, err := mem.VirtualMemory()
	if err != nil {
		slog.Error("Failed to collect virtual memory metrics", "error", err)
		return
	}
	a.gauges["TotalMemory"] = float64(v.Total)
	a.gauges["FreeMemory"] = float64(v.Free)

	slog.Debug("Memory metrics collected",
		"total_memory_gb", float64(v.Total)/BytesInGB,
		"free_memory_gb", float64(v.Free)/BytesInGB,
		"used_percent", v.UsedPercent)

	c, err := cpu.Percent(0, true)
	if err != nil {
		slog.Error("Failed to collect CPU metrics", "error", err)
		return
	}

	cpuCount := 0
	for i, percent := range c {
		a.gauges[fmt.Sprintf("CPUutilization%d", i+1)] = percent
		cpuCount++
	}

	slog.Debug("CPU metrics collection completed", "cpu_count", cpuCount)
}
