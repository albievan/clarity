package users

import "time"

// ── Core models ───────────────────────────────────────────────────────────────

// User is the canonical representation of a platform user.
type User struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"-"`
	Email         string     `json:"email"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	DisplayName   string     `json:"display_name"`
	Status        string     `json:"status"`         // active | locked | deprovisioned
	AuthProvider  string     `json:"auth_provider"`  // local | google | apple
	AvatarURL     string     `json:"avatar_url,omitempty"`
	Roles         []string   `json:"roles,omitempty"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// OAuthIdentity is a linked social login identity for a user.
// One user can have multiple identities (e.g. local + Google).
type OAuthIdentity struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	TenantID    string    `json:"-"`
	Provider    string    `json:"provider"`     // google | apple
	ProviderUID string    `json:"provider_uid"` // subject claim from the provider
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// RoleAssignment is a row from the user_roles join table.
type RoleAssignment struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	TenantID   string    `json:"-"`
	RoleName   string    `json:"role_name"`
	GrantedBy  string    `json:"granted_by"`
	GrantedAt  time.Time `json:"granted_at"`
}

// ── Filters & pagination ──────────────────────────────────────────────────────

// Filter controls which users are returned by List/Search.
type Filter struct {
	Search       string // searches email, first_name, last_name, display_name
	Status       string // active | locked | deprovisioned | "" (all)
	AuthProvider string // local | google | apple | "" (all)
	RoleName     string // return only users with this role
}

// ── HTTP request types ────────────────────────────────────────────────────────

// CreateRequest is the body for POST /users (local user creation).
type CreateRequest struct {
	Email       string   `json:"email"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	Password    string   `json:"password"`   // required for local users
	Roles       []string `json:"roles"`      // optional — defaults to budget_requestor
}

// UpdateRequest is the body for PUT /users/{userId}.
type UpdateRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// AssignRoleRequest is the body for POST /users/{userId}/roles.
type AssignRoleRequest struct {
	RoleName string `json:"role_name"`
}

// LockRequest is the body for POST /users/{userId}/lock.
type LockRequest struct {
	DurationMinutes int    `json:"duration_minutes"` // 0 = indefinite
	Reason          string `json:"reason"`
}
