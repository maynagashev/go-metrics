package agent_test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/maynagashev/go-metrics/internal/agent"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/stretchr/testify/assert"
)

func TestAgent_CollectRuntimeMetrics(t *testing.T) {
	// Create a test agent with required parameters
	a := agent.New(
		"http://localhost:8080",
		time.Second,
		time.Second,
		"",
		5,
	)

	// Reset metrics before collection
	a.ResetMetrics()

	// Collect runtime metrics
	a.CollectRuntimeMetrics()

	// Check that metrics were collected
	metricsList := a.GetMetrics()
	assert.NotEmpty(t, metricsList)

	// Create a map for easier checking
	metricsMap := make(map[string]*metrics.Metric)
	for _, m := range metricsList {
		metricsMap[m.Name] = m
	}

	// Check for specific metrics
	expectedMetrics := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc",
	}

	for _, name := range expectedMetrics {
		_, exists := metricsMap[name]
		assert.True(t, exists, "Metric %s should be present", name)
	}
}

func TestAgent_CollectAdditionalMetrics(t *testing.T) {
	// Create a test agent with required parameters
	a := agent.New(
		"http://localhost:8080",
		time.Second,
		time.Second,
		"",
		5,
	)

	// Reset metrics before collection
	a.ResetMetrics()

	// Collect additional metrics
	a.CollectAdditionalMetrics()

	// Check that metrics were collected
	metricsList := a.GetMetrics()
	assert.NotEmpty(t, metricsList)

	// Create a map for easier checking
	metricsMap := make(map[string]*metrics.Metric)
	for _, m := range metricsList {
		metricsMap[m.Name] = m
	}

	// Check for TotalMemory
	_, ok := metricsMap["TotalMemory"]
	assert.True(t, ok, "TotalMemory metric should be present")

	// Check for FreeMemory
	_, ok = metricsMap["FreeMemory"]
	assert.True(t, ok, "FreeMemory metric should be present")

	// Check for CPU utilization metrics
	cpuCount := runtime.NumCPU()
	for i := range cpuCount {
		cpuMetricName := fmt.Sprintf("CPUutilization%d", i+1)
		_, ok := metricsMap[cpuMetricName]
		assert.True(t, ok, "CPU utilization metric %s should be present", cpuMetricName)
	}
}

func TestAgent_GetMetrics(t *testing.T) {
	// Create a test agent with required parameters
	a := agent.New(
		"http://localhost:8080",
		time.Second,
		time.Second,
		"",
		5,
	)

	// Reset metrics before collection
	a.ResetMetrics()

	// Collect both types of metrics
	a.CollectRuntimeMetrics()
	a.CollectAdditionalMetrics()

	// Get metrics
	metricsList := a.GetMetrics()
	assert.NotEmpty(t, metricsList)

	// Check that all metrics have the correct format
	for _, m := range metricsList {
		assert.NotEmpty(t, m.Name)
		assert.NotEmpty(t, m.MType)

		if m.MType == metrics.TypeGauge {
			assert.NotNil(t, m.Value)
			assert.Nil(t, m.Delta)
		} else if m.MType == metrics.TypeCounter {
			assert.NotNil(t, m.Delta)
			assert.Nil(t, m.Value)
		} else {
			t.Errorf("Unknown metric type: %s", m.MType)
		}
	}
}

func TestAgent_ResetMetrics(t *testing.T) {
	// Create a test agent with required parameters
	a := agent.New(
		"http://localhost:8080",
		time.Second,
		time.Second,
		"",
		5,
	)

	// Collect metrics
	a.CollectRuntimeMetrics()
	a.CollectAdditionalMetrics()

	// Check that metrics were collected
	metricsBefore := a.GetMetrics()
	assert.NotEmpty(t, metricsBefore)

	// Reset metrics
	a.ResetMetrics()

	// Check that metrics were reset
	metricsAfter := a.GetMetrics()
	assert.Empty(t, metricsAfter)
}

func TestAgent_IsRequestSigningEnabled(t *testing.T) {
	// Test with no private key
	a1 := agent.New(
		"http://localhost:8080",
		time.Second,
		time.Second,
		"",
		5,
	)
	assert.False(t, a1.IsRequestSigningEnabled())

	// Test with private key
	a2 := agent.New(
		"http://localhost:8080",
		time.Second,
		time.Second,
		"test-private-key",
		5,
	)
	assert.True(t, a2.IsRequestSigningEnabled())
}
