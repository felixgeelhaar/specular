package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LockfileVersion is the current lockfile format version
const LockfileVersion = "1.0.0"

// DefaultLockfilePath returns the default lockfile location
func DefaultLockfilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".specular/plugins.lock.json"
	}
	return filepath.Join(home, ".specular", "plugins.lock.json")
}

// PluginLock tracks all installed plugins
type PluginLock struct {
	Version string                   `json:"version"`
	Plugins map[string]LockedPlugin  `json:"plugins"`
	Updated time.Time                `json:"updated"`

	path string
	mu   sync.RWMutex
}

// LockedPlugin represents a single installed plugin in the lockfile
type LockedPlugin struct {
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	InstalledFrom string    `json:"installed_from"` // Original source (path, URL, registry name)
	Source        string    `json:"source"`         // Source type: local, github, registry
	Checksum      string    `json:"checksum"`       // SHA256 of manifest or directory
	InstalledAt   time.Time `json:"installed_at"`
	Dependencies  []string  `json:"dependencies,omitempty"`
	Enabled       bool      `json:"enabled"`
}

// NewPluginLock creates a new empty lockfile
func NewPluginLock(path string) *PluginLock {
	if path == "" {
		path = DefaultLockfilePath()
	}
	return &PluginLock{
		Version: LockfileVersion,
		Plugins: make(map[string]LockedPlugin),
		Updated: time.Now(),
		path:    path,
	}
}

// LoadPluginLock loads an existing lockfile or creates a new one
func LoadPluginLock(path string) (*PluginLock, error) {
	if path == "" {
		path = DefaultLockfilePath()
	}

	lock := &PluginLock{
		path: path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return a new empty lockfile
			lock.Version = LockfileVersion
			lock.Plugins = make(map[string]LockedPlugin)
			lock.Updated = time.Now()
			return lock, nil
		}
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}

	if err := json.Unmarshal(data, lock); err != nil {
		return nil, fmt.Errorf("failed to parse lockfile: %w", err)
	}

	// Initialize map if nil
	if lock.Plugins == nil {
		lock.Plugins = make(map[string]LockedPlugin)
	}

	return lock, nil
}

// Save writes the lockfile to disk
func (l *PluginLock) Save() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Updated = time.Now()

	// Ensure directory exists
	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create lockfile directory: %w", err)
	}

	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize lockfile: %w", err)
	}

	// Write atomically using temp file
	tmpPath := l.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}

	if err := os.Rename(tmpPath, l.path); err != nil {
		os.Remove(tmpPath) // Clean up temp file on failure
		return fmt.Errorf("failed to commit lockfile: %w", err)
	}

	return nil
}

// Add adds or updates a plugin in the lockfile
func (l *PluginLock) Add(plugin LockedPlugin) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if plugin.InstalledAt.IsZero() {
		plugin.InstalledAt = time.Now()
	}
	l.Plugins[plugin.Name] = plugin
}

// Remove removes a plugin from the lockfile
func (l *PluginLock) Remove(name string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.Plugins[name]; exists {
		delete(l.Plugins, name)
		return true
	}
	return false
}

// Get retrieves a plugin from the lockfile
func (l *PluginLock) Get(name string) (LockedPlugin, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugin, exists := l.Plugins[name]
	return plugin, exists
}

// Has checks if a plugin exists in the lockfile
func (l *PluginLock) Has(name string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	_, exists := l.Plugins[name]
	return exists
}

// List returns all plugins in the lockfile
func (l *PluginLock) List() []LockedPlugin {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugins := make([]LockedPlugin, 0, len(l.Plugins))
	for _, p := range l.Plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// SetEnabled updates the enabled state of a plugin
func (l *PluginLock) SetEnabled(name string, enabled bool) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if plugin, exists := l.Plugins[name]; exists {
		plugin.Enabled = enabled
		l.Plugins[name] = plugin
		return true
	}
	return false
}

// Path returns the lockfile path
func (l *PluginLock) Path() string {
	return l.path
}

// IsOutdated checks if a plugin version is outdated compared to available version
func (l *PluginLock) IsOutdated(name string, availableVersion string) (bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugin, exists := l.Plugins[name]
	if !exists {
		return false, fmt.Errorf("plugin not found: %s", name)
	}

	installed, err := ParseVersion(plugin.Version)
	if err != nil {
		return false, fmt.Errorf("invalid installed version: %w", err)
	}

	available, err := ParseVersion(availableVersion)
	if err != nil {
		return false, fmt.Errorf("invalid available version: %w", err)
	}

	return installed.Compare(available) < 0, nil
}

// GetOutdated returns all plugins that have newer versions available
func (l *PluginLock) GetOutdated(availableVersions map[string]string) ([]LockedPlugin, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var outdated []LockedPlugin
	for name, plugin := range l.Plugins {
		availableVersion, exists := availableVersions[name]
		if !exists {
			continue
		}

		installed, err := ParseVersion(plugin.Version)
		if err != nil {
			continue
		}

		available, err := ParseVersion(availableVersion)
		if err != nil {
			continue
		}

		if installed.Compare(available) < 0 {
			outdated = append(outdated, plugin)
		}
	}

	return outdated, nil
}

// ComputeChecksum calculates SHA256 checksum for a plugin directory
func ComputeChecksum(pluginPath string) (string, error) {
	// For directories, hash the manifest file
	manifestPath := filepath.Join(pluginPath, "plugin.yaml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifestPath = filepath.Join(pluginPath, "plugin.yml")
	}

	info, err := os.Stat(pluginPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat plugin path: %w", err)
	}

	if info.IsDir() {
		// Hash manifest file for directories
		return hashFile(manifestPath)
	}

	// Hash the file directly if it's a file
	return hashFile(pluginPath)
}

// hashFile computes SHA256 hash of a file
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyChecksum verifies a plugin's checksum against the lockfile
func (l *PluginLock) VerifyChecksum(name string, pluginPath string) (bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugin, exists := l.Plugins[name]
	if !exists {
		return false, fmt.Errorf("plugin not found: %s", name)
	}

	if plugin.Checksum == "" {
		// No checksum recorded, consider it valid
		return true, nil
	}

	currentChecksum, err := ComputeChecksum(pluginPath)
	if err != nil {
		return false, err
	}

	return currentChecksum == plugin.Checksum, nil
}

// UpdateChecksum updates the checksum for a plugin
func (l *PluginLock) UpdateChecksum(name string, pluginPath string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	plugin, exists := l.Plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	checksum, err := ComputeChecksum(pluginPath)
	if err != nil {
		return err
	}

	plugin.Checksum = checksum
	l.Plugins[name] = plugin
	return nil
}

// GetDependents returns plugins that depend on the given plugin
func (l *PluginLock) GetDependents(name string) []LockedPlugin {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var dependents []LockedPlugin
	for _, plugin := range l.Plugins {
		for _, dep := range plugin.Dependencies {
			if dep == name {
				dependents = append(dependents, plugin)
				break
			}
		}
	}
	return dependents
}

// GetMisssingDependencies returns dependencies that are not installed
func (l *PluginLock) GetMissingDependencies(name string) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugin, exists := l.Plugins[name]
	if !exists {
		return nil
	}

	var missing []string
	for _, dep := range plugin.Dependencies {
		if _, exists := l.Plugins[dep]; !exists {
			missing = append(missing, dep)
		}
	}
	return missing
}

// Clean removes entries for plugins that no longer exist on disk
func (l *PluginLock) Clean(pluginDirs []string) []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	var removed []string
	for name, plugin := range l.Plugins {
		// Check if plugin still exists in any plugin directory
		found := false
		for _, dir := range pluginDirs {
			pluginPath := filepath.Join(dir, name)
			if _, err := os.Stat(pluginPath); err == nil {
				found = true
				break
			}
			// Also check InstalledFrom if it's a local path
			if plugin.Source == "local" {
				if _, err := os.Stat(plugin.InstalledFrom); err == nil {
					found = true
					break
				}
			}
		}

		if !found {
			delete(l.Plugins, name)
			removed = append(removed, name)
		}
	}
	return removed
}

// Merge combines another lockfile into this one
// Existing entries are preserved unless force is true
func (l *PluginLock) Merge(other *PluginLock, force bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for name, plugin := range other.Plugins {
		if _, exists := l.Plugins[name]; !exists || force {
			l.Plugins[name] = plugin
		}
	}
}
