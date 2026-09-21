package attestation

import (
	"encoding/json"
	"strings"
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
	if att.Provenance.Governed {
		t.Fatal("expected governed=false by default")
	}
	if att.Provenance.SpecularVersion != "9.9.9" {
		t.Fatalf("version=%s", att.Provenance.SpecularVersion)
	}
	if att.GoalDigest == "" || !strings.HasPrefix(att.GoalDigest, "sha256:") {
		t.Fatalf("goalDigest=%q", att.GoalDigest)
	}
	if err := NewStandardVerifier().Verify(att); err != nil {
		t.Fatalf("verify: %v", err)
	}

	gov, err := gen.GenerateFromSession(SessionInput{
		ID: "g", Goal: "x", Harness: "claude-code", Status: "completed",
		CreatedAt: now, UpdatedAt: now, Governed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !gov.Provenance.Governed {
		t.Fatal("expected governed=true in provenance")
	}
}

func TestGenerateFromSessionRedactsSecrets(t *testing.T) {
	t.Parallel()
	signer, err := NewEphemeralSigner("auditor@example.com")
	if err != nil {
		t.Fatal(err)
	}
	gen := NewGenerator(signer, "1.0.0")
	secret := "ghp_abcdefghijklmnopqrstuvwxyz0123456789AB"
	goal := "ship fix using " + secret
	att, err := gen.GenerateFromSession(SessionInput{
		ID: "sec", Goal: goal, Harness: "claude-code", Status: "completed",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(att.Goal, "ghp_") {
		t.Fatalf("secret leaked in goal: %q", att.Goal)
	}
	if !strings.Contains(att.Goal, "[REDACTED]") {
		t.Fatalf("expected redaction: %q", att.Goal)
	}
	raw, err := json.Marshal(att)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secret) {
		t.Fatalf("secret leaked in attestation JSON")
	}
	if att.GoalDigest == "" {
		t.Fatal("missing goalDigest")
	}
}
