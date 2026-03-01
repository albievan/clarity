package auth

import "time"

// ── Database records ──────────────────────────────────────────────────────────

// UserRecord is the full user row as needed for authentication.
type UserRecord struct {
	ID               string
	TenantID         string
	Email            string
	PasswordHash     string
	Status           string // active | locked | deprovisioned
	FailedLoginCount int
	LockedUntil      *time.Time
	MFAEnabled       bool
	MFASecret        string
	MFASecretPending string
	RequirePWChange  bool
}

// SessionRecord is a row in the sessions table.
type SessionRecord struct {
	ID         string
	UserID     string
	TenantID   string
	TokenHash  string // SHA-256 of the refresh token (never store raw token)
	IPAddress  string
	UserAgent  string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	LastUsedAt time.Time
}

// PasswordResetRecord is a row in the password_reset_tokens table.
type PasswordResetRecord struct {
	ID        string
	UserID    string
	TenantID  string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// SecurityPolicy contains the per-tenant security settings needed by auth logic.
type SecurityPolicy struct {
	LockoutThreshold     int  // failed attempts before lock
	LockoutDurationMins  int  // how long the account is locked
	MFARequired          bool
	MinPasswordLength    int
	PasswordHistoryCount int
}

// ── HTTP request / response types ────────────────────────────────────────────

// LoginRequest is the body for POST /auth/login.
type LoginRequest struct {
	TenantID string `json:"tenant_id"` // tenant UUID, not name
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is returned by Login, Refresh and MFAVerify.
// Exactly one of the two branches is populated.
type LoginResponse struct {
	// Happy path
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	// MFA required branch
	MFARequired bool   `json:"mfa_required,omitempty"`
	MFAToken    string `json:"mfa_token,omitempty"` // short-lived JWT of type "mfa"
}

// RefreshRequest is the body for POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// MFASetupResponse is returned by POST /auth/mfa/setup.
type MFASetupResponse struct {
	OTPAuthURI string `json:"otpauth_uri"` // for QR code generation
	Secret     string `json:"secret"`      // base32 secret for manual entry
}

// MFAConfirmRequest is the body for POST /auth/mfa/confirm.
type MFAConfirmRequest struct {
	Code string `json:"code"` // 6-digit TOTP code from authenticator app
}

// MFAConfirmResponse is returned by POST /auth/mfa/confirm.
type MFAConfirmResponse struct {
	BackupCodes []string `json:"backup_codes"` // one-time use backup codes
}

// MFAVerifyRequest is the body for POST /auth/mfa/verify.
type MFAVerifyRequest struct {
	MFAToken string `json:"mfa_token"` // short-lived JWT issued at login
	Code     string `json:"code"`      // 6-digit TOTP or backup code
}

// MFADisableRequest is the body for POST /auth/mfa/disable.
type MFADisableRequest struct {
	Code string `json:"code"` // must confirm with current TOTP code
}

// PasswordChangeRequest is the body for POST /auth/password/change.
type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// PasswordResetRequestBody is the body for POST /auth/password/reset-request.
type PasswordResetRequestBody struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
}

// PasswordResetCompleteRequest is the body for POST /auth/password/reset.
type PasswordResetCompleteRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// Session is a summary of one active session, safe to return to the user.
type Session struct {
	ID         string `json:"id"`
	IPAddress  string `json:"ip_address"`
	UserAgent  string `json:"user_agent"`
	LastUsedAt string `json:"last_used_at"`
	IsCurrent  bool   `json:"is_current"` // true when this is the requesting session
}
