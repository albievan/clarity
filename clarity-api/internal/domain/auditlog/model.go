package auditlog

import (
	"encoding/json"
	"time"
)

// AuditEntry is a single row from the audit_log table.
// This table is append-only — entries are never updated or deleted.
type AuditEntry struct {
	ID          string           `json:"id"`
	TenantID    string           `json:"-"`
	ActorUserID string           `json:"actor_user_id"`
	EntityType  string           `json:"entity_type"` // e.g. "budgets", "users"
	EntityID    string           `json:"entity_id"`
	Action      string           `json:"action"` // INSERT | UPDATE | DELETE | SUBMIT | APPROVE …
	BeforeState *json.RawMessage `json:"before_state,omitempty"`
	AfterState  *json.RawMessage `json:"after_state,omitempty"`
	IPAddress   string           `json:"ip_address"`
	UserAgent   string           `json:"user_agent"`
	CreatedAt   time.Time        `json:"created_at"`
}

// Filter holds the optional query parameters for the list endpoint.
type Filter struct {
	EntityType  string // filter by table name
	EntityID    string // filter by a specific record
	ActorUserID string // filter by who made the change
	Action      string // filter by action constant
	From        *time.Time
	To          *time.Time
}
