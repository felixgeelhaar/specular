// Package auth provides enterprise authentication and authorization for Specular.
//
// It implements a pluggable authentication architecture supporting:
//   - Email/password with JWT tokens (legacy)
//   - SAML 2.0 for enterprise SSO (Okta)
//   - OAuth2/OIDC for modern SSO (Auth0)
//
// This package supports M9.2.1 (SSO/SAML Integration) in the v2.0 roadmap.
package auth

import (
	"context"
	"net/http"
	"time"
)

// Authenticator defines the interface for all authentication providers.
//
// Implementations must be thread-safe and handle concurrent requests.
// Each provider is responsible for validating credentials and creating sessions.
type Authenticator interface {
	// Name returns the authentication provider name (e.g., "okta_saml", "auth0_oidc", "legacy_jwt").
	Name() string

	// Authenticate validates credentials from the HTTP request and returns a session.
	// Returns ErrInvalidCredentials if authentication fails.
	// Returns ErrProviderUnavailable if the provider cannot be reached.
	Authenticate(ctx context.Context, req *http.Request) (*Session, error)

	// ValidateSession checks if a session is still valid.
	// Returns ErrSessionExpired if the session has expired.
	// Returns ErrSessionInvalid if the session cannot be validated.
	ValidateSession(ctx context.Context, session *Session) error

	// RefreshSession attempts to refresh an expired session using a refresh token.
	// Returns a new session with updated tokens.
	// Returns ErrRefreshFailed if refresh is not possible.
	RefreshSession(ctx context.Context, session *Session) (*Session, error)

	// Logout terminates a session and invalidates tokens.
	// Implementations should call IdP logout endpoints if supported.
	Logout(ctx context.Context, session *Session) error
}

// Session represents an authenticated user session.
//
// Sessions are provider-agnostic and use JWT tokens internally.
// All authentication providers (legacy, SAML, OIDC) create Session objects.
type Session struct {
	// UserID is the unique identifier for the user.
	// This is the primary key for user identity.
	UserID string

	// Email is the user's email address.
	Email string

	// OrganizationID is the user's organization identifier (required for ABAC).
	OrganizationID string

	// OrganizationRole is the user's role within the organization (required for ABAC).
	OrganizationRole string

	// TeamID is the user's team identifier (optional, for team-level ABAC).
	TeamID *string

	// TeamRole is the user's role within their team (optional, for team-level ABAC).
	TeamRole *string

	// Provider is the authentication provider that created this session.
	// Examples: "okta_saml", "auth0_oidc", "legacy_jwt"
	Provider string

	// Token is the authentication token (JWT format).
	// This token is used for subsequent API requests.
	Token string

	// RefreshToken is used to obtain a new access token when it expires.
	// Not all providers support refresh tokens.
	RefreshToken string

	// ExpiresAt is when the session expires.
	// After this time, the session must be refreshed or the user must re-authenticate.
	ExpiresAt time.Time

	// CreatedAt is when the session was created.
	CreatedAt time.Time

	// Attributes contains provider-specific user attributes.
	// Examples: name, groups, roles, custom claims from SAML/OIDC
	Attributes map[string]interface{}
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Manager coordinates multiple authentication providers.
//
// It implements the Strategy pattern, delegating authentication to registered providers.
// Providers are tried in registration order until one succeeds.
type Manager struct {
	providers    map[string]Authenticator
	sessionStore SessionStore
}

// NewManager creates a new authentication manager.
//
// The session store is used to persist sessions across requests.
// Use NewMemoryStore() for single-instance deployments.
// Use NewRedisStore() for distributed deployments (future: M10.2).
func NewManager(store SessionStore) *Manager {
	return &Manager{
		providers:    make(map[string]Authenticator),
		sessionStore: store,
	}
}

// Register registers an authentication provider.
//
// Returns ErrDuplicateProvider if a provider with the same name is already registered.
// Providers are tried in registration order during authentication.
func (m *Manager) Register(auth Authenticator) error {
	name := auth.Name()
	if _, exists := m.providers[name]; exists {
		return NewError(ErrDuplicateProvider, "provider already registered", map[string]interface{}{
			"provider": name,
		})
	}

	m.providers[name] = auth
	return nil
}

// Authenticate attempts authentication with all registered providers.
//
// Providers are tried in registration order until one succeeds.
// Returns the first successful session or an aggregate error if all providers fail.
//
// The HTTP request should contain provider-specific credentials:
//   - Authorization header with Bearer token (legacy JWT)
//   - SAML assertion in POST body
//   - OAuth2 authorization code in query parameters
func (m *Manager) Authenticate(ctx context.Context, req *http.Request) (*Session, error) {
	if len(m.providers) == 0 {
		return nil, NewError(ErrNoProviders, "no authentication providers registered", nil)
	}

	var lastErr error
	for name, provider := range m.providers {
		session, err := provider.Authenticate(ctx, req)
		if err == nil {
			// Store session for later validation
			if storeErr := m.sessionStore.Store(ctx, session.UserID, session); storeErr != nil {
				return nil, NewError(ErrSessionStoreFailed, "failed to store session", map[string]interface{}{
					"provider": name,
					"user_id":  session.UserID,
					"error":    storeErr.Error(),
				})
			}
			return session, nil
		}
		lastErr = err
	}

	// All providers failed
	return nil, lastErr
}

// ValidateSession validates a session token.
//
// First checks if the session exists in the session store.
// Then delegates to the provider for additional validation.
// Returns the session if valid, or an error if invalid or expired.
func (m *Manager) ValidateSession(ctx context.Context, token string) (*Session, error) {
	// Parse token to extract user ID and provider
	claims, err := ParseSessionToken(token)
	if err != nil {
		return nil, NewError(ErrSessionInvalid, "invalid session token", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Retrieve session from store
	session, err := m.sessionStore.Get(ctx, claims.UserID)
	if err != nil {
		return nil, NewError(ErrSessionNotFound, "session not found", map[string]interface{}{
			"user_id": claims.UserID,
		})
	}

	// Check if session is expired
	if session.IsExpired() {
		return nil, NewError(ErrSessionExpired, "session has expired", map[string]interface{}{
			"user_id":    session.UserID,
			"expires_at": session.ExpiresAt,
		})
	}

	// Delegate to provider for additional validation
	provider, exists := m.providers[session.Provider]
	if !exists {
		return nil, NewError(ErrProviderNotFound, "provider not found", map[string]interface{}{
			"provider": session.Provider,
		})
	}

	if validateErr := provider.ValidateSession(ctx, session); validateErr != nil {
		return nil, validateErr
	}

	return session, nil
}

// RefreshSession attempts to refresh an expired session.
//
// Delegates to the provider for token refresh.
// Returns a new session with updated tokens and expiration.
func (m *Manager) RefreshSession(ctx context.Context, session *Session) (*Session, error) {
	provider, exists := m.providers[session.Provider]
	if !exists {
		return nil, NewError(ErrProviderNotFound, "provider not found", map[string]interface{}{
			"provider": session.Provider,
		})
	}

	// Delegate to provider for refresh
	newSession, err := provider.RefreshSession(ctx, session)
	if err != nil {
		return nil, err
	}

	// Update session store with new session
	if storeErr := m.sessionStore.Store(ctx, newSession.UserID, newSession); storeErr != nil {
		return nil, NewError(ErrSessionStoreFailed, "failed to store refreshed session", map[string]interface{}{
			"user_id": newSession.UserID,
			"error":   storeErr.Error(),
		})
	}

	return newSession, nil
}

// Logout terminates a session and removes it from the session store.
//
// Delegates to the provider for IdP logout if supported.
func (m *Manager) Logout(ctx context.Context, session *Session) error {
	provider, exists := m.providers[session.Provider]
	if !exists {
		return NewError(ErrProviderNotFound, "provider not found", map[string]interface{}{
			"provider": session.Provider,
		})
	}

	// Delegate to provider for logout
	if err := provider.Logout(ctx, session); err != nil {
		return err
	}

	// Remove from session store
	if err := m.sessionStore.Delete(ctx, session.UserID); err != nil {
		return NewError(ErrSessionStoreFailed, "failed to delete session", map[string]interface{}{
			"user_id": session.UserID,
			"error":   err.Error(),
		})
	}

	return nil
}

// GetProvider returns a registered provider by name.
//
// Returns ErrProviderNotFound if the provider doesn't exist.
func (m *Manager) GetProvider(name string) (Authenticator, error) {
	provider, exists := m.providers[name]
	if !exists {
		return nil, NewError(ErrProviderNotFound, "provider not found", map[string]interface{}{
			"provider": name,
		})
	}
	return provider, nil
}

// ListProviders returns the names of all registered providers.
func (m *Manager) ListProviders() []string {
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}
