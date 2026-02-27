package audit

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/albievan/clarity/clarity-api/internal/claims"
)

// Action constants — match audit_log.action column values.
const (
	ActionInsert     = "INSERT"
	ActionUpdate     = "UPDATE"
	ActionDelete     = "DELETE"
	ActionSubmit     = "SUBMIT"
	ActionApprove    = "APPROVE"
	ActionReject     = "REJECT"
	ActionReturn     = "RETURN"
	ActionAmend      = "AMEND"
	ActionAssignRole = "ASSIGN_ROLE"
	ActionRevokeRole = "REVOKE_ROLE"
	ActionLock       = "LOCK"
	ActionUnlock     = "UNLOCK"
	ActionTPIConfirm = "TPI_CONFIRM"
	ActionBulkImport = "BULK_IMPORT"
	ActionReopen     = "REOPEN"
)

// Entry is a single audit log row.
type Entry struct {
	ID           string
	TenantID     string
	ActorUserID  string
	EntityType   string
	EntityID     string
	Action       string
	BeforeState  *json.RawMessage
	AfterState   *json.RawMessage
	IPAddress    string
	UserAgent    string
	CreatedAt    time.Time
}

// Writer is the interface that audit log writers implement.
type Writer interface {
	Write(ctx context.Context, tx *sql.Tx, e Entry) error
}

// Logger is the concrete audit log writer.
type Logger struct {
	db interface {
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	}
}

func New(db interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}) *Logger {
	return &Logger{db: db}
}

// Write appends a row to audit_log.
func (l *Logger) Write(ctx context.Context, tx *sql.Tx, e Entry) error {
	if e.ID == "" {
		e.ID = newID()
	}
	if e.ActorUserID == "" {
		e.ActorUserID = claims.UserID(ctx)
	}
	if e.TenantID == "" {
		e.TenantID = claims.TenantID(ctx)
	}
	e.CreatedAt = time.Now().UTC()

	var beforeJSON, afterJSON any
	if e.BeforeState != nil {
		beforeJSON = e.BeforeState
	}
	if e.AfterState != nil {
		afterJSON = e.AfterState
	}

	const q = `
		INSERT INTO audit_log
		  (id, tenant_id, actor_user_id, entity_type, entity_id, action,
		   before_state, after_state, ip_address, user_agent, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

	_, err := tx.ExecContext(ctx, q,
		e.ID, e.TenantID, e.ActorUserID, e.EntityType, e.EntityID, e.Action,
		beforeJSON, afterJSON, e.IPAddress, e.UserAgent, e.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("audit.Write: %w", err)
	}
	return nil
}

// Snapshot serialises a value to *json.RawMessage for before/after states.
func Snapshot(v any) *json.RawMessage {
	b, _ := json.Marshal(v)
	raw := json.RawMessage(b)
	return &raw
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
