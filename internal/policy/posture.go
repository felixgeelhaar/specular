package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GovernancePosture summarizes which opt-in progressive-trust knobs are
// configured in a policy file (PRODUCT_INTENT §12–§13 / §9). Informational —
// does not enforce anything by itself.
type GovernancePosture struct {
	Mode               string   `json:"mode"` // advisory | progressive
	RiskTiers          bool     `json:"risk_tiers"`
	RiskLevels         []string `json:"risk_levels,omitempty"`
	ProvenanceAttested bool     `json:"provenance_attested"`
	ProvenanceProtocol bool     `json:"provenance_protocol"`
	ProvenanceGoverned bool     `json:"provenance_governed"`
	Active             []string `json:"active,omitempty"`
	PolicyPath         string   `json:"policy_path,omitempty"`
}

// DescribeGovernance reports active risk / provenance enforce knobs.
func DescribeGovernance(p *Policy) GovernancePosture {
	out := GovernancePosture{Mode: "advisory"}
	if p == nil {
		return out
	}
	if p.HasRiskGovernance() {
		out.RiskTiers = true
		out.RiskLevels = configuredRiskLevels(p.Risk)
		out.Active = append(out.Active, "risk.tiers")
	}
	if p.Provenance != nil && p.Provenance.EnforcesAttested() {
		out.ProvenanceAttested = true
		out.Active = append(out.Active, "provenance.attested")
	}
	if p.Provenance != nil && p.Provenance.EnforcesProtocol() {
		out.ProvenanceProtocol = true
		out.Active = append(out.Active, "provenance.protocol")
	}
	if p.Provenance != nil && p.Provenance.EnforcesGoverned() {
		out.ProvenanceGoverned = true
		out.Active = append(out.Active, "provenance.governed")
	}
	if len(out.Active) > 0 {
		out.Mode = "progressive"
	}
	return out
}

func configuredRiskLevels(r *RiskGovernance) []string {
	if r == nil {
		return nil
	}
	var out []string
	if r.Low != nil {
		out = append(out, "low")
	}
	if r.Medium != nil {
		out = append(out, "medium")
	}
	if r.High != nil {
		out = append(out, "high")
	}
	if r.Critical != nil {
		out = append(out, "critical")
	}
	return out
}

// ResolvePolicyPath finds .specular/policy.yaml or policies.yaml under root.
// Empty root uses paths relative to the current working directory.
func ResolvePolicyPath(root string) string {
	candidates := []string{
		filepath.Join(root, ".specular", "policy.yaml"),
		filepath.Join(root, ".specular", "policies.yaml"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// LoadGovernancePosture loads policy from root (if present) and describes knobs.
func LoadGovernancePosture(root string) (GovernancePosture, error) {
	path := ResolvePolicyPath(root)
	if path == "" {
		return GovernancePosture{Mode: "advisory"}, nil
	}
	pol, err := LoadPolicy(path)
	if err != nil {
		return GovernancePosture{Mode: "advisory", PolicyPath: path}, err
	}
	posture := DescribeGovernance(pol)
	posture.PolicyPath = path
	return posture, nil
}

// FormatProgressiveTrust renders a short human block for doctor text output.
func FormatProgressiveTrust(p GovernancePosture) string {
	var b strings.Builder
	b.WriteString("Progressive trust:\n")
	if p.Mode != "progressive" {
		b.WriteString("  ○ Mode: advisory (no opt-in knobs)\n")
		b.WriteString("      enable via policy risk: / provenance.attested|protocol|governed: enforce\n")
		return b.String()
	}
	b.WriteString("  ✓ Mode: progressive")
	if n := len(p.Active); n > 0 {
		fmt.Fprintf(&b, " (%d knob", n)
		if n != 1 {
			b.WriteString("s")
		}
		b.WriteString(" active)")
	}
	b.WriteString("\n")
	if p.RiskTiers {
		b.WriteString("  ✓ risk.tiers: ")
		if len(p.RiskLevels) > 0 {
			b.WriteString(strings.Join(p.RiskLevels, ", "))
		} else {
			b.WriteString("configured")
		}
		b.WriteString("\n")
	}
	if p.ProvenanceAttested {
		b.WriteString("  ✓ provenance.attested: enforce\n")
	}
	if p.ProvenanceProtocol {
		b.WriteString("  ✓ provenance.protocol: enforce\n")
	}
	if p.ProvenanceGoverned {
		b.WriteString("  ✓ provenance.governed: enforce\n")
	}
	return b.String()
}
