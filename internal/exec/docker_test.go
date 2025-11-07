package exec

import (
	"testing"
)

func TestBuildDockerArgs(t *testing.T) {
	tests := []struct {
		name string
		step Step
		want []string
	}{
		{
			name: "basic step",
			step: Step{
				Image: "alpine:latest",
				Cmd:   []string{"echo", "hello"},
			},
			want: []string{
				"run", "--rm",
				"--read-only", "--pids-limit", "256", "--cap-drop", "ALL",
				"alpine:latest", "echo", "hello",
			},
		},
		{
			name: "step with network",
			step: Step{
				Image:   "alpine:latest",
				Cmd:     []string{"echo", "hello"},
				Network: "none",
			},
			want: []string{
				"run", "--rm",
				"--network", "none",
				"--read-only", "--pids-limit", "256", "--cap-drop", "ALL",
				"alpine:latest", "echo", "hello",
			},
		},
		{
			name: "step with resource limits",
			step: Step{
				Image: "alpine:latest",
				Cmd:   []string{"echo", "hello"},
				CPU:   "2",
				Mem:   "1g",
			},
			want: []string{
				"run", "--rm",
				"--cpus", "2", "--memory", "1g",
				"--read-only", "--pids-limit", "256", "--cap-drop", "ALL",
				"alpine:latest", "echo", "hello",
			},
		},
		{
			name: "step with workdir",
			step: Step{
				Image:   "alpine:latest",
				Cmd:     []string{"ls", "-la"},
				Workdir: "/project",
			},
			want: []string{
				"run", "--rm",
				"--read-only", "--pids-limit", "256", "--cap-drop", "ALL",
				"-v", "/project:/workspace", "-w", "/workspace",
				"alpine:latest", "ls", "-la",
			},
		},
		{
			name: "step with environment variables",
			step: Step{
				Image: "alpine:latest",
				Cmd:   []string{"env"},
				Env: map[string]string{
					"FOO": "bar",
					"BAZ": "qux",
				},
			},
			want: []string{
				"run", "--rm",
				"--read-only", "--pids-limit", "256", "--cap-drop", "ALL",
			},
		},
		{
			name: "full step with all options",
			step: Step{
				Image:   "golang:1.22",
				Cmd:     []string{"go", "test", "./..."},
				Workdir: "/go/src/app",
				Env: map[string]string{
					"GOOS":   "linux",
					"GOARCH": "amd64",
				},
				Network: "none",
				CPU:     "4",
				Mem:     "2g",
			},
			want: []string{
				"run", "--rm",
				"--network", "none",
				"--cpus", "4", "--memory", "2g",
				"--read-only", "--pids-limit", "256", "--cap-drop", "ALL",
				"-v", "/go/src/app:/workspace", "-w", "/workspace",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildDockerArgs(tt.step)

			// Verify required args are present
			if !containsAllStrings(got, []string{"run", "--rm", "--read-only", "--pids-limit", "--cap-drop"}) {
				t.Errorf("buildDockerArgs() missing required security args")
			}

			// Verify image is present
			if !containsString(got, tt.step.Image) {
				t.Errorf("buildDockerArgs() missing image %q", tt.step.Image)
			}

			// Verify command args are present
			for _, cmd := range tt.step.Cmd {
				if !containsString(got, cmd) {
					t.Errorf("buildDockerArgs() missing command arg %q", cmd)
				}
			}

			// Verify network if specified
			if tt.step.Network != "" {
				if !containsStrings(got, []string{"--network", tt.step.Network}) {
					t.Errorf("buildDockerArgs() missing network config")
				}
			}

			// Verify resource limits if specified
			if tt.step.CPU != "" {
				if !containsStrings(got, []string{"--cpus", tt.step.CPU}) {
					t.Errorf("buildDockerArgs() missing CPU limit")
				}
			}
			if tt.step.Mem != "" {
				if !containsStrings(got, []string{"--memory", tt.step.Mem}) {
					t.Errorf("buildDockerArgs() missing memory limit")
				}
			}

			// Verify workdir if specified
			if tt.step.Workdir != "" {
				if !containsStrings(got, []string{"-w", "/workspace"}) {
					t.Errorf("buildDockerArgs() missing workdir config")
				}
			}

			// Verify environment variables if specified
			for key, value := range tt.step.Env {
				envArg := key + "=" + value
				found := false
				for i, arg := range got {
					if arg == "-e" && i+1 < len(got) && got[i+1] == envArg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("buildDockerArgs() missing environment variable %s=%s", key, value)
				}
			}
		})
	}
}

// Helper functions
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func containsStrings(slice []string, substrs []string) bool {
	for i := 0; i <= len(slice)-len(substrs); i++ {
		match := true
		for j, substr := range substrs {
			if i+j >= len(slice) || slice[i+j] != substr {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func containsAllStrings(slice []string, substrs []string) bool {
	for _, substr := range substrs {
		if !containsString(slice, substr) {
			return false
		}
	}
	return true
}

func TestValidateDockerAvailable(t *testing.T) {
	// This test checks if Docker is available
	err := ValidateDockerAvailable()
	if err != nil {
		t.Skipf("Docker not available: %v (test skipped)", err)
	}
}

func TestImageExists(t *testing.T) {
	// Skip if Docker not available
	if err := ValidateDockerAvailable(); err != nil {
		t.Skip("Docker not available, skipping test")
	}

	tests := []struct {
		name       string
		image      string
		wantExists bool
		wantErr    bool
	}{
		{
			name:       "image that likely exists (alpine)",
			image:      "alpine:latest",
			wantExists: true, // Assuming alpine is pulled in CI/local
			wantErr:    false,
		},
		{
			name:       "image that definitely doesn't exist",
			image:      "nonexistent-image-12345:v999",
			wantExists: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the alpine test, ensure the image exists first
			if tt.wantExists {
				_ = PullImage(tt.image) // Ignore error, will be caught in ImageExists
			}

			exists, err := ImageExists(tt.image)

			if (err != nil) != tt.wantErr {
				t.Errorf("ImageExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if exists != tt.wantExists {
				t.Errorf("ImageExists() = %v, want %v", exists, tt.wantExists)
			}
		})
	}
}

func TestPullImage(t *testing.T) {
	// Skip if Docker not available
	if err := ValidateDockerAvailable(); err != nil {
		t.Skip("Docker not available, skipping test")
	}

	tests := []struct {
		name    string
		image   string
		wantErr bool
	}{
		{
			name:    "pull small valid image",
			image:   "alpine:latest",
			wantErr: false,
		},
		{
			name:    "pull nonexistent image",
			image:   "nonexistent-image-12345:v999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PullImage(tt.image)

			if (err != nil) != tt.wantErr {
				t.Errorf("PullImage() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If pull succeeded, verify image exists
			if !tt.wantErr && err == nil {
				exists, checkErr := ImageExists(tt.image)
				if checkErr != nil {
					t.Errorf("ImageExists() check after pull failed: %v", checkErr)
				}
				if !exists {
					t.Errorf("Image %s was pulled but ImageExists() returned false", tt.image)
				}
			}
		})
	}
}
