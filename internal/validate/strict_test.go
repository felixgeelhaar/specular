package validate

import (
	"bytes"
	"strings"
	"testing"
)

func TestDefaultStrictMode(t *testing.T) {
	mode := DefaultStrictMode()
	if mode.Enabled {
		t.Error("Default strict mode should be disabled")
	}
	if mode.FailOnWarnings {
		t.Error("Default strict mode should not fail on warnings")
	}
}

func TestNewStrictValidator(t *testing.T) {
	mode := StrictMode{Enabled: true, FailOnWarnings: true}
	v := NewStrictValidator(mode)

	if v == nil {
		t.Fatal("NewStrictValidator returned nil")
	}
	if v.Result == nil {
		t.Error("Result should not be nil")
	}
	if !v.Mode.Enabled {
		t.Error("Mode.Enabled should be true")
	}
}

func TestStrictValidator_AddError(t *testing.T) {
	v := NewStrictValidator(DefaultStrictMode())
	v.AddError("field", "value", "error message")

	if len(v.Result.Issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(v.Result.Issues))
	}
	if v.Result.Issues[0].Severity != SeverityError {
		t.Errorf("Expected error severity, got %v", v.Result.Issues[0].Severity)
	}
}

func TestStrictValidator_AddWarning_NormalMode(t *testing.T) {
	v := NewStrictValidator(StrictMode{Enabled: false})
	v.AddWarning("field", "value", "warning message")

	if len(v.Result.Issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(v.Result.Issues))
	}
	if v.Result.Issues[0].Severity != SeverityWarning {
		t.Errorf("In normal mode, warnings should stay warnings, got %v", v.Result.Issues[0].Severity)
	}
}

func TestStrictValidator_AddWarning_StrictMode(t *testing.T) {
	v := NewStrictValidator(StrictMode{Enabled: true})
	v.AddWarning("field", "value", "warning message")

	if len(v.Result.Issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(v.Result.Issues))
	}
	if v.Result.Issues[0].Severity != SeverityError {
		t.Errorf("In strict mode, warnings should become errors, got %v", v.Result.Issues[0].Severity)
	}
	if !strings.Contains(v.Result.Issues[0].Message, "strict mode") {
		t.Error("Error message should mention strict mode")
	}
}

func TestStrictValidator_ShouldFail(t *testing.T) {
	tests := []struct {
		name       string
		mode       StrictMode
		addError   bool
		addWarning bool
		wantFail   bool
	}{
		{
			name:       "no issues",
			mode:       DefaultStrictMode(),
			addError:   false,
			addWarning: false,
			wantFail:   false,
		},
		{
			name:       "error only",
			mode:       DefaultStrictMode(),
			addError:   true,
			addWarning: false,
			wantFail:   true,
		},
		{
			name:       "warning only - normal mode",
			mode:       StrictMode{Enabled: false, FailOnWarnings: false},
			addError:   false,
			addWarning: true,
			wantFail:   false,
		},
		{
			name:       "warning only - fail on warnings",
			mode:       StrictMode{Enabled: false, FailOnWarnings: true},
			addError:   false,
			addWarning: true,
			wantFail:   true,
		},
		{
			name:       "warning in strict mode becomes error",
			mode:       StrictMode{Enabled: true, FailOnWarnings: false},
			addError:   false,
			addWarning: true,
			wantFail:   true, // warning becomes error in strict mode
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewStrictValidator(tt.mode)
			if tt.addError {
				v.AddError("field", "", "error")
			}
			if tt.addWarning {
				v.AddWarning("field", "", "warning")
			}

			if got := v.ShouldFail(); got != tt.wantFail {
				t.Errorf("ShouldFail() = %v, want %v", got, tt.wantFail)
			}
		})
	}
}

func TestStrictValidator_GetError(t *testing.T) {
	tests := []struct {
		name       string
		mode       StrictMode
		addError   bool
		addWarning bool
		wantNil    bool
		wantMsg    string
	}{
		{
			name:       "no issues",
			mode:       DefaultStrictMode(),
			addError:   false,
			addWarning: false,
			wantNil:    true,
		},
		{
			name:       "error only",
			mode:       DefaultStrictMode(),
			addError:   true,
			addWarning: false,
			wantNil:    false,
			wantMsg:    "validation failed",
		},
		{
			name:       "warning only - normal mode",
			mode:       StrictMode{Enabled: false, FailOnWarnings: false},
			addError:   false,
			addWarning: true,
			wantNil:    true,
		},
		{
			name:       "warning only - fail on warnings",
			mode:       StrictMode{Enabled: false, FailOnWarnings: true},
			addError:   false,
			addWarning: true,
			wantNil:    false,
			wantMsg:    "strict mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewStrictValidator(tt.mode)
			if tt.addError {
				v.AddError("field", "", "error message")
			}
			if tt.addWarning {
				v.AddWarning("field", "", "warning message")
			}

			err := v.GetError()
			if tt.wantNil && err != nil {
				t.Errorf("GetError() = %v, want nil", err)
			}
			if !tt.wantNil && err == nil {
				t.Error("GetError() = nil, want error")
			}
			if !tt.wantNil && tt.wantMsg != "" && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("GetError() = %v, want containing %q", err, tt.wantMsg)
			}
		})
	}
}

func TestPrintValidationResult(t *testing.T) {
	result := NewValidationResult()
	result.AddErrorWithCode("field1", "value1", "error message", "E001")
	result.AddWarning("field2", "value2", "warning message")
	result.AddInfo("field3", "", "info message")

	tests := []struct {
		name     string
		useColor bool
		wantStr  []string
	}{
		{
			name:     "with color",
			useColor: true,
			wantStr:  []string{"Validation Report", "Errors (1)", "Warnings (1)", "Info (1)", "Summary"},
		},
		{
			name:     "without color",
			useColor: false,
			wantStr:  []string{"Validation Report", "Errors (1)", "Warnings (1)", "Info (1)", "Summary"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PrintValidationResult(&buf, result, tt.useColor)
			output := buf.String()

			for _, want := range tt.wantStr {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing %q:\n%s", want, output)
				}
			}
		})
	}
}

func TestPrintValidationResult_Empty(t *testing.T) {
	result := NewValidationResult()
	var buf bytes.Buffer
	PrintValidationResult(&buf, result, false)

	if buf.Len() != 0 {
		t.Errorf("Empty result should produce no output, got %q", buf.String())
	}
}

func TestPrintValidationResult_Nil(t *testing.T) {
	var buf bytes.Buffer
	PrintValidationResult(&buf, nil, false)

	if buf.Len() != 0 {
		t.Errorf("Nil result should produce no output, got %q", buf.String())
	}
}

func TestPrintWarnings(t *testing.T) {
	result := NewValidationResult()
	result.AddError("f1", "", "error") // errors should not appear
	result.AddWarning("f2", "val", "warning message")
	result.AddWarning("f3", "", "another warning")

	var buf bytes.Buffer
	PrintWarnings(&buf, result, false)
	output := buf.String()

	if !strings.Contains(output, "Warnings (2)") {
		t.Errorf("Should show 2 warnings, got: %s", output)
	}
	if strings.Contains(output, "error") {
		t.Errorf("Should not show errors, got: %s", output)
	}
}

func TestPrintWarnings_Empty(t *testing.T) {
	result := NewValidationResult()
	result.AddError("f1", "", "error") // only errors, no warnings

	var buf bytes.Buffer
	PrintWarnings(&buf, result, false)

	if buf.Len() != 0 {
		t.Errorf("No warnings should produce no output, got %q", buf.String())
	}
}

func TestWarnIfEmpty(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantIssues int
	}{
		{"non-empty", "value", 0},
		{"empty", "", 1},
		{"whitespace only", "   ", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewValidationResult()
			WarnIfEmpty(result, "field", tt.value)
			if len(result.Issues) != tt.wantIssues {
				t.Errorf("WarnIfEmpty() added %d issues, want %d", len(result.Issues), tt.wantIssues)
			}
		})
	}
}

func TestWarnIfDeprecated(t *testing.T) {
	result := NewValidationResult()
	WarnIfDeprecated(result, "config", "old-format", "new-format")

	if len(result.Issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want SeverityWarning", issue.Severity)
	}
	if !strings.Contains(issue.Message, "deprecated") {
		t.Errorf("Message should contain 'deprecated', got %q", issue.Message)
	}
	if !strings.Contains(issue.Message, "new-format") {
		t.Errorf("Message should mention replacement, got %q", issue.Message)
	}
}

func TestWarnIfNotRecommended(t *testing.T) {
	result := NewValidationResult()
	WarnIfNotRecommended(result, "timeout", "5s", "increase timeout to at least 30s for network operations")

	if len(result.Issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want SeverityWarning", issue.Severity)
	}
	if !strings.Contains(issue.Message, "not recommended") {
		t.Errorf("Message should contain 'not recommended', got %q", issue.Message)
	}
}

func TestValidateWithResult(t *testing.T) {
	alwaysFailValidator := func(s string) error {
		return &ValidationError{Field: "test", Message: "always fails"}
	}
	alwaysPassValidator := func(s string) error {
		return nil
	}

	tests := []struct {
		name       string
		validator  func(string) error
		wantIssues int
	}{
		{"passing validator", alwaysPassValidator, 0},
		{"failing validator", alwaysFailValidator, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewValidationResult()
			ValidateWithResult(result, SeverityError, "field", "value", tt.validator)
			if len(result.Issues) != tt.wantIssues {
				t.Errorf("ValidateWithResult() added %d issues, want %d", len(result.Issues), tt.wantIssues)
			}
		})
	}
}
