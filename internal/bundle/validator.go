package bundle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

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

	// Extract bundle to temporary directory using shared function
	tempDir, extractErr := extractBundle(bundlePath)
	if extractErr != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    ErrCodeCorruptedBundle,
			Message: fmt.Sprintf("failed to extract bundle: %v", extractErr),
		})
		return result, nil
	}
	defer cleanupOnError(tempDir)

	// Load and validate manifest
	if loadManifestErr := v.loadManifest(tempDir); loadManifestErr != nil {
		result.Valid = false
		result.ChecksumValid = false

		// Provide user-friendly error message
		bundleErr := ErrInvalidManifest("manifest file is missing or unreadable", loadManifestErr)
		result.Errors = append(result.Errors, ValidationError{
			Code:    ErrCodeInvalidManifest,
			Message: bundleErr.Error(),
			Field:   "manifest",
		})
		return result, nil
	}

	// Validate manifest structure
	if validateErr := v.bundle.Manifest.Validate(); validateErr != nil {
		result.Valid = false
		if verr, ok := validateErr.(*ValidationError); ok {
			result.Errors = append(result.Errors, *verr)
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeInvalidManifest,
				Message: validateErr.Error(),
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
		if loadApprovalsErr := v.loadApprovals(tempDir); loadApprovalsErr != nil {
			result.Valid = false
			result.ApprovalsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeMissingApproval,
				Message: fmt.Sprintf("failed to load approvals: %v", loadApprovalsErr),
			})
		} else if !v.verifyApprovals(result) {
			result.Valid = false
			result.ApprovalsValid = false
		}
	}

	// Verify attestation if required
	if v.opts.RequireAttestation {
		if loadAttestationErr := v.loadAttestation(tempDir); loadAttestationErr != nil {
			result.Valid = false
			result.AttestationValid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeAttestationFailed,
				Message: fmt.Sprintf("failed to load attestation: %v", loadAttestationErr),
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

// loadManifest loads the manifest from the extracted bundle.
func (v *Validator) loadManifest(tempDir string) error {
	manifestPath := filepath.Join(tempDir, ManifestFileName)
	data, readErr := os.ReadFile(manifestPath)
	if readErr != nil {
		return fmt.Errorf("failed to read manifest: %w", readErr)
	}

	var manifest Manifest
	if unmarshalErr := yaml.Unmarshal(data, &manifest); unmarshalErr != nil {
		return fmt.Errorf("failed to parse manifest: %w", unmarshalErr)
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

			// Use improved error message with actionable suggestion
			bundleErr := ErrChecksumMismatch(fileEntry.Path, fileEntry.Checksum, checksum)
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeChecksumMismatch,
				Message: bundleErr.Error(),
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
	file, openErr := os.Open(filePath)
	if openErr != nil {
		return "", openErr
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close file %s: %v\n", filePath, closeErr)
		}
	}()

	hash := sha256.New()
	if _, copyErr := io.Copy(hash, file); copyErr != nil {
		return "", copyErr
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// loadApprovals loads approvals from the extracted bundle.
func (v *Validator) loadApprovals(tempDir string) error {
	approvalsDir := filepath.Join(tempDir, "approvals")

	// Check if approvals directory exists
	if _, statErr := os.Stat(approvalsDir); os.IsNotExist(statErr) {
		return fmt.Errorf("approvals directory not found")
	}

	// Read all approval files
	entries, readDirErr := os.ReadDir(approvalsDir)
	if readDirErr != nil {
		return fmt.Errorf("failed to read approvals directory: %w", readDirErr)
	}

	approvals := []*Approval{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		approvalPath := filepath.Join(approvalsDir, entry.Name())
		data, readErr := os.ReadFile(approvalPath)
		if readErr != nil {
			return fmt.Errorf("failed to read approval file %s: %w", entry.Name(), readErr)
		}

		var approval Approval
		if unmarshalErr := yaml.Unmarshal(data, &approval); unmarshalErr != nil {
			return fmt.Errorf("failed to parse approval file %s: %w", entry.Name(), unmarshalErr)
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
		if validateErr := approval.Validate(); validateErr != nil {
			allValid = false
			if verr, ok := validateErr.(*ValidationError); ok {
				result.Errors = append(result.Errors, *verr)
			} else {
				result.Errors = append(result.Errors, ValidationError{
					Code:    ErrCodeInvalidSignature,
					Message: validateErr.Error(),
				})
			}
			continue
		}

		// Verify approval signature
		if verifyErr := verifier.VerifyApproval(approval); verifyErr != nil {
			allValid = false
			result.Errors = append(result.Errors, ValidationError{
				Code: ErrCodeInvalidSignature,
				Message: fmt.Sprintf("signature verification failed for role %s (%s): %v",
					approval.Role, approval.User, verifyErr),
				Field: "approvals",
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

			// Use improved error message with actionable suggestion
			bundleErr := ErrMissingApproval(role)
			result.Errors = append(result.Errors, ValidationError{
				Code:    ErrCodeMissingApproval,
				Message: bundleErr.Error(),
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
	if _, statErr := os.Stat(attestationPath); os.IsNotExist(statErr) {
		return fmt.Errorf("attestation file not found")
	}

	data, readErr := os.ReadFile(attestationPath)
	if readErr != nil {
		return fmt.Errorf("failed to read attestation: %w", readErr)
	}

	var attestation Attestation
	if unmarshalErr := yaml.Unmarshal(data, &attestation); unmarshalErr != nil {
		return fmt.Errorf("failed to parse attestation: %w", unmarshalErr)
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
