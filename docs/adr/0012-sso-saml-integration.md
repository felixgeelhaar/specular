# ADR-0012: SSO/SAML Integration Architecture

**Status:** Proposed
**Date:** 2025-11-20
**Deciders:** Engineering Team
**Related:** [ADR-0011](0011-v2-architecture-enterprise-readiness.md) (M9.2: Enterprise Security)

## Context

As part of the v2.0 enterprise readiness initiative (M9.2: Enterprise Security), Specular needs to support Single Sign-On (SSO) integration with enterprise identity providers. This enables organizations to manage user authentication centrally and enforce their security policies.

### Requirements

**Primary Goals:**
- Support SAML 2.0 authentication with Okta
- Support OAuth2/OIDC authentication with Auth0
- Integrate with existing JWT-based authentication system
- Enable SSO for the `serve` command (health check endpoints)
- Maintain backward compatibility with email/password authentication

**Enterprise Identity Providers:**
- **Okta**: Industry-leading SAML provider used by enterprises
- **Auth0**: Modern authentication platform with OAuth2/OIDC support

**Security Requirements:**
- Validate SAML assertions cryptographically
- Verify JWT tokens from OAuth2/OIDC flows
- Implement secure session management
- Support multi-tenant authentication (future: M10.1)

## Decision

We will implement a **pluggable authentication architecture** that supports multiple authentication methods:

1. **Legacy authentication**: Email/password with JWT tokens (existing)
2. **SAML 2.0**: Enterprise SSO via Okta
3. **OAuth2/OIDC**: Modern SSO via Auth0

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Specular CLI/Server                      │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │            Authentication Middleware                    │ │
│  │                                                         │ │
│  │  ┌───────────┐  ┌──────────┐  ┌──────────────┐        │ │
│  │  │  Legacy   │  │   SAML   │  │  OAuth2/OIDC │        │ │
│  │  │  (JWT)    │  │  (Okta)  │  │   (Auth0)    │        │ │
│  │  └─────┬─────┘  └────┬─────┘  └──────┬───────┘        │ │
│  │        │             │                │                │ │
│  │        └─────────────┴────────────────┘                │ │
│  │                      │                                  │ │
│  │               ┌──────▼──────┐                          │ │
│  │               │   Session   │                          │ │
│  │               │   Manager   │                          │ │
│  │               └──────┬──────┘                          │ │
│  └──────────────────────┼───────────────────────────────────┘
│                         │                                   │
│         ┌───────────────▼────────────────┐                  │
│         │   HTTP Server (health probes)  │                  │
│         │   /health/live, /health/ready  │                  │
│         └────────────────────────────────┘                  │
└─────────────────────────────────────────────────────────────┘
                          │
          ┌───────────────┴───────────────┐
          │                               │
    ┌─────▼──────┐                  ┌────▼─────┐
    │    Okta    │                  │  Auth0   │
    │   (SAML)   │                  │ (OIDC)   │
    └────────────┘                  └──────────┘
```

### Package Structure

```
internal/auth/
├── auth.go              # Core authentication interfaces
├── middleware.go        # HTTP middleware for authentication
├── session.go           # Session management (JWT-based)
├── saml/
│   ├── provider.go      # SAML service provider implementation
│   ├── okta.go          # Okta-specific configuration
│   ├── metadata.go      # SAML metadata parsing
│   └── saml_test.go     # SAML integration tests
├── oidc/
│   ├── provider.go      # OAuth2/OIDC client implementation
│   ├── auth0.go         # Auth0-specific configuration
│   ├── token.go         # Token validation and refresh
│   └── oidc_test.go     # OIDC integration tests
└── config.go            # Authentication configuration
```

### Core Interfaces

```go
package auth

import (
	"context"
	"net/http"
)

// Authenticator defines the interface for all authentication providers
type Authenticator interface {
	// Name returns the authentication provider name
	Name() string

	// Authenticate validates credentials and returns a session
	Authenticate(ctx context.Context, req *http.Request) (*Session, error)

	// ValidateSession checks if a session is still valid
	ValidateSession(ctx context.Context, session *Session) error

	// RefreshSession attempts to refresh an expired session
	RefreshSession(ctx context.Context, session *Session) (*Session, error)

	// Logout terminates a session
	Logout(ctx context.Context, session *Session) error
}

// Session represents an authenticated user session
type Session struct {
	// UserID is the unique identifier for the user
	UserID string

	// Email is the user's email address
	Email string

	// Provider is the authentication provider that created this session
	Provider string

	// Token is the authentication token (JWT, SAML assertion, etc.)
	Token string

	// RefreshToken is used to obtain a new access token
	RefreshToken string

	// ExpiresAt is when the session expires
	ExpiresAt time.Time

	// Attributes contains provider-specific user attributes
	Attributes map[string]interface{}
}

// Manager coordinates multiple authentication providers
type Manager struct {
	providers map[string]Authenticator
	sessions  SessionStore
}

// NewManager creates a new authentication manager
func NewManager(store SessionStore) *Manager

// Register registers an authentication provider
func (m *Manager) Register(auth Authenticator) error

// Authenticate attempts authentication with all registered providers
func (m *Manager) Authenticate(ctx context.Context, req *http.Request) (*Session, error)

// ValidateSession validates a session token
func (m *Manager) ValidateSession(ctx context.Context, token string) (*Session, error)
```

### SAML 2.0 Implementation (Okta)

**SAML Flow:**

1. User accesses protected endpoint
2. Server redirects to Okta SAML SSO URL
3. User authenticates with Okta
4. Okta returns SAML assertion to Server callback URL
5. Server validates SAML assertion signature
6. Server creates JWT session token
7. Server returns session token to client

**Key Components:**

```go
package saml

import (
	"crypto/x509"
	"encoding/xml"
)

// Config holds SAML service provider configuration
type Config struct {
	// EntityID is the unique identifier for this service provider
	EntityID string

	// AssertionConsumerServiceURL is where SAML responses are sent
	AssertionConsumerServiceURL string

	// SingleLogoutServiceURL for logout requests
	SingleLogoutServiceURL string

	// IDPMetadataURL is the Okta metadata endpoint
	IDPMetadataURL string

	// Certificate for signing SAML requests
	Certificate *x509.Certificate

	// PrivateKey for signing SAML requests
	PrivateKey interface{}
}

// Provider implements SAML service provider
type Provider struct {
	config     *Config
	idpMetadata *IDPMetadata
}

// NewProvider creates a SAML provider
func NewProvider(cfg *Config) (*Provider, error)

// InitiateLogin starts the SAML authentication flow
func (p *Provider) InitiateLogin(w http.ResponseWriter, r *http.Request) error

// HandleCallback processes SAML assertions from IdP
func (p *Provider) HandleCallback(w http.ResponseWriter, r *http.Request) (*SAMLAssertion, error)

// ValidateAssertion cryptographically validates a SAML assertion
func (p *Provider) ValidateAssertion(assertion *SAMLAssertion) error
```

**Okta Configuration:**

```yaml
# .specular/auth_config.yaml
auth:
  providers:
    - name: okta_saml
      type: saml
      enabled: true
      config:
        entity_id: https://specular.example.com/saml/metadata
        assertion_consumer_service_url: https://specular.example.com/saml/acs
        single_logout_service_url: https://specular.example.com/saml/slo
        idp_metadata_url: https://dev-12345.okta.com/app/abc123/sso/saml/metadata
        certificate_path: /path/to/sp-cert.pem
        private_key_path: /path/to/sp-key.pem
```

### OAuth2/OIDC Implementation (Auth0)

**OIDC Flow (Authorization Code Flow with PKCE):**

1. User accesses protected endpoint
2. Server generates PKCE code verifier and challenge
3. Server redirects to Auth0 authorization endpoint
4. User authenticates with Auth0
5. Auth0 redirects back with authorization code
6. Server exchanges code for tokens using PKCE verifier
7. Server validates ID token signature and claims
8. Server creates session from ID token
9. Server returns session token to client

**Key Components:**

```go
package oidc

import (
	"context"
	"golang.org/x/oauth2"
)

// Config holds OAuth2/OIDC configuration
type Config struct {
	// ClientID from Auth0 application
	ClientID string

	// ClientSecret from Auth0 application
	ClientSecret string

	// Issuer is the Auth0 domain (e.g., https://dev-abc123.us.auth0.com/)
	Issuer string

	// RedirectURL where Auth0 sends the authorization code
	RedirectURL string

	// Scopes to request (openid, profile, email)
	Scopes []string
}

// Provider implements OAuth2/OIDC client
type Provider struct {
	config       *Config
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

// NewProvider creates an OIDC provider
func NewProvider(cfg *Config) (*Provider, error)

// InitiateLogin starts the OAuth2/OIDC flow with PKCE
func (p *Provider) InitiateLogin(w http.ResponseWriter, r *http.Request) error

// HandleCallback processes authorization code and exchanges for tokens
func (p *Provider) HandleCallback(w http.ResponseWriter, r *http.Request) (*TokenResponse, error)

// ValidateToken validates an ID token
func (p *Provider) ValidateToken(ctx context.Context, rawToken string) (*IDToken, error)

// RefreshToken uses refresh token to obtain new access token
func (p *Provider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
```

**Auth0 Configuration:**

```yaml
# .specular/auth_config.yaml
auth:
  providers:
    - name: auth0_oidc
      type: oidc
      enabled: true
      config:
        client_id: YOUR_AUTH0_CLIENT_ID
        client_secret: YOUR_AUTH0_CLIENT_SECRET
        issuer: https://dev-abc123.us.auth0.com/
        redirect_url: https://specular.example.com/auth/callback
        scopes:
          - openid
          - profile
          - email
```

### Session Management

**JWT-Based Sessions:**

All authentication providers (legacy, SAML, OIDC) ultimately create a standardized JWT session token:

```go
package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

// SessionClaims represents JWT claims for a session
type SessionClaims struct {
	jwt.RegisteredClaims

	// UserID is the unique user identifier
	UserID string `json:"user_id"`

	// Email is the user's email address
	Email string `json:"email"`

	// Provider is the authentication provider
	Provider string `json:"provider"`

	// Attributes contains provider-specific claims
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// SessionManager handles session creation and validation
type SessionManager struct {
	signingKey []byte
	issuer     string
}

// CreateSession creates a new JWT session token
func (sm *SessionManager) CreateSession(s *Session) (string, error)

// ValidateSession validates and parses a JWT session token
func (sm *SessionManager) ValidateSession(token string) (*Session, error)
```

**Session Storage:**

For the `serve` command with health endpoints, sessions are stored in-memory with optional Redis backend for distributed deployments:

```go
// SessionStore defines the interface for session persistence
type SessionStore interface {
	// Store saves a session
	Store(ctx context.Context, sessionID string, session *Session) error

	// Get retrieves a session
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Delete removes a session
	Delete(ctx context.Context, sessionID string) error

	// Cleanup removes expired sessions
	Cleanup(ctx context.Context) error
}

// MemoryStore implements in-memory session storage
type MemoryStore struct {
	sessions sync.Map
}

// RedisStore implements Redis-backed session storage (future: M10.2)
type RedisStore struct {
	client *redis.Client
}
```

### Authentication Middleware

HTTP middleware integrates authentication into the serve command:

```go
package auth

import (
	"net/http"
)

// Middleware is HTTP middleware for authentication
type Middleware struct {
	manager *Manager
}

// NewMiddleware creates authentication middleware
func NewMiddleware(manager *Manager) *Middleware

// Protect wraps an HTTP handler with authentication
func (m *Middleware) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header or cookie
		token := extractToken(r)
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate session
		session, err := m.manager.ValidateSession(r.Context(), token)
		if err != nil {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}

		// Add session to request context
		ctx := context.WithValue(r.Context(), sessionKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Optional returns middleware that allows both authenticated and anonymous access
func (m *Middleware) Optional(next http.Handler) http.Handler

// extractToken extracts the authentication token from the request
func extractToken(r *http.Request) string
```

### Integration with Serve Command

Update `internal/cmd/serve.go` to support authentication:

```go
package cmd

import (
	"github.com/felixgeelhaar/specular/internal/auth"
	"github.com/felixgeelhaar/specular/internal/auth/saml"
	"github.com/felixgeelhaar/specular/internal/auth/oidc"
	"github.com/felixgeelhaar/specular/internal/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server with health endpoints",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load authentication configuration
		authConfig, err := loadAuthConfig()
		if err != nil {
			return err
		}

		// Create authentication manager
		authManager := auth.NewManager(auth.NewMemoryStore())

		// Register SAML provider if configured
		if authConfig.SAML.Enabled {
			samlProvider, err := saml.NewProvider(&authConfig.SAML)
			if err != nil {
				return err
			}
			authManager.Register(samlProvider)
		}

		// Register OIDC provider if configured
		if authConfig.OIDC.Enabled {
			oidcProvider, err := oidc.NewProvider(&authConfig.OIDC)
			if err != nil {
				return err
			}
			authManager.Register(oidcProvider)
		}

		// Create authentication middleware
		authMiddleware := auth.NewMiddleware(authManager)

		// Create server with authentication
		srv := server.NewServer(probeManager, cfg)
		srv.SetAuthMiddleware(authMiddleware)

		return srv.Start()
	},
}
```

Update `internal/server/server.go` to support optional authentication:

```go
package server

import (
	"github.com/felixgeelhaar/specular/internal/auth"
)

type Server struct {
	// ... existing fields ...
	authMiddleware *auth.Middleware
}

// SetAuthMiddleware configures authentication for protected endpoints
func (s *Server) SetAuthMiddleware(middleware *auth.Middleware) {
	s.authMiddleware = middleware
}

func NewServer(probeManager *health.ProbeManager, cfg Config) *Server {
	// ... existing code ...

	// Health endpoints remain unauthenticated (for Kubernetes)
	mux.HandleFunc("/health/live", s.handleLiveness)
	mux.HandleFunc("/health/ready", s.handleReadiness)
	mux.HandleFunc("/health/startup", s.handleStartup)

	// Authentication endpoints
	mux.HandleFunc("/auth/login", s.handleAuthLogin)
	mux.HandleFunc("/auth/saml/initiate", s.handleSAMLInitiate)
	mux.HandleFunc("/auth/saml/acs", s.handleSAMLCallback)
	mux.HandleFunc("/auth/oidc/initiate", s.handleOIDCInitiate)
	mux.HandleFunc("/auth/oidc/callback", s.handleOIDCCallback)
	mux.HandleFunc("/auth/logout", s.handleLogout)

	// Future: Protected endpoints with authentication
	// mux.Handle("/api/v2/...", s.authMiddleware.Protect(handler))

	// ... rest of existing code ...
}
```

## Implementation Plan

### Phase 1: Core Authentication Infrastructure (Week 1)

1. **Create `internal/auth` package**
   - Define core interfaces (Authenticator, Session, Manager)
   - Implement SessionManager with JWT support
   - Implement MemoryStore for session storage
   - Write unit tests

2. **Create authentication middleware**
   - Implement HTTP middleware for token validation
   - Add context-based session injection
   - Write middleware tests

### Phase 2: SAML Implementation (Week 2)

3. **Implement SAML service provider**
   - Create `internal/auth/saml` package
   - Implement SAML metadata parsing
   - Implement SAML assertion validation
   - Add Okta-specific configuration

4. **Add SAML endpoints to serve command**
   - `/auth/saml/metadata` - Service provider metadata
   - `/auth/saml/initiate` - Initiate SAML login
   - `/auth/saml/acs` - Assertion Consumer Service
   - `/auth/saml/slo` - Single Logout Service

5. **SAML testing**
   - Unit tests for SAML assertion validation
   - Integration tests with Okta sandbox
   - End-to-end authentication flow tests

### Phase 3: OAuth2/OIDC Implementation (Week 3)

6. **Implement OAuth2/OIDC client**
   - Create `internal/auth/oidc` package
   - Implement PKCE flow
   - Implement token validation
   - Add Auth0-specific configuration

7. **Add OIDC endpoints to serve command**
   - `/auth/oidc/initiate` - Initiate OAuth2 flow
   - `/auth/oidc/callback` - Handle authorization code
   - `/auth/oidc/refresh` - Refresh access token

8. **OIDC testing**
   - Unit tests for token validation
   - Integration tests with Auth0 sandbox
   - PKCE flow tests

### Phase 4: Integration and Documentation (Week 4)

9. **CLI integration**
   - Add `--auth-config` flag to serve command
   - Update `specular auth` commands for SSO
   - Add `specular auth sso login` command

10. **Documentation**
    - Okta integration guide
    - Auth0 integration guide
    - Configuration examples
    - Kubernetes deployment with SSO
    - Security best practices

11. **Kubernetes deployment updates**
    - Update deployment manifests for SSO
    - Add auth configuration via ConfigMap/Secret
    - Document cert management for SAML

## Consequences

### Benefits

1. **Enterprise Adoption**: Support for enterprise identity providers enables Fortune 500 adoption
2. **Security**: Centralized authentication and SSO enforce organizational security policies
3. **User Experience**: Users authenticate once via SSO instead of managing separate credentials
4. **Flexibility**: Pluggable architecture supports future authentication providers
5. **Standards Compliance**: SAML 2.0 and OIDC are industry-standard protocols

### Drawbacks

1. **Complexity**: Additional authentication flows increase code complexity
2. **Testing Overhead**: Integration testing requires Okta and Auth0 sandboxes
3. **Certificate Management**: SAML requires managing X.509 certificates
4. **Backward Compatibility**: Must maintain legacy email/password authentication

### Risks

1. **SAML Signature Validation**: Improper signature validation could lead to security vulnerabilities
   - **Mitigation**: Use battle-tested SAML libraries, comprehensive security testing

2. **Token Lifetime Management**: Long-lived tokens could be security risks
   - **Mitigation**: Short-lived access tokens, refresh token rotation

3. **Configuration Complexity**: Misconfigured SSO could lock out users
   - **Mitigation**: Clear documentation, validation, health checks

## Dependencies

**Go Libraries:**
- [`github.com/crewjam/saml`](https://github.com/crewjam/saml) - SAML service provider library
- [`github.com/coreos/go-oidc/v3`](https://github.com/coreos/go-oidc) - OIDC client library
- [`golang.org/x/oauth2`](https://pkg.go.dev/golang.org/x/oauth2) - OAuth2 client
- [`github.com/golang-jwt/jwt/v5`](https://github.com/golang-jwt/jwt) - JWT handling

**External Services:**
- **Okta Developer Account**: For SAML testing and integration
- **Auth0 Account**: For OIDC testing and integration

## Testing Strategy

1. **Unit Tests**:
   - SAML assertion parsing and validation
   - OIDC token validation and refresh
   - JWT session creation and validation
   - Authentication middleware logic

2. **Integration Tests**:
   - Okta SAML authentication flow
   - Auth0 OIDC authentication flow
   - Session management and expiration
   - Multi-provider authentication

3. **End-to-End Tests**:
   - Full SSO login flow with UI
   - Token refresh and logout
   - Protected endpoint access
   - Cross-browser compatibility

4. **Security Tests**:
   - SAML signature validation bypass attempts
   - Token tampering detection
   - Session hijacking prevention
   - CSRF protection

## Alternatives Considered

### Alternative 1: Use External Authentication Proxy (OAuth2 Proxy, Keycloak)

**Pros:**
- Offload authentication to dedicated service
- Mature, battle-tested solutions
- Support for multiple providers out of the box

**Cons:**
- Additional infrastructure dependency
- Less control over authentication flow
- Harder to customize for specific use cases
- Additional operational complexity

**Rejected:** We want full control over the authentication experience and minimal external dependencies for the CLI.

### Alternative 2: Support Only OIDC (Not SAML)

**Pros:**
- Simpler implementation (only one protocol)
- OIDC is more modern than SAML
- Fewer libraries and dependencies

**Cons:**
- Many enterprises still mandate SAML
- Limits adoption by large organizations
- Okta supports both, but some legacy systems only support SAML

**Rejected:** Enterprise adoption requires SAML support due to existing infrastructure.

### Alternative 3: Delegate All Authentication to Backend API

**Pros:**
- Single source of truth for authentication
- CLI is stateless (no token storage)
- Centralized session management

**Cons:**
- Requires backend API for all operations
- Network dependency for authentication checks
- Doesn't work for standalone CLI usage

**Rejected:** Specular CLI should work independently without requiring a backend API.

## References

- [ADR-0011: v2.0 Architecture - Enterprise Readiness](0011-v2-architecture-enterprise-readiness.md)
- [SAML 2.0 Specification](https://docs.oasis-open.org/security/saml/v2.0/)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [OAuth 2.0 RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749)
- [PKCE RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636)
- [Okta SAML Documentation](https://developer.okta.com/docs/guides/saml-application-setup/)
- [Auth0 OIDC Documentation](https://auth0.com/docs/authenticate/protocols/openid-connect-protocol)

## Approval

**Proposed By:** Engineering Team
**Review Date:** TBD
**Approved By:** TBD
**Implementation Target:** v2.0 M9.2.1

## Changelog

- **2025-11-20**: Initial proposal created
