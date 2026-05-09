package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSource_Local(t *testing.T) {
	// Create a temp directory for local path tests
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		source   string
		wantPath string
		wantErr  bool
	}{
		{
			name:     "relative path with dot",
			source:   tmpDir,
			wantPath: tmpDir,
			wantErr:  false,
		},
		{
			name:    "non-existent path",
			source:  "/nonexistent/path/to/plugin",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := ParseSource(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if src.Type != SourceTypeLocal {
				t.Errorf("Type = %v, want %v", src.Type, SourceTypeLocal)
			}
			if !tt.wantErr && src.Path != tt.wantPath {
				t.Errorf("Path = %v, want %v", src.Path, tt.wantPath)
			}
		})
	}
}

func TestParseSource_LocalFile(t *testing.T) {
	// Create a temp file (not directory)
	tmpFile, err := os.CreateTemp("", "plugin-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	_, err = ParseSource(tmpFile.Name())
	if err == nil {
		t.Error("ParseSource() should error for file paths (not directories)")
	}
}

func TestParseSource_GitHub(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		wantOwner   string
		wantRepo    string
		wantVersion string
		wantSubpath string
		wantErr     bool
	}{
		{
			name:      "shorthand",
			source:    "github.com/specular/slack-notifier",
			wantOwner: "specular",
			wantRepo:  "slack-notifier",
			wantErr:   false,
		},
		{
			name:        "shorthand with version",
			source:      "github.com/specular/slack-notifier@v1.0.0",
			wantOwner:   "specular",
			wantRepo:    "slack-notifier",
			wantVersion: "v1.0.0",
			wantErr:     false,
		},
		{
			name:        "shorthand with branch",
			source:      "github.com/specular/slack-notifier@main",
			wantOwner:   "specular",
			wantRepo:    "slack-notifier",
			wantVersion: "main",
			wantErr:     false,
		},
		{
			name:        "shorthand with subpath",
			source:      "github.com/specular/plugins@v1.0.0/notifiers/slack",
			wantOwner:   "specular",
			wantRepo:    "plugins",
			wantVersion: "v1.0.0",
			wantSubpath: "notifiers/slack",
			wantErr:     false,
		},
		{
			name:      "https URL",
			source:    "https://github.com/specular/my-plugin",
			wantOwner: "specular",
			wantRepo:  "my-plugin",
			wantErr:   false,
		},
		{
			name:        "https URL with version",
			source:      "https://github.com/specular/my-plugin@v2.0.0",
			wantOwner:   "specular",
			wantRepo:    "my-plugin",
			wantVersion: "v2.0.0",
			wantErr:     false,
		},
		{
			name:      "URL with .git suffix",
			source:    "https://github.com/specular/my-plugin.git",
			wantOwner: "specular",
			wantRepo:  "my-plugin",
			wantErr:   false,
		},
		{
			name:    "invalid version starts with dash",
			source:  "github.com/specular/my-plugin@--upload-pack=evil",
			wantErr: true,
		},
		{
			name:    "invalid subpath traversal",
			source:  "github.com/specular/my-plugin@v1.0.0/../../etc",
			wantErr: true,
		},
		{
			name:    "invalid subpath absolute",
			source:  "github.com/specular/my-plugin@v1.0.0//etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := ParseSource(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if src.Type != SourceTypeGitHub {
				t.Errorf("Type = %v, want %v", src.Type, SourceTypeGitHub)
			}
			if src.Owner != tt.wantOwner {
				t.Errorf("Owner = %v, want %v", src.Owner, tt.wantOwner)
			}
			if src.Repo != tt.wantRepo {
				t.Errorf("Repo = %v, want %v", src.Repo, tt.wantRepo)
			}
			if src.Version != tt.wantVersion {
				t.Errorf("Version = %v, want %v", src.Version, tt.wantVersion)
			}
			if src.Subpath != tt.wantSubpath {
				t.Errorf("Subpath = %v, want %v", src.Subpath, tt.wantSubpath)
			}
		})
	}
}

func TestParseSource_Registry(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		wantName    string
		wantVersion string
		wantErr     bool
	}{
		{
			name:     "simple name",
			source:   "registry:slack-notifier",
			wantName: "slack-notifier",
			wantErr:  false,
		},
		{
			name:        "with version",
			source:      "registry:slack-notifier@1.0.0",
			wantName:    "slack-notifier",
			wantVersion: "1.0.0",
			wantErr:     false,
		},
		{
			name:        "with semver constraint",
			source:      "registry:my-plugin@^2.0.0",
			wantName:    "my-plugin",
			wantVersion: "^2.0.0",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := ParseSource(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if src.Type != SourceTypeRegistry {
				t.Errorf("Type = %v, want %v", src.Type, SourceTypeRegistry)
			}
			if src.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", src.Name, tt.wantName)
			}
			if src.Version != tt.wantVersion {
				t.Errorf("Version = %v, want %v", src.Version, tt.wantVersion)
			}
		})
	}
}

func TestParseSource_Empty(t *testing.T) {
	_, err := ParseSource("")
	if err == nil {
		t.Error("ParseSource() should error for empty string")
	}

	_, err = ParseSource("   ")
	if err == nil {
		t.Error("ParseSource() should error for whitespace-only string")
	}
}

func TestPluginSource_String(t *testing.T) {
	tests := []struct {
		name   string
		source *PluginSource
		want   string
	}{
		{
			name: "local",
			source: &PluginSource{
				Type: SourceTypeLocal,
				Path: "/path/to/plugin",
			},
			want: "/path/to/plugin",
		},
		{
			name: "github simple",
			source: &PluginSource{
				Type:  SourceTypeGitHub,
				Owner: "user",
				Repo:  "repo",
			},
			want: "github.com/user/repo",
		},
		{
			name: "github with version",
			source: &PluginSource{
				Type:    SourceTypeGitHub,
				Owner:   "user",
				Repo:    "repo",
				Version: "v1.0.0",
			},
			want: "github.com/user/repo@v1.0.0",
		},
		{
			name: "github with subpath",
			source: &PluginSource{
				Type:    SourceTypeGitHub,
				Owner:   "user",
				Repo:    "monorepo",
				Version: "v1.0.0",
				Subpath: "plugins/my-plugin",
			},
			want: "github.com/user/monorepo@v1.0.0/plugins/my-plugin",
		},
		{
			name: "registry simple",
			source: &PluginSource{
				Type: SourceTypeRegistry,
				Name: "my-plugin",
			},
			want: "registry:my-plugin",
		},
		{
			name: "registry with version",
			source: &PluginSource{
				Type:    SourceTypeRegistry,
				Name:    "my-plugin",
				Version: "1.0.0",
			},
			want: "registry:my-plugin@1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.source.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPluginSource_GitHubURLs(t *testing.T) {
	src := &PluginSource{
		Type:    SourceTypeGitHub,
		Owner:   "specular",
		Repo:    "my-plugin",
		Version: "v1.2.3",
	}

	if url := src.GitHubURL(); url != "https://github.com/specular/my-plugin" {
		t.Errorf("GitHubURL() = %v", url)
	}

	if url := src.GitHubCloneURL(); url != "https://github.com/specular/my-plugin.git" {
		t.Errorf("GitHubCloneURL() = %v", url)
	}

	if url := src.GitHubTagTarballURL(); url != "https://github.com/specular/my-plugin/archive/refs/tags/v1.2.3.tar.gz" {
		t.Errorf("GitHubTagTarballURL() = %v", url)
	}

	// Test with no version
	srcNoVersion := &PluginSource{
		Type:  SourceTypeGitHub,
		Owner: "specular",
		Repo:  "my-plugin",
	}

	if url := srcNoVersion.GitHubTarballURL(); url != "https://github.com/specular/my-plugin/archive/refs/heads/main.tar.gz" {
		t.Errorf("GitHubTarballURL() without version = %v", url)
	}
}

func TestPluginSource_GitHubURLs_NonGitHub(t *testing.T) {
	src := &PluginSource{
		Type: SourceTypeLocal,
		Path: "/some/path",
	}

	if url := src.GitHubURL(); url != "" {
		t.Errorf("GitHubURL() for non-GitHub source = %v, want empty", url)
	}
	if url := src.GitHubCloneURL(); url != "" {
		t.Errorf("GitHubCloneURL() for non-GitHub source = %v, want empty", url)
	}
}

func TestPluginSource_IsVersioned(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"with version", "v1.0.0", true},
		{"empty version", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &PluginSource{Version: tt.version}
			if got := src.IsVersioned(); got != tt.want {
				t.Errorf("IsVersioned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPluginSource_IsLocal(t *testing.T) {
	tests := []struct {
		name string
		typ  SourceType
		want bool
	}{
		{"local", SourceTypeLocal, true},
		{"github", SourceTypeGitHub, false},
		{"registry", SourceTypeRegistry, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &PluginSource{Type: tt.typ}
			if got := src.IsLocal(); got != tt.want {
				t.Errorf("IsLocal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPluginSource_IsRemote(t *testing.T) {
	tests := []struct {
		name string
		typ  SourceType
		want bool
	}{
		{"local", SourceTypeLocal, false},
		{"github", SourceTypeGitHub, true},
		{"registry", SourceTypeRegistry, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &PluginSource{Type: tt.typ}
			if got := src.IsRemote(); got != tt.want {
				t.Errorf("IsRemote() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPluginSource_GetPluginName(t *testing.T) {
	tests := []struct {
		name   string
		source *PluginSource
		want   string
	}{
		{
			name: "local",
			source: &PluginSource{
				Type: SourceTypeLocal,
				Path: "/path/to/my-plugin",
			},
			want: "my-plugin",
		},
		{
			name: "github repo",
			source: &PluginSource{
				Type:  SourceTypeGitHub,
				Owner: "user",
				Repo:  "awesome-plugin",
			},
			want: "awesome-plugin",
		},
		{
			name: "github with subpath",
			source: &PluginSource{
				Type:    SourceTypeGitHub,
				Owner:   "user",
				Repo:    "monorepo",
				Subpath: "plugins/slack-notifier",
			},
			want: "slack-notifier",
		},
		{
			name: "registry",
			source: &PluginSource{
				Type: SourceTypeRegistry,
				Name: "my-registry-plugin",
			},
			want: "my-registry-plugin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.source.GetPluginName(); got != tt.want {
				t.Errorf("GetPluginName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPluginSource_WithVersion(t *testing.T) {
	original := &PluginSource{
		Type:  SourceTypeGitHub,
		Owner: "user",
		Repo:  "repo",
	}

	updated := original.WithVersion("v2.0.0")

	// Original should be unchanged
	if original.Version != "" {
		t.Error("WithVersion() modified original")
	}

	// Updated should have new version
	if updated.Version != "v2.0.0" {
		t.Errorf("WithVersion() Version = %v, want v2.0.0", updated.Version)
	}

	// Other fields should be preserved
	if updated.Owner != "user" || updated.Repo != "repo" {
		t.Error("WithVersion() didn't preserve other fields")
	}
}

func TestPluginSource_ValidateVersion(t *testing.T) {
	tests := []struct {
		name    string
		source  *PluginSource
		wantErr bool
	}{
		{
			name: "local with version (invalid)",
			source: &PluginSource{
				Type:    SourceTypeLocal,
				Version: "v1.0.0",
			},
			wantErr: true,
		},
		{
			name: "local without version (valid)",
			source: &PluginSource{
				Type: SourceTypeLocal,
			},
			wantErr: false,
		},
		{
			name: "github with tag",
			source: &PluginSource{
				Type:    SourceTypeGitHub,
				Version: "v1.0.0",
			},
			wantErr: false,
		},
		{
			name: "github with branch",
			source: &PluginSource{
				Type:    SourceTypeGitHub,
				Version: "main",
			},
			wantErr: false,
		},
		{
			name: "registry with semver",
			source: &PluginSource{
				Type:    SourceTypeRegistry,
				Version: "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "registry with constraint",
			source: &PluginSource{
				Type:    SourceTypeRegistry,
				Version: "^1.0.0",
			},
			wantErr: false,
		},
		{
			name: "registry with invalid version",
			source: &PluginSource{
				Type:    SourceTypeRegistry,
				Version: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.source.ValidateVersion()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMustParseSource(t *testing.T) {
	// Should not panic for valid source
	tmpDir := t.TempDir()
	src := MustParseSource(tmpDir)
	if src == nil {
		t.Error("MustParseSource() returned nil for valid source")
	}
}

func TestMustParseSource_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseSource() should panic for invalid source")
		}
	}()

	MustParseSource("/nonexistent/path/that/does/not/exist")
}

func TestParseSource_CurrentDirectory(t *testing.T) {
	// Create a temp directory and change to it
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Create a subdirectory
	pluginDir := filepath.Join(tmpDir, "my-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	src, err := ParseSource("./my-plugin")
	if err != nil {
		t.Fatalf("ParseSource() error = %v", err)
	}

	if src.Type != SourceTypeLocal {
		t.Errorf("Type = %v, want %v", src.Type, SourceTypeLocal)
	}

	// Path should be absolute
	if !filepath.IsAbs(src.Path) {
		t.Errorf("Path should be absolute, got %v", src.Path)
	}
}
