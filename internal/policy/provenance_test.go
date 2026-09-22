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
		g2 := &ProvenanceGovernance{Attested: v}
		if !g2.EnforcesAttested() {
			t.Fatalf("attested %q should enforce", v)
		}
		g3 := &ProvenanceGovernance{Governed: v}
		if !g3.EnforcesGoverned() {
			t.Fatalf("governed %q should enforce", v)
		}
	}
	for _, v := range []string{"", "advisory", "off", "false", "no"} {
		g := &ProvenanceGovernance{Protocol: v, Attested: v, Governed: v}
		if g.EnforcesProtocol() || g.EnforcesAttested() || g.EnforcesGoverned() {
			t.Fatalf("%q should not enforce", v)
		}
	}
}

func TestLoadPolicyProvenanceAttestedEnforce(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte("provenance:\n  attested: enforce\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	pol, err := LoadPolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	if !pol.HasProvenanceGovernance() || !pol.Provenance.EnforcesAttested() {
		t.Fatal("expected attested enforce")
	}
	if pol.Provenance.EnforcesProtocol() {
		t.Fatal("protocol should stay advisory")
	}
}
