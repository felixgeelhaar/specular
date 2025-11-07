package exec

import (
	"testing"

	"github.com/felixgeelhaar/specular/internal/policy"
)

func TestEnforcePolicy(t *testing.T) {
	tests := []struct {
		name    string
		step    Step
		policy  *policy.Policy
		wantErr bool
		errMsg  string
	}{
		{
			name: "local execution allowed",
			step: Step{
				ID:     "test-1",
				Runner: "local",
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					AllowLocal: true,
				},
			},
			wantErr: false,
		},
		{
			name: "local execution blocked",
			step: Step{
				ID:     "test-1",
				Runner: "local",
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					AllowLocal: false,
				},
			},
			wantErr: true,
			errMsg:  "local execution not allowed",
		},
		{
			name: "docker execution with allowed image",
			step: Step{
				ID:     "test-1",
				Runner: "docker",
				Image:  "golang:1.22",
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					AllowLocal: false,
					Docker: policy.DockerPolicy{
						Required:       true,
						ImageAllowlist: []string{"golang:*"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "docker execution with disallowed image",
			step: Step{
				ID:     "test-1",
				Runner: "docker",
				Image:  "malicious:latest",
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					AllowLocal: false,
					Docker: policy.DockerPolicy{
						Required:       true,
						ImageAllowlist: []string{"golang:*", "node:*"},
					},
				},
			},
			wantErr: true,
			errMsg:  "image not in allowlist",
		},
		{
			name: "docker execution with network policy violation",
			step: Step{
				ID:      "test-1",
				Runner:  "docker",
				Image:   "alpine:latest",
				Network: "bridge",
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					AllowLocal: false,
					Docker: policy.DockerPolicy{
						Required:       true,
						ImageAllowlist: []string{"alpine:*"},
						Network:        "none",
					},
				},
			},
			wantErr: true,
			errMsg:  "network mode 'bridge' not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnforcePolicy(tt.step, tt.policy)

			if (err != nil) != tt.wantErr {
				t.Errorf("EnforcePolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !contains(err.Error(), tt.errMsg) {
					t.Errorf("EnforcePolicy() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMatchesImagePattern(t *testing.T) {
	tests := []struct {
		name    string
		image   string
		pattern string
		want    bool
	}{
		{
			name:    "exact match",
			image:   "golang:1.22",
			pattern: "golang:1.22",
			want:    true,
		},
		{
			name:    "wildcard match - version",
			image:   "golang:1.22",
			pattern: "golang:*",
			want:    true,
		},
		{
			name:    "wildcard match - tag",
			image:   "node:20-alpine",
			pattern: "node:*",
			want:    true,
		},
		{
			name:    "no match - different image",
			image:   "python:3.11",
			pattern: "golang:*",
			want:    false,
		},
		{
			name:    "no match - different registry",
			image:   "docker.io/golang:1.22",
			pattern: "golang:*",
			want:    false,
		},
		{
			name:    "wildcard match - full prefix",
			image:   "ghcr.io/owner/image:v1.0.0",
			pattern: "ghcr.io/owner/*",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesImagePattern(tt.image, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesImagePattern(%q, %q) = %v, want %v", tt.image, tt.pattern, got, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
