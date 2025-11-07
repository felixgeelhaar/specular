package detect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test helper functions

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{
			name:  "item exists",
			slice: []string{"go", "python", "javascript"},
			item:  "python",
			want:  true,
		},
		{
			name:  "item does not exist",
			slice: []string{"go", "python", "javascript"},
			item:  "rust",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			item:  "go",
			want:  false,
		},
		{
			name:  "nil slice",
			slice: nil,
			item:  "go",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.item)
			if got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasInFile(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := `{
  "name": "test-project",
  "dependencies": {
    "react": "^18.0.0",
    "express": "^4.18.0"
  }
}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		filename     string
		searchString string
		want         bool
	}{
		{
			name:         "string exists",
			filename:     testFile,
			searchString: "react",
			want:         true,
		},
		{
			name:         "string does not exist",
			filename:     testFile,
			searchString: "vue",
			want:         false,
		},
		{
			name:         "file does not exist",
			filename:     filepath.Join(tmpDir, "nonexistent.txt"),
			searchString: "anything",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasInFile(tt.filename, tt.searchString)
			if got != tt.want {
				t.Errorf("hasInFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test Context methods

func TestContext_GetRecommendedProviders(t *testing.T) {
	tests := []struct {
		name      string
		ctx       *Context
		wantCount int
		wantFirst string
	}{
		{
			name: "ollama available prioritized",
			ctx: &Context{
				Providers: map[string]ProviderInfo{
					"ollama": {Available: true},
					"openai": {Available: true, EnvSet: true},
				},
			},
			wantCount: 2,
			wantFirst: "ollama",
		},
		{
			name: "only API providers with keys",
			ctx: &Context{
				Providers: map[string]ProviderInfo{
					"openai":    {Available: true, EnvSet: true},
					"anthropic": {Available: true, EnvSet: true},
					"gemini":    {Available: true, EnvSet: false},
				},
			},
			wantCount: 2,
		},
		{
			name: "no providers - recommends ollama",
			ctx: &Context{
				Providers: map[string]ProviderInfo{},
			},
			wantCount: 1,
			wantFirst: "ollama",
		},
		{
			name: "providers available but no API keys",
			ctx: &Context{
				Providers: map[string]ProviderInfo{
					"openai": {Available: true, EnvSet: false},
					"claude": {Available: true, EnvSet: false},
				},
			},
			wantCount: 1,
			wantFirst: "ollama",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ctx.GetRecommendedProviders()
			if len(got) != tt.wantCount {
				t.Errorf("GetRecommendedProviders() count = %d, want %d", len(got), tt.wantCount)
			}
			if tt.wantFirst != "" && len(got) > 0 && got[0] != tt.wantFirst {
				t.Errorf("GetRecommendedProviders() first = %s, want %s", got[0], tt.wantFirst)
			}
		})
	}
}

func TestContext_Summary(t *testing.T) {
	ctx := &Context{
		Runtime: "docker",
		Docker: ContainerRuntime{
			Available: true,
			Version:   "24.0.7",
			Running:   true,
		},
		Providers: map[string]ProviderInfo{
			"ollama": {
				Name:      "ollama",
				Available: true,
				Type:      "local",
				Version:   "0.1.0",
			},
			"openai": {
				Name:      "openai",
				Available: true,
				Type:      "api",
				EnvVar:    "OPENAI_API_KEY",
				EnvSet:    true,
			},
		},
		Languages:  []string{"go", "javascript"},
		Frameworks: []string{"react", "gin"},
		Git: GitContext{
			Initialized: true,
			Root:        "/Users/test/project",
			Branch:      "main",
			Dirty:       true,
			Uncommitted: 5,
		},
		CI: CIInfo{
			Detected: true,
			Name:     "github",
		},
	}

	summary := ctx.Summary()

	// Check that summary contains expected sections
	expectedStrings := []string{
		"Detected Context:",
		"Container Runtime:",
		"docker",
		"24.0.7",
		"AI Providers:",
		"ollama",
		"openai",
		"Languages:",
		"go",
		"javascript",
		"Frameworks:",
		"react",
		"gin",
		"Git Repository:",
		"project",
		"main",
		"CI Environment:",
		"github",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(summary, expected) {
			t.Errorf("Summary() missing expected string %q", expected)
		}
	}
}

func TestContext_Summary_Minimal(t *testing.T) {
	ctx := &Context{
		Providers: map[string]ProviderInfo{},
	}

	summary := ctx.Summary()

	// Should still have basic structure
	if !strings.Contains(summary, "Detected Context:") {
		t.Error("Summary() missing header")
	}
	if !strings.Contains(summary, "Container Runtime: Not detected") {
		t.Error("Summary() should indicate no container runtime")
	}
}

// Test struct initialization

func TestContainerRuntime_Zero(t *testing.T) {
	runtime := ContainerRuntime{}
	if runtime.Available {
		t.Error("Zero ContainerRuntime should not be available")
	}
	if runtime.Running {
		t.Error("Zero ContainerRuntime should not be running")
	}
	if runtime.Version != "" {
		t.Error("Zero ContainerRuntime should have empty version")
	}
}

func TestProviderInfo_Zero(t *testing.T) {
	info := ProviderInfo{}
	if info.Available {
		t.Error("Zero ProviderInfo should not be available")
	}
	if info.EnvSet {
		t.Error("Zero ProviderInfo should not have env set")
	}
}

func TestGitContext_Zero(t *testing.T) {
	git := GitContext{}
	if git.Initialized {
		t.Error("Zero GitContext should not be initialized")
	}
	if git.Dirty {
		t.Error("Zero GitContext should not be dirty")
	}
	if git.Uncommitted != 0 {
		t.Error("Zero GitContext should have 0 uncommitted")
	}
}

func TestCIInfo_Zero(t *testing.T) {
	ci := CIInfo{}
	if ci.Detected {
		t.Error("Zero CIInfo should not be detected")
	}
	if ci.Name != "" {
		t.Error("Zero CIInfo should have empty name")
	}
}

// Integration test markers
// These functions require external commands and should be tested in integration tests:
// - DetectAll()
// - detectDocker()
// - detectPodman()
// - detectOllama()
// - detectClaude()
// - detectOpenAI()
// - detectGemini()
// - detectAnthropic()
// - detectLanguagesAndFrameworks() (can be partially tested with temp files)
// - detectGit()
// - detectCI()

func TestDetectLanguagesAndFrameworks_EmptyDirectory(t *testing.T) {
	// Create a temporary empty directory
	tmpDir := t.TempDir()

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	languages, frameworks := detectLanguagesAndFrameworks()

	if len(languages) != 0 {
		t.Errorf("Empty directory should detect 0 languages, got %d", len(languages))
	}
	if len(frameworks) != 0 {
		t.Errorf("Empty directory should detect 0 frameworks, got %d", len(frameworks))
	}
}

func TestDetectLanguagesAndFrameworks_GoProject(t *testing.T) {
	// Create a temporary directory with go.mod
	tmpDir := t.TempDir()

	goModContent := `module github.com/test/project

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	languages, frameworks := detectLanguagesAndFrameworks()

	if !contains(languages, "go") {
		t.Error("Should detect Go language")
	}
	if !contains(frameworks, "gin") {
		t.Error("Should detect Gin framework")
	}
}

func TestDetectLanguagesAndFrameworks_NodeProject(t *testing.T) {
	// Create a temporary directory with package.json
	tmpDir := t.TempDir()

	packageJSON := `{
  "name": "test-project",
  "dependencies": {
    "react": "^18.0.0",
    "express": "^4.18.0"
  }
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	languages, frameworks := detectLanguagesAndFrameworks()

	if !contains(languages, "javascript") {
		t.Error("Should detect JavaScript language")
	}
	if !contains(frameworks, "react") {
		t.Error("Should detect React framework")
	}
	if !contains(frameworks, "express") {
		t.Error("Should detect Express framework")
	}
}

func TestDetectCI(t *testing.T) {
	// Save original env vars
	origGitHub := os.Getenv("GITHUB_ACTIONS")
	origGitLab := os.Getenv("GITLAB_CI")
	defer func() {
		if origGitHub != "" {
			os.Setenv("GITHUB_ACTIONS", origGitHub)
		} else {
			os.Unsetenv("GITHUB_ACTIONS")
		}
		if origGitLab != "" {
			os.Setenv("GITLAB_CI", origGitLab)
		} else {
			os.Unsetenv("GITLAB_CI")
		}
	}()

	tests := []struct {
		name       string
		envVar     string
		envValue   string
		wantName   string
		wantDetect bool
	}{
		{
			name:       "GitHub Actions",
			envVar:     "GITHUB_ACTIONS",
			envValue:   "true",
			wantName:   "github",
			wantDetect: true,
		},
		{
			name:       "GitLab CI",
			envVar:     "GITLAB_CI",
			envValue:   "true",
			wantName:   "gitlab",
			wantDetect: true,
		},
		{
			name:       "No CI",
			envVar:     "",
			envValue:   "",
			wantName:   "",
			wantDetect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all CI env vars
			os.Unsetenv("GITHUB_ACTIONS")
			os.Unsetenv("GITLAB_CI")
			os.Unsetenv("JENKINS_HOME")
			os.Unsetenv("CIRCLECI")
			os.Unsetenv("TRAVIS")
			os.Unsetenv("BUILDKITE")

			// Set test env var if specified
			if tt.envVar != "" {
				os.Setenv(tt.envVar, tt.envValue)
			}

			ci := detectCI()

			if ci.Detected != tt.wantDetect {
				t.Errorf("detectCI().Detected = %v, want %v", ci.Detected, tt.wantDetect)
			}
			if ci.Name != tt.wantName {
				t.Errorf("detectCI().Name = %s, want %s", ci.Name, tt.wantName)
			}
		})
	}
}
