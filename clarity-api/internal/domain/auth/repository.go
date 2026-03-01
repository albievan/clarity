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

// Repository is the data-access contract for the auth domain.
type Repository interface {
	// User lookups
	FindUserByEmail(ctx context.Context, tenantID, email string) (*UserRecord, error)
	FindUserByID(ctx context.Context, tenantID, userID string) (*UserRecord, error)
	IncrementFailedLogin(ctx context.Context, tenantID, userID string) error
	LockUser(ctx context.Context, tenantID, userID string, until *time.Time) error
	ResetFailedLogin(ctx context.Context, tenantID, userID string) error
	// Session management
	CreateSession(ctx context.Context, s SessionRecord) error
	GetSession(ctx context.Context, sessionID string) (*SessionRecord, error)
	UpdateSessionLastUsed(ctx context.Context, sessionID string) error
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeAllUserSessions(ctx context.Context, tenantID, userID string) error
	ListActiveSessions(ctx context.Context, tenantID, userID string) ([]SessionRecord, error)
	// MFA
	SetMFAPending(ctx context.Context, tenantID, userID, secret string) error
	ActivateMFA(ctx context.Context, tenantID, userID string) error
	DisableMFA(ctx context.Context, tenantID, userID string) error
	StoreBackupCodes(ctx context.Context, tenantID, userID string, codeHashes []string) error
	ValidateBackupCode(ctx context.Context, tenantID, userID, codeHash string) (bool, error)
	// Password
	UpdatePasswordHash(ctx context.Context, tenantID, userID, hash string) error
	// Password reset tokens
	CreatePasswordResetToken(ctx context.Context, rec PasswordResetRecord) error
	GetPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetRecord, error)
	MarkPasswordResetTokenUsed(ctx context.Context, tokenHash string) error
	// Security policy
	GetSecurityPolicy(ctx context.Context, tenantID string) (*SecurityPolicy, error)
}

type repository struct{ db *db.DB }

func NewRepository(database *db.DB) Repository { return &repository{db: database} }

// ─────────────────────────────────────────────────────────────────────────────
// NOTE ON SQL PLACEHOLDERS
// Queries use ? (MariaDB/MySQL driver name: "mysql").
// The go-mssqldb driver (SQL Server) also accepts ? — it maps them to @p1, @p2
// automatically in the current release. If you hit issues, replace ? with @p1,
// @p2, ... in each query when DB_DRIVER=sqlserver.
// ─────────────────────────────────────────────────────────────────────────────

func (r *repository) FindUserByEmail(ctx context.Context, tenantID, email string) (*UserRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, email, password_hash, status,
		       failed_login_count, locked_until,
		       mfa_enabled,
		       COALESCE(mfa_secret,''),
		       COALESCE(mfa_secret_pending,''),
		       require_pw_change
		FROM users
		WHERE tenant_id = ? AND email = ? AND status != 'deprovisioned'
		LIMIT 1`,
		tenantID, email,
	)
	return scanUser(row)
}

func (r *repository) FindUserByID(ctx context.Context, tenantID, userID string) (*UserRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, email, password_hash, status,
		       failed_login_count, locked_until,
		       mfa_enabled,
		       COALESCE(mfa_secret,''),
		       COALESCE(mfa_secret_pending,''),
		       require_pw_change
		FROM users
		WHERE tenant_id = ? AND id = ? AND status != 'deprovisioned'
		LIMIT 1`,
		tenantID, userID,
	)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*UserRecord, error) {
	var u UserRecord
	err := row.Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Status,
		&u.FailedLoginCount, &u.LockedUntil,
		&u.MFAEnabled, &u.MFASecret, &u.MFASecretPending,
		&u.RequirePWChange,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth.scanUser: %w", err)
	}
	return &u, nil
}

func (r *repository) IncrementFailedLogin(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET failed_login_count = failed_login_count + 1, updated_at = NOW()
		 WHERE tenant_id = ? AND id = ?`,
		tenantID, userID,
	)
	return err
}

func (r *repository) LockUser(ctx context.Context, tenantID, userID string, until *time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET status = 'locked', locked_until = ?, updated_at = NOW()
		 WHERE tenant_id = ? AND id = ?`,
		until, tenantID, userID,
	)
	return err
}

func (r *repository) ResetFailedLogin(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users
		 SET failed_login_count = 0, locked_until = NULL, status = 'active', updated_at = NOW()
		 WHERE tenant_id = ? AND id = ?`,
		tenantID, userID,
	)
	return err
}

func (r *repository) CreateSession(ctx context.Context, s SessionRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions
		  (id, user_id, tenant_id, token_hash, ip_address, user_agent, expires_at, last_used_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW())`,
		s.ID, s.UserID, s.TenantID, s.TokenHash, s.IPAddress, s.UserAgent, s.ExpiresAt,
	)
	return err
}

func (r *repository) GetSession(ctx context.Context, sessionID string) (*SessionRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, tenant_id, token_hash, ip_address, user_agent,
		       expires_at, revoked_at, last_used_at
		FROM sessions
		WHERE id = ? AND revoked_at IS NULL AND expires_at > NOW()
		LIMIT 1`,
		sessionID,
	)
	var s SessionRecord
	err := row.Scan(
		&s.ID, &s.UserID, &s.TenantID, &s.TokenHash,
		&s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.RevokedAt, &s.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth.GetSession: %w", err)
	}
	return &s, nil
}

func (r *repository) UpdateSessionLastUsed(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET last_used_at = NOW() WHERE id = ?`, sessionID,
	)
	return err
}

func (r *repository) RevokeSession(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET revoked_at = NOW() WHERE id = ?`, sessionID,
	)
	return err
}

func (r *repository) RevokeAllUserSessions(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET revoked_at = NOW()
		 WHERE tenant_id = ? AND user_id = ? AND revoked_at IS NULL`,
		tenantID, userID,
	)
	return err
}

func (r *repository) ListActiveSessions(ctx context.Context, tenantID, userID string) ([]SessionRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, tenant_id, token_hash, ip_address, user_agent,
		       expires_at, revoked_at, last_used_at
		FROM sessions
		WHERE tenant_id = ? AND user_id = ?
		  AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY last_used_at DESC`,
		tenantID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("auth.ListActiveSessions: %w", err)
	}
	defer rows.Close()
	var out []SessionRecord
	for rows.Next() {
		var s SessionRecord
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.TenantID, &s.TokenHash,
			&s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.RevokedAt, &s.LastUsedAt,
		); err != nil {
			return nil, fmt.Errorf("auth.ListActiveSessions scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *repository) SetMFAPending(ctx context.Context, tenantID, userID, secret string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET mfa_secret_pending = ?, updated_at = NOW()
		 WHERE tenant_id = ? AND id = ?`,
		secret, tenantID, userID,
	)
	return err
}

func (r *repository) ActivateMFA(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET mfa_enabled = 1, mfa_secret = mfa_secret_pending,
		    mfa_secret_pending = NULL, updated_at = NOW()
		WHERE tenant_id = ? AND id = ?`,
		tenantID, userID,
	)
	return err
}

func (r *repository) DisableMFA(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET mfa_enabled = 0, mfa_secret = NULL,
		    mfa_secret_pending = NULL, updated_at = NOW()
		WHERE tenant_id = ? AND id = ?`,
		tenantID, userID,
	)
	return err
}

// StoreBackupCodes replaces any existing backup codes for the user.
// Codes are stored as SHA-256 hashes; the plaintext is only shown once at setup.
func (r *repository) StoreBackupCodes(ctx context.Context, tenantID, userID string, codeHashes []string) error {
	// Delete existing codes first
	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM mfa_backup_codes WHERE tenant_id = ? AND user_id = ?`,
		tenantID, userID,
	); err != nil {
		return fmt.Errorf("auth.StoreBackupCodes delete: %w", err)
	}
	for _, h := range codeHashes {
		if _, err := r.db.ExecContext(ctx,
			`INSERT INTO mfa_backup_codes (id, tenant_id, user_id, code_hash) VALUES (?, ?, ?, ?)`,
			newID(), tenantID, userID, h,
		); err != nil {
			return fmt.Errorf("auth.StoreBackupCodes insert: %w", err)
		}
	}
	return nil
}

// ValidateBackupCode checks whether the hash exists and has not been used.
// If found, it marks the code as consumed.
func (r *repository) ValidateBackupCode(ctx context.Context, tenantID, userID, codeHash string) (bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id FROM mfa_backup_codes
		WHERE tenant_id = ? AND user_id = ? AND code_hash = ? AND used_at IS NULL
		LIMIT 1`,
		tenantID, userID, codeHash,
	)
	var id string
	if err := row.Scan(&id); err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("auth.ValidateBackupCode: %w", err)
	}
	// Consume the code
	_, err := r.db.ExecContext(ctx,
		`UPDATE mfa_backup_codes SET used_at = NOW() WHERE id = ?`, id,
	)
	return true, err
}

func (r *repository) UpdatePasswordHash(ctx context.Context, tenantID, userID, hash string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET password_hash = ?, require_pw_change = 0, updated_at = NOW()
		WHERE tenant_id = ? AND id = ?`,
		hash, tenantID, userID,
	)
	return err
}

func (r *repository) CreatePasswordResetToken(ctx context.Context, rec PasswordResetRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO password_reset_tokens (id, user_id, tenant_id, token_hash, expires_at)
		VALUES (?, ?, ?, ?, ?)`,
		rec.ID, rec.UserID, rec.TenantID, rec.TokenHash, rec.ExpiresAt,
	)
	return err
}

func (r *repository) GetPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, tenant_id, token_hash, expires_at, used_at
		FROM password_reset_tokens
		WHERE token_hash = ? AND used_at IS NULL AND expires_at > NOW()
		LIMIT 1`,
		tokenHash,
	)
	var rec PasswordResetRecord
	err := row.Scan(&rec.ID, &rec.UserID, &rec.TenantID, &rec.TokenHash, &rec.ExpiresAt, &rec.UsedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth.GetPasswordResetToken: %w", err)
	}
	return &rec, nil
}

func (r *repository) MarkPasswordResetTokenUsed(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE token_hash = ?`, tokenHash,
	)
	return err
}

func (r *repository) GetSecurityPolicy(ctx context.Context, tenantID string) (*SecurityPolicy, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT lockout_threshold, lockout_duration_mins, mfa_required,
		       min_password_length, password_history_count
		FROM security_policy
		WHERE tenant_id = ?
		LIMIT 1`,
		tenantID,
	)
	var p SecurityPolicy
	err := row.Scan(
		&p.LockoutThreshold, &p.LockoutDurationMins,
		&p.MFARequired, &p.MinPasswordLength, &p.PasswordHistoryCount,
	)
	if err == sql.ErrNoRows {
		// Safe defaults when no policy row has been configured yet
		return &SecurityPolicy{
			LockoutThreshold:     5,
			LockoutDurationMins:  30,
			MFARequired:          false,
			MinPasswordLength:    12,
			PasswordHistoryCount: 5,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth.GetSecurityPolicy: %w", err)
	}
	return &p, nil
}

// newID generates a random 32-hex-char ID (16 bytes of entropy).
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
