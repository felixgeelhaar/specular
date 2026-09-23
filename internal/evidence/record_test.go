package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/gate"
)

func TestWriteLoadLatestRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	res := &gate.Result{
		Verdict: gate.Allow,
		Reason:  "drift skipped; policy skipped; provenance unattested",
		Change:  gate.ChangeSection{Branch: "main", Commit: "abc123def456", Dirty: false},
		Drift:   gate.DriftSection{Status: gate.StatusSkipped, Note: "brownfield"},
		Policy:  gate.PolicySection{Status: gate.StatusSkipped},
		Provenance: gate.ProvenanceSection{
			Status: gate.StatusSkipped,
			Note:   "unattested",
		},
	}
	rec, err := NewFromGate(root, res)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rec.ID, "ev_") {
		t.Fatalf("id=%s", rec.ID)
	}
	if rec.Commit != "abc123def456" {
		t.Fatalf("commit=%q", rec.Commit)
	}
	if err := Write(root, rec); err != nil {
		t.Fatal(err)
	}
	got, err := LoadLatest(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != rec.ID || got.Gate.Verdict != gate.Allow {
		t.Fatalf("%+v", got)
	}
	path := filepath.Join(Dir(root), rec.ID+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestFormatExplainDenyWithFindings(t *testing.T) {
	t.Parallel()
	rec := &Record{
		ID:        "ev_test",
		CreatedAt: mustTime(t, "2026-09-21T09:41:00Z"),
		Root:      "/repos/payments-api",
		Branch:    "feature/auth",
		Gate: &gate.Result{
			Verdict: gate.Deny,
			Reason:  "hash mismatch",
			Change:  gate.ChangeSection{Dirty: true, Files: 2, Branch: "feature/auth"},
			Drift: gate.DriftSection{
				Status: gate.StatusFail,
				Errors: 1,
				Note:   "hash mismatch",
				Findings: []gate.FindingDetail{{
					Category: "code",
					Code:     "HASH_MISMATCH",
					Severity: "error",
					Message:  "file hash mismatch",
					Location: "src/a.go",
				}},
			},
			Policy:     gate.PolicySection{Status: gate.StatusSkipped},
			Provenance: gate.ProvenanceSection{Status: gate.StatusSkipped},
			Approvals: gate.ApprovalsSection{
				Count: 1,
				Exceptions: []gate.ApprovalSummary{{
					Type:       "exception",
					ResourceID: "exception-EX-192",
					Reason:     "emergency auth hotfix",
					Scope:      "internal/auth/**",
					ApprovedBy: "alice",
				}},
				Note: "1 open exception(s)",
			},
		},
	}
	text := FormatExplain(rec)
	for _, want := range []string{
		"AI CHANGE RECORD",
		"Decision     DENY",
		"Reason       hash mismatch",
		"Timestamp    2026-09-21T09:41:00Z",
		"Evidence     ev_test",
		"Repository   payments-api",
		"Branch       feature/auth",
		"Files        2 uncommitted",
		"Status       UNATTESTED",
		"✗ FAIL",
		"HASH_MISMATCH",
		"at src/a.go",
		"Approvals",
		"exception-EX-192",
		"emergency auth hotfix",
		"scope=internal/auth/**",
		"open exception(s) on local trail",
		"Why",
		"Drift failed",
		"→ DENY",
		".specular/evidence/ev_test.json",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
	if !strings.Contains(text, "⚠") || !strings.Contains(text, "exception") {
		t.Fatalf("missing exception mark:\n%s", text)
	}
	if strings.Contains(text, "SPECULAR EXPLAIN") {
		t.Fatalf("legacy title still present:\n%s", text)
	}
}

func TestFormatExplainAllowAttested(t *testing.T) {
	t.Parallel()
	rec := &Record{
		ID: "ev_ok",
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Reason:  "drift pass; policy pass",
			Change:  gate.ChangeSection{Dirty: false, Files: 0, Branch: "main", Root: "/tmp/demo"},
			Drift:   gate.DriftSection{Status: gate.StatusPass},
			Policy: gate.PolicySection{
				Status:  gate.StatusPass,
				Passed:  3,
				Failed:  0,
				Skipped: 1,
			},
			Provenance: gate.ProvenanceSection{
				Status:           gate.StatusPass,
				Attested:         true,
				Harnesses:        []string{"Claude Code"},
				Sessions:         []string{"sp_92d1"},
				WorktreePaths:    []string{"/tmp/wt/auth"},
				WorktreeBranches: []string{"specular/auth"},
				WorktreeNames:    []string{"auth"},
				Governed:         true,
				Note:             "session attestation(s) present; worktree isolated; governed",
			},
		},
	}
	text := FormatExplain(rec)
	for _, want := range []string{
		"AI CHANGE RECORD",
		"Decision     ALLOW",
		"Status       ATTESTED",
		"Harness      Claude Code",
		"Session      sp_92d1",
		"Worktree     /tmp/wt/auth",
		"WtBranch     specular/auth",
		"WtName       auth",
		"Governed     true",
		"✓ PASS",
		"Summary      None detected",
		"Checks       passed=3 failed=0 skipped=1",
		"→ ALLOW",
		"Files        clean working tree",
		"• Worktree /tmp/wt/auth · branch=specular/auth",
		"• Governed true",
		"Session      specular session show sp_92d1",
		"             specular explain --session sp_92d1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
}

func TestFormatExplainRiskAndSoftAllow(t *testing.T) {
	t.Parallel()
	rec := &Record{
		ID: "ev_soft",
		Gate: &gate.Result{
			Verdict: gate.Allow,
			Reason:  "exception-EX-1 overruled drift DENY (scope→a.go); drift fail (exception soft-ALLOW)",
			Drift: gate.DriftSection{
				Status: gate.StatusFail,
				Note:   "hash mismatch",
				Findings: []gate.FindingDetail{{
					Code: "HASH_MISMATCH", Severity: "error", Path: "a.go",
				}},
			},
			Policy:     gate.PolicySection{Status: gate.StatusSkipped},
			Provenance: gate.ProvenanceSection{Status: gate.StatusSkipped},
			Risk: gate.RiskSection{
				Level:    "HIGH",
				Enforced: true,
				Factors:  []string{"authentication / authorization paths modified"},
				Required: []string{"security"},
				Observed: []string{"security"},
				Note:     "risk-adaptive requirements satisfied",
			},
			Approvals: gate.ApprovalsSection{
				Count: 1,
				Exceptions: []gate.ApprovalSummary{{
					ResourceID: "exception-EX-1",
					Reason:     "hotfix",
					Scope:      "a.go",
					Policy:     "drift",
				}},
				Overrules: []gate.ExceptionOverrule{{
					ResourceID: "exception-EX-1",
					Kind:       "drift",
					Binding:    "scope→a.go",
				}},
				Note: "1 open exception(s); 1 overruled drift DENY → soft-ALLOW",
			},
		},
	}
	text := FormatExplain(rec)
	for _, want := range []string{
		"Risk",
		"Level        HIGH (enforced)",
		"+ authentication / authorization paths modified",
		"Required     security",
		"Observed     security",
		"Overruled",
		"soft-ALLOW drift",
		"exception-EX-1",
		"scope→a.go",
		"Show         specular approvals show exception-EX-1",
		"List         specular approvals list --evidence ev_soft",
		"Pending      specular approvals pending",
		"Doctor       specular doctor",
		"Approval     specular approvals show exception-EX-1",
		"Open         specular approvals list --evidence ev_soft",
		"exception soft-ALLOW overrule",
		"ALLOW via scoped exception soft-ALLOW",
		"Risk level HIGH (enforced)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
}

func TestFormatExplainDenyWithoutApprovalsHint(t *testing.T) {
	t.Parallel()
	rec := &Record{
		ID: "ev_deny",
		Gate: &gate.Result{
			Verdict:    gate.Deny,
			Reason:     "policy failed",
			Drift:      gate.DriftSection{Status: gate.StatusSkipped},
			Policy:     gate.PolicySection{Status: gate.StatusFail, Note: "policy failed"},
			Provenance: gate.ProvenanceSection{Status: gate.StatusSkipped},
			Approvals:  gate.ApprovalsSection{},
		},
	}
	text := FormatExplain(rec)
	for _, want := range []string{
		"Approvals",
		"none recorded",
		"specular approve exception-",
		"--policy policy",
		"No local exception/approval trail for this DENY",
		"Pending      specular approvals pending",
		"Doctor       specular doctor",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
}

func TestFormatExplainProvenanceDenyHint(t *testing.T) {
	t.Parallel()
	rec := &Record{
		ID: "ev_prov_deny",
		Gate: &gate.Result{
			Verdict:    gate.Deny,
			Reason:     "provenance failed",
			Provenance: gate.ProvenanceSection{Status: gate.StatusFail},
			Approvals:  gate.ApprovalsSection{},
		},
	}
	text := FormatExplain(rec)
	if !strings.Contains(text, "--policy provenance") {
		t.Fatalf("expected provenance hint:\n%s", text)
	}
	if !strings.Contains(text, "Next steps") || !strings.Contains(text, "session attest") {
		t.Fatalf("expected Next steps remediation:\n%s", text)
	}
}

func TestFormatExplainNil(t *testing.T) {
	t.Parallel()
	if got := FormatExplain(nil); got != "No evidence record.\n" {
		t.Fatalf("got %q", got)
	}
}

func TestSoftAllowResourceIDsDedup(t *testing.T) {
	t.Parallel()
	overrules := []gate.ExceptionOverrule{
		{ResourceID: "exception-a", Kind: "drift"},
		{ResourceID: "exception-a", Kind: "policy"},
		{ResourceID: "  ", Kind: "risk"},
		{ResourceID: "exception-b", Kind: "risk"},
	}
	ids := gate.SoftAllowResourceIDs(overrules)
	if len(ids) != 2 || ids[0] != "exception-a" || ids[1] != "exception-b" {
		t.Fatalf("ids=%v", ids)
	}
	if gate.SoftAllowResourceIDs(nil) != nil {
		t.Fatal("expected nil for empty")
	}
}

func mustTime(t *testing.T, rfc3339 string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func TestListIDsOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	r1, err := NewFromGate(root, &gate.Result{
		Verdict: gate.Allow,
		Reason:  "a",
		Drift:   gate.DriftSection{Status: gate.StatusSkipped},
		Policy:  gate.PolicySection{Status: gate.StatusSkipped},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(root, r1); err != nil {
		t.Fatal(err)
	}
	// Distinct gate payload → distinct id
	r2, err := NewFromGate(root, &gate.Result{
		Verdict: gate.Deny,
		Reason:  "b",
		Drift:   gate.DriftSection{Status: gate.StatusFail, Note: "x"},
		Policy:  gate.PolicySection{Status: gate.StatusSkipped},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(root, r2); err != nil {
		t.Fatal(err)
	}
	ids, err := ListIDs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("ids=%v", ids)
	}
	latest, err := LoadLatest(root)
	if err != nil {
		t.Fatal(err)
	}
	if latest.ID != r2.ID {
		t.Fatalf("latest=%s want %s", latest.ID, r2.ID)
	}
}
