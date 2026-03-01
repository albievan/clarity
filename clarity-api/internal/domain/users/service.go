package users

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/claims"
)

// PasswordHasher is the same interface as in auth domain; inject the shared implementation.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) bool
}

// defaultHasher mirrors stubHasher from auth — replace with bcrypt after go mod tidy.
type defaultHasher struct{}

func (defaultHasher) Hash(p string) (string, error) {
	h := sha256.Sum256([]byte(p))
	return "sha256:" + hex.EncodeToString(h[:]), nil
}
func (defaultHasher) Verify(hash, p string) bool {
	h := sha256.Sum256([]byte(p))
	return hash == "sha256:"+hex.EncodeToString(h[:])
}

// Service defines the business logic contract for the users domain.
type Service interface {
	List(ctx context.Context, tenantID, callerID string, f Filter, page, perPage int) ([]User, int, error)
	Get(ctx context.Context, tenantID, callerID, userID string) (*User, error)
	Create(ctx context.Context, tenantID, callerID string, req CreateRequest) (*User, error)
	Update(ctx context.Context, tenantID, callerID, userID string, req UpdateRequest) (*User, error)
	Deprovision(ctx context.Context, tenantID, callerID, userID string) error
	Lock(ctx context.Context, tenantID, callerID, userID string, req LockRequest) error
	Unlock(ctx context.Context, tenantID, callerID, userID string) error
	// Roles
	ListRoles(ctx context.Context, tenantID, callerID, userID string) ([]RoleAssignment, error)
	AssignRole(ctx context.Context, tenantID, callerID, userID string, req AssignRoleRequest) (*RoleAssignment, error)
	RevokeRole(ctx context.Context, tenantID, callerID, userID, assignmentID string) error
	// OAuth identities
	ListIdentities(ctx context.Context, tenantID, callerID, userID string) ([]OAuthIdentity, error)
	DeleteIdentity(ctx context.Context, tenantID, callerID, userID, identityID string) error
	// Internal — called by auth OAuth handlers
	FindOrCreateOAuthUser(ctx context.Context, tenantID string, id OAuthIdentity) (*User, bool, error)
}

type service struct {
	repo   Repository
	hasher PasswordHasher
}

func NewService(repo Repository, h ...PasswordHasher) Service {
	var hasher PasswordHasher = defaultHasher{}
	if len(h) > 0 {
		hasher = h[0]
	}
	return &service{repo: repo, hasher: hasher}
}

// ── Access helpers ────────────────────────────────────────────────────────────

func isAdmin(ctx context.Context) bool {
	return claims.HasRole(ctx, claims.RoleITAdmin, claims.RoleFinanceController)
}

// ── List / Get ────────────────────────────────────────────────────────────────

func (s *service) List(ctx context.Context, tenantID, callerID string, f Filter, page, perPage int) ([]User, int, error) {
	if !isAdmin(ctx) {
		return nil, 0, apierr.Forbidden("only admins may list users")
	}
	return s.repo.List(ctx, tenantID, f, page, perPage)
}

func (s *service) Get(ctx context.Context, tenantID, callerID, userID string) (*User, error) {
	if !isAdmin(ctx) && callerID != userID {
		return nil, apierr.Forbidden("you may only view your own profile")
	}
	u, err := s.repo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, apierr.NotFound("user")
	}
	return u, nil
}

// ── Create ────────────────────────────────────────────────────────────────────

func (s *service) Create(ctx context.Context, tenantID, callerID string, req CreateRequest) (*User, error) {
	if !isAdmin(ctx) {
		return nil, apierr.Forbidden("only admins may create users")
	}
	if req.Email == "" {
		return nil, apierr.BadRequest("email is required")
	}
	if req.Password == "" {
		return nil, apierr.BadRequest("password is required for local users")
	}
	if len(req.Password) < 12 {
		return nil, apierr.BadRequest("password must be at least 12 characters")
	}

	existing, err := s.repo.GetByEmail(ctx, tenantID, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apierr.Conflict("a user with this email already exists")
	}

	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("users.Create hash: %w", err)
	}

	displayName := req.FirstName + " " + req.LastName
	u := User{
		TenantID:    tenantID,
		Email:       req.Email,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		DisplayName: displayName,
		Status:      "active",
		AuthProvider: "local",
	}
	created, err := s.repo.Create(ctx, u, hash)
	if err != nil {
		return nil, fmt.Errorf("users.Create: %w", err)
	}

	// Assign requested roles (defaults to budget_requestor)
	roles := req.Roles
	if len(roles) == 0 {
		roles = []string{claims.RoleBudgetRequestor}
	}
	for _, role := range roles {
		if _, err := s.repo.AssignRole(ctx, tenantID, created.ID, role, callerID); err != nil {
			// Non-fatal — log and continue
			_ = err
		}
	}
	created.Roles = roles
	return created, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (s *service) Update(ctx context.Context, tenantID, callerID, userID string, req UpdateRequest) (*User, error) {
	if !isAdmin(ctx) && callerID != userID {
		return nil, apierr.Forbidden("you may only update your own profile")
	}
	u, err := s.repo.GetByID(ctx, tenantID, userID)
	if err != nil || u == nil {
		return nil, apierr.NotFound("user")
	}
	return s.repo.Update(ctx, tenantID, userID, req)
}

// ── Deprovision / Lock / Unlock ───────────────────────────────────────────────

func (s *service) Deprovision(ctx context.Context, tenantID, callerID, userID string) error {
	if !isAdmin(ctx) {
		return apierr.Forbidden("only admins may deprovision users")
	}
	if callerID == userID {
		return apierr.BadRequest("you cannot deprovision your own account")
	}
	return s.repo.Deprovision(ctx, tenantID, userID)
}

func (s *service) Lock(ctx context.Context, tenantID, callerID, userID string, req LockRequest) error {
	if !isAdmin(ctx) {
		return apierr.Forbidden("only admins may lock users")
	}
	if callerID == userID {
		return apierr.BadRequest("you cannot lock your own account")
	}
	var until *time.Time
	if req.DurationMinutes > 0 {
		t := time.Now().Add(time.Duration(req.DurationMinutes) * time.Minute)
		until = &t
	}
	return s.repo.Lock(ctx, tenantID, userID, until)
}

func (s *service) Unlock(ctx context.Context, tenantID, callerID, userID string) error {
	if !isAdmin(ctx) {
		return apierr.Forbidden("only admins may unlock users")
	}
	return s.repo.Unlock(ctx, tenantID, userID)
}

// ── Roles ─────────────────────────────────────────────────────────────────────

func (s *service) ListRoles(ctx context.Context, tenantID, callerID, userID string) ([]RoleAssignment, error) {
	if !isAdmin(ctx) && callerID != userID {
		return nil, apierr.Forbidden("insufficient permissions")
	}
	return s.repo.ListRoles(ctx, tenantID, userID)
}

func (s *service) AssignRole(ctx context.Context, tenantID, callerID, userID string, req AssignRoleRequest) (*RoleAssignment, error) {
	if !isAdmin(ctx) {
		return nil, apierr.Forbidden("only admins may assign roles")
	}
	if req.RoleName == "" {
		return nil, apierr.BadRequest("role_name is required")
	}
	return s.repo.AssignRole(ctx, tenantID, userID, req.RoleName, callerID)
}

func (s *service) RevokeRole(ctx context.Context, tenantID, callerID, userID, assignmentID string) error {
	if !isAdmin(ctx) {
		return apierr.Forbidden("only admins may revoke roles")
	}
	return s.repo.RevokeRole(ctx, tenantID, assignmentID)
}

// ── OAuth identities ──────────────────────────────────────────────────────────

func (s *service) ListIdentities(ctx context.Context, tenantID, callerID, userID string) ([]OAuthIdentity, error) {
	if !isAdmin(ctx) && callerID != userID {
		return nil, apierr.Forbidden("insufficient permissions")
	}
	return s.repo.ListOAuthIdentities(ctx, tenantID, userID)
}

func (s *service) DeleteIdentity(ctx context.Context, tenantID, callerID, userID, identityID string) error {
	if !isAdmin(ctx) && callerID != userID {
		return apierr.Forbidden("insufficient permissions")
	}
	// Prevent removing the only identity if no local password
	identities, err := s.repo.ListOAuthIdentities(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	user, err := s.repo.GetByID(ctx, tenantID, userID)
	if err != nil || user == nil {
		return apierr.NotFound("user")
	}
	if user.AuthProvider != "local" && len(identities) <= 1 {
		return apierr.BadRequest("cannot remove the only login method; set a local password first")
	}
	return s.repo.DeleteOAuthIdentity(ctx, tenantID, identityID)
}

// FindOrCreateOAuthUser is called by the OAuth callback handlers.
// It finds an existing user by OAuth identity, or creates a new one.
// Returns (user, isNewUser, error).
func (s *service) FindOrCreateOAuthUser(ctx context.Context, tenantID string, id OAuthIdentity) (*User, bool, error) {
	// Check if identity already exists
	existing, err := s.repo.FindOAuthIdentity(ctx, tenantID, id.Provider, id.ProviderUID)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		user, err := s.repo.GetByID(ctx, tenantID, existing.UserID)
		if err != nil {
			return nil, false, err
		}
		_ = s.repo.UpdateLastLogin(ctx, tenantID, existing.UserID)
		return user, false, nil
	}

	// Check if a local account with this email already exists — link it
	userByEmail, err := s.repo.GetByEmail(ctx, tenantID, id.Email)
	if err != nil {
		return nil, false, err
	}

	var user *User
	isNew := false

	if userByEmail != nil {
		user = userByEmail
	} else {
		// Create a new user for this OAuth identity
		newUser := User{
			TenantID:    tenantID,
			Email:       id.Email,
			FirstName:   id.DisplayName,
			LastName:    "",
			DisplayName: id.DisplayName,
			Status:      "active",
			AuthProvider: id.Provider,
			AvatarURL:   id.AvatarURL,
		}
		created, err := s.repo.Create(ctx, newUser, "")
		if err != nil {
			return nil, false, fmt.Errorf("users.FindOrCreateOAuthUser create: %w", err)
		}
		// Assign default role
		_, _ = s.repo.AssignRole(ctx, tenantID, created.ID, claims.RoleBudgetRequestor, "oauth")
		user = created
		isNew = true
	}

	// Link the OAuth identity
	id.UserID = user.ID
	id.TenantID = tenantID
	if _, err := s.repo.CreateOAuthIdentity(ctx, id); err != nil {
		return nil, isNew, fmt.Errorf("users.FindOrCreateOAuthUser link identity: %w", err)
	}

	_ = s.repo.UpdateLastLogin(ctx, tenantID, user.ID)
	return user, isNew, nil
}
