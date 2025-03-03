package agent_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/maynagashev/go-metrics/internal/agent"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent_CollectMetrics(t *testing.T) {
	// Create a test agent
	a := agent.New(nil)
	ctx := context.Background()

	// Collect metrics
	err := a.CollectMetrics(ctx)
	require.NoError(t, err)

	// Check that metrics were collected
	assert.NotEmpty(t, a.GetMetrics())

	// Check for specific metrics
	metrics := a.GetMetrics()
	assert.Contains(t, metrics, "Alloc")
	assert.Contains(t, metrics, "BuckHashSys")
	assert.Contains(t, metrics, "Frees")
	assert.Contains(t, metrics, "GCCPUFraction")
	assert.Contains(t, metrics, "GCSys")
	assert.Contains(t, metrics, "HeapAlloc")
	assert.Contains(t, metrics, "HeapIdle")
	assert.Contains(t, metrics, "HeapInuse")
	assert.Contains(t, metrics, "HeapObjects")
	assert.Contains(t, metrics, "HeapReleased")
	assert.Contains(t, metrics, "HeapSys")
	assert.Contains(t, metrics, "LastGC")
	assert.Contains(t, metrics, "Lookups")
	assert.Contains(t, metrics, "MCacheInuse")
	assert.Contains(t, metrics, "MCacheSys")
	assert.Contains(t, metrics, "MSpanInuse")
	assert.Contains(t, metrics, "MSpanSys")
	assert.Contains(t, metrics, "Mallocs")
	assert.Contains(t, metrics, "NextGC")
	assert.Contains(t, metrics, "NumForcedGC")
	assert.Contains(t, metrics, "NumGC")
	assert.Contains(t, metrics, "OtherSys")
	assert.Contains(t, metrics, "PauseTotalNs")
	assert.Contains(t, metrics, "StackInuse")
	assert.Contains(t, metrics, "StackSys")
	assert.Contains(t, metrics, "Sys")
	assert.Contains(t, metrics, "TotalAlloc")
	assert.Contains(t, metrics, "PollCount")
}

func TestAgent_CollectAdditionalMetrics(t *testing.T) {
	// Create a test agent
	a := agent.New(nil)
	ctx := context.Background()

	// Collect additional metrics
	err := a.CollectAdditionalMetrics(ctx)
	require.NoError(t, err)

	// Check that metrics were collected
	metrics := a.GetMetrics()

	// Check for TotalMemory
	_, ok := metrics["TotalMemory"]
	assert.True(t, ok, "TotalMemory metric should be present")

	// Check for FreeMemory
	_, ok = metrics["FreeMemory"]
	assert.True(t, ok, "FreeMemory metric should be present")

	// Check for CPU utilization metrics
	cpuCount := runtime.NumCPU()
	for i := 0; i < cpuCount; i++ {
		cpuMetricName := "CPUutilization" + string(rune('1'+i))
		_, ok := metrics[cpuMetricName]
		assert.True(t, ok, "CPU utilization metric %s should be present", cpuMetricName)
	}
}

func TestAgent_ReportMetrics(t *testing.T) {
	// Create a test agent
	a := agent.New(nil)
	ctx := context.Background()

	// Collect metrics
	err := a.CollectMetrics(ctx)
	require.NoError(t, err)

	// Get metrics before reporting
	metricsBefore := a.GetMetrics()
	assert.NotEmpty(t, metricsBefore)

	// Report metrics (this should reset PollCount)
	metricsToReport := a.PrepareMetricsToReport()

	// Check that metrics were prepared for reporting
	assert.NotEmpty(t, metricsToReport)

	// Check that PollCount was included
	var foundPollCount bool
	for _, m := range metricsToReport {
		if m.Name == "PollCount" {
			foundPollCount = true
			break
		}
	}
	assert.True(t, foundPollCount, "PollCount metric should be included in report")

	// Check that PollCount was reset
	a.ResetPollCount()
	metricsAfter := a.GetMetrics()
	assert.Equal(t, float64(0), metricsAfter["PollCount"])
}

func TestAgent_PrepareMetricsToReport(t *testing.T) {
	// Create a test agent
	a := agent.New(nil)
	ctx := context.Background()

	// Collect metrics
	err := a.CollectMetrics(ctx)
	require.NoError(t, err)

	// Prepare metrics for reporting
	metricsToReport := a.PrepareMetricsToReport()

	// Check that metrics were prepared
	assert.NotEmpty(t, metricsToReport)

	// Check that all metrics have the correct format
	for _, m := range metricsToReport {
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

func TestAgent_Run(t *testing.T) {
	// Create a test agent with short intervals
	cfg := &agent.Config{
		PollInterval:   100 * time.Millisecond,
		ReportInterval: 200 * time.Millisecond,
	}
	a := agent.New(cfg)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Run the agent in a goroutine
	errCh := make(chan error)
	go func() {
		errCh <- a.Run(ctx)
	}()

	// Wait for the context to be done
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-ctx.Done():
		// Context timed out, which is expected
	}

	// Check that metrics were collected
	metrics := a.GetMetrics()
	assert.NotEmpty(t, metrics)
}
