package eval

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func resetGoCacheVars() {
	goCacheDirOnce = sync.Once{}
	goCacheDir = ""
}

func TestParseGoTestOutputParsesMetrics(t *testing.T) {
	output := `
--- PASS: TestFoo
--- FAIL: TestBar
--- SKIP: TestBaz
coverage: 88.5% of statements
`

	result := parseGoTestOutput(output)
	if result.Total != 3 {
		t.Fatalf("expected 3 parsed tests, got %d", result.Total)
	}
	if result.Failed != 1 {
		t.Fatalf("expected 1 failed test, got %d", result.Failed)
	}
	if result.Skipped != 1 {
		t.Fatalf("expected 1 skipped test, got %d", result.Skipped)
	}
	if result.Coverage <= 0.0 {
		t.Fatalf("expected coverage to be parsed, got %f", result.Coverage)
	}
}

func TestCountLintIssuesMatches(t *testing.T) {
	output := `
file.go:10:5: something wrong
other.go:20:10: another issue
not an issue line
`
	count := countLintIssues(output)
	if count != 2 {
		t.Fatalf("expected 2 lint issues, got %d", count)
	}
}

func TestCountSecrets(t *testing.T) {
	output := `
Finding: AWS_SECRET_KEY
Finding: API_TOKEN
`
	if got := countSecrets(output); got != 2 {
		t.Fatalf("expected 2 secrets, got %d", got)
	}
}

func TestCountVulnerabilities(t *testing.T) {
	withNumbers := "Vulnerability #1\nVulnerability #2"
	if got := countVulnerabilities(withNumbers); got != 2 {
		t.Fatalf("expected 2 numbered vulnerabilities, got %d", got)
	}

	withFoundIn := "found in: module1\nfound in: module2"
	if got := countVulnerabilities(withFoundIn); got != 2 {
		t.Fatalf("expected 2 found-in vulnerabilities, got %d", got)
	}
}

func TestPrepareGoCommandInjectsGoCache(t *testing.T) {
	resetGoCacheVars()
	cmd := exec.Command("true")
	prepareGoCommand(cmd)
	found := false
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "GOCACHE=") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected GOCACHE to be injected into environment")
	}
}

func TestRunGoTestsAndCheckCoverage(t *testing.T) {
	stubDir := setupStubPath(t)
	writeGoStub(t, stubDir)

	t.Setenv("SPECULAR_GO_SIM", "go_test")
	resetGoCacheVars()
	result, err := RunGoTests(".", nil)
	if err != nil {
		t.Fatalf("RunGoTests returned error: %v", err)
	}
	if !result.Passed {
		t.Fatalf("expected RunGoTests to pass, got %+v", result)
	}
	t.Logf("RunGoTests output: %q", result.Output)
	if result.Coverage < 0.88 {
		t.Fatalf("expected coverage >= 0.88, got %f", result.Coverage)
	}

	t.Setenv("SPECULAR_GO_SIM", "go_check")
	checkResult, err := CheckCoverage(".", 0.6)
	if err != nil {
		t.Fatalf("CheckCoverage returned error: %v", err)
	}
	if !checkResult.Passed {
		t.Fatalf("expected coverage check to pass, got %+v", checkResult)
	}

	t.Setenv("SPECULAR_GO_SIM", "go_check")
	failureResult, err := CheckCoverage(".", 0.9)
	if err != nil {
		t.Fatalf("CheckCoverage (fail) returned error: %v", err)
	}
	if failureResult.Passed {
		t.Fatalf("expected coverage check to fail for high threshold, got %+v", failureResult)
	}
}

func TestRunGoTestsHandlesCommandError(t *testing.T) {
	stubDir := setupStubPath(t)
	writeGoStub(t, stubDir)
	t.Setenv("SPECULAR_GO_SIM", "go_fail")

	result, err := RunGoTests(".", nil)
	if err != nil {
		t.Fatalf("RunGoTests error = %v", err)
	}
	if result.Passed {
		t.Fatalf("expected RunGoTests to report failure, got %+v", result)
	}
}

func TestCheckCoverageHandlesCommandError(t *testing.T) {
	stubDir := setupStubPath(t)
	writeGoStub(t, stubDir)
	t.Setenv("SPECULAR_GO_SIM", "go_fail")

	result, err := CheckCoverage(".", 0.0)
	if err != nil {
		t.Fatalf("CheckCoverage error = %v", err)
	}
	if result.Passed {
		t.Fatalf("expected CheckCoverage to report failure, got %+v", result)
	}
}

func TestRunLinterRecordsIssues(t *testing.T) {
	stubDir := setupStubPath(t)
	script := `#!/bin/sh
printf 'file.go:1:1: issue\n'
exit 1
`
	writeScript(t, stubDir, "linter", script)

	result, err := RunLinter(".", "linter")
	if err != nil {
		t.Fatalf("RunLinter returned error: %v", err)
	}
	if result.Passed {
		t.Fatalf("expected linter to fail, got %+v", result)
	}
	if result.Issues != 1 {
		t.Fatalf("expected 1 issue, got %d", result.Issues)
	}
}

func TestRunLinterCommandMissing(t *testing.T) {
	result, err := RunLinter(".", "missing-linter")
	if err != nil {
		t.Fatalf("RunLinter error = %v", err)
	}
	if result.Passed {
		t.Fatalf("expected RunLinter to fail for missing command, got %+v", result)
	}
	if !strings.Contains(result.Output, "failed to prepare linter") {
		t.Fatalf("unexpected output for missing linter: %s", result.Output)
	}
}

func TestRunSecretsScanCountsFindings(t *testing.T) {
	stubDir := setupStubPath(t)
	script := `#!/bin/sh
printf 'Finding: SECRET\n'
exit 1
`
	writeScript(t, stubDir, "gitleaks", script)

	result, err := RunSecretsScan(".")
	if err != nil {
		t.Fatalf("RunSecretsScan returned error: %v", err)
	}
	if result.Passed {
		t.Fatalf("expected secrets scan to fail, got %+v", result)
	}
	if result.Secrets != 1 {
		t.Fatalf("expected 1 secret, got %d", result.Secrets)
	}
}

func TestRunSecretsScanCommandMissing(t *testing.T) {
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	result, err := RunSecretsScan(".")
	if err != nil {
		t.Fatalf("RunSecretsScan error = %v", err)
	}
	if !result.Passed {
		t.Fatalf("expected secrets scan to pass when command missing, got %+v", result)
	}
	if !strings.Contains(result.Output, "gitleaks not found") {
		t.Fatalf("unexpected output: %s", result.Output)
	}
}

func TestRunDependencyScanReportsVulnerabilities(t *testing.T) {
	stubDir := setupStubPath(t)
	script := `#!/bin/sh
printf 'Vulnerability #1\n'
exit 1
`
	writeScript(t, stubDir, "govulncheck", script)

	result, err := RunDependencyScan(".")
	if err != nil {
		t.Fatalf("RunDependencyScan returned error: %v", err)
	}
	if result.Passed {
		t.Fatalf("expected dependency scan to fail when vulnerabilities found, got %+v", result)
	}
	if result.Vulnerabilities != 1 {
		t.Fatalf("expected 1 vulnerability, got %d", result.Vulnerabilities)
	}
}

func TestRunDependencyScanCommandMissing(t *testing.T) {
	stubDir := setupStubPath(t)
	writeGoStub(t, stubDir)
	t.Setenv("PATH", stubDir)
	t.Setenv("SPECULAR_GO_SIM", "go_list")

	result, err := RunDependencyScan(".")
	if err != nil {
		t.Fatalf("RunDependencyScan error = %v", err)
	}
	if !result.Passed {
		t.Fatalf("expected dependency scan to pass when govulncheck missing, got %+v", result)
	}
	if !strings.Contains(result.Output, "govulncheck not found") {
		t.Fatalf("unexpected output: %s", result.Output)
	}
}

func setupStubPath(t *testing.T) string {
	stubDir := t.TempDir()
	orig := os.Getenv("PATH")
	newPath := stubDir
	if orig != "" {
		newPath = stubDir + string(os.PathListSeparator) + orig
	}
	t.Setenv("PATH", newPath)
	return stubDir
}

func writeScript(t *testing.T, dir, name, body string) {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("failed to write script %s: %v", name, err)
	}
}

func writeGoStub(t *testing.T, dir string) {
	body := `#!/bin/sh
case "$SPECULAR_GO_SIM" in
  go_test)
    printf '%s\n' '--- PASS: TestStub' 'coverage: 88.8% of statements'
    exit 0
    ;;
  go_check)
    printf '%s\n' '--- PASS: Coverage' 'coverage: 65.0% of statements'
    exit 0
    ;;
  go_fail)
    printf '%s\n' '--- FAIL: TestFail' 'coverage: 0.0% of statements'
    exit 1
    ;;
  go_list)
    printf '%s\n' 'module stub'
    exit 0
    ;;
esac
printf '%s\n' '--- PASS: Default' 'coverage: 100.0% of statements'
exit 0
`
	writeScript(t, dir, "go", body)
}
