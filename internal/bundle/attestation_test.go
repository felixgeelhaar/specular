package bundle

import (
	"testing"
	"time"
)

// TestAttestation_IsExpired tests the IsExpired method
func TestAttestation_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		timestamp time.Time
		maxAge    time.Duration
		want      bool
	}{
		{
			name:      "not expired - recent attestation",
			timestamp: time.Now().Add(-1 * time.Hour),
			maxAge:    24 * time.Hour,
			want:      false,
		},
		{
			name:      "expired - old attestation",
			timestamp: time.Now().Add(-48 * time.Hour),
			maxAge:    24 * time.Hour,
			want:      true,
		},
		{
			name:      "no expiration - maxAge zero",
			timestamp: time.Now().Add(-1000 * time.Hour),
			maxAge:    0,
			want:      false,
		},
		{
			name:      "just expired - at boundary",
			timestamp: time.Now().Add(-24*time.Hour - 1*time.Second),
			maxAge:    24 * time.Hour,
			want:      true,
		},
		{
			name:      "not expired - just created",
			timestamp: time.Now(),
			maxAge:    1 * time.Hour,
			want:      false,
		},
		{
			name:      "expired - way past maxAge",
			timestamp: time.Now().Add(-100 * time.Hour),
			maxAge:    1 * time.Hour,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attestation := &Attestation{
				Timestamp: tt.timestamp,
			}

			got := attestation.IsExpired(tt.maxAge)
			if got != tt.want {
				t.Errorf("IsExpired(%v) = %v, want %v", tt.maxAge, got, tt.want)
			}
		})
	}
}

// TestAttestation_HasRekorEntry tests the HasRekorEntry method
func TestAttestation_HasRekorEntry(t *testing.T) {
	tests := []struct {
		name        string
		rekorEntry  *RekorEntry
		want        bool
		description string
	}{
		{
			name: "has valid rekor entry",
			rekorEntry: &RekorEntry{
				UUID:           "rekor-uuid-12345",
				LogIndex:       12345,
				IntegratedTime: time.Now().Unix(),
			},
			want:        true,
			description: "Should return true when RekorEntry exists with valid UUID",
		},
		{
			name:        "no rekor entry",
			rekorEntry:  nil,
			want:        false,
			description: "Should return false when RekorEntry is nil",
		},
		{
			name: "rekor entry with empty UUID",
			rekorEntry: &RekorEntry{
				UUID:           "",
				LogIndex:       12345,
				IntegratedTime: time.Now().Unix(),
			},
			want:        false,
			description: "Should return false when UUID is empty",
		},
		{
			name: "rekor entry with only UUID",
			rekorEntry: &RekorEntry{
				UUID: "minimal-uuid",
			},
			want:        true,
			description: "Should return true when UUID is present (other fields optional)",
		},
		{
			name: "rekor entry with all fields",
			rekorEntry: &RekorEntry{
				UUID:           "full-uuid",
				LogIndex:       9999,
				IntegratedTime: time.Now().Unix(),
				InclusionProof: "proof-data",
				Body:           "body-data",
			},
			want:        true,
			description: "Should return true when all fields are present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attestation := &Attestation{
				RekorEntry: tt.rekorEntry,
			}

			got := attestation.HasRekorEntry()
			if got != tt.want {
				t.Errorf("HasRekorEntry() = %v, want %v - %s", got, tt.want, tt.description)
			}
		})
	}
}

// TestAttestation_Validate tests the Validate method (already tested but at 69.2%)
// Additional edge cases for better coverage
func TestAttestation_Validate_AdditionalCases(t *testing.T) {
	t.Run("valid complete attestation", func(t *testing.T) {
		attestation := &Attestation{
			Format: AttestationFormatSigstore,
			Subject: AttestationSubject{
				Name: "test-bundle",
				Digest: map[string]string{
					"sha256": "abc123",
				},
			},
			PredicateType: "https://slsa.dev/provenance/v1",
			Signature: AttestationSignature{
				Signature: "signature-data",
			},
			Timestamp: time.Now(),
		}

		err := attestation.Validate()
		if err != nil {
			t.Errorf("Validate() error = %v, want nil for valid attestation", err)
		}
	})

	t.Run("missing format", func(t *testing.T) {
		attestation := &Attestation{
			Format: "",
			Subject: AttestationSubject{
				Name: "test",
				Digest: map[string]string{
					"sha256": "abc123",
				},
			},
			PredicateType: "https://slsa.dev/provenance/v1",
			Signature: AttestationSignature{
				Signature: "sig",
			},
			Timestamp: time.Now(),
		}

		err := attestation.Validate()
		if err == nil {
			t.Error("Validate() should return error for missing format")
		}

		valErr, ok := err.(*ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
		} else if valErr.Field != "format" {
			t.Errorf("ValidationError.Field = %s, want format", valErr.Field)
		}
	})

	t.Run("missing subject name", func(t *testing.T) {
		attestation := &Attestation{
			Format: AttestationFormatSLSA,
			Subject: AttestationSubject{
				Name: "",
				Digest: map[string]string{
					"sha256": "abc123",
				},
			},
			PredicateType: "https://slsa.dev/provenance/v1",
			Signature: AttestationSignature{
				Signature: "sig",
			},
			Timestamp: time.Now(),
		}

		err := attestation.Validate()
		if err == nil {
			t.Error("Validate() should return error for missing subject name")
		}

		valErr, ok := err.(*ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
		} else if valErr.Field != "subject.name" {
			t.Errorf("ValidationError.Field = %s, want subject.name", valErr.Field)
		}
	})

	t.Run("empty subject digest", func(t *testing.T) {
		attestation := &Attestation{
			Format: AttestationFormatInToto,
			Subject: AttestationSubject{
				Name:   "test",
				Digest: map[string]string{},
			},
			PredicateType: "https://in-toto.io/Statement/v1",
			Signature: AttestationSignature{
				Signature: "sig",
			},
			Timestamp: time.Now(),
		}

		err := attestation.Validate()
		if err == nil {
			t.Error("Validate() should return error for empty digest")
		}

		valErr, ok := err.(*ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
		} else if valErr.Field != "subject.digest" {
			t.Errorf("ValidationError.Field = %s, want subject.digest", valErr.Field)
		}
	})

	t.Run("missing predicate type", func(t *testing.T) {
		attestation := &Attestation{
			Format: AttestationFormatCustom,
			Subject: AttestationSubject{
				Name: "test",
				Digest: map[string]string{
					"sha256": "abc123",
				},
			},
			PredicateType: "",
			Signature: AttestationSignature{
				Signature: "sig",
			},
			Timestamp: time.Now(),
		}

		err := attestation.Validate()
		if err == nil {
			t.Error("Validate() should return error for missing predicate type")
		}

		valErr, ok := err.(*ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
		} else if valErr.Field != "predicate_type" {
			t.Errorf("ValidationError.Field = %s, want predicate_type", valErr.Field)
		}
	})

	t.Run("missing signature", func(t *testing.T) {
		attestation := &Attestation{
			Format: AttestationFormatSigstore,
			Subject: AttestationSubject{
				Name: "test",
				Digest: map[string]string{
					"sha256": "abc123",
				},
			},
			PredicateType: "https://slsa.dev/provenance/v1",
			Signature: AttestationSignature{
				Signature: "",
			},
			Timestamp: time.Now(),
		}

		err := attestation.Validate()
		if err == nil {
			t.Error("Validate() should return error for missing signature")
		}

		valErr, ok := err.(*ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
		} else if valErr.Field != "signature.signature" {
			t.Errorf("ValidationError.Field = %s, want signature.signature", valErr.Field)
		}
	})

	t.Run("zero timestamp", func(t *testing.T) {
		attestation := &Attestation{
			Format: AttestationFormatSigstore,
			Subject: AttestationSubject{
				Name: "test",
				Digest: map[string]string{
					"sha256": "abc123",
				},
			},
			PredicateType: "https://slsa.dev/provenance/v1",
			Signature: AttestationSignature{
				Signature: "sig",
			},
			Timestamp: time.Time{},
		}

		err := attestation.Validate()
		if err == nil {
			t.Error("Validate() should return error for zero timestamp")
		}

		valErr, ok := err.(*ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
		} else if valErr.Field != "timestamp" {
			t.Errorf("ValidationError.Field = %s, want timestamp", valErr.Field)
		}
	})
}
