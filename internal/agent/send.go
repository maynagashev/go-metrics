package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/pkg/crypto"
	"github.com/maynagashev/go-metrics/pkg/middleware/gzip"
	"github.com/maynagashev/go-metrics/pkg/sign"
)

const backoffFactor = 2

// Отправка очередного списка метрик из очереди на отправку, с помощью воркеров.
func (a *agent) sendMetrics(items []*metrics.Metric, workerID int) error {
	// Отправляем все метрики пачкой на новый маршрут /updates
	// Ошибки подключения при отправке метрик можно повторить, но не более 3-х раз (retriable errors)
	for i := 0; i <= maxSendRetries; i++ {
		// Пауза перед повторной отправкой
		if i > 0 {
			//nolint:gomnd // количество секунд для паузы зависит от номера попытки
			sleepSeconds := i*backoffFactor - 1 // 1, 3, 5, 7, 9, 11, ...
			slog.Info(
				fmt.Sprintf("retrying to send metrics (try=%d) in %d seconds", sleepSeconds, i),
				"workerID", workerID,
			)
			time.Sleep(time.Duration(sleepSeconds) * time.Second)
		}

		err := a.makeUpdatesRequest(items, i, workerID)
		// Если нет ошибок выходим из цикла и функции
		if err == nil {
			return nil
		}

		// Логируем ошибку
		slog.Error(
			fmt.Sprintf("failed to send metrics (try=%d): %s", i, err),
			"workerID",
			workerID,
			"metrics",
			items,
		)

		// Если ошибка не retriable, то выходим из цикла и функции, иначе продолжаем попытки
		if !a.isRetriableSendError(err) {
			slog.Debug("non-retriable error, stopping retries", "workerID", workerID, "err", err)
			return err
		}
	}

	return fmt.Errorf("failed to send metrics after %d retries", maxSendRetries)
}

// Проверяет, является ли ошибка временной и можно ли повторить запрос.
func (a *agent) isRetriableSendError(err error) bool {
	slog.Debug(fmt.Sprintf("isRetriableSendError: %#v", err))

	// Проверяем, является ли ошибка общей ошибкой сети, временной или таймаутом
	var netErr net.Error
	if errors.As(err, &netErr) {
		slog.Debug(fmt.Sprintf("isRetriableSendError => AS net.Error: %#v", netErr))
		// Проверяем, является ли ошибка временной
		if netErr.Timeout() {
			return true
		}
	}

	// Проверяем, является ли ошибка ошибкой сети
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		slog.Debug("isRetriableSendError => AS net.OpError", "err", err)
		return true
	}

	// Если ошибка не является временной, возвращаем false
	return false
}

// При ошибках подключения запрос можно повторить, но не более 3-х раз (retriable errors).
func (a *agent) makeUpdatesRequest(items []*metrics.Metric, try int, workerID int) error {
	var err error
	url := fmt.Sprintf("%s/updates", a.ServerURL)
	slog.Info(
		fmt.Sprintf("sending metrics batch (try=%d)", try),
		"workerID",
		workerID,
		"url",
		url,
		"metrics",
		items,
	)

	// Создаем новый запрос
	req := a.client.R()
	req.Debug = true // Включаем отладочный режим, чтобы видеть все детали запроса, в частности, использование сжатия
	req.SetHeader("Content-Type", "application/json")

	// Преобразуем метрики в JSON
	bytesBody, err := json.Marshal(items)
	if err != nil {
		return err
	}

	// Если задан приватный ключ, добавляем хэш в заголовок запроса
	if a.IsRequestSigningEnabled() {
		hash := sign.ComputeHMACSHA256(bytesBody, a.PrivateKey)
		req.SetHeader(sign.HeaderKey, hash)
	}

	// Если включено шифрование, шифруем данные перед отправкой
	if a.IsEncryptionEnabled() {
		slog.Debug("encrypting data before sending", "workerID", workerID)
		encryptedData, encryptErr := crypto.EncryptLargeData(a.PublicKey, bytesBody)
		if encryptErr != nil {
			return fmt.Errorf("failed to encrypt data: %w", encryptErr)
		}
		bytesBody = encryptedData
		req.SetHeader("Content-Encrypted", "true")
	}

	// Если включена сразу отправка сжатых данных, добавляем соответствующий заголовок
	// Go клиент автоматом также добавляет заголовок "Accept-Encoding: gzip"
	if a.SendCompressedData {
		req.SetHeader("Content-Encoding", "gzip")
		bytesBody, err = gzip.Compress(bytesBody)
		if err != nil {
			return err
		}
	}

	req.SetBody(bytesBody)

	res, err := req.Post(url)
	if err != nil {
		return err
	}

	// Обрабатываем ответ сервера
	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode())
	}

	return nil
}

// Вспомогательные функции для тестирования

// IsRetriableSendError - вспомогательная функция для тестирования,
// которая предоставляет доступ к методу isRetriableSendError.
func IsRetriableSendError(err error) bool {
	// Создаем минимальный экземпляр агента только для тестирования этой функции
	a := &agent{}
	return a.isRetriableSendError(err)
}

// SendMetrics - вспомогательная функция для тестирования, которая предоставляет доступ к методу makeUpdatesRequest.
func SendMetrics(a Agent, metrics []*metrics.Metric, workerID int) error {
	// Мы можем получить доступ к методам конкретного типа только через приведение типа
	if agentImpl, ok := a.(*agent); ok {
		return agentImpl.makeUpdatesRequest(metrics, 0, workerID)
	}
	return errors.New("invalid agent implementation")
}
