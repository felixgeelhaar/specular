package policy

import (
	"testing"
)

func TestPolicyLoading(t *testing.T) {
	// Test default policy creation
	pol := DefaultPolicy()
	if pol == nil {
		t.Fatal("DefaultPolicy() returned nil")
	}

	if pol.Execution.Docker.Required != true {
		t.Error("Default policy should require Docker")
	}

	if pol.Tests.RequirePass != true {
		t.Error("Default policy should require tests to pass")
	}

	if pol.Tests.MinCoverage != 0.70 {
		t.Errorf("Default policy min coverage = %f, want 0.70", pol.Tests.MinCoverage)
	}
}

func TestValidateToolConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  ToolConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  ToolConfig{Enabled: true, Cmd: "golangci-lint run"},
			wantErr: false,
		},
		{
			name:    "disabled tool",
			config:  ToolConfig{Enabled: false, Cmd: ""},
			wantErr: false,
		},
		{
			name:    "enabled without command",
			config:  ToolConfig{Enabled: true, Cmd: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToolConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
