package bundle

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Validator verifies bundle integrity, checksums, and signatures.
type Validator struct {
	opts       VerifyOptions
	bundle     *Bundle
	bundlePath string
}

// NewValidator creates a new bundle validator with the given options.
func NewValidator(opts VerifyOptions) *Validator {
	return &Validator{
		opts:   opts,
		bundle: &Bundle{},
	}
}

// Verify validates a bundle and returns the validation result.
func (v *Validator) Verify(bundlePath string) (*ValidationResult, error) {
	// Store bundle path for signature verification
	v.bundlePath = bundlePath
	result := &ValidationResult{
		Valid:            true,
		Errors:           []ValidationError{},
		Warnings:         []ValidationWarning{},
		ChecksumValid:    true,
		ApprovalsValid:   true,
		AttestationValid: true,
		PolicyCompliant:  true,
	}

	// Extract bundle to temporary directory
	tempDir, err := v.extractBundle(bundlePath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    ErrCodeCorruptedBundle,
			Message: fmt.Sprintf("failed to extract bundle: %v", err),
		})
		return result, nil
	}
	defer os.RemoveAll(tempDir)

	// Load and validate manifest
	if err := v.loadManifest(tempDir); err != nil {
		result.Valid = false
		result.ChecksumValid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    ErrCodeInvalidManifest,
			Message: fmt.Sprintf("failed to load manifest: %v", err),
			Field:   "manifest",
		})
		return result, nil
	}

	// Validate manifest structure
	if err := v.bundle.Manifest.Validate(); err != nil {
		result.Valid = false
		if verr, ok := err.(*ValidationError); ok {
			result.Errors = append(result.Errors, *verr)
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeInvalidManifest,
				Message: err.Error(),
			})
		}
	}

	// Verify file checksums
	if !v.verifyChecksums(tempDir, result) {
		result.Valid = false
		result.ChecksumValid = false
	}

	// Verify approvals if required
	if v.opts.RequireApprovals {
		if err := v.loadApprovals(tempDir); err != nil {
			result.Valid = false
			result.ApprovalsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeMissingApproval,
				Message: fmt.Sprintf("failed to load approvals: %v", err),
			})
		} else if !v.verifyApprovals(result) {
			result.Valid = false
			result.ApprovalsValid = false
		}
	}

	// Verify attestation if required
	if v.opts.RequireAttestation {
		if err := v.loadAttestation(tempDir); err != nil {
			result.Valid = false
			result.AttestationValid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeAttestationFailed,
				Message: fmt.Sprintf("failed to load attestation: %v", err),
			})
		} else if !v.verifyAttestation(result) {
			result.Valid = false
			result.AttestationValid = false
		}
	}

	// Apply strict mode validation
	if v.opts.Strict && !result.Valid {
		return result, fmt.Errorf("bundle validation failed in strict mode")
	}

	return result, nil
}

// extractBundle extracts a .sbundle.tgz file to a temporary directory.
func (v *Validator) extractBundle(bundlePath string) (string, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "bundle-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Open bundle file
	file, err := os.Open(bundlePath)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to open bundle: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract all files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			os.RemoveAll(tempDir)
			return "", fmt.Errorf("failed to read tar: %w", err)
		}

		// Construct target path
		targetPath := filepath.Join(tempDir, header.Name)

		// Ensure target path is within temp directory (prevent path traversal)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(tempDir)) {
			os.RemoveAll(tempDir)
			return "", fmt.Errorf("invalid file path in bundle: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				os.RemoveAll(tempDir)
				return "", fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			// Create parent directory
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				os.RemoveAll(tempDir)
				return "", fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				os.RemoveAll(tempDir)
				return "", fmt.Errorf("failed to create file: %w", err)
			}

			// Copy data
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				os.RemoveAll(tempDir)
				return "", fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()
		}
	}

	return tempDir, nil
}

// loadManifest loads the manifest from the extracted bundle.
func (v *Validator) loadManifest(tempDir string) error {
	manifestPath := filepath.Join(tempDir, ManifestFileName)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	v.bundle.Manifest = &manifest
	return nil
}

// verifyChecksums verifies all file checksums match the manifest.
func (v *Validator) verifyChecksums(tempDir string, result *ValidationResult) bool {
	allValid := true

	for _, fileEntry := range v.bundle.Manifest.Files {
		filePath := filepath.Join(tempDir, fileEntry.Path)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			allValid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeMissingFile,
				Message: fmt.Sprintf("file not found: %s", fileEntry.Path),
				Field:   fileEntry.Path,
			})
			continue
		}

		// Calculate checksum
		checksum, err := v.calculateFileChecksum(filePath)
		if err != nil {
			allValid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeChecksumMismatch,
				Message: fmt.Sprintf("failed to calculate checksum for %s: %v", fileEntry.Path, err),
				Field:   fileEntry.Path,
			})
			continue
		}

		// Verify checksum matches
		if checksum != fileEntry.Checksum {
			allValid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeChecksumMismatch,
				Message: fmt.Sprintf("checksum mismatch for %s", fileEntry.Path),
				Field:   fileEntry.Path,
				Details: map[string]interface{}{
					"expected": fileEntry.Checksum,
					"actual":   checksum,
				},
			})
		}
	}

	return allValid
}

// calculateFileChecksum calculates the SHA-256 checksum of a file.
func (v *Validator) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// loadApprovals loads approvals from the extracted bundle.
func (v *Validator) loadApprovals(tempDir string) error {
	approvalsDir := filepath.Join(tempDir, "approvals")

	// Check if approvals directory exists
	if _, err := os.Stat(approvalsDir); os.IsNotExist(err) {
		return fmt.Errorf("approvals directory not found")
	}

	// Read all approval files
	entries, err := os.ReadDir(approvalsDir)
	if err != nil {
		return fmt.Errorf("failed to read approvals directory: %w", err)
	}

	approvals := []*Approval{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		approvalPath := filepath.Join(approvalsDir, entry.Name())
		data, err := os.ReadFile(approvalPath)
		if err != nil {
			return fmt.Errorf("failed to read approval file %s: %w", entry.Name(), err)
		}

		var approval Approval
		if err := yaml.Unmarshal(data, &approval); err != nil {
			return fmt.Errorf("failed to parse approval file %s: %w", entry.Name(), err)
		}

		approvals = append(approvals, &approval)
	}

	v.bundle.Approvals = approvals
	return nil
}

// verifyApprovals verifies all required approvals are present and valid.
func (v *Validator) verifyApprovals(result *ValidationResult) bool {
	if v.bundle.Manifest == nil {
		result.Errors = append(result.Errors, ValidationError{
			Code:    ErrCodeInvalidManifest,
			Message: "manifest is required for approval verification",
		})
		return false
	}

	requiredRoles := v.bundle.Manifest.RequiredApprovals
	if len(requiredRoles) == 0 {
		// No approvals required
		return true
	}

	// Compute bundle digest for signature verification
	bundleDigest, err := ComputeBundleDigest(v.bundlePath)
	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Code:    ErrCodeChecksumMismatch,
			Message: fmt.Sprintf("failed to compute bundle digest: %v", err),
		})
		return false
	}

	// Create verifier for signature validation
	verifier := NewVerifier(ApprovalVerificationOptions{
		BundleDigest: bundleDigest,
	})

	// Check each required role has an approval
	approvedRoles := make(map[string]bool)
	allValid := true

	for _, approval := range v.bundle.Approvals {
		// Validate approval structure
		if err := approval.Validate(); err != nil {
			allValid = false
			if verr, ok := err.(*ValidationError); ok {
				result.Errors = append(result.Errors, *verr)
			} else {
				result.Errors = append(result.Errors, ValidationError{
					Code:    ErrCodeInvalidSignature,
					Message: err.Error(),
				})
			}
			continue
		}

		// Verify approval signature
		if err := verifier.VerifyApproval(approval); err != nil {
			allValid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeInvalidSignature,
				Message: fmt.Sprintf("signature verification failed for role %s (%s): %v",
					approval.Role, approval.User, err),
				Field:   "approvals",
				Details: map[string]interface{}{
					"role": approval.Role,
					"user": approval.User,
				},
			})
			continue
		}

		approvedRoles[approval.Role] = true
	}

	// Check all required roles are approved
	for _, role := range requiredRoles {
		if !approvedRoles[role] {
			allValid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeMissingApproval,
				Message: fmt.Sprintf("missing valid approval for role: %s", role),
				Field:   "approvals",
				Details: map[string]interface{}{
					"role": role,
				},
			})
		}
	}

	return allValid
}

// loadAttestation loads attestation from the extracted bundle.
func (v *Validator) loadAttestation(tempDir string) error {
	attestationPath := filepath.Join(tempDir, "attestations", "attestation.yaml")

	// Check if attestation file exists
	if _, err := os.Stat(attestationPath); os.IsNotExist(err) {
		return fmt.Errorf("attestation file not found")
	}

	data, err := os.ReadFile(attestationPath)
	if err != nil {
		return fmt.Errorf("failed to read attestation: %w", err)
	}

	var attestation Attestation
	if err := yaml.Unmarshal(data, &attestation); err != nil {
		return fmt.Errorf("failed to parse attestation: %w", err)
	}

	v.bundle.Attestation = &attestation
	return nil
}

// verifyAttestation verifies the attestation is valid.
func (v *Validator) verifyAttestation(result *ValidationResult) bool {
	if v.bundle.Attestation == nil {
		result.Errors = append(result.Errors, ValidationError{
			Code:    ErrCodeAttestationFailed,
			Message: "attestation is missing",
		})
		return false
	}

	// Validate attestation structure
	if err := v.bundle.Attestation.Validate(); err != nil {
		if verr, ok := err.(*ValidationError); ok {
			result.Errors = append(result.Errors, *verr)
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeAttestationFailed,
				Message: err.Error(),
			})
		}
		return false
	}

	// Perform cryptographic verification using AttestationVerifier
	verifyOpts := AttestationVerificationOptions{
		VerifySignature:   true,
		RequireRekorEntry: false, // Rekor not yet fully implemented
		VerifyTimestamp:   true,
		MaxAge:            0, // No age restriction by default
	}

	verifier := NewAttestationVerifier(verifyOpts)

	// Verify attestation against bundle
	ctx := context.Background()
	if err := verifier.VerifyAttestation(ctx, v.bundle.Attestation, v.bundlePath); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Code:    ErrCodeAttestationFailed,
			Message: fmt.Sprintf("attestation verification failed: %v", err),
			Field:   "attestation",
		})
		return false
	}

	// Add informational warning about Rekor if entry exists but verification is disabled
	if v.bundle.Attestation.HasRekorEntry() && !verifyOpts.RequireRekorEntry {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Code:    "REKOR_NOT_VERIFIED",
			Message: "attestation includes Rekor entry but Rekor verification is not yet fully implemented",
			Field:   "attestation.rekor_entry",
		})
	}

	return true
}

// LoadBundle loads a bundle from a .sbundle.tgz file.
func LoadBundle(bundlePath string) (*Bundle, error) {
	validator := NewValidator(VerifyOptions{
		Strict:             false,
		RequireApprovals:   false,
		RequireAttestation: false,
	})

	result, err := validator.Verify(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load bundle: %w", err)
	}

	if !result.Valid {
		return nil, fmt.Errorf("bundle validation failed: %d errors", len(result.Errors))
	}

	return validator.bundle, nil
}
