package attestation

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
)

// EphemeralSigner implements keyless signing using ephemeral keys
// This is a simplified implementation for Phase 1
// Full Sigstore OIDC integration can be added later
type EphemeralSigner struct {
	privateKey *ecdsa.PrivateKey
	identity   string
}

// NewEphemeralSigner creates a new ephemeral signer
func NewEphemeralSigner(identity string) (*EphemeralSigner, error) {
	// Generate ephemeral ECDSA key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	return &EphemeralSigner{
		privateKey: privateKey,
		identity:   identity,
	}, nil
}

// Sign generates a signature for the data
func (s *EphemeralSigner) Sign(data []byte) (signature []byte, publicKey crypto.PublicKey, err error) {
	// Hash the data
	hash := sha256.Sum256(data)

	// Sign the hash
	r, sigS, err := ecdsa.Sign(rand.Reader, s.privateKey, hash[:])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign: %w", err)
	}

	// Encode signature (r || s)
	signature = append(r.Bytes(), sigS.Bytes()...)

	return signature, &s.privateKey.PublicKey, nil
}

// Identity returns the identity of the signer
func (s *EphemeralSigner) Identity() string {
	return s.identity
}

// PublicKey returns the public key as PEM-encoded bytes
func (s *EphemeralSigner) PublicKey() ([]byte, error) {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&s.privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	return pubKeyBytes, nil
}

// EncodeSignature encodes a signature to base64
func EncodeSignature(signature []byte) string {
	return base64.StdEncoding.EncodeToString(signature)
}

// DecodeSignature decodes a base64 signature
func DecodeSignature(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// EncodePublicKey encodes a public key to base64
func EncodePublicKey(publicKey []byte) string {
	return base64.StdEncoding.EncodeToString(publicKey)
}

// DecodePublicKey decodes a base64 public key
func DecodePublicKey(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
