//go:build integration

package bundle

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewAttestationGenerator tests attestation generator initialization
func TestNewAttestationGenerator(t *testing.T) {
	tests := []struct {
		name string
		opts AttestationOptions
		want struct {
			rekorURL      string
			fulcioURL     string
			predicateType string
		}
	}{
		{
			name: "defaults applied when empty",
			opts: AttestationOptions{},
			want: struct {
				rekorURL      string
				fulcioURL     string
				predicateType string
			}{
				rekorURL:      "https://rekor.sigstore.dev",
				fulcioURL:     "https://fulcio.sigstore.dev",
				predicateType: "https://in-toto.io/Statement/v1",
			},
		},
		{
			name: "custom URLs preserved",
			opts: AttestationOptions{
				RekorURL:      "https://custom-rekor.example.com",
				FulcioURL:     "https://custom-fulcio.example.com",
				PredicateType: "https://custom.predicate.type/v1",
			},
			want: struct {
				rekorURL      string
				fulcioURL     string
				predicateType string
			}{
				rekorURL:      "https://custom-rekor.example.com",
				fulcioURL:     "https://custom-fulcio.example.com",
				predicateType: "https://custom.predicate.type/v1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewAttestationGenerator(tt.opts)

			if gen == nil {
				t.Fatal("NewAttestationGenerator returned nil")
			}

			if gen.opts.RekorURL != tt.want.rekorURL {
				t.Errorf("RekorURL = %v, want %v", gen.opts.RekorURL, tt.want.rekorURL)
			}

			if gen.opts.FulcioURL != tt.want.fulcioURL {
				t.Errorf("FulcioURL = %v, want %v", gen.opts.FulcioURL, tt.want.fulcioURL)
			}

			if gen.opts.PredicateType != tt.want.predicateType {
				t.Errorf("PredicateType = %v, want %v", gen.opts.PredicateType, tt.want.predicateType)
			}
		})
	}
}

// TestCreateSLSAProvenance tests SLSA provenance creation
func TestCreateSLSAProvenance(t *testing.T) {
	gen := NewAttestationGenerator(AttestationOptions{
		Format: AttestationFormatSLSA,
	})

	bundlePath := "testdata/test-bundle.tar"
	provenance := gen.createSLSAProvenance(bundlePath)

	if provenance == nil {
		t.Fatal("createSLSAProvenance returned nil")
	}

	// Verify build type
	if provenance.BuildType != "https://specular.dev/bundle/v1" {
		t.Errorf("BuildType = %v, want https://specular.dev/bundle/v1", provenance.BuildType)
	}

	// Verify builder ID
	if provenance.Builder.ID != "https://specular.dev/builder@v1" {
		t.Errorf("Builder.ID = %v, want https://specular.dev/builder@v1", provenance.Builder.ID)
	}

	// Verify builder version
	if provenance.Builder.Version["specular"] != "v1.3.0" {
		t.Errorf("Builder.Version[specular] = %v, want v1.3.0", provenance.Builder.Version["specular"])
	}

	// Verify invocation config source
	if provenance.Invocation.ConfigSource.URI != bundlePath {
		t.Errorf("Invocation.ConfigSource.URI = %v, want %v", provenance.Invocation.ConfigSource.URI, bundlePath)
	}

	// Verify metadata completeness
	if !provenance.Metadata.Completeness.Parameters {
		t.Error("Metadata.Completeness.Parameters should be true")
	}
	if !provenance.Metadata.Completeness.Environment {
		t.Error("Metadata.Completeness.Environment should be true")
	}
	if !provenance.Metadata.Completeness.Materials {
		t.Error("Metadata.Completeness.Materials should be true")
	}
	if !provenance.Metadata.Reproducible {
		t.Error("Metadata.Reproducible should be true")
	}

	// Verify timestamps are set
	if provenance.Metadata.BuildStartedOn == "" {
		t.Error("BuildStartedOn should not be empty")
	}
	if provenance.Metadata.BuildFinishedOn == "" {
		t.Error("BuildFinishedOn should not be empty")
	}

	// Verify materials
	if len(provenance.Materials) != 1 {
		t.Errorf("Materials length = %d, want 1", len(provenance.Materials))
	}
	if len(provenance.Materials) > 0 && provenance.Materials[0].URI != bundlePath {
		t.Errorf("Materials[0].URI = %v, want %v", provenance.Materials[0].URI, bundlePath)
	}

	t.Logf("SLSA Provenance: BuildType=%s, Builder=%s, Materials=%d",
		provenance.BuildType, provenance.Builder.ID, len(provenance.Materials))
}

// TestCreateInTotoStatement tests in-toto statement creation
func TestCreateInTotoStatement(t *testing.T) {
	gen := NewAttestationGenerator(AttestationOptions{
		Format: AttestationFormatInToto,
	})

	bundlePath := "testdata/test-bundle.tar"
	statement := gen.createInTotoStatement(bundlePath)

	if statement == nil {
		t.Fatal("createInTotoStatement returned nil")
	}

	// Verify builder
	builder, ok := statement["builder"].(map[string]string)
	if !ok {
		t.Fatal("builder field missing or wrong type")
	}
	if builder["id"] != "https://specular.dev/builder@v1" {
		t.Errorf("builder.id = %v, want https://specular.dev/builder@v1", builder["id"])
	}

	// Verify metadata
	metadata, ok := statement["metadata"].(map[string]string)
	if !ok {
		t.Fatal("metadata field missing or wrong type")
	}
	if metadata["buildStartedOn"] == "" {
		t.Error("buildStartedOn should not be empty")
	}
	if metadata["buildFinishedOn"] == "" {
		t.Error("buildFinishedOn should not be empty")
	}

	// Verify materials
	materials, ok := statement["materials"].([]map[string]string)
	if !ok {
		t.Fatal("materials field missing or wrong type")
	}
	if len(materials) != 1 {
		t.Errorf("materials length = %d, want 1", len(materials))
	}
	if len(materials) > 0 && materials[0]["uri"] != bundlePath {
		t.Errorf("materials[0].uri = %v, want %v", materials[0]["uri"], bundlePath)
	}

	t.Logf("In-Toto Statement: builder=%v, materials=%d", builder["id"], len(materials))
}

// TestGenerateAttestationWithKey tests key-based attestation generation
func TestGenerateAttestationWithKey(t *testing.T) {
	// Get path to test key
	keyPath := filepath.Join("testdata", "test-ec-key.pem")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Skipf("Test key not found: %s", keyPath)
	}

	// Get path to test bundle
	bundlePath := filepath.Join("testdata", "test-bundle.tar")
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		t.Skipf("Test bundle not found: %s", bundlePath)
	}

	tests := []struct {
		name   string
		format AttestationFormat
	}{
		{
			name:   "SLSA format",
			format: AttestationFormatSLSA,
		},
		{
			name:   "Sigstore format",
			format: AttestationFormatSigstore,
		},
		{
			name:   "InToto format",
			format: AttestationFormatInToto,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewAttestationGenerator(AttestationOptions{
				Format:  tt.format,
				KeyPath: keyPath,
				Metadata: map[string]string{
					"test": "attestation",
				},
			})

			ctx := context.Background()
			attestation, err := gen.GenerateAttestation(ctx, bundlePath)
			if err != nil {
				t.Fatalf("GenerateAttestation failed: %v", err)
			}

			// Verify attestation structure
			if attestation == nil {
				t.Fatal("Attestation is nil")
			}

			// Verify format
			if attestation.Format != tt.format {
				t.Errorf("Format = %v, want %v", attestation.Format, tt.format)
			}

			// Verify subject
			if attestation.Subject.Name != bundlePath {
				t.Errorf("Subject.Name = %v, want %v", attestation.Subject.Name, bundlePath)
			}

			// Verify subject has sha256 digest
			if _, ok := attestation.Subject.Digest["sha256"]; !ok {
				t.Error("Subject.Digest missing sha256")
			}

			// Verify signature
			if attestation.Signature.Signature == "" {
				t.Error("Signature.Signature is empty")
			}
			if attestation.Signature.PublicKey == "" {
				t.Error("Signature.PublicKey is empty")
			}
			if attestation.Signature.SignatureAlgorithm != "ECDSA-SHA256" {
				t.Errorf("SignatureAlgorithm = %v, want ECDSA-SHA256", attestation.Signature.SignatureAlgorithm)
			}

			// Verify predicate type matches format
			expectedPredicateType := ""
			switch tt.format {
			case AttestationFormatSLSA, AttestationFormatSigstore:
				expectedPredicateType = "https://slsa.dev/provenance/v1"
			case AttestationFormatInToto:
				expectedPredicateType = "https://in-toto.io/Statement/v1"
			}
			if attestation.PredicateType != expectedPredicateType {
				t.Errorf("PredicateType = %v, want %v", attestation.PredicateType, expectedPredicateType)
			}

			// Verify timestamp
			if attestation.Timestamp.IsZero() {
				t.Error("Timestamp is zero")
			}

			// Verify metadata
			if attestation.Metadata["test"] != "attestation" {
				t.Error("Metadata not preserved")
			}

			// Verify no Rekor entry (not requested)
			if attestation.RekorEntry != nil {
				t.Error("RekorEntry should be nil when not requested")
			}

			t.Logf("Generated attestation: Format=%s, Signature=%d bytes, PubKey=%d bytes",
				attestation.Format, len(attestation.Signature.Signature), len(attestation.Signature.PublicKey))
		})
	}
}

// TestGenerateAttestationKeylessError tests keyless signing error
func TestGenerateAttestationKeylessError(t *testing.T) {
	gen := NewAttestationGenerator(AttestationOptions{
		Format:     AttestationFormatSLSA,
		UseKeyless: true,
	})

	ctx := context.Background()
	bundlePath := filepath.Join("testdata", "test-bundle.tar")

	_, err := gen.GenerateAttestation(ctx, bundlePath)
	if err == nil {
		t.Error("Expected error for keyless signing, got nil")
	}

	// Error should mention keyless not implemented
	expectedMsg := "keyless signing not yet implemented"
	if err != nil && err.Error() != expectedMsg {
		t.Logf("Got error: %v", err)
	}
}

// TestGenerateAttestationNoKeyError tests error when no key provided
func TestGenerateAttestationNoKeyError(t *testing.T) {
	gen := NewAttestationGenerator(AttestationOptions{
		Format: AttestationFormatSLSA,
		// No KeyPath and UseKeyless=false
	})

	ctx := context.Background()
	bundlePath := filepath.Join("testdata", "test-bundle.tar")

	_, err := gen.GenerateAttestation(ctx, bundlePath)
	if err == nil {
		t.Error("Expected error when no key provided, got nil")
	}

	expectedMsg := "either keyless signing or key path must be provided"
	if err != nil && err.Error() != expectedMsg {
		t.Logf("Got error: %v", err)
	}
}

// TestNewAttestationVerifier tests verifier initialization
func TestNewAttestationVerifier(t *testing.T) {
	tests := []struct {
		name     string
		opts     AttestationVerificationOptions
		wantURL  string
	}{
		{
			name:    "defaults applied",
			opts:    AttestationVerificationOptions{},
			wantURL: "https://rekor.sigstore.dev",
		},
		{
			name: "custom URL preserved",
			opts: AttestationVerificationOptions{
				RekorURL: "https://custom-rekor.example.com",
			},
			wantURL: "https://custom-rekor.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := NewAttestationVerifier(tt.opts)

			if verifier == nil {
				t.Fatal("NewAttestationVerifier returned nil")
			}

			if verifier.opts.RekorURL != tt.wantURL {
				t.Errorf("RekorURL = %v, want %v", verifier.opts.RekorURL, tt.wantURL)
			}
		})
	}
}

// TestVerifyAttestation tests attestation verification
func TestVerifyAttestation(t *testing.T) {
	// First generate an attestation
	keyPath := filepath.Join("testdata", "test-ec-key.pem")
	bundlePath := filepath.Join("testdata", "test-bundle.tar")

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Skipf("Test key not found: %s", keyPath)
	}
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		t.Skipf("Test bundle not found: %s", bundlePath)
	}

	gen := NewAttestationGenerator(AttestationOptions{
		Format:  AttestationFormatSLSA,
		KeyPath: keyPath,
	})

	ctx := context.Background()
	attestation, err := gen.GenerateAttestation(ctx, bundlePath)
	if err != nil {
		t.Fatalf("Failed to generate test attestation: %v", err)
	}

	// Now verify the attestation
	tests := []struct {
		name       string
		opts       AttestationVerificationOptions
		wantErr    bool
		errContains string
	}{
		{
			name: "basic validation succeeds",
			opts: AttestationVerificationOptions{
				MaxAge: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "expired attestation",
			opts: AttestationVerificationOptions{
				MaxAge: 1 * time.Nanosecond, // Instant expiration
			},
			wantErr:     true,
			errContains: "expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh attestation for each test
			testAttestation := *attestation
			if tt.name == "expired attestation" {
				// Modify timestamp to be old
				testAttestation.Timestamp = time.Now().Add(-1 * time.Hour)
			}

			verifier := NewAttestationVerifier(tt.opts)
			err := verifier.VerifyAttestation(ctx, &testAttestation, bundlePath)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if err != nil {
				t.Logf("Verification error: %v", err)
			}
		})
	}
}

// TestVerifyAttestationDigestMismatch tests digest verification
func TestVerifyAttestationDigestMismatch(t *testing.T) {
	// Create attestation with wrong digest
	attestation := &Attestation{
		Format: AttestationFormatSLSA,
		Subject: AttestationSubject{
			Name: "test-bundle.tar",
			Digest: map[string]string{
				"sha256": "wrong_digest_value",
			},
		},
		PredicateType: "https://slsa.dev/provenance/v1",
		Predicate:     map[string]interface{}{},
		Signature: AttestationSignature{
			Signature:          "fake_signature",
			PublicKey:          "fake_public_key",
			SignatureAlgorithm: "ECDSA-SHA256",
		},
		Timestamp: time.Now(),
	}

	verifier := NewAttestationVerifier(AttestationVerificationOptions{})
	bundlePath := filepath.Join("testdata", "test-bundle.tar")

	ctx := context.Background()
	err := verifier.VerifyAttestation(ctx, attestation, bundlePath)

	if err == nil {
		t.Error("Expected digest mismatch error, got nil")
	}

	if err != nil && err.Error() != "" {
		t.Logf("Got expected error: %v", err)
	}
}
