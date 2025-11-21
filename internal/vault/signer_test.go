package vault

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockVaultServerForSigner creates a mock Vault server that simulates KV v2 operations for signer testing.
func mockVaultServerForSigner(t *testing.T, storedKey *storedKeyData) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/secret/data/test-key" && r.Method == "GET":
			// Check if key exists
			if storedKey.PrivateKey == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			// Return stored key
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"data": map[string]interface{}{
						"private_key": storedKey.PrivateKey,
						"public_key":  storedKey.PublicKey,
						"algorithm":   storedKey.Algorithm,
						"created_at":  storedKey.CreatedAt,
						"identity":    storedKey.Identity,
					},
					"metadata": map[string]interface{}{
						"version": 1,
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		case r.URL.Path == "/v1/secret/data/test-key" && r.Method == "POST":
			// Store new key
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			json.Unmarshal(body, &payload)

			data := payload["data"].(map[string]interface{})
			storedKey.PrivateKey = data["private_key"].(string)
			storedKey.PublicKey = data["public_key"].(string)
			storedKey.Algorithm = data["algorithm"].(string)
			storedKey.CreatedAt = data["created_at"].(string)
			storedKey.Identity = data["identity"].(string)

			w.WriteHeader(http.StatusOK)

		case r.URL.Path == "/v1/secret/data/nonexistent" && r.Method == "GET":
			// Key not found
			w.WriteHeader(http.StatusNotFound)

		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
}

type storedKeyData struct {
	PrivateKey string
	PublicKey  string
	Algorithm  string
	CreatedAt  string
	Identity   string
}

func TestNewVaultSigner_AutoGenerate(t *testing.T) {
	storedKey := &storedKeyData{}
	server := mockVaultServerForSigner(t, storedKey)
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	signer, err := client.NewSigner(context.Background(), SignerConfig{
		KeyPath:      "test-key",
		Identity:     "test@example.com",
		AutoGenerate: true,
		CacheTTL:     1 * time.Minute,
	})

	require.NoError(t, err)
	assert.NotNil(t, signer)
	assert.Equal(t, "test@example.com", signer.Identity())

	// Verify key was generated
	assert.NotEmpty(t, storedKey.PrivateKey)
	assert.NotEmpty(t, storedKey.PublicKey)
	assert.Equal(t, "ECDSA-P256", storedKey.Algorithm)
	assert.Equal(t, "test@example.com", storedKey.Identity)
}

func TestNewVaultSigner_NoAutoGenerate(t *testing.T) {
	storedKey := &storedKeyData{}
	server := mockVaultServerForSigner(t, storedKey)
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	// Try to create signer without auto-generate for nonexistent key
	signer, err := client.NewSigner(context.Background(), SignerConfig{
		KeyPath:      "nonexistent",
		Identity:     "test@example.com",
		AutoGenerate: false,
	})

	assert.Error(t, err)
	assert.Nil(t, signer)
	assert.Contains(t, err.Error(), "key not found")
}

func TestVaultSigner_Sign(t *testing.T) {
	storedKey := &storedKeyData{}
	server := mockVaultServerForSigner(t, storedKey)
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	signer, err := client.NewSigner(context.Background(), SignerConfig{
		KeyPath:      "test-key",
		Identity:     "test@example.com",
		AutoGenerate: true,
	})
	require.NoError(t, err)

	// Test data to sign
	testData := []byte("Hello, Vault!")

	// Sign the data
	signature, publicKey, err := signer.Sign(testData)
	require.NoError(t, err)
	assert.NotNil(t, signature)
	assert.NotNil(t, publicKey)

	// Verify signature length (64 bytes for ECDSA P-256: r || s)
	assert.Equal(t, 64, len(signature))

	// Verify the signature is valid
	valid, err := signer.VerifySignature(testData, signature, publicKey)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestVaultSigner_SignMultiple(t *testing.T) {
	storedKey := &storedKeyData{}
	server := mockVaultServerForSigner(t, storedKey)
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	signer, err := client.NewSigner(context.Background(), SignerConfig{
		KeyPath:      "test-key",
		Identity:     "test@example.com",
		AutoGenerate: true,
	})
	require.NoError(t, err)

	// Sign multiple different pieces of data
	testData := [][]byte{
		[]byte("First message"),
		[]byte("Second message"),
		[]byte("Third message"),
	}

	signatures := make([][]byte, len(testData))
	publicKeys := make([][]byte, len(testData))

	for i, data := range testData {
		sig, pubKey, err := signer.Sign(data)
		require.NoError(t, err)
		signatures[i] = sig
		publicKeys[i] = pubKey
	}

	// Verify all signatures
	for i, data := range testData {
		valid, err := signer.VerifySignature(data, signatures[i], publicKeys[i])
		require.NoError(t, err)
		assert.True(t, valid, "Signature %d should be valid", i)
	}

	// Verify signatures are different
	assert.NotEqual(t, signatures[0], signatures[1])
	assert.NotEqual(t, signatures[1], signatures[2])

	// Public keys should be the same
	assert.Equal(t, publicKeys[0], publicKeys[1])
	assert.Equal(t, publicKeys[1], publicKeys[2])
}

func TestVaultSigner_InvalidSignature(t *testing.T) {
	storedKey := &storedKeyData{}
	server := mockVaultServerForSigner(t, storedKey)
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	signer, err := client.NewSigner(context.Background(), SignerConfig{
		KeyPath:      "test-key",
		Identity:     "test@example.com",
		AutoGenerate: true,
	})
	require.NoError(t, err)

	testData := []byte("Original data")
	signature, publicKey, err := signer.Sign(testData)
	require.NoError(t, err)

	// Try to verify with modified data
	modifiedData := []byte("Modified data")
	valid, err := signer.VerifySignature(modifiedData, signature, publicKey)
	require.NoError(t, err)
	assert.False(t, valid, "Signature should be invalid for modified data")
}

func TestVaultSigner_Caching(t *testing.T) {
	storedKey := &storedKeyData{}
	server := mockVaultServerForSigner(t, storedKey)
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	signer, err := client.NewSigner(context.Background(), SignerConfig{
		KeyPath:      "test-key",
		Identity:     "test@example.com",
		AutoGenerate: true,
		CacheTTL:     1 * time.Second,
	})
	require.NoError(t, err)

	// First sign should cache the key
	sig1, _, err := signer.Sign([]byte("test1"))
	require.NoError(t, err)

	// Second sign should use cached key
	sig2, _, err := signer.Sign([]byte("test2"))
	require.NoError(t, err)

	// Signatures should be different (different data)
	assert.NotEqual(t, sig1, sig2)

	// Clear cache
	signer.ClearCache()

	// Sign again (should fetch from Vault)
	sig3, _, err := signer.Sign([]byte("test3"))
	require.NoError(t, err)
	assert.NotNil(t, sig3)
}

func TestVaultSigner_RotateKey(t *testing.T) {
	storedKey := &storedKeyData{}
	server := mockVaultServerForSigner(t, storedKey)
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	signer, err := client.NewSigner(context.Background(), SignerConfig{
		KeyPath:      "test-key",
		Identity:     "test@example.com",
		AutoGenerate: true,
	})
	require.NoError(t, err)

	// Get public key before rotation
	pubKey1, err := signer.GetPublicKey(context.Background())
	require.NoError(t, err)

	// Rotate the key
	err = signer.RotateKey(context.Background())
	require.NoError(t, err)

	// Get public key after rotation
	pubKey2, err := signer.GetPublicKey(context.Background())
	require.NoError(t, err)

	// Public keys should be different after rotation
	assert.NotEqual(t, pubKey1, pubKey2)
}

func TestVaultSigner_GetKeyInfo(t *testing.T) {
	storedKey := &storedKeyData{}
	server := mockVaultServerForSigner(t, storedKey)
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	signer, err := client.NewSigner(context.Background(), SignerConfig{
		KeyPath:      "test-key",
		Identity:     "test@example.com",
		AutoGenerate: true,
	})
	require.NoError(t, err)

	info, err := signer.GetKeyInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "ECDSA-P256", info.Algorithm)
	assert.Equal(t, "test@example.com", info.Identity)
	assert.NotEmpty(t, info.CreatedAt)
}

func TestVaultSigner_GenerateKey(t *testing.T) {
	storedKey := &storedKeyData{}
	server := mockVaultServerForSigner(t, storedKey)
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	signer := &VaultSigner{
		client:   client,
		keyPath:  "test-key",
		identity: "test@example.com",
		cacheTTL: 1 * time.Minute,
	}

	// Generate new key
	err = signer.GenerateKey(context.Background())
	require.NoError(t, err)

	// Verify key was stored
	assert.NotEmpty(t, storedKey.PrivateKey)
	assert.NotEmpty(t, storedKey.PublicKey)
	assert.Equal(t, "ECDSA-P256", storedKey.Algorithm)
	assert.Equal(t, "test@example.com", storedKey.Identity)

	// Verify key is cached
	assert.NotNil(t, signer.cachedKey)
	assert.NotNil(t, signer.cachedPubKey)
	assert.False(t, signer.cacheExpiry.IsZero())
}

func TestVaultSigner_VerifySignature_ValidFormat(t *testing.T) {
	// Create a test signer
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	signer := &VaultSigner{
		identity: "test@example.com",
	}

	testData := []byte("Test data for signing")
	hash := sha256.Sum256(testData)

	// Sign with Go's ECDSA
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	require.NoError(t, err)

	// Format signature as r || s (64 bytes)
	signature := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)

	// Verify with VaultSigner
	valid, err := signer.VerifySignature(testData, signature, publicKeyBytes)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestVaultSigner_VerifySignature_InvalidLength(t *testing.T) {
	// Generate a valid public key for this test
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	signer := &VaultSigner{
		identity: "test@example.com",
	}

	testData := []byte("Test data")
	invalidSignature := []byte("short") // Invalid length signature

	valid, err := signer.VerifySignature(testData, invalidSignature, publicKeyBytes)
	assert.Error(t, err)
	assert.False(t, valid)
	assert.Contains(t, err.Error(), "invalid signature length")
}

func TestSignerConfig_Validation(t *testing.T) {
	client, err := NewClient(Config{
		Address: "https://vault.example.com",
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	tests := []struct {
		name    string
		config  SignerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing key path",
			config: SignerConfig{
				Identity: "test@example.com",
			},
			wantErr: true,
			errMsg:  "key path is required",
		},
		{
			name: "missing identity",
			config: SignerConfig{
				KeyPath: "test-key",
			},
			wantErr: true,
			errMsg:  "identity is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer, err := client.NewSigner(context.Background(), tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, signer)
			}
		})
	}
}

func TestVaultSigner_IntegrationWithAuthz(t *testing.T) {
	// This test verifies that VaultSigner implements the authz.Signer interface
	storedKey := &storedKeyData{}
	server := mockVaultServerForSigner(t, storedKey)
	defer server.Close()

	client, err := NewClient(Config{
		Address: server.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)
	defer client.Close()

	signer, err := client.NewSigner(context.Background(), SignerConfig{
		KeyPath:      "test-key",
		Identity:     "system@specular.dev",
		AutoGenerate: true,
	})
	require.NoError(t, err)

	// Create a mock audit entry (simulated JSON)
	testData := []byte(`{"timestamp":"2024-01-01T00:00:00Z","allowed":true}`)

	// Sign the data (this is what SignedAuditLogger would do)
	signature, publicKey, err := signer.Sign(testData)
	require.NoError(t, err)

	// Verify signature
	assert.NotNil(t, signature)
	assert.NotNil(t, publicKey)
	assert.Equal(t, 64, len(signature))

	// Encode to base64 (as would be done in audit entry)
	signatureB64 := base64.StdEncoding.EncodeToString(signature)
	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKey)

	assert.NotEmpty(t, signatureB64)
	assert.NotEmpty(t, publicKeyB64)

	// Verify identity
	assert.Equal(t, "system@specular.dev", signer.Identity())

	// Decode and verify (as would be done by AuditVerifier)
	decodedSig, err := base64.StdEncoding.DecodeString(signatureB64)
	require.NoError(t, err)

	decodedPubKey, err := base64.StdEncoding.DecodeString(publicKeyB64)
	require.NoError(t, err)

	valid, err := signer.VerifySignature(testData, decodedSig, decodedPubKey)
	require.NoError(t, err)
	assert.True(t, valid)
}
