package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDescribeGovernanceAdvisory(t *testing.T) {
	t.Parallel()
	p := DescribeGovernance(nil)
	if p.Mode != "advisory" || len(p.Active) != 0 {
		t.Fatalf("%+v", p)
	}
	p = DescribeGovernance(&Policy{})
	if p.Mode != "advisory" {
		t.Fatalf("%+v", p)
	}
}

func TestDescribeGovernanceProgressive(t *testing.T) {
	t.Parallel()
	pol := &Policy{
		Risk: &RiskGovernance{
			Low:    &RiskTier{},
			High:   &RiskTier{},
			Medium: &RiskTier{},
		},
		Provenance: &ProvenanceGovernance{
			Attested: "enforce",
			Protocol: "require",
		},
	}
	p := DescribeGovernance(pol)
	if p.Mode != "progressive" {
		t.Fatalf("mode=%s", p.Mode)
	}
	if !p.RiskTiers || !p.ProvenanceAttested || !p.ProvenanceProtocol {
		t.Fatalf("%+v", p)
	}
	wantLevels := []string{"low", "medium", "high"}
	if strings.Join(p.RiskLevels, ",") != strings.Join(wantLevels, ",") {
		t.Fatalf("levels=%v", p.RiskLevels)
	}
	wantActive := []string{"risk.tiers", "provenance.attested", "provenance.protocol"}
	if strings.Join(p.Active, ",") != strings.Join(wantActive, ",") {
		t.Fatalf("active=%v", p.Active)
	}
}

func TestLoadGovernancePosture(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	empty, err := LoadGovernancePosture(root)
	if err != nil || empty.Mode != "advisory" {
		t.Fatalf("%+v err=%v", empty, err)
	}

	dir := filepath.Join(root, ".specular")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	body := `
risk:
  critical:
    approvals: [security]
provenance:
  attested: enforce
`
	if err := os.WriteFile(filepath.Join(dir, "policy.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := LoadGovernancePosture(root)
	if err != nil {
		t.Fatal(err)
	}
	if p.Mode != "progressive" || !p.RiskTiers || !p.ProvenanceAttested || p.ProvenanceProtocol {
		t.Fatalf("%+v", p)
	}
	if p.PolicyPath == "" {
		t.Fatal("expected policy path")
	}
}

func TestFormatProgressiveTrust(t *testing.T) {
	t.Parallel()
	advisory := FormatProgressiveTrust(GovernancePosture{Mode: "advisory"})
	if !strings.Contains(advisory, "advisory") ||
		!strings.Contains(advisory, "examples/policy/progressive-trust.yaml") {
		t.Fatalf("%s", advisory)
	}
	prog := FormatProgressiveTrust(GovernancePosture{
		Mode:               "progressive",
		RiskTiers:          true,
		RiskLevels:         []string{"high", "critical"},
		ProvenanceAttested: true,
		Active:             []string{"risk.tiers", "provenance.attested"},
	})
	for _, want := range []string{
		"Mode: progressive (2 knobs active)",
		"risk.tiers: high, critical",
		"provenance.attested: enforce",
	} {
		if !strings.Contains(prog, want) {
			t.Fatalf("missing %q in:\n%s", want, prog)
		}
	}
}
