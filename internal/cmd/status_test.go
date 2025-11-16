package cmd

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/detect"
)

func TestBuildEnvironmentStatus(t *testing.T) {
	tests := []struct {
		name          string
		ctx           *detect.Context
		wantRuntime   string
		wantProviders int
		wantAPIKeys   int
		wantHealthy   bool
	}{
		{
			name: "healthy environment with docker and providers",
			ctx: &detect.Context{
				Runtime: "docker",
				Providers: map[string]detect.ProviderInfo{
					"ollama": {Available: true},
					"openai": {Available: true, EnvSet: true},
				},
			},
			wantRuntime:   "docker",
			wantProviders: 2,
			wantAPIKeys:   1,
			wantHealthy:   true,
		},
		{
			name: "no runtime",
			ctx: &detect.Context{
				Runtime: "",
				Providers: map[string]detect.ProviderInfo{
					"ollama": {Available: true},
				},
			},
			wantRuntime:   "",
			wantProviders: 1,
			wantAPIKeys:   0,
			wantHealthy:   false,
		},
		{
			name: "no providers",
			ctx: &detect.Context{
				Runtime:   "docker",
				Providers: map[string]detect.ProviderInfo{},
			},
			wantRuntime:   "docker",
			wantProviders: 0,
			wantAPIKeys:   0,
			wantHealthy:   false,
		},
		{
			name: "providers without API keys",
			ctx: &detect.Context{
				Runtime: "podman",
				Providers: map[string]detect.ProviderInfo{
					"ollama": {Available: true},
					"claude": {Available: true, EnvSet: false},
				},
			},
			wantRuntime:   "podman",
			wantProviders: 2,
			wantAPIKeys:   0,
			wantHealthy:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildEnvironmentStatus(tt.ctx)

			if got.Runtime != tt.wantRuntime {
				t.Errorf("Runtime = %s, want %s", got.Runtime, tt.wantRuntime)
			}
			if len(got.Providers) != tt.wantProviders {
				t.Errorf("Providers count = %d, want %d", len(got.Providers), tt.wantProviders)
			}
			if got.APIKeys != tt.wantAPIKeys {
				t.Errorf("APIKeys = %d, want %d", got.APIKeys, tt.wantAPIKeys)
			}
			if got.Healthy != tt.wantHealthy {
				t.Errorf("Healthy = %v, want %v", got.Healthy, tt.wantHealthy)
			}
		})
	}
}

func TestBuildProjectStatus(t *testing.T) {
	tests := []struct {
		name         string
		ctx          *detect.Context
		wantGitRepo  bool
		wantGitDirty bool
	}{
		{
			name: "git repository clean",
			ctx: &detect.Context{
				Git: detect.GitContext{
					Initialized: true,
					Branch:      "main",
					Dirty:       false,
					Uncommitted: 0,
				},
			},
			wantGitRepo:  true,
			wantGitDirty: false,
		},
		{
			name: "git repository dirty",
			ctx: &detect.Context{
				Git: detect.GitContext{
					Initialized: true,
					Branch:      "feature",
					Dirty:       true,
					Uncommitted: 5,
				},
			},
			wantGitRepo:  true,
			wantGitDirty: true,
		},
		{
			name: "not a git repository",
			ctx: &detect.Context{
				Git: detect.GitContext{
					Initialized: false,
				},
			},
			wantGitRepo:  false,
			wantGitDirty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildProjectStatus(tt.ctx)

			if got.GitRepo != tt.wantGitRepo {
				t.Errorf("GitRepo = %v, want %v", got.GitRepo, tt.wantGitRepo)
			}
			if got.GitDirty != tt.wantGitDirty {
				t.Errorf("GitDirty = %v, want %v", got.GitDirty, tt.wantGitDirty)
			}
			if tt.wantGitRepo && got.GitBranch == "" {
				t.Error("GitBranch should not be empty for git repository")
			}
		})
	}
}

func TestAnalyzeStatus(t *testing.T) {
	tests := []struct {
		name          string
		report        *StatusReport
		wantIssues    int
		wantWarnings  int
		wantNextSteps int
	}{
		{
			name: "no runtime",
			report: &StatusReport{
				Environment: EnvironmentStatus{
					Runtime:   "",
					Providers: []string{},
					Healthy:   false,
				},
				Project: ProjectStatus{
					Initialized: true,
				},
			},
			wantIssues:    2, // no runtime + no providers
			wantNextSteps: 3, // install docker, install providers, create spec
		},
		{
			name: "not initialized",
			report: &StatusReport{
				Environment: EnvironmentStatus{
					Runtime:   "docker",
					Providers: []string{"ollama"},
					Healthy:   true,
				},
				Project: ProjectStatus{
					Initialized: false,
				},
			},
			wantIssues:    1,
			wantNextSteps: 1,
		},
		{
			name: "no spec",
			report: &StatusReport{
				Environment: EnvironmentStatus{
					Runtime:   "docker",
					Providers: []string{"ollama"},
					Healthy:   true,
				},
				Project: ProjectStatus{
					Initialized: true,
				},
				Spec: SpecStatus{
					Exists: false,
				},
			},
			wantIssues:    0,
			wantNextSteps: 1,
		},
		{
			name: "spec not locked",
			report: &StatusReport{
				Environment: EnvironmentStatus{
					Runtime:   "docker",
					Providers: []string{"ollama"},
					Healthy:   true,
				},
				Project: ProjectStatus{
					Initialized: true,
				},
				Spec: SpecStatus{
					Exists: true,
					Locked: false,
				},
			},
			wantIssues:    0,
			wantWarnings:  1,
			wantNextSteps: 1,
		},
		{
			name: "ready to build",
			report: &StatusReport{
				Environment: EnvironmentStatus{
					Runtime:   "docker",
					Providers: []string{"ollama"},
					Healthy:   true,
				},
				Project: ProjectStatus{
					Initialized: true,
				},
				Spec: SpecStatus{
					Exists: true,
					Locked: true,
				},
				Plan: PlanStatus{
					Exists: true,
				},
			},
			wantIssues:    0,
			wantWarnings:  0,
			wantNextSteps: 1, // Execute plan
		},
		{
			name: "git dirty warning",
			report: &StatusReport{
				Environment: EnvironmentStatus{
					Runtime:   "docker",
					Providers: []string{"ollama"},
					Healthy:   true,
				},
				Project: ProjectStatus{
					Initialized: true,
					GitRepo:     true,
					GitDirty:    true,
				},
				Spec: SpecStatus{
					Exists: true,
					Locked: true,
				},
				Plan: PlanStatus{
					Exists: true,
				},
			},
			wantIssues:    0,
			wantWarnings:  1, // git dirty
			wantNextSteps: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset report slices
			tt.report.Issues = []string{}
			tt.report.Warnings = []string{}
			tt.report.NextSteps = []string{}

			analyzeStatus(tt.report)

			if len(tt.report.Issues) != tt.wantIssues {
				t.Errorf("Issues count = %d, want %d. Issues: %v",
					len(tt.report.Issues), tt.wantIssues, tt.report.Issues)
			}
			if len(tt.report.Warnings) != tt.wantWarnings {
				t.Errorf("Warnings count = %d, want %d. Warnings: %v",
					len(tt.report.Warnings), tt.wantWarnings, tt.report.Warnings)
			}
			if len(tt.report.NextSteps) != tt.wantNextSteps {
				t.Errorf("NextSteps count = %d, want %d. NextSteps: %v",
					len(tt.report.NextSteps), tt.wantNextSteps, tt.report.NextSteps)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{
			name: "just now",
			time: now.Add(-30 * time.Second),
			want: "just now",
		},
		{
			name: "1 minute ago",
			time: now.Add(-1 * time.Minute),
			want: "1 minute ago",
		},
		{
			name: "5 minutes ago",
			time: now.Add(-5 * time.Minute),
			want: "5 minutes ago",
		},
		{
			name: "1 hour ago",
			time: now.Add(-1 * time.Hour),
			want: "1 hour ago",
		},
		{
			name: "3 hours ago",
			time: now.Add(-3 * time.Hour),
			want: "3 hours ago",
		},
		{
			name: "1 day ago",
			time: now.Add(-24 * time.Hour),
			want: "1 day ago",
		},
		{
			name: "5 days ago",
			time: now.Add(-5 * 24 * time.Hour),
			want: "5 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTime(tt.time)
			if got != tt.want {
				t.Errorf("formatTime() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestBuildSpecStatus(t *testing.T) {
	// This is primarily integration-tested as it depends on filesystem
	// We can test the structure
	status := buildSpecStatus()

	// Should always return a valid status even if files don't exist
	if status.Features < 0 {
		t.Error("Features count should not be negative")
	}
}

func TestBuildPlanStatus(t *testing.T) {
	// This is primarily integration-tested as it depends on filesystem
	// We can test the structure
	status := buildPlanStatus()

	// Should always return a valid status even if files don't exist
	if status.Tasks < 0 {
		t.Error("Tasks count should not be negative")
	}
}
