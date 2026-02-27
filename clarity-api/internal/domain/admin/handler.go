package admin

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/claims"
	"github.com/albievan/clarity/clarity-api/internal/pagination"
	"github.com/albievan/clarity/clarity-api/internal/response"
)

// Handler holds the HTTP handler functions for the admin domain.
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

// Get handles GET /admin/{id}.
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

// Create handles POST /admin.
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

// Update handles PUT /admin/{id}.
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

// Delete handles DELETE /admin/{id}.
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

// GetTenant handles the GetTenant action.
func (h *Handler) GetTenant(w http.ResponseWriter, r *http.Request) {
	// TODO: implement GetTenant
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "GetTenant not yet implemented"})
}

// UpdateTenant handles the UpdateTenant action.
func (h *Handler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	// TODO: implement UpdateTenant
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "UpdateTenant not yet implemented"})
}

// GetSecurityPolicy handles the GetSecurityPolicy action.
func (h *Handler) GetSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	// TODO: implement GetSecurityPolicy
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "GetSecurityPolicy not yet implemented"})
}

// UpdateSecurityPolicy handles the UpdateSecurityPolicy action.
func (h *Handler) UpdateSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	// TODO: implement UpdateSecurityPolicy
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "UpdateSecurityPolicy not yet implemented"})
}

// ListFeatureFlags handles the ListFeatureFlags action.
func (h *Handler) ListFeatureFlags(w http.ResponseWriter, r *http.Request) {
	// TODO: implement ListFeatureFlags
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "ListFeatureFlags not yet implemented"})
}

// UpdateFeatureFlag handles the UpdateFeatureFlag action.
func (h *Handler) UpdateFeatureFlag(w http.ResponseWriter, r *http.Request) {
	// TODO: implement UpdateFeatureFlag
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "UpdateFeatureFlag not yet implemented"})
}
