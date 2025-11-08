package bundle

import "time"

// Attestation represents cryptographic attestation of a bundle.
// Attestations provide tamper-proof evidence of bundle origin, integrity,
// and compliance using industry-standard formats like Sigstore and in-toto.
type Attestation struct {
	// Format is the attestation format ("sigstore", "in-toto", "custom")
	Format AttestationFormat `json:"format" yaml:"format"`

	// Subject identifies what is being attested (the bundle)
	Subject AttestationSubject `json:"subject" yaml:"subject"`

	// Predicate contains the attestation content
	// Structure depends on Format and PredicateType
	Predicate interface{} `json:"predicate" yaml:"predicate"`

	// PredicateType identifies the type of predicate
	// Examples: "https://slsa.dev/provenance/v1", "https://in-toto.io/Statement/v1"
	PredicateType string `json:"predicate_type" yaml:"predicate_type"`

	// Signature contains the cryptographic signature bundle
	Signature AttestationSignature `json:"signature" yaml:"signature"`

	// RekorEntry contains the Rekor transparency log entry (optional)
	// Provides tamper-proof audit trail via public ledger
	RekorEntry *RekorEntry `json:"rekor_entry,omitempty" yaml:"rekor_entry,omitempty"`

	// Timestamp is when the attestation was created
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`

	// Metadata contains additional attestation metadata
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// AttestationFormat represents the attestation format type.
type AttestationFormat string

const (
	// AttestationFormatSigstore indicates Sigstore/Cosign attestation
	AttestationFormatSigstore AttestationFormat = "sigstore"

	// AttestationFormatInToto indicates in-toto attestation format
	AttestationFormatInToto AttestationFormat = "in-toto"

	// AttestationFormatSLSA indicates SLSA provenance attestation
	AttestationFormatSLSA AttestationFormat = "slsa"

	// AttestationFormatCustom indicates a custom attestation format
	AttestationFormatCustom AttestationFormat = "custom"
)

// AttestationSubject identifies the artifact being attested.
type AttestationSubject struct {
	// Name is the subject name (e.g., bundle ID)
	Name string `json:"name" yaml:"name"`

	// Digest contains cryptographic digests of the subject
	// Format: map[algorithm]digest (e.g., {"sha256": "abc123..."})
	Digest map[string]string `json:"digest" yaml:"digest"`
}

// AttestationSignature contains the cryptographic signature information.
type AttestationSignature struct {
	// Signature is the cryptographic signature bytes (base64 encoded)
	Signature string `json:"signature" yaml:"signature"`

	// PublicKey is the public key used for verification (optional)
	// Not needed for Sigstore keyless signing
	PublicKey string `json:"public_key,omitempty" yaml:"public_key,omitempty"`

	// Certificate is the X.509 certificate chain (for Sigstore)
	Certificate string `json:"certificate,omitempty" yaml:"certificate,omitempty"`

	// SignatureAlgorithm is the signature algorithm used
	SignatureAlgorithm string `json:"signature_algorithm,omitempty" yaml:"signature_algorithm,omitempty"`
}

// RekorEntry represents an entry in the Rekor transparency log.
type RekorEntry struct {
	// UUID is the unique identifier for this Rekor entry
	UUID string `json:"uuid" yaml:"uuid"`

	// LogIndex is the index in the Rekor log
	LogIndex int64 `json:"log_index" yaml:"log_index"`

	// IntegratedTime is when the entry was integrated into the log
	IntegratedTime int64 `json:"integrated_time" yaml:"integrated_time"`

	// InclusionProof is the cryptographic proof of inclusion
	InclusionProof string `json:"inclusion_proof,omitempty" yaml:"inclusion_proof,omitempty"`

	// Body is the Rekor entry body
	Body string `json:"body,omitempty" yaml:"body,omitempty"`
}

// SLSAProvenance represents SLSA provenance predicate.
// See: https://slsa.dev/spec/v1.0/provenance
type SLSAProvenance struct {
	// BuildType identifies the build system that produced this artifact
	BuildType string `json:"buildType" yaml:"buildType"`

	// Builder identifies the transitive closure of the build system
	Builder SLSABuilder `json:"builder" yaml:"builder"`

	// Invocation describes how the build was invoked
	Invocation SLSAInvocation `json:"invocation,omitempty" yaml:"invocation,omitempty"`

	// BuildConfig contains the input to the build
	BuildConfig interface{} `json:"buildConfig,omitempty" yaml:"buildConfig,omitempty"`

	// Metadata contains additional metadata about the build
	Metadata SLSAMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Materials lists the artifacts that influenced the build
	Materials []SLSAMaterial `json:"materials,omitempty" yaml:"materials,omitempty"`
}

// SLSABuilder identifies the builder that produced the artifact.
type SLSABuilder struct {
	// ID is the unique identifier for the builder
	ID string `json:"id" yaml:"id"`

	// Version is the builder version (optional)
	Version map[string]string `json:"version,omitempty" yaml:"version,omitempty"`

	// BuilderDependencies lists dependencies of the builder itself
	BuilderDependencies []SLSAMaterial `json:"builderDependencies,omitempty" yaml:"builderDependencies,omitempty"`
}

// SLSAInvocation describes how the build was invoked.
type SLSAInvocation struct {
	// ConfigSource describes where the config file came from
	ConfigSource SLSAConfigSource `json:"configSource,omitempty" yaml:"configSource,omitempty"`

	// Parameters contains the parameters passed to the build
	Parameters interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`

	// Environment describes the build environment
	Environment interface{} `json:"environment,omitempty" yaml:"environment,omitempty"`
}

// SLSAConfigSource describes the source of the build configuration.
type SLSAConfigSource struct {
	// URI is the URI of the config source
	URI string `json:"uri,omitempty" yaml:"uri,omitempty"`

	// Digest is the cryptographic digest of the config
	Digest map[string]string `json:"digest,omitempty" yaml:"digest,omitempty"`

	// EntryPoint is the entry point within the config
	EntryPoint string `json:"entryPoint,omitempty" yaml:"entryPoint,omitempty"`
}

// SLSAMetadata contains metadata about the build.
type SLSAMetadata struct {
	// BuildInvocationID is a unique identifier for this build execution
	BuildInvocationID string `json:"buildInvocationId,omitempty" yaml:"buildInvocationId,omitempty"`

	// BuildStartedOn is when the build started (RFC3339)
	BuildStartedOn string `json:"buildStartedOn,omitempty" yaml:"buildStartedOn,omitempty"`

	// BuildFinishedOn is when the build finished (RFC3339)
	BuildFinishedOn string `json:"buildFinishedOn,omitempty" yaml:"buildFinishedOn,omitempty"`

	// Completeness describes the completeness of the provenance
	Completeness SLSACompleteness `json:"completeness,omitempty" yaml:"completeness,omitempty"`

	// Reproducible indicates if the build is reproducible
	Reproducible bool `json:"reproducible,omitempty" yaml:"reproducible,omitempty"`
}

// SLSACompleteness describes the completeness guarantees of the provenance.
type SLSACompleteness struct {
	// Parameters indicates if all parameters are included
	Parameters bool `json:"parameters,omitempty" yaml:"parameters,omitempty"`

	// Environment indicates if the environment is fully described
	Environment bool `json:"environment,omitempty" yaml:"environment,omitempty"`

	// Materials indicates if all materials are listed
	Materials bool `json:"materials,omitempty" yaml:"materials,omitempty"`
}

// SLSAMaterial represents an artifact that influenced the build.
type SLSAMaterial struct {
	// URI is the URI of the material
	URI string `json:"uri,omitempty" yaml:"uri,omitempty"`

	// Digest is the cryptographic digest of the material
	Digest map[string]string `json:"digest,omitempty" yaml:"digest,omitempty"`
}

// AttestationOptions contains options for creating attestations.
type AttestationOptions struct {
	// Format is the attestation format to create
	Format AttestationFormat

	// PredicateType is the type of predicate to include
	PredicateType string

	// KeyPath is the path to the signing key (optional for Sigstore)
	KeyPath string

	// UseKeyless enables Sigstore keyless signing
	UseKeyless bool

	// RekorURL is the URL of the Rekor server
	// Default: "https://rekor.sigstore.dev"
	RekorURL string

	// FulcioURL is the URL of the Fulcio CA server
	// Default: "https://fulcio.sigstore.dev"
	FulcioURL string

	// OIDCIssuer is the OIDC issuer for keyless signing
	OIDCIssuer string

	// OIDCClientID is the OIDC client ID for keyless signing
	OIDCClientID string

	// IncludeRekorEntry enables Rekor transparency log inclusion
	IncludeRekorEntry bool

	// Metadata contains additional attestation metadata
	Metadata map[string]string
}

// AttestationVerificationOptions contains options for verifying attestations.
type AttestationVerificationOptions struct {
	// TrustedRootPath is the path to the trusted root certificates
	TrustedRootPath string

	// RekorURL is the URL of the Rekor server for verification
	RekorURL string

	// RequireRekorEntry requires a valid Rekor entry
	RequireRekorEntry bool

	// TrustedIdentities lists trusted identities for keyless signing
	// Format: email, subject, or issuer patterns
	TrustedIdentities []string

	// VerifySignature enables signature verification
	VerifySignature bool

	// VerifyTimestamp enables timestamp verification
	VerifyTimestamp bool

	// MaxAge is the maximum age of attestations (0 means no limit)
	MaxAge time.Duration
}

// Validate checks if the attestation is valid.
func (a *Attestation) Validate() error {
	if a.Format == "" {
		return &ValidationError{
			Code:    ErrCodeAttestationFailed,
			Message: "attestation format is required",
			Field:   "format",
		}
	}

	if a.Subject.Name == "" {
		return &ValidationError{
			Code:    ErrCodeAttestationFailed,
			Message: "attestation subject name is required",
			Field:   "subject.name",
		}
	}

	if len(a.Subject.Digest) == 0 {
		return &ValidationError{
			Code:    ErrCodeAttestationFailed,
			Message: "attestation subject digest is required",
			Field:   "subject.digest",
		}
	}

	if a.PredicateType == "" {
		return &ValidationError{
			Code:    ErrCodeAttestationFailed,
			Message: "predicate type is required",
			Field:   "predicate_type",
		}
	}

	if a.Signature.Signature == "" {
		return &ValidationError{
			Code:    ErrCodeAttestationFailed,
			Message: "attestation signature is required",
			Field:   "signature.signature",
		}
	}

	if a.Timestamp.IsZero() {
		return &ValidationError{
			Code:    ErrCodeAttestationFailed,
			Message: "attestation timestamp is required",
			Field:   "timestamp",
		}
	}

	return nil
}

// IsExpired checks if the attestation is older than the given duration.
func (a *Attestation) IsExpired(maxAge time.Duration) bool {
	if maxAge == 0 {
		return false
	}
	return time.Since(a.Timestamp) > maxAge
}

// HasRekorEntry checks if the attestation includes a Rekor transparency log entry.
func (a *Attestation) HasRekorEntry() bool {
	return a.RekorEntry != nil && a.RekorEntry.UUID != ""
}
