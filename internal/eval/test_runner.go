package eval

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/felixgeelhaar/specular/internal/policy"
)

var (
	goCacheDirOnce sync.Once
	goCacheDir     string
)

// RunGoTests executes Go tests and returns results
func RunGoTests(projectRoot string, pol *policy.Policy) (*TestResult, error) {
	cmd := exec.Command("go", "test", "-v", "-race", "-coverprofile=coverage.txt", "-covermode=atomic", "./...")
	cmd.Dir = projectRoot
	prepareGoCommand(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String() + stderr.String()
	result := parseGoTestOutput(output)

	// Test failed if command returned error
	if err != nil {
		result.Passed = false
	} else {
		result.Passed = true
	}

	result.Output = output

	return result, nil
}

// parseGoTestOutput parses go test output to extract test results
func parseGoTestOutput(output string) *TestResult {
	result := &TestResult{}

	// Count test results
	// Look for patterns like: "--- PASS: TestName" or "--- FAIL: TestName"
	passRegex := regexp.MustCompile(`--- PASS:`)
	failRegex := regexp.MustCompile(`--- FAIL:`)
	skipRegex := regexp.MustCompile(`--- SKIP:`)

	passes := passRegex.FindAllString(output, -1)
	fails := failRegex.FindAllString(output, -1)
	skips := skipRegex.FindAllString(output, -1)

	result.Total = len(passes) + len(fails) + len(skips)
	result.Failed = len(fails)
	result.Skipped = len(skips)

	// Extract coverage if available
	// Look for: "coverage: XX.X% of statements"
	coverageRegex := regexp.MustCompile(`coverage:\s+(\d+\.?\d*)%`)
	if matches := coverageRegex.FindStringSubmatch(output); len(matches) > 1 {
		if cov, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Coverage = cov / 100.0 // Convert to decimal
		}
	}

	result.Passed = result.Failed == 0

	return result
}

// CheckCoverage verifies test coverage meets minimum threshold
func CheckCoverage(projectRoot string, minCoverage float64) (*TestResult, error) {
	// Run tests with coverage
	cmd := exec.Command("go", "test", "-coverprofile=coverage.txt", "-covermode=atomic", "./...")
	cmd.Dir = projectRoot
	prepareGoCommand(cmd)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	err := cmd.Run()

	output := stdout.String()
	result := parseGoTestOutput(output)

	// Check if coverage meets minimum
	if result.Coverage < minCoverage {
		result.Passed = false
		result.Output = fmt.Sprintf("Coverage %.1f%% below minimum %.1f%%\n%s",
			result.Coverage*100, minCoverage*100, output)
	} else {
		result.Passed = err == nil
		result.Output = output
	}

	return result, nil
}

// RunLinter executes a linter command
func RunLinter(projectRoot string, linterCmd string) (*LintResult, error) {
	// Split command into parts
	parts := strings.Fields(linterCmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty linter command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String() + stderr.String()

	result := &LintResult{
		Output: output,
	}

	// Linter passed if command succeeded
	if err != nil {
		result.Passed = false
		// Try to count issues from output
		result.Issues = countLintIssues(output)
	} else {
		result.Passed = true
		result.Issues = 0
	}

	return result, nil
}

// countLintIssues attempts to count issues in linter output
func countLintIssues(output string) int {
	// Count lines that look like linter issues
	// Most linters output: file:line:col: message
	issueRegex := regexp.MustCompile(`^\S+:\d+:\d+:`)
	lines := strings.Split(output, "\n")

	count := 0
	for _, line := range lines {
		if issueRegex.MatchString(strings.TrimSpace(line)) {
			count++
		}
	}

	return count
}

// RunSecretsScan scans for secrets in the codebase
func RunSecretsScan(projectRoot string) (*SecurityResult, error) {
	// Check if gitleaks is available
	if _, err := exec.LookPath("gitleaks"); err != nil {
		return &SecurityResult{
			Passed: true,
			Output: "gitleaks not found, skipping secrets scan",
		}, nil
	}

	cmd := exec.Command("gitleaks", "detect", "--no-git", "-v")
	cmd.Dir = projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String() + stderr.String()

	result := &SecurityResult{
		Output: output,
	}

	// gitleaks exits with 1 if secrets found
	if err != nil {
		result.Passed = false
		result.Secrets = countSecrets(output)
	} else {
		result.Passed = true
		result.Secrets = 0
	}

	return result, nil
}

// countSecrets counts secrets found in gitleaks output
func countSecrets(output string) int {
	// gitleaks outputs: "Finding: <secret-type>"
	findingRegex := regexp.MustCompile(`(?i)finding:`)
	return len(findingRegex.FindAllString(output, -1))
}

func prepareGoCommand(cmd *exec.Cmd) {
	cacheDir := ensureGoCacheDir()
	if cacheDir == "" {
		return
	}

	env := os.Environ()
	env = append(env, "GOCACHE="+cacheDir)
	cmd.Env = env
}

func ensureGoCacheDir() string {
	goCacheDirOnce.Do(func() {
		dir := filepath.Join(os.TempDir(), "specular-go-cache")
		if err := os.MkdirAll(dir, 0o755); err == nil {
			goCacheDir = dir
		}
	})
	return goCacheDir
}

// RunDependencyScan scans for vulnerable dependencies using govulncheck
func RunDependencyScan(projectRoot string) (*SecurityResult, error) {
	// Check if govulncheck is available
	if _, err := exec.LookPath("govulncheck"); err != nil {
		// Fall back to basic dependency listing if govulncheck not available
		cmd := exec.Command("go", "list", "-m", "all")
		cmd.Dir = projectRoot
		prepareGoCommand(cmd)

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			return &SecurityResult{
				Passed: false,
				Output: fmt.Sprintf("Failed to list dependencies: %v", err),
			}, nil
		}

		return &SecurityResult{
			Passed: true,
			Output: "govulncheck not found, skipped vulnerability scan\n" + stdout.String(),
		}, nil
	}

	// Run govulncheck to scan for vulnerabilities
	cmd := exec.Command("govulncheck", "./...")
	cmd.Dir = projectRoot
	prepareGoCommand(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String() + stderr.String()

	result := &SecurityResult{
		Output: output,
	}

	// govulncheck exits with non-zero if vulnerabilities found
	if err != nil {
		result.Passed = false
		result.Vulnerabilities = countVulnerabilities(output)
	} else {
		result.Passed = true
		result.Vulnerabilities = 0
	}

	return result, nil
}

// countVulnerabilities counts vulnerabilities found in govulncheck output
func countVulnerabilities(output string) int {
	// Count unique "Vulnerability #" markers (preferred format)
	vulnNumRegex := regexp.MustCompile(`(?i)vulnerability #\d+`)
	vulnMatches := vulnNumRegex.FindAllString(output, -1)

	if len(vulnMatches) > 0 {
		return len(vulnMatches)
	}

	// Fall back to counting "found in:" occurrences if no numbered vulnerabilities
	foundInRegex := regexp.MustCompile(`(?i)found in:`)
	return len(foundInRegex.FindAllString(output, -1))
}
