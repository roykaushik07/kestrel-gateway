package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/roykaushik07/kestrel-gateway/internal/auth"
	"github.com/roykaushik07/kestrel-gateway/internal/proxy"
	"github.com/roykaushik07/kestrel-gateway/internal/ratelimit"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func cors(allowed []string) func(http.Handler) http.Handler {
	set := map[string]bool{}
	for _, o := range allowed {
		set[strings.TrimSpace(o)] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if o := r.Header.Get("Origin"); set[o] {
				w.Header().Set("Access-Control-Allow-Origin", o)
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := proxy.LoadConfig(env("ROUTES_FILE", "config/routes.yaml"))
	if err != nil {
		slog.Error("load routes", "err", err)
		os.Exit(1)
	}
	verifier, err := auth.NewVerifier(ctx, os.Getenv("JWKS_URL"), env("JWT_AUDIENCE", "api://kestrel"))
	if err != nil {
		slog.Error("init auth", "err", err)
		os.Exit(1)
	}
	limiter := ratelimit.New(os.Getenv("REDIS_ADDR"))

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Use(cors(strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	subject := func(req *http.Request) string {
		if c, ok := auth.FromContext(req.Context()); ok {
			return c.Subject
		}
		return req.RemoteAddr
	}
	for _, rt := range cfg.Routes {
		rp, err := proxy.New(cfg.Upstreams[rt.Upstream])
		if err != nil {
			slog.Error("bad upstream", "route", rt.Prefix, "err", err)
			os.Exit(1)
		}
		r.Route(rt.Prefix, func(sub chi.Router) {
			sub.Use(verifier.Middleware(rt.Scopes))
			sub.Use(limiter.Middleware(rt.Prefix, rt.Rate.RPS, rt.Rate.Burst, subject))
			sub.Handle("/*", rp)
			sub.Handle("/", rp)
		})
	}

	hub := proxy.NewHub()
	go hub.Run(ctx, cfg.Upstreams[cfg.SSE.Upstream]+cfg.SSE.Path)
	r.With(verifier.Middleware([]string{"telemetry:read"})).
		Get("/v1/stream", hub.Handler(time.Duration(cfg.SSE.HeartbeatSeconds)*time.Second))

	srv := &http.Server{Addr: ":" + env("PORT", "8080"), Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	slog.Info("gateway listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
