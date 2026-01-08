package oidc

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/specular/internal/auth"
)

// TestValidateConfig tests configuration validation
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError string
	}{
		{
			name:      "empty_client_id",
			config:    &Config{},
			wantError: "client ID is required",
		},
		{
			name: "empty_client_secret",
			config: &Config{
				ClientID: "test-client-id",
			},
			wantError: "client secret is required",
		},
		{
			name: "empty_issuer",
			config: &Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
			},
			wantError: "issuer is required",
		},
		{
			name: "empty_redirect_url",
			config: &Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				Issuer:       "https://auth.example.com",
			},
			wantError: "redirect URL is required",
		},
		{
			name: "valid_config",
			config: &Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				Issuer:       "https://auth.example.com",
				RedirectURL:  "https://app.example.com/callback",
			},
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestNewProvider_InvalidConfig tests provider creation with invalid configs
func TestNewProvider_InvalidConfig(t *testing.T) {
	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	tests := []struct {
		name   string
		config *Config
	}{
		{
			name:   "nil_config_fields",
			config: &Config{},
		},
		{
			name: "missing_secret",
			config: &Config{
				ClientID:    "test-client-id",
				Issuer:      "https://auth.example.com",
				RedirectURL: "https://app.example.com/callback",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProvider(tt.config, sessionMgr)
			assert.Error(t, err)
			assert.True(t, auth.IsAuthError(err, auth.ErrOIDCAuthorizationFailed))
		})
	}
}

// TestNewProvider_DiscoveryFailure tests provider creation when discovery fails
func TestNewProvider_DiscoveryFailure(t *testing.T) {
	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		Issuer:       "http://localhost:1/nonexistent", // Will fail to connect
		RedirectURL:  "https://app.example.com/callback",
	}

	_, err := NewProvider(cfg, sessionMgr)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrOIDCAuthorizationFailed))
}

// TestProvider_Name tests provider name
func TestProvider_Name(t *testing.T) {
	p := &Provider{}
	assert.Equal(t, "oidc_auth0", p.Name())
}

// TestProvider_Authenticate tests authenticate method (should return error)
func TestProvider_Authenticate(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	req := httptest.NewRequest(http.MethodGet, "/callback", nil)
	_, err := p.Authenticate(ctx, req)

	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrInvalidCredentials))
	assert.Contains(t, err.Error(), "use HandleCallback for OIDC authentication")
}

// TestProvider_ValidateSession tests session validation (pass-through)
func TestProvider_ValidateSession(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	session := &auth.Session{
		UserID:   "user123",
		Email:    "user@example.com",
		Provider: "oidc_auth0",
	}

	err := p.ValidateSession(ctx, session)
	assert.NoError(t, err)
}

// TestProvider_Logout tests logout (pass-through)
func TestProvider_Logout(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	session := &auth.Session{
		UserID:   "user123",
		Email:    "user@example.com",
		Provider: "oidc_auth0",
	}

	err := p.Logout(ctx, session)
	assert.NoError(t, err)
}

// TestProvider_RefreshSession_NoRefreshToken tests refresh without token
func TestProvider_RefreshSession_NoRefreshToken(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	session := &auth.Session{
		UserID:       "user123",
		Email:        "user@example.com",
		Provider:     "oidc_auth0",
		RefreshToken: "", // No refresh token
	}

	_, err := p.RefreshSession(ctx, session)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrRefreshFailed))
	assert.Contains(t, err.Error(), "no refresh token available")
}

// TestPKCEStateStore tests PKCE state storage operations
func TestPKCEStateStore(t *testing.T) {
	store := newPKCEStateStore()

	// Test Store and Get
	state := "test-state-123"
	pkce := &pkceState{
		CodeVerifier: "test-verifier",
		CreatedAt:    time.Now(),
	}

	store.Store(state, pkce)

	retrieved, exists := store.Get(state)
	assert.True(t, exists)
	assert.Equal(t, "test-verifier", retrieved.CodeVerifier)

	// Test Get non-existent
	_, exists = store.Get("nonexistent")
	assert.False(t, exists)

	// Test Delete
	store.Delete(state)
	_, exists = store.Get(state)
	assert.False(t, exists)
}

// TestPKCEStateStore_Cleanup tests automatic cleanup of old states
func TestPKCEStateStore_Cleanup(t *testing.T) {
	store := &pkceStateStore{
		states: make(map[string]*pkceState),
	}

	// Add an old state (15 minutes old)
	oldState := "old-state"
	store.states[oldState] = &pkceState{
		CodeVerifier: "old-verifier",
		CreatedAt:    time.Now().Add(-15 * time.Minute),
	}

	// Add a fresh state
	freshState := "fresh-state"
	store.states[freshState] = &pkceState{
		CodeVerifier: "fresh-verifier",
		CreatedAt:    time.Now(),
	}

	// Run cleanup
	store.cleanup()

	// Old state should be removed
	_, exists := store.Get(oldState)
	assert.False(t, exists)

	// Fresh state should still exist
	_, exists = store.Get(freshState)
	assert.True(t, exists)
}

// TestGenerateRandomString tests random string generation
func TestGenerateRandomString(t *testing.T) {
	// Test generation
	str1, err := generateRandomString(32)
	require.NoError(t, err)
	assert.NotEmpty(t, str1)

	// Test uniqueness
	str2, err := generateRandomString(32)
	require.NoError(t, err)
	assert.NotEqual(t, str1, str2)

	// Test different lengths
	shortStr, err := generateRandomString(16)
	require.NoError(t, err)
	assert.True(t, len(shortStr) >= 16) // Base64 encoding makes it slightly longer
}

// TestGeneratePKCEChallenge tests PKCE challenge generation
func TestGeneratePKCEChallenge(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

	challenge := generatePKCEChallenge(verifier)

	// Verify it's SHA256 base64url encoded
	expectedHash := sha256.Sum256([]byte(verifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(expectedHash[:])

	assert.Equal(t, expectedChallenge, challenge)
}

// TestHandleCallback_MissingCode tests callback without authorization code
func TestHandleCallback_MissingCode(t *testing.T) {
	p := &Provider{
		pkceStates: newPKCEStateStore(),
		config:     &Config{UsePKCE: true},
	}

	req := httptest.NewRequest(http.MethodGet, "/callback?state=test-state", nil)
	rr := httptest.NewRecorder()

	_, err := p.HandleCallback(rr, req)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrOIDCAuthorizationFailed))
	assert.Contains(t, err.Error(), "missing authorization code")
}

// TestHandleCallback_MissingState tests callback without state parameter
func TestHandleCallback_MissingState(t *testing.T) {
	p := &Provider{
		pkceStates: newPKCEStateStore(),
		config:     &Config{UsePKCE: true},
	}

	req := httptest.NewRequest(http.MethodGet, "/callback?code=auth-code", nil)
	rr := httptest.NewRecorder()

	_, err := p.HandleCallback(rr, req)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrOIDCAuthorizationFailed))
	assert.Contains(t, err.Error(), "missing state parameter")
}

// TestHandleCallback_InvalidState tests callback with invalid/expired state
func TestHandleCallback_InvalidState(t *testing.T) {
	p := &Provider{
		pkceStates: newPKCEStateStore(),
		config:     &Config{UsePKCE: true},
	}

	req := httptest.NewRequest(http.MethodGet, "/callback?code=auth-code&state=unknown-state", nil)
	rr := httptest.NewRecorder()

	_, err := p.HandleCallback(rr, req)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrOIDCAuthorizationFailed))
	assert.Contains(t, err.Error(), "invalid or expired state")
}

// TestConfig_DefaultScopes tests default scopes configuration
func TestConfig_DefaultScopes(t *testing.T) {
	cfg := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Issuer:       "https://auth.example.com",
		RedirectURL:  "https://app.example.com/callback",
		Scopes:       nil, // Empty - should use defaults
	}

	// Validate config first
	err := validateConfig(cfg)
	require.NoError(t, err)

	// Note: The actual default scopes are set in NewProvider
	// We can verify the behavior by checking that empty scopes don't cause validation error
}

// TestConfig_CustomScopes tests custom scopes configuration
func TestConfig_CustomScopes(t *testing.T) {
	cfg := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Issuer:       "https://auth.example.com",
		RedirectURL:  "https://app.example.com/callback",
		Scopes:       []string{"openid", "profile", "email", "custom:scope"},
	}

	err := validateConfig(cfg)
	require.NoError(t, err)
	assert.Equal(t, 4, len(cfg.Scopes))
}

// TestConfig_PKCEEnabled tests PKCE configuration
func TestConfig_PKCEEnabled(t *testing.T) {
	cfg := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Issuer:       "https://auth.example.com",
		RedirectURL:  "https://app.example.com/callback",
		UsePKCE:      true,
	}

	err := validateConfig(cfg)
	require.NoError(t, err)
	assert.True(t, cfg.UsePKCE)
}

// TestConfig_PKCEDisabled tests configuration with PKCE disabled
func TestConfig_PKCEDisabled(t *testing.T) {
	cfg := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Issuer:       "https://auth.example.com",
		RedirectURL:  "https://app.example.com/callback",
		UsePKCE:      false,
	}

	err := validateConfig(cfg)
	require.NoError(t, err)
	// Note: UsePKCE will be set to true in NewProvider as a default
}

// MockOIDCServer creates a mock OIDC provider server for testing
// This is a helper for integration-style tests
type MockOIDCServer struct {
	*httptest.Server
	tokenEndpoint     string
	authorizationURL  string
	userInfoEndpoint  string
	jwksURI           string
	discoveryEndpoint string
}

// NewMockOIDCServer creates a new mock OIDC server
func NewMockOIDCServer() *MockOIDCServer {
	mock := &MockOIDCServer{}

	mux := http.NewServeMux()

	// Discovery endpoint
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"issuer": "` + mock.Server.URL + `",
			"authorization_endpoint": "` + mock.Server.URL + `/authorize",
			"token_endpoint": "` + mock.Server.URL + `/token",
			"userinfo_endpoint": "` + mock.Server.URL + `/userinfo",
			"jwks_uri": "` + mock.Server.URL + `/jwks"
		}`))
	})

	// JWKS endpoint (empty for testing)
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"keys": []}`))
	})

	mock.Server = httptest.NewServer(mux)
	mock.discoveryEndpoint = mock.Server.URL + "/.well-known/openid-configuration"
	mock.authorizationURL = mock.Server.URL + "/authorize"
	mock.tokenEndpoint = mock.Server.URL + "/token"
	mock.userInfoEndpoint = mock.Server.URL + "/userinfo"
	mock.jwksURI = mock.Server.URL + "/jwks"

	return mock
}

// TestNewProvider_WithMockServer tests provider creation with mock OIDC server
func TestNewProvider_WithMockServer(t *testing.T) {
	mockServer := NewMockOIDCServer()
	defer mockServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		Issuer:       mockServer.URL, // Use mock server
		RedirectURL:  "https://app.example.com/callback",
	}

	provider, err := NewProvider(cfg, sessionMgr)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "oidc_auth0", provider.Name())
}

// TestInitiateLogin_WithPKCE tests login initiation with PKCE
func TestInitiateLogin_WithPKCE(t *testing.T) {
	mockServer := NewMockOIDCServer()
	defer mockServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		Issuer:       mockServer.URL,
		RedirectURL:  "https://app.example.com/callback",
		UsePKCE:      true,
	}

	provider, err := NewProvider(cfg, sessionMgr)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()

	err = provider.InitiateLogin(rr, req)
	require.NoError(t, err)

	// Check redirect
	assert.Equal(t, http.StatusFound, rr.Code)
	location := rr.Header().Get("Location")

	// Parse the redirect URL
	redirectURL, err := url.Parse(location)
	require.NoError(t, err)

	// Verify PKCE parameters
	assert.NotEmpty(t, redirectURL.Query().Get("code_challenge"))
	assert.Equal(t, "S256", redirectURL.Query().Get("code_challenge_method"))
	assert.NotEmpty(t, redirectURL.Query().Get("state"))
}

// TestInitiateLogin_WithoutPKCE tests login initiation without PKCE
func TestInitiateLogin_WithoutPKCE(t *testing.T) {
	mockServer := NewMockOIDCServer()
	defer mockServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		Issuer:       mockServer.URL,
		RedirectURL:  "https://app.example.com/callback",
		UsePKCE:      false, // Note: This will be set to true by NewProvider
	}

	provider, err := NewProvider(cfg, sessionMgr)
	require.NoError(t, err)

	// Manually disable PKCE for this test
	provider.config.UsePKCE = false

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()

	err = provider.InitiateLogin(rr, req)
	require.NoError(t, err)

	// Check redirect
	assert.Equal(t, http.StatusFound, rr.Code)
	location := rr.Header().Get("Location")

	// Parse the redirect URL
	redirectURL, err := url.Parse(location)
	require.NoError(t, err)

	// Verify no PKCE parameters (when disabled)
	assert.Empty(t, redirectURL.Query().Get("code_challenge"))
	assert.Empty(t, redirectURL.Query().Get("code_challenge_method"))
	// State should still be present for CSRF protection
	assert.NotEmpty(t, redirectURL.Query().Get("state"))
}

// TestHandleCallback_PKCEStateCleanup tests that PKCE state is cleaned up after use
func TestHandleCallback_PKCEStateCleanup(t *testing.T) {
	// Use mock server to create a valid provider
	mockServer := NewMockOIDCServer()
	defer mockServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		Issuer:       mockServer.URL,
		RedirectURL:  "https://app.example.com/callback",
		UsePKCE:      true,
	}

	p, err := NewProvider(cfg, sessionMgr)
	require.NoError(t, err)

	// Add a PKCE state
	state := "test-state-cleanup"
	p.pkceStates.Store(state, &pkceState{
		CodeVerifier: "test-verifier",
		CreatedAt:    time.Now(),
	})

	// Verify it exists
	_, exists := p.pkceStates.Get(state)
	assert.True(t, exists)

	// HandleCallback will fail at token exchange, but should clean up state first
	req := httptest.NewRequest(http.MethodGet, "/callback?code=test&state="+state, nil)
	rr := httptest.NewRecorder()

	// This will fail at token exchange, but state should be deleted
	_, callbackErr := p.HandleCallback(rr, req)
	// We expect an error because the mock server doesn't handle token exchange
	assert.Error(t, callbackErr)

	// Verify state was deleted after HandleCallback attempt
	_, exists = p.pkceStates.Get(state)
	assert.False(t, exists)
}

// TestRefreshSession_InvalidRefreshToken tests refresh with invalid token
func TestRefreshSession_InvalidRefreshToken(t *testing.T) {
	mockServer := NewMockOIDCServer()
	defer mockServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		Issuer:       mockServer.URL,
		RedirectURL:  "https://app.example.com/callback",
	}

	provider, err := NewProvider(cfg, sessionMgr)
	require.NoError(t, err)

	session := &auth.Session{
		UserID:       "user123",
		Email:        "user@example.com",
		Provider:     "oidc_auth0",
		RefreshToken: "invalid-refresh-token",
	}

	ctx := context.Background()
	_, err = provider.RefreshSession(ctx, session)

	// Should fail because the mock server doesn't handle token refresh
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrRefreshFailed))
}
