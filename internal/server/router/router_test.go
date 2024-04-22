package router_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maynagashev/go-metrics/internal/server/storage/memory"

	"github.com/maynagashev/go-metrics/internal/server/router"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRequest(
	t *testing.T,
	ts *httptest.Server,
	method, path string,
) (*http.Response, string, error) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	if err != nil {
		return nil, "", err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return resp, string(respBody), resp.Body.Close()
}

func TestNew(t *testing.T) {
	st := memory.New()
	st.UpdateGauge("test", 0.123)
	st.UpdateCounter("test", 5)

	ts := httptest.NewServer(router.New(st))
	defer ts.Close()

	var tests = []struct {
		url    string
		want   string
		status int
	}{
		{"/", "", http.StatusOK},

		{"/value/counter/test", "5", http.StatusOK},
		{"/value/gauge/test", "0.123", http.StatusOK},

		{"/value/counter/not_exist", "counter not_exist not found\n", http.StatusNotFound},
		{"/value/gauge/not_exist", "gauge not_exist not found\n", http.StatusNotFound},
	}

	for _, tt := range tests {
		resp, get, err := testRequest(t, ts, http.MethodGet, tt.url)
		require.NoError(t, err, "request failed")

		assert.Equal(t, tt.status, resp.StatusCode)
		if tt.want != "" {
			assert.Equal(t, tt.want, get)
		}

		err = resp.Body.Close()
		require.NoError(t, err, "failed to close response body")
	}
}
