package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/specular/internal/safeutil"
)

var githubRepoPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

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
	// LockfilePath is the path to the plugin lockfile
	LockfilePath string
}

// InstallOptions configures plugin installation behavior
type InstallOptions struct {
	// Force overwrites existing plugins without prompting
	Force bool
	// Upgrade updates existing plugins to newer versions
	Upgrade bool
	// Version specifies a specific version to install
	Version string
	// SkipDependencies skips installing plugin dependencies
	SkipDependencies bool
}

// UpdateResult contains the result of an update operation
type UpdateResult struct {
	// Name is the plugin name
	Name string
	// OldVersion is the version before update
	OldVersion string
	// NewVersion is the version after update
	NewVersion string
	// Updated indicates if the plugin was actually updated
	Updated bool
	// Error contains any error that occurred
	Error error
}

// DefaultManagerConfig returns default configuration
func DefaultManagerConfig() ManagerConfig {
	config := ManagerConfig{
		AutoDiscover: true,
		Timeout:      30 * time.Second,
		PluginDirs:   []string{"/usr/local/share/specular/plugins"},
	}

	// Add user-specific plugin directory if home directory is available
	if homeDir, err := os.UserHomeDir(); err == nil {
		config.PluginDirs = append([]string{filepath.Join(homeDir, ".specular", "plugins")}, config.PluginDirs...)
	}

	return config
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
	if strings.TrimSpace(plugin.Path) == "" {
		return nil, fmt.Errorf("plugin path is empty")
	}

	// Get entrypoint path
	entrypointPath := plugin.Manifest.Entrypoint
	if strings.TrimSpace(entrypointPath) == "" {
		return nil, fmt.Errorf("plugin entrypoint is empty")
	}
	if strings.ContainsRune(entrypointPath, '\x00') {
		return nil, fmt.Errorf("plugin entrypoint contains null byte")
	}

	if !filepath.IsAbs(entrypointPath) {
		resolvedPath, err := safeutil.JoinInsideBase(plugin.Path, entrypointPath)
		if err != nil {
			return nil, fmt.Errorf("resolve plugin entrypoint: %w", err)
		}
		entrypointPath = resolvedPath
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
	cmd, err := safeutil.SafeCommand(execCtx, entrypointPath)
	if err != nil {
		return nil, fmt.Errorf("prepare plugin command: %w", err)
	}
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
	repo = strings.TrimSpace(repo)

	if !githubRepoPattern.MatchString(repo) {
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

	cmd, err := safeutil.SafeCommand(context.Background(), "git", "clone", "--depth", "1", cloneURL, tmpDir)
	if err != nil {
		return fmt.Errorf("prepare git clone: %w", err)
	}
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

	// Update lockfile
	lock, err := LoadPluginLock(m.config.LockfilePath)
	if err == nil {
		lock.Remove(name)
		_ = lock.Save() // Best effort
	}

	delete(m.plugins, name)
	return nil
}

// InstallWithOptions installs a plugin with additional options
func (m *Manager) InstallWithOptions(source string, opts InstallOptions) error {
	// Parse source
	src, err := ParseSource(source)
	if err != nil {
		return fmt.Errorf("parse source: %w", err)
	}

	// Apply version override if specified
	if opts.Version != "" && !src.IsLocal() {
		src = src.WithVersion(opts.Version)
	}

	// Validate version
	if err := src.ValidateVersion(); err != nil {
		return err
	}

	switch src.Type {
	case SourceTypeLocal:
		return m.installFromLocalWithOptions(src.Path, opts)
	case SourceTypeGitHub:
		return m.installFromGitHubWithOptions(src, opts)
	case SourceTypeRegistry:
		return fmt.Errorf("registry installation not yet implemented")
	default:
		return fmt.Errorf("unsupported source type: %s", src.Type)
	}
}

// installFromLocalWithOptions installs from local path with options
func (m *Manager) installFromLocalWithOptions(sourcePath string, opts InstallOptions) error {
	// Verify source exists and is a directory
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("source path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source must be a directory")
	}

	// Find manifest file
	manifestPath := filepath.Join(sourcePath, "plugin.yaml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifestPath = filepath.Join(sourcePath, "plugin.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			return fmt.Errorf("no plugin.yaml or plugin.json found in %s", sourcePath)
		}
	}

	// Load and validate manifest
	plugin, err := m.loadPlugin(sourcePath, manifestPath)
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
		if !opts.Force && !opts.Upgrade {
			return fmt.Errorf("plugin %s already installed (use --force to overwrite or --upgrade to update)",
				plugin.Manifest.Name)
		}
		// Remove existing
		if err := os.RemoveAll(destPath); err != nil {
			return fmt.Errorf("remove existing plugin: %w", err)
		}
	}

	// Copy plugin directory
	if err := copyDir(sourcePath, destPath); err != nil {
		return fmt.Errorf("copy plugin: %w", err)
	}

	// Update lockfile
	lock, err := LoadPluginLock(m.config.LockfilePath)
	if err != nil {
		lock = NewPluginLock(m.config.LockfilePath)
	}

	checksum, _ := ComputeChecksum(destPath)
	lock.Add(LockedPlugin{
		Name:          plugin.Manifest.Name,
		Version:       plugin.Manifest.Version,
		InstalledFrom: sourcePath,
		Source:        string(SourceTypeLocal),
		Checksum:      checksum,
		Enabled:       true,
	})

	if err := lock.Save(); err != nil {
		fmt.Printf("Warning: failed to update lockfile: %v\n", err)
	}

	fmt.Printf("✓ Installed plugin: %s v%s\n", plugin.Manifest.Name, plugin.Manifest.Version)
	fmt.Printf("  Path: %s\n", destPath)

	return nil
}

// installFromGitHubWithOptions installs from GitHub with options
func (m *Manager) installFromGitHubWithOptions(src *PluginSource, opts InstallOptions) error {
	// Create temporary directory for cloning
	tmpDir, err := os.MkdirTemp("", "specular-plugin-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Clone repository
	fmt.Printf("Cloning %s...\n", src.String())
	cloneURL := src.GitHubCloneURL()
	if err := src.ValidateVersion(); err != nil {
		return fmt.Errorf("invalid source version: %w", err)
	}

	var gitArgs []string
	if src.Version != "" {
		gitArgs = []string{"clone", "--depth", "1", "--branch", src.Version, cloneURL, tmpDir}
	} else {
		gitArgs = []string{"clone", "--depth", "1", cloneURL, tmpDir}
	}

	cmd, err := safeutil.SafeCommand(context.Background(), "git", gitArgs...)
	if err != nil {
		return fmt.Errorf("prepare git clone: %w", err)
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, output)
	}

	// If there's a subpath, use that as the source
	sourcePath := tmpDir
	if src.Subpath != "" {
		sourcePath, err = safeutil.JoinInsideBase(tmpDir, src.Subpath)
		if err != nil {
			return fmt.Errorf("invalid plugin subpath: %w", err)
		}
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			return fmt.Errorf("subpath not found in repository: %s", src.Subpath)
		}
	}

	// Install from cloned directory
	if err := m.installFromLocalWithOptions(sourcePath, opts); err != nil {
		return err
	}

	// Update lockfile with GitHub source info
	lock, err := LoadPluginLock(m.config.LockfilePath)
	if err == nil {
		pluginName := src.GetPluginName()
		if p, exists := lock.Get(pluginName); exists {
			p.InstalledFrom = src.String()
			p.Source = string(SourceTypeGitHub)
			lock.Add(p)
			_ = lock.Save()
		}
	}

	return nil
}

// Update updates a specific plugin to a new version
func (m *Manager) Update(name string, version string) (*UpdateResult, error) {
	result := &UpdateResult{Name: name}

	// Get plugin info from lockfile
	lock, err := LoadPluginLock(m.config.LockfilePath)
	if err != nil {
		return nil, fmt.Errorf("load lockfile: %w", err)
	}

	lockedPlugin, exists := lock.Get(name)
	if !exists {
		return nil, fmt.Errorf("plugin not found in lockfile: %s", name)
	}

	result.OldVersion = lockedPlugin.Version

	// Parse the original source
	src, err := ParseSource(lockedPlugin.InstalledFrom)
	if err != nil {
		// If source parsing fails, try to construct from repository field
		if plugin, ok := m.Get(name); ok && plugin.Manifest.Repository != "" {
			src, err = ParseSource(plugin.Manifest.Repository)
			if err != nil {
				return nil, fmt.Errorf("cannot determine plugin source: %w", err)
			}
		} else {
			return nil, fmt.Errorf("cannot determine plugin source for update")
		}
	}

	// Local plugins can't be updated automatically
	if src.IsLocal() {
		return nil, fmt.Errorf("cannot update local plugin %s (reinstall from source)", name)
	}

	// Apply version
	if version != "" {
		src = src.WithVersion(version)
	}

	// Install with upgrade flag
	opts := InstallOptions{
		Upgrade: true,
		Version: version,
	}

	if err := m.InstallWithOptions(src.String(), opts); err != nil {
		result.Error = err
		return result, err
	}

	// Get new version
	lock, _ = LoadPluginLock(m.config.LockfilePath)
	if updated, exists := lock.Get(name); exists {
		result.NewVersion = updated.Version
		result.Updated = result.OldVersion != result.NewVersion
	}

	return result, nil
}

// UpdateAll updates all installed plugins
func (m *Manager) UpdateAll() ([]UpdateResult, error) {
	lock, err := LoadPluginLock(m.config.LockfilePath)
	if err != nil {
		return nil, fmt.Errorf("load lockfile: %w", err)
	}

	var results []UpdateResult
	for _, p := range lock.List() {
		// Skip local plugins
		if p.Source == string(SourceTypeLocal) {
			results = append(results, UpdateResult{
				Name:       p.Name,
				OldVersion: p.Version,
				NewVersion: p.Version,
				Updated:    false,
				Error:      fmt.Errorf("local plugin, cannot auto-update"),
			})
			continue
		}

		result, err := m.Update(p.Name, "")
		if err != nil {
			results = append(results, UpdateResult{
				Name:       p.Name,
				OldVersion: p.Version,
				NewVersion: p.Version,
				Updated:    false,
				Error:      err,
			})
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// GetLockfile returns the plugin lockfile
func (m *Manager) GetLockfile() (*PluginLock, error) {
	return LoadPluginLock(m.config.LockfilePath)
}

// VerifyIntegrity checks all installed plugins against their checksums
func (m *Manager) VerifyIntegrity() (map[string]bool, error) {
	lock, err := LoadPluginLock(m.config.LockfilePath)
	if err != nil {
		return nil, fmt.Errorf("load lockfile: %w", err)
	}

	results := make(map[string]bool)
	for _, p := range lock.List() {
		plugin, ok := m.Get(p.Name)
		if !ok {
			results[p.Name] = false
			continue
		}

		valid, err := lock.VerifyChecksum(p.Name, plugin.Path)
		if err != nil {
			results[p.Name] = false
			continue
		}
		results[p.Name] = valid
	}

	return results, nil
}
