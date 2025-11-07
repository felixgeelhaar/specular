package drift

import (
	"testing"

	"github.com/felixgeelhaar/specular/internal/exec"
	"github.com/felixgeelhaar/specular/internal/policy"
)

func TestDetectInfraDrift(t *testing.T) {
	tests := []struct {
		name         string
		opts         InfraDriftOptions
		wantFindings int
		wantCodes    []string
	}{
		{
			name: "no policy or plan",
			opts: InfraDriftOptions{},
			wantFindings: 0,
			wantCodes:    []string{},
		},
		{
			name: "compliant configuration",
			opts: InfraDriftOptions{
				Policy: &policy.Policy{
					Execution: policy.ExecutionPolicy{
						AllowLocal: false,
						Docker: policy.DockerPolicy{
							Required: true,
							ImageAllowlist: []string{
								"golang:*",
								"node:*",
							},
							Network:  "none",
							CPULimit: "2",
							MemLimit: "2g",
						},
					},
					Tests: policy.TestPolicy{
						RequirePass: true,
					},
					Security: policy.SecurityPolicy{
						SecretsScan: true,
						DepScan:     true,
					},
				},
				TaskImages: map[string]string{
					"task-001": "golang:1.22",
				},
			},
			wantFindings: 0,
			wantCodes:    []string{},
		},
		{
			name: "disallowed Docker image",
			opts: InfraDriftOptions{
				Policy: &policy.Policy{
					Execution: policy.ExecutionPolicy{
						AllowLocal: false,
						Docker: policy.DockerPolicy{
							Required: true,
							ImageAllowlist: []string{
								"golang:*",
							},
							Network:  "none",
							CPULimit: "2",
							MemLimit: "2g",
						},
					},
					Tests: policy.TestPolicy{
						RequirePass: true,
					},
					Security: policy.SecurityPolicy{
						SecretsScan: true,
						DepScan:     true,
					},
				},
				TaskImages: map[string]string{
					"task-001": "python:3.9",
				},
			},
			wantFindings: 1,
			wantCodes:    []string{"DISALLOWED_DOCKER_IMAGE"},
		},
		{
			name: "missing Docker image",
			opts: InfraDriftOptions{
				Policy: &policy.Policy{
					Execution: policy.ExecutionPolicy{
						AllowLocal: false,
						Docker: policy.DockerPolicy{
							Required: true,
							ImageAllowlist: []string{
								"golang:*",
							},
							Network:  "none",
							CPULimit: "2",
							MemLimit: "2g",
						},
					},
					Tests: policy.TestPolicy{
						RequirePass: true,
					},
					Security: policy.SecurityPolicy{
						SecretsScan: true,
						DepScan:     true,
					},
				},
				TaskImages: map[string]string{
					"task-001": "",
				},
			},
			wantFindings: 1,
			wantCodes:    []string{"MISSING_DOCKER_IMAGE"},
		},
		{
			name: "policy warnings",
			opts: InfraDriftOptions{
				Policy: &policy.Policy{
					Execution: policy.ExecutionPolicy{
						AllowLocal: true,
						Docker: policy.DockerPolicy{
							Required: true,
							ImageAllowlist: []string{
								"golang:*",
							},
							Network: "bridge",
						},
					},
					Tests: policy.TestPolicy{
						RequirePass: false,
					},
					Security: policy.SecurityPolicy{
						SecretsScan: false,
						DepScan:     false,
					},
				},
				TaskImages: map[string]string{
					"task-001": "golang:1.22",
				},
			},
			wantFindings: 7,
			wantCodes: []string{
				"ALLOW_LOCAL_EXECUTION",
				"NETWORK_ACCESS_ENABLED",
				"MISSING_CPU_LIMIT",
				"MISSING_MEMORY_LIMIT",
				"TESTS_NOT_REQUIRED",
				"SECRETS_SCAN_DISABLED",
				"DEPENDENCY_SCAN_DISABLED",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := DetectInfraDrift(tt.opts)

			if len(findings) != tt.wantFindings {
				t.Errorf("DetectInfraDrift() found %d findings, want %d", len(findings), tt.wantFindings)
				for _, f := range findings {
					t.Logf("  Finding: %s - %s", f.Code, f.Message)
				}
			}

			// Check specific finding codes
			foundCodes := make(map[string]int)
			for _, f := range findings {
				foundCodes[f.Code]++
			}

			wantCodeCounts := make(map[string]int)
			for _, code := range tt.wantCodes {
				wantCodeCounts[code]++
			}

			for code, wantCount := range wantCodeCounts {
				if gotCount := foundCodes[code]; gotCount != wantCount {
					t.Errorf("DetectInfraDrift() found %d occurrences of %s, want %d", gotCount, code, wantCount)
				}
			}
		})
	}
}

func TestCheckDockerImagePolicy(t *testing.T) {
	tests := []struct {
		name         string
		taskImages   map[string]string
		policy       *policy.Policy
		wantFindings int
		wantCode     string
	}{
		{
			name: "allowed image - exact match",
			taskImages: map[string]string{
				"task-001": "golang:1.22",
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					Docker: policy.DockerPolicy{
						Required:       true,
						ImageAllowlist: []string{"golang:1.22"},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "allowed image - wildcard match",
			taskImages: map[string]string{
				"task-001": "golang:1.22",
				"task-002": "golang:1.21",
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					Docker: policy.DockerPolicy{
						Required:       true,
						ImageAllowlist: []string{"golang:*"},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "allowed image - prefix match",
			taskImages: map[string]string{
				"task-001": "ghcr.io/acme/builder:latest",
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					Docker: policy.DockerPolicy{
						Required:       true,
						ImageAllowlist: []string{"ghcr.io/acme/*"},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "disallowed image",
			taskImages: map[string]string{
				"task-001": "python:3.9",
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					Docker: policy.DockerPolicy{
						Required:       true,
						ImageAllowlist: []string{"golang:*"},
					},
				},
			},
			wantFindings: 1,
			wantCode:     "DISALLOWED_DOCKER_IMAGE",
		},
		{
			name: "missing image",
			taskImages: map[string]string{
				"task-001": "",
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					Docker: policy.DockerPolicy{
						Required:       true,
						ImageAllowlist: []string{"golang:*"},
					},
				},
			},
			wantFindings: 1,
			wantCode:     "MISSING_DOCKER_IMAGE",
		},
		{
			name: "Docker not required",
			taskImages: map[string]string{
				"task-001": "python:3.9",
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					Docker: policy.DockerPolicy{
						Required: false,
					},
				},
			},
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := checkDockerImagePolicy(tt.taskImages, tt.policy)

			if len(findings) != tt.wantFindings {
				t.Errorf("checkDockerImagePolicy() found %d findings, want %d", len(findings), tt.wantFindings)
			}

			if tt.wantCode != "" && len(findings) > 0 {
				if findings[0].Code != tt.wantCode {
					t.Errorf("checkDockerImagePolicy() code = %s, want %s", findings[0].Code, tt.wantCode)
				}
			}
		})
	}
}

func TestCheckExecutionPolicy(t *testing.T) {
	tests := []struct {
		name         string
		policy       *policy.Policy
		wantFindings int
		wantCodes    []string
	}{
		{
			name: "secure configuration",
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					AllowLocal: false,
					Docker: policy.DockerPolicy{
						Required: true,
						Network:  "none",
						CPULimit: "2",
						MemLimit: "2g",
					},
				},
				Tests: policy.TestPolicy{
					RequirePass: true,
				},
				Security: policy.SecurityPolicy{
					SecretsScan: true,
					DepScan:     true,
				},
			},
			wantFindings: 0,
			wantCodes:    []string{},
		},
		{
			name: "all warnings",
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					AllowLocal: true,
					Docker: policy.DockerPolicy{
						Required: true,
						Network:  "bridge",
					},
				},
				Tests: policy.TestPolicy{
					RequirePass: false,
				},
				Security: policy.SecurityPolicy{
					SecretsScan: false,
					DepScan:     false,
				},
			},
			wantFindings: 7,
			wantCodes: []string{
				"ALLOW_LOCAL_EXECUTION",
				"NETWORK_ACCESS_ENABLED",
				"MISSING_CPU_LIMIT",
				"MISSING_MEMORY_LIMIT",
				"TESTS_NOT_REQUIRED",
				"SECRETS_SCAN_DISABLED",
				"DEPENDENCY_SCAN_DISABLED",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := checkExecutionPolicy(tt.policy)

			if len(findings) != tt.wantFindings {
				t.Errorf("checkExecutionPolicy() found %d findings, want %d", len(findings), tt.wantFindings)
				for _, f := range findings {
					t.Logf("  Finding: %s - %s", f.Code, f.Message)
				}
			}

			foundCodes := make(map[string]bool)
			for _, f := range findings {
				foundCodes[f.Code] = true
			}

			for _, wantCode := range tt.wantCodes {
				if !foundCodes[wantCode] {
					t.Errorf("checkExecutionPolicy() missing expected finding code: %s", wantCode)
				}
			}
		})
	}
}

func TestCheckRunManifests(t *testing.T) {
	tests := []struct {
		name         string
		manifests    []exec.RunManifest
		policy       *policy.Policy
		wantFindings int
		wantCodes    []string
	}{
		{
			name: "all executions successful",
			manifests: []exec.RunManifest{
				{
					StepID:   "step-001",
					ExitCode: 0,
					Image:    "golang:1.22",
				},
				{
					StepID:   "step-002",
					ExitCode: 0,
					Image:    "golang:1.22",
				},
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					Docker: policy.DockerPolicy{
						Required:       true,
						ImageAllowlist: []string{"golang:*"},
					},
				},
			},
			wantFindings: 0,
			wantCodes:    []string{},
		},
		{
			name: "execution failed",
			manifests: []exec.RunManifest{
				{
					StepID:   "step-001",
					ExitCode: 1,
					Image:    "golang:1.22",
				},
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					Docker: policy.DockerPolicy{
						Required: true,
					},
				},
			},
			wantFindings: 1,
			wantCodes:    []string{"EXECUTION_FAILED"},
		},
		{
			name: "disallowed image used",
			manifests: []exec.RunManifest{
				{
					StepID:   "step-001",
					ExitCode: 0,
					Image:    "python:3.9",
				},
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					Docker: policy.DockerPolicy{
						Required:       true,
						ImageAllowlist: []string{"golang:*"},
					},
				},
			},
			wantFindings: 1,
			wantCodes:    []string{"DISALLOWED_EXECUTION_IMAGE"},
		},
		{
			name: "mixed results",
			manifests: []exec.RunManifest{
				{
					StepID:   "step-001",
					ExitCode: 0,
					Image:    "golang:1.22",
				},
				{
					StepID:   "step-002",
					ExitCode: 1,
					Image:    "python:3.9",
				},
			},
			policy: &policy.Policy{
				Execution: policy.ExecutionPolicy{
					Docker: policy.DockerPolicy{
						Required:       true,
						ImageAllowlist: []string{"golang:*"},
					},
				},
			},
			wantFindings: 2,
			wantCodes:    []string{"EXECUTION_FAILED", "DISALLOWED_EXECUTION_IMAGE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := checkRunManifests(tt.manifests, tt.policy)

			if len(findings) != tt.wantFindings {
				t.Errorf("checkRunManifest() found %d findings, want %d", len(findings), tt.wantFindings)
				for _, f := range findings {
					t.Logf("  Finding: %s - %s", f.Code, f.Message)
				}
			}

			foundCodes := make(map[string]bool)
			for _, f := range findings {
				foundCodes[f.Code] = true
			}

			for _, wantCode := range tt.wantCodes {
				if !foundCodes[wantCode] {
					t.Errorf("checkRunManifest() missing expected finding code: %s", wantCode)
				}
			}
		})
	}
}

func TestIsImageAllowed(t *testing.T) {
	tests := []struct {
		name      string
		image     string
		allowlist []string
		want      bool
	}{
		{
			name:      "exact match",
			image:     "golang:1.22",
			allowlist: []string{"golang:1.22", "node:20"},
			want:      true,
		},
		{
			name:      "wildcard match",
			image:     "golang:1.22",
			allowlist: []string{"golang:*"},
			want:      true,
		},
		{
			name:      "prefix match",
			image:     "ghcr.io/acme/builder:latest",
			allowlist: []string{"ghcr.io/acme/*"},
			want:      true,
		},
		{
			name:      "no match",
			image:     "python:3.9",
			allowlist: []string{"golang:*", "node:*"},
			want:      false,
		},
		{
			name:      "wildcard doesn't match different base",
			image:     "python:3.9",
			allowlist: []string{"golang:*"},
			want:      false,
		},
		{
			name:      "prefix doesn't match different registry",
			image:     "docker.io/acme/builder:latest",
			allowlist: []string{"ghcr.io/acme/*"},
			want:      false,
		},
		{
			name:      "empty allowlist",
			image:     "golang:1.22",
			allowlist: []string{},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isImageAllowed(tt.image, tt.allowlist)
			if got != tt.want {
				t.Errorf("isImageAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}
