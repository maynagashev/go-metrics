package update_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maynagashev/go-metrics/internal/server/storage"
	"github.com/maynagashev/go-metrics/internal/server/storage/memory"

	"github.com/maynagashev/go-metrics/internal/server/handlers/update"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

// [New]. Тест проверяет корректность обработки запроса на обновление метрики.
func TestUpdateHandler(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name    string
		target  string
		storage storage.Repository
		want
	}{
		{
			name:    "update gauge",
			target:  "/update/gauge/test_gauge/1.1",
			storage: memory.New(),
			want: want{
				code:        200,
				contentType: "text/plain",
			},
		},
		{
			name:    "update counter",
			target:  "/update/counter/test_counter/1",
			storage: memory.New(),
			want: want{
				code:        200,
				contentType: "text/plain",
			},
		},
		{
			name:    "invalid metrics type",
			target:  "/update/invalid/test_counter/1",
			storage: memory.New(),
			want: want{
				code:        400,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:    "invalid url",
			target:  "/update/gauge/1",
			storage: memory.New(),
			want: want{
				code:        404,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.target, nil)
			w := httptest.NewRecorder()
			update.New(tt.storage)(w, request)

			res := w.Result()
			assert.Equal(t, tt.want.code, res.StatusCode)

			defer res.Body.Close()

			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.NotEmpty(t, string(resBody))
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}
