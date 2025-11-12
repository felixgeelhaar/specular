package cmd

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/bundle"
)

// TestParseMetadataFlags tests the parseMetadataFlags function with various input scenarios
func TestParseMetadataFlags(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: map[string]string{},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: map[string]string{},
		},
		{
			name:  "single valid key=value",
			input: []string{"key1=value1"},
			expected: map[string]string{
				"key1": "value1",
			},
		},
		{
			name:  "multiple valid key=value pairs",
			input: []string{"key1=value1", "key2=value2", "key3=value3"},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			name:     "missing equals sign - should be skipped",
			input:    []string{"invalidentry"},
			expected: map[string]string{},
		},
		{
			name:  "mixed valid and invalid entries",
			input: []string{"key1=value1", "invalid", "key2=value2"},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:  "multiple equals signs - only split on first",
			input: []string{"key1=value=with=equals"},
			expected: map[string]string{
				"key1": "value=with=equals",
			},
		},
		{
			name:  "empty key",
			input: []string{"=value1"},
			expected: map[string]string{
				"": "value1",
			},
		},
		{
			name:  "empty value",
			input: []string{"key1="},
			expected: map[string]string{
				"key1": "",
			},
		},
		{
			name:  "whitespace in key and value",
			input: []string{"key with spaces=value with spaces"},
			expected: map[string]string{
				"key with spaces": "value with spaces",
			},
		},
		{
			name:  "special characters in key and value",
			input: []string{"key-1.2@test=value_3!4#test"},
			expected: map[string]string{
				"key-1.2@test": "value_3!4#test",
			},
		},
		{
			name:  "duplicate keys - last value wins",
			input: []string{"key1=value1", "key1=value2"},
			expected: map[string]string{
				"key1": "value2",
			},
		},
		{
			name:  "url-like values",
			input: []string{"homepage=https://example.com", "repo=git@github.com:user/repo.git"},
			expected: map[string]string{
				"homepage": "https://example.com",
				"repo":     "git@github.com:user/repo.git",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMetadataFlags(tt.input)

			// Check that result has same length as expected
			if len(result) != len(tt.expected) {
				t.Errorf("parseMetadataFlags() result length = %d, expected %d", len(result), len(tt.expected))
			}

			// Check each expected key-value pair
			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("parseMetadataFlags() missing key %q", key)
				} else if actualValue != expectedValue {
					t.Errorf("parseMetadataFlags() key %q = %q, expected %q", key, actualValue, expectedValue)
				}
			}

			// Check for unexpected keys in result
			for key := range result {
				if _, exists := tt.expected[key]; !exists {
					t.Errorf("parseMetadataFlags() unexpected key %q with value %q", key, result[key])
				}
			}
		})
	}
}

// TestCheckRequiredRoles tests the checkRequiredRoles function with various scenarios
func TestCheckRequiredRoles(t *testing.T) {
	// Helper function to create mock approval
	mockApproval := func(role string) *bundle.Approval {
		return &bundle.Approval{
			Role:      role,
			Signature: "mock-signature",
		}
	}

	tests := []struct {
		name          string
		requiredRoles []string
		verifiedRoles map[string]*bundle.Approval
		wantErr       bool
		errContains   string
	}{
		{
			name:          "no required roles - should pass",
			requiredRoles: []string{},
			verifiedRoles: map[string]*bundle.Approval{},
			wantErr:       false,
		},
		{
			name:          "nil required roles - should pass",
			requiredRoles: nil,
			verifiedRoles: map[string]*bundle.Approval{},
			wantErr:       false,
		},
		{
			name:          "single required role satisfied",
			requiredRoles: []string{"developer"},
			verifiedRoles: map[string]*bundle.Approval{
				"developer": mockApproval("developer"),
			},
			wantErr: false,
		},
		{
			name:          "multiple required roles all satisfied",
			requiredRoles: []string{"developer", "reviewer", "security"},
			verifiedRoles: map[string]*bundle.Approval{
				"developer": mockApproval("developer"),
				"reviewer":  mockApproval("reviewer"),
				"security":  mockApproval("security"),
			},
			wantErr: false,
		},
		{
			name:          "single required role missing",
			requiredRoles: []string{"developer"},
			verifiedRoles: map[string]*bundle.Approval{},
			wantErr:       true,
			errContains:   "bundle requires approvals from: developer",
		},
		{
			name:          "some required roles missing",
			requiredRoles: []string{"developer", "reviewer", "security"},
			verifiedRoles: map[string]*bundle.Approval{
				"developer": mockApproval("developer"),
			},
			wantErr:     true,
			errContains: "bundle requires approvals from:",
		},
		{
			name:          "all required roles missing",
			requiredRoles: []string{"developer", "reviewer"},
			verifiedRoles: map[string]*bundle.Approval{},
			wantErr:       true,
			errContains:   "bundle requires approvals from:",
		},
		{
			name:          "extra verified roles present - should not affect outcome",
			requiredRoles: []string{"developer"},
			verifiedRoles: map[string]*bundle.Approval{
				"developer": mockApproval("developer"),
				"reviewer":  mockApproval("reviewer"),
				"security":  mockApproval("security"),
			},
			wantErr: false,
		},
		{
			name:          "missing one of many required roles",
			requiredRoles: []string{"developer", "reviewer", "security", "qa"},
			verifiedRoles: map[string]*bundle.Approval{
				"developer": mockApproval("developer"),
				"reviewer":  mockApproval("reviewer"),
				"security":  mockApproval("security"),
			},
			wantErr:     true,
			errContains: "qa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout to prevent test output pollution
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the function
			err := checkRequiredRoles(tt.requiredRoles, tt.verifiedRoles)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			io.ReadAll(r) // Discard captured output

			// Check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("checkRequiredRoles() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("checkRequiredRoles() error = %q, should contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("checkRequiredRoles() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestCheckRequiredRolesErrorMessage tests the error message format for missing roles
func TestCheckRequiredRolesErrorMessage(t *testing.T) {
	tests := []struct {
		name          string
		requiredRoles []string
		verifiedRoles map[string]*bundle.Approval
		expectedRoles []string // Roles that should appear in error message
	}{
		{
			name:          "single missing role",
			requiredRoles: []string{"developer"},
			verifiedRoles: map[string]*bundle.Approval{},
			expectedRoles: []string{"developer"},
		},
		{
			name:          "two missing roles",
			requiredRoles: []string{"developer", "reviewer"},
			verifiedRoles: map[string]*bundle.Approval{},
			expectedRoles: []string{"developer", "reviewer"},
		},
		{
			name:          "three missing roles",
			requiredRoles: []string{"developer", "reviewer", "security"},
			verifiedRoles: map[string]*bundle.Approval{},
			expectedRoles: []string{"developer", "reviewer", "security"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout to prevent test output pollution
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the function
			err := checkRequiredRoles(tt.requiredRoles, tt.verifiedRoles)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			io.ReadAll(r) // Discard captured output

			if err == nil {
				t.Fatal("checkRequiredRoles() expected error, got nil")
			}

			// Verify all expected roles appear in error message
			errMsg := err.Error()
			for _, role := range tt.expectedRoles {
				if !strings.Contains(errMsg, role) {
					t.Errorf("checkRequiredRoles() error message %q should contain role %q", errMsg, role)
				}
			}

			// Verify error message format
			expectedPrefix := "bundle requires approvals from:"
			if !strings.HasPrefix(errMsg, expectedPrefix) {
				t.Errorf("checkRequiredRoles() error message should start with %q, got %q", expectedPrefix, errMsg)
			}
		})
	}
}

// TestCheckRequiredRolesConsoleOutput tests that proper console messages are printed
func TestCheckRequiredRolesConsoleOutput(t *testing.T) {
	mockApproval := func(role string) *bundle.Approval {
		return &bundle.Approval{
			Role:      role,
			Signature: "mock-signature",
		}
	}

	tests := []struct {
		name           string
		requiredRoles  []string
		verifiedRoles  map[string]*bundle.Approval
		shouldContain  []string // Strings that should appear in output
		shouldNotExist []string // Strings that should NOT appear in output
	}{
		{
			name:          "all roles satisfied - should show success messages",
			requiredRoles: []string{"developer", "reviewer"},
			verifiedRoles: map[string]*bundle.Approval{
				"developer": mockApproval("developer"),
				"reviewer":  mockApproval("reviewer"),
			},
			shouldContain: []string{
				"Checking required roles",
				"✓ developer: Approved",
				"✓ reviewer: Approved",
				"✓ All required roles have approved",
			},
			shouldNotExist: []string{"✗", "Missing or invalid approval"},
		},
		{
			name:          "some roles missing - should show failure messages",
			requiredRoles: []string{"developer", "reviewer"},
			verifiedRoles: map[string]*bundle.Approval{
				"developer": mockApproval("developer"),
			},
			shouldContain: []string{
				"Checking required roles",
				"✓ developer: Approved",
				"✗ reviewer: Missing or invalid approval",
				"⚠ Bundle is missing",
			},
			shouldNotExist: []string{"✓ All required roles have approved"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the function
			_ = checkRequiredRoles(tt.requiredRoles, tt.verifiedRoles)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = oldStdout
			captured, _ := io.ReadAll(r)
			output := string(captured)

			// Check for expected strings
			for _, expected := range tt.shouldContain {
				if !strings.Contains(output, expected) {
					t.Errorf("checkRequiredRoles() output should contain %q, got:\n%s", expected, output)
				}
			}

			// Check for strings that should not exist
			for _, unexpected := range tt.shouldNotExist {
				if strings.Contains(output, unexpected) {
					t.Errorf("checkRequiredRoles() output should NOT contain %q, got:\n%s", unexpected, output)
				}
			}
		})
	}
}

// Benchmark for parseMetadataFlags
func BenchmarkParseMetadataFlags(b *testing.B) {
	input := []string{
		"key1=value1",
		"key2=value2",
		"key3=value3",
		"key4=value4",
		"key5=value5",
		"invalid",
		"key6=value=with=equals",
		"homepage=https://example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseMetadataFlags(input)
	}
}

// Benchmark for checkRequiredRoles
func BenchmarkCheckRequiredRoles(b *testing.B) {
	requiredRoles := []string{"developer", "reviewer", "security", "qa", "admin"}
	verifiedRoles := map[string]*bundle.Approval{
		"developer": {Role: "developer", Signature: "sig"},
		"reviewer":  {Role: "reviewer", Signature: "sig"},
		"security":  {Role: "security", Signature: "sig"},
		"qa":        {Role: "qa", Signature: "sig"},
		"admin":     {Role: "admin", Signature: "sig"},
	}

	// Capture stdout to prevent benchmark pollution
	oldStdout := os.Stdout
	os.Stdout = nil

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checkRequiredRoles(requiredRoles, verifiedRoles)
	}

	// Restore stdout
	os.Stdout = oldStdout
}

// TestParseMetadataFlagsWithRealWorldExamples tests with realistic metadata
func TestParseMetadataFlagsWithRealWorldExamples(t *testing.T) {
	input := []string{
		"version=1.0.0",
		"author=John Doe",
		"email=john@example.com",
		"description=A production-grade bundle for deployment",
		"homepage=https://github.com/user/repo",
		"license=Apache-2.0",
		"created=2025-01-12T10:30:00Z",
		"env=production",
	}

	result := parseMetadataFlags(input)

	expected := map[string]string{
		"version":     "1.0.0",
		"author":      "John Doe",
		"email":       "john@example.com",
		"description": "A production-grade bundle for deployment",
		"homepage":    "https://github.com/user/repo",
		"license":     "Apache-2.0",
		"created":     "2025-01-12T10:30:00Z",
		"env":         "production",
	}

	if len(result) != len(expected) {
		t.Errorf("parseMetadataFlags() with real-world data: result length = %d, expected %d", len(result), len(expected))
	}

	for key, expectedValue := range expected {
		if actualValue, exists := result[key]; !exists {
			t.Errorf("parseMetadataFlags() with real-world data: missing key %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("parseMetadataFlags() with real-world data: key %q = %q, expected %q", key, actualValue, expectedValue)
		}
	}
}

// TestCheckRequiredRolesWithComplexScenario tests realistic approval scenarios
func TestCheckRequiredRolesWithComplexScenario(t *testing.T) {
	// Simulate a production deployment scenario with multiple approval tiers
	requiredRoles := []string{"developer", "senior-developer", "qa", "security", "devops", "product-owner"}

	// Scenario 1: All approvals present
	t.Run("production deployment all approvals", func(t *testing.T) {
		verifiedRoles := map[string]*bundle.Approval{
			"developer":        {Role: "developer", Signature: "sig1"},
			"senior-developer": {Role: "senior-developer", Signature: "sig2"},
			"qa":               {Role: "qa", Signature: "sig3"},
			"security":         {Role: "security", Signature: "sig4"},
			"devops":           {Role: "devops", Signature: "sig5"},
			"product-owner":    {Role: "product-owner", Signature: "sig6"},
		}

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := checkRequiredRoles(requiredRoles, verifiedRoles)

		w.Close()
		os.Stdout = oldStdout
		io.ReadAll(r)

		if err != nil {
			t.Errorf("checkRequiredRoles() with all approvals should not return error, got: %v", err)
		}
	})

	// Scenario 2: Missing critical security approval
	t.Run("production deployment missing security", func(t *testing.T) {
		verifiedRoles := map[string]*bundle.Approval{
			"developer":        {Role: "developer", Signature: "sig1"},
			"senior-developer": {Role: "senior-developer", Signature: "sig2"},
			"qa":               {Role: "qa", Signature: "sig3"},
			"devops":           {Role: "devops", Signature: "sig5"},
			"product-owner":    {Role: "product-owner", Signature: "sig6"},
		}

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := checkRequiredRoles(requiredRoles, verifiedRoles)

		w.Close()
		os.Stdout = oldStdout
		io.ReadAll(r)

		if err == nil {
			t.Error("checkRequiredRoles() should return error when security approval is missing")
		}

		if !strings.Contains(err.Error(), "security") {
			t.Errorf("checkRequiredRoles() error should mention 'security', got: %v", err)
		}
	})
}

// Note: Example function removed because parseMetadataFlags is unexported.
// See TestParseMetadataFlagsWithRealWorldExamples for usage examples.
