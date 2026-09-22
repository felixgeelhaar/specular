package gate

import (
	"strings"
	"testing"
)

func TestDenyNextStepsEmptyOnAllow(t *testing.T) {
	t.Parallel()
	if steps := DenyNextSteps(&Result{Verdict: Allow}); len(steps) != 0 {
		t.Fatalf("ALLOW steps=%v", steps)
	}
	if steps := DenyNextSteps(nil); len(steps) != 0 {
		t.Fatalf("nil steps=%v", steps)
	}
}

func TestDenyNextStepsAttested(t *testing.T) {
	t.Parallel()
	steps := DenyNextSteps(&Result{
		Verdict:    Deny,
		Provenance: ProvenanceSection{Status: StatusFail, Note: "attested provenance required (--require-attested)"},
	})
	joined := strings.Join(steps, "\n")
	if !strings.Contains(joined, "session attest") || !strings.Contains(joined, "session integrate") {
		t.Fatalf("steps=%v", steps)
	}
}

func TestDenyNextStepsProtocol(t *testing.T) {
	t.Parallel()
	steps := DenyNextSteps(&Result{
		Verdict: Deny,
		Provenance: ProvenanceSection{
			Status:   StatusFail,
			Attested: true,
			Note:     "APP protocol enforce: 0/1 docs valid (schema/binding check failed)",
		},
	})
	joined := strings.Join(steps, "\n")
	if !strings.Contains(joined, "session attest") || !strings.Contains(joined, "provenance verify") {
		t.Fatalf("steps=%v", steps)
	}
	if strings.Contains(joined, "--require-governed") {
		t.Fatalf("protocol DENY should not suggest governed path: %v", steps)
	}
}

func TestDenyNextStepsGoverned(t *testing.T) {
	t.Parallel()
	steps := DenyNextSteps(&Result{
		Verdict: Deny,
		Provenance: ProvenanceSection{
			Status:   StatusFail,
			Attested: true,
			Note:     "governed provenance required — no governed session attestations",
		},
	})
	joined := strings.Join(steps, "\n")
	if !strings.Contains(joined, "start --governed") || !strings.Contains(joined, "--require-governed") {
		t.Fatalf("steps=%v", steps)
	}
}

func TestDenyNextStepsDriftPolicyRisk(t *testing.T) {
	t.Parallel()
	steps := DenyNextSteps(&Result{
		Verdict: Deny,
		Drift:   DriftSection{Status: StatusFail},
		Policy:  PolicySection{Status: StatusFail},
		Risk:    RiskSection{Enforced: true, Missing: []string{"security"}},
	})
	joined := strings.Join(steps, "\n")
	for _, want := range []string{
		"baseline capture",
		"gate --policy",
		"missing: security",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %v", want, steps)
		}
	}
}

func TestFormatTextIncludesNextSteps(t *testing.T) {
	t.Parallel()
	res := &Result{
		Verdict:    Deny,
		Reason:     "attested provenance required",
		Provenance: ProvenanceSection{Status: StatusFail, Note: "attested provenance required (--require-attested)"},
	}
	text := FormatText(res)
	if !strings.Contains(text, "Next steps:") || !strings.Contains(text, "session attest") {
		t.Fatalf("text:\n%s", text)
	}
	md := FormatMarkdown(res)
	if !strings.Contains(md, "**Next steps:**") || !strings.Contains(md, "session attest") {
		t.Fatalf("md:\n%s", md)
	}
}
