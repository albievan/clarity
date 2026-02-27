package purchaseorders

import (
	"encoding/json"
	"net/http"

	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/claims"
	"github.com/albievan/clarity/clarity-api/internal/pagination"
	"github.com/albievan/clarity/clarity-api/internal/response"
	"github.com/go-chi/chi/v5"
)

// Handler holds the HTTP handler functions for the purchaseorders domain.
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

// Get handles GET /purchaseorders/{id}.
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

// Create handles POST /purchaseorders.
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

// Update handles PUT /purchaseorders/{id}.
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

// Delete handles DELETE /purchaseorders/{id}.
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

// Close handles the Close action.
func (h *Handler) Close(w http.ResponseWriter, r *http.Request) {
	// TODO: implement Close
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "Close not yet implemented"})
}

// ListLines handles the ListLines action.
func (h *Handler) ListLines(w http.ResponseWriter, r *http.Request) {
	// TODO: implement ListLines
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "ListLines not yet implemented"})
}

// AddLine handles the AddLine action.
func (h *Handler) AddLine(w http.ResponseWriter, r *http.Request) {
	// TODO: implement AddLine
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "AddLine not yet implemented"})
}

// UpdateLine handles the UpdateLine action.
func (h *Handler) UpdateLine(w http.ResponseWriter, r *http.Request) {
	// TODO: implement UpdateLine
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "UpdateLine not yet implemented"})
}

// DeleteLine handles the DeleteLine action.
func (h *Handler) DeleteLine(w http.ResponseWriter, r *http.Request) {
	// TODO: implement DeleteLine
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "DeleteLine not yet implemented"})
}

// ListReceipts handles the ListReceipts action.
func (h *Handler) ListReceipts(w http.ResponseWriter, r *http.Request) {
	// TODO: implement ListReceipts
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "ListReceipts not yet implemented"})
}

// RecordReceipt handles the RecordReceipt action.
func (h *Handler) RecordReceipt(w http.ResponseWriter, r *http.Request) {
	// TODO: implement RecordReceipt
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "RecordReceipt not yet implemented"})
}

// ListDisputes handles the ListDisputes action.
func (h *Handler) ListDisputes(w http.ResponseWriter, r *http.Request) {
	// TODO: implement ListDisputes
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "ListDisputes not yet implemented"})
}

// RaiseDispute handles the RaiseDispute action.
func (h *Handler) RaiseDispute(w http.ResponseWriter, r *http.Request) {
	// TODO: implement RaiseDispute
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "RaiseDispute not yet implemented"})
}

// ResolveDispute handles the ResolveDispute action.
func (h *Handler) ResolveDispute(w http.ResponseWriter, r *http.Request) {
	// TODO: implement ResolveDispute
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "ResolveDispute not yet implemented"})
}
