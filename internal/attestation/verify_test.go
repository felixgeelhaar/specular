package attestation

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/auto"
)

func TestVerifyValidAttestation(t *testing.T) {
	// Create a valid attestation
	signer, err := NewEphemeralSigner("test@example.com")
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	generator := NewGenerator(signer, "1.0.0")

	now := time.Now()
	result := &auto.Result{
		Duration:      time.Hour,
		TotalCost:     1.0,
		TasksExecuted: 5,
		TasksFailed:   0,
		AutoOutput: &auto.AutoOutput{
			Goal:   "test goal",
			Status: "completed",
			Audit: auto.AuditTrail{
				CheckpointID: "test-workflow",
				StartedAt:    now.Add(-1 * time.Hour),
				CompletedAt:  now,
			},
		},
	}

	config := &auto.Config{
		Goal: "test goal",
	}

	planJSON := []byte(`{"plan": "test"}`)
	outputJSON := []byte(`{"output": "test"}`)

	att, err := generator.Generate(result, config, planJSON, outputJSON)
	if err != nil {
		t.Fatalf("Failed to generate attestation: %v", err)
	}

	// Verify the attestation
	verifier := NewStandardVerifier()
	if err := verifier.Verify(att); err != nil {
		t.Errorf("Verification failed: %v", err)
	}
}

func TestVerifyProvenanceValid(t *testing.T) {
	att := &Attestation{
		Status:    "success",
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
		Provenance: Provenance{
			Hostname:        "test-host",
			Platform:        "linux",
			Arch:            "amd64",
			SpecularVersion: "1.0.0",
			TotalCost:       1.0,
			TasksExecuted:   5,
			TasksFailed:     0,
			GitDirty:        false,
		},
	}

	verifier := NewStandardVerifier()
	if err := verifier.VerifyProvenance(att); err != nil {
		t.Errorf("Provenance verification failed: %v", err)
	}
}

func TestVerifyProvenanceInvalid(t *testing.T) {
	tests := []struct {
		name string
		att  *Attestation
	}{
		{
			name: "missing hostname",
			att: &Attestation{
				Status:    "success",
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now(),
				Provenance: Provenance{
					Platform:        "linux",
					Arch:            "amd64",
					SpecularVersion: "1.0.0",
				},
			},
		},
		{
			name: "missing platform",
			att: &Attestation{
				Status:    "success",
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now(),
				Provenance: Provenance{
					Hostname:        "test",
					Arch:            "amd64",
					SpecularVersion: "1.0.0",
				},
			},
		},
		{
			name: "negative cost",
			att: &Attestation{
				Status:    "success",
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now(),
				Provenance: Provenance{
					Hostname:        "test",
					Platform:        "linux",
					Arch:            "amd64",
					SpecularVersion: "1.0.0",
					TotalCost:       -1.0,
				},
			},
		},
		{
			name: "end before start",
			att: &Attestation{
				Status:    "success",
				StartTime: time.Now(),
				EndTime:   time.Now().Add(-1 * time.Hour),
				Provenance: Provenance{
					Hostname:        "test",
					Platform:        "linux",
					Arch:            "amd64",
					SpecularVersion: "1.0.0",
				},
			},
		},
	}

	verifier := NewStandardVerifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := verifier.VerifyProvenance(tt.att); err == nil {
				t.Error("Expected verification to fail, but it passed")
			}
		})
	}
}

func TestVerifyHashesValid(t *testing.T) {
	planJSON := []byte(`{"plan": "test"}`)
	outputJSON := []byte(`{"output": "test"}`)

	att := &Attestation{
		PlanHash:   hashData(planJSON),
		OutputHash: hashData(outputJSON),
	}

	verifier := NewStandardVerifier()
	if err := verifier.VerifyHashes(att, planJSON, outputJSON); err != nil {
		t.Errorf("Hash verification failed: %v", err)
	}
}

func TestVerifyHashesInvalid(t *testing.T) {
	planJSON := []byte(`{"plan": "test"}`)
	outputJSON := []byte(`{"output": "test"}`)
	wrongJSON := []byte(`{"wrong": "data"}`)

	att := &Attestation{
		PlanHash:   hashData(planJSON),
		OutputHash: hashData(outputJSON),
	}

	verifier := NewStandardVerifier()

	// Wrong plan
	if err := verifier.VerifyHashes(att, wrongJSON, outputJSON); err == nil {
		t.Error("Expected hash verification to fail for wrong plan")
	}

	// Wrong output
	if err := verifier.VerifyHashes(att, planJSON, wrongJSON); err == nil {
		t.Error("Expected hash verification to fail for wrong output")
	}
}

func TestVerifyMaxAge(t *testing.T) {
	// Create attestation signed 2 hours ago
	att := &Attestation{
		SignedAt:  time.Now().Add(-2 * time.Hour),
		Signature: "sig",
		PublicKey: "key",
	}

	// Verifier with 1 hour max age
	verifier := NewStandardVerifier(WithMaxAge(1 * time.Hour))
	if err := verifier.Verify(att); err == nil {
		t.Error("Expected verification to fail for old attestation")
	}

	// Verifier with 3 hour max age should pass signature check (but fail other checks)
	verifier2 := NewStandardVerifier(WithMaxAge(3 * time.Hour))
	err := verifier2.Verify(att)
	// Will fail because signature is invalid, but not because of age
	if err != nil && err.Error() == "attestation too old" {
		t.Error("Should not fail due to age with 3 hour max")
	}
}

func TestVerifyAllowedIdentities(t *testing.T) {
	att := &Attestation{
		SignedBy:  "test@example.com",
		SignedAt:  time.Now(),
		Signature: "sig",
		PublicKey: "key",
	}

	// Allowed identity
	verifier := NewStandardVerifier(WithAllowedIdentities([]string{"test@example.com"}))
	err := verifier.Verify(att)
	// Will fail for other reasons, but not identity
	if err != nil && err.Error() == "signer identity not allowed: test@example.com" {
		t.Error("Should not fail due to allowed identity")
	}

	// Disallowed identity
	verifier2 := NewStandardVerifier(WithAllowedIdentities([]string{"other@example.com"}))
	err = verifier2.Verify(att)
	if err == nil || err.Error() != "signer identity not allowed: test@example.com" {
		t.Error("Should fail due to disallowed identity")
	}
}

func TestVerifyRequireGitClean(t *testing.T) {
	tests := []struct {
		name       string
		gitDirty   bool
		shouldFail bool
	}{
		{"clean git", false, false},
		{"dirty git", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			att := &Attestation{
				Status:    "success",
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now(),
				Provenance: Provenance{
					Hostname:        "test",
					Platform:        "linux",
					Arch:            "amd64",
					SpecularVersion: "1.0.0",
					GitDirty:        tt.gitDirty,
				},
			}

			verifier := NewStandardVerifier(WithRequireGitClean(true))
			err := verifier.VerifyProvenance(att)

			if tt.shouldFail && err == nil {
				t.Error("Expected verification to fail for dirty git")
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("Expected verification to pass for clean git, got: %v", err)
			}
		})
	}
}

func TestVerifyMissingSignature(t *testing.T) {
	att := &Attestation{
		SignedAt:  time.Now(),
		Signature: "", // Missing signature
		PublicKey: "key",
	}

	verifier := NewStandardVerifier()
	err := verifier.Verify(att)
	if err == nil {
		t.Error("Expected verification to fail for missing signature")
	}
}
