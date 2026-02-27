package auth

import (
	"context"
	"fmt"

	"github.com/albievan/clarity/clarity-api/internal/config"
)

// Service defines the business logic contract for the auth domain.
type Service interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	Logout(ctx context.Context, tenantID, sessionID string) error
	Refresh(ctx context.Context, refreshToken string) (*LoginResponse, error)
	MFASetup(ctx context.Context, tenantID, userID string) (*MFASetupResponse, error)
	MFAConfirm(ctx context.Context, tenantID, userID, code string) (*MFAConfirmResponse, error)
	MFAVerify(ctx context.Context, mfaToken, code string) (*LoginResponse, error)
	MFADisable(ctx context.Context, tenantID, userID, code string) error
	PasswordChange(ctx context.Context, tenantID, userID string, req PasswordChangeRequest) error
	PasswordResetRequest(ctx context.Context, email string) error
	PasswordReset(ctx context.Context, token, newPassword string) error
	ListSessions(ctx context.Context, tenantID, userID string) ([]Session, error)
	RevokeSession(ctx context.Context, tenantID, userID, sessionID string) error
}

// MFASetupResponse is the response for the MFA setup endpoint.
type MFASetupResponse struct {
	OTPAuthURI string `json:"otpauth_uri"`
	Secret     string `json:"secret"`
}

// MFAConfirmResponse contains the one-time backup codes shown on MFA activation.
type MFAConfirmResponse struct {
	BackupCodes []string `json:"backup_codes"`
}

// Session is a user's active session record.
type Session struct {
	ID         string `json:"id"`
	IPAddress  string `json:"ip_address"`
	UserAgent  string `json:"user_agent"`
	LastUsedAt string `json:"last_used_at"`
	IsCurrent  bool   `json:"is_current"`
}

type service struct {
	repo Repository
	cfg  config.JWTConfig
}

// NewService constructs the auth service.
func NewService(repo Repository, cfg config.JWTConfig) Service {
	return &service{repo: repo, cfg: cfg}
}

func (s *service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	// TODO:
	// 1. Look up user by email + tenant_id. Return apierr.Unauthorized if not found.
	// 2. Verify bcrypt password. Increment failed_login_count on failure.
	// 3. Check account lock status. Return apierr.AccountLocked if locked.
	// 4. If mfa_enabled: create pending session, return {mfa_required:true, mfa_token}.
	// 5. Create session row. Issue access + refresh JWTs via jwtutil.Sign.
	return nil, fmt.Errorf("auth.Login: not implemented")
}

func (s *service) Logout(ctx context.Context, tenantID, sessionID string) error {
	// TODO: revoke session row in DB.
	return fmt.Errorf("auth.Logout: not implemented")
}

func (s *service) Refresh(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// TODO: validate refresh token, rotate session, issue new pair.
	return nil, fmt.Errorf("auth.Refresh: not implemented")
}

func (s *service) MFASetup(ctx context.Context, tenantID, userID string) (*MFASetupResponse, error) {
	// TODO: generate TOTP secret, store in mfa_secret_pending.
	return nil, fmt.Errorf("auth.MFASetup: not implemented")
}

func (s *service) MFAConfirm(ctx context.Context, tenantID, userID, code string) (*MFAConfirmResponse, error) {
	// TODO: validate TOTP code against pending secret, activate MFA, generate backup codes.
	return nil, fmt.Errorf("auth.MFAConfirm: not implemented")
}

func (s *service) MFAVerify(ctx context.Context, mfaToken, code string) (*LoginResponse, error) {
	// TODO: validate mfa_token session, check TOTP/backup code, upgrade to full session.
	return nil, fmt.Errorf("auth.MFAVerify: not implemented")
}

func (s *service) MFADisable(ctx context.Context, tenantID, userID, code string) error {
	// TODO: validate TOTP code, check mfa_required policy, clear mfa_secret.
	return fmt.Errorf("auth.MFADisable: not implemented")
}

func (s *service) PasswordChange(ctx context.Context, tenantID, userID string, req PasswordChangeRequest) error {
	// TODO: verify current password, validate new password policy, update hash, revoke other sessions.
	return fmt.Errorf("auth.PasswordChange: not implemented")
}

func (s *service) PasswordResetRequest(ctx context.Context, email string) error {
	// TODO: look up user, generate token, send email. Always return nil (no enumeration).
	return nil
}

func (s *service) PasswordReset(ctx context.Context, token, newPassword string) error {
	// TODO: validate token, update password hash, revoke all sessions.
	return fmt.Errorf("auth.PasswordReset: not implemented")
}

func (s *service) ListSessions(ctx context.Context, tenantID, userID string) ([]Session, error) {
	// TODO: query sessions for user, mark is_current by session ID from context.
	return nil, fmt.Errorf("auth.ListSessions: not implemented")
}

func (s *service) RevokeSession(ctx context.Context, tenantID, userID, sessionID string) error {
	// TODO: verify ownership, set revoked_at.
	return fmt.Errorf("auth.RevokeSession: not implemented")
}
