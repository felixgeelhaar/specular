package gate

import (
	"fmt"
	"strings"
)

// DenyNextSteps returns PRODUCT_INTENT §5-style remediation lines for a DENY
// result (root-cause CLI fixes). Empty when ALLOW / nil / no actionable
// section failures. Soft-ALLOW exception tokens stay on Approvals Hint
// (SoftAllowPolicyHint); this list is the fix-the-failure path.
func DenyNextSteps(res *Result) []string {
	if res == nil || res.Verdict != Deny {
		return nil
	}
	var steps []string
	steps = append(steps, provenanceNextSteps(res.Provenance)...)
	if res.Drift.Status == StatusFail {
		steps = append(steps,
			"inspect Drift findings / SARIF above",
			"align change with spec/intent, or revert out-of-scope paths",
			"brownfield debt: specular baseline capture",
		)
	}
	if res.Policy.Status == StatusFail {
		steps = append(steps,
			"fix failed policy checks (see Policy Note)",
			"re-run: specular gate --policy <file>",
		)
	}
	if res.Risk.Enforced && len(res.Risk.Missing) > 0 {
		steps = append(steps, fmt.Sprintf(
			"satisfy missing role(s): specular approve exception-<id> --reason \"...\" --policy <role> (missing: %s)",
			strings.Join(res.Risk.Missing, ", ")))
	}
	return uniqPreserve(steps)
}

func provenanceNextSteps(p ProvenanceSection) []string {
	if p.Status != StatusFail {
		return nil
	}
	note := p.Note
	var steps []string
	switch {
	case strings.Contains(note, "APP protocol"):
		steps = append(steps,
			"emit/fix APP sibling: specular session attest <id>",
			"verify schema+binding: specular provenance verify <id>",
		)
	case strings.Contains(note, "governed provenance required"):
		steps = append(steps,
			"start governed session: specular session start --governed --harness <harness> \"…\"",
			"or hooks: specular session integrate <harness> --enforce --require-governed",
		)
	case strings.Contains(note, "attested provenance required") || !p.Attested:
		steps = append(steps,
			"write attestation: specular session attest <id>",
			"or install Stop hooks: specular session integrate <harness>",
		)
	default:
		steps = append(steps,
			"write attestation: specular session attest <id>",
			"verify APP: specular provenance verify <id>",
		)
	}
	return steps
}

func uniqPreserve(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func writeDenyNextSteps(b *strings.Builder, res *Result) {
	steps := DenyNextSteps(res)
	if len(steps) == 0 {
		return
	}
	b.WriteString("Next steps:\n")
	for _, s := range steps {
		fmt.Fprintf(b, "  • %s\n", s)
	}
}

func writeMarkdownDenyNextSteps(b *strings.Builder, res *Result) {
	steps := DenyNextSteps(res)
	if len(steps) == 0 {
		return
	}
	b.WriteString("**Next steps:**\n\n")
	for _, s := range steps {
		fmt.Fprintf(b, "- %s\n", s)
	}
	b.WriteString("\n")
}
