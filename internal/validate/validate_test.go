package validate

import (
	"strings"
	"testing"
)

func TestGoal(t *testing.T) {
	tests := []struct {
		name    string
		goal    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid goal",
			goal:    "Build a REST API for user management",
			wantErr: false,
		},
		{
			name:    "empty goal",
			goal:    "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "whitespace only",
			goal:    "   ",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "too short",
			goal:    "ab",
			wantErr: true,
			errMsg:  "at least 3 characters",
		},
		{
			name:    "goal with newlines",
			goal:    "Build an API\nwith multiple features",
			wantErr: false,
		},
		{
			name:    "goal with control char",
			goal:    "Build an API\x00 with injection",
			wantErr: true,
			errMsg:  "control characters",
		},
		{
			name:    "very long goal",
			goal:    strings.Repeat("a", MaxGoalLength+1),
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:    "max length goal",
			goal:    strings.Repeat("a", MaxGoalLength),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Goal(tt.goal)
			if (err != nil) != tt.wantErr {
				t.Errorf("Goal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Goal() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestProfileName(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default",
			profile: "default",
			wantErr: false,
		},
		{
			name:    "valid with hyphen",
			profile: "my-profile",
			wantErr: false,
		},
		{
			name:    "valid with underscore",
			profile: "my_profile",
			wantErr: false,
		},
		{
			name:    "valid with numbers",
			profile: "profile123",
			wantErr: false,
		},
		{
			name:    "empty profile",
			profile: "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "starts with number",
			profile: "123profile",
			wantErr: true,
			errMsg:  "must start with a letter",
		},
		{
			name:    "contains spaces",
			profile: "my profile",
			wantErr: true,
			errMsg:  "must start with a letter",
		},
		{
			name:    "contains special chars",
			profile: "profile!@#",
			wantErr: true,
			errMsg:  "must start with a letter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProfileName(tt.profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProfileName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ProfileName() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestProjectName(t *testing.T) {
	tests := []struct {
		name    string
		project string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid name",
			project: "my-project",
			wantErr: false,
		},
		{
			name:    "valid with spaces",
			project: "My Project",
			wantErr: false,
		},
		{
			name:    "empty name",
			project: "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "path traversal",
			project: "../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name:    "contains backslash",
			project: "project\\name",
			wantErr: true,
			errMsg:  "invalid characters",
		},
		{
			name:    "contains colon",
			project: "project:name",
			wantErr: true,
			errMsg:  "invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProjectName(tt.project)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProjectName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ProjectName() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestModelName(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid claude model",
			model:   "claude-3-sonnet-20240229",
			wantErr: false,
		},
		{
			name:    "valid gpt model",
			model:   "gpt-4-turbo",
			wantErr: false,
		},
		{
			name:    "valid with colon",
			model:   "anthropic:claude-3",
			wantErr: false,
		},
		{
			name:    "valid with slash",
			model:   "anthropic/claude-3",
			wantErr: false,
		},
		{
			name:    "empty model",
			model:   "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "contains spaces",
			model:   "claude 3",
			wantErr: true,
			errMsg:  "invalid characters",
		},
		{
			name:    "starts with special char",
			model:   "-claude-3",
			wantErr: true,
			errMsg:  "invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ModelName(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("ModelName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ModelName() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestURL(t *testing.T) {
	tests := []struct {
		name    string
		urlStr  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid https URL",
			urlStr:  "https://api.example.com/v1",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			urlStr:  "http://localhost:8080",
			wantErr: false,
		},
		{
			name:    "empty URL",
			urlStr:  "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "missing scheme",
			urlStr:  "api.example.com",
			wantErr: true,
			errMsg:  "must include a scheme",
		},
		{
			name:    "invalid scheme",
			urlStr:  "ftp://files.example.com",
			wantErr: true,
			errMsg:  "must be http or https",
		},
		{
			name:    "missing host",
			urlStr:  "https:///path",
			wantErr: true,
			errMsg:  "must include a host",
		},
		{
			name:    "file scheme",
			urlStr:  "file:///etc/passwd",
			wantErr: true,
			errMsg:  "must be http or https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := URL(tt.urlStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("URL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("URL() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestFilePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid relative path",
			path:    "src/main.go",
			wantErr: false,
		},
		{
			name:    "valid absolute path",
			path:    "/home/user/project",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "null byte injection",
			path:    "/path/to\x00/file",
			wantErr: true,
			errMsg:  "null bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FilePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("FilePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("FilePath() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestCheckpointID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid checkpoint ID",
			id:      "auto-1762811730",
			wantErr: false,
		},
		{
			name:    "valid with underscores",
			id:      "checkpoint_123_abc",
			wantErr: false,
		},
		{
			name:    "empty ID",
			id:      "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "starts with number",
			id:      "123-checkpoint",
			wantErr: true,
			errMsg:  "invalid characters",
		},
		{
			name:    "contains spaces",
			id:      "auto 123",
			wantErr: true,
			errMsg:  "invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckpointID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckpointID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("CheckpointID() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestPositiveFloat(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     float64
		allowZero bool
		wantErr   bool
	}{
		{
			name:      "positive value",
			fieldName: "budget",
			value:     5.0,
			allowZero: false,
			wantErr:   false,
		},
		{
			name:      "zero allowed",
			fieldName: "budget",
			value:     0,
			allowZero: true,
			wantErr:   false,
		},
		{
			name:      "zero not allowed",
			fieldName: "budget",
			value:     0,
			allowZero: false,
			wantErr:   true,
		},
		{
			name:      "negative value",
			fieldName: "budget",
			value:     -5.0,
			allowZero: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PositiveFloat(tt.fieldName, tt.value, tt.allowZero)
			if (err != nil) != tt.wantErr {
				t.Errorf("PositiveFloat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPositiveInt(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     int
		allowZero bool
		wantErr   bool
	}{
		{
			name:      "positive value",
			fieldName: "retries",
			value:     3,
			allowZero: false,
			wantErr:   false,
		},
		{
			name:      "zero allowed",
			fieldName: "retries",
			value:     0,
			allowZero: true,
			wantErr:   false,
		},
		{
			name:      "zero not allowed",
			fieldName: "retries",
			value:     0,
			allowZero: false,
			wantErr:   true,
		},
		{
			name:      "negative value",
			fieldName: "retries",
			value:     -1,
			allowZero: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PositiveInt(tt.fieldName, tt.value, tt.allowZero)
			if (err != nil) != tt.wantErr {
				t.Errorf("PositiveInt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInRange(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     int
		min       int
		max       int
		wantErr   bool
	}{
		{
			name:      "in range",
			fieldName: "steps",
			value:     5,
			min:       1,
			max:       10,
			wantErr:   false,
		},
		{
			name:      "at min",
			fieldName: "steps",
			value:     1,
			min:       1,
			max:       10,
			wantErr:   false,
		},
		{
			name:      "at max",
			fieldName: "steps",
			value:     10,
			min:       1,
			max:       10,
			wantErr:   false,
		},
		{
			name:      "below min",
			fieldName: "steps",
			value:     0,
			min:       1,
			max:       10,
			wantErr:   true,
		},
		{
			name:      "above max",
			fieldName: "steps",
			value:     11,
			min:       1,
			max:       10,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InRange(tt.fieldName, tt.value, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("InRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOneOf(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		allowed   []string
		wantErr   bool
	}{
		{
			name:      "valid option",
			fieldName: "format",
			value:     "json",
			allowed:   []string{"json", "yaml", "toml"},
			wantErr:   false,
		},
		{
			name:      "invalid option",
			fieldName: "format",
			value:     "xml",
			allowed:   []string{"json", "yaml", "toml"},
			wantErr:   true,
		},
		{
			name:      "empty value",
			fieldName: "format",
			value:     "",
			allowed:   []string{"json", "yaml", "toml"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := OneOf(tt.fieldName, tt.value, tt.allowed)
			if (err != nil) != tt.wantErr {
				t.Errorf("OneOf() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ======== Severity and ValidationResult Tests ========

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		severity Severity
		want     string
	}{
		{SeverityError, "error"},
		{SeverityWarning, "warning"},
		{SeverityInfo, "info"},
		{Severity(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.severity.String(); got != tt.want {
			t.Errorf("Severity(%d).String() = %q, want %q", tt.severity, got, tt.want)
		}
	}
}

func TestSeverity_Symbol(t *testing.T) {
	tests := []struct {
		severity Severity
		want     string
	}{
		{SeverityError, "✗"},
		{SeverityWarning, "⚠"},
		{SeverityInfo, "ℹ"},
		{Severity(99), "?"},
	}

	for _, tt := range tests {
		if got := tt.severity.Symbol(); got != tt.want {
			t.Errorf("Severity(%d).Symbol() = %q, want %q", tt.severity, got, tt.want)
		}
	}
}

func TestValidationIssue_Error(t *testing.T) {
	tests := []struct {
		name  string
		issue ValidationIssue
		want  []string // strings that should be present
	}{
		{
			name: "full issue",
			issue: ValidationIssue{
				Severity: SeverityError,
				Field:    "username",
				Value:    "invalid@user",
				Message:  "contains invalid characters",
				Code:     "V001",
			},
			want: []string{"error", "V001", "username", "invalid characters", "invalid@user"},
		},
		{
			name: "minimal issue",
			issue: ValidationIssue{
				Severity: SeverityWarning,
				Message:  "deprecated feature",
			},
			want: []string{"warning", "deprecated feature"},
		},
		{
			name: "long value truncation",
			issue: ValidationIssue{
				Severity: SeverityInfo,
				Value:    strings.Repeat("a", 100),
				Message:  "test",
			},
			want: []string{"...", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.issue.Error()
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("ValidationIssue.Error() = %q, want to contain %q", got, w)
				}
			}
		})
	}
}

func TestNewValidationResult(t *testing.T) {
	result := NewValidationResult()
	if result == nil {
		t.Fatal("NewValidationResult() returned nil")
	}
	if len(result.Issues) != 0 {
		t.Errorf("NewValidationResult().Issues should be empty, got %d", len(result.Issues))
	}
}

func TestValidationResult_AddError(t *testing.T) {
	result := NewValidationResult()
	result.AddError("field", "value", "error message")

	if len(result.Issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityError {
		t.Errorf("Severity = %v, want SeverityError", issue.Severity)
	}
	if issue.Field != "field" {
		t.Errorf("Field = %q, want %q", issue.Field, "field")
	}
}

func TestValidationResult_AddWarning(t *testing.T) {
	result := NewValidationResult()
	result.AddWarning("field", "value", "warning message")

	if len(result.Issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want SeverityWarning", issue.Severity)
	}
}

func TestValidationResult_AddInfo(t *testing.T) {
	result := NewValidationResult()
	result.AddInfo("field", "value", "info message")

	if len(result.Issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityInfo {
		t.Errorf("Severity = %v, want SeverityInfo", issue.Severity)
	}
}

func TestValidationResult_HasErrors(t *testing.T) {
	result := NewValidationResult()
	if result.HasErrors() {
		t.Error("Empty result should not have errors")
	}

	result.AddWarning("field", "", "warning")
	if result.HasErrors() {
		t.Error("Result with only warnings should not have errors")
	}

	result.AddError("field", "", "error")
	if !result.HasErrors() {
		t.Error("Result with errors should have errors")
	}
}

func TestValidationResult_HasWarnings(t *testing.T) {
	result := NewValidationResult()
	if result.HasWarnings() {
		t.Error("Empty result should not have warnings")
	}

	result.AddError("field", "", "error")
	if result.HasWarnings() {
		t.Error("Result with only errors should not have warnings")
	}

	result.AddWarning("field", "", "warning")
	if !result.HasWarnings() {
		t.Error("Result with warnings should have warnings")
	}
}

func TestValidationResult_Count(t *testing.T) {
	result := NewValidationResult()
	result.AddError("f1", "", "e1")
	result.AddError("f2", "", "e2")
	result.AddWarning("f3", "", "w1")
	result.AddWarning("f4", "", "w2")
	result.AddWarning("f5", "", "w3")
	result.AddInfo("f6", "", "i1")

	errors, warnings, infos := result.Count()
	if errors != 2 {
		t.Errorf("errors = %d, want 2", errors)
	}
	if warnings != 3 {
		t.Errorf("warnings = %d, want 3", warnings)
	}
	if infos != 1 {
		t.Errorf("infos = %d, want 1", infos)
	}
}

func TestValidationResult_Errors(t *testing.T) {
	result := NewValidationResult()
	result.AddError("f1", "", "e1")
	result.AddWarning("f2", "", "w1")
	result.AddError("f3", "", "e2")

	errors := result.Errors()
	if len(errors) != 2 {
		t.Errorf("len(errors) = %d, want 2", len(errors))
	}
	for _, e := range errors {
		if e.Severity != SeverityError {
			t.Errorf("Expected all to be errors, got %v", e.Severity)
		}
	}
}

func TestValidationResult_Warnings(t *testing.T) {
	result := NewValidationResult()
	result.AddError("f1", "", "e1")
	result.AddWarning("f2", "", "w1")
	result.AddWarning("f3", "", "w2")

	warnings := result.Warnings()
	if len(warnings) != 2 {
		t.Errorf("len(warnings) = %d, want 2", len(warnings))
	}
	for _, w := range warnings {
		if w.Severity != SeverityWarning {
			t.Errorf("Expected all to be warnings, got %v", w.Severity)
		}
	}
}

func TestValidationResult_Error(t *testing.T) {
	result := NewValidationResult()

	// No issues - no error
	if err := result.Error(); err != nil {
		t.Errorf("Empty result should return nil error, got %v", err)
	}

	// Only warnings - no error
	result.AddWarning("field", "", "warning")
	if err := result.Error(); err != nil {
		t.Errorf("Warnings-only result should return nil error, got %v", err)
	}

	// Single error
	result.AddError("field", "", "single error")
	err := result.Error()
	if err == nil {
		t.Error("Result with error should return an error")
	}
	if !strings.Contains(err.Error(), "single error") {
		t.Errorf("Error message should contain the issue, got %v", err)
	}

	// Multiple errors
	result.AddError("field2", "", "second error")
	err = result.Error()
	if err == nil {
		t.Error("Result with multiple errors should return an error")
	}
	if !strings.Contains(err.Error(), "2 errors") {
		t.Errorf("Error message should mention count, got %v", err)
	}
}

func TestValidationResult_Merge(t *testing.T) {
	result1 := NewValidationResult()
	result1.AddError("f1", "", "e1")

	result2 := NewValidationResult()
	result2.AddWarning("f2", "", "w1")
	result2.AddError("f3", "", "e2")

	result1.Merge(result2)

	if len(result1.Issues) != 3 {
		t.Errorf("After merge, len(Issues) = %d, want 3", len(result1.Issues))
	}

	// Merge with nil should not panic
	result1.Merge(nil)
	if len(result1.Issues) != 3 {
		t.Errorf("Merge with nil should not change count, got %d", len(result1.Issues))
	}
}
