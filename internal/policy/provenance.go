package policy

import "strings"

// ProvenanceGovernance is opt-in provenance enforcement (PRODUCT_INTENT §9 /
// P1 #3+#10). When unset, gate Provenance stays advisory (counts APP docs /
// reports unattested but never flips ALLOW→DENY).
type ProvenanceGovernance struct {
	// Attested is "enforce" / "require" to DENY when no session attestations
	// are present (progressive trust beyond Level 0–1).
	Attested string `yaml:"attested,omitempty"`
	// Protocol is "enforce" / "require" / "required" to DENY when attested
	// sessions lack valid sibling .provenance.json documents.
	Protocol string `yaml:"protocol,omitempty"`
	// Governed is "enforce" / "require" to DENY when attested sessions lack
	// provenance.governed=true (safer native launch). Idle when unattested.
	Governed string `yaml:"governed,omitempty"`
}

// HasProvenanceGovernance reports whether any provenance enforce knob is on.
func (p *Policy) HasProvenanceGovernance() bool {
	return p != nil && p.Provenance != nil &&
		(p.Provenance.EnforcesAttested() || p.Provenance.EnforcesProtocol() ||
			p.Provenance.EnforcesGoverned())
}

// EnforcesAttested reports whether unattested trees must DENY.
func (g *ProvenanceGovernance) EnforcesAttested() bool {
	if g == nil {
		return false
	}
	return isEnforce(g.Attested)
}

// EnforcesProtocol reports whether APP protocol docs must validate.
func (g *ProvenanceGovernance) EnforcesProtocol() bool {
	if g == nil {
		return false
	}
	return isEnforce(g.Protocol)
}

// EnforcesGoverned reports whether at least one governed session is required.
func (g *ProvenanceGovernance) EnforcesGoverned() bool {
	if g == nil {
		return false
	}
	return isEnforce(g.Governed)
}

func isEnforce(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "enforce", "require", "required", "true", "yes", "on":
		return true
	default:
		return false
	}
}
