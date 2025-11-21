# SSO/SAML Integration Guide

This guide explains how to configure and use Single Sign-On (SSO) authentication with Specular, supporting both SAML 2.0 (Okta) and OAuth2/OIDC (Auth0).

## Table of Contents

- [Overview](#overview)
- [SAML 2.0 with Okta](#saml-20-with-okta)
- [OAuth2/OIDC with Auth0](#oauth2oidc-with-auth0)
- [Configuration](#configuration)
- [Usage Examples](#usage-examples)
- [Security Best Practices](#security-best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

Specular implements enterprise-grade SSO authentication with pluggable providers:

- **SAML 2.0 Service Provider** - For Okta and other SAML-based identity providers
- **OAuth2/OIDC Client** - For Auth0 and other OAuth2/OIDC providers
- **JWT Session Management** - Standardized session format across all providers
- **Secure Cookie Handling** - HttpOnly, Secure, SameSite protection

### Architecture

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐
│   Browser   │─────▶│  Specular    │─────▶│    IdP      │
│             │◀─────│  (Service    │◀─────│  (Okta/     │
│             │      │   Provider)  │      │   Auth0)    │
└─────────────┘      └──────────────┘      └─────────────┘
      │                     │
      │  JWT Token          │
      │  (Session)          │
      └─────────────────────┘
```

## SAML 2.0 with Okta

### Prerequisites

1. Okta account with admin access
2. SAML application created in Okta
3. X.509 certificate and private key for SAML signing

### Step 1: Create Okta SAML Application

1. Log in to Okta Admin Console
2. Navigate to **Applications** > **Create App Integration**
3. Select **SAML 2.0** and click **Next**

4. **General Settings:**
   - **App name:** Specular
   - **App logo:** (optional)

5. **SAML Settings:**
   - **Single sign-on URL:** `https://your-domain.com/auth/callback/okta_saml`
   - **Audience URI (SP Entity ID):** `https://your-domain.com/saml/metadata`
   - **Name ID format:** EmailAddress
   - **Application username:** Email

6. **Attribute Statements:**
   ```
   firstName  -> user.firstName
   lastName   -> user.lastName
   email      -> user.email
   groups     -> appuser.groups
   ```

7. Click **Next** and **Finish**

### Step 2: Download IdP Metadata

1. In the Okta application, go to the **Sign On** tab
2. Right-click **Identity Provider metadata** and save the XML file
3. Note the metadata URL (e.g., `https://your-org.okta.com/app/xxx/sso/saml/metadata`)

### Step 3: Generate SAML Certificate

```bash
# Generate private key
openssl genrsa -out saml-key.pem 2048

# Generate certificate signing request
openssl req -new -key saml-key.pem -out saml-csr.pem \
  -subj "/CN=your-domain.com"

# Generate self-signed certificate (valid for 365 days)
openssl x509 -req -in saml-csr.pem -signkey saml-key.pem \
  -out saml-cert.pem -days 365
```

### Step 4: Configure Specular

Create a configuration file `auth-config.yaml`:

```yaml
authentication:
  providers:
    - type: saml
      name: okta_saml
      entity_id: https://your-domain.com/saml/metadata
      acs_url: https://your-domain.com/auth/callback/okta_saml
      metadata_url: https://your-org.okta.com/app/xxx/sso/saml/metadata
      certificate_path: /path/to/saml-cert.pem
      private_key_path: /path/to/saml-key.pem

  session:
    signing_key: your-secret-key-at-least-32-bytes-long
    issuer: specular
    token_duration: 1h
    refresh_duration: 168h  # 7 days
```

Or use environment variables:

```bash
export SPECULAR_AUTH_SAML_ENTITY_ID=https://your-domain.com/saml/metadata
export SPECULAR_AUTH_SAML_ACS_URL=https://your-domain.com/auth/callback/okta_saml
export SPECULAR_AUTH_SAML_METADATA_URL=https://your-org.okta.com/app/xxx/sso/saml/metadata
export SPECULAR_AUTH_SAML_CERT_PATH=/path/to/saml-cert.pem
export SPECULAR_AUTH_SAML_KEY_PATH=/path/to/saml-key.pem
export SPECULAR_AUTH_SESSION_KEY=your-secret-key-at-least-32-bytes-long
```

### Step 5: Test Authentication

1. Start Specular server:
   ```bash
   specular serve --config auth-config.yaml
   ```

2. Navigate to login URL:
   ```
   http://localhost:8080/auth/login?provider=okta_saml
   ```

3. You will be redirected to Okta for authentication

4. After successful login, you'll be redirected back with a session cookie

## OAuth2/OIDC with Auth0

### Prerequisites

1. Auth0 account
2. Application created in Auth0 dashboard
3. Client ID and Client Secret

### Step 1: Create Auth0 Application

1. Log in to Auth0 Dashboard
2. Navigate to **Applications** > **Create Application**
3. Select **Regular Web Application**

4. **Settings:**
   - **Name:** Specular
   - **Application Type:** Regular Web Application
   - **Allowed Callback URLs:** `https://your-domain.com/auth/callback/auth0_oidc`
   - **Allowed Logout URLs:** `https://your-domain.com/auth/logout`
   - **Allowed Web Origins:** `https://your-domain.com`

5. Note down:
   - **Domain** (e.g., `your-tenant.auth0.com`)
   - **Client ID**
   - **Client Secret**

### Step 2: Configure Auth0 API

1. Navigate to **APIs** in Auth0 Dashboard
2. Note the **API Audience** for your API (e.g., `https://api.your-domain.com`)

### Step 3: Configure Specular

Create a configuration file `auth-config.yaml`:

```yaml
authentication:
  providers:
    - type: oidc
      name: auth0_oidc
      issuer_url: https://your-tenant.auth0.com
      client_id: your-client-id
      client_secret: your-client-secret
      redirect_url: https://your-domain.com/auth/callback/auth0_oidc
      scopes:
        - openid
        - profile
        - email
      audience: https://api.your-domain.com

  session:
    signing_key: your-secret-key-at-least-32-bytes-long
    issuer: specular
    token_duration: 1h
    refresh_duration: 168h  # 7 days
```

Or use environment variables:

```bash
export SPECULAR_AUTH_OIDC_ISSUER=https://your-tenant.auth0.com
export SPECULAR_AUTH_OIDC_CLIENT_ID=your-client-id
export SPECULAR_AUTH_OIDC_CLIENT_SECRET=your-client-secret
export SPECULAR_AUTH_OIDC_REDIRECT_URL=https://your-domain.com/auth/callback/auth0_oidc
export SPECULAR_AUTH_OIDC_AUDIENCE=https://api.your-domain.com
export SPECULAR_AUTH_SESSION_KEY=your-secret-key-at-least-32-bytes-long
```

### Step 4: Test Authentication

1. Start Specular server:
   ```bash
   specular serve --config auth-config.yaml
   ```

2. Navigate to login URL:
   ```
   http://localhost:8080/auth/login?provider=auth0_oidc
   ```

3. You will be redirected to Auth0 for authentication

4. After successful login, you'll be redirected back with a session cookie

## Configuration

### Complete Configuration Example

```yaml
authentication:
  # Multiple providers can be configured
  providers:
    # SAML 2.0 Provider (Okta)
    - type: saml
      name: okta_saml
      entity_id: https://your-domain.com/saml/metadata
      acs_url: https://your-domain.com/auth/callback/okta_saml
      metadata_url: https://your-org.okta.com/app/xxx/sso/saml/metadata
      certificate_path: /etc/specular/certs/saml-cert.pem
      private_key_path: /etc/specular/certs/saml-key.pem

    # OAuth2/OIDC Provider (Auth0)
    - type: oidc
      name: auth0_oidc
      issuer_url: https://your-tenant.auth0.com
      client_id: your-client-id
      client_secret: your-client-secret
      redirect_url: https://your-domain.com/auth/callback/auth0_oidc
      scopes:
        - openid
        - profile
        - email
        - offline_access  # For refresh tokens
      audience: https://api.your-domain.com

  # Session configuration
  session:
    # HMAC-SHA256 signing key (min 32 bytes)
    signing_key: ${SPECULAR_AUTH_SESSION_KEY}

    # JWT issuer
    issuer: specular

    # Access token duration
    token_duration: 1h

    # Refresh token duration
    refresh_duration: 168h  # 7 days

  # Cookie configuration
  cookies:
    # Use secure cookies (HTTPS only) in production
    secure: true

    # HttpOnly prevents JavaScript access
    http_only: true

    # SameSite protection (Lax or Strict)
    same_site: lax

  # Session store (memory for single instance, redis for distributed)
  store:
    type: memory
    cleanup_interval: 5m
```

### Environment Variables

All configuration can be provided via environment variables:

**SAML Configuration:**
```bash
SPECULAR_AUTH_SAML_ENTITY_ID=https://your-domain.com/saml/metadata
SPECULAR_AUTH_SAML_ACS_URL=https://your-domain.com/auth/callback/okta_saml
SPECULAR_AUTH_SAML_METADATA_URL=https://your-org.okta.com/app/xxx/sso/saml/metadata
SPECULAR_AUTH_SAML_CERT_PATH=/etc/specular/certs/saml-cert.pem
SPECULAR_AUTH_SAML_KEY_PATH=/etc/specular/certs/saml-key.pem
```

**OIDC Configuration:**
```bash
SPECULAR_AUTH_OIDC_ISSUER=https://your-tenant.auth0.com
SPECULAR_AUTH_OIDC_CLIENT_ID=your-client-id
SPECULAR_AUTH_OIDC_CLIENT_SECRET=your-client-secret
SPECULAR_AUTH_OIDC_REDIRECT_URL=https://your-domain.com/auth/callback/auth0_oidc
SPECULAR_AUTH_OIDC_AUDIENCE=https://api.your-domain.com
SPECULAR_AUTH_OIDC_SCOPES=openid,profile,email,offline_access
```

**Session Configuration:**
```bash
SPECULAR_AUTH_SESSION_KEY=your-secret-key-at-least-32-bytes-long
SPECULAR_AUTH_SESSION_ISSUER=specular
SPECULAR_AUTH_TOKEN_DURATION=1h
SPECULAR_AUTH_REFRESH_DURATION=168h
```

## Usage Examples

### Programmatic Usage

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/felixgeelhaar/specular/internal/auth"
	"github.com/felixgeelhaar/specular/internal/auth/saml"
	"github.com/felixgeelhaar/specular/internal/auth/oidc"
)

func main() {
	// Create session manager
	signingKey := []byte("your-secret-key-at-least-32-bytes-long")
	sessionMgr := auth.NewSessionManager(signingKey, "specular")

	// Create session store
	sessionStore := auth.NewMemoryStore()

	// Create auth manager
	authManager := auth.NewManager(sessionStore)

	// Register SAML provider
	samlProvider, err := saml.NewProvider(&saml.Config{
		EntityID:     "https://your-domain.com/saml/metadata",
		ACSURL:       "https://your-domain.com/auth/callback/okta_saml",
		MetadataURL:  "https://your-org.okta.com/app/xxx/sso/saml/metadata",
		CertPath:     "/etc/specular/certs/saml-cert.pem",
		KeyPath:      "/etc/specular/certs/saml-key.pem",
	}, sessionMgr)
	if err != nil {
		log.Fatal(err)
	}
	authManager.Register(samlProvider)

	// Register OIDC provider
	oidcProvider, err := oidc.NewProvider(&oidc.Config{
		IssuerURL:    "https://your-tenant.auth0.com",
		ClientID:     "your-client-id",
		ClientSecret: "your-client-secret",
		RedirectURL:  "https://your-domain.com/auth/callback/auth0_oidc",
		Scopes:       []string{"openid", "profile", "email"},
	}, sessionMgr)
	if err != nil {
		log.Fatal(err)
	}
	authManager.Register(oidcProvider)

	// Create HTTP handlers
	handlers := auth.NewHandlers(authManager, sessionMgr, sessionStore, true)
	middleware := auth.NewMiddleware(authManager, sessionMgr)

	// Setup routes
	http.HandleFunc("/auth/login", handlers.HandleLogin)
	http.HandleFunc("/auth/callback", handlers.HandleCallback)
	http.HandleFunc("/auth/logout", handlers.HandleLogout)
	http.HandleFunc("/auth/refresh", handlers.HandleRefresh)
	http.HandleFunc("/auth/me", handlers.HandleMe)

	// Protected route
	http.Handle("/api/data", middleware.RequireAuth(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := auth.MustGetSession(r.Context())
			// Access user info: session.UserID, session.Email, session.Attributes
			w.Write([]byte("Hello, " + session.Email))
		}),
	))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Protecting Routes

```go
// Require authentication
http.Handle("/api/protected", middleware.RequireAuth(handler))

// Optional authentication (allow both authenticated and unauthenticated)
http.Handle("/api/public", middleware.OptionalAuth(handler))

// Access session in handler
func handler(w http.ResponseWriter, r *http.Request) {
	session := auth.GetSession(r.Context())
	if session != nil {
		// User is authenticated
		fmt.Fprintf(w, "Hello, %s", session.Email)
	} else {
		// User is not authenticated
		fmt.Fprintf(w, "Hello, guest")
	}
}
```

### Refresh Token Flow

```bash
# Client sends refresh token
curl -X POST https://your-domain.com/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "your-refresh-token"}'

# Response includes new access and refresh tokens
{
  "success": true,
  "access_token": "new-jwt-token",
  "refresh_token": "new-refresh-token",
  "expires_at": 1640995200
}
```

## Security Best Practices

### 1. Secret Management

- **Never commit secrets to version control**
- Use environment variables or secret management systems (Vault, AWS Secrets Manager)
- Rotate signing keys regularly (at least annually)
- Use different keys for development, staging, and production

```bash
# Generate secure random key
openssl rand -base64 32
```

### 2. HTTPS Only

- Always use HTTPS in production
- Set `secure: true` for cookies
- Configure HSTS headers

```yaml
cookies:
  secure: true  # Cookies only sent over HTTPS
```

### 3. Token Expiration

- Use short-lived access tokens (1 hour recommended)
- Use longer-lived refresh tokens (7 days recommended)
- Implement token rotation for refresh tokens

```yaml
session:
  token_duration: 1h        # Access token
  refresh_duration: 168h    # Refresh token (7 days)
```

### 4. PKCE for OAuth2

PKCE (Proof Key for Code Exchange) is automatically enabled for OIDC flows to prevent authorization code interception attacks.

### 5. SAML Certificate Management

- Store certificates securely (file permissions 600)
- Use strong key sizes (2048-bit RSA minimum)
- Rotate certificates before expiration

```bash
# Set proper permissions
chmod 600 /etc/specular/certs/saml-key.pem
chmod 644 /etc/specular/certs/saml-cert.pem
chown specular:specular /etc/specular/certs/*
```

### 6. Session Storage

For distributed deployments, use Redis for session storage:

```yaml
store:
  type: redis
  redis_url: redis://localhost:6379
  redis_password: ${REDIS_PASSWORD}
  key_prefix: "specular:session:"
```

### 7. Audit Logging

Enable audit logging for authentication events:

```yaml
logging:
  audit:
    enabled: true
    events:
      - login
      - logout
      - token_refresh
      - session_expired
```

## Troubleshooting

### SAML Issues

**Problem: "Invalid SAML assertion"**

Solutions:
- Verify metadata URL is accessible
- Check clock synchronization between SP and IdP (use NTP)
- Validate certificate and private key match
- Review attribute mappings in IdP

```bash
# Check certificate validity
openssl x509 -in saml-cert.pem -text -noout

# Verify metadata accessibility
curl -I https://your-org.okta.com/app/xxx/sso/saml/metadata
```

**Problem: "Signature verification failed"**

Solutions:
- Ensure IdP metadata includes valid signing certificate
- Check that private key matches public certificate
- Verify IdP is using correct certificate for signing

### OIDC Issues

**Problem: "Token exchange failed"**

Solutions:
- Verify client ID and client secret are correct
- Check redirect URL matches exactly (including protocol and port)
- Ensure issuer URL is correct and accessible
- Review OIDC discovery endpoint: `{issuer}/.well-known/openid-configuration`

```bash
# Test OIDC discovery
curl https://your-tenant.auth0.com/.well-known/openid-configuration
```

**Problem: "Invalid redirect URI"**

Solutions:
- Add callback URL to allowed redirect URIs in IdP
- Ensure URL matches exactly (case-sensitive, including trailing slash)
- Check for URL encoding issues

### Session Issues

**Problem: "Session not found"**

Solutions:
- Check session store is running (Redis if using distributed store)
- Verify session hasn't expired
- Review cleanup interval settings

**Problem: "Token signature invalid"**

Solutions:
- Verify signing key matches across all instances
- Check for key rotation without session invalidation
- Ensure JWT token hasn't been tampered with

### Debugging

Enable debug logging:

```yaml
logging:
  level: debug
  format: json
```

Or via environment variable:

```bash
export SPECULAR_LOG_LEVEL=debug
```

Check logs for detailed error messages:

```bash
# View authentication logs
tail -f /var/log/specular/auth.log | grep -i "auth"

# Filter SAML events
tail -f /var/log/specular/auth.log | grep -i "saml"

# Filter OIDC events
tail -f /var/log/specular/auth.log | grep -i "oidc"
```

## Additional Resources

- [ADR-0012: SSO/SAML Integration Architecture](../adr/0012-sso-saml-integration.md)
- [SAML 2.0 Specification](https://docs.oasis-open.org/security/saml/v2.0/)
- [OAuth 2.0 and OIDC Specifications](https://openid.net/developers/specs/)
- [PKCE RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636)
- [Okta SAML Documentation](https://developer.okta.com/docs/guides/build-sso-integration/saml2/)
- [Auth0 OIDC Documentation](https://auth0.com/docs/authenticate/protocols/openid-connect-protocol)

## Support

For issues or questions:
- GitHub Issues: https://github.com/felixgeelhaar/specular/issues
- Documentation: https://docs.specular.dev
- Community: https://community.specular.dev
