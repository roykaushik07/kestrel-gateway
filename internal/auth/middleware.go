package auth

import (
	"context"
	"net/http"
)

// Middleware rejects requests without a valid bearer token and attaches the
// verified claims to the request context.
func (v *Verifier) Middleware(requiredScopes []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearer(r)
			if raw == "" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			claims, err := v.Verify(r.Context(), raw)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			if !claims.HasAnyScope(requiredScopes) {
				http.Error(w, "insufficient scope", http.StatusForbidden)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
