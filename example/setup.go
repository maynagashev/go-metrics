// Package example содержит примеры использования API сервера метрик.
package example

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// ServerAddr - адрес сервера для тестов.
const ServerAddr = "http://localhost:8080"

// Время ожидания для сохранения метрики.
const saveDelay = 100 * time.Millisecond

// SetupTestMetric создает тестовую метрику для примеров.
func SetupTestMetric() error {
	metric := metrics.Metric{
		Name:  "TestGauge",
		MType: "gauge",
		Value: new(float64),
	}
	*metric.Value = 123.45

	body, _ := json.Marshal(metric)
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ServerAddr+"/update", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Даем время на сохранение метрики
	time.Sleep(saveDelay)
	return nil
}
