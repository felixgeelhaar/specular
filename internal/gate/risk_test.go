package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if advisoryRiskLevel(factors) != "HIGH" {
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
	if res.Risk.Level != "HIGH" {
		t.Fatalf("level=%s factors=%v", res.Risk.Level, res.Risk.Factors)
	}

	text := FormatText(res)
	for _, want := range []string{
		"Risk",
		"Level          HIGH",
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
		"| Risk | `HIGH` (advisory) |",
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
	if !strings.Contains(text, "Level          MEDIUM") {
		t.Fatalf("missing level:\n%s", text)
	}
	if !strings.Contains(text, "+ "+FactorDependencyPaths) {
		t.Fatalf("missing factor:\n%s", text)
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
