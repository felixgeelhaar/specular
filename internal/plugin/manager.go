package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Manager handles plugin discovery, loading, and execution
type Manager struct {
	mu         sync.RWMutex
	plugins    map[string]*Plugin
	pluginDirs []string
	config     ManagerConfig
}

// ManagerConfig contains configuration for the plugin manager
type ManagerConfig struct {
	// AutoDiscover enables automatic plugin discovery
	AutoDiscover bool
	// Timeout is the default execution timeout
	Timeout time.Duration
	// PluginDirs are directories to search for plugins
	PluginDirs []string
}

// DefaultManagerConfig returns default configuration
func DefaultManagerConfig() ManagerConfig {
	homeDir, _ := os.UserHomeDir()
	return ManagerConfig{
		AutoDiscover: true,
		Timeout:      30 * time.Second,
		PluginDirs: []string{
			filepath.Join(homeDir, ".specular", "plugins"),
			"/usr/local/share/specular/plugins",
		},
	}
}

// NewManager creates a new plugin manager
func NewManager(config ManagerConfig) *Manager {
	return &Manager{
		plugins:    make(map[string]*Plugin),
		pluginDirs: config.PluginDirs,
		config:     config,
	}
}

// Discover searches for plugins in configured directories
func (m *Manager) Discover() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, dir := range m.pluginDirs {
		if err := m.discoverInDir(dir); err != nil {
			// Log but continue searching other directories
			continue
		}
	}

	return nil
}

// discoverInDir searches a single directory for plugins
func (m *Manager) discoverInDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, skip
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(dir, entry.Name())
		manifestPath := filepath.Join(pluginPath, "plugin.yaml")

		// Also check for plugin.json
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			manifestPath = filepath.Join(pluginPath, "plugin.json")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				continue // No manifest found
			}
		}

		plugin, err := m.loadPlugin(pluginPath, manifestPath)
		if err != nil {
			// Create error entry for failed plugins
			m.plugins[entry.Name()] = &Plugin{
				Manifest: Manifest{Name: entry.Name()},
				Path:     pluginPath,
				State:    PluginStateError,
				Error:    err.Error(),
				LoadedAt: time.Now(),
			}
			continue
		}

		m.plugins[plugin.Manifest.Name] = plugin
	}

	return nil
}

// loadPlugin loads a plugin from its manifest
func (m *Manager) loadPlugin(pluginPath, manifestPath string) (*Plugin, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if strings.HasSuffix(manifestPath, ".yaml") || strings.HasSuffix(manifestPath, ".yml") {
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse yaml manifest: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse json manifest: %w", err)
		}
	}

	// Validate required fields
	if manifest.Name == "" {
		return nil, fmt.Errorf("manifest missing required field: name")
	}
	if manifest.Version == "" {
		return nil, fmt.Errorf("manifest missing required field: version")
	}
	if manifest.Type == "" {
		return nil, fmt.Errorf("manifest missing required field: type")
	}
	if manifest.Entrypoint == "" {
		return nil, fmt.Errorf("manifest missing required field: entrypoint")
	}

	// Resolve entrypoint path
	entrypointPath := manifest.Entrypoint
	if !filepath.IsAbs(entrypointPath) {
		entrypointPath = filepath.Join(pluginPath, entrypointPath)
	}

	// Verify entrypoint exists
	if _, err := os.Stat(entrypointPath); err != nil {
		return nil, fmt.Errorf("entrypoint not found: %s", entrypointPath)
	}

	return &Plugin{
		Manifest: manifest,
		Path:     pluginPath,
		State:    PluginStateLoaded,
		LoadedAt: time.Now(),
		Config:   make(map[string]interface{}),
	}, nil
}

// List returns all discovered plugins
func (m *Manager) List() []*Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// Get returns a specific plugin by name
func (m *Manager) Get(name string) (*Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.plugins[name]
	return p, ok
}

// ListByType returns plugins of a specific type
func (m *Manager) ListByType(pluginType PluginType) []*Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var plugins []*Plugin
	for _, p := range m.plugins {
		if p.Manifest.Type == pluginType && p.State == PluginStateLoaded {
			plugins = append(plugins, p)
		}
	}
	return plugins
}

// Execute runs a plugin with the given request
func (m *Manager) Execute(ctx context.Context, name string, request interface{}) (*PluginResponse, error) {
	plugin, ok := m.Get(name)
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	if plugin.State == PluginStateError {
		return nil, fmt.Errorf("plugin in error state: %s", plugin.Error)
	}

	return m.executePlugin(ctx, plugin, request)
}

// executePlugin runs a plugin executable
func (m *Manager) executePlugin(ctx context.Context, plugin *Plugin, request interface{}) (*PluginResponse, error) {
	// Get entrypoint path
	entrypointPath := plugin.Manifest.Entrypoint
	if !filepath.IsAbs(entrypointPath) {
		entrypointPath = filepath.Join(plugin.Path, entrypointPath)
	}

	// Serialize request
	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("serialize request: %w", err)
	}

	// Create timeout context
	execCtx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	// Execute plugin
	cmd := exec.CommandContext(execCtx, entrypointPath)
	cmd.Stdin = bytes.NewReader(requestData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("plugin execution timed out after %v", m.config.Timeout)
		}
		return nil, fmt.Errorf("plugin execution failed: %w (stderr: %s)", err, stderr.String())
	}

	// Parse response
	var response PluginResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("parse plugin response: %w (output: %s)", err, stdout.String())
	}

	return &response, nil
}

// Health checks if a plugin is healthy
func (m *Manager) Health(ctx context.Context, name string) (*HealthResponse, error) {
	plugin, ok := m.Get(name)
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	request := HealthRequest{Action: "health"}
	resp, err := m.executePlugin(ctx, plugin, request)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("health check failed: %s", resp.Error)
	}

	// Extract health response from result
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal health result: %w", err)
	}

	var health HealthResponse
	if err := json.Unmarshal(resultData, &health); err != nil {
		return nil, fmt.Errorf("parse health response: %w", err)
	}

	return &health, nil
}

// Enable marks a plugin as enabled
func (m *Manager) Enable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	if plugin.State == PluginStateError {
		return fmt.Errorf("cannot enable plugin in error state: %s", plugin.Error)
	}

	plugin.State = PluginStateEnabled
	return nil
}

// Disable marks a plugin as disabled
func (m *Manager) Disable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	plugin.State = PluginStateDisabled
	return nil
}

// SetConfig sets configuration for a plugin
func (m *Manager) SetConfig(name string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	plugin.Config = config
	return nil
}

// Install installs a plugin from a path or URL
func (m *Manager) Install(source string) error {
	// Determine source type
	if strings.HasPrefix(source, "github.com/") || strings.HasPrefix(source, "https://github.com/") {
		return m.installFromGitHub(source)
	}

	// Assume local directory
	return m.installFromLocal(source)
}

// installFromLocal installs a plugin from a local directory
func (m *Manager) installFromLocal(sourcePath string) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Verify source exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("source path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source must be a directory")
	}

	// Find manifest file
	manifestPath := filepath.Join(absPath, "plugin.yaml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifestPath = filepath.Join(absPath, "plugin.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			return fmt.Errorf("no plugin.yaml or plugin.json found in %s", absPath)
		}
	}

	// Load and validate manifest
	plugin, err := m.loadPlugin(absPath, manifestPath)
	if err != nil {
		return fmt.Errorf("invalid plugin: %w", err)
	}

	// Get user plugin directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}
	pluginDir := filepath.Join(homeDir, ".specular", "plugins")

	// Create plugin directory if needed
	if err := os.MkdirAll(pluginDir, 0750); err != nil {
		return fmt.Errorf("create plugin directory: %w", err)
	}

	// Destination path
	destPath := filepath.Join(pluginDir, plugin.Manifest.Name)

	// Check if plugin already exists
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("plugin %s already installed (use 'specular plugin uninstall %s' first)",
			plugin.Manifest.Name, plugin.Manifest.Name)
	}

	// Copy plugin directory
	if err := copyDir(absPath, destPath); err != nil {
		return fmt.Errorf("copy plugin: %w", err)
	}

	fmt.Printf("✓ Installed plugin: %s v%s\n", plugin.Manifest.Name, plugin.Manifest.Version)
	fmt.Printf("  Path: %s\n", destPath)

	return nil
}

// installFromGitHub installs a plugin from a GitHub repository
func (m *Manager) installFromGitHub(source string) error {
	// Parse GitHub URL
	repo := strings.TrimPrefix(source, "https://github.com/")
	repo = strings.TrimPrefix(repo, "github.com/")
	repo = strings.TrimSuffix(repo, ".git")

	if !strings.Contains(repo, "/") {
		return fmt.Errorf("invalid GitHub repository format (expected: github.com/user/repo)")
	}

	// Create temporary directory for cloning
	tmpDir, err := os.MkdirTemp("", "specular-plugin-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Clone repository
	fmt.Printf("Cloning %s...\n", repo)
	cloneURL := fmt.Sprintf("https://github.com/%s.git", repo)

	cmd := exec.Command("git", "clone", "--depth", "1", cloneURL, tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, output)
	}

	// Install from cloned directory
	return m.installFromLocal(tmpDir)
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return os.MkdirAll(destPath, 0750)
		}

		// Copy file
		return copyFile(path, destPath, info.Mode())
	})
}

// copyFile copies a single file
func copyFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, mode)
}

// Uninstall removes a plugin
func (m *Manager) Uninstall(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	// Remove plugin directory
	if err := os.RemoveAll(plugin.Path); err != nil {
		return fmt.Errorf("remove plugin directory: %w", err)
	}

	delete(m.plugins, name)
	return nil
}
