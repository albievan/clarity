package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/albievan/clarity/clarity-api/internal/apierr"
	"github.com/albievan/clarity/clarity-api/internal/config"
	"github.com/albievan/clarity/clarity-api/internal/domain/users"
	"github.com/albievan/clarity/clarity-api/internal/jwtutil"
	"github.com/albievan/clarity/clarity-api/internal/response"
)

// ── OAuth state store (in-memory; swap for Redis in production) ───────────────

type oauthState struct {
	TenantID  string
	Provider  string
	ExpiresAt time.Time
}

// oauthStateMap is a simple in-memory state store.
// For multi-instance deployments, replace with a Redis-backed store.
var oauthStateMap = make(map[string]oauthState)

func createOAuthState(tenantID, provider string) string {
	state := newID()
	oauthStateMap[state] = oauthState{
		TenantID:  tenantID,
		Provider:  provider,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	return state
}

func consumeOAuthState(state string) (*oauthState, bool) {
	st, ok := oauthStateMap[state]
	delete(oauthStateMap, state)
	if !ok || time.Now().After(st.ExpiresAt) {
		return nil, false
	}
	return &st, true
}

// ── OAuthHandler ──────────────────────────────────────────────────────────────

// OAuthHandler handles the Google and Apple OAuth2 flows.
type OAuthHandler struct {
	userSvc users.Service
	cfg     config.Config
}

// NewOAuthHandler constructs the OAuth handler.
func NewOAuthHandler(userSvc users.Service, cfg config.Config) *OAuthHandler {
	return &OAuthHandler{userSvc: userSvc, cfg: cfg}
}

// ── Google ────────────────────────────────────────────────────────────────────

// GoogleInit handles GET /v1/auth/oauth/google/init?tenant_id=xxx
func (h *OAuthHandler) GoogleInit(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		response.Error(w, apierr.BadRequest("tenant_id is required"))
		return
	}
	if h.cfg.OAuthGoogle.ClientID == "" {
		response.Error(w, apierr.BadRequest("Google OAuth is not configured on this server"))
		return
	}
	state := createOAuthState(tenantID, "google")
	params := url.Values{
		"client_id":     {h.cfg.OAuthGoogle.ClientID},
		"redirect_uri":  {h.cfg.OAuthGoogle.RedirectURL},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
		"access_type":   {"offline"},
		"prompt":        {"select_account"},
	}
	response.OK(w, map[string]string{
		"redirect_url": "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode(),
	})
}

// GoogleCallback handles GET /v1/auth/oauth/google/callback?code=xxx&state=xxx
func (h *OAuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errParam := q.Get("error"); errParam != "" {
		h.redirectError(w, r, errParam)
		return
	}
	st, ok := consumeOAuthState(q.Get("state"))
	if !ok {
		h.redirectError(w, r, "invalid_state")
		return
	}
	tokenResp, err := googleExchangeCode(q.Get("code"), h.cfg.OAuthGoogle)
	if err != nil {
		h.redirectError(w, r, "token_exchange_failed")
		return
	}
	info, err := googleGetUserInfo(tokenResp.AccessToken)
	if err != nil {
		h.redirectError(w, r, "userinfo_failed")
		return
	}
	user, _, err := h.userSvc.FindOrCreateOAuthUser(r.Context(), st.TenantID, users.OAuthIdentity{
		Provider:    "google",
		ProviderUID: info.Sub,
		Email:       info.Email,
		DisplayName: info.Name,
		AvatarURL:   info.Picture,
	})
	if err != nil {
		h.redirectError(w, r, "user_error")
		return
	}
	h.redirectWithTokens(w, r, user)
}

// ── Apple ─────────────────────────────────────────────────────────────────────

// AppleInit handles GET /v1/auth/oauth/apple/init?tenant_id=xxx
func (h *OAuthHandler) AppleInit(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		response.Error(w, apierr.BadRequest("tenant_id is required"))
		return
	}
	if h.cfg.OAuthApple.ClientID == "" {
		response.Error(w, apierr.BadRequest("Apple OAuth is not configured on this server"))
		return
	}
	state := createOAuthState(tenantID, "apple")
	params := url.Values{
		"client_id":     {h.cfg.OAuthApple.ClientID},
		"redirect_uri":  {h.cfg.OAuthApple.RedirectURL},
		"response_type": {"code"},
		"scope":         {"name email"},
		"state":         {state},
		"response_mode": {"form_post"}, // Apple uses POST
	}
	response.OK(w, map[string]string{
		"redirect_url": "https://appleid.apple.com/auth/authorize?" + params.Encode(),
	})
}

// AppleCallback handles POST /v1/auth/oauth/apple/callback (Apple sends a form POST)
func (h *OAuthHandler) AppleCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.redirectError(w, r, "parse_failed")
		return
	}
	if errParam := r.FormValue("error"); errParam != "" {
		h.redirectError(w, r, errParam)
		return
	}
	st, ok := consumeOAuthState(r.FormValue("state"))
	if !ok {
		h.redirectError(w, r, "invalid_state")
		return
	}
	idToken := r.FormValue("id_token")
	code := r.FormValue("code")

	appleInfo, err := parseAppleIDToken(idToken)
	if err != nil {
		// Fall back to code exchange if id_token is missing/invalid
		appleInfo, err = appleExchangeCode(code, h.cfg.OAuthApple)
		if err != nil {
			h.redirectError(w, r, "token_exchange_failed")
			return
		}
	}

	// Apple sends the user's name only on the very first authorisation
	displayName := appleInfo.Email
	if userJSON := r.FormValue("user"); userJSON != "" {
		var au struct {
			Name struct {
				FirstName string `json:"firstName"`
				LastName  string `json:"lastName"`
			} `json:"name"`
		}
		if json.Unmarshal([]byte(userJSON), &au) == nil && au.Name.FirstName != "" {
			displayName = strings.TrimSpace(au.Name.FirstName + " " + au.Name.LastName)
		}
	}

	user, _, err := h.userSvc.FindOrCreateOAuthUser(r.Context(), st.TenantID, users.OAuthIdentity{
		Provider:    "apple",
		ProviderUID: appleInfo.Sub,
		Email:       appleInfo.Email,
		DisplayName: displayName,
	})
	if err != nil {
		h.redirectError(w, r, "user_error")
		return
	}
	h.redirectWithTokens(w, r, user)
}

// ── Shared helpers ────────────────────────────────────────────────────────────

func (h *OAuthHandler) redirectWithTokens(w http.ResponseWriter, r *http.Request, user *users.User) {
	sessionID := newID()
	accessClaims := jwtutil.NewAccessClaims(
		user.ID, user.TenantID, sessionID, user.Roles, h.cfg.JWT.AccessTTL,
	)
	accessToken, err := jwtutil.Sign(h.cfg.JWT.Secret, accessClaims)
	if err != nil {
		h.redirectError(w, r, "sign_failed")
		return
	}
	refreshToken, err := jwtutil.Sign(h.cfg.JWT.Secret,
		jwtutil.NewRefreshClaims(user.ID, user.TenantID, sessionID, h.cfg.JWT.RefreshTTL),
	)
	if err != nil {
		h.redirectError(w, r, "sign_failed")
		return
	}
	dest := fmt.Sprintf("%s/auth/callback#access_token=%s&refresh_token=%s",
		h.cfg.FrontendURL, url.QueryEscape(accessToken), url.QueryEscape(refreshToken))
	http.Redirect(w, r, dest, http.StatusFound)
}

func (h *OAuthHandler) redirectError(w http.ResponseWriter, r *http.Request, errCode string) {
	http.Redirect(w, r, h.cfg.FrontendURL+"/auth/callback?error="+url.QueryEscape(errCode), http.StatusFound)
}

// ── Google API ────────────────────────────────────────────────────────────────

type googleTokenResponse struct{ AccessToken string `json:"access_token"` }
type googleUserInfo struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func googleExchangeCode(code string, cfg config.OAuthConfig) (*googleTokenResponse, error) {
	resp, err := http.PostForm("https://oauth2.googleapis.com/token", url.Values{
		"code": {code}, "client_id": {cfg.ClientID},
		"client_secret": {cfg.ClientSecret}, "redirect_uri": {cfg.RedirectURL},
		"grant_type": {"authorization_code"},
	})
	if err != nil {
		return nil, fmt.Errorf("google exchange: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google exchange: %d: %s", resp.StatusCode, data)
	}
	var tr googleTokenResponse
	return &tr, json.Unmarshal(data, &tr)
}

func googleGetUserInfo(accessToken string) (*googleUserInfo, error) {
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"https://www.googleapis.com/oauth2/v3/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google userinfo: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo: %d", resp.StatusCode)
	}
	var info googleUserInfo
	return &info, json.Unmarshal(data, &info)
}

// ── Apple JWT ─────────────────────────────────────────────────────────────────

type appleIDClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
}

// parseAppleIDToken decodes the Apple id_token payload WITHOUT signature verification.
// For production, verify against Apple's public keys at https://appleid.apple.com/auth/keys
func parseAppleIDToken(idToken string) (*appleIDClaims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid id_token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	var c appleIDClaims
	return &c, json.Unmarshal(payload, &c)
}

func appleExchangeCode(code string, cfg config.OAuthConfig) (*appleIDClaims, error) {
	// cfg.ClientSecret must be a pre-generated ES256-signed JWT
	// See: https://developer.apple.com/documentation/sign_in_with_apple/generate_and_validate_tokens
	resp, err := http.PostForm("https://appleid.apple.com/auth/token", url.Values{
		"code": {code}, "client_id": {cfg.ClientID},
		"client_secret": {cfg.ClientSecret}, "redirect_uri": {cfg.RedirectURL},
		"grant_type": {"authorization_code"},
	})
	if err != nil {
		return nil, fmt.Errorf("apple exchange: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apple exchange: %d: %s", resp.StatusCode, data)
	}
	var tr struct{ IDToken string `json:"id_token"` }
	if err := json.Unmarshal(data, &tr); err != nil {
		return nil, err
	}
	return parseAppleIDToken(tr.IDToken)
}
