package auditlog

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/albievan/clarity/clarity-api/internal/db"
)

// Repository defines the data-access contract for the auditlog domain.
// The audit_log table is append-only — only reads are exposed here.
// Writes go through internal/audit/audit.go inside transactions.
type Repository interface {
	List(ctx context.Context, tenantID string, f Filter, page, perPage int) ([]AuditEntry, int, error)
	GetByID(ctx context.Context, tenantID, id string) (*AuditEntry, error)
}

type repository struct{ db *db.DB }

func NewRepository(database *db.DB) Repository { return &repository{db: database} }

func (r *repository) List(ctx context.Context, tenantID string, f Filter, page, perPage int) ([]AuditEntry, int, error) {
	// Build a dynamic WHERE clause from the filter fields.
	// Placeholders use ? (MariaDB). For SQL Server swap to @p1, @p2, ...
	where := []string{"tenant_id = ?"}
	args := []any{tenantID}

	if f.EntityType != "" {
		where = append(where, "entity_type = ?")
		args = append(args, f.EntityType)
	}
	if f.EntityID != "" {
		where = append(where, "entity_id = ?")
		args = append(args, f.EntityID)
	}
	if f.ActorUserID != "" {
		where = append(where, "actor_user_id = ?")
		args = append(args, f.ActorUserID)
	}
	if f.Action != "" {
		where = append(where, "action = ?")
		args = append(args, f.Action)
	}
	if f.From != nil {
		where = append(where, "created_at >= ?")
		args = append(args, f.From)
	}
	if f.To != nil {
		where = append(where, "created_at <= ?")
		args = append(args, f.To)
	}

	clause := strings.Join(where, " AND ")

	// Count total rows
	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE `+clause, countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("auditlog.List count: %w", err)
	}

	// Fetch page
	args = append(args, perPage, (page-1)*perPage)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, actor_user_id, entity_type, entity_id, action,
		       before_state, after_state, ip_address, user_agent, created_at
		FROM audit_log
		WHERE `+clause+`
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`,
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("auditlog.List: %w", err)
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.ActorUserID,
			&e.EntityType, &e.EntityID, &e.Action,
			&e.BeforeState, &e.AfterState,
			&e.IPAddress, &e.UserAgent, &e.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("auditlog.List scan: %w", err)
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*AuditEntry, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, actor_user_id, entity_type, entity_id, action,
		       before_state, after_state, ip_address, user_agent, created_at
		FROM audit_log
		WHERE tenant_id = ? AND id = ?
		LIMIT 1`,
		tenantID, id,
	)
	var e AuditEntry
	err := row.Scan(
		&e.ID, &e.TenantID, &e.ActorUserID,
		&e.EntityType, &e.EntityID, &e.Action,
		&e.BeforeState, &e.AfterState,
		&e.IPAddress, &e.UserAgent, &e.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auditlog.GetByID: %w", err)
	}
	return &e, nil
}
