package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/approval"
)

func TestDetectRiskFactorsPathHeuristics(t *testing.T) {
	t.Parallel()
	factors := detectRiskFactors([]string{
		"internal/auth/token.go",
		"go.mod",
		".github/workflows/ci.yml",
		"config/secrets/prod.env",
		"README.md",
	}, true)
	want := []string{
		FactorAuthPaths,
		FactorDependencyPaths,
		FactorInfraCIPaths,
		FactorSecretsPaths,
	}
	if len(factors) != len(want) {
		t.Fatalf("factors=%v want %v", factors, want)
	}
	for i := range want {
		if factors[i] != want[i] {
			t.Fatalf("factors=%v want %v", factors, want)
		}
	}
	if advisoryRiskLevel(factors) != "CRITICAL" {
		t.Fatalf("level=%s", advisoryRiskLevel(factors))
	}
}

func TestDetectRiskFactorsUnattestedOnly(t *testing.T) {
	t.Parallel()
	factors := detectRiskFactors(nil, false)
	if len(factors) != 1 || factors[0] != FactorUnattested {
		t.Fatalf("factors=%v", factors)
	}
	if advisoryRiskLevel(factors) != "LOW" {
		t.Fatalf("level=%s", advisoryRiskLevel(factors))
	}
}

func TestDetectRiskFactorsAttestedClean(t *testing.T) {
	t.Parallel()
	factors := detectRiskFactors([]string{"README.md", "docs/guide.md"}, true)
	if len(factors) != 0 {
		t.Fatalf("factors=%v", factors)
	}
	if advisoryRiskLevel(factors) != "NONE" {
		t.Fatalf("level=%s", advisoryRiskLevel(factors))
	}
}

func TestParsePorcelainPaths(t *testing.T) {
	t.Parallel()
	out := "" +
		" M internal/auth/token.go\n" +
		"?? go.mod\n" +
		"R  old.env -> .env.production\n" +
		"A  \"path with spaces/file.go\"\n"
	got := parsePorcelainPaths(out)
	want := []string{
		"internal/auth/token.go",
		"go.mod",
		".env.production",
		"path with spaces/file.go",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestEvaluateRiskAdvisoryStillAllow(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	// Sensitive paths in the dirty tree — advisory factors only.
	authDir := filepath.Join(root, "internal", "auth")
	if err := os.MkdirAll(authDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "token.go"), []byte("package auth\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wf := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(wf, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wf, "ci.yml"), []byte("name: ci\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != Allow {
		t.Fatalf("verdict=%s reason=%s — risk must not deny", res.Verdict, res.Reason)
	}
	for _, want := range []string{
		FactorAuthPaths,
		FactorDependencyPaths,
		FactorInfraCIPaths,
		FactorSecretsPaths,
		FactorUnattested,
	} {
		if !containsFactor(res.Risk.Factors, want) {
			t.Fatalf("missing factor %q in %v", want, res.Risk.Factors)
		}
	}
	if res.Risk.Level != "CRITICAL" {
		t.Fatalf("level=%s factors=%v", res.Risk.Level, res.Risk.Factors)
	}

	text := FormatText(res)
	for _, want := range []string{
		"Risk",
		"Level          CRITICAL (advisory)",
		"+ " + FactorAuthPaths,
		"VERDICT: ALLOW",
		"advisory only",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("board missing %q:\n%s", want, text)
		}
	}

	md := FormatMarkdown(res)
	for _, want := range []string{
		"| Risk | `CRITICAL` (advisory) |",
		"_Risk factors (advisory):_",
		FactorAuthPaths,
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
}

func TestDecideIgnoresRiskFactors(t *testing.T) {
	t.Parallel()
	res := &Result{
		Drift:  DriftSection{Status: StatusSkipped},
		Policy: PolicySection{Status: StatusSkipped},
		Provenance: ProvenanceSection{
			Status:   StatusSkipped,
			Attested: false,
		},
		Risk: RiskSection{
			Level: "HIGH",
			Factors: []string{
				FactorAuthPaths,
				FactorSecretsPaths,
				FactorUnattested,
			},
		},
	}
	v, reason := decide(res)
	if v != Allow {
		t.Fatalf("verdict=%s reason=%s", v, reason)
	}
}

func TestFormatTextRiskSection(t *testing.T) {
	t.Parallel()
	res := &Result{
		Verdict: Allow,
		Reason:  "drift skipped; policy skipped; provenance unattested",
		Risk: RiskSection{
			Level:   "MEDIUM",
			Factors: []string{FactorDependencyPaths, FactorUnattested},
			Note:    "advisory only — does not change gate verdict",
		},
	}
	text := FormatText(res)
	if !strings.Contains(text, "Level          MEDIUM (advisory)") {
		t.Fatalf("missing level:\n%s", text)
	}
	if !strings.Contains(text, "+ "+FactorDependencyPaths) {
		t.Fatalf("missing factor:\n%s", text)
	}
}

func TestEvaluateRiskAdaptiveDenyMissingApproval(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	writeRiskPolicy(t, root, `
risk:
  high:
    approvals: [security]
  critical:
    approvals: [security, platform]
`)
	// Auth + deps + infra + secrets + unattested → CRITICAL
	plantRiskFiles(t, root)

	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != Deny {
		t.Fatalf("verdict=%s reason=%s", res.Verdict, res.Reason)
	}
	if !res.Risk.Enforced {
		t.Fatal("expected risk enforced")
	}
	if res.Risk.Level != "CRITICAL" {
		t.Fatalf("level=%s", res.Risk.Level)
	}
	if !containsFactor(res.Risk.Missing, "security") || !containsFactor(res.Risk.Missing, "platform") {
		t.Fatalf("missing=%v required=%v", res.Risk.Missing, res.Risk.Required)
	}
	if !strings.Contains(res.Reason, "missing") {
		t.Fatalf("reason=%s", res.Reason)
	}

	text := FormatText(res)
	for _, want := range []string{
		"Level          CRITICAL (enforced)",
		"Required       security, platform",
		"Missing        security, platform",
		"VERDICT: DENY",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("board missing %q:\n%s", want, text)
		}
	}
	md := FormatMarkdown(res)
	for _, want := range []string{
		"| Risk | `CRITICAL` (enforced) |",
		"_Risk required:_",
		"**missing**",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
}

func TestEvaluateRiskAdaptiveAllowWithException(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	writeRiskPolicy(t, root, `
risk:
  critical:
    approvals: [security]
`)
	plantRiskFiles(t, root)

	rec := &approval.Record{
		Type:       approval.TypeException,
		ResourceID: "exception-risk-security",
		ApprovedBy: "sec-reviewer",
		Reason:     "reviewed auth change",
		Policy:     "security",
		Scope:      "auth-paths",
	}
	if _, err := approval.Write(root, rec); err != nil {
		t.Fatal(err)
	}

	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != Allow {
		t.Fatalf("verdict=%s reason=%s risk=%+v", res.Verdict, res.Reason, res.Risk)
	}
	if !res.Risk.Enforced || len(res.Risk.Missing) != 0 {
		t.Fatalf("enforced=%v missing=%v", res.Risk.Enforced, res.Risk.Missing)
	}
	if !containsFactor(res.Risk.Observed, "security") {
		t.Fatalf("observed=%v", res.Risk.Observed)
	}
	if !strings.Contains(res.Reason, "risk approvals satisfied") {
		t.Fatalf("reason=%s", res.Reason)
	}
	text := FormatText(res)
	if !strings.Contains(text, "OpenExceptions") || !strings.Contains(text, "exception-risk-security") {
		t.Fatalf("expected exception on board:\n%s", text)
	}
}

func TestDecideRiskEnforcedMissingDenies(t *testing.T) {
	t.Parallel()
	res := &Result{
		Drift:  DriftSection{Status: StatusSkipped},
		Policy: PolicySection{Status: StatusSkipped},
		Risk: RiskSection{
			Level:    "HIGH",
			Enforced: true,
			Required: []string{"security"},
			Missing:  []string{"security"},
		},
	}
	v, reason := decide(res)
	if v != Deny || !strings.Contains(reason, "security") {
		t.Fatalf("%s %s", v, reason)
	}
}

func writeRiskPolicy(t *testing.T, root, body string) {
	t.Helper()
	dir := filepath.Join(root, ".specular")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "policy.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func plantRiskFiles(t *testing.T, root string) {
	t.Helper()
	authDir := filepath.Join(root, "internal", "auth")
	if err := os.MkdirAll(authDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "token.go"), []byte("package auth\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wf := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(wf, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wf, "ci.yml"), []byte("name: ci\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func containsFactor(factors []string, want string) bool {
	for _, f := range factors {
		if f == want {
			return true
		}
	}
	return false
}
