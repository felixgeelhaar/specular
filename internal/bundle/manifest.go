package bundle

import "time"

// Manifest contains bundle metadata and integrity information.
// The manifest is the single source of truth for bundle identity,
// versioning, and verification requirements.
type Manifest struct {
	// Schema is the bundle format schema version (e.g., "specular.bundle/v1")
	Schema string `json:"schema" yaml:"schema"`

	// ID is the unique bundle identifier (e.g., "acme/healthcare-api")
	ID string `json:"id" yaml:"id"`

	// Version is the bundle version (e.g., "1.3.0")
	Version string `json:"version" yaml:"version"`

	// Created is the bundle creation timestamp
	Created time.Time `json:"created" yaml:"created"`

	// Integrity contains cryptographic integrity information
	Integrity IntegrityInfo `json:"integrity" yaml:"integrity"`

	// GovernanceLevel indicates the governance maturity level (L1-L4)
	// L1: Reactive - Basic policies and approvals
	// L2: Managed - Structured workflows and validation
	// L3: Optimized - Automated compliance and attestation
	// L4: Autonomous - Self-healing and adaptive governance
	GovernanceLevel string `json:"governance_level,omitempty" yaml:"governance_level,omitempty"`

	// RequiredApprovals lists the roles that must approve this bundle
	RequiredApprovals []string `json:"required_approvals,omitempty" yaml:"required_approvals,omitempty"`

	// Metadata contains additional bundle metadata
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Description provides human-readable bundle description
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Authors lists bundle creators or maintainers
	Authors []string `json:"authors,omitempty" yaml:"authors,omitempty"`

	// Tags provides searchable labels for bundle categorization
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Dependencies lists other bundles this bundle depends on
	Dependencies []BundleDependency `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`

	// Files lists all files included in the bundle with checksums
	Files []FileEntry `json:"files" yaml:"files"`
}

// IntegrityInfo contains cryptographic integrity information for the bundle.
type IntegrityInfo struct {
	// Algorithm is the hash algorithm used (e.g., "sha256")
	Algorithm string `json:"algorithm" yaml:"algorithm"`

	// Digest is the cryptographic digest of the bundle contents
	// Format: "algorithm:hexdigest" (e.g., "sha256:6f1a2e...")
	Digest string `json:"digest" yaml:"digest"`

	// ManifestDigest is the digest of the manifest file itself
	// Used for tamper detection of the manifest
	ManifestDigest string `json:"manifest_digest,omitempty" yaml:"manifest_digest,omitempty"`
}

// BundleDependency represents a dependency on another bundle.
type BundleDependency struct {
	// ID is the bundle identifier (e.g., "acme/shared-policies")
	ID string `json:"id" yaml:"id"`

	// Version is the required version or version constraint
	// Supports semver ranges (e.g., "^1.2.0", ">=1.0.0 <2.0.0")
	Version string `json:"version" yaml:"version"`

	// Optional indicates if the dependency is optional
	Optional bool `json:"optional,omitempty" yaml:"optional,omitempty"`

	// Digest is the expected integrity digest for the dependency
	Digest string `json:"digest,omitempty" yaml:"digest,omitempty"`
}

// FileEntry represents a file included in the bundle with integrity information.
type FileEntry struct {
	// Path is the relative file path within the bundle
	Path string `json:"path" yaml:"path"`

	// Size is the file size in bytes
	Size int64 `json:"size" yaml:"size"`

	// Checksum is the SHA-256 checksum of the file
	Checksum string `json:"checksum" yaml:"checksum"`

	// Mode is the Unix file mode (permissions)
	Mode uint32 `json:"mode,omitempty" yaml:"mode,omitempty"`

	// ContentType is the MIME type of the file (optional)
	ContentType string `json:"content_type,omitempty" yaml:"content_type,omitempty"`
}

// Validate checks if the manifest is valid.
func (m *Manifest) Validate() error {
	if m.Schema == "" {
		return &ValidationError{
			Code:    ErrCodeInvalidManifest,
			Message: "manifest schema is required",
			Field:   "schema",
		}
	}

	if m.ID == "" {
		return &ValidationError{
			Code:    ErrCodeInvalidManifest,
			Message: "bundle ID is required",
			Field:   "id",
		}
	}

	if m.Version == "" {
		return &ValidationError{
			Code:    ErrCodeInvalidManifest,
			Message: "bundle version is required",
			Field:   "version",
		}
	}

	if m.Integrity.Algorithm == "" {
		return &ValidationError{
			Code:    ErrCodeInvalidManifest,
			Message: "integrity algorithm is required",
			Field:   "integrity.algorithm",
		}
	}

	if m.Integrity.Digest == "" {
		return &ValidationError{
			Code:    ErrCodeInvalidManifest,
			Message: "integrity digest is required",
			Field:   "integrity.digest",
		}
	}

	if len(m.Files) == 0 {
		return &ValidationError{
			Code:    ErrCodeInvalidManifest,
			Message: "bundle must contain at least one file",
			Field:   "files",
		}
	}

	return nil
}

// GetFile returns the file entry for the given path, or nil if not found.
func (m *Manifest) GetFile(path string) *FileEntry {
	for i := range m.Files {
		if m.Files[i].Path == path {
			return &m.Files[i]
		}
	}
	return nil
}

// HasFile checks if the manifest contains a file with the given path.
func (m *Manifest) HasFile(path string) bool {
	return m.GetFile(path) != nil
}
