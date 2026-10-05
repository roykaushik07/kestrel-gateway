package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareFailsOpenWithoutRedis(t *testing.T) {
	l := New("127.0.0.1:1") // nothing listens here
	h := l.Middleware("missions", 1, 1, func(*http.Request) string { return "u1" })(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected request to pass when redis is down, got %d", rec.Code)
	}
}
