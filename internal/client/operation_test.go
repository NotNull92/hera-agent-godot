package client

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestMutationResponseLossDoesNotRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	_, err := New(server.URL).Post("operation", map[string]any{"action": "submit", "id": "session:1:counter"})
	if err == nil || calls.Load() != 1 {
		t.Fatalf("lost response: error=%v calls=%d, want error and exactly one request", err, calls.Load())
	}
}
