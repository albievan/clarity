package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/ctxkeys"
	"github.com/albievan/clarity/clarity-api/internal/response"
)

// IdempotencyStore is the cache interface for storing idempotency responses.
type IdempotencyStore interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration)
}

type cachedResponse struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body"`
}

// Idempotency returns middleware that caches POST/PUT/DELETE responses keyed by
// (tenant_id + Idempotency-Key header) for the configured TTL.
func Idempotency(store IdempotencyStore, ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to mutating methods.
			if r.Method == http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			idKey := r.Header.Get("Idempotency-Key")
			if idKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			tid, _ := r.Context().Value(ctxkeys.TenantIDKey).(string)
			cacheKey := "idempotency:" + tid + ":" + idKey

			// Return cached response if present.
			if data, ok := store.Get(r.Context(), cacheKey); ok {
				var cached cachedResponse
				if err := json.Unmarshal(data, &cached); err == nil {
					for k, vals := range cached.Headers {
						for _, v := range vals {
							w.Header().Add(k, v)
						}
					}
					w.Header().Set("X-Idempotency-Replayed", "true")
					response.JSON(w, cached.Status, cached.Body)
					return
				}
			}

			// Validate the key hasn't been used concurrently.
			lockKey := "idempotency-lock:" + tid + ":" + idKey
			if _, exists := store.Get(r.Context(), lockKey); exists {
				response.Error(w, apierr.Conflict("idempotency key is currently being processed"))
				return
			}
			store.Set(r.Context(), lockKey, []byte("1"), 30*time.Second)

			// Capture the response.
			rec := &responseRecorder{ResponseWriter: w, buf: &bytes.Buffer{}, status: 200}
			next.ServeHTTP(rec, r)

			// Cache the response.
			cached := cachedResponse{
				Status:  rec.status,
				Headers: w.Header().Clone(),
				Body:    json.RawMessage(rec.buf.Bytes()),
			}
			if data, err := json.Marshal(cached); err == nil {
				store.Set(r.Context(), cacheKey, data, ttl)
			}
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	buf    *bytes.Buffer
	status int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	rr.buf.Write(b)
	return rr.ResponseWriter.Write(b)
}
