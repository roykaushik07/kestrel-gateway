package proxy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigRejectsUnknownUpstream(t *testing.T) {
	p := filepath.Join(t.TempDir(), "routes.yaml")
	body := "upstreams:\n  a: http://a:1\nroutes:\n  - prefix: /x\n    upstream: b\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(p); err == nil {
		t.Fatal("expected error for unknown upstream")
	}
}
