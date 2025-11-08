package bundle

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test end-to-end bundle creation and loading
func TestBundleLifecycle(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	specPath := filepath.Join(tempDir, "spec.yaml")
	specContent := `product: test-bundle
goals:
  - Test bundle validation
features: []
non_functional:
  performance: []
  security: []
  scalability: []
acceptance: []
milestones: []
`
	require.NoError(t, os.WriteFile(specPath, []byte(specContent), 0644))

	lockPath := filepath.Join(tempDir, "spec.lock.json")
	lockContent := `{
  "version": "1.0.0",
  "spec_hash": "abc123",
  "locked_at": "2024-01-01T00:00:00Z"
}
`
	require.NoError(t, os.WriteFile(lockPath, []byte(lockContent), 0644))

	routingPath := filepath.Join(tempDir, "routing.yaml")
	routingContent := `default_model: gpt-4
fallback_models:
  - gpt-3.5-turbo
`
	require.NoError(t, os.WriteFile(routingPath, []byte(routingContent), 0644))

	// Build bundle
	opts := BundleOptions{
		SpecPath:        specPath,
		LockPath:        lockPath,
		RoutingPath:     routingPath,
		GovernanceLevel: "L2",
		Metadata: map[string]string{
			"team":    "platform",
			"project": "test",
		},
	}

	builder, err := NewBuilder(opts)
	require.NoError(t, err)
	assert.NotNil(t, builder)

	bundlePath := filepath.Join(tempDir, "test.sbundle.tgz")
	err = builder.Build(bundlePath)
	require.NoError(t, err)

	// Verify bundle file exists
	stat, err := os.Stat(bundlePath)
	require.NoError(t, err)
	assert.Greater(t, stat.Size(), int64(0))

	// Get bundle info
	info, err := GetBundleInfo(bundlePath)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "test-bundle", info.ID)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "L2", info.GovernanceLevel)
	assert.Equal(t, BundleSchemaVersion, info.Schema)

	// Verify bundle
	validator := NewValidator(VerifyOptions{})
	result, err := validator.Verify(bundlePath)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.True(t, result.ChecksumValid)
	assert.Len(t, result.Errors, 0)

	// Load bundle
	bundle, err := LoadBundle(bundlePath)
	require.NoError(t, err)
	assert.NotNil(t, bundle)
	assert.NotNil(t, bundle.Manifest)
	assert.Equal(t, "test-bundle", bundle.Manifest.ID)
}

func TestBundleWithPolicies(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	specPath := filepath.Join(tempDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte("product: policy-test\ngoals: []\nfeatures: []\nnon_functional:\n  performance: []\n  security: []\n  scalability: []\nacceptance: []\nmilestones: []\n"), 0644))

	lockPath := filepath.Join(tempDir, "spec.lock.json")
	require.NoError(t, os.WriteFile(lockPath, []byte(`{"version": "1.0.0"}`), 0644))

	routingPath := filepath.Join(tempDir, "routing.yaml")
	require.NoError(t, os.WriteFile(routingPath, []byte("default_model: gpt-4\n"), 0644))

	// Create policy file
	policyPath := filepath.Join(tempDir, "policy.yaml")
	policyContent := `name: docker-policy
description: Docker image allowlist policy
rules:
  - allow: docker.io/library/*
  - allow: ghcr.io/myorg/*
`
	require.NoError(t, os.WriteFile(policyPath, []byte(policyContent), 0644))

	// Build bundle with policy
	opts := BundleOptions{
		SpecPath:    specPath,
		LockPath:    lockPath,
		RoutingPath: routingPath,
		PolicyPaths: []string{policyPath},
	}

	builder, err := NewBuilder(opts)
	require.NoError(t, err)

	bundlePath := filepath.Join(tempDir, "policy-bundle.sbundle.tgz")
	err = builder.Build(bundlePath)
	require.NoError(t, err)

	// Verify bundle
	validator := NewValidator(VerifyOptions{})
	result, err := validator.Verify(bundlePath)
	require.NoError(t, err)
	assert.True(t, result.Valid)
}

func TestBundleWithApprovals(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	specPath := filepath.Join(tempDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte("product: approval-test\ngoals: []\nfeatures: []\nnon_functional:\n  performance: []\n  security: []\n  scalability: []\nacceptance: []\nmilestones: []\n"), 0644))

	lockPath := filepath.Join(tempDir, "spec.lock.json")
	require.NoError(t, os.WriteFile(lockPath, []byte(`{"version": "1.0.0"}`), 0644))

	routingPath := filepath.Join(tempDir, "routing.yaml")
	require.NoError(t, os.WriteFile(routingPath, []byte("default_model: gpt-4\n"), 0644))

	// Build bundle with required approvals
	opts := BundleOptions{
		SpecPath:         specPath,
		LockPath:         lockPath,
		RoutingPath:      routingPath,
		RequireApprovals: []string{"pm", "lead", "security"},
	}

	builder, err := NewBuilder(opts)
	require.NoError(t, err)

	bundlePath := filepath.Join(tempDir, "approval-bundle.sbundle.tgz")
	err = builder.Build(bundlePath)
	require.NoError(t, err)

	// Verify without requiring approvals
	validator := NewValidator(VerifyOptions{
		RequireApprovals: false,
	})
	result, err := validator.Verify(bundlePath)
	require.NoError(t, err)
	assert.True(t, result.Valid)

	// Verify with requiring approvals (should fail - no approvals present)
	validatorStrict := NewValidator(VerifyOptions{
		RequireApprovals: true,
	})
	resultStrict, err := validatorStrict.Verify(bundlePath)
	require.NoError(t, err)
	assert.False(t, resultStrict.Valid)
	assert.Greater(t, len(resultStrict.Errors), 0)
}

func TestBundleChecksumValidation(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	specPath := filepath.Join(tempDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte("product: checksum-test\ngoals: []\nfeatures: []\nnon_functional:\n  performance: []\n  security: []\n  scalability: []\nacceptance: []\nmilestones: []\n"), 0644))

	lockPath := filepath.Join(tempDir, "spec.lock.json")
	require.NoError(t, os.WriteFile(lockPath, []byte(`{"version": "1.0.0"}`), 0644))

	routingPath := filepath.Join(tempDir, "routing.yaml")
	require.NoError(t, os.WriteFile(routingPath, []byte("default_model: gpt-4\n"), 0644))

	// Build bundle
	opts := BundleOptions{
		SpecPath:    specPath,
		LockPath:    lockPath,
		RoutingPath: routingPath,
	}

	builder, err := NewBuilder(opts)
	require.NoError(t, err)

	bundlePath := filepath.Join(tempDir, "checksum-bundle.sbundle.tgz")
	err = builder.Build(bundlePath)
	require.NoError(t, err)

	// Verify checksums
	validator := NewValidator(VerifyOptions{})
	result, err := validator.Verify(bundlePath)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.True(t, result.ChecksumValid)
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Code:    ErrCodeChecksumMismatch,
		Message: "checksum does not match",
		Field:   "spec.yaml",
	}

	assert.Equal(t, "CHECKSUM_MISMATCH: checksum does not match (field: spec.yaml)", err.Error())

	errNoField := &ValidationError{
		Code:    ErrCodeInvalidManifest,
		Message: "invalid manifest",
	}

	assert.Equal(t, "INVALID_MANIFEST: invalid manifest", errNoField.Error())
}

func TestValidationWarning(t *testing.T) {
	warning := &ValidationWarning{
		Code:    WarnCodeNoAttestation,
		Message: "no attestation provided",
		Field:   "attestation",
	}

	assert.Equal(t, "NO_ATTESTATION: no attestation provided (field: attestation)", warning.Error())
}

func TestApproval_Validate(t *testing.T) {
	tests := []struct {
		name     string
		approval *Approval
		wantErr  bool
	}{
		{
			name: "valid approval",
			approval: &Approval{
				Role:          "pm",
				User:          "john.doe@example.com",
				SignedAt:      time.Now(),
				Signature:     "valid-signature-data",
				SignatureType: SignatureTypeSSH,
				PublicKey:     "ssh-rsa AAAA...",
			},
			wantErr: false,
		},
		{
			name: "missing role",
			approval: &Approval{
				User:          "john.doe@example.com",
				SignedAt:      time.Now(),
				Signature:     "valid-signature-data",
				SignatureType: SignatureTypeSSH,
			},
			wantErr: true,
		},
		{
			name: "missing user",
			approval: &Approval{
				Role:          "pm",
				SignedAt:      time.Now(),
				Signature:     "valid-signature-data",
				SignatureType: SignatureTypeSSH,
			},
			wantErr: true,
		},
		{
			name: "missing signature",
			approval: &Approval{
				Role:          "pm",
				User:          "john.doe@example.com",
				SignedAt:      time.Now(),
				SignatureType: SignatureTypeSSH,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.approval.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAttestation_Validate(t *testing.T) {
	tests := []struct {
		name        string
		attestation *Attestation
		wantErr     bool
	}{
		{
			name: "valid sigstore attestation",
			attestation: &Attestation{
				Format:    AttestationFormatSigstore,
				Timestamp: time.Now(),
				Subject: AttestationSubject{
					Name:   "test-bundle@1.0.0",
					Digest: map[string]string{"sha256": "abc123"},
				},
				PredicateType: "https://slsa.dev/provenance/v1",
				Signature: AttestationSignature{
					Signature: "valid-attestation-signature",
				},
			},
			wantErr: false,
		},
		{
			name: "valid in-toto attestation",
			attestation: &Attestation{
				Format:    AttestationFormatInToto,
				Timestamp: time.Now(),
				Subject: AttestationSubject{
					Name:   "test-bundle@1.0.0",
					Digest: map[string]string{"sha256": "abc123"},
				},
				PredicateType: "https://in-toto.io/Statement/v1",
				Signature: AttestationSignature{
					Signature: "valid-attestation-signature",
				},
			},
			wantErr: false,
		},
		{
			name: "missing subject name",
			attestation: &Attestation{
				Format:    AttestationFormatSigstore,
				Timestamp: time.Now(),
				Subject: AttestationSubject{
					Digest: map[string]string{"sha256": "abc123"},
				},
				Signature: AttestationSignature{
					Signature: "valid-attestation-signature",
				},
			},
			wantErr: true,
		},
		{
			name: "missing signature",
			attestation: &Attestation{
				Format:    AttestationFormatSigstore,
				Timestamp: time.Now(),
				Subject: AttestationSubject{
					Name:   "test-bundle@1.0.0",
					Digest: map[string]string{"sha256": "abc123"},
				},
				Signature: AttestationSignature{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.attestation.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
