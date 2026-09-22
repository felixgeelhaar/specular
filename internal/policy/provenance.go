package policy

import "strings"

// ProvenanceGovernance is opt-in Agent Provenance Protocol enforcement
// (PRODUCT_INTENT §9 / P1 #3+#10). When unset, gate Provenance stays
// advisory (counts APP docs but never flips ALLOW→DENY).
type ProvenanceGovernance struct {
	// Protocol is "enforce" / "require" / "required" to DENY when attested
	// sessions lack valid sibling .provenance.json documents.
	Protocol string `yaml:"protocol,omitempty"`
}

// HasProvenanceGovernance reports whether protocol enforcement is configured.
func (p *Policy) HasProvenanceGovernance() bool {
	return p != nil && p.Provenance != nil && p.Provenance.EnforcesProtocol()
}

// EnforcesProtocol reports whether APP protocol docs must validate.
func (g *ProvenanceGovernance) EnforcesProtocol() bool {
	if g == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(g.Protocol)) {
	case "enforce", "require", "required", "true", "yes", "on":
		return true
	default:
		return false
	}
}
