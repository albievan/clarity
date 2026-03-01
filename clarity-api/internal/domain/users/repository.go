package users

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/albievan/clarity/clarity-api/internal/db"
)

// Repository defines the data-access contract for the users domain.
type Repository interface {
	// Local user CRUD
	List(ctx context.Context, tenantID string, f Filter, page, perPage int) ([]User, int, error)
	GetByID(ctx context.Context, tenantID, userID string) (*User, error)
	GetByEmail(ctx context.Context, tenantID, email string) (*User, error)
	Create(ctx context.Context, u User, passwordHash string) (*User, error)
	Update(ctx context.Context, tenantID, userID string, req UpdateRequest) (*User, error)
	Lock(ctx context.Context, tenantID, userID string, until *time.Time) error
	Unlock(ctx context.Context, tenantID, userID string) error
	Deprovision(ctx context.Context, tenantID, userID string) error
	// Role management
	ListRoles(ctx context.Context, tenantID, userID string) ([]RoleAssignment, error)
	AssignRole(ctx context.Context, tenantID, userID, roleName, grantedBy string) (*RoleAssignment, error)
	RevokeRole(ctx context.Context, tenantID, assignmentID string) error
	// OAuth identity management
	FindOAuthIdentity(ctx context.Context, tenantID, provider, providerUID string) (*OAuthIdentity, error)
	CreateOAuthIdentity(ctx context.Context, id OAuthIdentity) (*OAuthIdentity, error)
	DeleteOAuthIdentity(ctx context.Context, tenantID, identityID string) error
	ListOAuthIdentities(ctx context.Context, tenantID, userID string) ([]OAuthIdentity, error)
	// Internal: used by auth domain
	UpdateLastLogin(ctx context.Context, tenantID, userID string) error
}

type repository struct{ db *db.DB }

func NewRepository(database *db.DB) Repository { return &repository{db: database} }

// ── Helpers ───────────────────────────────────────────────────────────────────

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ── User CRUD ─────────────────────────────────────────────────────────────────

func (r *repository) List(ctx context.Context, tenantID string, f Filter, page, perPage int) ([]User, int, error) {
	where := []string{"u.tenant_id = ?"}
	args := []any{tenantID}

	if f.Search != "" {
		pattern := "%" + f.Search + "%"
		where = append(where,
			"(u.email LIKE ? OR u.first_name LIKE ? OR u.last_name LIKE ? OR u.display_name LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern)
	}
	if f.Status != "" {
		where = append(where, "u.status = ?")
		args = append(args, f.Status)
	}
	if f.AuthProvider != "" {
		where = append(where, "u.auth_provider = ?")
		args = append(args, f.AuthProvider)
	}
	if f.RoleName != "" {
		where = append(where, `EXISTS (
			SELECT 1 FROM user_roles ur
			WHERE ur.user_id = u.id AND ur.tenant_id = u.tenant_id AND ur.role_name = ?
		)`)
		args = append(args, f.RoleName)
	}

	clause := strings.Join(where, " AND ")

	// Count
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users u WHERE `+clause, countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("users.List count: %w", err)
	}

	// Data
	pageArgs := append(args, perPage, (page-1)*perPage)
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.tenant_id, u.email, u.first_name, u.last_name, u.display_name,
		       u.status, u.auth_provider, COALESCE(u.avatar_url,''),
		       u.last_login_at, u.created_at, u.updated_at
		FROM users u
		WHERE `+clause+`
		ORDER BY u.created_at DESC
		LIMIT ? OFFSET ?`,
		pageArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("users.List: %w", err)
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Attach roles
	for i := range out {
		roles, _ := r.ListRoles(ctx, tenantID, out[i].ID)
		for _, ra := range roles {
			out[i].Roles = append(out[i].Roles, ra.RoleName)
		}
	}

	return out, total, nil
}

func (r *repository) GetByID(ctx context.Context, tenantID, userID string) (*User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, email, first_name, last_name, display_name,
		       status, auth_provider, COALESCE(avatar_url,''),
		       last_login_at, created_at, updated_at
		FROM users WHERE tenant_id = ? AND id = ? LIMIT 1`,
		tenantID, userID,
	)
	u, err := scanUserRow(row)
	if err != nil {
		return nil, fmt.Errorf("users.GetByID: %w", err)
	}
	if u != nil {
		roles, _ := r.ListRoles(ctx, tenantID, u.ID)
		for _, ra := range roles {
			u.Roles = append(u.Roles, ra.RoleName)
		}
	}
	return u, nil
}

func (r *repository) GetByEmail(ctx context.Context, tenantID, email string) (*User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, email, first_name, last_name, display_name,
		       status, auth_provider, COALESCE(avatar_url,''),
		       last_login_at, created_at, updated_at
		FROM users WHERE tenant_id = ? AND email = ? LIMIT 1`,
		tenantID, email,
	)
	return scanUserRow(row)
}

func (r *repository) Create(ctx context.Context, u User, passwordHash string) (*User, error) {
	if u.ID == "" {
		u.ID = newID()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users
		  (id, tenant_id, email, first_name, last_name, display_name,
		   status, auth_provider, avatar_url, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.TenantID, u.Email, u.FirstName, u.LastName, u.DisplayName,
		u.Status, u.AuthProvider, u.AvatarURL, passwordHash, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("users.Create: %w", err)
	}
	u.CreatedAt = now
	u.UpdatedAt = now
	return &u, nil
}

func (r *repository) Update(ctx context.Context, tenantID, userID string, req UpdateRequest) (*User, error) {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET first_name = ?, last_name = ?,
		    display_name = CONCAT(?, ' ', ?),
		    updated_at = NOW()
		WHERE tenant_id = ? AND id = ?`,
		req.FirstName, req.LastName, req.FirstName, req.LastName, tenantID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("users.Update: %w", err)
	}
	return r.GetByID(ctx, tenantID, userID)
}

func (r *repository) Lock(ctx context.Context, tenantID, userID string, until *time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET status = 'locked', locked_until = ?, updated_at = NOW()
		WHERE tenant_id = ? AND id = ?`,
		until, tenantID, userID,
	)
	return err
}

func (r *repository) Unlock(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET status = 'active', locked_until = NULL,
		    failed_login_count = 0, updated_at = NOW()
		WHERE tenant_id = ? AND id = ?`,
		tenantID, userID,
	)
	return err
}

func (r *repository) Deprovision(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET status = 'deprovisioned', updated_at = NOW()
		WHERE tenant_id = ? AND id = ?`,
		tenantID, userID,
	)
	return err
}

func (r *repository) UpdateLastLogin(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE tenant_id = ? AND id = ?`,
		tenantID, userID,
	)
	return err
}

// ── Role management ───────────────────────────────────────────────────────────

func (r *repository) ListRoles(ctx context.Context, tenantID, userID string) ([]RoleAssignment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, tenant_id, role_name, granted_by, granted_at
		FROM user_roles
		WHERE tenant_id = ? AND user_id = ?
		ORDER BY granted_at`,
		tenantID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("users.ListRoles: %w", err)
	}
	defer rows.Close()
	var out []RoleAssignment
	for rows.Next() {
		var ra RoleAssignment
		if err := rows.Scan(&ra.ID, &ra.UserID, &ra.TenantID, &ra.RoleName, &ra.GrantedBy, &ra.GrantedAt); err != nil {
			return nil, fmt.Errorf("users.ListRoles scan: %w", err)
		}
		out = append(out, ra)
	}
	return out, rows.Err()
}

func (r *repository) AssignRole(ctx context.Context, tenantID, userID, roleName, grantedBy string) (*RoleAssignment, error) {
	ra := RoleAssignment{
		ID:        newID(),
		UserID:    userID,
		TenantID:  tenantID,
		RoleName:  roleName,
		GrantedBy: grantedBy,
		GrantedAt: time.Now().UTC(),
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_roles (id, user_id, tenant_id, role_name, granted_by, granted_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE granted_at = VALUES(granted_at)`,
		ra.ID, ra.UserID, ra.TenantID, ra.RoleName, ra.GrantedBy, ra.GrantedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("users.AssignRole: %w", err)
	}
	return &ra, nil
}

func (r *repository) RevokeRole(ctx context.Context, tenantID, assignmentID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM user_roles WHERE tenant_id = ? AND id = ?`, tenantID, assignmentID,
	)
	return err
}

// ── OAuth identity management ─────────────────────────────────────────────────

func (r *repository) FindOAuthIdentity(ctx context.Context, tenantID, provider, providerUID string) (*OAuthIdentity, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, tenant_id, provider, provider_uid,
		       email, COALESCE(display_name,''), COALESCE(avatar_url,''), created_at
		FROM oauth_identities
		WHERE tenant_id = ? AND provider = ? AND provider_uid = ?
		LIMIT 1`,
		tenantID, provider, providerUID,
	)
	var id OAuthIdentity
	err := row.Scan(
		&id.ID, &id.UserID, &id.TenantID, &id.Provider, &id.ProviderUID,
		&id.Email, &id.DisplayName, &id.AvatarURL, &id.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("users.FindOAuthIdentity: %w", err)
	}
	return &id, nil
}

func (r *repository) CreateOAuthIdentity(ctx context.Context, id OAuthIdentity) (*OAuthIdentity, error) {
	if id.ID == "" {
		id.ID = newID()
	}
	id.CreatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oauth_identities
		  (id, user_id, tenant_id, provider, provider_uid, email, display_name, avatar_url, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id.ID, id.UserID, id.TenantID, id.Provider, id.ProviderUID,
		id.Email, id.DisplayName, id.AvatarURL, id.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("users.CreateOAuthIdentity: %w", err)
	}
	return &id, nil
}

func (r *repository) DeleteOAuthIdentity(ctx context.Context, tenantID, identityID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM oauth_identities WHERE tenant_id = ? AND id = ?`, tenantID, identityID,
	)
	return err
}

func (r *repository) ListOAuthIdentities(ctx context.Context, tenantID, userID string) ([]OAuthIdentity, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, tenant_id, provider, provider_uid,
		       email, COALESCE(display_name,''), COALESCE(avatar_url,''), created_at
		FROM oauth_identities
		WHERE tenant_id = ? AND user_id = ?
		ORDER BY provider`,
		tenantID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("users.ListOAuthIdentities: %w", err)
	}
	defer rows.Close()
	var out []OAuthIdentity
	for rows.Next() {
		var id OAuthIdentity
		if err := rows.Scan(
			&id.ID, &id.UserID, &id.TenantID, &id.Provider, &id.ProviderUID,
			&id.Email, &id.DisplayName, &id.AvatarURL, &id.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ── Row scanner helpers ───────────────────────────────────────────────────────

type scannable interface {
	Scan(dest ...any) error
}

func scanUser(rows *sql.Rows) (*User, error) {
	var u User
	err := rows.Scan(
		&u.ID, &u.TenantID, &u.Email, &u.FirstName, &u.LastName, &u.DisplayName,
		&u.Status, &u.AuthProvider, &u.AvatarURL, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("users scan: %w", err)
	}
	return &u, nil
}

func scanUserRow(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID, &u.TenantID, &u.Email, &u.FirstName, &u.LastName, &u.DisplayName,
		&u.Status, &u.AuthProvider, &u.AvatarURL, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("users scanRow: %w", err)
	}
	return &u, nil
}
