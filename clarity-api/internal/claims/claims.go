package claims

import (
	"context"
	"errors"

	"github.com/albievan/clarity/clarity-api/internal/ctxkeys"
)

// Claims holds the decoded JWT payload injected by the auth middleware.
type Claims struct {
	Subject   string   // user UUID
	TenantID  string   // tenant UUID
	Roles     []string // role names
	SessionID string   // session UUID (for logout)
}

func FromCtx(ctx context.Context) (*Claims, error) {
	c, ok := ctx.Value(ctxkeys.ClaimsKey).(*Claims)
	if !ok || c == nil {
		return nil, errors.New("claims not found in context")
	}
	return c, nil
}

func TenantID(ctx context.Context) string {
	c, _ := FromCtx(ctx)
	if c == nil {
		return ""
	}
	return c.TenantID
}

func UserID(ctx context.Context) string {
	c, _ := FromCtx(ctx)
	if c == nil {
		return ""
	}
	return c.Subject
}

// HasRole returns true if any of the provided role names are present in the claims.
func HasRole(ctx context.Context, roles ...string) bool {
	c, _ := FromCtx(ctx)
	if c == nil {
		return false
	}
	for _, want := range roles {
		for _, have := range c.Roles {
			if have == want {
				return true
			}
		}
	}
	return false
}

// Role constants mirror the roles table.
const (
	RoleITAdmin           = "it_admin"
	RoleBudgetOwner       = "budget_owner"
	RoleBudgetApprover    = "budget_approver"
	RoleDeptHead          = "dept_head"
	RoleFinanceController = "finance_controller"
	RoleBudgetRequestor   = "budget_requestor"
)
