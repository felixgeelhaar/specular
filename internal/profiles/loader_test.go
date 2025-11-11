package profiles

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoader_LoadBuiltin(t *testing.T) {
	loader := NewLoader()

	tests := []struct {
		name        string
		profileName string
		wantErr     bool
	}{
		{
			name:        "load default profile",
			profileName: "default",
			wantErr:     false,
		},
		{
			name:        "load ci profile",
			profileName: "ci",
			wantErr:     false,
		},
		{
			name:        "load strict profile",
			profileName: "strict",
			wantErr:     false,
		},
		{
			name:        "nonexistent profile",
			profileName: "nonexistent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := loader.Load(tt.profileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Loader.Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if profile.Name != tt.profileName {
					t.Errorf("expected profile name %s, got %s", tt.profileName, profile.Name)
				}
				if err := profile.Validate(); err != nil {
					t.Errorf("loaded profile failed validation: %v", err)
				}
			}
		})
	}
}

func TestLoader_DefaultProfileValidation(t *testing.T) {
	loader := NewLoader()
	profile, err := loader.Load("default")
	if err != nil {
		t.Fatalf("failed to load default profile: %v", err)
	}

	// Validate structure
	if profile.Approvals.Mode != ApprovalModeCriticalOnly {
		t.Errorf("expected critical_only mode, got %s", profile.Approvals.Mode)
	}
	if !profile.Approvals.Interactive {
		t.Error("expected interactive to be true")
	}
	if profile.Safety.MaxSteps != 12 {
		t.Errorf("expected max_steps 12, got %d", profile.Safety.MaxSteps)
	}
	if profile.Safety.Timeout != 25*time.Minute {
		t.Errorf("expected timeout 25m, got %s", profile.Safety.Timeout)
	}
	if profile.Safety.MaxCostUSD != 5.0 {
		t.Errorf("expected max_cost_usd 5.0, got %.2f", profile.Safety.MaxCostUSD)
	}
	if profile.Execution.EnableTUI != true {
		t.Error("expected enable_tui to be true")
	}
}

func TestLoader_CIProfileValidation(t *testing.T) {
	loader := NewLoader()
	profile, err := loader.Load("ci")
	if err != nil {
		t.Fatalf("failed to load ci profile: %v", err)
	}

	// Validate CI-specific settings
	if profile.Approvals.Mode != ApprovalModeNone {
		t.Errorf("expected none mode, got %s", profile.Approvals.Mode)
	}
	if profile.Approvals.Interactive {
		t.Error("expected interactive to be false")
	}
	if profile.Safety.MaxSteps != 8 {
		t.Errorf("expected max_steps 8, got %d", profile.Safety.MaxSteps)
	}
	if profile.Execution.JSONOutput != true {
		t.Error("expected json_output to be true")
	}
	if profile.Execution.EnableTUI != false {
		t.Error("expected enable_tui to be false")
	}
}

func TestLoader_StrictProfileValidation(t *testing.T) {
	loader := NewLoader()
	profile, err := loader.Load("strict")
	if err != nil {
		t.Fatalf("failed to load strict profile: %v", err)
	}

	// Validate strict-specific settings
	if profile.Approvals.Mode != ApprovalModeAll {
		t.Errorf("expected all mode, got %s", profile.Approvals.Mode)
	}
	if profile.Safety.MaxSteps != 5 {
		t.Errorf("expected max_steps 5, got %d", profile.Safety.MaxSteps)
	}
	if profile.Safety.MaxCostUSD != 1.0 {
		t.Errorf("expected max_cost_usd 1.0, got %.2f", profile.Safety.MaxCostUSD)
	}
}

func TestLoader_List(t *testing.T) {
	loader := NewLoader()
	names, err := loader.List()
	if err != nil {
		t.Fatalf("Loader.List() error = %v", err)
	}

	// Should have at least the 3 built-in profiles
	if len(names) < 3 {
		t.Errorf("expected at least 3 profiles, got %d", len(names))
	}

	// Check that built-in profiles are present
	expected := map[string]bool{
		"default": false,
		"ci":      false,
		"strict":  false,
	}
	for _, name := range names {
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected built-in profile %s not found", name)
		}
	}
}

func TestLoader_Caching(t *testing.T) {
	loader := NewLoader()

	// Load profile twice
	profile1, err := loader.Load("default")
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}

	profile2, err := loader.Load("default")
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}

	// Should return the same cached instance
	if profile1 != profile2 {
		t.Error("expected cached profile to be returned")
	}
}

func TestLoader_LoadFromFile(t *testing.T) {
	// Create a temporary profile file
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "test.profiles.yaml")

	content := `schema: "specular.auto.profiles/v1"
profiles:
  test:
    description: "Test profile"
    approvals:
      mode: "critical_only"
      interactive: true
      auto_approve: []
      require_approval: []
    safety:
      max_steps: 10
      timeout: "20m"
      max_cost_usd: 3.0
      max_cost_per_task: 0.3
      max_retries: 2
      require_policy: true
      allowed_step_types: []
      blocked_step_types: []
    routing:
      preferred_agent: "cline"
      fallback_agent: "openai"
      temperature: 0.6
      model_preferences: {}
    policies:
      enabled: true
      policy_files: []
      enforcement: "strict"
    execution:
      trace_logging: true
      save_patches: false
      checkpoint_frequency: 1
      json_output: false
      enable_tui: true
    hooks: {}
`

	if err := os.WriteFile(profilePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test profile: %v", err)
	}

	loader := NewLoader()
	profile, err := loader.LoadFromFile(profilePath, "test")
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if profile.Name != "test" {
		t.Errorf("expected profile name 'test', got %s", profile.Name)
	}
	if profile.Safety.MaxSteps != 10 {
		t.Errorf("expected max_steps 10, got %d", profile.Safety.MaxSteps)
	}
	if profile.Safety.Timeout != 20*time.Minute {
		t.Errorf("expected timeout 20m, got %s", profile.Safety.Timeout)
	}
}

func TestMergeWithCLIFlags(t *testing.T) {
	profile := &Profile{
		Name:        "test",
		Description: "Test profile",
		Approvals: ApprovalConfig{
			Mode:        ApprovalModeCriticalOnly,
			Interactive: true,
		},
		Safety: SafetyConfig{
			MaxSteps:       12,
			Timeout:        25 * time.Minute,
			MaxCostUSD:     5.0,
			MaxCostPerTask: 0.5,
			MaxRetries:     3,
		},
		Execution: ExecutionConfig{
			TraceLogging: true,
			SavePatches:  false,
			EnableTUI:    true,
		},
	}

	t.Run("override require approval", func(t *testing.T) {
		requireApproval := false
		flags := &CLIFlags{
			RequireApproval: &requireApproval,
		}

		merged := MergeWithCLIFlags(profile, flags)
		if merged.Approvals.Interactive {
			t.Error("expected interactive to be false")
		}
		if merged.Approvals.Mode != ApprovalModeNone {
			t.Errorf("expected mode none, got %s", merged.Approvals.Mode)
		}
	})

	t.Run("override max steps", func(t *testing.T) {
		maxSteps := 20
		flags := &CLIFlags{
			MaxSteps: &maxSteps,
		}

		merged := MergeWithCLIFlags(profile, flags)
		if merged.Safety.MaxSteps != 20 {
			t.Errorf("expected max_steps 20, got %d", merged.Safety.MaxSteps)
		}
	})

	t.Run("override timeout", func(t *testing.T) {
		timeout := 30 * time.Minute
		flags := &CLIFlags{
			Timeout: &timeout,
		}

		merged := MergeWithCLIFlags(profile, flags)
		if merged.Safety.Timeout != 30*time.Minute {
			t.Errorf("expected timeout 30m, got %s", merged.Safety.Timeout)
		}
	})

	t.Run("override max cost", func(t *testing.T) {
		maxCost := 10.0
		flags := &CLIFlags{
			MaxCostUSD: &maxCost,
		}

		merged := MergeWithCLIFlags(profile, flags)
		if merged.Safety.MaxCostUSD != 10.0 {
			t.Errorf("expected max_cost_usd 10.0, got %.2f", merged.Safety.MaxCostUSD)
		}
	})

	t.Run("override max cost per task", func(t *testing.T) {
		maxCostPerTask := 1.0
		flags := &CLIFlags{
			MaxCostPerTask: &maxCostPerTask,
		}

		merged := MergeWithCLIFlags(profile, flags)
		if merged.Safety.MaxCostPerTask != 1.0 {
			t.Errorf("expected max_cost_per_task 1.0, got %.2f", merged.Safety.MaxCostPerTask)
		}
	})

	t.Run("override trace logging", func(t *testing.T) {
		trace := false
		flags := &CLIFlags{
			Trace: &trace,
		}

		merged := MergeWithCLIFlags(profile, flags)
		if merged.Execution.TraceLogging {
			t.Error("expected trace_logging to be false")
		}
	})

	t.Run("override save patches", func(t *testing.T) {
		savePatches := true
		flags := &CLIFlags{
			SavePatches: &savePatches,
		}

		merged := MergeWithCLIFlags(profile, flags)
		if !merged.Execution.SavePatches {
			t.Error("expected save_patches to be true")
		}
	})

	t.Run("no flags preserves profile", func(t *testing.T) {
		flags := &CLIFlags{}

		merged := MergeWithCLIFlags(profile, flags)
		if merged.Safety.MaxSteps != 12 {
			t.Errorf("expected max_steps 12, got %d", merged.Safety.MaxSteps)
		}
		if merged.Safety.MaxCostUSD != 5.0 {
			t.Errorf("expected max_cost_usd 5.0, got %.2f", merged.Safety.MaxCostUSD)
		}
	})
}

func TestLoader_EnvironmentVariableExpansion(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_WEBHOOK_URL", "https://example.com/webhook")
	defer os.Unsetenv("TEST_WEBHOOK_URL")

	// Create a temporary profile file with environment variable
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "test.profiles.yaml")

	content := `schema: "specular.auto.profiles/v1"
profiles:
  test:
    description: "Test profile with env var"
    approvals:
      mode: "none"
      interactive: false
      auto_approve: []
      require_approval: []
    safety:
      max_steps: 10
      timeout: "20m"
      max_cost_usd: 3.0
      max_cost_per_task: 0.3
      max_retries: 2
      require_policy: false
      allowed_step_types: []
      blocked_step_types: []
    routing:
      preferred_agent: "cline"
      fallback_agent: "openai"
      temperature: 0.6
      model_preferences: {}
    policies:
      enabled: false
      policy_files: []
      enforcement: "none"
    execution:
      trace_logging: false
      save_patches: false
      checkpoint_frequency: 1
      json_output: false
      enable_tui: false
    hooks:
      on_complete:
        - type: "webhook"
          url: "${TEST_WEBHOOK_URL}"
`

	if err := os.WriteFile(profilePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test profile: %v", err)
	}

	loader := NewLoader()
	profile, err := loader.LoadFromFile(profilePath, "test")
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	// Environment variable should be expanded
	if len(profile.Hooks.OnComplete) == 0 {
		t.Fatal("expected hooks.on_complete to have at least one hook")
	}

	hook := profile.Hooks.OnComplete[0]
	if url, ok := hook.Config["url"].(string); ok {
		if url != "https://example.com/webhook" {
			t.Errorf("expected URL to be expanded, got %s", url)
		}
	} else {
		t.Error("expected url field in hook config")
	}
}
