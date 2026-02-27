package intakerequests

import (
	"encoding/json"
	"net/http"

	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/claims"
	"github.com/albievan/clarity/clarity-api/internal/pagination"
	"github.com/albievan/clarity/clarity-api/internal/response"
	"github.com/go-chi/chi/v5"
)

// Handler holds the HTTP handler functions for the intakerequests domain.
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

// Get handles GET /intakerequests/{id}.
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

// Create handles POST /intakerequests.
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

// Update handles PUT /intakerequests/{id}.
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

// Delete handles DELETE /intakerequests/{id}.
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

// Submit handles the Submit action.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	// TODO: implement Submit
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "Submit not yet implemented"})
}

// Approve handles the Approve action.
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	// TODO: implement Approve
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "Approve not yet implemented"})
}

// Reject handles the Reject action.
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	// TODO: implement Reject
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "Reject not yet implemented"})
}

// Convert handles the Convert action.
func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	// TODO: implement Convert
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "Convert not yet implemented"})
}
