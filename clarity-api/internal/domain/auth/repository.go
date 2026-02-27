package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/albievan/clarity/clarity-api/internal/db"
)

// UserRecord is the DB row for authentication purposes.
type UserRecord struct {
	ID               string
	TenantID         string
	Email            string
	PasswordHash     string
	Status           string
	FailedLoginCount int
	LockedUntil      *time.Time
	MFAEnabled       bool
	MFASecret        string
	MFASecretPending string
	RequirePWChange  bool
}

// SessionRecord is the DB row for an active session.
type SessionRecord struct {
	ID         string
	UserID     string
	TenantID   string
	TokenHash  string
	IPAddress  string
	UserAgent  string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	LastUsedAt time.Time
}

// Repository is the data-access contract for the auth domain.
type Repository interface {
	FindUserByEmail(ctx context.Context, tenantID, email string) (*UserRecord, error)
	IncrementFailedLogin(ctx context.Context, tenantID, userID string) error
	LockUser(ctx context.Context, tenantID, userID string, until *time.Time) error
	ResetFailedLogin(ctx context.Context, tenantID, userID string) error
	CreateSession(ctx context.Context, s SessionRecord) error
	GetSession(ctx context.Context, sessionID string) (*SessionRecord, error)
	RevokeSession(ctx context.Context, sessionID string) error
	ListActiveSessions(ctx context.Context, tenantID, userID string) ([]SessionRecord, error)
	SetMFAPending(ctx context.Context, tenantID, userID, secret string) error
	ActivateMFA(ctx context.Context, tenantID, userID string) error
	DisableMFA(ctx context.Context, tenantID, userID string) error
	UpdatePasswordHash(ctx context.Context, tenantID, userID, hash string) error
	GetSecurityPolicy(ctx context.Context, tenantID string) (*SecurityPolicy, error)
}

// SecurityPolicy is a subset of the security_policy table needed by the auth service.
type SecurityPolicy struct {
	LockoutThreshold     int
	MFARequired          bool
	MinPasswordLength    int
	PasswordHistoryCount int
}

type repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) Repository {
	return &repository{db: database}
}

func (r *repository) FindUserByEmail(ctx context.Context, tenantID, email string) (*UserRecord, error) {
	// TODO: SELECT ... FROM users WHERE tenant_id = ? AND email = ? AND status != 'deprovisioned'
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, email, password_hash, status, failed_login_count,
		        locked_until, mfa_enabled, COALESCE(mfa_secret,''), COALESCE(mfa_secret_pending,''), require_pw_change
		 FROM users WHERE tenant_id = ? AND email = ? LIMIT 1`,
		tenantID, email,
	)
	var u UserRecord
	err := row.Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Status,
		&u.FailedLoginCount, &u.LockedUntil, &u.MFAEnabled, &u.MFASecret,
		&u.MFASecretPending, &u.RequirePWChange)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth.FindUserByEmail: %w", err)
	}
	return &u, nil
}

func (r *repository) IncrementFailedLogin(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET failed_login_count = failed_login_count + 1, updated_at = NOW()
		 WHERE tenant_id = ? AND id = ?`, tenantID, userID)
	return err
}

func (r *repository) LockUser(ctx context.Context, tenantID, userID string, until *time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET status = 'locked', locked_until = ?, updated_at = NOW()
		 WHERE tenant_id = ? AND id = ?`, until, tenantID, userID)
	return err
}

func (r *repository) ResetFailedLogin(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET failed_login_count = 0, locked_until = NULL, updated_at = NOW()
		 WHERE tenant_id = ? AND id = ?`, tenantID, userID)
	return err
}

func (r *repository) CreateSession(ctx context.Context, s SessionRecord) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, tenant_id, token_hash, ip_address, user_agent, expires_at, last_used_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, NOW())`,
		s.ID, s.UserID, s.TenantID, s.TokenHash, s.IPAddress, s.UserAgent, s.ExpiresAt)
	return err
}

func (r *repository) GetSession(ctx context.Context, sessionID string) (*SessionRecord, error) {
	// TODO: SELECT ... FROM sessions WHERE id = ? AND revoked_at IS NULL
	return nil, fmt.Errorf("GetSession: not implemented")
}

func (r *repository) RevokeSession(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET revoked_at = NOW() WHERE id = ?`, sessionID)
	return err
}

func (r *repository) ListActiveSessions(ctx context.Context, tenantID, userID string) ([]SessionRecord, error) {
	// TODO: SELECT ... FROM sessions WHERE tenant_id=? AND user_id=? AND revoked_at IS NULL AND expires_at > NOW()
	return nil, fmt.Errorf("ListActiveSessions: not implemented")
}

func (r *repository) SetMFAPending(ctx context.Context, tenantID, userID, secret string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET mfa_secret_pending = ?, updated_at = NOW() WHERE tenant_id = ? AND id = ?`,
		secret, tenantID, userID)
	return err
}

func (r *repository) ActivateMFA(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET mfa_enabled = true, mfa_secret = mfa_secret_pending, mfa_secret_pending = NULL, updated_at = NOW()
		 WHERE tenant_id = ? AND id = ?`, tenantID, userID)
	return err
}

func (r *repository) DisableMFA(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET mfa_enabled = false, mfa_secret = NULL, updated_at = NOW()
		 WHERE tenant_id = ? AND id = ?`, tenantID, userID)
	return err
}

func (r *repository) UpdatePasswordHash(ctx context.Context, tenantID, userID, hash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, require_pw_change = false, updated_at = NOW()
		 WHERE tenant_id = ? AND id = ?`, hash, tenantID, userID)
	return err
}

func (r *repository) GetSecurityPolicy(ctx context.Context, tenantID string) (*SecurityPolicy, error) {
	// TODO: SELECT lockout_threshold, mfa_required, min_password_length, password_history_count
	//       FROM security_policy WHERE tenant_id = ?
	return &SecurityPolicy{LockoutThreshold: 5, MinPasswordLength: 12, PasswordHistoryCount: 5}, nil
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
