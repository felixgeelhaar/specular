package authz

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"math/big"
	"time"
)

// AuditVerifier verifies cryptographic signatures on audit entries.
type AuditVerifier struct {
	// maxAge is the maximum age allowed for entries (0 = no limit)
	maxAge time.Duration

	// allowedSigners restricts which signer identities are trusted (nil = allow all)
	allowedSigners []string
}

// VerifierOption configures an AuditVerifier.
type VerifierOption func(*AuditVerifier)

// WithMaxAge sets the maximum age for audit entries.
func WithMaxAge(maxAge time.Duration) VerifierOption {
	return func(v *AuditVerifier) {
		v.maxAge = maxAge
	}
}

// WithAllowedSigners restricts verification to specific signer identities.
func WithAllowedSigners(signers []string) VerifierOption {
	return func(v *AuditVerifier) {
		v.allowedSigners = signers
	}
}

// NewAuditVerifier creates a new audit entry verifier.
func NewAuditVerifier(opts ...VerifierOption) *AuditVerifier {
	v := &AuditVerifier{
		maxAge:         0, // No age limit by default
		allowedSigners: nil,
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// VerificationResult contains the result of verifying an audit entry.
type VerificationResult struct {
	Valid      bool        `json:"valid"`
	Entry      *AuditEntry `json:"entry"`
	Reason     string      `json:"reason,omitempty"`
	VerifiedAt time.Time   `json:"verified_at"`
}

// Verify checks the cryptographic signature on an audit entry.
func (v *AuditVerifier) Verify(entry *AuditEntry) (*VerificationResult, error) {
	result := &VerificationResult{
		Entry:      entry,
		VerifiedAt: time.Now(),
	}

	// 1. Check if entry is signed
	if entry.Signature == "" || entry.PublicKey == "" {
		result.Valid = false
		result.Reason = "entry is not signed"
		return result, nil
	}

	// 2. Check signer identity (if restricted)
	if len(v.allowedSigners) > 0 {
		allowed := false
		for _, signer := range v.allowedSigners {
			if entry.SignedBy == signer {
				allowed = true
				break
			}
		}
		if !allowed {
			result.Valid = false
			result.Reason = fmt.Sprintf("signer not allowed: %s", entry.SignedBy)
			return result, nil
		}
	}

	// 3. Check entry age (if maxAge is set)
	if v.maxAge > 0 {
		age := time.Since(entry.Timestamp)
		if age > v.maxAge {
			result.Valid = false
			result.Reason = fmt.Sprintf("entry too old: %v (max %v)", age, v.maxAge)
			return result, nil
		}
	}

	// 4. Decode signature and public key
	signature, err := decodeBase64(entry.Signature)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	publicKeyBytes, err := decodeBase64(entry.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	// 5. Parse public key
	publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	publicKey, ok := publicKeyInterface.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not ECDSA")
	}

	// 6. Recreate the canonical data that was signed
	dataToVerify, err := v.recreateSignedData(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to recreate signed data: %w", err)
	}

	// 7. Hash the data
	hash := sha256.Sum256(dataToVerify)

	// 8. Verify signature (ECDSA P-256 produces 64-byte signatures: r || s)
	if len(signature) != 64 {
		result.Valid = false
		result.Reason = fmt.Sprintf("invalid signature length: %d (expected 64)", len(signature))
		return result, nil
	}

	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	if !ecdsa.Verify(publicKey, hash[:], r, s) {
		result.Valid = false
		result.Reason = "signature verification failed"
		return result, nil
	}

	// Signature is valid!
	result.Valid = true
	result.Reason = "signature verified successfully"
	return result, nil
}

// VerifyBatch verifies multiple audit entries and returns results for each.
func (v *AuditVerifier) VerifyBatch(entries []*AuditEntry) ([]*VerificationResult, error) {
	results := make([]*VerificationResult, len(entries))

	for i, entry := range entries {
		result, err := v.Verify(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to verify entry %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

// recreateSignedData recreates the canonical JSON that was signed.
func (v *AuditVerifier) recreateSignedData(entry *AuditEntry) ([]byte, error) {
	// Create a copy without signature fields
	copy := *entry
	copy.Signature = ""
	copy.PublicKey = ""
	copy.SignedBy = ""

	// Serialize to canonical JSON (same format as signing)
	return json.MarshalIndent(copy, "", "  ")
}

// VerificationSummary provides statistics about a batch verification.
type VerificationSummary struct {
	Total      int       `json:"total"`
	Valid      int       `json:"valid"`
	Invalid    int       `json:"invalid"`
	Unsigned   int       `json:"unsigned"`
	VerifiedAt time.Time `json:"verified_at"`
}

// Summarize creates a summary of verification results.
func Summarize(results []*VerificationResult) *VerificationSummary {
	summary := &VerificationSummary{
		Total:      len(results),
		VerifiedAt: time.Now(),
	}

	for _, result := range results {
		if result.Entry.Signature == "" {
			summary.Unsigned++
		} else if result.Valid {
			summary.Valid++
		} else {
			summary.Invalid++
		}
	}

	return summary
}
