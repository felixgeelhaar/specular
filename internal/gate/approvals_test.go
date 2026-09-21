package gate

import (
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/approval"
)

func TestDiscoverApprovalsOpenException(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Now().UTC()
	exp := now.Add(24 * time.Hour)
	if _, err := approval.Write(root, &approval.Record{
		Type:       approval.TypeException,
		ResourceID: "exception-EX-192",
		ApprovedBy: "alice",
		ApprovedAt: now,
		Reason:     "emergency hotfix",
		Scope:      "internal/auth/**",
		Policy:     "SEC-17",
		ExpiresAt:  &exp,
	}); err != nil {
		t.Fatal(err)
	}

	sec := discoverApprovals(root)
	if sec.Count != 1 {
		t.Fatalf("count=%d", sec.Count)
	}
	if len(sec.Exceptions) != 1 || sec.Exceptions[0].ResourceID != "exception-EX-192" {
		t.Fatalf("exceptions=%+v", sec.Exceptions)
	}
	if !strings.Contains(sec.Note, "open exception") {
		t.Fatalf("note=%q", sec.Note)
	}
}

func TestFormatTextApprovalsOnDeny(t *testing.T) {
	t.Parallel()
	res := &Result{
		Verdict: Deny,
		Reason:  "policy evaluation failed",
		Drift:   DriftSection{Status: StatusSkipped},
		Policy:  PolicySection{Status: StatusFail, Note: "policy evaluation failed"},
		Risk:    RiskSection{Level: "LOW", Note: "advisory"},
		Approvals: ApprovalsSection{
			Count: 1,
			Exceptions: []ApprovalSummary{{
				Type:       approval.TypeException,
				ResourceID: "exception-EX-192",
				ApprovedBy: "alice",
				Reason:     "emergency hotfix",
				Scope:      "internal/auth/**",
				Path:       ".specular/approvals/exception-20260921.yaml",
			}},
			Recent: []ApprovalSummary{{
				Type:       approval.TypeException,
				ResourceID: "exception-EX-192",
				ApprovedBy: "alice",
				Reason:     "emergency hotfix",
			}},
			Note: "1 open exception(s); trail is advisory — does not change gate verdict",
		},
	}
	text := FormatText(res)
	for _, want := range []string{
		"Approvals",
		"OpenExceptions",
		"exception-EX-192",
		"emergency hotfix",
		"scope=internal/auth/**",
		"VERDICT: DENY",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
}

func TestFormatTextApprovalsHintWhenEmptyDeny(t *testing.T) {
	t.Parallel()
	res := &Result{
		Verdict:   Deny,
		Reason:    "drift failed",
		Drift:     DriftSection{Status: StatusFail},
		Approvals: ApprovalsSection{Note: "no local approval/exception records under .specular/approvals/"},
	}
	text := FormatText(res)
	if !strings.Contains(text, "specular approve exception-") {
		t.Fatalf("expected hint:\n%s", text)
	}
}

func TestFormatMarkdownApprovals(t *testing.T) {
	t.Parallel()
	res := &Result{
		Verdict: Deny,
		Reason:  "deny",
		Approvals: ApprovalsSection{
			Count: 1,
			Exceptions: []ApprovalSummary{{
				ResourceID: "exception-EX-192",
				Reason:     "hotfix",
			}},
		},
	}
	md := FormatMarkdown(res)
	if !strings.Contains(md, "Open exceptions") || !strings.Contains(md, "exception-EX-192") {
		t.Fatalf("md:\n%s", md)
	}
	if !strings.Contains(md, "| Approvals |") {
		t.Fatalf("missing approvals row:\n%s", md)
	}
}

func TestEvaluateLoadsApprovals(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := approval.Write(root, &approval.Record{
		Type:       approval.TypeBundle,
		ResourceID: "bundle-demo",
		ApprovedBy: "carol",
		ApprovedAt: time.Now().UTC(),
		Message:    "ok",
	}); err != nil {
		t.Fatal(err)
	}

	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.Approvals.Count < 1 {
		t.Fatalf("approvals=%+v", res.Approvals)
	}
}
