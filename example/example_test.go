// Package example_test демонстрирует использование API сервера метрик в клиентском приложении.
package example_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/maynagashev/go-metrics/example"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
)

// Example_update демонстрирует обновление метрики через API.
func Example_update() {
	// Создаем метрику типа gauge.
	metric := metrics.Metric{
		Name:  "TestGauge",
		MType: "gauge",
		Value: new(float64),
	}
	*metric.Value = 123.45

	// Кодируем метрику в JSON.
	body, err := json.Marshal(metric)
	if err != nil {
		fmt.Printf("Ошибка кодирования метрики: %v\n", err)
		return
	}

	// Создаем запрос с контекстом
	ctx := context.Background()
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		example.ServerAddr+"/update",
		bytes.NewBuffer(body),
	)
	if err != nil {
		fmt.Printf("Ошибка создания запроса: %v\n", err)
		return
	}

	// Отправляем запрос
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Ошибка отправки запроса: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Статус ответа: %d\n", resp.StatusCode)
	// Output: Статус ответа: 200
}

// Example_getValue показывает как получить значение метрики.
func Example_getValue() {
	// Сначала создаем тестовую метрику
	if err := example.SetupTestMetric(); err != nil {
		fmt.Printf("Ошибка создания тестовой метрики: %v\n", err)
		return
	}

	// Создаем запрос на получение значения
	metric := metrics.Metric{
		Name:  "TestGauge",
		MType: "gauge",
	}

	body, err := json.Marshal(metric)
	if err != nil {
		fmt.Printf("Ошибка кодирования метрики: %v\n", err)
		return
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		example.ServerAddr+"/value",
		bytes.NewBuffer(body),
	)
	if err != nil {
		fmt.Printf("Ошибка создания запроса: %v\n", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Ошибка отправки запроса: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result metrics.Metric
	decodeErr := json.NewDecoder(resp.Body).Decode(&result)
	if decodeErr != nil {
		fmt.Printf("Ошибка декодирования ответа: %v\n", decodeErr)
		return
	}

	if result.Value == nil {
		fmt.Println("Значение метрики не найдено")
		return
	}

	fmt.Printf("Значение метрики %s: %f\n", result.Name, *result.Value)
	// Output: Значение метрики TestGauge: 123.450000
}

// Example_updateBatch демонстрирует пакетное обновление метрик.
func Example_updateBatch() {
	// Создаем набор метрик для пакетного обновления.
	metrics := []metrics.Metric{
		{
			Name:  "TestGauge1",
			MType: "gauge",
			Value: new(float64),
		},
		{
			Name:  "TestCounter1",
			MType: "counter",
			Delta: new(int64),
		},
	}
	*metrics[0].Value = 123.45
	*metrics[1].Delta = 42

	body, err := json.Marshal(metrics)
	if err != nil {
		fmt.Printf("Ошибка кодирования метрик: %v\n", err)
		return
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		example.ServerAddr+"/updates/",
		bytes.NewBuffer(body),
	)
	if err != nil {
		fmt.Printf("Ошибка создания запроса: %v\n", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Ошибка отправки запроса: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Статус ответа: %d\n", resp.StatusCode)
	// Output: Статус ответа: 200
}

// Example_ping демонстрирует проверку подключения к БД.
func Example_ping() {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, example.ServerAddr+"/ping", nil)
	if err != nil {
		fmt.Printf("Ошибка создания запроса: %v\n", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Ошибка отправки запроса: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Статус ответа: %d\n", resp.StatusCode)
	// Output: Статус ответа: 200
}
