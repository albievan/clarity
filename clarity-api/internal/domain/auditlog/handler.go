package auditlog

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/claims"
	"github.com/albievan/clarity/clarity-api/internal/pagination"
	"github.com/albievan/clarity/clarity-api/internal/response"
)

// Handler holds the HTTP handler functions for the auditlog domain.
type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// List handles GET /v1/audit-log
//
// Supported query parameters:
//
//	?entity_type=budgets      filter by table name
//	?entity_id=<uuid>         filter by a specific record's ID
//	?actor_user_id=<uuid>     filter by who performed the action
//	?action=APPROVE           filter by action type
//	?from=2024-01-01T00:00:00Z  entries on or after this time (RFC3339)
//	?to=2024-12-31T23:59:59Z    entries on or before this time (RFC3339)
//	?page=1&per_page=25
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}

	q := r.URL.Query()
	f := Filter{
		EntityType:  q.Get("entity_type"),
		EntityID:    q.Get("entity_id"),
		ActorUserID: q.Get("actor_user_id"),
		Action:      q.Get("action"),
		From:        parseTime(q.Get("from")),
		To:          parseTime(q.Get("to")),
	}

	p := pagination.Parse(r)
	items, total, err := h.svc.List(r.Context(), c.TenantID, c.Subject, f, p.Page, p.PerPage)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.PageOf(w, items, p.Page, p.PerPage, total)
}

// Get handles GET /v1/audit-log/{entryId}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	id := chi.URLParam(r, "entryId")
	entry, err := h.svc.Get(r.Context(), c.TenantID, c.Subject, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, entry)
}
