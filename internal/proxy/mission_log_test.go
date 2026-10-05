package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogMissionStatusPreservesBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/missions/m1", nil)
	body := `{"id":"m1","status":"in_flight"}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
	if err := logMissionStatus(resp); err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	if string(got) != body {
		t.Fatalf("body altered: %s", got)
	}
}
