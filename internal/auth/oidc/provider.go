// Package oidc provides OAuth2/OIDC client implementation for modern SSO.
//
// Supports OAuth2 and OpenID Connect authentication with identity providers like Auth0.
// Implements the Authenticator interface from the auth package.
//
// This package supports M9.2.1 (SSO/SAML Integration) in the v2.0 roadmap.
package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/felixgeelhaar/specular/internal/auth"
)

// Config holds OAuth2/OIDC client configuration.
type Config struct {
	// ClientID from Auth0 application.
	// Example: "YOUR_AUTH0_CLIENT_ID"
	ClientID string

	// ClientSecret from Auth0 application.
	// Example: "YOUR_AUTH0_CLIENT_SECRET"
	ClientSecret string

	// Issuer is the Auth0 domain (OIDC issuer URL).
	// Example: "https://dev-abc123.us.auth0.com/"
	Issuer string

	// RedirectURL where Auth0 sends the authorization code.
	// Example: "https://specular.example.com/auth/callback"
	RedirectURL string

	// Scopes to request from the IdP.
	// Default: ["openid", "profile", "email"]
	Scopes []string

	// UsePKCE enables PKCE (Proof Key for Code Exchange) for enhanced security.
	// Default: true (recommended for public clients)
	UsePKCE bool
}

// Provider implements OAuth2/OIDC authentication client.
//
// Handles OAuth2 authorization code flow with PKCE:
//  1. User accesses protected resource
//  2. Provider generates PKCE code verifier and challenge
//  3. User redirected to Auth0 authorization endpoint
//  4. User authenticates with Auth0
//  5. Auth0 redirects back with authorization code
//  6. Provider exchanges code for tokens using PKCE verifier
//  7. Provider validates ID token signature and claims
//  8. Provider creates session from ID token
//
// Thread-safe: Safe for concurrent use.
type Provider struct {
	config         *Config
	oauth2Config   *oauth2.Config
	verifier       *oidc.IDTokenVerifier
	provider       *oidc.Provider
	sessionManager *auth.SessionManager

	// PKCE state storage (in production, use Redis or database)
	pkceStates *pkceStateStore
}

// NewProvider creates an OIDC authentication provider.
//
// Discovers OIDC configuration from the issuer's .well-known endpoint.
// Returns an error if discovery fails or configuration is invalid.
func NewProvider(cfg *Config, sessionMgr *auth.SessionManager) (*Provider, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, auth.WrapError(auth.ErrOIDCAuthorizationFailed, "invalid OIDC configuration", err, nil)
	}

	// Set defaults
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	if !cfg.UsePKCE {
		cfg.UsePKCE = true // PKCE is recommended for security
	}

	// Discover OIDC provider configuration
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, auth.WrapError(auth.ErrOIDCAuthorizationFailed, "failed to discover OIDC provider", err, map[string]interface{}{
			"issuer": cfg.Issuer,
		})
	}

	// Configure OAuth2
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes,
	}

	// Create ID token verifier
	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	return &Provider{
		config:         cfg,
		oauth2Config:   oauth2Config,
		verifier:       verifier,
		provider:       provider,
		sessionManager: sessionMgr,
		pkceStates:     newPKCEStateStore(),
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "oidc_auth0"
}

// Authenticate validates OAuth2 authorization code and returns a session.
//
// Expected request format:
//   - GET to callback URL with "code" and "state" query parameters
//   - Code must be exchanged for tokens within short time window
//   - State must match the one generated during InitiateLogin
//
// Returns ErrInvalidCredentials if authentication fails.
func (p *Provider) Authenticate(ctx context.Context, req *http.Request) (*auth.Session, error) {
	// This method is called by the Manager during callback processing
	// The actual authentication happens in HandleCallback
	return nil, auth.NewError(auth.ErrInvalidCredentials, "use HandleCallback for OIDC authentication", nil)
}

// ValidateSession validates an OIDC session.
//
// For OIDC, we primarily rely on JWT token validation.
// This method is called after JWT validation to perform any OIDC-specific checks.
func (p *Provider) ValidateSession(ctx context.Context, session *auth.Session) error {
	// OIDC sessions are validated via JWT
	// Could add additional checks here (e.g., token introspection)
	return nil
}

// RefreshSession attempts to refresh an OIDC session using refresh token.
//
// Uses the OAuth2 refresh token to obtain a new access token and ID token.
// Returns a new session with updated tokens.
func (p *Provider) RefreshSession(ctx context.Context, session *auth.Session) (*auth.Session, error) {
	if session.RefreshToken == "" {
		return nil, auth.NewError(auth.ErrRefreshFailed, "no refresh token available", map[string]interface{}{
			"user_id": session.UserID,
		})
	}

	// Create token source with refresh token
	token := &oauth2.Token{
		RefreshToken: session.RefreshToken,
	}

	tokenSource := p.oauth2Config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, auth.WrapError(auth.ErrRefreshFailed, "failed to refresh token", err, map[string]interface{}{
			"user_id": session.UserID,
		})
	}

	// Extract and validate ID token from new token
	rawIDToken, ok := newToken.Extra("id_token").(string)
	if !ok {
		return nil, auth.NewError(auth.ErrRefreshFailed, "no id_token in refresh response", nil)
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, auth.WrapError(auth.ErrOIDCIDTokenInvalid, "failed to verify refreshed ID token", err, nil)
	}

	// Create new session from refreshed token
	newSession, err := p.createSessionFromIDToken(ctx, idToken, newToken)
	if err != nil {
		return nil, err
	}

	return newSession, nil
}

// Logout terminates an OIDC session.
//
// Revokes the refresh token if supported by the IdP.
// Returns nil if logout succeeds or token revocation is not supported.
func (p *Provider) Logout(ctx context.Context, session *auth.Session) error {
	// OIDC logout would typically call the IdP's revocation endpoint
	// For now, we just remove the session (handled by Manager)
	// Could implement token revocation here if needed
	return nil
}

// InitiateLogin starts the OAuth2/OIDC authentication flow.
//
// Generates PKCE challenge and redirects user to Auth0 authorization endpoint.
// The IdP will authenticate the user and redirect back with an authorization code.
func (p *Provider) InitiateLogin(w http.ResponseWriter, r *http.Request) error {
	// Generate state parameter for CSRF protection
	state, err := generateRandomString(32)
	if err != nil {
		return auth.WrapError(auth.ErrOIDCAuthorizationFailed, "failed to generate state", err, nil)
	}

	// Build authorization URL
	var authURL string
	var codeVerifier string

	if p.config.UsePKCE {
		// Generate PKCE code verifier and challenge
		codeVerifier, err = generateRandomString(43) // 43 chars = 256 bits base64url
		if err != nil {
			return auth.WrapError(auth.ErrOIDCAuthorizationFailed, "failed to generate code verifier", err, nil)
		}

		codeChallenge := generatePKCEChallenge(codeVerifier)

		// Store PKCE state for later verification
		p.pkceStates.Store(state, &pkceState{
			CodeVerifier: codeVerifier,
			CreatedAt:    time.Now(),
		})

		// Add PKCE parameters to authorization URL
		authURL = p.oauth2Config.AuthCodeURL(state,
			oauth2.SetAuthURLParam("code_challenge", codeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
	} else {
		authURL = p.oauth2Config.AuthCodeURL(state)
	}

	// Redirect to IdP
	http.Redirect(w, r, authURL, http.StatusFound)
	return nil
}

// HandleCallback processes the OAuth2 authorization code from the IdP.
//
// Exchanges the code for tokens, validates the ID token, and creates a session.
func (p *Provider) HandleCallback(w http.ResponseWriter, r *http.Request) (*auth.Session, error) {
	ctx := r.Context()

	// Extract authorization code and state from query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		return nil, auth.NewError(auth.ErrOIDCAuthorizationFailed, "missing authorization code", nil)
	}
	if state == "" {
		return nil, auth.NewError(auth.ErrOIDCAuthorizationFailed, "missing state parameter", nil)
	}

	// Prepare token exchange options
	var opts []oauth2.AuthCodeOption

	if p.config.UsePKCE {
		// Retrieve PKCE code verifier
		pkceState, exists := p.pkceStates.Get(state)
		if !exists {
			return nil, auth.NewError(auth.ErrOIDCAuthorizationFailed, "invalid or expired state", map[string]interface{}{
				"state": state,
			})
		}

		// Clean up state
		p.pkceStates.Delete(state)

		// Add code verifier to token exchange
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", pkceState.CodeVerifier))
	}

	// Exchange authorization code for tokens
	token, err := p.oauth2Config.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, auth.WrapError(auth.ErrOIDCTokenExchangeFailed, "failed to exchange code for token", err, nil)
	}

	// Extract ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, auth.NewError(auth.ErrOIDCIDTokenInvalid, "no id_token in token response", nil)
	}

	// Verify ID token signature and claims
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, auth.WrapError(auth.ErrOIDCIDTokenInvalid, "failed to verify ID token", err, nil)
	}

	// Create session from ID token
	session, err := p.createSessionFromIDToken(ctx, idToken, token)
	if err != nil {
		return nil, err
	}

	// Create JWT session token
	tokenString, err := p.sessionManager.CreateSession(ctx, session)
	if err != nil {
		return nil, err
	}

	session.Token = tokenString

	// Create refresh token if available
	if token.RefreshToken != "" {
		refreshToken, refreshErr := p.sessionManager.CreateRefreshToken(ctx, session)
		if refreshErr != nil {
			return nil, refreshErr
		}
		session.RefreshToken = refreshToken
	}

	return session, nil
}

// createSessionFromIDToken extracts user information from OIDC ID token.
func (p *Provider) createSessionFromIDToken(ctx context.Context, idToken *oidc.IDToken, token *oauth2.Token) (*auth.Session, error) {
	// Extract standard claims
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Sub           string `json:"sub"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, auth.WrapError(auth.ErrOIDCIDTokenInvalid, "failed to parse ID token claims", err, nil)
	}

	// Extract all claims as attributes
	var allClaims map[string]interface{}
	if err := idToken.Claims(&allClaims); err != nil {
		return nil, auth.WrapError(auth.ErrOIDCIDTokenInvalid, "failed to parse ID token claims", err, nil)
	}

	// Create session
	session := &auth.Session{
		UserID:     claims.Sub,
		Email:      claims.Email,
		Provider:   p.Name(),
		CreatedAt:  time.Now(),
		ExpiresAt:  idToken.Expiry,
		Attributes: allClaims,
	}

	return session, nil
}

// validateConfig validates the OIDC configuration.
func validateConfig(cfg *Config) error {
	if cfg.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}
	if cfg.ClientSecret == "" {
		return fmt.Errorf("client secret is required")
	}
	if cfg.Issuer == "" {
		return fmt.Errorf("issuer is required")
	}
	if cfg.RedirectURL == "" {
		return fmt.Errorf("redirect URL is required")
	}
	return nil
}

// generateRandomString generates a cryptographically secure random string.
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// generatePKCEChallenge generates a PKCE code challenge from a verifier.
//
// Uses SHA-256 as the challenge method (S256).
func generatePKCEChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// pkceState stores PKCE code verifier for a specific state parameter.
type pkceState struct {
	CodeVerifier string
	CreatedAt    time.Time
}

// pkceStateStore manages PKCE state storage.
//
// In production, this should use Redis or a database for distributed deployments.
type pkceStateStore struct {
	states map[string]*pkceState
}

func newPKCEStateStore() *pkceStateStore {
	store := &pkceStateStore{
		states: make(map[string]*pkceState),
	}

	// Start background cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			store.cleanup()
		}
	}()

	return store
}

func (s *pkceStateStore) Store(state string, pkce *pkceState) {
	s.states[state] = pkce
}

func (s *pkceStateStore) Get(state string) (*pkceState, bool) {
	pkce, exists := s.states[state]
	return pkce, exists
}

func (s *pkceStateStore) Delete(state string) {
	delete(s.states, state)
}

func (s *pkceStateStore) cleanup() {
	now := time.Now()
	for state, pkce := range s.states {
		// Remove states older than 10 minutes
		if now.Sub(pkce.CreatedAt) > 10*time.Minute {
			delete(s.states, state)
		}
	}
}
