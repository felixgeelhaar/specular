package awssm

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"
)

// AWSSecretsManagerSigner implements the authz.Signer interface using AWS Secrets Manager-stored ECDSA keys.
//
// This signer:
// - Stores ECDSA P-256 private keys securely in AWS Secrets Manager
// - Retrieves keys on-demand for signing operations
// - Supports key rotation with versioning (AWSCURRENT, AWSPREVIOUS)
// - Provides cryptographic non-repudiation for audit logs
// - Uses the same key format as VaultSigner for interoperability
type AWSSecretsManagerSigner struct {
	client     *Client
	secretName string
	identity   string

	// Cached key (optional, for performance)
	cachedKey    *ecdsa.PrivateKey
	cachedPubKey []byte
	cacheExpiry  time.Time
	cacheTTL     time.Duration
}

// SignerConfig holds configuration for an AWS Secrets Manager-backed signer.
type SignerConfig struct {
	// SecretName is the AWS Secrets Manager secret name where the ECDSA key is stored
	// Example: "specular/audit/signing-key"
	SecretName string

	// Identity is the signer identity (e.g., email or service name)
	// Example: "system@specular.dev"
	Identity string

	// CacheTTL is how long to cache the key in memory (default: 5 minutes)
	// Set to 0 to disable caching
	CacheTTL time.Duration

	// AutoGenerate will generate a new key if one doesn't exist
	AutoGenerate bool

	// Tags are applied to the secret when creating a new key
	Tags map[string]string
}

// NewSigner creates a new AWS Secrets Manager-backed signer.
func (c *Client) NewSigner(ctx context.Context, cfg SignerConfig) (*AWSSecretsManagerSigner, error) {
	if cfg.SecretName == "" {
		return nil, fmt.Errorf("secret name is required")
	}
	if cfg.Identity == "" {
		return nil, fmt.Errorf("identity is required")
	}

	// Set default cache TTL
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 5 * time.Minute
	}

	signer := &AWSSecretsManagerSigner{
		client:     c,
		secretName: cfg.SecretName,
		identity:   cfg.Identity,
		cacheTTL:   cfg.CacheTTL,
	}

	// Check if key exists
	secrets := c.Secrets()
	_, err := secrets.Get(ctx, cfg.SecretName)
	if err != nil {
		if cfg.AutoGenerate {
			// Generate new key
			if genErr := signer.GenerateKey(ctx, cfg.Tags); genErr != nil {
				return nil, fmt.Errorf("failed to generate key: %w", genErr)
			}
		} else {
			return nil, fmt.Errorf("key not found at %s and auto-generate is disabled: %w", cfg.SecretName, err)
		}
	}

	return signer, nil
}

// GenerateKey generates a new ECDSA P-256 key pair and stores it in AWS Secrets Manager.
func (s *AWSSecretsManagerSigner) GenerateKey(ctx context.Context, tags map[string]string) error {
	// Generate ECDSA P-256 key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate ECDSA key: %w", err)
	}

	// Encode private key in PKCS#8 format
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Encode public key in PKIX format
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Store in AWS Secrets Manager (base64-encoded for JSON safety)
	// Uses same format as VaultSigner for interoperability
	data := map[string]interface{}{
		"private_key": base64.StdEncoding.EncodeToString(privateKeyBytes),
		"public_key":  base64.StdEncoding.EncodeToString(publicKeyBytes),
		"algorithm":   "ECDSA-P256",
		"created_at":  time.Now().UTC().Format(time.RFC3339),
		"identity":    s.identity,
	}

	// Add algorithm tag
	if tags == nil {
		tags = make(map[string]string)
	}
	tags["algorithm"] = "ECDSA-P256"
	tags["identity"] = s.identity

	secrets := s.client.Secrets()
	if storeErr := secrets.PutWithTags(ctx, s.secretName, data, tags); storeErr != nil {
		return fmt.Errorf("failed to store key in AWS Secrets Manager: %w", storeErr)
	}

	// Update cache
	s.cachedKey = privateKey
	s.cachedPubKey = publicKeyBytes
	s.cacheExpiry = time.Now().Add(s.cacheTTL)

	return nil
}

// Sign generates a signature for the provided data.
//
// This implements the authz.Signer interface:
//
//	type Signer interface {
//	    Sign(data []byte) (signature []byte, publicKey []byte, err error)
//	    Identity() string
//	}
func (s *AWSSecretsManagerSigner) Sign(data []byte) (signature []byte, publicKey []byte, err error) {
	ctx := context.Background()

	// Get private key (from cache or AWS Secrets Manager)
	privateKey, pubKey, err := s.getKey(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get signing key: %w", err)
	}

	// Hash the data with SHA-256
	hash := sha256.Sum256(data)

	// Sign with ECDSA
	r, sValue, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign data: %w", err)
	}

	// Encode signature as r || s (64 bytes total: 32 + 32)
	sig := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := sValue.Bytes()

	// Pad r and s to 32 bytes
	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):64], sBytes)

	return sig, pubKey, nil
}

// Identity returns the signer identity.
func (s *AWSSecretsManagerSigner) Identity() string {
	return s.identity
}

// getKey retrieves the private key from cache or AWS Secrets Manager.
func (s *AWSSecretsManagerSigner) getKey(ctx context.Context) (*ecdsa.PrivateKey, []byte, error) {
	// Check cache
	if s.cachedKey != nil && time.Now().Before(s.cacheExpiry) {
		return s.cachedKey, s.cachedPubKey, nil
	}

	// Fetch from AWS Secrets Manager
	secrets := s.client.Secrets()
	secret, err := secrets.Get(ctx, s.secretName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read key from AWS Secrets Manager: %w", err)
	}

	// Decode private key
	privateKeyB64, ok := secret.Data["private_key"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("private_key not found in secret data")
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	// Parse PKCS#8 private key
	privateKeyInterface, err := x509.ParsePKCS8PrivateKey(privateKeyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	privateKey, ok := privateKeyInterface.(*ecdsa.PrivateKey)
	if !ok {
		return nil, nil, fmt.Errorf("key is not ECDSA private key")
	}

	// Decode public key
	publicKeyB64, ok := secret.Data["public_key"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("public_key not found in secret data")
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	// Update cache
	s.cachedKey = privateKey
	s.cachedPubKey = publicKeyBytes
	s.cacheExpiry = time.Now().Add(s.cacheTTL)

	return privateKey, publicKeyBytes, nil
}

// RotateKey generates a new key and stores it as the current version.
//
// The old key version remains available via AWSPREVIOUS stage for verifying old signatures,
// but new signatures will use the new key (AWSCURRENT).
func (s *AWSSecretsManagerSigner) RotateKey(ctx context.Context, tags map[string]string) error {
	// Clear cache to force new key generation
	s.cachedKey = nil
	s.cachedPubKey = nil
	s.cacheExpiry = time.Time{}

	// Generate and store new key (creates new version automatically)
	return s.GenerateKey(ctx, tags)
}

// GetPublicKey returns the public key without performing a signature.
func (s *AWSSecretsManagerSigner) GetPublicKey(ctx context.Context) ([]byte, error) {
	_, pubKey, err := s.getKey(ctx)
	return pubKey, err
}

// ClearCache clears the cached key, forcing next signature to fetch from AWS Secrets Manager.
func (s *AWSSecretsManagerSigner) ClearCache() {
	s.cachedKey = nil
	s.cachedPubKey = nil
	s.cacheExpiry = time.Time{}
}

// VerifySignature verifies a signature using the public key.
// This is useful for testing and validation.
func (s *AWSSecretsManagerSigner) VerifySignature(data, signature, publicKey []byte) (bool, error) {
	// Parse public key
	publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKey)
	if err != nil {
		return false, fmt.Errorf("failed to parse public key: %w", err)
	}

	ecdsaPubKey, ok := publicKeyInterface.(*ecdsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("public key is not ECDSA")
	}

	// Hash the data
	hash := sha256.Sum256(data)

	// Extract r and s from signature
	if len(signature) != 64 {
		return false, fmt.Errorf("invalid signature length: %d (expected 64)", len(signature))
	}

	r := new(big.Int).SetBytes(signature[:32])
	sValue := new(big.Int).SetBytes(signature[32:])

	// Verify signature
	valid := ecdsa.Verify(ecdsaPubKey, hash[:], r, sValue)
	return valid, nil
}

// KeyInfo returns information about the signing key.
type KeyInfo struct {
	Algorithm    string `json:"algorithm"`
	Identity     string `json:"identity"`
	CreatedAt    string `json:"created_at"`
	SecretName   string `json:"secret_name"`
	VersionID    string `json:"version_id,omitempty"`
	VersionStage string `json:"version_stage,omitempty"`
}

// GetKeyInfo retrieves metadata about the signing key.
func (s *AWSSecretsManagerSigner) GetKeyInfo(ctx context.Context) (*KeyInfo, error) {
	secrets := s.client.Secrets()
	secret, err := secrets.Get(ctx, s.secretName)
	if err != nil {
		return nil, fmt.Errorf("failed to read key from AWS Secrets Manager: %w", err)
	}

	info := &KeyInfo{
		SecretName:   s.secretName,
		VersionID:    secret.VersionID,
		VersionStage: secret.VersionStage,
	}

	if algorithm, ok := secret.Data["algorithm"].(string); ok {
		info.Algorithm = algorithm
	}

	if identity, ok := secret.Data["identity"].(string); ok {
		info.Identity = identity
	}

	if createdAt, ok := secret.Data["created_at"].(string); ok {
		info.CreatedAt = createdAt
	}

	return info, nil
}

// GetPreviousKey retrieves the previous version of the key for signature verification.
// This is useful during key rotation when verifying signatures made with the old key.
func (s *AWSSecretsManagerSigner) GetPreviousKey(ctx context.Context) ([]byte, error) {
	secrets := s.client.Secrets()
	secret, err := secrets.GetByStage(ctx, s.secretName, "AWSPREVIOUS")
	if err != nil {
		return nil, fmt.Errorf("failed to read previous key from AWS Secrets Manager: %w", err)
	}

	publicKeyB64, ok := secret.Data["public_key"].(string)
	if !ok {
		return nil, fmt.Errorf("public_key not found in previous secret data")
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode previous public key: %w", err)
	}

	return publicKeyBytes, nil
}
