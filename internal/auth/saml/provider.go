// Package saml provides SAML 2.0 service provider implementation for enterprise SSO.
//
// Supports SAML authentication with identity providers like Okta.
// Implements the Authenticator interface from the auth package.
//
// This package supports M9.2.1 (SSO/SAML Integration) in the v2.0 roadmap.
package saml

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"

	"github.com/felixgeelhaar/specular/internal/auth"
)

// Config holds SAML service provider configuration.
type Config struct {
	// EntityID is the unique identifier for this service provider.
	// Example: "https://specular.example.com/saml/metadata"
	EntityID string

	// AssertionConsumerServiceURL is where SAML responses are sent.
	// Example: "https://specular.example.com/saml/acs"
	AssertionConsumerServiceURL string

	// SingleLogoutServiceURL for logout requests (optional).
	// Example: "https://specular.example.com/saml/slo"
	SingleLogoutServiceURL string

	// IDPMetadataURL is the identity provider metadata endpoint.
	// For Okta: "https://dev-12345.okta.com/app/abc123/sso/saml/metadata"
	IDPMetadataURL string

	// Certificate for signing SAML requests (optional for some IdPs).
	Certificate *x509.Certificate

	// PrivateKey for signing SAML requests (optional for some IdPs).
	PrivateKey *rsa.PrivateKey

	// AllowIDPInitiated enables IdP-initiated SSO (default: false for security).
	AllowIDPInitiated bool

	// SignRequest enables request signing (required by some IdPs).
	SignRequest bool

	// ForceAuthn forces re-authentication even if user has existing session.
	ForceAuthn bool

	// NameIDFormat specifies the name identifier format.
	// Default: urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress
	NameIDFormat string
}

// Provider implements SAML 2.0 service provider.
//
// Handles SAML authentication flows:
//  1. SP-initiated: User accesses protected resource, redirected to IdP
//  2. IdP-initiated: User starts at IdP, assertion sent to SP
//
// Thread-safe: Safe for concurrent use.
type Provider struct {
	config          *Config
	serviceProvider *saml.ServiceProvider
	sessionManager  *auth.SessionManager
}

// NewProvider creates a SAML authentication provider.
//
// Fetches and parses IdP metadata from the configured URL.
// Returns an error if metadata cannot be retrieved or is invalid.
func NewProvider(cfg *Config, sessionMgr *auth.SessionManager) (*Provider, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, auth.WrapError(auth.ErrSAMLMetadataFailed, "invalid SAML configuration", err, nil)
	}

	// Parse entity ID URL
	entityIDURL, err := url.Parse(cfg.EntityID)
	if err != nil {
		return nil, auth.WrapError(auth.ErrSAMLMetadataFailed, "invalid entity ID", err, map[string]interface{}{
			"entity_id": cfg.EntityID,
		})
	}

	// Parse ACS URL
	acsURL, err := url.Parse(cfg.AssertionConsumerServiceURL)
	if err != nil {
		return nil, auth.WrapError(auth.ErrSAMLMetadataFailed, "invalid ACS URL", err, map[string]interface{}{
			"acs_url": cfg.AssertionConsumerServiceURL,
		})
	}

	// Create service provider
	sp := &saml.ServiceProvider{
		EntityID:          entityIDURL.String(),
		AcsURL:            *acsURL,
		Certificate:       cfg.Certificate,
		Key:               cfg.PrivateKey,
		AllowIDPInitiated: cfg.AllowIDPInitiated,
		AuthnNameIDFormat: saml.EmailAddressNameIDFormat,
	}

	// Set custom NameID format if specified
	if cfg.NameIDFormat != "" {
		sp.AuthnNameIDFormat = saml.NameIDFormat(cfg.NameIDFormat)
	}

	// Parse SLO URL if provided
	if cfg.SingleLogoutServiceURL != "" {
		sloURL, sloErr := url.Parse(cfg.SingleLogoutServiceURL)
		if sloErr != nil {
			return nil, auth.WrapError(auth.ErrSAMLMetadataFailed, "invalid SLO URL", sloErr, map[string]interface{}{
				"slo_url": cfg.SingleLogoutServiceURL,
			})
		}
		sp.SloURL = *sloURL
	}

	// Fetch IdP metadata
	idpMetadata, err := fetchIDPMetadata(cfg.IDPMetadataURL)
	if err != nil {
		return nil, auth.WrapError(auth.ErrSAMLMetadataFailed, "failed to fetch IdP metadata", err, map[string]interface{}{
			"metadata_url": cfg.IDPMetadataURL,
		})
	}

	sp.IDPMetadata = idpMetadata

	return &Provider{
		config:          cfg,
		serviceProvider: sp,
		sessionManager:  sessionMgr,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "saml_okta"
}

// Authenticate validates SAML assertion from HTTP request and returns a session.
//
// Expected request format:
//   - POST to ACS URL with SAMLResponse parameter (SAML assertion)
//   - Assertion must be signed by trusted IdP
//   - Assertion must not be expired
//
// Returns ErrInvalidCredentials if authentication fails.
func (p *Provider) Authenticate(ctx context.Context, req *http.Request) (*auth.Session, error) {
	// Check if this is a SAML response
	if req.Method != http.MethodPost {
		return nil, auth.NewError(auth.ErrInvalidCredentials, "expected POST request", map[string]interface{}{
			"method": req.Method,
		})
	}

	// Parse SAML response
	if err := req.ParseForm(); err != nil {
		return nil, auth.WrapError(auth.ErrInvalidCredentials, "failed to parse form", err, nil)
	}

	samlResponse := req.PostForm.Get("SAMLResponse")
	if samlResponse == "" {
		return nil, auth.NewError(auth.ErrInvalidCredentials, "missing SAMLResponse parameter", nil)
	}

	// Parse and validate assertion
	assertion, err := p.serviceProvider.ParseResponse(req, []string{})
	if err != nil {
		return nil, auth.WrapError(auth.ErrSAMLAssertionInvalid, "failed to parse SAML response", err, nil)
	}

	// Extract user information from assertion
	session, err := p.createSessionFromAssertion(ctx, assertion)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// ValidateSession validates a SAML session.
//
// For SAML, we primarily rely on JWT token validation.
// This method is called after JWT validation to perform any SAML-specific checks.
func (p *Provider) ValidateSession(ctx context.Context, session *auth.Session) error {
	// SAML sessions are validated via JWT
	// No additional IdP checks needed (stateless)
	return nil
}

// RefreshSession attempts to refresh a SAML session.
//
// SAML doesn't support token refresh - user must re-authenticate.
// Returns ErrRefreshFailed to indicate refresh is not supported.
func (p *Provider) RefreshSession(ctx context.Context, session *auth.Session) (*auth.Session, error) {
	return nil, auth.NewError(auth.ErrRefreshFailed, "SAML does not support session refresh", map[string]interface{}{
		"provider": "saml_okta",
	})
}

// Logout terminates a SAML session.
//
// Performs Single Logout (SLO) if supported by the IdP.
// Returns nil if logout succeeds or SLO is not configured.
func (p *Provider) Logout(ctx context.Context, session *auth.Session) error {
	// SAML logout is handled via SLO endpoints
	// Session is removed from session store by Manager
	return nil
}

// InitiateLogin starts the SAML authentication flow.
//
// Redirects the user to the IdP SSO URL with a signed AuthnRequest.
// The IdP will authenticate the user and POST the assertion to the ACS URL.
func (p *Provider) InitiateLogin(w http.ResponseWriter, r *http.Request) error {
	// Generate relay state (return URL after authentication)
	relayState := r.URL.Query().Get("relay_state")
	if relayState == "" {
		relayState = "/"
	}

	// Create AuthnRequest
	binding := saml.HTTPRedirectBinding
	bindingLocation := p.serviceProvider.GetSSOBindingLocation(binding)

	authnRequest, err := p.serviceProvider.MakeAuthenticationRequest(bindingLocation, binding, saml.HTTPPostBinding)
	if err != nil {
		return auth.WrapError(auth.ErrSAMLMetadataFailed, "failed to create AuthnRequest", err, nil)
	}

	// Set ForceAuthn if configured
	if p.config.ForceAuthn {
		authnRequest.ForceAuthn = &p.config.ForceAuthn
	}

	// Create redirect URL using the saml library's redirect mechanism
	redirectURL, err := authnRequest.Redirect(relayState, p.serviceProvider)
	if err != nil {
		return auth.WrapError(auth.ErrSAMLMetadataFailed, "failed to create redirect", err, nil)
	}

	// Redirect to IdP
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
	return nil
}

// HandleCallback processes SAML assertions from the IdP.
//
// Validates the assertion signature and extracts user information.
// Creates a JWT session and returns it for storage.
func (p *Provider) HandleCallback(w http.ResponseWriter, r *http.Request) (*auth.Session, error) {
	// Parse and validate SAML response
	assertion, err := p.serviceProvider.ParseResponse(r, []string{})
	if err != nil {
		return nil, auth.WrapError(auth.ErrSAMLAssertionInvalid, "failed to parse SAML response", err, nil)
	}

	// Create session from assertion
	session, err := p.createSessionFromAssertion(r.Context(), assertion)
	if err != nil {
		return nil, err
	}

	// Create JWT token for session
	tokenString, err := p.sessionManager.CreateSession(r.Context(), session)
	if err != nil {
		return nil, err
	}

	session.Token = tokenString
	return session, nil
}

// createSessionFromAssertion extracts user information from SAML assertion.
func (p *Provider) createSessionFromAssertion(ctx context.Context, assertion *saml.Assertion) (*auth.Session, error) {
	// Extract NameID (typically email)
	nameID := assertion.Subject.NameID.Value
	if nameID == "" {
		return nil, auth.NewError(auth.ErrSAMLAssertionInvalid, "missing NameID in assertion", nil)
	}

	// Extract attributes
	attributes := make(map[string]interface{})
	for _, attr := range assertion.AttributeStatements[0].Attributes {
		if len(attr.Values) > 0 {
			if len(attr.Values) == 1 {
				attributes[attr.Name] = attr.Values[0].Value
			} else {
				values := make([]string, len(attr.Values))
				for i, v := range attr.Values {
					values[i] = v.Value
				}
				attributes[attr.Name] = values
			}
		}
	}

	// Extract email (use NameID as fallback)
	email := nameID
	if emailAttr, ok := attributes["email"].(string); ok && emailAttr != "" {
		email = emailAttr
	}

	// Create session
	session := &auth.Session{
		UserID:     nameID,
		Email:      email,
		Provider:   p.Name(),
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(1 * time.Hour), // Will be set by SessionManager
		Attributes: attributes,
	}

	return session, nil
}

// GetMetadata returns the service provider metadata XML.
//
// This metadata should be uploaded to the IdP configuration.
func (p *Provider) GetMetadata() ([]byte, error) {
	metadata := p.serviceProvider.Metadata()

	// Marshal metadata to XML
	metadataXML, err := xml.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return metadataXML, nil
}

// validateConfig validates the SAML configuration.
func validateConfig(cfg *Config) error {
	if cfg.EntityID == "" {
		return fmt.Errorf("entity ID is required")
	}
	if cfg.AssertionConsumerServiceURL == "" {
		return fmt.Errorf("assertion consumer service URL is required")
	}
	if cfg.IDPMetadataURL == "" {
		return fmt.Errorf("IdP metadata URL is required")
	}
	return nil
}

// fetchIDPMetadata fetches and parses IdP metadata from URL.
func fetchIDPMetadata(metadataURL string) (*saml.EntityDescriptor, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	// Fetch metadata
	resp, err := client.Get(metadataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	//nolint:errcheck // Deferred close, error not critical
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata request failed with status: %d", resp.StatusCode)
	}

	// Read response body
	metadataXML, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Parse metadata
	metadata, err := samlsp.ParseMetadata(metadataXML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return metadata, nil
}
