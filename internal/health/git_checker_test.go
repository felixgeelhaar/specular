package health

import (
	"context"
	"testing"
)

func TestNewGitChecker(t *testing.T) {
	checker := NewGitChecker()

	if checker == nil {
		t.Fatal("NewGitChecker returned nil")
	}
}

func TestGitCheckerName(t *testing.T) {
	checker := NewGitChecker()

	name := checker.Name()
	if name != "git-binary" {
		t.Errorf("Name() = %q, want %q", name, "git-binary")
	}
}

func TestGitCheckerCheck(t *testing.T) {
	checker := NewGitChecker()
	ctx := context.Background()

	result := checker.Check(ctx)

	// We expect git to be installed on the test machine
	if result == nil {
		t.Fatal("Check() returned nil")
	}

	if result.Status == "" {
		t.Error("Status should not be empty")
	}

	if result.Message == "" {
		t.Error("Message should not be empty")
	}

	if result.Details == nil {
		t.Error("Details should be initialized")
	}

	// Git should be installed on CI/development machines
	// But we can't guarantee version, so we just check structure
	validStatuses := map[Status]bool{
		StatusHealthy:   true,
		StatusDegraded:  true,
		StatusUnhealthy: true,
	}

	if !validStatuses[result.Status] {
		t.Errorf("Status = %v, want one of [healthy, degraded, unhealthy]", result.Status)
	}

	// If healthy or degraded, should have git_path
	if result.Status == StatusHealthy || result.Status == StatusDegraded {
		if _, ok := result.Details["git_path"]; !ok {
			t.Error("Result should include git_path when git is found")
		}
	}

	// If healthy, should have version
	if result.Status == StatusHealthy {
		if _, ok := result.Details["version"]; !ok {
			t.Error("Healthy result should include version")
		}
	}

	// If unhealthy, should have suggestion
	if result.Status == StatusUnhealthy {
		if _, ok := result.Details["suggestion"]; !ok {
			t.Error("Unhealthy result should include suggestion")
		}
	}
}

func TestGitCheckerCheckWithCancelledContext(t *testing.T) {
	checker := NewGitChecker()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := checker.Check(ctx)

	if result == nil {
		t.Fatal("Check() returned nil")
	}

	// With a cancelled context, the check might fail or succeed quickly
	// depending on timing, but it should complete
	if result.Status == "" {
		t.Error("Status should be set even with cancelled context")
	}
}

func TestParseGitVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard format",
			input:    "git version 2.42.0",
			expected: "2.42.0",
		},
		{
			name:     "windows format",
			input:    "git version 2.42.0.windows.1",
			expected: "2.42.0",
		},
		{
			name:     "darwin format",
			input:    "git version 2.39.2.darwin.1",
			expected: "2.39.2",
		},
		{
			name:     "linux format",
			input:    "git version 2.34.1.linux",
			expected: "2.34.1",
		},
		{
			name:     "minimal format",
			input:    "git version 2.0",
			expected: "2.0",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "invalid format",
			input:    "not a version string",
			expected: "",
		},
		{
			name:     "short format",
			input:    "git version",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGitVersion(tt.input)
			if got != tt.expected {
				t.Errorf("parseGitVersion(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetMajorVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "version 2.42.0",
			input:    "2.42.0",
			expected: "2",
		},
		{
			name:     "version 1.9.5",
			input:    "1.9.5",
			expected: "1",
		},
		{
			name:     "version 3.0.0",
			input:    "3.0.0",
			expected: "3",
		},
		{
			name:     "version 2.0",
			input:    "2.0",
			expected: "2",
		},
		{
			name:     "single digit",
			input:    "2",
			expected: "2",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "invalid version",
			input:    "invalid.version",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMajorVersion(tt.input)
			if got != tt.expected {
				t.Errorf("getMajorVersion(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
