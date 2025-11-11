package profiles

import (
	"testing"
	"time"
)

func TestApprovalConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ApprovalConfig
		wantErr bool
	}{
		{
			name: "valid all mode",
			config: ApprovalConfig{
				Mode:        ApprovalModeAll,
				Interactive: true,
			},
			wantErr: false,
		},
		{
			name: "valid critical_only mode",
			config: ApprovalConfig{
				Mode:            ApprovalModeCriticalOnly,
				Interactive:     true,
				AutoApprove:     []string{"spec:update", "plan:gen"},
				RequireApproval: []string{"spec:lock", "build:run"},
			},
			wantErr: false,
		},
		{
			name: "valid none mode",
			config: ApprovalConfig{
				Mode:        ApprovalModeNone,
				Interactive: false,
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: ApprovalConfig{
				Mode:        "invalid",
				Interactive: true,
			},
			wantErr: true,
		},
		{
			name: "invalid step type in auto_approve",
			config: ApprovalConfig{
				Mode:        ApprovalModeCriticalOnly,
				Interactive: true,
				AutoApprove: []string{"invalid:type"},
			},
			wantErr: true,
		},
		{
			name: "invalid step type in require_approval",
			config: ApprovalConfig{
				Mode:            ApprovalModeCriticalOnly,
				Interactive:     true,
				RequireApproval: []string{"invalid:type"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ApprovalConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSafetyConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  SafetyConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: SafetyConfig{
				MaxSteps:         12,
				Timeout:          25 * time.Minute,
				MaxCostUSD:       5.0,
				MaxCostPerTask:   0.5,
				MaxRetries:       3,
				RequirePolicy:    true,
				AllowedStepTypes: []string{"spec:update", "spec:lock", "plan:gen", "build:run"},
			},
			wantErr: false,
		},
		{
			name: "max_steps too low",
			config: SafetyConfig{
				MaxSteps:       0,
				Timeout:        25 * time.Minute,
				MaxCostUSD:     5.0,
				MaxCostPerTask: 0.5,
				MaxRetries:     3,
			},
			wantErr: true,
		},
		{
			name: "max_steps too high",
			config: SafetyConfig{
				MaxSteps:       101,
				Timeout:        25 * time.Minute,
				MaxCostUSD:     5.0,
				MaxCostPerTask: 0.5,
				MaxRetries:     3,
			},
			wantErr: true,
		},
		{
			name: "timeout too low",
			config: SafetyConfig{
				MaxSteps:       12,
				Timeout:        0,
				MaxCostUSD:     5.0,
				MaxCostPerTask: 0.5,
				MaxRetries:     3,
			},
			wantErr: true,
		},
		{
			name: "max_cost_usd too low",
			config: SafetyConfig{
				MaxSteps:       12,
				Timeout:        25 * time.Minute,
				MaxCostUSD:     0,
				MaxCostPerTask: 0.5,
				MaxRetries:     3,
			},
			wantErr: true,
		},
		{
			name: "max_cost_per_task exceeds max_cost_usd",
			config: SafetyConfig{
				MaxSteps:       12,
				Timeout:        25 * time.Minute,
				MaxCostUSD:     5.0,
				MaxCostPerTask: 10.0,
				MaxRetries:     3,
			},
			wantErr: true,
		},
		{
			name: "max_retries too high",
			config: SafetyConfig{
				MaxSteps:       12,
				Timeout:        25 * time.Minute,
				MaxCostUSD:     5.0,
				MaxCostPerTask: 0.5,
				MaxRetries:     11,
			},
			wantErr: true,
		},
		{
			name: "invalid allowed step type",
			config: SafetyConfig{
				MaxSteps:         12,
				Timeout:          25 * time.Minute,
				MaxCostUSD:       5.0,
				MaxCostPerTask:   0.5,
				MaxRetries:       3,
				AllowedStepTypes: []string{"invalid:type"},
			},
			wantErr: true,
		},
		{
			name: "invalid blocked step type",
			config: SafetyConfig{
				MaxSteps:         12,
				Timeout:          25 * time.Minute,
				MaxCostUSD:       5.0,
				MaxCostPerTask:   0.5,
				MaxRetries:       3,
				BlockedStepTypes: []string{"invalid:type"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SafetyConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoutingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  RoutingConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: RoutingConfig{
				PreferredAgent: "cline",
				FallbackAgent:  "openai",
				Temperature:    0.7,
			},
			wantErr: false,
		},
		{
			name: "missing preferred agent",
			config: RoutingConfig{
				PreferredAgent: "",
				FallbackAgent:  "openai",
				Temperature:    0.7,
			},
			wantErr: true,
		},
		{
			name: "missing fallback agent",
			config: RoutingConfig{
				PreferredAgent: "cline",
				FallbackAgent:  "",
				Temperature:    0.7,
			},
			wantErr: true,
		},
		{
			name: "temperature too low",
			config: RoutingConfig{
				PreferredAgent: "cline",
				FallbackAgent:  "openai",
				Temperature:    -0.1,
			},
			wantErr: true,
		},
		{
			name: "temperature too high",
			config: RoutingConfig{
				PreferredAgent: "cline",
				FallbackAgent:  "openai",
				Temperature:    1.1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RoutingConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPolicyConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  PolicyConfig
		wantErr bool
	}{
		{
			name: "valid strict",
			config: PolicyConfig{
				Enabled:     true,
				Enforcement: PolicyEnforcementStrict,
			},
			wantErr: false,
		},
		{
			name: "valid warn",
			config: PolicyConfig{
				Enabled:     true,
				Enforcement: PolicyEnforcementWarn,
			},
			wantErr: false,
		},
		{
			name: "valid none",
			config: PolicyConfig{
				Enabled:     false,
				Enforcement: PolicyEnforcementNone,
			},
			wantErr: false,
		},
		{
			name: "invalid enforcement",
			config: PolicyConfig{
				Enabled:     true,
				Enforcement: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PolicyConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecutionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ExecutionConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ExecutionConfig{
				TraceLogging:        true,
				SavePatches:         false,
				CheckpointFrequency: 1,
				JSONOutput:          false,
				EnableTUI:           true,
			},
			wantErr: false,
		},
		{
			name: "invalid checkpoint frequency",
			config: ExecutionConfig{
				TraceLogging:        true,
				SavePatches:         false,
				CheckpointFrequency: 0,
				JSONOutput:          false,
				EnableTUI:           true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecutionConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProfile_ShouldRequireApproval(t *testing.T) {
	tests := []struct {
		name     string
		profile  Profile
		stepType string
		want     bool
	}{
		{
			name: "mode all requires approval",
			profile: Profile{
				Approvals: ApprovalConfig{
					Mode: ApprovalModeAll,
				},
			},
			stepType: "spec:update",
			want:     true,
		},
		{
			name: "mode none never requires approval",
			profile: Profile{
				Approvals: ApprovalConfig{
					Mode: ApprovalModeNone,
				},
			},
			stepType: "spec:lock",
			want:     false,
		},
		{
			name: "critical_only with critical step",
			profile: Profile{
				Approvals: ApprovalConfig{
					Mode:        ApprovalModeCriticalOnly,
					AutoApprove: []string{"spec:update", "plan:gen"},
				},
			},
			stepType: "spec:lock",
			want:     true,
		},
		{
			name: "critical_only with auto-approve step",
			profile: Profile{
				Approvals: ApprovalConfig{
					Mode:        ApprovalModeCriticalOnly,
					AutoApprove: []string{"spec:update", "plan:gen"},
				},
			},
			stepType: "spec:update",
			want:     false,
		},
		{
			name: "critical_only with explicit require",
			profile: Profile{
				Approvals: ApprovalConfig{
					Mode:            ApprovalModeCriticalOnly,
					RequireApproval: []string{"spec:update"},
				},
			},
			stepType: "spec:update",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.ShouldRequireApproval(tt.stepType)
			if got != tt.want {
				t.Errorf("Profile.ShouldRequireApproval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfile_IsStepTypeAllowed(t *testing.T) {
	tests := []struct {
		name     string
		profile  Profile
		stepType string
		want     bool
	}{
		{
			name: "blocked step",
			profile: Profile{
				Safety: SafetyConfig{
					BlockedStepTypes: []string{"build:run"},
				},
			},
			stepType: "build:run",
			want:     false,
		},
		{
			name: "allowed step with whitelist",
			profile: Profile{
				Safety: SafetyConfig{
					AllowedStepTypes: []string{"spec:update", "plan:gen"},
				},
			},
			stepType: "spec:update",
			want:     true,
		},
		{
			name: "disallowed step with whitelist",
			profile: Profile{
				Safety: SafetyConfig{
					AllowedStepTypes: []string{"spec:update", "plan:gen"},
				},
			},
			stepType: "build:run",
			want:     false,
		},
		{
			name: "no whitelist allows all",
			profile: Profile{
				Safety: SafetyConfig{
					AllowedStepTypes: []string{},
				},
			},
			stepType: "build:run",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.IsStepTypeAllowed(tt.stepType)
			if got != tt.want {
				t.Errorf("Profile.IsStepTypeAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfile_Merge(t *testing.T) {
	base := &Profile{
		Name:        "base",
		Description: "Base profile",
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
		Routing: RoutingConfig{
			PreferredAgent: "cline",
			FallbackAgent:  "openai",
			Temperature:    0.7,
		},
	}

	override := &Profile{
		Name:        "override",
		Description: "Override profile",
		Approvals: ApprovalConfig{
			Mode:        ApprovalModeNone,
			Interactive: false,
		},
		Safety: SafetyConfig{
			MaxSteps:   10,
			MaxCostUSD: 3.0,
		},
		Routing: RoutingConfig{
			Temperature: 0.5,
		},
	}

	merged := base.Merge(override)

	// Check that override values take precedence
	if merged.Name != "override" {
		t.Errorf("expected name 'override', got %s", merged.Name)
	}
	if merged.Approvals.Mode != ApprovalModeNone {
		t.Errorf("expected approval mode 'none', got %s", merged.Approvals.Mode)
	}
	if merged.Safety.MaxSteps != 10 {
		t.Errorf("expected max_steps 10, got %d", merged.Safety.MaxSteps)
	}
	if merged.Safety.MaxCostUSD != 3.0 {
		t.Errorf("expected max_cost_usd 3.0, got %.2f", merged.Safety.MaxCostUSD)
	}
	if merged.Routing.Temperature != 0.5 {
		t.Errorf("expected temperature 0.5, got %.2f", merged.Routing.Temperature)
	}

	// Check that base values are preserved for non-overridden fields
	if merged.Safety.Timeout != 25*time.Minute {
		t.Errorf("expected timeout 25m, got %s", merged.Safety.Timeout)
	}
	if merged.Safety.MaxCostPerTask != 0.5 {
		t.Errorf("expected max_cost_per_task 0.5, got %.2f", merged.Safety.MaxCostPerTask)
	}
	if merged.Routing.PreferredAgent != "cline" {
		t.Errorf("expected preferred_agent 'cline', got %s", merged.Routing.PreferredAgent)
	}
}
