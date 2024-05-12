package ping_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/handlers/json/ping"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	config := &app.Config{
		Database: struct {
			DSN string
		}{
			DSN: "postgres://metrics:password@localhost:5432/metrics",
		},
	}
	log, _ := zap.NewDevelopmentConfig().Build()

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
		// {
		//	name:   "ping",
		//	target: "/ping",
		//	want: want{
		//		code:        200,
		//		contentType: "application/json",
		//	},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.target, nil)
			w := httptest.NewRecorder()
			ping.New(config, log)(w, request)

			res := w.Result()
			var body []byte
			_, err := res.Body.Read(body)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want.code, res.StatusCode, res.Body)

			defer res.Body.Close()

			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.NotEmpty(t, string(resBody))
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}
