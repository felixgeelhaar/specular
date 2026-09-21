package security

import (
	"strings"
	"testing"
)

func TestRedactSecretsGitHubToken(t *testing.T) {
	t.Parallel()
	raw := "deploy with token ghp_abcdefghijklmnopqrstuvwxyz0123456789AB and finish"
	got := RedactSecrets(raw)
	if strings.Contains(got, "ghp_") {
		t.Fatalf("token leaked: %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("expected placeholder: %q", got)
	}
	if !ContainsSecret(raw) {
		t.Fatal("expected ContainsSecret")
	}
}

func TestRedactSecretsAWSKey(t *testing.T) {
	t.Parallel()
	raw := "use key AKIAIOSFODNN7EXAMPLE for s3"
	got := RedactSecrets(raw)
	if strings.Contains(got, "AKIA") {
		t.Fatalf("leaked: %q", got)
	}
}

func TestRedactSecretsCleanGoal(t *testing.T) {
	t.Parallel()
	raw := "Harden JWT validation in auth middleware"
	if RedactSecrets(raw) != raw {
		t.Fatalf("unexpected redact: %q", RedactSecrets(raw))
	}
	if ContainsSecret(raw) {
		t.Fatal("false positive")
	}
}

func TestGoalDigestStable(t *testing.T) {
	t.Parallel()
	goal := "secret goal with ghp_abcdefghijklmnopqrstuvwxyz0123456789AB"
	d1 := GoalDigest(goal)
	d2 := GoalDigest(goal)
	if d1 != d2 || !strings.HasPrefix(d1, "sha256:") {
		t.Fatalf("%s vs %s", d1, d2)
	}
	redacted, digest := SafeGoal(goal)
	if digest != d1 {
		t.Fatalf("digest mismatch")
	}
	if strings.Contains(redacted, "ghp_") {
		t.Fatalf("leaked: %q", redacted)
	}
}
