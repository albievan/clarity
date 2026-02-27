package middleware

import (
	"net/http"
	"strings"

	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/claims"
	"github.com/albievan/clarity/clarity-api/internal/ctxkeys"
	"github.com/albievan/clarity/clarity-api/internal/jwtutil"
	"github.com/albievan/clarity/clarity-api/internal/response"
)

// Auth validates the JWT Bearer token and injects Claims into the request context.
// Endpoints that do not require authentication must be registered on an un-protected sub-router.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			if !strings.HasPrefix(authorization, "Bearer ") {
				response.Error(w, apierr.Unauthorized("missing bearer token"))
				return
			}
			tokenStr := strings.TrimPrefix(authorization, "Bearer ")

			parsed, err := jwtutil.Parse(jwtSecret, tokenStr)
			if err != nil {
				response.Error(w, apierr.Unauthorized("invalid or expired token"))
				return
			}

			if parsed.TokenType != "access" {
				response.Error(w, apierr.Unauthorized("token type must be 'access'"))
				return
			}

			c := &claims.Claims{
				Subject:   parsed.Subject,
				TenantID:  parsed.TenantID,
				Roles:     parsed.Roles,
				SessionID: parsed.SessionID,
			}

			ctx := r.Context()
			ctx = ctxWith(ctx, ctxkeys.ClaimsKey, c)
			ctx = ctxWith(ctx, ctxkeys.TenantIDKey, parsed.TenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that enforces at least one of the given roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !claims.HasRole(r.Context(), roles...) {
				response.Error(w, apierr.Forbidden("insufficient role"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
