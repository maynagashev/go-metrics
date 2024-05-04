// Package decompress содержит middleware которое отвечает за обработку сжатых запросов,
// когда от клиента пришел заголовок Content-Encoding: gzip.
package decompress

import (
	"compress/gzip"
	"net/http"

	"go.uber.org/zap"
)

func New(log *zap.Logger) func(next http.Handler) http.Handler {
	// Возвращаем функцию, которая принимает следующий обработчик
	return func(next http.Handler) http.Handler {
		log.Info("decompress middleware enabled")

		// Функция-обработчик запроса
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Проверяем заголовок Content-Encoding: gzip
			if r.Header.Get("Content-Encoding") == "gzip" {
				log.Debug("content encoded with gzip, replacing body with gzip.Reader")

				gz, err := gzip.NewReader(r.Body)
				if err != nil {
					log.Error("error while decompressing request body", zap.Error(err))
					http.Error(w, "Ошибка при декомпрессии содержимого запроса gzip", http.StatusBadRequest)
					return
				}

				defer func() {
					err = gz.Close()
					if err != nil {
						log.Error("error while closing decompression stream", zap.Error(err))
						http.Error(w, "Ошибка при закрытии потока декомпрессии", http.StatusInternalServerError)
						return
					}
				}()

				// Заменяем тело запроса на декомпрессированный поток
				r.Body = gz
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
