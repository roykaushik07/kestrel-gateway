// Package auth validates bearer tokens issued by the identity provider.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type ctxKey struct{}

// Claims is the subset of token claims the gateway cares about.
type Claims struct {
	Subject string
	Scopes  map[string]bool
}

// Verifier validates JWTs against a cached remote JWKS.
type Verifier struct {
	cache    *jwk.Cache
	jwksURL  string
	audience string
}

// NewVerifier registers the JWKS endpoint and warms the key cache.
func NewVerifier(ctx context.Context, jwksURL, audience string) (*Verifier, error) {
	c := jwk.NewCache(ctx)
	if err := c.Register(jwksURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		return nil, fmt.Errorf("register jwks: %w", err)
	}
	if _, err := c.Refresh(ctx, jwksURL); err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	return &Verifier{cache: c, jwksURL: jwksURL, audience: audience}, nil
}

// Verify parses and validates a raw bearer token.
func (v *Verifier) Verify(ctx context.Context, raw string) (*Claims, error) {
	set, err := v.cache.Get(ctx, v.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("load keyset: %w", err)
	}
	tok, err := jwt.Parse([]byte(raw),
		jwt.WithKeySet(set),
		jwt.WithValidate(true),
		jwt.WithAudience(v.audience),
		jwt.WithAcceptableSkew(30*time.Second),
	)
	if err != nil {
		return nil, err
	}
	claims := &Claims{Subject: tok.Subject(), Scopes: map[string]bool{}}
	if raw, ok := tok.Get("scp"); ok {
		if list, ok := raw.([]interface{}); ok {
			for _, s := range list {
				if str, ok := s.(string); ok {
					claims.Scopes[str] = true
				}
			}
		}
	}
	return claims, nil
}

// FromContext returns the claims attached by Middleware.
func FromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(ctxKey{}).(*Claims)
	return c, ok
}

// HasAnyScope reports whether the caller holds at least one of the scopes.
func (c *Claims) HasAnyScope(scopes []string) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, s := range scopes {
		if c.Scopes[s] {
			return true
		}
	}
	return false
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return h[7:]
	}
	return ""
}
