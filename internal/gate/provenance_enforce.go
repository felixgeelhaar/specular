package gate

import (
	"fmt"
	"strings"
)

// applyProvenanceGovernance overlays opt-in policy provenance.protocol: enforce
// onto res.Provenance. No-op when policy is missing or protocol is advisory.
// When enforced and sessions are attested without valid APP docs, Status=FAIL.
func applyProvenanceGovernance(res *Result, root, policyPath string) {
	if res == nil {
		return
	}
	pol := loadPolicyForRisk(root, policyPath)
	if pol == nil || !pol.HasProvenanceGovernance() {
		return
	}
	res.Provenance.Enforced = true
	if !res.Provenance.Attested {
		res.Provenance.Note = firstNonEmpty(res.Provenance.Note,
			"protocol enforce idle — no session attestations")
		return
	}

	docs := res.Provenance.ProtocolDocs
	ok := res.Provenance.ProtocolOK
	switch {
	case docs == 0:
		res.Provenance.Status = StatusFail
		res.Provenance.Note = "APP protocol enforce: attested sessions missing .provenance.json"
	case ok < docs:
		res.Provenance.Status = StatusFail
		res.Provenance.Note = fmt.Sprintf(
			"APP protocol enforce: %d/%d docs valid (schema check failed)", ok, docs)
	default:
		note := fmt.Sprintf("APP protocol enforce: %d/%d docs ok", ok, docs)
		if res.Provenance.Note != "" && !strings.Contains(res.Provenance.Note, "APP protocol enforce") {
			res.Provenance.Note = res.Provenance.Note + "; " + note
		} else {
			res.Provenance.Note = note
		}
	}
}
