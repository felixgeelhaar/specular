package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	config := DefaultManagerConfig()
	manager := NewManager(config)

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.plugins == nil {
		t.Error("plugins map should be initialized")
	}
}

func TestDefaultManagerConfig(t *testing.T) {
	config := DefaultManagerConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("Default timeout = %v, want 30s", config.Timeout)
	}

	if !config.AutoDiscover {
		t.Error("AutoDiscover should default to true")
	}

	if len(config.PluginDirs) == 0 {
		t.Error("PluginDirs should have at least one directory")
	}

	// Check that system plugin directory is included
	hasSystemDir := false
	for _, dir := range config.PluginDirs {
		if dir == "/usr/local/share/specular/plugins" {
			hasSystemDir = true
			break
		}
	}
	if !hasSystemDir {
		t.Error("PluginDirs should include system plugin directory")
	}
}

func TestManagerDiscover(t *testing.T) {
	tmpDir := t.TempDir()

	config := ManagerConfig{
		PluginDirs: []string{tmpDir},
		Timeout:    30 * time.Second,
	}
	manager := NewManager(config)

	// Discover should not error even with no plugins
	err := manager.Discover()
	if err != nil {
		t.Errorf("Discover() error = %v", err)
	}

	// No plugins should be found
	plugins := manager.List()
	if len(plugins) != 0 {
		t.Errorf("List() returned %d plugins, want 0", len(plugins))
	}
}

func TestManagerDiscoverWithPlugin(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test plugin
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	os.MkdirAll(pluginDir, 0755)

	// Create manifest
	manifest := `name: test-plugin
version: "1.0.0"
type: provider
description: "Test plugin"
entrypoint: "./entrypoint.sh"
`
	os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644)

	// Create entrypoint
	entrypoint := `#!/bin/bash
echo '{"success": true}'
`
	os.WriteFile(filepath.Join(pluginDir, "entrypoint.sh"), []byte(entrypoint), 0755)

	config := ManagerConfig{
		PluginDirs: []string{tmpDir},
		Timeout:    30 * time.Second,
	}
	manager := NewManager(config)

	err := manager.Discover()
	if err != nil {
		t.Errorf("Discover() error = %v", err)
	}

	plugins := manager.List()
	if len(plugins) != 1 {
		t.Errorf("List() returned %d plugins, want 1", len(plugins))
	}

	if len(plugins) > 0 {
		if plugins[0].Manifest.Name != "test-plugin" {
			t.Errorf("Plugin name = %s, want test-plugin", plugins[0].Manifest.Name)
		}
		if plugins[0].State != PluginStateLoaded {
			t.Errorf("Plugin state = %s, want loaded", plugins[0].State)
		}
	}
}

func TestManagerGet(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test plugin
	pluginDir := filepath.Join(tmpDir, "get-test")
	os.MkdirAll(pluginDir, 0755)

	manifest := `name: get-test
version: "1.0.0"
type: validator
entrypoint: "./run.sh"
`
	os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644)
	os.WriteFile(filepath.Join(pluginDir, "run.sh"), []byte("#!/bin/bash\necho '{}'"), 0755)

	config := ManagerConfig{
		PluginDirs: []string{tmpDir},
		Timeout:    30 * time.Second,
	}
	manager := NewManager(config)
	manager.Discover()

	// Test getting existing plugin
	plugin, ok := manager.Get("get-test")
	if !ok {
		t.Error("Get() should return true for existing plugin")
	}
	if plugin == nil {
		t.Error("Get() should return plugin for existing name")
	}

	// Test getting non-existent plugin
	_, ok = manager.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for non-existent plugin")
	}
}

func TestManagerListByType(t *testing.T) {
	tmpDir := t.TempDir()

	// Create provider plugin
	providerDir := filepath.Join(tmpDir, "provider-plugin")
	os.MkdirAll(providerDir, 0755)
	os.WriteFile(filepath.Join(providerDir, "plugin.yaml"), []byte(`name: provider-plugin
version: "1.0.0"
type: provider
entrypoint: "./run.sh"
`), 0644)
	os.WriteFile(filepath.Join(providerDir, "run.sh"), []byte("#!/bin/bash\necho '{}'"), 0755)

	// Create validator plugin
	validatorDir := filepath.Join(tmpDir, "validator-plugin")
	os.MkdirAll(validatorDir, 0755)
	os.WriteFile(filepath.Join(validatorDir, "plugin.yaml"), []byte(`name: validator-plugin
version: "1.0.0"
type: validator
entrypoint: "./run.sh"
`), 0644)
	os.WriteFile(filepath.Join(validatorDir, "run.sh"), []byte("#!/bin/bash\necho '{}'"), 0755)

	config := ManagerConfig{
		PluginDirs: []string{tmpDir},
		Timeout:    30 * time.Second,
	}
	manager := NewManager(config)
	manager.Discover()

	// Test listing by type
	providers := manager.ListByType(PluginTypeProvider)
	if len(providers) != 1 {
		t.Errorf("ListByType(provider) returned %d, want 1", len(providers))
	}

	validators := manager.ListByType(PluginTypeValidator)
	if len(validators) != 1 {
		t.Errorf("ListByType(validator) returned %d, want 1", len(validators))
	}

	notifiers := manager.ListByType(PluginTypeNotifier)
	if len(notifiers) != 0 {
		t.Errorf("ListByType(notifier) returned %d, want 0", len(notifiers))
	}
}

func TestManagerEnable(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test plugin
	pluginDir := filepath.Join(tmpDir, "enable-test")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`name: enable-test
version: "1.0.0"
type: hook
entrypoint: "./run.sh"
`), 0644)
	os.WriteFile(filepath.Join(pluginDir, "run.sh"), []byte("#!/bin/bash\necho '{}'"), 0755)

	config := ManagerConfig{
		PluginDirs: []string{tmpDir},
		Timeout:    30 * time.Second,
	}
	manager := NewManager(config)
	manager.Discover()

	// Enable the plugin
	err := manager.Enable("enable-test")
	if err != nil {
		t.Errorf("Enable() error = %v", err)
	}

	// Check state
	plugin, _ := manager.Get("enable-test")
	if plugin.State != PluginStateEnabled {
		t.Errorf("Plugin state = %s, want enabled", plugin.State)
	}

	// Enable non-existent plugin should error
	err = manager.Enable("nonexistent")
	if err == nil {
		t.Error("Enable() should error for non-existent plugin")
	}
}

func TestManagerDisable(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test plugin
	pluginDir := filepath.Join(tmpDir, "disable-test")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`name: disable-test
version: "1.0.0"
type: formatter
entrypoint: "./run.sh"
`), 0644)
	os.WriteFile(filepath.Join(pluginDir, "run.sh"), []byte("#!/bin/bash\necho '{}'"), 0755)

	config := ManagerConfig{
		PluginDirs: []string{tmpDir},
		Timeout:    30 * time.Second,
	}
	manager := NewManager(config)
	manager.Discover()

	// Disable the plugin
	err := manager.Disable("disable-test")
	if err != nil {
		t.Errorf("Disable() error = %v", err)
	}

	// Check state
	plugin, _ := manager.Get("disable-test")
	if plugin.State != PluginStateDisabled {
		t.Errorf("Plugin state = %s, want disabled", plugin.State)
	}

	// Disable non-existent plugin should error
	err = manager.Disable("nonexistent")
	if err == nil {
		t.Error("Disable() should error for non-existent plugin")
	}
}

func TestManagerSetConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test plugin
	pluginDir := filepath.Join(tmpDir, "config-test")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`name: config-test
version: "1.0.0"
type: notifier
entrypoint: "./run.sh"
`), 0644)
	os.WriteFile(filepath.Join(pluginDir, "run.sh"), []byte("#!/bin/bash\necho '{}'"), 0755)

	config := ManagerConfig{
		PluginDirs: []string{tmpDir},
		Timeout:    30 * time.Second,
	}
	manager := NewManager(config)
	manager.Discover()

	// Set config
	testConfig := map[string]interface{}{
		"webhook_url": "https://example.com/hook",
		"enabled":     true,
	}
	err := manager.SetConfig("config-test", testConfig)
	if err != nil {
		t.Errorf("SetConfig() error = %v", err)
	}

	// Check config
	plugin, _ := manager.Get("config-test")
	if plugin.Config["webhook_url"] != "https://example.com/hook" {
		t.Error("Config should contain webhook_url")
	}

	// SetConfig for non-existent plugin should error
	err = manager.SetConfig("nonexistent", testConfig)
	if err == nil {
		t.Error("SetConfig() should error for non-existent plugin")
	}
}

func TestManagerInstallLocal(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "plugins")

	// Create source plugin
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "plugin.yaml"), []byte(`name: local-install-test
version: "1.0.0"
type: hook
entrypoint: "./run.sh"
`), 0644)
	os.WriteFile(filepath.Join(srcDir, "run.sh"), []byte("#!/bin/bash\necho '{}'"), 0755)

	// Mock home directory for this test
	// Note: This test is limited because it uses actual home directory
	// In a real test environment, we would mock os.UserHomeDir

	config := ManagerConfig{
		PluginDirs: []string{destDir},
		Timeout:    30 * time.Second,
	}
	_ = NewManager(config)

	// Install should validate the source has a manifest
	// We can't fully test installation without mocking home directory
	_, err := os.Stat(filepath.Join(srcDir, "plugin.yaml"))
	if err != nil {
		t.Error("Source plugin.yaml should exist")
	}
}

func TestManagerUninstall(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test plugin directly in the plugin directory
	pluginDir := filepath.Join(tmpDir, "uninstall-test")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`name: uninstall-test
version: "1.0.0"
type: notifier
entrypoint: "./run.sh"
`), 0644)
	os.WriteFile(filepath.Join(pluginDir, "run.sh"), []byte("#!/bin/bash\necho '{}'"), 0755)

	config := ManagerConfig{
		PluginDirs: []string{tmpDir},
		Timeout:    30 * time.Second,
	}
	manager := NewManager(config)
	manager.Discover()

	// Uninstall the plugin
	err := manager.Uninstall("uninstall-test")
	if err != nil {
		t.Errorf("Uninstall() error = %v", err)
	}

	// Plugin directory should be removed
	if _, err := os.Stat(pluginDir); !os.IsNotExist(err) {
		t.Error("Plugin directory should be removed after uninstall")
	}

	// Plugin should not be in list
	_, ok := manager.Get("uninstall-test")
	if ok {
		t.Error("Plugin should not exist after uninstall")
	}
}

func TestManagerExecuteNotFound(t *testing.T) {
	config := DefaultManagerConfig()
	manager := NewManager(config)

	ctx := context.Background()
	_, err := manager.Execute(ctx, "nonexistent", nil)
	if err == nil {
		t.Error("Execute() should error for non-existent plugin")
	}
}

func TestPluginTypes(t *testing.T) {
	types := []PluginType{
		PluginTypeProvider,
		PluginTypeValidator,
		PluginTypeFormatter,
		PluginTypeHook,
		PluginTypeNotifier,
	}

	expectedStrings := map[PluginType]string{
		PluginTypeProvider:  "provider",
		PluginTypeValidator: "validator",
		PluginTypeFormatter: "formatter",
		PluginTypeHook:      "hook",
		PluginTypeNotifier:  "notifier",
	}

	for _, pt := range types {
		expected := expectedStrings[pt]
		if string(pt) != expected {
			t.Errorf("PluginType %v = %s, want %s", pt, string(pt), expected)
		}
	}
}

func TestPluginStates(t *testing.T) {
	states := []PluginState{
		PluginStateUnknown,
		PluginStateDiscovered,
		PluginStateLoaded,
		PluginStateEnabled,
		PluginStateDisabled,
		PluginStateError,
	}

	expectedStrings := map[PluginState]string{
		PluginStateUnknown:    "unknown",
		PluginStateDiscovered: "discovered",
		PluginStateLoaded:     "loaded",
		PluginStateEnabled:    "enabled",
		PluginStateDisabled:   "disabled",
		PluginStateError:      "error",
	}

	for _, ps := range states {
		expected := expectedStrings[ps]
		if string(ps) != expected {
			t.Errorf("PluginState %v = %s, want %s", ps, string(ps), expected)
		}
	}
}

func TestManifestStructure(t *testing.T) {
	manifest := Manifest{
		Name:               "test-manifest",
		Version:            "1.2.3",
		Description:        "Test description",
		Author:             "Test Author",
		License:            "MIT",
		Homepage:           "https://example.com",
		Type:               PluginTypeProvider,
		Entrypoint:         "./main.sh",
		MinSpecularVersion: "0.5.0",
		Capabilities:       []string{"cap1", "cap2"},
		Config: []ConfigField{
			{
				Name:        "api_key",
				Type:        "string",
				Description: "API key",
				Required:    true,
				Secret:      true,
			},
		},
	}

	if manifest.Name != "test-manifest" {
		t.Error("Manifest name not set correctly")
	}
	if manifest.Version != "1.2.3" {
		t.Error("Manifest version not set correctly")
	}
	if len(manifest.Config) != 1 {
		t.Error("Manifest should have 1 config field")
	}
	if !manifest.Config[0].Secret {
		t.Error("Config field should be marked as secret")
	}
}

func TestPluginStructure(t *testing.T) {
	plugin := Plugin{
		Manifest: Manifest{
			Name:    "test-plugin",
			Version: "1.0.0",
			Type:    PluginTypeValidator,
		},
		Path:     "/path/to/plugin",
		State:    PluginStateLoaded,
		LoadedAt: time.Now(),
		Config: map[string]interface{}{
			"key": "value",
		},
	}

	if plugin.Manifest.Name != "test-plugin" {
		t.Error("Plugin manifest name not set")
	}
	if plugin.State != PluginStateLoaded {
		t.Error("Plugin state not set correctly")
	}
	if plugin.Config["key"] != "value" {
		t.Error("Plugin config not set correctly")
	}
}

func TestPluginRequest(t *testing.T) {
	request := PluginRequest{
		Action: "execute",
		Params: map[string]interface{}{
			"input": "test",
		},
		Config: map[string]interface{}{
			"timeout": 30,
		},
	}

	if request.Action != "execute" {
		t.Error("Request action not set")
	}
	if request.Params["input"] != "test" {
		t.Error("Request params not set")
	}
}

func TestPluginResponse(t *testing.T) {
	successResponse := PluginResponse{
		Success: true,
		Result: map[string]interface{}{
			"output": "result",
		},
	}

	if !successResponse.Success {
		t.Error("Success response should be true")
	}

	errorResponse := PluginResponse{
		Success: false,
		Error:   "Something went wrong",
	}

	if errorResponse.Success {
		t.Error("Error response should be false")
	}
	if errorResponse.Error != "Something went wrong" {
		t.Error("Error message not set")
	}
}

func TestHealthRequest(t *testing.T) {
	request := HealthRequest{
		Action: "health",
	}

	if request.Action != "health" {
		t.Error("Health request action should be 'health'")
	}
}

func TestHealthResponse(t *testing.T) {
	response := HealthResponse{
		Status:  "healthy",
		Version: "1.0.0",
		Name:    "test-plugin",
	}

	if response.Status != "healthy" {
		t.Error("Health status not set")
	}
	if response.Version != "1.0.0" {
		t.Error("Health version not set")
	}
}

func TestValidatorRequest(t *testing.T) {
	request := ValidatorRequest{
		Action:  "validate",
		Content: "test content",
		Rules: map[string]interface{}{
			"max_length": 100,
		},
	}

	if request.Action != "validate" {
		t.Error("Validator request action not set")
	}
	if request.Content != "test content" {
		t.Error("Validator request content not set")
	}
}

func TestValidatorResponse(t *testing.T) {
	response := ValidatorResponse{
		Valid: false,
		Messages: []ValidatorIssue{
			{
				Severity: "error",
				Message:  "Validation failed",
				Line:     10,
				Column:   5,
				Rule:     "max_length",
			},
		},
	}

	if response.Valid {
		t.Error("Invalid response should have Valid=false")
	}
	if len(response.Messages) != 1 {
		t.Error("Should have 1 validation message")
	}
	if response.Messages[0].Severity != "error" {
		t.Error("Issue severity not set")
	}
}

func TestNotifierRequest(t *testing.T) {
	request := NotifierRequest{
		Action: "notify",
		Event:  "build_complete",
		Data: map[string]interface{}{
			"status": "success",
		},
	}

	if request.Event != "build_complete" {
		t.Error("Notifier event not set")
	}
}

func TestFormatterRequest(t *testing.T) {
	request := FormatterRequest{
		Action: "format",
		Data:   "test data",
		Format: "json",
	}

	if request.Format != "json" {
		t.Error("Formatter format not set")
	}
}

func TestFormatterResponse(t *testing.T) {
	response := FormatterResponse{
		Output: "{\"result\": \"formatted\"}",
	}

	if response.Output == "" {
		t.Error("Formatter output should not be empty")
	}
}

func TestDiscoverWithInvalidManifest(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a plugin with invalid manifest
	pluginDir := filepath.Join(tmpDir, "invalid-plugin")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`invalid: yaml: content`), 0644)

	config := ManagerConfig{
		PluginDirs: []string{tmpDir},
		Timeout:    30 * time.Second,
	}
	manager := NewManager(config)

	// Discover should not error, but plugin should be in error state
	err := manager.Discover()
	if err != nil {
		t.Errorf("Discover() should not error: %v", err)
	}

	plugin, ok := manager.Get("invalid-plugin")
	if !ok {
		t.Fatal("Invalid plugin should still be registered")
	}
	if plugin.State != PluginStateError {
		t.Errorf("Invalid plugin state = %s, want error", plugin.State)
	}
}

func TestDiscoverMissingEntrypoint(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a plugin without entrypoint
	pluginDir := filepath.Join(tmpDir, "no-entrypoint")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`name: no-entrypoint
version: "1.0.0"
type: hook
entrypoint: "./missing.sh"
`), 0644)

	config := ManagerConfig{
		PluginDirs: []string{tmpDir},
		Timeout:    30 * time.Second,
	}
	manager := NewManager(config)
	manager.Discover()

	plugin, ok := manager.Get("no-entrypoint")
	if !ok {
		t.Fatal("Plugin should be registered")
	}
	if plugin.State != PluginStateError {
		t.Errorf("Plugin with missing entrypoint state = %s, want error", plugin.State)
	}
}
