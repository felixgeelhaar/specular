package eval

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/specular/internal/policy"
)

// GateOptions configures the eval gate
type GateOptions struct {
	Policy      *policy.Policy
	ProjectRoot string
	Verbose     bool
}

// RunEvalGate executes all quality checks based on policy
func RunEvalGate(opts GateOptions) (*GateReport, error) {
	if opts.Policy == nil {
		return nil, fmt.Errorf("policy is required")
	}

	startTime := time.Now()
	var checks []CheckResult

	// Run tests if policy requires
	if opts.Policy.Tests.RequirePass {
		testCheck := runTests(opts)
		checks = append(checks, testCheck)
	}

	// Check test coverage if minimum is set
	if opts.Policy.Tests.MinCoverage > 0 {
		coverageCheck := checkCoverage(opts)
		checks = append(checks, coverageCheck)
	}

	// Run linters
	lintChecks := runLinters(opts)
	checks = append(checks, lintChecks...)

	// Run security scans
	if opts.Policy.Security.SecretsScan {
		secretsCheck := runSecretscan(opts)
		checks = append(checks, secretsCheck)
	}

	if opts.Policy.Security.DepScan {
		depCheck := runDependencyScan(opts)
		checks = append(checks, depCheck)
	}

	// Calculate summary
	report := &GateReport{
		Checks:   checks,
		Duration: time.Since(startTime),
	}

	for _, check := range checks {
		if check.Passed {
			report.TotalPassed++
		} else if check.Required {
			report.TotalFailed++
		} else {
			report.TotalSkipped++
		}
	}

	report.AllPassed = report.TotalFailed == 0

	return report, nil
}

// runTests executes the test suite
func runTests(opts GateOptions) CheckResult {
	startTime := time.Now()

	result, err := RunGoTests(opts.ProjectRoot, opts.Policy)
	if err != nil {
		return CheckResult{
			Name:     "Tests",
			Passed:   false,
			Message:  fmt.Sprintf("Test execution failed: %v", err),
			Details:  "",
			Duration: time.Since(startTime),
			Required: opts.Policy.Tests.RequirePass,
		}
	}

	message := fmt.Sprintf("Ran %d tests: %d failed, %d skipped",
		result.Total, result.Failed, result.Skipped)

	if result.Coverage > 0 {
		message += fmt.Sprintf(" (coverage: %.1f%%)", result.Coverage*100)
	}

	return CheckResult{
		Name:     "Tests",
		Passed:   result.Passed,
		Message:  message,
		Details:  result.Output,
		Duration: time.Since(startTime),
		Required: opts.Policy.Tests.RequirePass,
	}
}

// checkCoverage validates test coverage meets minimum threshold
func checkCoverage(opts GateOptions) CheckResult {
	startTime := time.Now()

	result, err := CheckCoverage(opts.ProjectRoot, opts.Policy.Tests.MinCoverage)
	if err != nil {
		return CheckResult{
			Name:     "Coverage",
			Passed:   false,
			Message:  fmt.Sprintf("Coverage check failed: %v", err),
			Duration: time.Since(startTime),
			Required: true,
		}
	}

	message := fmt.Sprintf("Coverage: %.1f%% (minimum: %.1f%%)",
		result.Coverage*100, opts.Policy.Tests.MinCoverage*100)

	return CheckResult{
		Name:     "Coverage",
		Passed:   result.Passed,
		Message:  message,
		Details:  result.Output,
		Duration: time.Since(startTime),
		Required: true,
	}
}

// runLinters executes configured linters
func runLinters(opts GateOptions) []CheckResult {
	var checks []CheckResult

	// Go linter
	if linter, ok := opts.Policy.Linters["go"]; ok && linter.Enabled {
		check := runGoLinter(opts)
		checks = append(checks, check)
	}

	// JavaScript linter
	if linter, ok := opts.Policy.Linters["javascript"]; ok && linter.Enabled {
		check := runJavaScriptLinter(opts)
		checks = append(checks, check)
	}

	// Python linter
	if linter, ok := opts.Policy.Linters["python"]; ok && linter.Enabled {
		check := runPythonLinter(opts)
		checks = append(checks, check)
	}

	return checks
}

// runGoLinter executes Go linting
//
//nolint:dupl // Linter patterns are similar by design, extraction would reduce clarity
func runGoLinter(opts GateOptions) CheckResult {
	startTime := time.Now()

	linter := opts.Policy.Linters["go"]
	result, err := RunLinter(opts.ProjectRoot, linter.Cmd)
	if err != nil {
		return CheckResult{
			Name:     "Go Lint",
			Passed:   false,
			Message:  fmt.Sprintf("Linter execution failed: %v", err),
			Duration: time.Since(startTime),
			Required: linter.Enabled,
		}
	}

	message := "No issues found"
	if result.Issues > 0 {
		message = fmt.Sprintf("Found %d issues", result.Issues)
	}

	return CheckResult{
		Name:     "Go Lint",
		Passed:   result.Passed,
		Message:  message,
		Details:  result.Output,
		Duration: time.Since(startTime),
		Required: linter.Enabled,
	}
}

// runJavaScriptLinter executes JavaScript linting
func runJavaScriptLinter(opts GateOptions) CheckResult {
	startTime := time.Now()

	linter := opts.Policy.Linters["javascript"]
	result, err := RunLinter(opts.ProjectRoot, linter.Cmd)
	if err != nil {
		return CheckResult{
			Name:     "JavaScript Lint",
			Passed:   false,
			Message:  fmt.Sprintf("Linter execution failed: %v", err),
			Duration: time.Since(startTime),
			Required: linter.Enabled,
		}
	}

	message := "No issues found"
	if result.Issues > 0 {
		message = fmt.Sprintf("Found %d issues", result.Issues)
	}

	return CheckResult{
		Name:     "JavaScript Lint",
		Passed:   result.Passed,
		Message:  message,
		Details:  result.Output,
		Duration: time.Since(startTime),
		Required: linter.Enabled,
	}
}

// runPythonLinter executes Python linting
func runPythonLinter(opts GateOptions) CheckResult {
	startTime := time.Now()

	linter := opts.Policy.Linters["python"]
	result, err := RunLinter(opts.ProjectRoot, linter.Cmd)
	if err != nil {
		return CheckResult{
			Name:     "Python Lint",
			Passed:   false,
			Message:  fmt.Sprintf("Linter execution failed: %v", err),
			Duration: time.Since(startTime),
			Required: linter.Enabled,
		}
	}

	message := "No issues found"
	if result.Issues > 0 {
		message = fmt.Sprintf("Found %d issues", result.Issues)
	}

	return CheckResult{
		Name:     "Python Lint",
		Passed:   result.Passed,
		Message:  message,
		Details:  result.Output,
		Duration: time.Since(startTime),
		Required: linter.Enabled,
	}
}

// runSecretscan scans for secrets
func runSecretscan(opts GateOptions) CheckResult {
	startTime := time.Now()

	result, err := RunSecretsScan(opts.ProjectRoot)
	if err != nil {
		return CheckResult{
			Name:     "Secrets Scan",
			Passed:   false,
			Message:  fmt.Sprintf("Secrets scan failed: %v", err),
			Duration: time.Since(startTime),
			Required: opts.Policy.Security.SecretsScan,
		}
	}

	message := "No secrets found"
	if result.Secrets > 0 {
		message = fmt.Sprintf("Found %d potential secrets", result.Secrets)
	}

	return CheckResult{
		Name:     "Secrets Scan",
		Passed:   result.Passed,
		Message:  message,
		Details:  result.Output,
		Duration: time.Since(startTime),
		Required: opts.Policy.Security.SecretsScan,
	}
}

// runDependencyScan scans for vulnerable dependencies
func runDependencyScan(opts GateOptions) CheckResult {
	startTime := time.Now()

	result, err := RunDependencyScan(opts.ProjectRoot)
	if err != nil {
		return CheckResult{
			Name:     "Dependency Scan",
			Passed:   false,
			Message:  fmt.Sprintf("Dependency scan failed: %v", err),
			Duration: time.Since(startTime),
			Required: opts.Policy.Security.DepScan,
		}
	}

	message := "No vulnerable dependencies found"
	if result.Vulnerabilities > 0 {
		message = fmt.Sprintf("Found %d vulnerabilities", result.Vulnerabilities)
	}

	return CheckResult{
		Name:     "Dependency Scan",
		Passed:   result.Passed,
		Message:  message,
		Details:  result.Output,
		Duration: time.Since(startTime),
		Required: opts.Policy.Security.DepScan,
	}
}
