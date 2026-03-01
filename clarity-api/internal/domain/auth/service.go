package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/config"
	"github.com/albievan/clarity/clarity-api/internal/jwtutil"
)

// ── Pluggable crypto interfaces ───────────────────────────────────────────────
//
// These interfaces keep the service compiling before `go mod tidy` resolves
// golang.org/x/crypto and github.com/pquerna/otp. After running `go mod tidy`,
// swap in the real implementations shown below.

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) bool
}

// TOTPProvider generates and validates TOTP secrets.
type TOTPProvider interface {
	Generate(accountName, issuer string) (secret, uri string, err error)
	Validate(secret, code string) bool
}

// ── Real implementations (uncomment after go mod tidy) ───────────────────────
//
// import (
//     "golang.org/x/crypto/bcrypt"
//     "github.com/pquerna/otp/totp"
// )
//
// type BcryptHasher struct{}
// func (BcryptHasher) Hash(p string) (string, error) {
//     b, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
//     return string(b), err
// }
// func (BcryptHasher) Verify(hash, p string) bool {
//     return bcrypt.CompareHashAndPassword([]byte(hash), []byte(p)) == nil
// }
//
// type TOTPImpl struct{}
// func (TOTPImpl) Generate(account, issuer string) (secret, uri string, err error) {
//     key, err := totp.Generate(totp.GenerateOpts{Issuer: issuer, AccountName: account})
//     if err != nil { return "", "", err }
//     return key.Secret(), key.URL(), nil
// }
// func (TOTPImpl) Validate(secret, code string) bool { return totp.Validate(code, secret) }
//
// Then in your main / wire-up replace NewService with:
//   auth.NewServiceWith(repo, cfg, auth.BcryptHasher{}, auth.TOTPImpl{})

// ── Stub implementations (compile-time only) ─────────────────────────────────

// stubHasher uses SHA-256 so the service is runnable for development.
// It is NOT safe for production — replace with BcryptHasher before launch.
type stubHasher struct{}

func (stubHasher) Hash(password string) (string, error) {
	h := sha256.Sum256([]byte(password))
	return "sha256:" + hex.EncodeToString(h[:]), nil
}
func (stubHasher) Verify(hash, password string) bool {
	h := sha256.Sum256([]byte(password))
	return hash == "sha256:"+hex.EncodeToString(h[:])
}

// stubTOTP generates real base32 secrets and URIs, but Validate always succeeds
// in dev mode (accepts any 6-digit code). Replace with TOTPImpl before launch.
type stubTOTP struct{ devMode bool }

func (t stubTOTP) Generate(accountName, issuer string) (secret, uri string, err error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("totp generate: %w", err)
	}
	secret = strings.TrimRight(base32.StdEncoding.EncodeToString(b), "=")
	uri = fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		issuer, accountName, secret, issuer)
	return secret, uri, nil
}
func (t stubTOTP) Validate(secret, code string) bool {
	if t.devMode {
		// In dev mode accept any 6-digit numeric code so you can test MFA flows
		// without a real authenticator app
		return len(code) == 6 && isDigits(code)
	}
	return false
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ── Service ───────────────────────────────────────────────────────────────────

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
	PasswordResetRequest(ctx context.Context, tenantID, email string) error
	PasswordReset(ctx context.Context, token, newPassword string) error
	ListSessions(ctx context.Context, tenantID, userID, currentSessionID string) ([]Session, error)
	RevokeSession(ctx context.Context, tenantID, userID, sessionID string) error
}

type service struct {
	repo   Repository
	cfg    config.JWTConfig
	hasher PasswordHasher
	totp   TOTPProvider
}

// NewService constructs the auth service with development-safe stub crypto.
// Replace with NewServiceWith(repo, cfg, BcryptHasher{}, TOTPImpl{}) for production.
func NewService(repo Repository, cfg config.JWTConfig) Service {
	return &service{
		repo:   repo,
		cfg:    cfg,
		hasher: stubHasher{},
		totp:   stubTOTP{devMode: true},
	}
}

// NewServiceWith allows injecting production-grade crypto implementations.
func NewServiceWith(repo Repository, cfg config.JWTConfig, h PasswordHasher, t TOTPProvider) Service {
	return &service{repo: repo, cfg: cfg, hasher: h, totp: t}
}

// ── Login ─────────────────────────────────────────────────────────────────────

func (s *service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	if req.TenantID == "" || req.Email == "" || req.Password == "" {
		return nil, apierr.BadRequest("tenant_id, email and password are required")
	}

	user, err := s.repo.FindUserByEmail(ctx, req.TenantID, req.Email)
	if err != nil {
		return nil, err
	}
	// Identical error for missing user and wrong password to prevent enumeration.
	invalidCreds := apierr.Unauthorized("invalid credentials")
	if user == nil {
		return nil, invalidCreds
	}

	// Check account lock — re-check expiry in case lock has since lifted
	if user.Status == "locked" {
		if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
			return nil, apierr.AccountLocked(
				fmt.Sprintf("account locked until %s", user.LockedUntil.UTC().Format(time.RFC3339)),
			)
		}
		// Lock window has passed — reset silently and continue
		_ = s.repo.ResetFailedLogin(ctx, req.TenantID, user.ID)
		user.Status = "active"
	}

	// Verify password
	if !s.hasher.Verify(user.PasswordHash, req.Password) {
		_ = s.repo.IncrementFailedLogin(ctx, req.TenantID, user.ID)
		policy, _ := s.repo.GetSecurityPolicy(ctx, req.TenantID)
		if policy != nil && user.FailedLoginCount+1 >= policy.LockoutThreshold {
			lockUntil := time.Now().Add(time.Duration(policy.LockoutDurationMins) * time.Minute)
			_ = s.repo.LockUser(ctx, req.TenantID, user.ID, &lockUntil)
		}
		return nil, invalidCreds
	}

	// Successful credential check — reset counter and lock state
	_ = s.repo.ResetFailedLogin(ctx, req.TenantID, user.ID)

	// MFA gate: issue a short-lived mfa-type JWT; client must call /auth/mfa/verify
	if user.MFAEnabled {
		mfaClaims := jwtutil.NewAccessClaims(user.ID, req.TenantID, "", nil, 10*time.Minute)
		mfaClaims.TokenType = "mfa"
		mfaToken, err := jwtutil.Sign(s.cfg.Secret, mfaClaims)
		if err != nil {
			return nil, fmt.Errorf("auth.Login sign mfa token: %w", err)
		}
		return &LoginResponse{MFARequired: true, MFAToken: mfaToken}, nil
	}

	return s.issueTokenPair(ctx, user)
}

// issueTokenPair creates a session row and returns an access + refresh JWT pair.
// The refresh token is an opaque random value embedded inside a signed JWT.
// Only a SHA-256 hash of that value is stored in the sessions table.
func (s *service) issueTokenPair(ctx context.Context, user *UserRecord) (*LoginResponse, error) {
	sessionID := newID()
	rawRefresh := newID() + newID() // 64-char opaque value
	refreshHash := hashToken(rawRefresh)

	if err := s.repo.CreateSession(ctx, SessionRecord{
		ID:        sessionID,
		UserID:    user.ID,
		TenantID:  user.TenantID,
		TokenHash: refreshHash,
		ExpiresAt: time.Now().Add(s.cfg.RefreshTTL),
	}); err != nil {
		return nil, fmt.Errorf("auth.issueTokenPair create session: %w", err)
	}

	accessToken, err := jwtutil.Sign(s.cfg.Secret,
		jwtutil.NewAccessClaims(user.ID, user.TenantID, sessionID, nil, s.cfg.AccessTTL),
	)
	if err != nil {
		return nil, fmt.Errorf("auth.issueTokenPair sign access: %w", err)
	}

	// Embed rawRefresh in the SessionID field of the refresh JWT so we can
	// look it up and verify it on the next Refresh call.
	refreshToken, err := jwtutil.Sign(s.cfg.Secret,
		jwtutil.NewRefreshClaims(user.ID, user.TenantID, rawRefresh, s.cfg.RefreshTTL),
	)
	if err != nil {
		return nil, fmt.Errorf("auth.issueTokenPair sign refresh: %w", err)
	}

	return &LoginResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// ── Logout ────────────────────────────────────────────────────────────────────

func (s *service) Logout(ctx context.Context, tenantID, sessionID string) error {
	return s.repo.RevokeSession(ctx, sessionID)
}

// ── Refresh ───────────────────────────────────────────────────────────────────

func (s *service) Refresh(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	c, err := jwtutil.Parse(s.cfg.Secret, refreshToken)
	if err != nil {
		return nil, apierr.Unauthorized("invalid refresh token")
	}
	if c.TokenType != "refresh" {
		return nil, apierr.Unauthorized("not a refresh token")
	}

	// The raw opaque value is in SessionID (see issueTokenPair)
	rawRefresh := c.SessionID
	tokenHash := hashToken(rawRefresh)

	session, err := s.repo.GetSession(ctx, c.Subject) // subject = userID in refresh token
	if err != nil {
		return nil, err
	}
	if session == nil || session.TokenHash != tokenHash {
		return nil, apierr.Unauthorized("refresh token not found or expired")
	}

	// Rotate: revoke old session, issue new pair
	_ = s.repo.RevokeSession(ctx, session.ID)
	return s.issueTokenPair(ctx, &UserRecord{ID: session.UserID, TenantID: session.TenantID})
}

// ── MFA setup ─────────────────────────────────────────────────────────────────

func (s *service) MFASetup(ctx context.Context, tenantID, userID string) (*MFASetupResponse, error) {
	user, err := s.repo.FindUserByID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, apierr.NotFound("user")
	}
	if user.MFAEnabled {
		return nil, apierr.Conflict("MFA is already enabled; disable it first")
	}

	secret, uri, err := s.totp.Generate(user.Email, "Clarity")
	if err != nil {
		return nil, fmt.Errorf("auth.MFASetup generate: %w", err)
	}
	if err := s.repo.SetMFAPending(ctx, tenantID, userID, secret); err != nil {
		return nil, fmt.Errorf("auth.MFASetup store pending: %w", err)
	}
	return &MFASetupResponse{OTPAuthURI: uri, Secret: secret}, nil
}

// ── MFA confirm ───────────────────────────────────────────────────────────────

func (s *service) MFAConfirm(ctx context.Context, tenantID, userID, code string) (*MFAConfirmResponse, error) {
	user, err := s.repo.FindUserByID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, apierr.NotFound("user")
	}
	if user.MFASecretPending == "" {
		return nil, apierr.BadRequest("no MFA setup in progress; call POST /auth/mfa/setup first")
	}
	if !s.totp.Validate(user.MFASecretPending, code) {
		return nil, apierr.BadRequest("invalid TOTP code")
	}

	if err := s.repo.ActivateMFA(ctx, tenantID, userID); err != nil {
		return nil, fmt.Errorf("auth.MFAConfirm activate: %w", err)
	}

	// Generate 8 one-time backup codes; show plaintext once, store hashes
	plainCodes := make([]string, 8)
	hashes := make([]string, 8)
	for i := range plainCodes {
		b := make([]byte, 5)
		_, _ = rand.Read(b)
		plainCodes[i] = hex.EncodeToString(b) // 10-char hex code
		hashes[i] = hashToken(plainCodes[i])
	}
	if err := s.repo.StoreBackupCodes(ctx, tenantID, userID, hashes); err != nil {
		return nil, fmt.Errorf("auth.MFAConfirm store backup codes: %w", err)
	}

	return &MFAConfirmResponse{BackupCodes: plainCodes}, nil
}

// ── MFA verify (completes login after MFA gate) ───────────────────────────────

func (s *service) MFAVerify(ctx context.Context, mfaToken, code string) (*LoginResponse, error) {
	c, err := jwtutil.Parse(s.cfg.Secret, mfaToken)
	if err != nil || c.TokenType != "mfa" {
		return nil, apierr.Unauthorized("invalid MFA token")
	}

	user, err := s.repo.FindUserByID(ctx, c.TenantID, c.Subject)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.MFAEnabled {
		return nil, apierr.Unauthorized("MFA not configured")
	}

	// Accept either a live TOTP code or a one-time backup code
	totpValid := s.totp.Validate(user.MFASecret, code)
	if !totpValid {
		// Try backup code path
		codeHash := hashToken(code)
		used, err := s.repo.ValidateBackupCode(ctx, c.TenantID, c.Subject, codeHash)
		if err != nil {
			return nil, err
		}
		if !used {
			return nil, apierr.Unauthorized("invalid MFA code")
		}
	}

	return s.issueTokenPair(ctx, user)
}

// ── MFA disable ───────────────────────────────────────────────────────────────

func (s *service) MFADisable(ctx context.Context, tenantID, userID, code string) error {
	policy, err := s.repo.GetSecurityPolicy(ctx, tenantID)
	if err != nil {
		return err
	}
	if policy.MFARequired {
		return apierr.Forbidden("tenant policy requires MFA; it cannot be disabled")
	}

	user, err := s.repo.FindUserByID(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return apierr.NotFound("user")
	}
	if !user.MFAEnabled {
		return apierr.BadRequest("MFA is not enabled")
	}
	if !s.totp.Validate(user.MFASecret, code) {
		return apierr.Unauthorized("invalid TOTP code")
	}

	return s.repo.DisableMFA(ctx, tenantID, userID)
}

// ── Password change ───────────────────────────────────────────────────────────

func (s *service) PasswordChange(ctx context.Context, tenantID, userID string, req PasswordChangeRequest) error {
	user, err := s.repo.FindUserByID(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return apierr.NotFound("user")
	}

	if !s.hasher.Verify(user.PasswordHash, req.CurrentPassword) {
		return apierr.Unauthorized("current password is incorrect")
	}
	if req.NewPassword == req.CurrentPassword {
		return apierr.BadRequest("new password must differ from current password")
	}

	policy, err := s.repo.GetSecurityPolicy(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(req.NewPassword) < policy.MinPasswordLength {
		return apierr.BadRequest(fmt.Sprintf("password must be at least %d characters", policy.MinPasswordLength))
	}

	hash, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		return fmt.Errorf("auth.PasswordChange hash: %w", err)
	}
	if err := s.repo.UpdatePasswordHash(ctx, tenantID, userID, hash); err != nil {
		return fmt.Errorf("auth.PasswordChange update: %w", err)
	}
	// Revoke all sessions so the old password can no longer be used for refresh
	return s.repo.RevokeAllUserSessions(ctx, tenantID, userID)
}

// ── Password reset ────────────────────────────────────────────────────────────

func (s *service) PasswordResetRequest(ctx context.Context, tenantID, email string) error {
	// Always return nil — never reveal whether the email or tenant exists.
	user, err := s.repo.FindUserByEmail(ctx, tenantID, email)
	if err != nil || user == nil {
		return nil
	}

	rawToken := newID() + newID() // 64-char opaque value
	rec := PasswordResetRecord{
		ID:        newID(),
		UserID:    user.ID,
		TenantID:  tenantID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	_ = s.repo.CreatePasswordResetToken(ctx, rec)

	// TODO: wire in an email service and send the reset link:
	//   emailSvc.Send(user.Email, "Password Reset", baseURL+"/reset?token="+rawToken)
	// The rawToken is intentionally not logged.
	return nil
}

func (s *service) PasswordReset(ctx context.Context, token, newPassword string) error {
	tokenHash := hashToken(token)
	rec, err := s.repo.GetPasswordResetToken(ctx, tokenHash)
	if err != nil {
		return err
	}
	if rec == nil {
		return apierr.BadRequest("invalid or expired reset token")
	}

	policy, err := s.repo.GetSecurityPolicy(ctx, rec.TenantID)
	if err != nil {
		return err
	}
	if len(newPassword) < policy.MinPasswordLength {
		return apierr.BadRequest(fmt.Sprintf("password must be at least %d characters", policy.MinPasswordLength))
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("auth.PasswordReset hash: %w", err)
	}
	if err := s.repo.UpdatePasswordHash(ctx, rec.TenantID, rec.UserID, hash); err != nil {
		return err
	}
	_ = s.repo.MarkPasswordResetTokenUsed(ctx, tokenHash)
	_ = s.repo.RevokeAllUserSessions(ctx, rec.TenantID, rec.UserID)
	return nil
}

// ── Sessions ──────────────────────────────────────────────────────────────────

func (s *service) ListSessions(ctx context.Context, tenantID, userID, currentSessionID string) ([]Session, error) {
	rows, err := s.repo.ListActiveSessions(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Session, 0, len(rows))
	for _, r := range rows {
		out = append(out, Session{
			ID:         r.ID,
			IPAddress:  r.IPAddress,
			UserAgent:  r.UserAgent,
			LastUsedAt: r.LastUsedAt.UTC().Format(time.RFC3339),
			IsCurrent:  r.ID == currentSessionID,
		})
	}
	return out, nil
}

func (s *service) RevokeSession(ctx context.Context, tenantID, userID, sessionID string) error {
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if sess == nil {
		return apierr.NotFound("session")
	}
	if sess.UserID != userID || sess.TenantID != tenantID {
		return apierr.Forbidden("cannot revoke another user's session")
	}
	return s.repo.RevokeSession(ctx, sessionID)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// hashToken returns the hex SHA-256 digest of a raw token string.
// Tokens are never stored in plain text.
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
