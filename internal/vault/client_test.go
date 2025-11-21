package vault

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: Config{
				Address:   "https://vault.example.com",
				Token:     "test-token",
				MountPath: "secret",
			},
			wantErr: false,
		},
		{
			name: "missing address",
			config: Config{
				Token: "test-token",
			},
			wantErr: true,
			errMsg:  "vault address is required",
		},
		{
			name: "missing token",
			config: Config{
				Address: "https://vault.example.com",
			},
			wantErr: true,
			errMsg:  "vault token is required",
		},
		{
			name: "default mount path",
			config: Config{
				Address: "https://vault.example.com",
				Token:   "test-token",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.config.Address, client.Address())

				// Check default mount path
				if tt.config.MountPath == "" {
					assert.Equal(t, "secret", client.MountPath())
				} else {
					assert.Equal(t, tt.config.MountPath, client.MountPath())
				}

				// Cleanup
				client.Close()
			}
		})
	}
}

func TestNewClient_TokenFromEnv(t *testing.T) {
	// Set environment variable
	os.Setenv("VAULT_TOKEN", "env-token")
	defer os.Unsetenv("VAULT_TOKEN")

	config := Config{
		Address: "https://vault.example.com",
	}

	client, err := NewClient(config)
	require.NoError(t, err)
	assert.NotNil(t, client)

	client.Close()
}

func TestClient_Health(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "healthy vault",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "standby vault",
			statusCode: http.StatusTooManyRequests,
			wantErr:    false,
		},
		{
			name:       "unhealthy vault",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/sys/health", r.URL.Path)
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client, err := NewClient(Config{
				Address: server.URL,
				Token:   "test-token",
			})
			require.NoError(t, err)
			defer client.Close()

			err = client.Health(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_TokenRenewal(t *testing.T) {
	renewalCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/auth/token/renew-self" {
			renewalCalled = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Address:  server.URL,
		Token:    "test-token",
		TokenTTL: 100 * time.Millisecond, // Short TTL for testing
	})
	require.NoError(t, err)
	defer client.Close()

	// Wait for renewal (80% of 100ms = 80ms)
	time.Sleep(150 * time.Millisecond)

	assert.True(t, renewalCalled, "Token renewal should have been called")
}

func TestClient_Close(t *testing.T) {
	client, err := NewClient(Config{
		Address: "https://vault.example.com",
		Token:   "test-token",
	})
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

func TestClient_Namespace(t *testing.T) {
	client, err := NewClient(Config{
		Address:   "https://vault.example.com",
		Token:     "test-token",
		Namespace: "my-namespace",
	})
	require.NoError(t, err)
	defer client.Close()

	assert.Equal(t, "my-namespace", client.Namespace())
}

func TestClient_TLSConfig(t *testing.T) {
	// This test verifies that TLS configuration is created correctly
	// We can't easily test the actual TLS handshake without a real server

	t.Run("valid TLS config with CA cert path", func(t *testing.T) {
		httpClient, err := createHTTPClient(&TLSConfig{
			CAPath: "/tmp", // Use a path that exists
		})
		assert.NoError(t, err)
		assert.NotNil(t, httpClient)
	})

	t.Run("valid TLS config with insecure skip verify", func(t *testing.T) {
		httpClient, err := createHTTPClient(&TLSConfig{
			InsecureSkipVerify: true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, httpClient)
		assert.True(t, httpClient.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("valid TLS config with server name", func(t *testing.T) {
		httpClient, err := createHTTPClient(&TLSConfig{
			TLSServerName: "vault.example.com",
		})
		assert.NoError(t, err)
		assert.NotNil(t, httpClient)
		assert.Equal(t, "vault.example.com", httpClient.Transport.(*http.Transport).TLSClientConfig.ServerName)
	})
}

func TestClient_AddHeaders(t *testing.T) {
	client, err := NewClient(Config{
		Address: "https://vault.example.com",
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	// Verify client was created successfully
	assert.NotNil(t, client)
	assert.Equal(t, "test-token", client.token)
}
