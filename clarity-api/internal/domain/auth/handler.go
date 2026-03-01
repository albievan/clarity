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

// ── Public endpoints (no JWT required) ───────────────────────────────────────

// Login handles POST /v1/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	resp, err := h.svc.Login(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, resp)
}

// Refresh handles POST /v1/auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	resp, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, resp)
}

// MFAVerify handles POST /v1/auth/mfa/verify
// Called after Login returns {mfa_required: true, mfa_token: "..."}.
func (h *Handler) MFAVerify(w http.ResponseWriter, r *http.Request) {
	var req MFAVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	resp, err := h.svc.MFAVerify(r.Context(), req.MFAToken, req.Code)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, resp)
}

// PasswordResetRequest handles POST /v1/auth/password/reset-request
// Always returns 204 regardless of whether the email exists (prevents enumeration).
func (h *Handler) PasswordResetRequest(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetRequestBody
	_ = json.NewDecoder(r.Body).Decode(&req) // silently ignore decode errors
	_ = h.svc.PasswordResetRequest(r.Context(), req.TenantID, req.Email)
	response.NoContent(w)
}

// PasswordReset handles POST /v1/auth/password/reset
func (h *Handler) PasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	if err := h.svc.PasswordReset(r.Context(), req.Token, req.NewPassword); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ── Authenticated endpoints (JWT required) ────────────────────────────────────

// Logout handles POST /v1/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	if err := h.svc.Logout(r.Context(), c.TenantID, c.SessionID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// MFASetup handles POST /v1/auth/mfa/setup
// Returns the TOTP secret and an otpauth:// URI for QR code generation.
func (h *Handler) MFASetup(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	resp, err := h.svc.MFASetup(r.Context(), c.TenantID, c.Subject)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, resp)
}

// MFAConfirm handles POST /v1/auth/mfa/confirm
// Validates a TOTP code against the pending secret and activates MFA.
// Returns one-time backup codes — these are only shown once.
func (h *Handler) MFAConfirm(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	var req MFAConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	resp, err := h.svc.MFAConfirm(r.Context(), c.TenantID, c.Subject, req.Code)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, resp)
}

// MFADisable handles POST /v1/auth/mfa/disable
// Requires current TOTP confirmation. Blocked if tenant policy mandates MFA.
func (h *Handler) MFADisable(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	var req MFADisableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	if err := h.svc.MFADisable(r.Context(), c.TenantID, c.Subject, req.Code); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// PasswordChange handles POST /v1/auth/password/change
// Verifies the current password before updating. Revokes all other sessions.
func (h *Handler) PasswordChange(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	var req PasswordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierr.BadRequest("invalid request body"))
		return
	}
	if err := h.svc.PasswordChange(r.Context(), c.TenantID, c.Subject, req); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ListSessions handles GET /v1/auth/sessions
// Returns all active sessions for the calling user; marks the current one.
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	sessions, err := h.svc.ListSessions(r.Context(), c.TenantID, c.Subject, c.SessionID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.OK(w, sessions)
}

// RevokeSession handles DELETE /v1/auth/sessions/{sessionId}
// To revoke the current session use POST /auth/logout instead.
func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	c, err := claims.FromCtx(r.Context())
	if err != nil {
		response.Error(w, apierr.Unauthorized("missing claims"))
		return
	}
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == c.SessionID {
		response.Error(w, apierr.BadRequest("use POST /auth/logout to revoke the current session"))
		return
	}
	if err := h.svc.RevokeSession(r.Context(), c.TenantID, c.Subject, sessionID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}
