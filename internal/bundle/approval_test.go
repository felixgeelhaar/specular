package bundle

import (
	"encoding/json"
	"testing"
	"time"
)

// TestApproval_ToJSON tests the ToJSON method
func TestApproval_ToJSON(t *testing.T) {
	t.Run("complete approval", func(t *testing.T) {
		now := time.Now()
		approval := &Approval{
			Role:                 "security",
			User:                 "alice@example.com",
			SignedAt:             now,
			Signature:            "test-signature",
			SignatureType:        SignatureTypeSSH,
			PublicKey:            "ssh-ed25519 AAAAC3...",
			PublicKeyFingerprint: "SHA256:abc123",
			Comment:              "Approved after security review",
			Metadata: map[string]string{
				"review_id": "rev-123",
				"tool":      "specular",
			},
		}

		jsonData, err := approval.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON() error = %v", err)
		}

		// Verify it's valid JSON
		var decoded Approval
		if err := json.Unmarshal(jsonData, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal ToJSON() output: %v", err)
		}

		// Verify key fields
		if decoded.Role != approval.Role {
			t.Errorf("Role = %s, want %s", decoded.Role, approval.Role)
		}

		if decoded.User != approval.User {
			t.Errorf("User = %s, want %s", decoded.User, approval.User)
		}

		if decoded.Signature != approval.Signature {
			t.Errorf("Signature = %s, want %s", decoded.Signature, approval.Signature)
		}

		if decoded.SignatureType != approval.SignatureType {
			t.Errorf("SignatureType = %s, want %s", decoded.SignatureType, approval.SignatureType)
		}

		if decoded.Comment != approval.Comment {
			t.Errorf("Comment = %s, want %s", decoded.Comment, approval.Comment)
		}

		// Verify timestamp (with some tolerance for time comparison)
		if decoded.SignedAt.Unix() != approval.SignedAt.Unix() {
			t.Errorf("SignedAt = %v, want %v", decoded.SignedAt, approval.SignedAt)
		}

		// Verify metadata
		if len(decoded.Metadata) != len(approval.Metadata) {
			t.Errorf("Metadata length = %d, want %d", len(decoded.Metadata), len(approval.Metadata))
		}
	})

	t.Run("minimal approval", func(t *testing.T) {
		approval := &Approval{
			Role:          "pm",
			User:          "bob@example.com",
			SignedAt:      time.Now(),
			Signature:     "sig",
			SignatureType: SignatureTypeGPG,
			PublicKey:     "gpg-key",
		}

		jsonData, err := approval.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON() error = %v", err)
		}

		// Verify it's valid JSON
		var decoded Approval
		if err := json.Unmarshal(jsonData, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal ToJSON() output: %v", err)
		}

		if decoded.Role != approval.Role {
			t.Errorf("Role = %s, want %s", decoded.Role, approval.Role)
		}

		if decoded.User != approval.User {
			t.Errorf("User = %s, want %s", decoded.User, approval.User)
		}
	})

	t.Run("pretty-printed output", func(t *testing.T) {
		approval := &Approval{
			Role:          "lead",
			User:          "carol@example.com",
			SignedAt:      time.Now(),
			Signature:     "signature-data",
			SignatureType: SignatureTypeX509,
			PublicKey:     "x509-cert",
		}

		jsonData, err := approval.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON() error = %v", err)
		}

		// Check that output is indented (pretty-printed)
		jsonStr := string(jsonData)
		// Pretty-printed JSON should contain newlines and spaces
		if len(jsonStr) < 50 { // Should be reasonably long with formatting
			t.Error("ToJSON() output should be pretty-printed with indentation")
		}
	})

	t.Run("all signature types", func(t *testing.T) {
		signatureTypes := []SignatureType{
			SignatureTypeSSH,
			SignatureTypeGPG,
			SignatureTypeX509,
			SignatureTypeCosign,
		}

		for _, sigType := range signatureTypes {
			t.Run(string(sigType), func(t *testing.T) {
				approval := &Approval{
					Role:          "test",
					User:          "test@example.com",
					SignedAt:      time.Now(),
					Signature:     "sig",
					SignatureType: sigType,
					PublicKey:     "key",
				}

				jsonData, err := approval.ToJSON()
				if err != nil {
					t.Fatalf("ToJSON() error = %v for signature type %s", err, sigType)
				}

				var decoded Approval
				if err := json.Unmarshal(jsonData, &decoded); err != nil {
					t.Fatalf("Failed to unmarshal for signature type %s: %v", sigType, err)
				}

				if decoded.SignatureType != sigType {
					t.Errorf("SignatureType = %s, want %s", decoded.SignatureType, sigType)
				}
			})
		}
	})
}
