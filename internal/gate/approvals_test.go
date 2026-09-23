package gate

import (
	"os"
	"path/filepath"
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
		"Show           specular approvals show exception-EX-192",
		"List           specular approvals list --status open",
		"Pending        specular approvals pending",
		"Doctor         specular doctor",
		"VERDICT: DENY",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
	md := FormatMarkdown(res)
	for _, want := range []string{
		"Show: `specular approvals show exception-EX-192`",
		"List: `specular approvals list --status open`",
		"Pending: `specular approvals pending`",
		"Doctor: `specular doctor`",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing markdown Soft trail %q:\n%s", want, md)
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
	if !strings.Contains(text, "--policy drift") {
		t.Fatalf("expected drift-specific policy hint:\n%s", text)
	}
}

func TestFormatTextApprovalsHintWhenProvenanceDeny(t *testing.T) {
	t.Parallel()
	res := &Result{
		Verdict:    Deny,
		Reason:     "provenance.attested: enforce requires attestation",
		Provenance: ProvenanceSection{Status: StatusFail, Note: "unattested"},
		Approvals:  ApprovalsSection{},
	}
	text := FormatText(res)
	if !strings.Contains(text, "--policy provenance") {
		t.Fatalf("expected provenance policy hint:\n%s", text)
	}
	md := FormatMarkdown(res)
	if !strings.Contains(md, "--policy provenance") {
		t.Fatalf("expected markdown provenance policy hint:\n%s", md)
	}
}

func TestSoftAllowListHint(t *testing.T) {
	t.Parallel()
	if SoftAllowListHint("ev_abc") != "specular approvals list --evidence ev_abc" {
		t.Fatal(SoftAllowListHint("ev_abc"))
	}
	if SoftAllowListHint("  ") != "specular approvals list --status open" {
		t.Fatal(SoftAllowListHint(""))
	}
	if SoftAllowPendingHint != "specular approvals pending" {
		t.Fatal(SoftAllowPendingHint)
	}
	if SoftAllowDoctorHint != "specular doctor" {
		t.Fatal(SoftAllowDoctorHint)
	}
}

func TestSoftAllowResourceIDsDedup(t *testing.T) {
	t.Parallel()
	ids := SoftAllowResourceIDs([]ExceptionOverrule{
		{ResourceID: "exception-a", Kind: "drift"},
		{ResourceID: "exception-a", Kind: "policy"},
		{ResourceID: "  ", Kind: "risk"},
		{ResourceID: "exception-b", Kind: "risk"},
	})
	if len(ids) != 2 || ids[0] != "exception-a" || ids[1] != "exception-b" {
		t.Fatalf("ids=%v", ids)
	}
	if SoftAllowResourceIDs(nil) != nil {
		t.Fatal("expected nil for empty")
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

func TestDiscoverApprovalsAfterClose(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Now().UTC()
	exp := now.Add(24 * time.Hour)
	if _, err := approval.Write(root, &approval.Record{
		Type:       approval.TypeException,
		ResourceID: "exception-app",
		ApprovedBy: "alice",
		ApprovedAt: now,
		Reason:     "migrate hooks",
		Policy:     "provenance",
		ExpiresAt:  &exp,
	}); err != nil {
		t.Fatal(err)
	}
	sec := discoverApprovals(root)
	if len(sec.Exceptions) != 1 {
		t.Fatalf("before close exceptions=%+v", sec.Exceptions)
	}
	if _, err := approval.Close(root, "exception-app", approval.CloseOptions{
		Now: now.Add(time.Minute),
		By:  "alice",
	}); err != nil {
		t.Fatal(err)
	}
	sec = discoverApprovals(root)
	if len(sec.Exceptions) != 0 {
		t.Fatalf("after close exceptions=%+v", sec.Exceptions)
	}
	if sec.Count != 1 {
		t.Fatalf("count should still list the file: %d", sec.Count)
	}
}

func TestSoftAllowGoneAfterClose(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	writeRiskPolicy(t, root, "provenance:\n  protocol: enforce\n")
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	att := `{"provenance":{"harness":"claude-code"}}`
	if err := os.WriteFile(filepath.Join(dir, "auth.attestation.json"), []byte(att), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := approval.Write(root, &approval.Record{
		Type:       approval.TypeException,
		ResourceID: "exception-app-protocol",
		ApprovedBy: "platform",
		ApprovedAt: time.Now().UTC(),
		Reason:     "migrate",
		Policy:     "provenance",
	}); err != nil {
		t.Fatal(err)
	}

	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != Allow {
		t.Fatalf("with open exception expected ALLOW, got %s %s", res.Verdict, res.Reason)
	}

	if _, err := approval.Close(root, "exception-app-protocol", approval.CloseOptions{
		Now: time.Now().UTC(),
		By:  "platform",
	}); err != nil {
		t.Fatal(err)
	}

	res2, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Verdict != Deny {
		t.Fatalf("after close expected DENY, got %s %s", res2.Verdict, res2.Reason)
	}
	if len(res2.Approvals.Overrules) != 0 {
		t.Fatalf("overrules=%+v", res2.Approvals.Overrules)
	}
}
