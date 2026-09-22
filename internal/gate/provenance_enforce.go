package gate

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/specular/internal/policy"
)

// applyProvenanceGovernance overlays opt-in policy provenance knobs onto
// res.Provenance. No-op when policy is missing or all knobs are advisory.
//
//   - attested: enforce → DENY when no session attestations
//   - protocol: enforce → DENY when attested sessions lack valid APP docs
//
// Protocol-only enforce stays idle (advisory) on unattested trees.
func applyProvenanceGovernance(res *Result, root, policyPath string) {
	if res == nil {
		return
	}
	pol := loadPolicyForRisk(root, policyPath)
	if pol == nil || !pol.HasProvenanceGovernance() {
		return
	}
	res.Provenance.Enforced = true
	gov := pol.Provenance

	if !res.Provenance.Attested {
		applyUnattestedGovernance(&res.Provenance, gov)
		return
	}
	if gov.EnforcesProtocol() {
		applyProtocolGovernance(&res.Provenance)
		return
	}
	if gov.EnforcesAttested() {
		note := "attested provenance enforce: sessions present"
		if res.Provenance.Note != "" && !strings.Contains(res.Provenance.Note, "attested provenance enforce") {
			res.Provenance.Note = res.Provenance.Note + "; " + note
		} else {
			res.Provenance.Note = note
		}
	}
}

func applyUnattestedGovernance(sec *ProvenanceSection, gov *policy.ProvenanceGovernance) {
	if gov.EnforcesAttested() {
		sec.Status = StatusFail
		sec.Note = "attested provenance required — no session attestations found"
		return
	}
	// protocol: enforce alone stays idle on unattested trees.
	sec.Note = "protocol enforce idle — no session attestations"
}

func applyProtocolGovernance(sec *ProvenanceSection) {
	docs := sec.ProtocolDocs
	ok := sec.ProtocolOK
	switch {
	case docs == 0:
		sec.Status = StatusFail
		sec.Note = "APP protocol enforce: attested sessions missing .provenance.json"
	case ok < docs:
		sec.Status = StatusFail
		sec.Note = fmt.Sprintf(
			"APP protocol enforce: %d/%d docs valid (schema check failed)", ok, docs)
	default:
		note := fmt.Sprintf("APP protocol enforce: %d/%d docs ok", ok, docs)
		if sec.Note != "" && !strings.Contains(sec.Note, "APP protocol enforce") {
			sec.Note = sec.Note + "; " + note
		} else {
			sec.Note = note
		}
	}
}
