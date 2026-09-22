package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPolicyRiskGovernance(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	body := `
risk:
  low:
    approval: none
  medium:
    approval:
      - code-owner
  high:
    approvals:
      - security
      - service-owner
  critical:
    autonomous_execution: false
    approvals:
      - security
      - platform
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	pol, err := LoadPolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	if !pol.HasRiskGovernance() {
		t.Fatal("expected risk governance")
	}
	if got := pol.Risk.Low.RequiredApprovals(); len(got) != 0 {
		t.Fatalf("low approvals=%v", got)
	}
	if got := pol.Risk.Medium.RequiredApprovals(); len(got) != 1 || got[0] != "code-owner" {
		t.Fatalf("medium=%v", got)
	}
	if got := pol.Risk.High.RequiredApprovals(); len(got) != 2 {
		t.Fatalf("high=%v", got)
	}
	crit := pol.Risk.TierFor("CRITICAL")
	if crit == nil || !crit.BlocksAutonomous() {
		t.Fatalf("critical=%v", crit)
	}
	if got := crit.RequiredApprovals(); len(got) != 2 || got[0] != "security" {
		t.Fatalf("critical approvals=%v", got)
	}
	// CRITICAL falls back to HIGH when critical unset
	pol.Risk.Critical = nil
	if pol.Risk.TierFor("CRITICAL") != pol.Risk.High {
		t.Fatal("expected HIGH fallback for CRITICAL")
	}
}

func TestStringListUnmarshal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte("risk:\n  low:\n    approval: none\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	pol, err := LoadPolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := pol.Risk.Low.RequiredApprovals(); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}
