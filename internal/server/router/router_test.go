package router

import (
	"github.com/maynagashev/go-metrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testRequest(t *testing.T, ts *httptest.Server, method, path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestNew(t *testing.T) {
	st := storage.New()
	st.UpdateGauge("test", 0.123)
	st.UpdateCounter("test", 5)

	ts := httptest.NewServer(New(st))
	defer ts.Close()

	var tests = []struct {
		url    string
		want   string
		status int
	}{
		{"/value/counter/test", "5", http.StatusOK},
		{"/value/gauge/test", "0.123", http.StatusOK},

		{"/value/counter/not_exist", "counter not_exist not found\n", http.StatusNotFound},
		{"/value/gauge/not_exist", "gauge not_exist not found\n", http.StatusNotFound},
	}

	for _, tt := range tests {
		resp, get := testRequest(t, ts, http.MethodGet, tt.url)
		assert.Equal(t, tt.status, resp.StatusCode)
		assert.Equal(t, tt.want, get)
	}
}
