package jwtutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims extends standard JWT claims with Clarity-specific fields.
type CustomClaims struct {
	TenantID  string   `json:"tid"`
	Roles     []string `json:"roles"`
	SessionID string   `json:"sid"`
	TokenType string   `json:"type"` // "access" | "refresh" | "mfa"
	jwt.RegisteredClaims
}

// Sign creates a signed JWT string.
func Sign(secret string, claims CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// Parse validates and returns the claims from a JWT string.
func Parse(secret, tokenStr string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("parse jwt: %w", err)
	}
	c, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return c, nil
}

// NewAccessClaims builds the Claims for a short-lived access token.
func NewAccessClaims(userID, tenantID, sessionID string, roles []string, ttl time.Duration) CustomClaims {
	now := time.Now()
	return CustomClaims{
		TenantID:  tenantID,
		Roles:     roles,
		SessionID: sessionID,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
}

// NewRefreshClaims builds the Claims for a long-lived refresh token.
func NewRefreshClaims(userID, tenantID, sessionID string, ttl time.Duration) CustomClaims {
	now := time.Now()
	return CustomClaims{
		TenantID:  tenantID,
		TokenType: "refresh",
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
}
