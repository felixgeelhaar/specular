package awssm

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAWSSecretsManagerSigner_SignatureFormat(t *testing.T) {
	// Test that signature format matches VaultSigner format
	// Generate a test key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Test data
	data := []byte("test data for signing")

	// Hash and sign
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	require.NoError(t, err)

	// Create signature in the same format as the signer
	sig := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):64], sBytes)

	// Verify signature length
	assert.Len(t, sig, 64, "Signature should be 64 bytes")

	// Verify we can extract r and s back
	rExtracted := new(big.Int).SetBytes(sig[:32])
	sExtracted := new(big.Int).SetBytes(sig[32:])

	// Verify signature
	valid := ecdsa.Verify(&privateKey.PublicKey, hash[:], rExtracted, sExtracted)
	assert.True(t, valid, "Signature should be valid")
}

func TestAWSSecretsManagerSigner_KeyFormat(t *testing.T) {
	// Test that key encoding matches VaultSigner format
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Encode private key in PKCS#8 format
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	// Encode public key in PKIX format
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	// Base64 encode (as stored in Secrets Manager)
	privateKeyB64 := base64.StdEncoding.EncodeToString(privateKeyBytes)
	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKeyBytes)

	// Decode and parse back
	decodedPrivate, err := base64.StdEncoding.DecodeString(privateKeyB64)
	require.NoError(t, err)

	decodedPublic, err := base64.StdEncoding.DecodeString(publicKeyB64)
	require.NoError(t, err)

	// Parse private key
	parsedPrivateKey, err := x509.ParsePKCS8PrivateKey(decodedPrivate)
	require.NoError(t, err)

	ecdsaPrivateKey, ok := parsedPrivateKey.(*ecdsa.PrivateKey)
	assert.True(t, ok, "Should be ECDSA private key")

	// Parse public key
	parsedPublicKey, err := x509.ParsePKIXPublicKey(decodedPublic)
	require.NoError(t, err)

	ecdsaPublicKey, ok := parsedPublicKey.(*ecdsa.PublicKey)
	assert.True(t, ok, "Should be ECDSA public key")

	// Verify the keys match
	assert.Equal(t, ecdsaPrivateKey.PublicKey.X, ecdsaPublicKey.X)
	assert.Equal(t, ecdsaPrivateKey.PublicKey.Y, ecdsaPublicKey.Y)
}

func TestAWSSecretsManagerSigner_VerifySignature(t *testing.T) {
	// Test signature verification
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Encode public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	// Create a mock signer for testing VerifySignature
	signer := &AWSSecretsManagerSigner{}

	// Sign some data
	data := []byte("test message for verification")
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	require.NoError(t, err)

	// Create signature
	sig := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):64], sBytes)

	// Verify
	valid, err := signer.VerifySignature(data, sig, publicKeyBytes)
	require.NoError(t, err)
	assert.True(t, valid, "Signature should be valid")

	// Test with tampered data
	tamperedData := []byte("tampered message")
	valid, err = signer.VerifySignature(tamperedData, sig, publicKeyBytes)
	require.NoError(t, err)
	assert.False(t, valid, "Signature should be invalid for tampered data")

	// Test with invalid signature length
	invalidSig := []byte("too short")
	valid, err = signer.VerifySignature(data, invalidSig, publicKeyBytes)
	assert.Error(t, err)
	assert.False(t, valid)
	assert.Contains(t, err.Error(), "invalid signature length")
}

func TestAWSSecretsManagerSigner_Identity(t *testing.T) {
	signer := &AWSSecretsManagerSigner{
		identity: "test@example.com",
	}

	assert.Equal(t, "test@example.com", signer.Identity())
}

func TestAWSSecretsManagerSigner_ClearCache(t *testing.T) {
	// Create a signer with cached key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	signer := &AWSSecretsManagerSigner{
		cachedKey:    privateKey,
		cachedPubKey: publicKeyBytes,
		cacheExpiry:  time.Now().Add(5 * time.Minute),
		cacheTTL:     5 * time.Minute,
	}

	// Verify cache is set
	assert.NotNil(t, signer.cachedKey)
	assert.NotNil(t, signer.cachedPubKey)
	assert.False(t, signer.cacheExpiry.IsZero())

	// Clear cache
	signer.ClearCache()

	// Verify cache is cleared
	assert.Nil(t, signer.cachedKey)
	assert.Nil(t, signer.cachedPubKey)
	assert.True(t, signer.cacheExpiry.IsZero())
}

func TestKeyInfo_Structure(t *testing.T) {
	info := &KeyInfo{
		Algorithm:    "ECDSA-P256",
		Identity:     "system@specular.dev",
		CreatedAt:    "2024-01-01T00:00:00Z",
		SecretName:   "specular/audit/signing-key",
		VersionID:    "abc123",
		VersionStage: "AWSCURRENT",
	}

	assert.Equal(t, "ECDSA-P256", info.Algorithm)
	assert.Equal(t, "system@specular.dev", info.Identity)
	assert.Equal(t, "2024-01-01T00:00:00Z", info.CreatedAt)
	assert.Equal(t, "specular/audit/signing-key", info.SecretName)
	assert.Equal(t, "abc123", info.VersionID)
	assert.Equal(t, "AWSCURRENT", info.VersionStage)
}

func TestSignerConfig_Defaults(t *testing.T) {
	cfg := SignerConfig{
		SecretName: "test-key",
		Identity:   "test@example.com",
	}

	// Default values should be applied when creating signer
	assert.Equal(t, time.Duration(0), cfg.CacheTTL, "Default CacheTTL is 0 in config")
	assert.False(t, cfg.AutoGenerate, "Default AutoGenerate is false")
	assert.Nil(t, cfg.Tags, "Default Tags is nil")
}

func TestAWSSecretsManagerSigner_KeyDataFormat(t *testing.T) {
	// Test that the key data format matches what we expect
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	identity := "system@specular.dev"
	createdAt := time.Now().UTC().Format(time.RFC3339)

	// This is the format stored in AWS Secrets Manager
	data := map[string]interface{}{
		"private_key": base64.StdEncoding.EncodeToString(privateKeyBytes),
		"public_key":  base64.StdEncoding.EncodeToString(publicKeyBytes),
		"algorithm":   "ECDSA-P256",
		"created_at":  createdAt,
		"identity":    identity,
	}

	// Verify all required fields are present
	assert.Contains(t, data, "private_key")
	assert.Contains(t, data, "public_key")
	assert.Contains(t, data, "algorithm")
	assert.Contains(t, data, "created_at")
	assert.Contains(t, data, "identity")

	// Verify algorithm
	assert.Equal(t, "ECDSA-P256", data["algorithm"])
	assert.Equal(t, identity, data["identity"])
}
