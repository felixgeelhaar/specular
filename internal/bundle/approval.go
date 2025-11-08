package bundle

import (
	"encoding/json"
	"time"
)

// Approval represents a team member's approval signature on a bundle.
// Approvals provide multi-stakeholder sign-off for governance bundles,
// ensuring that critical artifacts are reviewed and approved before use.
type Approval struct {
	// Role is the approval role (e.g., "pm", "lead", "security", "legal")
	Role string `json:"role" yaml:"role"`

	// User is the email or identifier of the approver
	User string `json:"user" yaml:"user"`

	// SignedAt is the timestamp when the approval was signed
	SignedAt time.Time `json:"signed_at" yaml:"signed_at"`

	// Signature is the cryptographic signature of the bundle digest
	// Format depends on SignatureType (SSH, GPG, etc.)
	Signature string `json:"signature" yaml:"signature"`

	// SignatureType indicates the signature format ("ssh", "gpg", "x509")
	SignatureType SignatureType `json:"signature_type" yaml:"signature_type"`

	// PublicKey is the public key used for verification
	// Format depends on SignatureType
	PublicKey string `json:"public_key" yaml:"public_key"`

	// PublicKeyFingerprint is the fingerprint of the public key
	// Used for quick key identification without storing full key
	PublicKeyFingerprint string `json:"public_key_fingerprint,omitempty" yaml:"public_key_fingerprint,omitempty"`

	// Comment is an optional comment from the approver
	Comment string `json:"comment,omitempty" yaml:"comment,omitempty"`

	// Metadata contains additional approval metadata
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// SignatureType represents the type of cryptographic signature used.
type SignatureType string

const (
	// SignatureTypeSSH indicates an SSH signature (ssh-keygen -Y sign)
	SignatureTypeSSH SignatureType = "ssh"

	// SignatureTypeGPG indicates a GPG signature
	SignatureTypeGPG SignatureType = "gpg"

	// SignatureTypeX509 indicates an X.509 certificate signature
	SignatureTypeX509 SignatureType = "x509"

	// SignatureTypeCosign indicates a Sigstore Cosign signature
	SignatureTypeCosign SignatureType = "cosign"
)

// ApprovalRequest contains information needed to create an approval signature.
type ApprovalRequest struct {
	// BundleDigest is the digest of the bundle being approved
	BundleDigest string

	// Role is the approval role being claimed
	Role string

	// User is the approver's identifier
	User string

	// Comment is an optional approval comment
	Comment string

	// SignatureType is the type of signature to create
	SignatureType SignatureType

	// KeyPath is the path to the private key for signing (optional)
	// If not provided, default keys will be used
	KeyPath string
}

// ApprovalVerificationOptions contains options for verifying approvals.
type ApprovalVerificationOptions struct {
	// BundleDigest is the expected bundle digest
	BundleDigest string

	// AllowedRoles lists roles that are valid for this bundle
	AllowedRoles []string

	// RequireAllRoles indicates if all allowed roles must have approvals
	RequireAllRoles bool

	// TrustedKeys lists trusted public keys or fingerprints
	// If empty, any key is accepted (not recommended for production)
	TrustedKeys []string

	// MaxAge is the maximum age of approvals (0 means no limit)
	MaxAge time.Duration

	// RequireComment indicates if comments are required
	RequireComment bool
}

// Validate checks if the approval is valid.
func (a *Approval) Validate() error {
	if a.Role == "" {
		return &ValidationError{
			Code:    ErrCodeInvalidSignature,
			Message: "approval role is required",
			Field:   "role",
		}
	}

	if a.User == "" {
		return &ValidationError{
			Code:    ErrCodeInvalidSignature,
			Message: "approval user is required",
			Field:   "user",
		}
	}

	if a.Signature == "" {
		return &ValidationError{
			Code:    ErrCodeInvalidSignature,
			Message: "approval signature is required",
			Field:   "signature",
		}
	}

	if a.SignatureType == "" {
		return &ValidationError{
			Code:    ErrCodeInvalidSignature,
			Message: "signature type is required",
			Field:   "signature_type",
		}
	}

	if a.PublicKey == "" {
		return &ValidationError{
			Code:    ErrCodeInvalidSignature,
			Message: "public key is required",
			Field:   "public_key",
		}
	}

	if a.SignedAt.IsZero() {
		return &ValidationError{
			Code:    ErrCodeInvalidSignature,
			Message: "signed timestamp is required",
			Field:   "signed_at",
		}
	}

	// Validate signature type is one of the supported types
	switch a.SignatureType {
	case SignatureTypeSSH, SignatureTypeGPG, SignatureTypeX509, SignatureTypeCosign:
		// Valid signature type
	default:
		return &ValidationError{
			Code:    ErrCodeInvalidSignature,
			Message: "unsupported signature type: " + string(a.SignatureType),
			Field:   "signature_type",
		}
	}

	return nil
}

// IsExpired checks if the approval is older than the given duration.
func (a *Approval) IsExpired(maxAge time.Duration) bool {
	if maxAge == 0 {
		return false
	}
	return time.Since(a.SignedAt) > maxAge
}

// MatchesFingerprint checks if the approval's public key matches the given fingerprint.
func (a *Approval) MatchesFingerprint(fingerprint string) bool {
	if a.PublicKeyFingerprint == "" {
		return false
	}
	return a.PublicKeyFingerprint == fingerprint
}

// ToJSON marshals the approval to pretty-printed JSON.
func (a *Approval) ToJSON() ([]byte, error) {
	return json.MarshalIndent(a, "", "  ")
}
