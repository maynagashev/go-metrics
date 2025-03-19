// Package crypto предоставляет middleware для обработки зашифрованных запросов.
package crypto

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/pkg/crypto"
	"github.com/maynagashev/go-metrics/pkg/sign"
)

// Middleware представляет middleware для обработки шифрования и подписи запросов.
type Middleware struct {
	log              *zap.Logger
	config           *app.Config
	processedBodyKey ContextKey
}

// ContextKey - тип для ключей контекста.
type ContextKey struct {
	name string
}

// String возвращает строковое представление ключа контекста.
func (k ContextKey) String() string {
	return "crypto middleware context key: " + k.name
}

// New создает новый middleware для обработки шифрования и подписи запросов.
func New(config *app.Config, log *zap.Logger) func(http.Handler) http.Handler {
	m := &Middleware{
		log:              log,
		config:           config,
		processedBodyKey: ContextKey{"processed_body"},
	}

	return m.Handler
}

// Handler обрабатывает запрос, выполняя дешифрование и проверку подписи при необходимости.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, есть ли тело запроса
		if r.Body == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Читаем тело запроса
		body, err := io.ReadAll(r.Body)
		if err != nil {
			m.log.Error("failed to read request body", zap.Error(err))
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Обрабатываем шифрование и подпись
		processedBody, ok := m.processRequestBody(w, r, body)
		if !ok {
			return // Ошибка уже обработана в processRequestBody
		}

		// Заменяем тело запроса обработанными данными
		r.Body = io.NopCloser(bytes.NewReader(processedBody))

		// Сохраняем обработанное тело в контексте для возможного использования в обработчиках
		ctx := r.Context()
		ctx = context.WithValue(ctx, m.processedBodyKey, processedBody)
		r = r.WithContext(ctx)

		// Создаем обертку для ResponseWriter, чтобы добавить подпись к ответу
		if m.config.IsRequestSigningEnabled() {
			w = &signedResponseWriter{
				ResponseWriter: w,
				privateKey:     m.config.PrivateKey,
			}
		}

		next.ServeHTTP(w, r)
	})
}

// Возвращает обработанное тело и флаг успешности операции.
func (m *Middleware) processRequestBody(
	w http.ResponseWriter,
	r *http.Request,
	body []byte,
) ([]byte, bool) {
	// Обрабатываем шифрование
	processedBody, ok := m.handleEncryption(w, r, body)
	if !ok {
		return nil, false
	}

	// Проверяем подпись запроса
	if isValid := m.verifyRequestSignature(w, r, processedBody); !isValid {
		return nil, false
	}

	return processedBody, true
}

// Возвращает обработанное тело и флаг успешности операции.
func (m *Middleware) handleEncryption(
	w http.ResponseWriter,
	r *http.Request,
	body []byte,
) ([]byte, bool) {
	if r.Header.Get("Content-Encrypted") != "true" {
		return body, true
	}

	if !m.config.IsEncryptionEnabled() {
		m.log.Error("received encrypted data but server has no private key configured")
		http.Error(w, "Server is not configured for encryption", http.StatusBadRequest)
		return nil, false
	}

	m.log.Debug("decrypting request body")
	decrypted, err := crypto.DecryptLargeData(m.config.PrivateRSAKey, body)
	if err != nil {
		m.log.Error("failed to decrypt data", zap.Error(err))
		http.Error(w, "Failed to decrypt request body", http.StatusBadRequest)
		return nil, false
	}

	return decrypted, true
}

// Возвращает флаг успешности операции.
func (m *Middleware) verifyRequestSignature(
	w http.ResponseWriter,
	r *http.Request,
	body []byte,
) bool {
	if !m.config.IsRequestSigningEnabled() {
		return true
	}

	hashFromRequest := r.Header.Get(sign.HeaderKey)
	if hashFromRequest == "" {
		return true // Нет подписи, пропускаем проверку
	}

	hash, vErr := sign.VerifyHMACSHA256(body, m.config.PrivateKey, hashFromRequest)
	m.log.Debug(
		"validateRequest => sign.VerifyHMACSHA256",
		zap.String("hash_from_request", hashFromRequest),
		zap.Error(vErr),
		zap.String("calc_hash", hash),
	)
	if vErr != nil {
		m.log.Error("failed to verify request signature", zap.Error(vErr))
		http.Error(w, "Invalid request signature", http.StatusBadRequest)
		return false
	}

	return true
}

// signedResponseWriter - обертка для http.ResponseWriter, которая добавляет подпись к ответу.
type signedResponseWriter struct {
	http.ResponseWriter
	privateKey string
}

// Write перехватывает запись в ResponseWriter и добавляет подпись.
func (w *signedResponseWriter) Write(b []byte) (int, error) {
	// Вычисляем подпись
	hash := sign.ComputeHMACSHA256(b, w.privateKey)

	// Добавляем подпись в заголовок
	w.Header().Set(sign.HeaderKey, hash)

	// Записываем данные в оригинальный ResponseWriter
	return w.ResponseWriter.Write(b)
}
