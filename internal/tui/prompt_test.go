package tui

import (
	"os"
	"testing"
)

func TestIsInteractive(t *testing.T) {
	// Note: This test's result depends on how tests are run
	// It will return false in CI environments
	result := IsInteractive()
	// We can't assert a specific value since it depends on the environment
	// Just ensure the function doesn't panic
	_ = result
}

func TestShouldPrompt(t *testing.T) {
	tests := []struct {
		name       string
		envVars    map[string]string
		wantPrompt bool
	}{
		{
			name:       "no CI environment",
			envVars:    map[string]string{},
			wantPrompt: true, // Assumes tests run in interactive mode
		},
		{
			name: "GitHub Actions",
			envVars: map[string]string{
				"GITHUB_ACTIONS": "true",
			},
			wantPrompt: false,
		},
		{
			name: "GitLab CI",
			envVars: map[string]string{
				"GITLAB_CI": "true",
			},
			wantPrompt: false,
		},
		{
			name: "Jenkins",
			envVars: map[string]string{
				"JENKINS_URL": "http://jenkins.local",
			},
			wantPrompt: false,
		},
		{
			name: "Generic CI",
			envVars: map[string]string{
				"CI": "true",
			},
			wantPrompt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
			}

			// Set test environment
			for key, value := range tt.envVars {
				if err := os.Setenv(key, value); err != nil {
					t.Fatalf("failed to set env var %s: %v", key, err)
				}
			}

			// Restore original environment after test
			defer func() {
				for key, value := range originalEnv {
					if value == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, value)
					}
				}
				// Clean up test env vars
				for key := range tt.envVars {
					if _, exists := originalEnv[key]; !exists {
						os.Unsetenv(key)
					}
				}
			}()

			got := ShouldPrompt()

			// In CI environments, prompts should be disabled
			if len(tt.envVars) > 0 && got != tt.wantPrompt {
				t.Errorf("ShouldPrompt() = %v, want %v (with env: %v)", got, tt.wantPrompt, tt.envVars)
			}
		})
	}
}

func TestPromptForSelect(t *testing.T) {
	// Test error case: no options
	_, err := PromptForSelect("Choose:", []string{})
	if err == nil {
		t.Error("expected error when no options provided, got nil")
	}
}

func TestPromptForMultiSelect(t *testing.T) {
	// Test error case: no options
	_, err := PromptForMultiSelect("Choose:", []string{})
	if err == nil {
		t.Error("expected error when no options provided, got nil")
	}
}

// Note: Interactive prompts (PromptForString, PromptForConfirmation, etc.)
// are difficult to test in automated tests since they require user input.
// These should be tested manually or with integration tests that mock stdin.
