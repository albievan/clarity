package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/config"
	"github.com/albievan/clarity/clarity-api/internal/ctxkeys"
	"github.com/albievan/clarity/clarity-api/internal/response"
)

// RateLimiter is the interface required by the rate limit middleware.
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (remaining int, resetAt time.Time, allowed bool)
}

// RateLimit enforces per-tenant rate limits.
func RateLimit(limiter RateLimiter, cfg config.RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			if tid, ok := r.Context().Value(ctxkeys.TenantIDKey).(string); ok && tid != "" {
				key = fmt.Sprintf("rl:tenant:%s", tid)
			}

			remaining, resetAt, allowed := limiter.Allow(r.Context(), key, cfg.Requests, cfg.Window)

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Requests))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

			if !allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(time.Until(resetAt).Milliseconds()/1000+1, 10))
				response.Error(w, apierr.TooManyRequests("rate limit exceeded"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
