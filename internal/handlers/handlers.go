package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func Update(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Request: %s %s\n", r.Method, r.URL)

	w.Header().Set("Content-Type", "text/plain")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Получаем части пути из URL /update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
	parts := strings.Split(r.URL.Path, "/")
	fmt.Printf("path: %s, len: %d, parts: %#v\n", r.URL.Path, len(parts), parts)
	if len(parts) != 5 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	metricType := parts[2]
	metricName := strings.TrimSpace(parts[3])
	metricValue := parts[4]

	// При попытке передать запрос с некорректным типом метрики или значением возвращать http.StatusBadRequest.
	if metricType != "counter" && metricType != "gauge" {
		http.Error(w, "Invalid metric type, must be: counter or gauge", http.StatusBadRequest)
		return
	}

	// Проверяем, что значение метрики является числом float64
	if _, err := strconv.ParseFloat(metricValue, 64); err != nil {
		http.Error(w, "Invalid metric value, must be convertable to float64", http.StatusBadRequest)
		return
	}

	// При попытке передать запрос без имени метрики возвращать http.StatusNotFound.
	if metricName == "" {
		http.Error(w, "Empty metric name", http.StatusNotFound)
		return
	}

	// Здесь вы можете обновить вашу метрику с полученными значениями
	// Например, можно сохранить эти значения в базе данных или просто вывести их
	fmt.Printf("Received metric update: Type=%s, Name=%s, Value=%s\n", metricType, metricName, metricValue)

	// Отправляем успешный ответ
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Metric %s/%s updated with value %s", metricType, metricName, metricValue)

}
