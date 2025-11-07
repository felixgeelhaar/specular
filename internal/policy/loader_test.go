package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPolicy(t *testing.T) {
	tests := []struct {
		name           string
		policyContent  string
		wantErr        bool
		errContains    string
		validatePolicy func(*testing.T, *Policy)
	}{
		{
			name: "valid complete policy",
			policyContent: `
execution:
  allow_local: false
  docker:
    required: true
    image_allowlist:
      - "golang:1.22"
      - "node:20"
    cpu_limit: "4"
    mem_limit: "4g"
    network: "none"
linters:
  go:
    enabled: true
    cmd: "golangci-lint run"
  javascript:
    enabled: true
    cmd: "eslint ."
formatters:
  go:
    enabled: true
    cmd: "gofmt -w ."
tests:
  require_pass: true
  min_coverage: 0.80
security:
  secrets_scan: true
  dep_scan: true
routing:
  allow_models:
    - provider: "anthropic"
      names: ["claude-3.5-sonnet"]
    - provider: "openai"
      names: ["gpt-4"]
  deny_tools: ["shell_local"]
`,
			wantErr: false,
			validatePolicy: func(t *testing.T, p *Policy) {
				if p.Execution.AllowLocal != false {
					t.Error("AllowLocal should be false")
				}
				if p.Execution.Docker.Required != true {
					t.Error("Docker.Required should be true")
				}
				if len(p.Execution.Docker.ImageAllowlist) != 2 {
					t.Errorf("ImageAllowlist length = %d, want 2", len(p.Execution.Docker.ImageAllowlist))
				}
				if p.Execution.Docker.CPULimit != "4" {
					t.Errorf("CPULimit = %s, want 4", p.Execution.Docker.CPULimit)
				}
				if p.Execution.Docker.MemLimit != "4g" {
					t.Errorf("MemLimit = %s, want 4g", p.Execution.Docker.MemLimit)
				}
				if p.Execution.Docker.Network != "none" {
					t.Errorf("Network = %s, want none", p.Execution.Docker.Network)
				}

				// Test linters
				if len(p.Linters) != 2 {
					t.Errorf("Linters length = %d, want 2", len(p.Linters))
				}
				if goLinter, ok := p.Linters["go"]; !ok {
					t.Error("go linter not found")
				} else {
					if !goLinter.Enabled {
						t.Error("go linter should be enabled")
					}
					if goLinter.Cmd != "golangci-lint run" {
						t.Errorf("go linter cmd = %s, want golangci-lint run", goLinter.Cmd)
					}
				}

				// Test formatters
				if len(p.Formatters) != 1 {
					t.Errorf("Formatters length = %d, want 1", len(p.Formatters))
				}

				// Test tests policy
				if !p.Tests.RequirePass {
					t.Error("RequirePass should be true")
				}
				if p.Tests.MinCoverage != 0.80 {
					t.Errorf("MinCoverage = %f, want 0.80", p.Tests.MinCoverage)
				}

				// Test security policy
				if !p.Security.SecretsScan {
					t.Error("SecretsScan should be true")
				}
				if !p.Security.DepScan {
					t.Error("DepScan should be true")
				}

				// Test routing policy
				if len(p.Routing.AllowModels) != 2 {
					t.Errorf("AllowModels length = %d, want 2", len(p.Routing.AllowModels))
				}
				if len(p.Routing.DenyTools) != 1 {
					t.Errorf("DenyTools length = %d, want 1", len(p.Routing.DenyTools))
				}
			},
		},
		{
			name: "minimal policy",
			policyContent: `
execution:
  allow_local: true
tests:
  require_pass: false
`,
			wantErr: false,
			validatePolicy: func(t *testing.T, p *Policy) {
				if !p.Execution.AllowLocal {
					t.Error("AllowLocal should be true")
				}
				if p.Tests.RequirePass {
					t.Error("RequirePass should be false")
				}
			},
		},
		{
			name:          "empty file",
			policyContent: ``,
			wantErr:       false,
			validatePolicy: func(t *testing.T, p *Policy) {
				// Should load with zero values
				if p == nil {
					t.Error("Policy should not be nil for empty file")
				}
			},
		},
		{
			name:          "invalid yaml",
			policyContent: `invalid: [yaml syntax`,
			wantErr:       true,
			errContains:   "unmarshal policy",
		},
		{
			name: "policy with only docker config",
			policyContent: `
execution:
  docker:
    required: true
    image_allowlist:
      - "alpine:*"
      - "ubuntu:*"
    network: "bridge"
`,
			wantErr: false,
			validatePolicy: func(t *testing.T, p *Policy) {
				if !p.Execution.Docker.Required {
					t.Error("Docker.Required should be true")
				}
				if len(p.Execution.Docker.ImageAllowlist) != 2 {
					t.Errorf("ImageAllowlist length = %d, want 2", len(p.Execution.Docker.ImageAllowlist))
				}
				if p.Execution.Docker.Network != "bridge" {
					t.Errorf("Network = %s, want bridge", p.Execution.Docker.Network)
				}
			},
		},
		{
			name: "policy with linters only",
			policyContent: `
linters:
  go:
    enabled: true
    cmd: "staticcheck ./..."
  python:
    enabled: false
    cmd: ""
`,
			wantErr: false,
			validatePolicy: func(t *testing.T, p *Policy) {
				if len(p.Linters) != 2 {
					t.Errorf("Linters length = %d, want 2", len(p.Linters))
				}
				if goLinter, ok := p.Linters["go"]; !ok {
					t.Error("go linter not found")
				} else if goLinter.Cmd != "staticcheck ./..." {
					t.Errorf("go linter cmd = %s, want staticcheck ./...", goLinter.Cmd)
				}
				if pyLinter, ok := p.Linters["python"]; !ok {
					t.Error("python linter not found")
				} else if pyLinter.Enabled {
					t.Error("python linter should be disabled")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			policyFile := filepath.Join(tmpDir, "policy.yaml")

			err := os.WriteFile(policyFile, []byte(tt.policyContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test policy file: %v", err)
			}

			// Test LoadPolicy
			policy, err := LoadPolicy(policyFile)

			if tt.wantErr {
				if err == nil {
					t.Error("LoadPolicy() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("LoadPolicy() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadPolicy() unexpected error = %v", err)
			}

			if policy == nil {
				t.Fatal("LoadPolicy() returned nil policy")
			}

			// Run validation function if provided
			if tt.validatePolicy != nil {
				tt.validatePolicy(t, policy)
			}
		})
	}
}

func TestLoadPolicy_FileNotFound(t *testing.T) {
	_, err := LoadPolicy("/nonexistent/path/policy.yaml")
	if err == nil {
		t.Error("LoadPolicy() expected error for nonexistent file, got nil")
	}
	if !contains(err.Error(), "read policy file") {
		t.Errorf("LoadPolicy() error = %v, want error containing 'read policy file'", err)
	}
}

func TestLoadPolicy_EmptyPath(t *testing.T) {
	_, err := LoadPolicy("")
	if err == nil {
		t.Error("LoadPolicy() expected error for empty path, got nil")
	}
}

func TestLoadPolicy_DirectoryPath(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadPolicy(tmpDir)
	if err == nil {
		t.Error("LoadPolicy() expected error for directory path, got nil")
	}
}

func TestDefaultPolicy_AllFields(t *testing.T) {
	pol := DefaultPolicy()

	// Execution policy
	if pol.Execution.AllowLocal != false {
		t.Error("DefaultPolicy() AllowLocal should be false")
	}
	if !pol.Execution.Docker.Required {
		t.Error("DefaultPolicy() Docker.Required should be true")
	}
	if pol.Execution.Docker.CPULimit != "2" {
		t.Errorf("DefaultPolicy() CPULimit = %s, want 2", pol.Execution.Docker.CPULimit)
	}
	if pol.Execution.Docker.MemLimit != "2g" {
		t.Errorf("DefaultPolicy() MemLimit = %s, want 2g", pol.Execution.Docker.MemLimit)
	}
	if pol.Execution.Docker.Network != "none" {
		t.Errorf("DefaultPolicy() Network = %s, want none", pol.Execution.Docker.Network)
	}
	if pol.Execution.Docker.ImageAllowlist == nil {
		t.Error("DefaultPolicy() ImageAllowlist should not be nil")
	}

	// Test that maps are initialized
	if pol.Linters == nil {
		t.Error("DefaultPolicy() Linters map should not be nil")
	}
	if pol.Formatters == nil {
		t.Error("DefaultPolicy() Formatters map should not be nil")
	}

	// Test policy
	if !pol.Tests.RequirePass {
		t.Error("DefaultPolicy() RequirePass should be true")
	}
	if pol.Tests.MinCoverage != 0.70 {
		t.Errorf("DefaultPolicy() MinCoverage = %f, want 0.70", pol.Tests.MinCoverage)
	}

	// Security policy
	if !pol.Security.SecretsScan {
		t.Error("DefaultPolicy() SecretsScan should be true")
	}
	if !pol.Security.DepScan {
		t.Error("DefaultPolicy() DepScan should be true")
	}

	// Routing policy
	if pol.Routing.AllowModels == nil {
		t.Error("DefaultPolicy() AllowModels should not be nil")
	}
	if pol.Routing.DenyTools == nil {
		t.Error("DefaultPolicy() DenyTools should not be nil")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
