package proxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/roykaushik07/kestrel-gateway/internal/auth"
)

// New builds a reverse proxy for one upstream. The caller's subject is
// forwarded as X-Kestrel-Subject; the Authorization header is dropped so
// internal services never see the raw token.
func New(target string) (*httputil.ReverseProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	rp := httputil.NewSingleHostReverseProxy(u)
	orig := rp.Director
	rp.Director = func(r *http.Request) {
		orig(r)
		r.Header.Del("Authorization")
		if c, ok := auth.FromContext(r.Context()); ok {
			r.Header.Set("X-Kestrel-Subject", c.Subject)
		}
	}
	rp.Transport = &http.Transport{
		MaxIdleConnsPerHost:   64,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("upstream error", "path", r.URL.Path, "err", err)
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}
	return rp, nil
}
