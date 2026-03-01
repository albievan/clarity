package users

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/claims"
	"github.com/albievan/clarity/clarity-api/internal/pagination"
	"github.com/albievan/clarity/clarity-api/internal/response"
)

// Handler holds HTTP handlers for the users domain.
type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// List handles GET /v1/users
// Query params: ?search=alice&status=active&auth_provider=google&role=budget_owner&page=1&per_page=25
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	q := r.URL.Query()
	f := Filter{
		Search:       q.Get("search"),
		Status:       q.Get("status"),
		AuthProvider: q.Get("auth_provider"),
		RoleName:     q.Get("role"),
	}
	p := pagination.Parse(r)
	items, total, err := h.svc.List(r.Context(), c.TenantID, c.Subject, f, p.Page, p.PerPage)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.PageOf(w, items, p.Page, p.PerPage, total)
}

// Get handles GET /v1/users/{userId}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	userID := chi.URLParam(r, "userId")
	u, err := h.svc.Get(r.Context(), c.TenantID, c.Subject, userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, u)
}

// Create handles POST /v1/users
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	u, err := h.svc.Create(r.Context(), c.TenantID, c.Subject, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, u)
}

// Update handles PUT /v1/users/{userId}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	userID := chi.URLParam(r, "userId")
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	u, err := h.svc.Update(r.Context(), c.TenantID, c.Subject, userID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, u)
}

// Deprovision handles DELETE /v1/users/{userId} (soft delete — status → deprovisioned)
func (h *Handler) Deprovision(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	userID := chi.URLParam(r, "userId")
	if err := h.svc.Deprovision(r.Context(), c.TenantID, c.Subject, userID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// Lock handles POST /v1/users/{userId}/lock
func (h *Handler) Lock(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	userID := chi.URLParam(r, "userId")
	var req LockRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body is optional
	if err := h.svc.Lock(r.Context(), c.TenantID, c.Subject, userID, req); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// Unlock handles POST /v1/users/{userId}/unlock
func (h *Handler) Unlock(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	userID := chi.URLParam(r, "userId")
	if err := h.svc.Unlock(r.Context(), c.TenantID, c.Subject, userID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ── Role handlers ─────────────────────────────────────────────────────────────

// ListRoles handles GET /v1/users/{userId}/roles
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	userID := chi.URLParam(r, "userId")
	roles, err := h.svc.ListRoles(r.Context(), c.TenantID, c.Subject, userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, roles)
}

// AssignRole handles POST /v1/users/{userId}/roles
func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	userID := chi.URLParam(r, "userId")
	var req AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	ra, err := h.svc.AssignRole(r.Context(), c.TenantID, c.Subject, userID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, ra)
}

// RevokeRole handles DELETE /v1/users/{userId}/roles/{assignmentId}
func (h *Handler) RevokeRole(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	userID := chi.URLParam(r, "userId")
	assignmentID := chi.URLParam(r, "assignmentId")
	if err := h.svc.RevokeRole(r.Context(), c.TenantID, c.Subject, userID, assignmentID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ── OAuth identity handlers ───────────────────────────────────────────────────

// ListIdentities handles GET /v1/users/{userId}/identities
func (h *Handler) ListIdentities(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	userID := chi.URLParam(r, "userId")
	ids, err := h.svc.ListIdentities(r.Context(), c.TenantID, c.Subject, userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, ids)
}

// DeleteIdentity handles DELETE /v1/users/{userId}/identities/{identityId}
func (h *Handler) DeleteIdentity(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	userID := chi.URLParam(r, "userId")
	identityID := chi.URLParam(r, "identityId")
	if err := h.svc.DeleteIdentity(r.Context(), c.TenantID, c.Subject, userID, identityID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}
