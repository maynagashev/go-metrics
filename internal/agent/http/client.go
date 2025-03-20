package http

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/pkg/crypto"
	"github.com/maynagashev/go-metrics/pkg/middleware/gzip"
	"github.com/maynagashev/go-metrics/pkg/sign"
)

const (
	maxSendRetries = 3
	backoffFactor  = 2
)

// Client представляет HTTP клиент для отправки метрик.
type Client struct {
	serverURL          string
	client             *resty.Client
	privateKey         string
	cryptoKeyPath      string         // путь к файлу с публичным ключом
	publicKey          *rsa.PublicKey // загруженный публичный ключ
	sendCompressedData bool
}

// New создает новый HTTP клиент.
func New(serverURL, privateKey string, cryptoKeyPath string, realIP string) *Client {
	// Загружаем публичный ключ для шифрования, если указан путь
	var publicKey *rsa.PublicKey
	if cryptoKeyPath != "" {
		var err error
		publicKey, err = crypto.LoadPublicKey(cryptoKeyPath)
		if err != nil {
			slog.Error("failed to load public key", "error", err, "path", cryptoKeyPath)
		} else {
			slog.Info("loaded public key for encryption", "path", cryptoKeyPath)
		}
	}

	return &Client{
		serverURL:          serverURL,
		client:             initHTTPClient(realIP),
		privateKey:         privateKey,
		cryptoKeyPath:      cryptoKeyPath,
		publicKey:          publicKey,
		sendCompressedData: true, // согласно условиям задачи, отправка сжатых данных включена по умолчанию
	}
}

// Close ничего не делает, т.к. HTTP клиент не требует закрытия.
func (c *Client) Close() error {
	return nil
}

// UpdateMetric отправляет метрику на сервер.
func (c *Client) UpdateMetric(ctx context.Context, metric *metrics.Metric) error {
	return c.UpdateBatch(ctx, []*metrics.Metric{metric})
}

// UpdateBatch отправляет пакет метрик на сервер.
func (c *Client) UpdateBatch(ctx context.Context, metrics []*metrics.Metric) error {
	// Если метрик нет, ничего не делаем
	if len(metrics) == 0 {
		return nil
	}

	// Отправляем все метрики пачкой на маршрут /updates
	// Ошибки подключения при отправке метрик можно повторить, но не более maxSendRetries раз
	for i := 0; i <= maxSendRetries; i++ {
		// Проверяем, не отменен ли контекст
		if err := ctx.Err(); err != nil {
			return err
		}

		// Пауза перед повторной отправкой
		if i > 0 {
			sleepSeconds := i*backoffFactor - 1 // 1, 3, 5, 7, 9, 11, ...
			slog.Info(
				fmt.Sprintf("retrying to send metrics (try=%d) in %d seconds", i, sleepSeconds),
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(sleepSeconds) * time.Second):
				// Продолжаем выполнение
			}
		}

		err := c.makeUpdatesRequest(ctx, metrics, i)
		// Если нет ошибок выходим из цикла и функции
		if err == nil {
			return nil
		}

		// Логируем ошибку
		slog.Error(
			fmt.Sprintf("failed to send metrics (try=%d): %s", i, err),
			"metrics", metrics,
		)

		// Если ошибка не retriable, то выходим из цикла и функции, иначе продолжаем попытки
		if !isRetriableSendError(err) {
			slog.Debug("non-retriable error, stopping retries", "err", err)
			return err
		}
	}

	return errors.New("failed to send metrics after all retries")
}

// Ping проверяет соединение с сервером.
func (c *Client) Ping(ctx context.Context) error {
	req := c.client.R()
	url := fmt.Sprintf("%s/ping", c.serverURL)

	// Устанавливаем таймаут из контекста
	req.SetContext(ctx)

	res, err := req.Get(url)
	if err != nil {
		return err
	}

	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("ping failed with status code: %d", res.StatusCode())
	}

	return nil
}

// StreamMetrics отправляет метрики потоком на сервер.
// Для HTTP клиента этот метод не реализован и возвращает ошибку,
// так как потоковая передача поддерживается только в gRPC режиме.
func (c *Client) StreamMetrics(_ context.Context, _ []*metrics.Metric) error {
	return errors.New("streaming metrics is not supported in HTTP mode, use gRPC mode instead")
}

// isRequestSigningEnabled возвращает true, если задан приватный ключ и агент должен отправлять хэш на его основе.
func (c *Client) isRequestSigningEnabled() bool {
	return c.privateKey != ""
}

// isEncryptionEnabled возвращает true, если задан публичный ключ и агент должен шифровать данные.
func (c *Client) isEncryptionEnabled() bool {
	return c.publicKey != nil
}

// makeUpdatesRequest отправляет запрос на сервер для обновления метрик.
func (c *Client) makeUpdatesRequest(ctx context.Context, items []*metrics.Metric, try int) error {
	var err error
	url := fmt.Sprintf("%s/updates", c.serverURL)
	slog.Info(
		fmt.Sprintf("sending metrics batch (try=%d)", try),
		"url", url,
		"metrics", items,
	)

	// Создаем новый запрос
	req := c.client.R()
	req.Debug = true // Включаем отладочный режим, чтобы видеть все детали запроса, в частности, использование сжатия
	req.SetHeader("Content-Type", "application/json")
	req.SetContext(ctx)

	// Преобразуем метрики в JSON
	bytesBody, err := json.Marshal(items)
	if err != nil {
		return err
	}

	// Если задан приватный ключ, добавляем хэш в заголовок запроса
	if c.isRequestSigningEnabled() {
		hash := sign.ComputeHMACSHA256(bytesBody, c.privateKey)
		req.SetHeader(sign.HeaderKey, hash)
	}

	// Если включено шифрование, шифруем данные перед отправкой
	if c.isEncryptionEnabled() {
		slog.Debug("encrypting data before sending")
		encryptedData, encryptErr := crypto.EncryptLargeData(c.publicKey, bytesBody)
		if encryptErr != nil {
			return fmt.Errorf("failed to encrypt data: %w", encryptErr)
		}
		bytesBody = encryptedData
		req.SetHeader("Content-Encrypted", "true")
	}

	// Если включена сразу отправка сжатых данных, добавляем соответствующий заголовок
	// Go клиент автоматом также добавляет заголовок "Accept-Encoding: gzip"
	if c.sendCompressedData {
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
	statusCode := res.StatusCode()
	slog.Debug("received server response",
		"status_code", statusCode,
		"response_body", string(res.Body()))

	if statusCode != http.StatusOK {
		return fmt.Errorf("server returned non-OK response: %d", statusCode)
	}

	return nil
}

// isRetriableSendError проверяет, является ли ошибка возможной для повторной отправки.
func isRetriableSendError(err error) bool {
	if err == nil {
		return false
	}

	// Добавляем основные сетевые ошибки, которые можно повторить
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Добавляем конкретные ошибки, для которых можно делать повторную отправку
	var netOpErr *net.OpError
	if errors.As(err, &netOpErr) {
		return true
	}

	// Проверяем на различные DNS ошибки
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// Добавляем проверку на конкретные коды ошибок HTTP
	if errors.Is(err, net.ErrClosed) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Для любой Connection refused ошибки повторяем тоже
	if errors.Is(err, errors.New("connection refused")) {
		return true
	}

	// Добавляем ошибки TLS, которые могут быть временными
	if errors.Is(err, errors.New("TLS handshake timeout")) {
		return true
	}

	// По умолчанию считаем, что ошибку нельзя повторить
	return false
}

// initHTTPClient создает и настраивает HTTP-клиент с перехватчиком для установки заголовка X-Real-IP.
func initHTTPClient(realIP string) *resty.Client {
	client := resty.New().SetHeader("Content-Type", "text/plain")

	// Добавляем перехватчик для установки заголовка X-Real-IP
	client.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
		// Если указан явный IP-адрес, используем его
		if realIP != "" {
			req.SetHeader("X-Real-IP", realIP)
			slog.Debug("set X-Real-IP header (explicit)", "ip", realIP)
			return nil
		}

		// Иначе получаем исходящий IP-адрес автоматически
		hostIP, err := getOutboundIP()
		if err == nil {
			// Устанавливаем заголовок X-Real-IP
			req.SetHeader("X-Real-IP", hostIP.String())
			slog.Debug("set X-Real-IP header (auto-detected)", "ip", hostIP.String())
		} else {
			slog.Error("failed to set X-Real-IP header", "error", err)
		}
		return nil
	})

	return client
}

// getOutboundIP получает исходящий IP-адрес для текущего хоста.
func getOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("unexpected address type: %T", conn.LocalAddr())
	}
	return localAddr.IP, nil
}
