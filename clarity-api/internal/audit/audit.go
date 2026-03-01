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
	ActionLogin      = "LOGIN"
	ActionLogout     = "LOGOUT"
	ActionMFAEnable  = "MFA_ENABLE"
	ActionMFADisable = "MFA_DISABLE"
	ActionPWChange   = "PW_CHANGE"
	ActionPWReset    = "PW_RESET"
)

// Entry is a single audit log row to be written.
type Entry struct {
	ID          string
	TenantID    string
	ActorUserID string
	EntityType  string           // matches table name, e.g. "budgets", "users"
	EntityID    string           // the PK of the changed row
	Action      string           // one of the Action constants above
	BeforeState *json.RawMessage // serialised row before change; nil for inserts
	AfterState  *json.RawMessage // serialised row after change; nil for deletes
	IPAddress   string
	UserAgent   string
	CreatedAt   time.Time
}

// Writer is the interface that audit log writers implement.
// Passing a *sql.Tx keeps the write in the same database transaction
// as the business operation, so both succeed or fail together.
type Writer interface {
	Write(ctx context.Context, tx *sql.Tx, e Entry) error
}

// Logger is the concrete audit log writer.
type Logger struct{}

// New returns a Logger. The db parameter is accepted but not stored;
// all writes are performed on the provided *sql.Tx in Write().
func New(_ interface{}) *Logger { return &Logger{} }

// Write appends one row to audit_log inside the supplied transaction.
// Fields not supplied by the caller (ID, ActorUserID, TenantID, CreatedAt)
// are auto-populated from the context or generated here.
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

	// Serialise before/after state as JSON; pass nil when not applicable.
	var beforeJSON, afterJSON interface{}
	if e.BeforeState != nil {
		beforeJSON = []byte(*e.BeforeState)
	}
	if e.AfterState != nil {
		afterJSON = []byte(*e.AfterState)
	}

	// Uses ? placeholders — compatible with both MariaDB ("mysql" driver)
	// and SQL Server (go-mssqldb maps ? → @p1, @p2, ... automatically).
	_, err := tx.ExecContext(ctx, `
		INSERT INTO audit_log
		  (id, tenant_id, actor_user_id, entity_type, entity_id, action,
		   before_state, after_state, ip_address, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.TenantID, e.ActorUserID,
		e.EntityType, e.EntityID, e.Action,
		beforeJSON, afterJSON,
		e.IPAddress, e.UserAgent, e.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("audit.Write: %w", err)
	}
	return nil
}

// Snapshot serialises any value to *json.RawMessage for before/after states.
//
// Usage:
//
//	auditLogger.Write(ctx, tx, audit.Entry{
//	    EntityType:  "budgets",
//	    EntityID:    budget.ID,
//	    Action:      audit.ActionUpdate,
//	    BeforeState: audit.Snapshot(oldBudget),
//	    AfterState:  audit.Snapshot(newBudget),
//	})
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
