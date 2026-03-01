package auditlog

import (
	"context"
	"fmt"
	"time"

	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/claims"
)

// Service defines the business logic for querying the audit log.
// The log is append-only — no create, update or delete methods are exposed.
type Service interface {
	List(ctx context.Context, tenantID, userID string, f Filter, page, perPage int) ([]AuditEntry, int, error)
	Get(ctx context.Context, tenantID, userID, id string) (*AuditEntry, error)
}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

// List returns a paginated, filtered view of the audit log.
// Only finance_controller and it_admin roles may query all entries.
// budget_owner and dept_head may only query entries for their own actions.
func (s *service) List(ctx context.Context, tenantID, userID string, f Filter, page, perPage int) ([]AuditEntry, int, error) {
	isAdmin := claims.HasRole(ctx, claims.RoleFinanceController, claims.RoleITAdmin)

	// Non-admin callers can only see their own audit entries
	if !isAdmin {
		f.ActorUserID = userID
	}

	entries, total, err := s.repo.List(ctx, tenantID, f, page, perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("auditlog.List: %w", err)
	}
	return entries, total, nil
}

// Get returns a single audit entry by ID.
func (s *service) Get(ctx context.Context, tenantID, userID, id string) (*AuditEntry, error) {
	entry, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("auditlog.Get: %w", err)
	}
	if entry == nil {
		return nil, apierr.NotFound("audit entry not found")
	}
	// Non-admins may only view their own entries
	isAdmin := claims.HasRole(ctx, claims.RoleFinanceController, claims.RoleITAdmin)
	if !isAdmin && entry.ActorUserID != userID {
		return nil, apierr.Forbidden("insufficient permissions to view this entry")
	}
	return entry, nil
}

// parseTime safely parses an optional time query param. Returns nil if empty.
func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
