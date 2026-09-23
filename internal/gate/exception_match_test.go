package gate

import (
	"strings"
	"testing"
)

func TestDecideSoftAllowDriftByScope(t *testing.T) {
	t.Parallel()
	res := &Result{
		Drift: DriftSection{
			Status: StatusFail,
			Note:   "file hash mismatch",
			Findings: []FindingDetail{{
				Code:     "HASH_MISMATCH",
				Severity: "error",
				Path:     "internal/auth/token.go",
				Location: "internal/auth/token.go:1",
			}},
		},
		Policy: PolicySection{Status: StatusSkipped},
		Approvals: ApprovalsSection{
			Count: 1,
			Exceptions: []ApprovalSummary{{
				Type:       "exception",
				ResourceID: "exception-EX-192",
				Reason:     "hotfix",
				Scope:      "internal/auth/**",
				Policy:     "drift",
			}},
		},
	}
	v, reason := decide(res)
	if v != Allow {
		t.Fatalf("verdict=%s reason=%s", v, reason)
	}
	if res.Drift.Status != StatusFail {
		t.Fatal("drift status must remain FAIL")
	}
	if len(res.Approvals.Overrules) != 1 || res.Approvals.Overrules[0].Kind != "drift" {
		t.Fatalf("overrules=%+v", res.Approvals.Overrules)
	}
	if !strings.Contains(reason, "overruled drift") {
		t.Fatalf("reason=%s", reason)
	}
	text := FormatText(res)
	if !strings.Contains(text, "Overruled") || !strings.Contains(text, "soft-ALLOW drift") {
		t.Fatalf("board:\n%s", text)
	}
	for _, want := range []string{
		"Show           specular approvals show exception-EX-192",
		"List           specular approvals list --status open",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing soft-ALLOW jump %q:\n%s", want, text)
		}
	}
	textEv := FormatTextWith(res, FormatTextOptions{EvidenceID: "ev_soft"})
	if !strings.Contains(textEv, "List           specular approvals list --evidence ev_soft") {
		t.Fatalf("expected --evidence list:\n%s", textEv)
	}
	md := FormatMarkdown(res)
	for _, want := range []string{
		"Show: `specular approvals show exception-EX-192`",
		"List: `specular approvals list --status open`",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing markdown soft-ALLOW jump %q:\n%s", want, md)
		}
	}
	mdEv := FormatMarkdownWith(res, FormatMarkdownOptions{EvidenceID: "ev_soft"})
	if !strings.Contains(mdEv, "List: `specular approvals list --evidence ev_soft`") {
		t.Fatalf("expected markdown --evidence list:\n%s", mdEv)
	}
}

func TestDecideNoSoftAllowUnboundException(t *testing.T) {
	t.Parallel()
	res := &Result{
		Drift: DriftSection{
			Status: StatusFail,
			Note:   "file hash mismatch",
			Findings: []FindingDetail{{
				Code:     "HASH_MISMATCH",
				Severity: "error",
				Path:     "internal/auth/token.go",
			}},
		},
		Policy: PolicySection{Status: StatusSkipped},
		Approvals: ApprovalsSection{
			Exceptions: []ApprovalSummary{{
				ResourceID: "exception-unrelated",
				Reason:     "other",
				Scope:      "docs/**",
				Policy:     "drift",
			}},
		},
	}
	v, reason := decide(res)
	if v != Deny {
		t.Fatalf("verdict=%s reason=%s", v, reason)
	}
	if len(res.Approvals.Overrules) != 0 {
		t.Fatalf("overrules=%v", res.Approvals.Overrules)
	}
}

func TestDecideSoftAllowPolicyByCheck(t *testing.T) {
	t.Parallel()
	res := &Result{
		Drift: DriftSection{Status: StatusSkipped},
		Policy: PolicySection{
			Status:       StatusFail,
			Note:         "policy verification failed",
			FailedChecks: []string{"Coverage"},
		},
		Approvals: ApprovalsSection{
			Exceptions: []ApprovalSummary{{
				ResourceID: "exception-cov",
				Reason:     "waiver",
				Policy:     "Coverage",
				Scope:      "payments",
			}},
		},
	}
	v, reason := decide(res)
	if v != Allow || !strings.Contains(reason, "overruled policy") {
		t.Fatalf("%s %s", v, reason)
	}
	if res.Policy.Status != StatusFail {
		t.Fatal("policy status must remain FAIL")
	}
}

func TestDecideSoftAllowRequiresBothKinds(t *testing.T) {
	t.Parallel()
	res := &Result{
		Drift: DriftSection{
			Status: StatusFail,
			Findings: []FindingDetail{{
				Code: "HASH_MISMATCH", Severity: "error", Path: "a.go",
			}},
		},
		Policy: PolicySection{
			Status:       StatusFail,
			FailedChecks: []string{"Tests"},
		},
		Approvals: ApprovalsSection{
			Exceptions: []ApprovalSummary{{
				ResourceID: "exception-drift-only",
				Policy:     "HASH_MISMATCH",
				Scope:      "a.go",
			}},
		},
	}
	v, _ := decide(res)
	if v != Deny {
		t.Fatalf("expected DENY when only drift overruled, got %s", v)
	}
}

func TestDecideSoftAllowBothKinds(t *testing.T) {
	t.Parallel()
	res := &Result{
		Drift: DriftSection{
			Status: StatusFail,
			Findings: []FindingDetail{{
				Code: "HASH_MISMATCH", Severity: "error", Path: "a.go",
			}},
		},
		Policy: PolicySection{
			Status:       StatusFail,
			FailedChecks: []string{"Tests"},
		},
		Approvals: ApprovalsSection{
			Exceptions: []ApprovalSummary{
				{ResourceID: "exception-drift", Policy: "HASH_MISMATCH", Scope: "a.go"},
				{ResourceID: "exception-tests", Policy: "Tests", Scope: "unit"},
			},
		},
	}
	v, reason := decide(res)
	if v != Allow {
		t.Fatalf("%s %s", v, reason)
	}
	if len(res.Approvals.Overrules) != 2 {
		t.Fatalf("overrules=%+v", res.Approvals.Overrules)
	}
}

func TestDecideRiskSoftAllowCategory(t *testing.T) {
	t.Parallel()
	res := &Result{
		Drift:  DriftSection{Status: StatusSkipped},
		Policy: PolicySection{Status: StatusSkipped},
		Risk: RiskSection{
			Level:    "CRITICAL",
			Enforced: true,
			Required: []string{"security"},
			Missing:  []string{"security"},
		},
		Approvals: ApprovalsSection{
			Exceptions: []ApprovalSummary{{
				ResourceID: "exception-risk-outage",
				Reason:     "SEV-1",
				Policy:     "risk",
				Scope:      "auth-service",
			}},
		},
	}
	v, reason := decide(res)
	if v != Allow || !strings.Contains(reason, "overruled risk") {
		t.Fatalf("%s %s", v, reason)
	}
	if len(res.Risk.Missing) == 0 {
		t.Fatal("Missing should remain for audit; soft-ALLOW does not clear roles")
	}
}

func TestScopeMatchesPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		scope, path string
		want        bool
	}{
		{"internal/auth/**", "internal/auth/token.go", true},
		{"internal/auth/**", "internal/other/x.go", false},
		{"a.go", "a.go", true},
		{"docs", "docs/readme.md", true},
		{"", "a.go", false},
	}
	for _, tc := range cases {
		if got := scopeMatchesPath(tc.scope, tc.path); got != tc.want {
			t.Fatalf("scope=%q path=%q got %v want %v", tc.scope, tc.path, got, tc.want)
		}
	}
}

func TestExceptionMatchesRequiresPolicyOrScope(t *testing.T) {
	t.Parallel()
	res := &Result{
		Drift: DriftSection{Status: StatusFail, Findings: []FindingDetail{{
			Code: "X", Severity: "error", Path: "a.go",
		}}},
	}
	_, ok := exceptionMatches(ApprovalSummary{ResourceID: "e", Reason: "only reason"}, DenyKindDrift, res)
	if ok {
		t.Fatal("reason-only must not match")
	}
}
