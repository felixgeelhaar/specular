package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPolicyProvenanceGovernance(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	body := `
provenance:
  protocol: enforce
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	pol, err := LoadPolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	if !pol.HasProvenanceGovernance() {
		t.Fatal("expected provenance governance")
	}
	if !pol.Provenance.EnforcesProtocol() {
		t.Fatal("expected protocol enforce")
	}
}

func TestProvenanceGovernanceAdvisoryDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte("provenance:\n  protocol: advisory\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	pol, err := LoadPolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	if pol.HasProvenanceGovernance() {
		t.Fatal("advisory must not enable governance")
	}
}

func TestProvenanceEnforcesProtocolAliases(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"enforce", "Require", "REQUIRED", "true", "yes", "on"} {
		g := &ProvenanceGovernance{Protocol: v}
		if !g.EnforcesProtocol() {
			t.Fatalf("%q should enforce", v)
		}
	}
	for _, v := range []string{"", "advisory", "off", "false", "no"} {
		g := &ProvenanceGovernance{Protocol: v}
		if g.EnforcesProtocol() {
			t.Fatalf("%q should not enforce", v)
		}
	}
}
