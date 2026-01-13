//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/plugin"
)

// TestPluginWorkflowGo tests the complete plugin workflow for Go plugins
func TestPluginWorkflowGo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-go-plugin")

	// Create plugin scaffold
	cfg := plugin.ScaffoldConfig{
		Name:    "test-go-plugin",
		Type:    "provider",
		Lang:    "go",
		Author:  "Integration Test",
		Version: "1.0.0",
	}

	err := plugin.Scaffold(pluginDir, cfg)
	if err != nil {
		t.Fatalf("Failed to scaffold plugin: %v", err)
	}

	// Verify manifest exists
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Fatal("plugin.yaml not created")
	}

	// Verify entrypoint exists
	entrypointPath := filepath.Join(pluginDir, "main.go")
	if _, err := os.Stat(entrypointPath); os.IsNotExist(err) {
		t.Fatal("main.go not created")
	}

	// Load manifest
	manifest, err := plugin.LoadManifest(pluginDir)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	if manifest.Name != "test-go-plugin" {
		t.Errorf("Manifest name = %s, want test-go-plugin", manifest.Name)
	}

	if manifest.Type != "provider" {
		t.Errorf("Manifest type = %s, want provider", manifest.Type)
	}
}

// TestPluginWorkflowPython tests the complete plugin workflow for Python plugins
func TestPluginWorkflowPython(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-python-plugin")

	cfg := plugin.ScaffoldConfig{
		Name:    "test-python-plugin",
		Type:    "validator",
		Lang:    "python",
		Author:  "Integration Test",
		Version: "1.0.0",
	}

	err := plugin.Scaffold(pluginDir, cfg)
	if err != nil {
		t.Fatalf("Failed to scaffold plugin: %v", err)
	}

	// Verify Python files
	if _, err := os.Stat(filepath.Join(pluginDir, "main.py")); os.IsNotExist(err) {
		t.Fatal("main.py not created")
	}

	if _, err := os.Stat(filepath.Join(pluginDir, "requirements.txt")); os.IsNotExist(err) {
		t.Fatal("requirements.txt not created")
	}

	// Load and verify manifest
	manifest, err := plugin.LoadManifest(pluginDir)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	if manifest.Type != "validator" {
		t.Errorf("Manifest type = %s, want validator", manifest.Type)
	}
}

// TestPluginWorkflowNode tests the complete plugin workflow for Node.js plugins
func TestPluginWorkflowNode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-node-plugin")

	cfg := plugin.ScaffoldConfig{
		Name:    "test-node-plugin",
		Type:    "formatter",
		Lang:    "node",
		Author:  "Integration Test",
		Version: "1.0.0",
	}

	err := plugin.Scaffold(pluginDir, cfg)
	if err != nil {
		t.Fatalf("Failed to scaffold plugin: %v", err)
	}

	// Verify Node.js files
	if _, err := os.Stat(filepath.Join(pluginDir, "index.js")); os.IsNotExist(err) {
		t.Fatal("index.js not created")
	}

	if _, err := os.Stat(filepath.Join(pluginDir, "package.json")); os.IsNotExist(err) {
		t.Fatal("package.json not created")
	}

	// Load and verify manifest
	manifest, err := plugin.LoadManifest(pluginDir)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	if manifest.Type != "formatter" {
		t.Errorf("Manifest type = %s, want formatter", manifest.Type)
	}
}

// TestPluginWorkflowShell tests the complete plugin workflow for Shell plugins
func TestPluginWorkflowShell(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-shell-plugin")

	cfg := plugin.ScaffoldConfig{
		Name:    "test-shell-plugin",
		Type:    "hook",
		Lang:    "shell",
		Author:  "Integration Test",
		Version: "1.0.0",
	}

	err := plugin.Scaffold(pluginDir, cfg)
	if err != nil {
		t.Fatalf("Failed to scaffold plugin: %v", err)
	}

	// Verify shell file
	if _, err := os.Stat(filepath.Join(pluginDir, "entrypoint.sh")); os.IsNotExist(err) {
		t.Fatal("entrypoint.sh not created")
	}

	// Load and verify manifest
	manifest, err := plugin.LoadManifest(pluginDir)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	if manifest.Type != "hook" {
		t.Errorf("Manifest type = %s, want hook", manifest.Type)
	}
}

// TestPluginManagerOperations tests plugin manager CRUD operations
func TestPluginManagerOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	pluginBaseDir := filepath.Join(tmpDir, "plugins")
	os.MkdirAll(pluginBaseDir, 0755)

	// Create a test plugin
	pluginDir := filepath.Join(pluginBaseDir, "test-manager-plugin")
	cfg := plugin.ScaffoldConfig{
		Name:    "test-manager-plugin",
		Type:    "notifier",
		Lang:    "shell",
		Author:  "Integration Test",
		Version: "1.0.0",
	}

	err := plugin.Scaffold(pluginDir, cfg)
	if err != nil {
		t.Fatalf("Failed to scaffold plugin: %v", err)
	}

	// Create manager with custom config
	managerConfig := plugin.ManagerConfig{
		PluginDirs: []string{pluginBaseDir},
		ConfigDir:  filepath.Join(tmpDir, "config"),
	}
	os.MkdirAll(managerConfig.ConfigDir, 0755)

	manager := plugin.NewManager(managerConfig)

	// Test discover
	err = manager.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Test list
	plugins := manager.List()
	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(plugins))
	}

	// Test get
	p, ok := manager.Get("test-manager-plugin")
	if !ok {
		t.Fatal("Plugin not found")
	}

	if p.Manifest.Name != "test-manager-plugin" {
		t.Errorf("Plugin name = %s, want test-manager-plugin", p.Manifest.Name)
	}

	// Test enable/disable
	err = manager.Disable("test-manager-plugin")
	if err != nil {
		t.Errorf("Disable failed: %v", err)
	}

	err = manager.Enable("test-manager-plugin")
	if err != nil {
		t.Errorf("Enable failed: %v", err)
	}

	// Re-discover to verify state persists
	manager2 := plugin.NewManager(managerConfig)
	err = manager2.Discover()
	if err != nil {
		t.Fatalf("Second discover failed: %v", err)
	}

	p2, ok := manager2.Get("test-manager-plugin")
	if !ok {
		t.Fatal("Plugin not found after second discover")
	}

	if p2.State == plugin.PluginStateDisabled {
		t.Error("Plugin should be enabled after re-discovery")
	}
}

// TestPluginInstallLocal tests local plugin installation
func TestPluginInstallLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	installDir := filepath.Join(tmpDir, "plugins")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(installDir, 0755)

	// Create source plugin
	cfg := plugin.ScaffoldConfig{
		Name:    "local-install-test",
		Type:    "provider",
		Lang:    "shell",
		Author:  "Integration Test",
		Version: "2.0.0",
	}

	err := plugin.Scaffold(sourceDir, cfg)
	if err != nil {
		t.Fatalf("Failed to scaffold plugin: %v", err)
	}

	// Create manager
	managerConfig := plugin.ManagerConfig{
		PluginDirs: []string{installDir},
		ConfigDir:  filepath.Join(tmpDir, "config"),
	}
	os.MkdirAll(managerConfig.ConfigDir, 0755)

	manager := plugin.NewManager(managerConfig)

	// Install
	err = manager.Install(sourceDir)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify installation
	err = manager.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	plugins := manager.List()
	found := false
	for _, p := range plugins {
		if p.Manifest.Name == "local-install-test" {
			found = true
			if p.Manifest.Version != "2.0.0" {
				t.Errorf("Version = %s, want 2.0.0", p.Manifest.Version)
			}
			break
		}
	}

	if !found {
		t.Error("Installed plugin not found")
	}
}

// TestPluginLockfile tests lockfile operations
func TestPluginLockfile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "plugins.lock.json")

	// Create lockfile
	lock := plugin.NewPluginLock(lockPath)

	// Add plugin
	err := lock.Add(plugin.LockedPlugin{
		Name:          "test-lock-plugin",
		Version:       "1.0.0",
		InstalledFrom: "./source",
		Source:        "local",
		InstalledAt:   time.Now(),
		Enabled:       true,
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Save
	err = lock.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load in new instance
	lock2, err := plugin.LoadPluginLock(lockPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	p, ok := lock2.Get("test-lock-plugin")
	if !ok {
		t.Fatal("Plugin not found in loaded lockfile")
	}

	if p.Version != "1.0.0" {
		t.Errorf("Version = %s, want 1.0.0", p.Version)
	}

	// Remove
	err = lock2.Remove("test-lock-plugin")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if _, ok := lock2.Get("test-lock-plugin"); ok {
		t.Error("Plugin should be removed")
	}
}

// TestPluginVersionParsing tests version parsing and comparison
func TestPluginVersionParsing(t *testing.T) {
	testCases := []struct {
		version string
		valid   bool
	}{
		{"1.0.0", true},
		{"0.1.0", true},
		{"10.20.30", true},
		{"1.0.0-alpha", true},
		{"1.0.0-alpha.1", true},
		{"1.0.0+build", true},
		{"1.0.0-beta+build.123", true},
		{"invalid", false},
		{"1.0", false},
		{"1.0.0.0", false},
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			v, err := plugin.ParseVersion(tc.version)
			if tc.valid {
				if err != nil {
					t.Errorf("Expected valid version, got error: %v", err)
				}
				if v == nil {
					t.Error("Expected non-nil version")
				}
			} else {
				if err == nil {
					t.Error("Expected error for invalid version")
				}
			}
		})
	}
}

// TestPluginVersionConstraints tests version constraint satisfaction
func TestPluginVersionConstraints(t *testing.T) {
	testCases := []struct {
		version    string
		constraint string
		satisfies  bool
	}{
		{"1.0.0", ">=1.0.0", true},
		{"1.0.0", ">1.0.0", false},
		{"2.0.0", ">=1.0.0", true},
		{"0.9.0", ">=1.0.0", false},
		{"1.5.0", "^1.0.0", true},
		{"2.0.0", "^1.0.0", false},
		{"1.0.5", "~1.0.0", true},
		{"1.1.0", "~1.0.0", false},
		{"1.0.0", "*", true},
		{"99.99.99", "*", true},
	}

	for _, tc := range testCases {
		t.Run(tc.version+"_"+tc.constraint, func(t *testing.T) {
			v, err := plugin.ParseVersion(tc.version)
			if err != nil {
				t.Fatalf("Failed to parse version: %v", err)
			}

			c, err := plugin.ParseConstraint(tc.constraint)
			if err != nil {
				t.Fatalf("Failed to parse constraint: %v", err)
			}

			result := v.Satisfies(c)
			if result != tc.satisfies {
				t.Errorf("Satisfies = %v, want %v", result, tc.satisfies)
			}
		})
	}
}

// TestPluginSourceParsing tests plugin source URL parsing
func TestPluginSourceParsing(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		source     string
		sourceType plugin.SourceType
		wantErr    bool
	}{
		{tmpDir, plugin.SourceTypeLocal, false},
		{"github.com/user/repo", plugin.SourceTypeGitHub, false},
		{"github.com/user/repo@v1.0.0", plugin.SourceTypeGitHub, false},
		{"github.com/user/repo@main/plugins/test", plugin.SourceTypeGitHub, false},
		{"registry:my-plugin", plugin.SourceTypeRegistry, false},
		{"registry:my-plugin@1.0.0", plugin.SourceTypeRegistry, false},
		{"/nonexistent/path", plugin.SourceTypeLocal, true},
	}

	for _, tc := range testCases {
		t.Run(tc.source, func(t *testing.T) {
			src, err := plugin.ParseSource(tc.source)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if src.Type != tc.sourceType {
				t.Errorf("Type = %s, want %s", src.Type, tc.sourceType)
			}
		})
	}
}

// TestPluginRegistryIntegration tests registry client integration
func TestPluginRegistryIntegration(t *testing.T) {
	// Create mock registry server
	index := &plugin.RegistryIndex{
		Version: "1.0.0",
		Updated: time.Now(),
		Plugins: map[string]plugin.RegistryPlugin{
			"test-plugin": {
				Name:        "test-plugin",
				Description: "A test plugin",
				Author:      "Test Author",
				Type:        plugin.PluginTypeProvider,
				Repository:  "github.com/test/test-plugin",
				Latest:      "1.0.0",
				Keywords:    []string{"test", "integration"},
				Downloads:   100,
				Stars:       10,
				Versions: map[string]plugin.RegistryVersion{
					"1.0.0": {
						Released: time.Now(),
						Checksum: "abc123",
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	}))
	defer server.Close()

	// Create registry client
	registry := plugin.NewRegistry(
		plugin.WithRegistryURL(server.URL),
		plugin.WithCacheDir(t.TempDir()),
	)

	// Test fetch
	err := registry.Fetch()
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Test get
	p, err := registry.Get("test-plugin")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if p.Name != "test-plugin" {
		t.Errorf("Name = %s, want test-plugin", p.Name)
	}

	// Test search
	results, err := registry.Search("test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Test version resolution
	version, err := registry.ResolveVersion("test-plugin", ">=1.0.0")
	if err != nil {
		t.Fatalf("ResolveVersion failed: %v", err)
	}

	if version != "1.0.0" {
		t.Errorf("Version = %s, want 1.0.0", version)
	}
}

// TestPluginDependencyResolution tests dependency resolver
func TestPluginDependencyResolution(t *testing.T) {
	// Create mock registry
	index := &plugin.RegistryIndex{
		Version: "1.0.0",
		Updated: time.Now(),
		Plugins: map[string]plugin.RegistryPlugin{
			"dep-a": {
				Name:       "dep-a",
				Type:       plugin.PluginTypeProvider,
				Repository: "github.com/test/dep-a",
				Latest:     "1.0.0",
				Versions: map[string]plugin.RegistryVersion{
					"1.0.0": {Released: time.Now()},
				},
			},
			"dep-b": {
				Name:       "dep-b",
				Type:       plugin.PluginTypeValidator,
				Repository: "github.com/test/dep-b",
				Latest:     "1.0.0",
				Versions: map[string]plugin.RegistryVersion{
					"1.0.0": {
						Released: time.Now(),
						Dependencies: []plugin.PluginDependency{
							{Name: "dep-a", Version: ">=1.0.0"},
						},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	}))
	defer server.Close()

	registry := plugin.NewRegistry(
		plugin.WithRegistryURL(server.URL),
		plugin.WithCacheDir(t.TempDir()),
	)

	resolver := plugin.NewDependencyResolver(nil, registry)

	manifest := &plugin.Manifest{
		Name:    "test-app",
		Version: "1.0.0",
		Dependencies: []plugin.PluginDependency{
			{Name: "dep-b", Version: "1.0.0"},
		},
	}

	result, err := resolver.Resolve(manifest)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if !result.IsSuccess() {
		t.Errorf("Expected successful resolution, got: conflicts=%d, circular=%d, missing=%d",
			len(result.Conflicts), len(result.Circular), len(result.Missing))
	}
}

// TestPluginHealthCheck tests health check functionality
func TestPluginHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test requires a running plugin, so we'll just test the infrastructure
	tmpDir := t.TempDir()
	pluginBaseDir := filepath.Join(tmpDir, "plugins")
	os.MkdirAll(pluginBaseDir, 0755)

	managerConfig := plugin.ManagerConfig{
		PluginDirs: []string{pluginBaseDir},
		ConfigDir:  filepath.Join(tmpDir, "config"),
	}
	os.MkdirAll(managerConfig.ConfigDir, 0755)

	manager := plugin.NewManager(managerConfig)

	// Test health check on non-existent plugin
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := manager.Health(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent plugin")
	}
}

// TestPluginMultipleLanguages tests creating plugins in all supported languages
func TestPluginMultipleLanguages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	languages := []string{"go", "python", "node", "shell"}
	pluginTypes := []string{"provider", "validator", "formatter", "hook", "notifier"}

	for _, lang := range languages {
		for _, pType := range pluginTypes {
			t.Run(lang+"_"+pType, func(t *testing.T) {
				tmpDir := t.TempDir()
				pluginDir := filepath.Join(tmpDir, "test-plugin")

				cfg := plugin.ScaffoldConfig{
					Name:    "test-plugin",
					Type:    pType,
					Lang:    lang,
					Author:  "Test",
					Version: "1.0.0",
				}

				err := plugin.Scaffold(pluginDir, cfg)
				if err != nil {
					t.Fatalf("Scaffold failed for %s/%s: %v", lang, pType, err)
				}

				// Verify manifest
				manifest, err := plugin.LoadManifest(pluginDir)
				if err != nil {
					t.Fatalf("LoadManifest failed: %v", err)
				}

				if string(manifest.Type) != pType {
					t.Errorf("Type = %s, want %s", manifest.Type, pType)
				}
			})
		}
	}
}
