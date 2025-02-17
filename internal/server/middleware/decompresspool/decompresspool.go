// Package decompresspool содержит middleware которое отвечает за обработку сжатых запросов,
// когда от клиента пришел заголовок Content-Encoding: gzip.
package decompresspool

import (
	"compress/gzip"
	"io"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

// Middleware содержит пулы для переиспользования объектов.
type Middleware struct {
	log        *zap.Logger
	readerPool sync.Pool
	closerPool sync.Pool
}

// New создает новый middleware для декомпрессии с пулами объектов.
func New(log *zap.Logger) func(next http.Handler) http.Handler {
	m := &Middleware{
		log: log,
		readerPool: sync.Pool{
			New: func() interface{} {
				return new(gzip.Reader)
			},
		},
		closerPool: sync.Pool{
			New: func() interface{} {
				return new(gzipReadCloser)
			},
		},
	}

	return m.Handler
}

// Handler возвращает функцию-обработчик запроса.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	m.log.Info("decompress middleware enabled")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			next.ServeHTTP(w, r)
			return
		}

		m.log.Debug("content encoded with gzip, replacing body with gzip.Reader")

		// Получаем Reader из пула
		reader, ok := m.readerPool.Get().(*gzip.Reader)
		if !ok {
			m.log.Error("error getting reader from pool")
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}

		// Инициализируем Reader новым источником данных
		if err := reader.Reset(r.Body); err != nil {
			m.log.Error("error initializing gzip reader", zap.Error(err))
			m.readerPool.Put(reader)
			http.Error(w, "Ошибка при декомпрессии содержимого запроса gzip", http.StatusBadRequest)
			return
		}

		// Получаем closer из пула
		closer, ok := m.closerPool.Get().(*gzipReadCloser)
		if !ok {
			m.log.Error("error getting closer from pool")
			m.readerPool.Put(reader)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}

		// Инициализируем closer
		closer.Reader = reader
		closer.middleware = m
		closer.log = m.log
		closer.originalBody = r.Body

		// Заменяем тело запроса
		r.Body = closer

		next.ServeHTTP(w, r)
	})
}

// gzipReadCloser оборачивает gzip.Reader для автоматического возврата в пул при закрытии.
type gzipReadCloser struct {
	*gzip.Reader
	middleware   *Middleware
	log          *zap.Logger
	originalBody io.ReadCloser
}

func (gz *gzipReadCloser) Close() error {
	// Закрываем gzip reader
	if err := gz.Reader.Close(); err != nil {
		gz.log.Error("error closing gzip reader", zap.Error(err))
		return err
	}

	// Закрываем оригинальное тело
	if err := gz.originalBody.Close(); err != nil {
		gz.log.Error("error closing original body", zap.Error(err))
		return err
	}

	// Возвращаем Reader в его пул
	gz.middleware.readerPool.Put(gz.Reader)

	// Очищаем поля перед возвратом в пул
	gz.Reader = nil
	gz.originalBody = nil
	gz.log = nil
	gz.middleware = nil

	// Возвращаем closer в его пул
	gz.middleware.closerPool.Put(gz)

	return nil
}
