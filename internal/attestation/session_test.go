package attestation

import (
	"testing"
	"time"
)

func TestGenerateFromSession(t *testing.T) {
	t.Parallel()
	signer, err := NewEphemeralSigner("auditor@example.com")
	if err != nil {
		t.Fatal(err)
	}
	gen := NewGenerator(signer, "9.9.9")
	now := time.Now().UTC()
	att, err := gen.GenerateFromSession(SessionInput{
		ID:             "auth",
		Goal:           "Harden JWT",
		Harness:        "codex",
		Profile:        "ci",
		Status:         "completed",
		CreatedAt:      now.Add(-time.Minute),
		UpdatedAt:      now,
		WorktreePath:   "/tmp/wt/auth",
		WorktreeBranch: "specular/auth",
		WorktreeName:   "auth",
	})
	if err != nil {
		t.Fatal(err)
	}
	if att.WorkflowID != "session-auth" {
		t.Fatalf("workflow=%s", att.WorkflowID)
	}
	if att.Status != "success" {
		t.Fatalf("status=%s", att.Status)
	}
	if att.Provenance.Harness != "codex" {
		t.Fatalf("harness=%s", att.Provenance.Harness)
	}
	if att.Provenance.WorktreePath != "/tmp/wt/auth" || att.Provenance.WorktreeBranch != "specular/auth" {
		t.Fatalf("%+v", att.Provenance)
	}
	if att.Provenance.SpecularVersion != "9.9.9" {
		t.Fatalf("version=%s", att.Provenance.SpecularVersion)
	}
	if err := NewStandardVerifier().Verify(att); err != nil {
		t.Fatalf("verify: %v", err)
	}
}
