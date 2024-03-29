// Агент (HTTP-клиент) для сбора рантайм-метрик и их последующей отправки на сервер по протоколу HTTP
package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type Agent struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerURL      string
	gauges         map[string]interface{}
	counters       map[string]int64
	mu             sync.Mutex
	wg             sync.WaitGroup
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	// создаем новый агент
	agent := &Agent{
		ServerURL:      "http://localhost:8080",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		gauges:         make(map[string]interface{}),
		counters:       make(map[string]int64),
	}

	// запускаем агент
	agent.Run()

	// Ожидаем завершения всех горутин
	agent.wg.Wait()
}

func (a *Agent) Run() {
	a.wg.Add(2)
	go a.runPolls()
	go a.runReports()
	fmt.Printf("agent is running\n")
}

func (a *Agent) runPolls() {
	defer a.wg.Done()
	for {
		a.mu.Lock()
		// Перезаписываем метрики свежими показаниями runtime.MemStats
		a.gauges = a.collectRuntimeMetrics()
		// Добавляем обновляемое рандомное значение по условию
		a.gauges["RandomValue"] = rand.Float64()
		a.counters["PollCount"]++
		fmt.Printf("%d ", a.counters["PollCount"])
		a.mu.Unlock()
		time.Sleep(a.PollInterval)
	}
}

func (a *Agent) runReports() {
	defer a.wg.Done()
	for {
		time.Sleep(a.ReportInterval)
		a.sendMetrics()
	}
}

func (a *Agent) collectRuntimeMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics["Alloc"] = m.Alloc
	metrics["BuckHashSys"] = m.BuckHashSys
	metrics["Frees"] = m.Frees
	metrics["GCCPUFraction"] = m.GCCPUFraction
	metrics["GCSys"] = m.GCSys
	metrics["HeapAlloc"] = m.HeapAlloc
	metrics["HeapIdle"] = m.HeapIdle
	metrics["HeapInuse"] = m.HeapInuse
	metrics["HeapObjects"] = m.HeapObjects
	metrics["HeapReleased"] = m.HeapReleased
	metrics["HeapSys"] = m.HeapSys
	metrics["LastGC"] = m.LastGC
	metrics["Lookups"] = m.Lookups
	metrics["MCacheInuse"] = m.MCacheInuse
	metrics["MCacheSys"] = m.MCacheSys
	metrics["MSpanInuse"] = m.MSpanInuse
	metrics["MSpanSys"] = m.MSpanSys
	metrics["Mallocs"] = m.Mallocs
	metrics["NextGC"] = m.NextGC
	metrics["NumForcedGC"] = m.NumForcedGC
	metrics["NumGC"] = m.NumGC
	metrics["OtherSys"] = m.OtherSys
	metrics["PauseTotalNs"] = m.PauseTotalNs
	metrics["StackInuse"] = m.StackInuse
	metrics["StackSys"] = m.StackSys
	metrics["Sys"] = m.Sys
	metrics["TotalAlloc"] = m.TotalAlloc

	return metrics
}

func (a *Agent) sendMetrics() {

	gauges := make(map[string]interface{})
	counters := make(map[string]int64)

	// Делаем копию метрик, чтобы данные не изменились во время отправки
	a.mu.Lock()
	fmt.Printf("\nSending metrics, current poll count: %d\n", a.counters["PollCount"])
	for name, value := range a.gauges {
		gauges[name] = value
	}
	for name, value := range a.counters {
		counters[name] = value
	}
	pollCount := a.counters["PollCount"]
	a.mu.Unlock()

	//Отправляем gauges
	for name, value := range gauges {
		err := a.sendMetric("gauges", name, value, pollCount)
		if err != nil {
			fmt.Printf("failed to send gauge %s: %v\n", name, err)
			return
		}
	}
	// Отправляем counters
	for name, value := range counters {
		err := a.sendMetric("counters", name, value, pollCount)
		if err != nil {
			fmt.Printf("failed to send counter %s: %v\n", name, err)
			return
		}
	}
}

func (a *Agent) sendMetric(metricType string, name string, value interface{}, pollCount int64) error {
	url := fmt.Sprintf("%s/update/%s/%s/%v", a.ServerURL, metricType, name, value)
	fmt.Printf("%d. sending metric: %s\n", pollCount, url)

	_, err := http.Post(url, "text/plain", nil)
	if err != nil {
		return err
	}
	return nil
}
