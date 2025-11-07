package eval

import "time"

// CheckResult represents the result of a single quality check
type CheckResult struct {
	Name     string        // Name of the check (e.g., "go test", "golangci-lint")
	Passed   bool          // Whether the check passed
	Message  string        // Summary message
	Details  string        // Detailed output
	Duration time.Duration // How long the check took
	Required bool          // Whether this check is required by policy
}

// GateReport contains all quality check results
type GateReport struct {
	Checks       []CheckResult
	TotalPassed  int
	TotalFailed  int
	TotalSkipped int
	AllPassed    bool
	Duration     time.Duration
}

// TestResult represents test execution results
type TestResult struct {
	Passed   bool
	Failed   int
	Skipped  int
	Total    int
	Coverage float64
	Output   string
}

// LintResult represents linting results
type LintResult struct {
	Passed bool
	Issues int
	Output string
}

// SecurityResult represents security scan results
type SecurityResult struct {
	Passed          bool
	Secrets         int
	Vulnerabilities int
	Output          string
}
