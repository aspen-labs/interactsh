package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWriteResponseFromDynamicRequest(t *testing.T) {
	t.Run("status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/?status=404", nil)
		w := httptest.NewRecorder()
		writeResponseFromDynamicRequest(w, req)

		resp := w.Result()
		require.Equal(t, http.StatusNotFound, resp.StatusCode, "could not get correct result")
	})
	t.Run("delay", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/?delay=1", nil)
		w := httptest.NewRecorder()
		now := time.Now()
		writeResponseFromDynamicRequest(w, req)
		took := time.Since(now)

		require.Greater(t, took, 1*time.Second, "could not get correct delay")
	})
	t.Run("body", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/?body=this+is+example+body", nil)
		w := httptest.NewRecorder()
		writeResponseFromDynamicRequest(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		require.Equal(t, "this is example body", string(body), "could not get correct result")
	})

	t.Run("b64_body", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/?b64_body=dGhpcyBpcyBleGFtcGxlIGJvZHk=", nil)
		w := httptest.NewRecorder()
		writeResponseFromDynamicRequest(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		require.Equal(t, "this is example body", string(body), "could not get correct result")
	})
	t.Run("header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/?header=Key:value&header=Test:Another", nil)
		w := httptest.NewRecorder()
		writeResponseFromDynamicRequest(w, req)

		resp := w.Result()
		require.Equal(t, resp.Header.Get("Key"), "value", "could not get correct result")
		require.Equal(t, resp.Header.Get("Test"), "Another", "could not get correct result")
	})
}

func TestApidocsDynamicEndpoint(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	// Register new endpoint
	body := `{"body":"hello","content_type":"text/plain","suburl":"foo"}`
	req := httptest.NewRequest("POST", "/storerequest", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.Server.storeHandler(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Retrieve the endpoint
	req2 := httptest.NewRequest("GET", "/apidocs/foo", nil)
	w2 := httptest.NewRecorder()
	ts.Server.apidocsHandler(w2, req2)
	resp2 := w2.Result()
	out, _ := io.ReadAll(resp2.Body)
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	require.Equal(t, "hello", string(out))
	require.Equal(t, "text/plain", resp2.Header.Get("Content-Type"))

	// Register same suburl again within 24h should fail
	req3 := httptest.NewRequest("POST", "/storerequest", strings.NewReader(body))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	ts.Server.storeHandler(w3, req3)
	resp3 := w3.Result()
	require.Equal(t, http.StatusTooManyRequests, resp3.StatusCode)

	// Register different suburl should succeed
	body2 := `{"body":"world","content_type":"text/plain","suburl":"bar"}`
	req4 := httptest.NewRequest("POST", "/storerequest", strings.NewReader(body2))
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()
	ts.Server.storeHandler(w4, req4)
	resp4 := w4.Result()
	require.Equal(t, http.StatusOK, resp4.StatusCode)

	// Retrieve registered endpoint
	req5 := httptest.NewRequest("GET", "/apidocs/bar", nil)
	w5 := httptest.NewRecorder()
	ts.Server.apidocsHandler(w5, req5)
	resp5 := w5.Result()
	out, _ = io.ReadAll(resp5.Body)
	require.Equal(t, http.StatusOK, resp5.StatusCode)
	require.Equal(t, "world", string(out))
	require.Equal(t, "text/plain", resp5.Header.Get("Content-Type"))

	// Retrieve non-existent suburl
	req6 := httptest.NewRequest("GET", "/apidocs/notfound", nil)
	w6 := httptest.NewRecorder()
	ts.Server.apidocsHandler(w6, req6)
	resp6 := w6.Result()
	require.Equal(t, http.StatusNotFound, resp6.StatusCode)
}

// newTestServer returns a minimal HTTPServer with required fields for handler testing
func newTestServer() *struct {
	Server *HTTPServer
	Close  func()
} {
	h := &HTTPServer{}
	h.dynamicEndpoints = make(map[string]dynamicEndpoint)
	h.dynMu = sync.RWMutex{}
	return &struct {
		Server *HTTPServer
		Close  func()
	}{Server: h, Close: func() {}}
}
