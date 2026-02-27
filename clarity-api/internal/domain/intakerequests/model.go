package intakerequests

import (
	"time"
)

// BudgetIntakeRequests is the canonical model for the budget_intake_requests table.
// Add, remove or rename fields to match the actual schema columns.
type IntakeRequest struct {
	ID        string    `json:"id"       db:"id"`
	TenantID  string    `json:"-"        db:"tenant_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	// TODO: add domain-specific fields
}

// List filters for GET endpoints
type Filter struct {
	TenantID string
	Status   string
	// TODO: add domain-specific filter fields
}

// CreateRequest is the decoded request body for POST (create) endpoints.
type CreateRequest struct {
	// TODO: define required fields
}

// UpdateRequest is the decoded request body for PUT (update) endpoints.
type UpdateRequest struct {
	// TODO: define fields that may be updated
}
