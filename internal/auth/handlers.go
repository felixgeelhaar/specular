package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Handlers provides HTTP handlers for authentication endpoints.
type Handlers struct {
	manager        *Manager
	sessionManager *SessionManager
	sessionStore   SessionStore
	secureCookies  bool // Set to true in production (HTTPS)
}

// NewHandlers creates new authentication HTTP handlers.
func NewHandlers(manager *Manager, sessionManager *SessionManager, sessionStore SessionStore, secureCookies bool) *Handlers {
	return &Handlers{
		manager:        manager,
		sessionManager: sessionManager,
		sessionStore:   sessionStore,
		secureCookies:  secureCookies,
	}
}

// HandleLogin initiates the authentication flow for a specific provider.
//
// GET /auth/login?provider=saml_okta
// GET /auth/login?provider=oidc_auth0
//
// Redirects the user to the identity provider's login page.
func (h *Handlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get provider name from query parameter
	providerName := r.URL.Query().Get("provider")
	if providerName == "" {
		h.writeError(w, http.StatusBadRequest, "AUTH_PROVIDER_REQUIRED", "provider parameter is required", nil)
		return
	}

	// Get provider
	provider, err := h.manager.GetProvider(providerName)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "AUTH_PROVIDER_NOT_FOUND", fmt.Sprintf("provider %s not found", providerName), nil)
		return
	}

	// Check if provider supports login initiation
	type LoginInitiator interface {
		InitiateLogin(w http.ResponseWriter, r *http.Request) error
	}

	initiator, ok := provider.(LoginInitiator)
	if !ok {
		h.writeError(w, http.StatusBadRequest, "AUTH_PROVIDER_NO_LOGIN", fmt.Sprintf("provider %s does not support login initiation", providerName), nil)
		return
	}

	// Initiate login (redirects to IdP)
	if initiateErr := initiator.InitiateLogin(w, r); initiateErr != nil {
		h.writeAuthError(w, initiateErr)
		return
	}
}

// HandleCallback processes the authentication callback from the identity provider.
//
// POST /auth/callback/saml_okta (SAML assertion via POST)
// GET /auth/callback/oidc_auth0 (OAuth2 code via query param)
//
// Creates a session and sets session cookie.
func (h *Handlers) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Get provider name from URL path
	// Expected pattern: /auth/callback/{provider}
	providerName := r.URL.Query().Get("provider")
	if providerName == "" {
		h.writeError(w, http.StatusBadRequest, "AUTH_PROVIDER_REQUIRED", "provider parameter is required", nil)
		return
	}

	// Get provider
	provider, err := h.manager.GetProvider(providerName)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "AUTH_PROVIDER_NOT_FOUND", fmt.Sprintf("provider %s not found", providerName), nil)
		return
	}

	// Check if provider supports callback handling
	type CallbackHandler interface {
		HandleCallback(w http.ResponseWriter, r *http.Request) (*Session, error)
	}

	handler, ok := provider.(CallbackHandler)
	if !ok {
		h.writeError(w, http.StatusBadRequest, "AUTH_PROVIDER_NO_CALLBACK", fmt.Sprintf("provider %s does not support callback handling", providerName), nil)
		return
	}

	// Handle callback and create session
	session, err := handler.HandleCallback(w, r)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	// Store session
	if storeErr := h.sessionStore.Store(r.Context(), session.UserID, session); storeErr != nil {
		h.writeError(w, http.StatusInternalServerError, "AUTH_SESSION_STORE_FAILED", "failed to store session", map[string]interface{}{
			"error": storeErr.Error(),
		})
		return
	}

	// Set session cookie
	SetSessionCookie(w, session.Token, session.ExpiresAt.Unix(), h.secureCookies)

	// Return session info
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"user_id":    session.UserID,
		"email":      session.Email,
		"provider":   session.Provider,
		"expires_at": session.ExpiresAt.Unix(),
	})
}

// HandleLogout terminates the user's session.
//
// POST /auth/logout
//
// Removes the session and clears the session cookie.
func (h *Handlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session from context
	session := GetSession(r.Context())
	if session == nil {
		// Already logged out
		ClearSessionCookie(w)
		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "logged out",
		})
		return
	}

	// Remove session from store
	if err := h.sessionStore.Delete(r.Context(), session.UserID); err != nil {
		// Log error but continue with logout
		fmt.Printf("Failed to delete session from store: %v\n", err)
	}

	// Call provider logout (for IdP logout if supported)
	if err := h.manager.Logout(r.Context(), session); err != nil {
		// Log error but continue with logout
		fmt.Printf("Provider logout failed: %v\n", err)
	}

	// Clear session cookie
	ClearSessionCookie(w)

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "logged out",
	})
}

// HandleRefresh refreshes an expired session using a refresh token.
//
// POST /auth/refresh
// Body: {"refresh_token": "..."}
//
// Returns a new access token and optionally a new refresh token.
func (h *Handlers) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "AUTH_INVALID_REQUEST", "invalid request body", nil)
		return
	}

	if req.RefreshToken == "" {
		h.writeError(w, http.StatusBadRequest, "AUTH_REFRESH_TOKEN_REQUIRED", "refresh_token is required", nil)
		return
	}

	// Validate refresh token
	claims, err := h.sessionManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	// Get provider
	provider, err := h.manager.GetProvider(claims.Provider)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "AUTH_PROVIDER_NOT_FOUND", fmt.Sprintf("provider %s not found", claims.Provider), nil)
		return
	}

	// Create session from refresh token claims
	session := &Session{
		UserID:       claims.UserID,
		Email:        claims.Email,
		Provider:     claims.Provider,
		RefreshToken: req.RefreshToken,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	// Attempt to refresh session with provider
	newSession, err := provider.RefreshSession(r.Context(), session)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	// Create new access token
	tokenString, err := h.sessionManager.CreateSession(r.Context(), newSession)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "AUTH_TOKEN_CREATION_FAILED", "failed to create session token", nil)
		return
	}
	newSession.Token = tokenString

	// Create new refresh token if provider supports it
	var newRefreshToken string
	if newSession.RefreshToken != "" {
		newRefreshToken, err = h.sessionManager.CreateRefreshToken(r.Context(), newSession)
		if err != nil {
			// Log error but continue (old refresh token still works)
			fmt.Printf("Failed to create new refresh token: %v\n", err)
			newRefreshToken = req.RefreshToken
		}
	}

	// Update session in store
	if storeErr := h.sessionStore.Store(r.Context(), newSession.UserID, newSession); storeErr != nil {
		h.writeError(w, http.StatusInternalServerError, "AUTH_SESSION_STORE_FAILED", "failed to store session", nil)
		return
	}

	// Set session cookie
	SetSessionCookie(w, newSession.Token, newSession.ExpiresAt.Unix(), h.secureCookies)

	// Return new tokens
	response := map[string]interface{}{
		"success":      true,
		"access_token": newSession.Token,
		"expires_at":   newSession.ExpiresAt.Unix(),
	}
	if newRefreshToken != "" {
		response["refresh_token"] = newRefreshToken
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleMe returns information about the authenticated user.
//
// GET /auth/me
//
// Requires authentication (use with RequireAuth middleware).
func (h *Handlers) HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session from context (set by RequireAuth middleware)
	session := MustGetSession(r.Context())

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":    session.UserID,
		"email":      session.Email,
		"provider":   session.Provider,
		"created_at": session.CreatedAt.Unix(),
		"expires_at": session.ExpiresAt.Unix(),
		"attributes": session.Attributes,
	})
}

// writeAuthError writes an authentication error response.
func (h *Handlers) writeAuthError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	authErr, ok := err.(*AuthError)
	if !ok {
		// Generic error
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck // Error response, ignore encoding errors
			"error":   "authentication_failed",
			"message": err.Error(),
		})
		return
	}

	// Determine HTTP status code
	statusCode := http.StatusUnauthorized
	switch authErr.Code {
	case ErrSAMLAssertionInvalid, ErrOIDCIDTokenInvalid:
		statusCode = http.StatusBadRequest
	case ErrRefreshFailed:
		statusCode = http.StatusUnauthorized
	}

	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck // Error response, ignore encoding errors
		"error":   authErr.Code,
		"message": authErr.Message,
		"context": authErr.Context,
	})
}

// writeError writes a generic error response.
func (h *Handlers) writeError(w http.ResponseWriter, statusCode int, code, message string, context map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck // Error response, ignore encoding errors
		"error":   code,
		"message": message,
		"context": context,
	})
}

// writeJSON writes a JSON response.
func (h *Handlers) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("Failed to encode JSON response: %v\n", err)
	}
}
