package vault

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Client wraps HashiCorp Vault API client with opinionated configuration.
//
// This client provides:
// - KV v2 secrets engine support
// - ECDSA key storage and retrieval
// - Automatic token renewal
// - TLS/mTLS configuration
// - Error wrapping with context
type Client struct {
	address string
	token   string

	// KV engine configuration
	mountPath string
	namespace string

	// HTTP client with TLS configuration
	httpClient *http.Client

	// Token renewal
	tokenTTL      time.Duration
	renewalTicker *time.Ticker
	stopRenewal   chan struct{}
}

// Config holds Vault client configuration.
type Config struct {
	// Address is the Vault server address (required)
	// Example: "https://vault.example.com:8200"
	Address string

	// Token is the Vault authentication token (required)
	// Can also be set via VAULT_TOKEN environment variable
	Token string

	// MountPath is the KV v2 mount path (default: "secret")
	MountPath string

	// Namespace is the Vault namespace (optional, Enterprise feature)
	Namespace string

	// TLSConfig for mTLS authentication (optional)
	TLSConfig *TLSConfig

	// TokenTTL is the token time-to-live for renewal (default: 24h)
	TokenTTL time.Duration
}

// TLSConfig holds TLS/mTLS configuration.
type TLSConfig struct {
	// CACert is the path to the CA certificate
	CACert string

	// CAPath is the path to a directory of CA certificates
	CAPath string

	// ClientCert is the path to the client certificate (for mTLS)
	ClientCert string

	// ClientKey is the path to the client private key (for mTLS)
	ClientKey string

	// TLSServerName is the server name to use for SNI
	TLSServerName string

	// InsecureSkipVerify disables certificate verification (NOT for production)
	InsecureSkipVerify bool
}

// NewClient creates a new Vault client with the provided configuration.
func NewClient(cfg Config) (*Client, error) {
	// Validate required fields
	if cfg.Address == "" {
		return nil, fmt.Errorf("vault address is required")
	}

	// Token can come from config or environment
	token := cfg.Token
	if token == "" {
		token = os.Getenv("VAULT_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("vault token is required (set via config or VAULT_TOKEN env var)")
	}

	// Set defaults
	if cfg.MountPath == "" {
		cfg.MountPath = "secret"
	}
	if cfg.TokenTTL == 0 {
		cfg.TokenTTL = 24 * time.Hour
	}

	// Create HTTP client with TLS configuration
	httpClient, err := createHTTPClient(cfg.TLSConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	client := &Client{
		address:       cfg.Address,
		token:         token,
		mountPath:     cfg.MountPath,
		namespace:     cfg.Namespace,
		httpClient:    httpClient,
		tokenTTL:      cfg.TokenTTL,
		stopRenewal:   make(chan struct{}),
	}

	// Start automatic token renewal
	client.startTokenRenewal()

	return client, nil
}

// createHTTPClient creates an HTTP client with TLS configuration.
func createHTTPClient(tlsCfg *TLSConfig) (*http.Client, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	if tlsCfg != nil {
		// Load CA certificate(s)
		if tlsCfg.CACert != "" || tlsCfg.CAPath != "" {
			caCertPool := x509.NewCertPool()

			if tlsCfg.CACert != "" {
				caCert, err := os.ReadFile(tlsCfg.CACert)
				if err != nil {
					return nil, fmt.Errorf("failed to read CA cert: %w", err)
				}
				if !caCertPool.AppendCertsFromPEM(caCert) {
					return nil, fmt.Errorf("failed to parse CA cert")
				}
			}

			if tlsCfg.CAPath != "" {
				// Load all certs from directory
				entries, err := os.ReadDir(tlsCfg.CAPath)
				if err != nil {
					return nil, fmt.Errorf("failed to read CA path: %w", err)
				}

				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}

					certPath := fmt.Sprintf("%s/%s", tlsCfg.CAPath, entry.Name())
					cert, err := os.ReadFile(certPath)
					if err != nil {
						continue // Skip files we can't read
					}
					caCertPool.AppendCertsFromPEM(cert)
				}
			}

			transport.TLSClientConfig.RootCAs = caCertPool
		}

		// Load client certificate for mTLS
		if tlsCfg.ClientCert != "" && tlsCfg.ClientKey != "" {
			clientCert, err := tls.LoadX509KeyPair(tlsCfg.ClientCert, tlsCfg.ClientKey)
			if err != nil {
				return nil, fmt.Errorf("failed to load client cert/key: %w", err)
			}
			transport.TLSClientConfig.Certificates = []tls.Certificate{clientCert}
		}

		// Set SNI server name
		if tlsCfg.TLSServerName != "" {
			transport.TLSClientConfig.ServerName = tlsCfg.TLSServerName
		}

		// Allow insecure skip verify (NOT for production)
		if tlsCfg.InsecureSkipVerify {
			transport.TLSClientConfig.InsecureSkipVerify = true
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}, nil
}

// startTokenRenewal starts automatic token renewal based on TTL.
func (c *Client) startTokenRenewal() {
	// Renew token at 80% of TTL
	renewInterval := time.Duration(float64(c.tokenTTL) * 0.8)
	c.renewalTicker = time.NewTicker(renewInterval)

	go func() {
		for {
			select {
			case <-c.renewalTicker.C:
				// Renew token
				if err := c.renewToken(context.Background()); err != nil {
					// Log error but don't stop renewal attempts
					fmt.Fprintf(os.Stderr, "vault: failed to renew token: %v\n", err)
				}
			case <-c.stopRenewal:
				c.renewalTicker.Stop()
				return
			}
		}
	}()
}

// renewToken renews the Vault token.
func (c *Client) renewToken(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "POST", c.address+"/v1/auth/token/renew-self", nil)
	if err != nil {
		return fmt.Errorf("failed to create renewal request: %w", err)
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to renew token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token renewal failed with status %d", resp.StatusCode)
	}

	return nil
}

// addHeaders adds required headers to Vault API requests.
func (c *Client) addHeaders(req *http.Request) {
	req.Header.Set("X-Vault-Token", c.token)
	req.Header.Set("Content-Type", "application/json")

	if c.namespace != "" {
		req.Header.Set("X-Vault-Namespace", c.namespace)
	}
}

// Close closes the Vault client and stops token renewal.
func (c *Client) Close() error {
	close(c.stopRenewal)
	return nil
}

// Health checks Vault server health and returns status.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.address+"/v1/sys/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	// Vault health endpoint returns 200 for healthy, 429 for standby, 5xx for errors
	if resp.StatusCode >= 500 {
		return fmt.Errorf("vault unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// MountPath returns the configured KV mount path.
func (c *Client) MountPath() string {
	return c.mountPath
}

// Address returns the Vault server address.
func (c *Client) Address() string {
	return c.address
}

// Namespace returns the configured Vault namespace.
func (c *Client) Namespace() string {
	return c.namespace
}
