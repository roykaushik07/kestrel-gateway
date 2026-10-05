// Package ratelimit implements a Redis-backed token bucket.
package ratelimit

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// The bucket is updated atomically in Lua so concurrent gateway replicas
// share one view of each caller's budget.
var script = redis.NewScript(`
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local data = redis.call('HMGET', key, 'tokens', 'ts')
local tokens = tonumber(data[1]) or burst
local ts = tonumber(data[2]) or now
tokens = math.min(burst, tokens + (now - ts) * rate / 1000)
local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end
redis.call('HSET', key, 'tokens', tokens, 'ts', now)
redis.call('PEXPIRE', key, math.ceil(burst / rate * 1000) + 1000)
return allowed
`)

// Limiter checks callers against per-route budgets.
type Limiter struct {
	rdb *redis.Client
}

func New(addr string) *Limiter {
	return &Limiter{rdb: redis.NewClient(&redis.Options{Addr: addr})}
}

// Allow consumes one token for key. It fails open if Redis is unreachable so
// a cache outage never takes the API down.
func (l *Limiter) Allow(ctx context.Context, key string, rps, burst int) bool {
	now := time.Now().UnixMilli()
	res, err := script.Run(ctx, l.rdb, []string{"rl:" + key}, rps, burst, now).Int()
	if err != nil {
		return true
	}
	return res == 1
}

// Middleware limits by caller subject (supplied by keyFn) for a route.
func (l *Limiter) Middleware(routeName string, rps, burst int, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.Allow(r.Context(), routeName+":"+keyFn(r), rps, burst) {
				w.Header().Set("Retry-After", strconv.Itoa(1))
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
