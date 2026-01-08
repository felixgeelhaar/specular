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
