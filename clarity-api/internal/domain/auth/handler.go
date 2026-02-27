package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/claims"
	"github.com/albievan/clarity/clarity-api/internal/config"
	"github.com/albievan/clarity/clarity-api/internal/response"
)

// Handler holds auth HTTP handlers.
type Handler struct {
	svc Service
	cfg config.JWTConfig
}

// NewHandler constructs an auth Handler.
func NewHandler(svc Service, cfg config.JWTConfig) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

// ── Request / Response types ──────────────────────────────────────

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TenantID string `json:"tenant_id"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	MFARequired  bool   `json:"mfa_required,omitempty"`
	MFAToken     string `json:"mfa_token,omitempty"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type MFAVerifyRequest struct {
	MFAToken string `json:"mfa_token"`
	Code     string `json:"code"`
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type PasswordResetRequest struct {
	Email string `json:"email"`
}

type PasswordResetCompleteRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ── Handlers ──────────────────────────────────────────────────────

// Login handles POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	// TODO: call h.svc.Login(r.Context(), req) → (LoginResponse, error)
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "Login not yet implemented"})
}

// Logout handles POST /auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	// TODO: call h.svc.Logout(r.Context(), c.TenantID, c.SessionID)
	_ = c
	response.NoContent(w)
}

// Refresh handles POST /auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	// TODO: call h.svc.Refresh(r.Context(), req.RefreshToken)
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "Refresh not yet implemented"})
}

// MFASetup handles POST /auth/mfa/setup
func (h *Handler) MFASetup(w http.ResponseWriter, r *http.Request) {
	c, _ := claims.FromCtx(r.Context())
	// TODO: call h.svc.MFASetup(r.Context(), c.TenantID, c.Subject)
	_ = c
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "MFASetup not yet implemented"})
}

// MFAConfirm handles POST /auth/mfa/confirm
func (h *Handler) MFAConfirm(w http.ResponseWriter, r *http.Request) {
	c, _ := claims.FromCtx(r.Context())
	_ = c
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "MFAConfirm not yet implemented"})
}

// MFAVerify handles POST /auth/mfa/verify
func (h *Handler) MFAVerify(w http.ResponseWriter, r *http.Request) {
	var req MFAVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	// TODO: call h.svc.MFAVerify(r.Context(), req.MFAToken, req.Code)
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "MFAVerify not yet implemented"})
}

// MFADisable handles POST /auth/mfa/disable
func (h *Handler) MFADisable(w http.ResponseWriter, r *http.Request) {
	c, _ := claims.FromCtx(r.Context())
	_ = c
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "MFADisable not yet implemented"})
}

// PasswordChange handles POST /auth/password/change
func (h *Handler) PasswordChange(w http.ResponseWriter, r *http.Request) {
	var req PasswordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	c, _ := claims.FromCtx(r.Context())
	_ = c
	// TODO: call h.svc.PasswordChange(r.Context(), c.TenantID, c.Subject, req)
	response.NoContent(w)
}

// PasswordResetRequest handles POST /auth/password/reset-request
func (h *Handler) PasswordResetRequest(w http.ResponseWriter, r *http.Request) {
	// Always return 204 regardless of email existence (prevents enumeration).
	response.NoContent(w)
}

// PasswordReset handles POST /auth/password/reset
func (h *Handler) PasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	// TODO: call h.svc.PasswordReset(r.Context(), req.Token, req.NewPassword)
	response.NoContent(w)
}

// ListSessions handles GET /auth/sessions
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	c, _ := claims.FromCtx(r.Context())
	_ = c
	// TODO: call h.svc.ListSessions(r.Context(), c.TenantID, c.Subject)
	response.JSON(w, http.StatusNotImplemented, map[string]string{"message": "ListSessions not yet implemented"})
}

// RevokeSession handles DELETE /auth/sessions/{sessionId}
func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	c, _ := claims.FromCtx(r.Context())
	if sessionID == c.SessionID {
		response.Error(w, apierr.BadRequest("use /auth/logout to revoke the current session"))
		return
	}
	// TODO: call h.svc.RevokeSession(r.Context(), c.TenantID, c.Subject, sessionID)
	response.NoContent(w)
}
