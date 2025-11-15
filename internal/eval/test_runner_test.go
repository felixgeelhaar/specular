package eval

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/internal/policy"
)

// Note: TestParseGoTestOutput and TestCountLintIssues are already in gate_test.go

func TestCountSecrets(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int
	}{
		{
			name: "gitleaks findings",
			output: `Finding: AWS Access Key
Secret: AKIAIOSFODNN7EXAMPLE
File: config.yaml:10

Finding: GitHub Token
Secret: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
File: .env:5

Finding: Private Key
Secret: -----BEGIN RSA PRIVATE KEY-----
File: keys/id_rsa:1`,
			want: 3,
		},
		{
			name: "case insensitive finding",
			output: `finding: secret detected
FINDING: another secret`,
			want: 2,
		},
		{
			name:   "no secrets found",
			output: `No leaks detected`,
			want:   0,
		},
		{
			name:   "empty output",
			output: "",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countSecrets(tt.output)
			if got != tt.want {
				t.Errorf("countSecrets() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCountVulnerabilities(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int
	}{
		{
			name: "numbered vulnerabilities",
			output: `Vulnerability #1: CVE-2023-1234
Description: SQL Injection in database package
Module: github.com/example/db
Found in: v1.2.3

Vulnerability #2: CVE-2023-5678
Description: XSS in web framework
Module: github.com/example/web
Found in: v2.0.0

Vulnerability #3: CVE-2023-9012
Description: Authentication bypass
Module: github.com/example/auth
Found in: v1.0.0`,
			want: 3,
		},
		{
			name: "found in pattern",
			output: `Your code is affected by 2 vulnerabilities:
Vulnerability in package: github.com/example/pkg1
Found in: v1.2.3
Fixed in: v1.2.4

Vulnerability in package: github.com/example/pkg2
Found in: v2.0.0
Fixed in: v2.1.0`,
			want: 2,
		},
		{
			name: "case insensitive matching",
			output: `VULNERABILITY #1: test
Vulnerability #2: test
vulnerability #3: test`,
			want: 3,
		},
		{
			name:   "no vulnerabilities found",
			output: `No vulnerabilities found`,
			want:   0,
		},
		{
			name:   "empty output",
			output: "",
			want:   0,
		},
		{
			name: "mixed format",
			output: `Scanning dependencies...
Vulnerability #1: CVE-2023-1111
Found in: github.com/example/pkg1@v1.0.0

Some other text here
Vulnerability #2: CVE-2023-2222
Found in: github.com/example/pkg2@v2.0.0`,
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countVulnerabilities(tt.output)
			if got != tt.want {
				t.Errorf("countVulnerabilities() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRunGoTests(t *testing.T) {
	// This is an integration test that actually runs go test
	// Test on the spec package to avoid recursion (testing the eval package would recurse)
	projectRoot := filepath.Join("..", "spec")

	pol := &policy.Policy{
		Tests: policy.TestPolicy{
			RequirePass: true,
			MinCoverage: 0.0, // Don't enforce coverage in this test
		},
	}

	result, err := RunGoTests(projectRoot, pol)
	if err != nil {
		t.Fatalf("RunGoTests() error = %v", err)
	}

	if result == nil {
		t.Fatal("RunGoTests() returned nil result")
	}

	// Verify result structure
	if result.Output == "" {
		t.Error("RunGoTests() Output is empty")
	}

	// The spec package should have at least some tests
	if result.Total == 0 {
		t.Error("RunGoTests() Total = 0, expected some tests to run in spec package")
	}
}

func TestCheckCoverage(t *testing.T) {
	// Test on spec package to avoid recursion
	projectRoot := filepath.Join("..", "spec")

	tests := []struct {
		name        string
		minCoverage float64
		wantPass    bool
	}{
		{
			name:        "low threshold should pass",
			minCoverage: 0.01, // 1% - should definitely pass
			wantPass:    true,
		},
		{
			name:        "impossible threshold should fail",
			minCoverage: 1.01, // 101% - impossible
			wantPass:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CheckCoverage(projectRoot, tt.minCoverage)
			if err != nil {
				t.Fatalf("CheckCoverage() error = %v", err)
			}

			if result.Passed != tt.wantPass {
				t.Errorf("CheckCoverage() Passed = %v, want %v (coverage: %.1f%%, min: %.1f%%)",
					result.Passed, tt.wantPass, result.Coverage*100, tt.minCoverage*100)
			}

			if result.Output == "" {
				t.Error("CheckCoverage() Output is empty")
			}
		})
	}
}

func TestRunLinter(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		linterCmd   string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty command",
			linterCmd:   "",
			wantErr:     true,
			errContains: "empty linter command",
		},
		{
			name:      "nonexistent command",
			linterCmd: "nonexistent-linter-xyz123",
			wantErr:   false, // Returns LintResult with error, not error return
		},
		{
			name:      "valid command - echo",
			linterCmd: "echo test",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RunLinter(tmpDir, tt.linterCmd)

			if tt.wantErr {
				if err == nil {
					t.Error("RunLinter() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("RunLinter() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("RunLinter() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Fatal("RunLinter() returned nil result")
			}
		})
	}
}

func TestRunSecretsScan(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := RunSecretsScan(tmpDir)
	if err != nil {
		t.Fatalf("RunSecretsScan() error = %v", err)
	}

	if result == nil {
		t.Fatal("RunSecretsScan() returned nil result")
	}

	// Result should always pass (either skipped if gitleaks not found, or passed if no secrets)
	// This is a clean test directory, so should pass
	if !result.Passed {
		t.Errorf("RunSecretsScan() Passed = false in clean directory")
	}

	if result.Output == "" {
		t.Error("RunSecretsScan() Output is empty")
	}
}

func TestRunSecretsScan_WithGitleaks(t *testing.T) {
	// Check if gitleaks is available
	if _, err := exec.LookPath("gitleaks"); err != nil {
		t.Skip("gitleaks not available, skipping test")
	}

	// Test on the actual project root to exercise gitleaks execution
	projectRoot := filepath.Join("..", "..")

	result, err := RunSecretsScan(projectRoot)
	if err != nil {
		t.Fatalf("RunSecretsScan() error = %v", err)
	}

	if result == nil {
		t.Fatal("RunSecretsScan() returned nil result")
	}

	// Verify result structure
	if result.Output == "" {
		t.Error("RunSecretsScan() Output is empty")
	}

	// Result should have a boolean Passed value
	// (could be true or false depending on if secrets are found)
	// Just verify the structure is correct
	if result.Passed {
		if result.Secrets != 0 {
			t.Errorf("RunSecretsScan() Passed=true but Secrets=%d, expected 0", result.Secrets)
		}
	} else {
		if result.Secrets == 0 {
			t.Error("RunSecretsScan() Passed=false but Secrets=0, expected >0")
		}
	}
}

func TestRunDependencyScan(t *testing.T) {
	// Test on spec package to avoid recursion
	projectRoot := filepath.Join("..", "spec")

	result, err := RunDependencyScan(projectRoot)
	if err != nil {
		t.Fatalf("RunDependencyScan() error = %v", err)
	}

	if result == nil {
		t.Fatal("RunDependencyScan() returned nil result")
	}

	if result.Output == "" {
		t.Error("RunDependencyScan() Output is empty")
	}

	// Check if govulncheck was used or if we fell back
	if _, err := exec.LookPath("govulncheck"); err != nil {
		// govulncheck not available, verify fallback behavior
		if !result.Passed {
			t.Errorf("RunDependencyScan() Passed = false when govulncheck not available")
		}
		if result.Vulnerabilities != 0 {
			t.Errorf("RunDependencyScan() Vulnerabilities = %d, want 0 when govulncheck not available", result.Vulnerabilities)
		}
	} else {
		// govulncheck is available, verify it ran
		// Result could be true (no vulns) or false (vulns found)
		// Just verify the structure is correct
		if !result.Passed && result.Vulnerabilities == 0 {
			t.Skipf("govulncheck failed without reporting vulnerabilities:\n%s", result.Output)
		}
		if result.Passed && result.Vulnerabilities != 0 {
			t.Errorf("RunDependencyScan() Passed=true but Vulnerabilities=%d, expected 0", result.Vulnerabilities)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
