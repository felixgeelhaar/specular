package attestation

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"math/big"
	"time"
)

// StandardVerifier implements basic signature verification
type StandardVerifier struct {
	// Configuration options
	maxAge            time.Duration
	requireGitClean   bool
	allowedIdentities []string
}

// VerifierOption is a functional option for configuring the verifier
type VerifierOption func(*StandardVerifier)

// WithMaxAge sets the maximum age for attestations
func WithMaxAge(maxAge time.Duration) VerifierOption {
	return func(v *StandardVerifier) {
		v.maxAge = maxAge
	}
}

// WithRequireGitClean requires clean git status
func WithRequireGitClean(require bool) VerifierOption {
	return func(v *StandardVerifier) {
		v.requireGitClean = require
	}
}

// WithAllowedIdentities restricts allowed signer identities
func WithAllowedIdentities(identities []string) VerifierOption {
	return func(v *StandardVerifier) {
		v.allowedIdentities = identities
	}
}

// NewStandardVerifier creates a new verifier
func NewStandardVerifier(opts ...VerifierOption) *StandardVerifier {
	v := &StandardVerifier{
		maxAge:            24 * time.Hour, // Default 24 hours
		requireGitClean:   false,
		allowedIdentities: nil, // nil means allow all
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// Verify checks the signature on an attestation
func (v *StandardVerifier) Verify(attestation *Attestation) error {
	// 1. Verify signature is present
	if attestation.Signature == "" || attestation.PublicKey == "" {
		return fmt.Errorf("attestation is not signed")
	}

	// 2. Verify attestation age
	if v.maxAge > 0 {
		age := time.Since(attestation.SignedAt)
		if age > v.maxAge {
			return fmt.Errorf("attestation too old: %v (max %v)", age, v.maxAge)
		}
	}

	// 3. Verify signer identity (if restricted)
	if len(v.allowedIdentities) > 0 {
		allowed := false
		for _, identity := range v.allowedIdentities {
			if attestation.SignedBy == identity {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("signer identity not allowed: %s", attestation.SignedBy)
		}
	}

	// 4. Decode signature and public key
	signature, err := DecodeSignature(attestation.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	publicKeyBytes, err := DecodePublicKey(attestation.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	// 5. Parse public key
	publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	publicKey, ok := publicKeyInterface.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not ECDSA")
	}

	// 6. Recreate the data that was signed
	dataToVerify, err := v.recreateSignedData(attestation)
	if err != nil {
		return fmt.Errorf("failed to recreate signed data: %w", err)
	}

	// 7. Hash the data
	hash := sha256.Sum256(dataToVerify)

	// 8. Verify signature
	// Signature is r || s, each 32 bytes for P-256
	if len(signature) != 64 {
		return fmt.Errorf("invalid signature length: %d", len(signature))
	}

	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	if !ecdsa.Verify(publicKey, hash[:], r, s) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// VerifyProvenance validates the provenance data
func (v *StandardVerifier) VerifyProvenance(attestation *Attestation) error {
	// 1. Verify required fields are present
	if attestation.Provenance.Hostname == "" {
		return fmt.Errorf("provenance missing hostname")
	}
	if attestation.Provenance.Platform == "" {
		return fmt.Errorf("provenance missing platform")
	}
	if attestation.Provenance.Arch == "" {
		return fmt.Errorf("provenance missing architecture")
	}
	if attestation.Provenance.SpecularVersion == "" {
		return fmt.Errorf("provenance missing specular version")
	}

	// 2. Verify git status if required
	if v.requireGitClean && attestation.Provenance.GitDirty {
		return fmt.Errorf("provenance indicates dirty git status")
	}

	// 3. Verify workflow succeeded (if checking provenance)
	if attestation.Status != "success" {
		return fmt.Errorf("workflow did not succeed: %s", attestation.Status)
	}

	// 4. Verify timing makes sense
	if attestation.EndTime.Before(attestation.StartTime) {
		return fmt.Errorf("end time before start time")
	}

	// 5. Verify cost and task counts are reasonable
	if attestation.Provenance.TotalCost < 0 {
		return fmt.Errorf("negative total cost")
	}
	if attestation.Provenance.TasksExecuted < 0 {
		return fmt.Errorf("negative tasks executed")
	}
	if attestation.Provenance.TasksFailed < 0 {
		return fmt.Errorf("negative tasks failed")
	}

	return nil
}

// VerifyHashes verifies the plan and output hashes
func (v *StandardVerifier) VerifyHashes(attestation *Attestation, planJSON []byte, outputJSON []byte) error {
	// Compute hashes
	planHash := hashData(planJSON)
	outputHash := hashData(outputJSON)

	// Verify plan hash
	if planHash != attestation.PlanHash {
		return fmt.Errorf("plan hash mismatch: expected %s, got %s",
			attestation.PlanHash, planHash)
	}

	// Verify output hash
	if outputHash != attestation.OutputHash {
		return fmt.Errorf("output hash mismatch: expected %s, got %s",
			attestation.OutputHash, outputHash)
	}

	return nil
}

// recreateSignedData recreates the canonical data that was signed
func (v *StandardVerifier) recreateSignedData(attestation *Attestation) ([]byte, error) {
	// Create a copy without signature fields
	copy := *attestation
	copy.Signature = ""
	copy.PublicKey = ""

	// Serialize to canonical JSON
	return json.MarshalIndent(copy, "", "  ")
}
