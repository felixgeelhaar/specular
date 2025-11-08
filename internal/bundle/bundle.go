package bundle

import (
	"time"

	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/spec"
)

// Bundle represents a complete Specular governance bundle.
// A bundle is a portable, signed, and verifiable archive containing all
// governance configuration, approvals, and attestations.
type Bundle struct {
	// Manifest contains bundle metadata and integrity information
	Manifest *Manifest `json:"manifest"`

	// Spec is the product specification
	Spec *spec.ProductSpec `json:"spec,omitempty"`

	// SpecLock is the locked specification snapshot
	SpecLock *spec.SpecLock `json:"spec_lock,omitempty"`

	// Routing contains AI provider routing configuration
	Routing *router.Router `json:"routing,omitempty"`

	// Policies contains governance policies
	Policies []*policy.Policy `json:"policies,omitempty"`

	// Approvals contains team approval signatures
	Approvals []*Approval `json:"approvals,omitempty"`

	// Attestation contains cryptographic attestation (optional)
	Attestation *Attestation `json:"attestation,omitempty"`

	// Checksums maps file paths to their SHA-256 checksums
	Checksums map[string]string `json:"checksums"`

	// AdditionalFiles contains any additional files included in the bundle
	AdditionalFiles map[string][]byte `json:"additional_files,omitempty"`
}

// BundleInfo contains basic information about a bundle without loading all content.
// Useful for listing and comparing bundles without full extraction.
type BundleInfo struct {
	// ID is the unique bundle identifier (e.g., "acme/healthcare-api")
	ID string `json:"id"`

	// Version is the bundle version (e.g., "1.3.0")
	Version string `json:"version"`

	// Schema is the bundle format schema version
	Schema string `json:"schema"`

	// Created is when the bundle was created
	Created time.Time `json:"created"`

	// IntegrityDigest is the SHA-256 digest of the manifest
	IntegrityDigest string `json:"integrity_digest"`

	// GovernanceLevel indicates the governance maturity level (L1-L4)
	GovernanceLevel string `json:"governance_level,omitempty"`

	// ApprovalStatus indicates approval completion
	ApprovalStatus *ApprovalStatus `json:"approval_status,omitempty"`

	// HasAttestation indicates if bundle includes cryptographic attestation
	HasAttestation bool `json:"has_attestation"`

	// Size is the bundle size in bytes
	Size int64 `json:"size,omitempty"`
}

// ApprovalStatus tracks the approval progress for a bundle.
type ApprovalStatus struct {
	// RequiredRoles are the roles that must approve
	RequiredRoles []string `json:"required_roles"`

	// Completed is the number of completed approvals
	Completed int `json:"completed"`

	// Total is the total number of required approvals
	Total int `json:"total"`

	// ApprovedBy lists users who have approved
	ApprovedBy []string `json:"approved_by"`

	// PendingRoles lists roles that haven't approved yet
	PendingRoles []string `json:"pending_roles"`

	// Complete indicates if all required approvals are present
	Complete bool `json:"complete"`
}

// BundleOptions contains options for bundle creation.
type BundleOptions struct {
	// SpecPath is the path to spec.yaml
	SpecPath string

	// LockPath is the path to spec.lock.json
	LockPath string

	// RoutingPath is the path to routing.yaml
	RoutingPath string

	// PolicyPaths are paths to policy files
	PolicyPaths []string

	// IncludePaths are additional files/directories to include
	IncludePaths []string

	// RequireApprovals lists required approval roles
	RequireApprovals []string

	// AttestationFormat specifies attestation type ("sigstore", "in-toto", "")
	AttestationFormat string

	// Metadata contains additional bundle metadata
	Metadata map[string]string

	// GovernanceLevel indicates target governance level (L1-L4)
	GovernanceLevel string
}

// VerifyOptions contains options for bundle verification.
type VerifyOptions struct {
	// Strict requires all checksums, approvals, and attestations to be valid
	Strict bool

	// RequireApprovals enforces approval requirement checking
	RequireApprovals bool

	// RequireAttestation enforces attestation requirement
	RequireAttestation bool

	// PolicyPath is an optional policy file to verify against
	PolicyPath string

	// TrustPublicKeys are public keys to trust for signature verification
	TrustPublicKeys []string

	// AllowOffline permits offline verification (cached attestations)
	AllowOffline bool
}

// ApplyOptions contains options for applying a bundle to a project.
type ApplyOptions struct {
	// TargetDir is the directory to apply the bundle to
	TargetDir string

	// DryRun shows what would be applied without making changes
	DryRun bool

	// Force overwrites existing files without prompting
	Force bool

	// Yes auto-confirms all prompts
	Yes bool

	// Exclude patterns for files to skip
	Exclude []string
}

// DiffOptions contains options for comparing bundles.
type DiffOptions struct {
	// Format specifies output format ("text", "json", "markdown")
	Format string

	// ShowContent includes file content differences
	ShowContent bool

	// ContextLines specifies number of context lines in diffs
	ContextLines int

	// IgnoreWhitespace ignores whitespace changes
	IgnoreWhitespace bool
}

// BundleDiff represents the difference between two bundles or bundle vs current state.
type BundleDiff struct {
	// Summary provides high-level diff statistics
	Summary DiffSummary `json:"summary"`

	// ModifiedFiles lists files that changed
	ModifiedFiles []FileDiff `json:"modified_files,omitempty"`

	// AddedFiles lists newly added files
	AddedFiles []string `json:"added_files,omitempty"`

	// RemovedFiles lists deleted files
	RemovedFiles []string `json:"removed_files,omitempty"`

	// ApprovalChanges describes changes in approvals
	ApprovalChanges *ApprovalDiff `json:"approval_changes,omitempty"`

	// AttestationChanges describes attestation differences
	AttestationChanges *AttestationDiff `json:"attestation_changes,omitempty"`
}

// DiffSummary provides high-level statistics about a bundle diff.
type DiffSummary struct {
	// FilesModified is the count of modified files
	FilesModified int `json:"files_modified"`

	// FilesAdded is the count of added files
	FilesAdded int `json:"files_added"`

	// FilesRemoved is the count of removed files
	FilesRemoved int `json:"files_removed"`

	// LinesAdded is the total lines added
	LinesAdded int `json:"lines_added"`

	// LinesRemoved is the total lines removed
	LinesRemoved int `json:"lines_removed"`

	// ApprovalsChanged indicates if approvals differ
	ApprovalsChanged bool `json:"approvals_changed"`

	// AttestationChanged indicates if attestation differs
	AttestationChanged bool `json:"attestation_changed"`
}

// FileDiff represents the difference for a single file.
type FileDiff struct {
	// Path is the file path
	Path string `json:"path"`

	// LinesAdded is lines added in this file
	LinesAdded int `json:"lines_added"`

	// LinesRemoved is lines removed in this file
	LinesRemoved int `json:"lines_removed"`

	// Patch is the unified diff patch (optional)
	Patch string `json:"patch,omitempty"`
}

// ApprovalDiff describes changes in approvals between bundles.
type ApprovalDiff struct {
	// Added lists newly added approvals
	Added []string `json:"added,omitempty"`

	// Removed lists removed approvals
	Removed []string `json:"removed,omitempty"`

	// RoleChanges describes changes in required roles
	RoleChanges *RoleDiff `json:"role_changes,omitempty"`
}

// RoleDiff describes changes in required approval roles.
type RoleDiff struct {
	// Added lists newly required roles
	Added []string `json:"added,omitempty"`

	// Removed lists roles no longer required
	Removed []string `json:"removed,omitempty"`
}

// AttestationDiff describes changes in attestation.
type AttestationDiff struct {
	// BeforeFormat is the attestation format in the first bundle
	BeforeFormat string `json:"before_format,omitempty"`

	// AfterFormat is the attestation format in the second bundle
	AfterFormat string `json:"after_format,omitempty"`

	// Changed indicates if attestation changed
	Changed bool `json:"changed"`

	// Details provides human-readable change description
	Details string `json:"details,omitempty"`
}

// ValidationResult contains the results of bundle verification.
type ValidationResult struct {
	// Valid indicates if bundle passed all checks
	Valid bool `json:"valid"`

	// Errors lists validation errors
	Errors []ValidationError `json:"errors,omitempty"`

	// Warnings lists validation warnings
	Warnings []ValidationWarning `json:"warnings,omitempty"`

	// ChecksumValid indicates if all checksums matched
	ChecksumValid bool `json:"checksum_valid"`

	// ApprovalsValid indicates if approvals are valid
	ApprovalsValid bool `json:"approvals_valid"`

	// AttestationValid indicates if attestation is valid
	AttestationValid bool `json:"attestation_valid"`

	// PolicyCompliant indicates if bundle meets policy requirements
	PolicyCompliant bool `json:"policy_compliant,omitempty"`
}

// ValidationError represents a validation error.
type ValidationError struct {
	// Code is the error code (e.g., "CHECKSUM_MISMATCH", "MISSING_APPROVAL")
	Code string `json:"code"`

	// Message is a human-readable error message
	Message string `json:"message"`

	// Field indicates which field/file caused the error
	Field string `json:"field,omitempty"`

	// Details provides additional error context
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return e.Code + ": " + e.Message + " (field: " + e.Field + ")"
	}
	return e.Code + ": " + e.Message
}

// ValidationWarning represents a validation warning.
type ValidationWarning struct {
	// Code is the warning code
	Code string `json:"code"`

	// Message is a human-readable warning message
	Message string `json:"message"`

	// Field indicates which field/file caused the warning
	Field string `json:"field,omitempty"`
}

// Error implements the error interface for warnings.
func (w *ValidationWarning) Error() string {
	if w.Field != "" {
		return w.Code + ": " + w.Message + " (field: " + w.Field + ")"
	}
	return w.Code + ": " + w.Message
}

// Error codes for bundle validation
const (
	ErrCodeChecksumMismatch  = "CHECKSUM_MISMATCH"
	ErrCodeMissingFile       = "MISSING_FILE"
	ErrCodeInvalidManifest   = "INVALID_MANIFEST"
	ErrCodeMissingApproval   = "MISSING_APPROVAL"
	ErrCodeInvalidSignature  = "INVALID_SIGNATURE"
	ErrCodeAttestationFailed = "ATTESTATION_FAILED"
	ErrCodePolicyViolation   = "POLICY_VIOLATION"
	ErrCodeUnsupportedSchema = "UNSUPPORTED_SCHEMA"
	ErrCodeCorruptedBundle   = "CORRUPTED_BUNDLE"
)

// Warning codes for bundle validation
const (
	WarnCodeOptionalFileMissing = "OPTIONAL_FILE_MISSING"
	WarnCodeExpiringSoon        = "EXPIRING_SOON"
	WarnCodeDeprecatedFeature   = "DEPRECATED_FEATURE"
	WarnCodeNoAttestation       = "NO_ATTESTATION"
	WarnCodePartialApprovals    = "PARTIAL_APPROVALS"
)
