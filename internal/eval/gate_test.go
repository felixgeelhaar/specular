package eval

import (
	"testing"

	"github.com/felixgeelhaar/specular/internal/policy"
)

func TestRunEvalGate(t *testing.T) {
	tests := []struct {
		name          string
		opts          GateOptions
		wantErr       bool
		wantAllPassed bool
	}{
		{
			name: "no policy",
			opts: GateOptions{
				Policy:      nil,
				ProjectRoot: ".",
			},
			wantErr:       true,
			wantAllPassed: false,
		},
		{
			name: "minimal policy - all checks disabled",
			opts: GateOptions{
				Policy: &policy.Policy{
					Tests: policy.TestPolicy{
						RequirePass: false,
						MinCoverage: 0,
					},
					Linters: map[string]policy.ToolConfig{
						"go":         {Enabled: false},
						"javascript": {Enabled: false},
						"python":     {Enabled: false},
					},
					Security: policy.SecurityPolicy{
						SecretsScan: false,
						DepScan:     false,
					},
				},
				ProjectRoot: ".",
			},
			wantErr:       false,
			wantAllPassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := RunEvalGate(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunEvalGate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if report.AllPassed != tt.wantAllPassed {
				t.Errorf("RunEvalGate() AllPassed = %v, want %v", report.AllPassed, tt.wantAllPassed)
				t.Logf("Report: %d passed, %d failed, %d skipped", report.TotalPassed, report.TotalFailed, report.TotalSkipped)
				for _, check := range report.Checks {
					t.Logf("  %s: passed=%v, message=%s", check.Name, check.Passed, check.Message)
				}
			}
		})
	}
}

func TestParseGoTestOutput(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		wantPassed   bool
		wantFailed   int
		wantCoverage float64
	}{
		{
			name: "all tests passed with coverage",
			output: `=== RUN   TestFoo
--- PASS: TestFoo (0.00s)
=== RUN   TestBar
--- PASS: TestBar (0.00s)
PASS
coverage: 85.8% of statements
ok  	example.com/pkg	1.234s	coverage: 85.8% of statements`,
			wantPassed:   true,
			wantFailed:   0,
			wantCoverage: 0.858,
		},
		{
			name: "some tests failed",
			output: `=== RUN   TestFoo
--- PASS: TestFoo (0.00s)
=== RUN   TestBar
--- FAIL: TestBar (0.00s)
FAIL
coverage: 50.0% of statements
ok  	example.com/pkg	1.234s	coverage: 50.0% of statements`,
			wantPassed:   false,
			wantFailed:   1,
			wantCoverage: 0.50,
		},
		{
			name: "no coverage",
			output: `=== RUN   TestFoo
--- PASS: TestFoo (0.00s)
PASS
ok  	example.com/pkg	1.234s`,
			wantPassed:   true,
			wantFailed:   0,
			wantCoverage: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGoTestOutput(tt.output)

			if result.Passed != tt.wantPassed {
				t.Errorf("parseGoTestOutput() Passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if result.Failed != tt.wantFailed {
				t.Errorf("parseGoTestOutput() Failed = %d, want %d", result.Failed, tt.wantFailed)
			}

			// Check coverage with small tolerance for floating point
			if diff := result.Coverage - tt.wantCoverage; diff > 0.001 || diff < -0.001 {
				t.Errorf("parseGoTestOutput() Coverage = %f, want %f", result.Coverage, tt.wantCoverage)
			}
		})
	}
}

func TestCountLintIssues(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int
	}{
		{
			name:   "no issues",
			output: "",
			want:   0,
		},
		{
			name: "golangci-lint format",
			output: `file.go:10:5: some issue (linter)
file.go:20:10: another issue (linter)
file.go:30:1: third issue (linter)`,
			want: 3,
		},
		{
			name: "eslint format",
			output: `/path/to/file.js:10:5: error message
/path/to/file.js:20:10: warning message`,
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countLintIssues(tt.output)
			if got != tt.want {
				t.Errorf("countLintIssues() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRunEvalGate_WithTestsEnabled(t *testing.T) {
	// Test on spec package which has good test coverage
	opts := GateOptions{
		Policy: &policy.Policy{
			Tests: policy.TestPolicy{
				RequirePass: true,
				MinCoverage: 0.50, // Lower threshold since we're testing
			},
			Linters: map[string]policy.ToolConfig{
				"go":         {Enabled: false},
				"javascript": {Enabled: false},
				"python":     {Enabled: false},
			},
			Security: policy.SecurityPolicy{
				SecretsScan: false,
				DepScan:     false,
			},
		},
		ProjectRoot: "../spec", // Test on spec package
	}

	report, err := RunEvalGate(opts)
	if err != nil {
		t.Fatalf("RunEvalGate() unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("RunEvalGate() returned nil report")
	}

	// Should have test check
	if len(report.Checks) == 0 {
		t.Error("RunEvalGate() returned no checks")
	}

	// Find test check
	var testCheck *CheckResult
	for i := range report.Checks {
		if report.Checks[i].Name == "Tests" {
			testCheck = &report.Checks[i]
			break
		}
	}

	if testCheck == nil {
		t.Error("RunEvalGate() did not include Tests check")
	}
}

func TestRunEvalGate_WithCoverageCheck(t *testing.T) {
	opts := GateOptions{
		Policy: &policy.Policy{
			Tests: policy.TestPolicy{
				RequirePass: false,
				MinCoverage: 0.50, // 50% minimum coverage
			},
			Linters: map[string]policy.ToolConfig{
				"go":         {Enabled: false},
				"javascript": {Enabled: false},
				"python":     {Enabled: false},
			},
			Security: policy.SecurityPolicy{
				SecretsScan: false,
				DepScan:     false,
			},
		},
		ProjectRoot: "../spec", // Test on spec package
	}

	report, err := RunEvalGate(opts)
	if err != nil {
		t.Fatalf("RunEvalGate() unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("RunEvalGate() returned nil report")
	}

	// Find coverage check
	var coverageCheck *CheckResult
	for i := range report.Checks {
		if report.Checks[i].Name == "Coverage" {
			coverageCheck = &report.Checks[i]
			break
		}
	}

	if coverageCheck == nil {
		t.Error("RunEvalGate() did not include Coverage check")
	}
}

func TestRunEvalGate_WithGoLinter(t *testing.T) {
	opts := GateOptions{
		Policy: &policy.Policy{
			Tests: policy.TestPolicy{
				RequirePass: false,
				MinCoverage: 0,
			},
			Linters: map[string]policy.ToolConfig{
				"go": {
					Enabled: true,
					Cmd:     "go vet ./...",
				},
				"javascript": {Enabled: false},
				"python":     {Enabled: false},
			},
			Security: policy.SecurityPolicy{
				SecretsScan: false,
				DepScan:     false,
			},
		},
		ProjectRoot: ".", // Test on eval package itself
	}

	report, err := RunEvalGate(opts)
	if err != nil {
		t.Fatalf("RunEvalGate() unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("RunEvalGate() returned nil report")
	}

	// Find Go lint check
	var lintCheck *CheckResult
	for i := range report.Checks {
		if report.Checks[i].Name == "Go Lint" {
			lintCheck = &report.Checks[i]
			break
		}
	}

	if lintCheck == nil {
		t.Error("RunEvalGate() did not include Go Lint check")
	}
}

func TestRunEvalGate_WithSecretsScan(t *testing.T) {
	opts := GateOptions{
		Policy: &policy.Policy{
			Tests: policy.TestPolicy{
				RequirePass: false,
				MinCoverage: 0,
			},
			Linters: map[string]policy.ToolConfig{
				"go":         {Enabled: false},
				"javascript": {Enabled: false},
				"python":     {Enabled: false},
			},
			Security: policy.SecurityPolicy{
				SecretsScan: true,
				DepScan:     false,
			},
		},
		ProjectRoot: ".", // Test on eval package itself
	}

	report, err := RunEvalGate(opts)
	if err != nil {
		t.Fatalf("RunEvalGate() unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("RunEvalGate() returned nil report")
	}

	// Find secrets scan check
	var secretsCheck *CheckResult
	for i := range report.Checks {
		if report.Checks[i].Name == "Secrets Scan" {
			secretsCheck = &report.Checks[i]
			break
		}
	}

	if secretsCheck == nil {
		t.Error("RunEvalGate() did not include Secrets Scan check")
	}
}

func TestRunEvalGate_WithDependencyScan(t *testing.T) {
	opts := GateOptions{
		Policy: &policy.Policy{
			Tests: policy.TestPolicy{
				RequirePass: false,
				MinCoverage: 0,
			},
			Linters: map[string]policy.ToolConfig{
				"go":         {Enabled: false},
				"javascript": {Enabled: false},
				"python":     {Enabled: false},
			},
			Security: policy.SecurityPolicy{
				SecretsScan: false,
				DepScan:     true,
			},
		},
		ProjectRoot: ".", // Test on eval package itself
	}

	report, err := RunEvalGate(opts)
	if err != nil {
		t.Fatalf("RunEvalGate() unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("RunEvalGate() returned nil report")
	}

	// Find dependency scan check
	var depCheck *CheckResult
	for i := range report.Checks {
		if report.Checks[i].Name == "Dependency Scan" {
			depCheck = &report.Checks[i]
			break
		}
	}

	if depCheck == nil {
		t.Error("RunEvalGate() did not include Dependency Scan check")
	}
}

func TestRunEvalGate_MultipleChecks(t *testing.T) {
	// Test with multiple checks enabled
	opts := GateOptions{
		Policy: &policy.Policy{
			Tests: policy.TestPolicy{
				RequirePass: true,
				MinCoverage: 0.40,
			},
			Linters: map[string]policy.ToolConfig{
				"go": {
					Enabled: true,
					Cmd:     "go vet ./...",
				},
				"javascript": {Enabled: false},
				"python":     {Enabled: false},
			},
			Security: policy.SecurityPolicy{
				SecretsScan: true,
				DepScan:     true,
			},
		},
		ProjectRoot: "../spec", // Test on spec package
	}

	report, err := RunEvalGate(opts)
	if err != nil {
		t.Fatalf("RunEvalGate() unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("RunEvalGate() returned nil report")
	}

	// Should have multiple checks
	expectedChecks := []string{"Tests", "Coverage", "Go Lint", "Secrets Scan", "Dependency Scan"}
	for _, name := range expectedChecks {
		found := false
		for _, check := range report.Checks {
			if check.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RunEvalGate() missing expected check: %s", name)
		}
	}

	// Report should have summary
	total := report.TotalPassed + report.TotalFailed + report.TotalSkipped
	if total != len(report.Checks) {
		t.Errorf("Report totals (%d passed + %d failed + %d skipped = %d) don't match check count %d",
			report.TotalPassed, report.TotalFailed, report.TotalSkipped, total, len(report.Checks))
	}
}

func TestRunEvalGate_WithJavaScriptLinter(t *testing.T) {
	opts := GateOptions{
		Policy: &policy.Policy{
			Tests: policy.TestPolicy{
				RequirePass: false,
			},
			Linters: map[string]policy.ToolConfig{
				"go":         {Enabled: false},
				"javascript": {Enabled: true, Cmd: "echo 'No issues found'"}, // Simple command that succeeds
				"python":     {Enabled: false},
			},
			Security: policy.SecurityPolicy{
				SecretsScan: false,
				DepScan:     false,
			},
		},
		ProjectRoot: ".",
	}

	report, err := RunEvalGate(opts)
	if err != nil {
		t.Fatalf("RunEvalGate() unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("RunEvalGate() returned nil report")
	}

	// Should have JavaScript Lint check
	found := false
	for _, check := range report.Checks {
		if check.Name == "JavaScript Lint" {
			found = true
			if !check.Passed {
				t.Errorf("JavaScript Lint check should pass with simple echo command")
			}
			break
		}
	}
	if !found {
		t.Error("RunEvalGate() missing JavaScript Lint check")
	}
}

func TestRunEvalGate_WithPythonLinter(t *testing.T) {
	opts := GateOptions{
		Policy: &policy.Policy{
			Tests: policy.TestPolicy{
				RequirePass: false,
			},
			Linters: map[string]policy.ToolConfig{
				"go":         {Enabled: false},
				"javascript": {Enabled: false},
				"python":     {Enabled: true, Cmd: "echo 'No issues found'"}, // Simple command that succeeds
			},
			Security: policy.SecurityPolicy{
				SecretsScan: false,
				DepScan:     false,
			},
		},
		ProjectRoot: ".",
	}

	report, err := RunEvalGate(opts)
	if err != nil {
		t.Fatalf("RunEvalGate() unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("RunEvalGate() returned nil report")
	}

	// Should have Python Lint check
	found := false
	for _, check := range report.Checks {
		if check.Name == "Python Lint" {
			found = true
			if !check.Passed {
				t.Errorf("Python Lint check should pass with simple echo command")
			}
			break
		}
	}
	if !found {
		t.Error("RunEvalGate() missing Python Lint check")
	}
}

func TestRunEvalGate_WithAllLinters(t *testing.T) {
	opts := GateOptions{
		Policy: &policy.Policy{
			Tests: policy.TestPolicy{
				RequirePass: false,
			},
			Linters: map[string]policy.ToolConfig{
				"go":         {Enabled: true, Cmd: "echo 'No issues'"},
				"javascript": {Enabled: true, Cmd: "echo 'No issues'"},
				"python":     {Enabled: true, Cmd: "echo 'No issues'"},
			},
			Security: policy.SecurityPolicy{
				SecretsScan: false,
				DepScan:     false,
			},
		},
		ProjectRoot: ".",
	}

	report, err := RunEvalGate(opts)
	if err != nil {
		t.Fatalf("RunEvalGate() unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("RunEvalGate() returned nil report")
	}

	// Should have all three linter checks
	expectedLinters := []string{"Go Lint", "JavaScript Lint", "Python Lint"}
	for _, name := range expectedLinters {
		found := false
		for _, check := range report.Checks {
			if check.Name == name {
				found = true
				if !check.Passed {
					t.Errorf("%s check should pass with simple echo command", name)
				}
				break
			}
		}
		if !found {
			t.Errorf("RunEvalGate() missing %s check", name)
		}
	}
}

func TestRunEvalGate_WithFailingLinter(t *testing.T) {
	opts := GateOptions{
		Policy: &policy.Policy{
			Tests: policy.TestPolicy{
				RequirePass: false,
			},
			Linters: map[string]policy.ToolConfig{
				"go":         {Enabled: true, Cmd: "false"}, // Command that always fails
				"javascript": {Enabled: false},
				"python":     {Enabled: false},
			},
			Security: policy.SecurityPolicy{
				SecretsScan: false,
				DepScan:     false,
			},
		},
		ProjectRoot: ".",
	}

	report, err := RunEvalGate(opts)
	if err != nil {
		t.Fatalf("RunEvalGate() unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("RunEvalGate() returned nil report")
	}

	// Should have Go Lint check that failed
	var lintCheck *CheckResult
	for i := range report.Checks {
		if report.Checks[i].Name == "Go Lint" {
			lintCheck = &report.Checks[i]
			break
		}
	}

	if lintCheck == nil {
		t.Fatal("RunEvalGate() missing Go Lint check")
	}

	if lintCheck.Passed {
		t.Error("Go Lint check should fail with 'false' command")
	}

	if !lintCheck.Required {
		t.Error("Go Lint check should be required when enabled")
	}

	// Report should have the failed check in TotalFailed
	if report.TotalFailed == 0 {
		t.Error("Report should have at least one failed check")
	}

	if report.AllPassed {
		t.Error("Report.AllPassed should be false when a check fails")
	}
}
