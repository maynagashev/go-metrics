// Package agent методы агента для сбора метрик.
package agent

import "runtime"

func (a *Agent) CollectRuntimeMetrics() map[string]float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mm := make(map[string]float64)
	mm["Alloc"] = float64(m.Alloc)
	mm["BuckHashSys"] = float64(m.BuckHashSys)
	mm["Frees"] = float64(m.Frees)
	mm["GCCPUFraction"] = m.GCCPUFraction
	mm["GCSys"] = float64(m.GCSys)
	mm["HeapAlloc"] = float64(m.HeapAlloc)
	mm["HeapIdle"] = float64(m.HeapIdle)
	mm["HeapInuse"] = float64(m.HeapInuse)
	mm["HeapObjects"] = float64(m.HeapObjects)
	mm["HeapReleased"] = float64(m.HeapReleased)
	mm["HeapSys"] = float64(m.HeapSys)
	mm["LastGC"] = float64(m.LastGC)
	mm["Lookups"] = float64(m.Lookups)
	mm["MCacheInuse"] = float64(m.MCacheInuse)
	mm["MCacheSys"] = float64(m.MCacheSys)
	mm["MSpanInuse"] = float64(m.MSpanInuse)
	mm["MSpanSys"] = float64(m.MSpanSys)
	mm["Mallocs"] = float64(m.Mallocs)
	mm["NextGC"] = float64(m.NextGC)
	mm["NumForcedGC"] = float64(m.NumForcedGC)
	mm["NumGC"] = float64(m.NumGC)
	mm["OtherSys"] = float64(m.OtherSys)
	mm["PauseTotalNs"] = float64(m.PauseTotalNs)
	mm["StackInuse"] = float64(m.StackInuse)
	mm["StackSys"] = float64(m.StackSys)
	mm["Sys"] = float64(m.Sys)
	mm["TotalAlloc"] = float64(m.TotalAlloc)

	return mm
}
