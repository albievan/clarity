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

// Handler holds the HTTP handler functions for the users domain.
type Handler struct {
	svc Service
}

// NewHandler constructs a Handler.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET list endpoint.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	p := pagination.Parse(r)
	items, total, err := h.svc.List(r.Context(), c.TenantID, c.Subject, p.Page, p.PerPage)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.PageOf(w, items, p.Page, p.PerPage, total)
}

// Get handles GET /users/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	id := chi.URLParam(r, "id")
	item, err := h.svc.Get(r.Context(), c.TenantID, c.Subject, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, item)
}

// Create handles POST /users.
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
	item, err := h.svc.Create(r.Context(), c.TenantID, c.Subject, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, item)
}

// Update handles PUT /users/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	id := chi.URLParam(r, "id")
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	item, err := h.svc.Update(r.Context(), c.TenantID, c.Subject, id, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, item)
}

// Delete handles DELETE /users/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), c.TenantID, c.Subject, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ListRoles handles the ListRoles action.
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	// TODO: implement ListRoles
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "ListRoles not yet implemented"})
}

// AssignRole handles the AssignRole action.
func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	// TODO: implement AssignRole
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "AssignRole not yet implemented"})
}

// RevokeRole handles the RevokeRole action.
func (h *Handler) RevokeRole(w http.ResponseWriter, r *http.Request) {
	// TODO: implement RevokeRole
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "RevokeRole not yet implemented"})
}

// Lock handles the Lock action.
func (h *Handler) Lock(w http.ResponseWriter, r *http.Request) {
	// TODO: implement Lock
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "Lock not yet implemented"})
}

// Unlock handles the Unlock action.
func (h *Handler) Unlock(w http.ResponseWriter, r *http.Request) {
	// TODO: implement Unlock
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "Unlock not yet implemented"})
}

// Deprovision handles the Deprovision action.
func (h *Handler) Deprovision(w http.ResponseWriter, r *http.Request) {
	// TODO: implement Deprovision
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "Deprovision not yet implemented"})
}
