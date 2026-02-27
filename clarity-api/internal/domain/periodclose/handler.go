package periodclose

import (
	"encoding/json"
	"net/http"

	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/claims"
	"github.com/albievan/clarity/clarity-api/internal/pagination"
	"github.com/albievan/clarity/clarity-api/internal/response"
	"github.com/go-chi/chi/v5"
)

// Handler holds the HTTP handler functions for the periodclose domain.
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

// Get handles GET /periodclose/{id}.
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

// Create handles POST /periodclose.
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

// Update handles PUT /periodclose/{id}.
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

// Delete handles DELETE /periodclose/{id}.
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

// StartSession handles the StartSession action.
func (h *Handler) StartSession(w http.ResponseWriter, r *http.Request) {
	// TODO: implement StartSession
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "StartSession not yet implemented"})
}

// GetSession handles the GetSession action.
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	// TODO: implement GetSession
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "GetSession not yet implemented"})
}

// CompleteStep handles the CompleteStep action.
func (h *Handler) CompleteStep(w http.ResponseWriter, r *http.Request) {
	// TODO: implement CompleteStep
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "CompleteStep not yet implemented"})
}

// ListCarryOverRules handles the ListCarryOverRules action.
func (h *Handler) ListCarryOverRules(w http.ResponseWriter, r *http.Request) {
	// TODO: implement ListCarryOverRules
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "ListCarryOverRules not yet implemented"})
}

// CreateCarryOverRule handles the CreateCarryOverRule action.
func (h *Handler) CreateCarryOverRule(w http.ResponseWriter, r *http.Request) {
	// TODO: implement CreateCarryOverRule
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "CreateCarryOverRule not yet implemented"})
}
