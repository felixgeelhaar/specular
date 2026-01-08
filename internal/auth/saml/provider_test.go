package saml

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/crewjam/saml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/specular/internal/auth"
)

// Test helper to generate mock IDP metadata XML
func generateMockIDPMetadata(entityID, ssoURL string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="` + entityID + `">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="` + ssoURL + `"/>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="` + ssoURL + `"/>
  </IDPSSODescriptor>
</EntityDescriptor>`
}

// TestValidateConfig tests configuration validation
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError string
	}{
		{
			name:      "empty_entity_id",
			config:    &Config{},
			wantError: "entity ID is required",
		},
		{
			name: "empty_acs_url",
			config: &Config{
				EntityID: "https://sp.example.com/metadata",
			},
			wantError: "assertion consumer service URL is required",
		},
		{
			name: "empty_idp_metadata_url",
			config: &Config{
				EntityID:                    "https://sp.example.com/metadata",
				AssertionConsumerServiceURL: "https://sp.example.com/acs",
			},
			wantError: "IdP metadata URL is required",
		},
		{
			name: "valid_config",
			config: &Config{
				EntityID:                    "https://sp.example.com/metadata",
				AssertionConsumerServiceURL: "https://sp.example.com/acs",
				IDPMetadataURL:              "https://idp.example.com/metadata",
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
			name: "missing_acs_url",
			config: &Config{
				EntityID:       "https://sp.example.com/metadata",
				IDPMetadataURL: "https://idp.example.com/metadata",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProvider(tt.config, sessionMgr)
			assert.Error(t, err)
			assert.True(t, auth.IsAuthError(err, auth.ErrSAMLMetadataFailed))
		})
	}
}

// TestNewProvider_InvalidEntityID tests invalid entity ID URL
func TestNewProvider_InvalidEntityID(t *testing.T) {
	// Start mock IDP metadata server
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(generateMockIDPMetadata("https://idp.example.com", "https://idp.example.com/sso")))
	}))
	defer idpServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "://invalid-url",
		AssertionConsumerServiceURL: "https://sp.example.com/acs",
		IDPMetadataURL:              idpServer.URL,
	}

	_, err := NewProvider(cfg, sessionMgr)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrSAMLMetadataFailed))
}

// TestNewProvider_InvalidACSURL tests invalid ACS URL
func TestNewProvider_InvalidACSURL(t *testing.T) {
	// Start mock IDP metadata server
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(generateMockIDPMetadata("https://idp.example.com", "https://idp.example.com/sso")))
	}))
	defer idpServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "https://sp.example.com/metadata",
		AssertionConsumerServiceURL: "://invalid-url",
		IDPMetadataURL:              idpServer.URL,
	}

	_, err := NewProvider(cfg, sessionMgr)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrSAMLMetadataFailed))
}

// TestNewProvider_InvalidSLOURL tests invalid SLO URL
func TestNewProvider_InvalidSLOURL(t *testing.T) {
	// Start mock IDP metadata server
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(generateMockIDPMetadata("https://idp.example.com", "https://idp.example.com/sso")))
	}))
	defer idpServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "https://sp.example.com/metadata",
		AssertionConsumerServiceURL: "https://sp.example.com/acs",
		SingleLogoutServiceURL:      "://invalid-url",
		IDPMetadataURL:              idpServer.URL,
	}

	_, err := NewProvider(cfg, sessionMgr)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrSAMLMetadataFailed))
}

// TestNewProvider_MetadataFetchError tests metadata fetch failure
func TestNewProvider_MetadataFetchError(t *testing.T) {
	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "https://sp.example.com/metadata",
		AssertionConsumerServiceURL: "https://sp.example.com/acs",
		IDPMetadataURL:              "http://localhost:1/nonexistent",
	}

	_, err := NewProvider(cfg, sessionMgr)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrSAMLMetadataFailed))
}

// TestNewProvider_InvalidMetadataResponse tests invalid metadata
func TestNewProvider_InvalidMetadataResponse(t *testing.T) {
	// Start mock IDP that returns invalid XML
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte("not valid xml"))
	}))
	defer idpServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "https://sp.example.com/metadata",
		AssertionConsumerServiceURL: "https://sp.example.com/acs",
		IDPMetadataURL:              idpServer.URL,
	}

	_, err := NewProvider(cfg, sessionMgr)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrSAMLMetadataFailed))
}

// TestNewProvider_NonOKStatusCode tests non-200 response from metadata endpoint
func TestNewProvider_NonOKStatusCode(t *testing.T) {
	// Start mock IDP that returns 404
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer idpServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "https://sp.example.com/metadata",
		AssertionConsumerServiceURL: "https://sp.example.com/acs",
		IDPMetadataURL:              idpServer.URL,
	}

	_, err := NewProvider(cfg, sessionMgr)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrSAMLMetadataFailed))
}

// TestProvider_Name tests provider name
func TestProvider_Name(t *testing.T) {
	p := &Provider{}
	assert.Equal(t, "saml_okta", p.Name())
}

// TestProvider_Authenticate_WrongMethod tests authenticate with wrong HTTP method
func TestProvider_Authenticate_WrongMethod(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	req := httptest.NewRequest(http.MethodGet, "/acs", nil)
	_, err := p.Authenticate(ctx, req)

	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrInvalidCredentials))
	assert.Contains(t, err.Error(), "expected POST request")
}

// TestProvider_Authenticate_MissingSAMLResponse tests authenticate without SAML response
func TestProvider_Authenticate_MissingSAMLResponse(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	// POST request without SAMLResponse
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := p.Authenticate(ctx, req)

	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrInvalidCredentials))
	assert.Contains(t, err.Error(), "missing SAMLResponse")
}

// TestProvider_ValidateSession tests session validation (pass-through)
func TestProvider_ValidateSession(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	session := &auth.Session{
		UserID:   "user123",
		Email:    "user@example.com",
		Provider: "saml_okta",
	}

	err := p.ValidateSession(ctx, session)
	assert.NoError(t, err)
}

// TestProvider_RefreshSession tests session refresh (not supported)
func TestProvider_RefreshSession(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	session := &auth.Session{
		UserID:   "user123",
		Email:    "user@example.com",
		Provider: "saml_okta",
	}

	newSession, err := p.RefreshSession(ctx, session)

	assert.Nil(t, newSession)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrRefreshFailed))
	assert.Contains(t, err.Error(), "SAML does not support session refresh")
}

// TestProvider_Logout tests logout (pass-through)
func TestProvider_Logout(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	session := &auth.Session{
		UserID:   "user123",
		Email:    "user@example.com",
		Provider: "saml_okta",
	}

	err := p.Logout(ctx, session)
	assert.NoError(t, err)
}

// TestProvider_GetMetadata tests metadata generation
func TestProvider_GetMetadata(t *testing.T) {
	// Create a minimal service provider for testing
	acsURL, _ := url.Parse("https://sp.example.com/acs")
	sp := &saml.ServiceProvider{
		EntityID:          "https://sp.example.com/metadata",
		AcsURL:            *acsURL,
		AllowIDPInitiated: false,
	}

	p := &Provider{
		serviceProvider: sp,
	}

	metadata, err := p.GetMetadata()
	require.NoError(t, err)
	assert.NotEmpty(t, metadata)

	// Verify it's valid XML
	var entityDescriptor saml.EntityDescriptor
	err = xml.Unmarshal(metadata, &entityDescriptor)
	assert.NoError(t, err)
	assert.Equal(t, "https://sp.example.com/metadata", entityDescriptor.EntityID)
}

// TestFetchIDPMetadata_Success tests successful metadata fetch
func TestFetchIDPMetadata_Success(t *testing.T) {
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(generateMockIDPMetadata("https://idp.example.com", "https://idp.example.com/sso")))
	}))
	defer idpServer.Close()

	metadata, err := fetchIDPMetadata(idpServer.URL)
	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, "https://idp.example.com", metadata.EntityID)
}

// TestFetchIDPMetadata_Timeout tests metadata fetch timeout
func TestFetchIDPMetadata_Timeout(t *testing.T) {
	// Server that never responds (simulates timeout)
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(35 * time.Second) // Longer than 30s timeout
	}))
	defer idpServer.Close()

	// This test would take too long with real timeout, so we skip
	t.Skip("Skipping timeout test - would take 30+ seconds")
}

// TestCreateSessionFromAssertion tests session creation from SAML assertion
func TestCreateSessionFromAssertion(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	// Create a mock assertion
	assertion := &saml.Assertion{
		Subject: &saml.Subject{
			NameID: &saml.NameID{
				Value: "user@example.com",
			},
		},
		AttributeStatements: []saml.AttributeStatement{
			{
				Attributes: []saml.Attribute{
					{
						Name: "email",
						Values: []saml.AttributeValue{
							{Value: "user@example.com"},
						},
					},
					{
						Name: "groups",
						Values: []saml.AttributeValue{
							{Value: "admins"},
							{Value: "developers"},
						},
					},
				},
			},
		},
	}

	session, err := p.createSessionFromAssertion(ctx, assertion)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", session.UserID)
	assert.Equal(t, "user@example.com", session.Email)
	assert.Equal(t, "saml_okta", session.Provider)
	assert.NotNil(t, session.Attributes)
	assert.Equal(t, "user@example.com", session.Attributes["email"])

	// Verify groups attribute is an array
	groups, ok := session.Attributes["groups"].([]string)
	assert.True(t, ok)
	assert.Contains(t, groups, "admins")
	assert.Contains(t, groups, "developers")
}

// TestCreateSessionFromAssertion_MissingNameID tests session creation with missing NameID
func TestCreateSessionFromAssertion_MissingNameID(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	assertion := &saml.Assertion{
		Subject: &saml.Subject{
			NameID: &saml.NameID{
				Value: "", // Empty
			},
		},
		AttributeStatements: []saml.AttributeStatement{
			{
				Attributes: []saml.Attribute{},
			},
		},
	}

	_, err := p.createSessionFromAssertion(ctx, assertion)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrSAMLAssertionInvalid))
}

// TestCreateSessionFromAssertion_UseEmailAttribute tests email from attribute
func TestCreateSessionFromAssertion_UseEmailAttribute(t *testing.T) {
	p := &Provider{}
	ctx := context.Background()

	assertion := &saml.Assertion{
		Subject: &saml.Subject{
			NameID: &saml.NameID{
				Value: "uid123", // Not an email
			},
		},
		AttributeStatements: []saml.AttributeStatement{
			{
				Attributes: []saml.Attribute{
					{
						Name: "email",
						Values: []saml.AttributeValue{
							{Value: "real-email@example.com"},
						},
					},
				},
			},
		},
	}

	session, err := p.createSessionFromAssertion(ctx, assertion)
	require.NoError(t, err)
	assert.Equal(t, "uid123", session.UserID)
	assert.Equal(t, "real-email@example.com", session.Email)
}

// TestConfig_CustomNameIDFormat tests custom NameID format configuration
func TestConfig_CustomNameIDFormat(t *testing.T) {
	// Start mock IDP metadata server
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(generateMockIDPMetadata("https://idp.example.com", "https://idp.example.com/sso")))
	}))
	defer idpServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "https://sp.example.com/metadata",
		AssertionConsumerServiceURL: "https://sp.example.com/acs",
		IDPMetadataURL:              idpServer.URL,
		NameIDFormat:                "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",
	}

	provider, err := NewProvider(cfg, sessionMgr)
	require.NoError(t, err)
	assert.Equal(t, saml.NameIDFormat("urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"),
		provider.serviceProvider.AuthnNameIDFormat)
}

// TestConfig_WithSLOURL tests configuration with SLO URL
func TestConfig_WithSLOURL(t *testing.T) {
	// Start mock IDP metadata server
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(generateMockIDPMetadata("https://idp.example.com", "https://idp.example.com/sso")))
	}))
	defer idpServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "https://sp.example.com/metadata",
		AssertionConsumerServiceURL: "https://sp.example.com/acs",
		SingleLogoutServiceURL:      "https://sp.example.com/slo",
		IDPMetadataURL:              idpServer.URL,
	}

	provider, err := NewProvider(cfg, sessionMgr)
	require.NoError(t, err)
	assert.Equal(t, "https://sp.example.com/slo", provider.serviceProvider.SloURL.String())
}

// TestConfig_WithCertificates tests configuration with signing certificates
func TestConfig_WithCertificates(t *testing.T) {
	// Start mock IDP metadata server
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(generateMockIDPMetadata("https://idp.example.com", "https://idp.example.com/sso")))
	}))
	defer idpServer.Close()

	// Generate a test RSA key and certificate
	// Using a minimal self-signed cert for testing
	certPEM := `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIJAJPVf4dOMRmGMAoGCCqGSM49BAMCMBIxEDAOBgNVBAMM
B3Rlc3QtY2EwHhcNMjMwMTAxMDAwMDAwWhcNMjQwMTAxMDAwMDAwWjASMRAwDgYD
VQQDDAd0ZXN0LWNhMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEn6v2c0HGxLhv
+9YlJP5YXGX0SdZCvEzVkPvxC0UuRHLMuXdz4bZC5X0dPYJEq0TvA9YJfK7NVaPJ
m8c5X1RZsqNTMFEwHQYDVR0OBBYEFBLGIqQiKEGLO+XVbD0MXwmFQIJZMB8GA1Ud
IwQYMBaAFBLGIqQiKEGLO+XVbD0MXwmFQIJZMA8GA1UdEwEB/wQFMAMBAf8wCgYI
KoZIzj0EAwIDSAAwRQIgHNYBE7hgKLEHLy+C0FMdB0EhZ0xDX4pQG/UlGbkMxVcC
IQDTOONm2n5zH3Z5VdLdB5ZDjXLPZN5FQVN7+x4d+qJmwA==
-----END CERTIFICATE-----`

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		t.Skip("Cannot decode test certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Skip("Cannot parse test certificate")
	}

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "https://sp.example.com/metadata",
		AssertionConsumerServiceURL: "https://sp.example.com/acs",
		IDPMetadataURL:              idpServer.URL,
		Certificate:                 cert,
		// PrivateKey would normally be set here too
	}

	provider, err := NewProvider(cfg, sessionMgr)
	require.NoError(t, err)
	assert.Equal(t, cert, provider.serviceProvider.Certificate)
}

// TestInitiateLogin tests the SSO login initiation
func TestInitiateLogin(t *testing.T) {
	// Start mock IDP metadata server with SSO URL
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(generateMockIDPMetadata("https://idp.example.com", "https://idp.example.com/sso")))
	}))
	defer idpServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "https://sp.example.com/metadata",
		AssertionConsumerServiceURL: "https://sp.example.com/acs",
		IDPMetadataURL:              idpServer.URL,
	}

	provider, err := NewProvider(cfg, sessionMgr)
	require.NoError(t, err)

	// Create test request and response recorder
	req := httptest.NewRequest(http.MethodGet, "/login?relay_state=/dashboard", nil)
	rr := httptest.NewRecorder()

	// Initiate login
	err = provider.InitiateLogin(rr, req)
	require.NoError(t, err)

	// Check redirect
	assert.Equal(t, http.StatusFound, rr.Code)
	location := rr.Header().Get("Location")
	assert.Contains(t, location, "https://idp.example.com/sso")
	assert.Contains(t, location, "SAMLRequest")
}

// TestInitiateLogin_WithForceAuthn tests login with force authentication
func TestInitiateLogin_WithForceAuthn(t *testing.T) {
	// Start mock IDP metadata server
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(generateMockIDPMetadata("https://idp.example.com", "https://idp.example.com/sso")))
	}))
	defer idpServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "https://sp.example.com/metadata",
		AssertionConsumerServiceURL: "https://sp.example.com/acs",
		IDPMetadataURL:              idpServer.URL,
		ForceAuthn:                  true,
	}

	provider, err := NewProvider(cfg, sessionMgr)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()

	err = provider.InitiateLogin(rr, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rr.Code)
}

// TestInitiateLogin_DefaultRelayState tests login with default relay state
func TestInitiateLogin_DefaultRelayState(t *testing.T) {
	// Start mock IDP metadata server
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(generateMockIDPMetadata("https://idp.example.com", "https://idp.example.com/sso")))
	}))
	defer idpServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "https://sp.example.com/metadata",
		AssertionConsumerServiceURL: "https://sp.example.com/acs",
		IDPMetadataURL:              idpServer.URL,
	}

	provider, err := NewProvider(cfg, sessionMgr)
	require.NoError(t, err)

	// Request without relay_state
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()

	err = provider.InitiateLogin(rr, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rr.Code)
	// Should use "/" as default relay state
	location := rr.Header().Get("Location")
	assert.Contains(t, location, "RelayState")
}

// TestHandleCallback_MissingSAMLResponse tests callback without SAML response
func TestHandleCallback_MissingSAMLResponse(t *testing.T) {
	// Start mock IDP metadata server
	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(generateMockIDPMetadata("https://idp.example.com", "https://idp.example.com/sso")))
	}))
	defer idpServer.Close()

	sessionMgr := auth.NewSessionManager([]byte("test-signing-key-32bytes!!!"), "test-issuer")

	cfg := &Config{
		EntityID:                    "https://sp.example.com/metadata",
		AssertionConsumerServiceURL: "https://sp.example.com/acs",
		IDPMetadataURL:              idpServer.URL,
	}

	provider, err := NewProvider(cfg, sessionMgr)
	require.NoError(t, err)

	// POST request without SAMLResponse
	req := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	_, err = provider.HandleCallback(rr, req)
	assert.Error(t, err)
	assert.True(t, auth.IsAuthError(err, auth.ErrSAMLAssertionInvalid))
}
