package plugin

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewPluginLock(t *testing.T) {
	lock := NewPluginLock("")
	if lock == nil {
		t.Fatal("NewPluginLock returned nil")
	}
	if lock.Version != LockfileVersion {
		t.Errorf("Version = %q, want %q", lock.Version, LockfileVersion)
	}
	if lock.Plugins == nil {
		t.Error("Plugins map should be initialized")
	}
	if len(lock.Plugins) != 0 {
		t.Errorf("Plugins should be empty, got %d entries", len(lock.Plugins))
	}
}

func TestNewPluginLockWithPath(t *testing.T) {
	customPath := "/custom/path/plugins.lock.json"
	lock := NewPluginLock(customPath)
	if lock.Path() != customPath {
		t.Errorf("Path() = %q, want %q", lock.Path(), customPath)
	}
}

func TestLoadPluginLock_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "nonexistent", "plugins.lock.json")

	lock, err := LoadPluginLock(lockPath)
	if err != nil {
		t.Fatalf("LoadPluginLock() error = %v, want nil for non-existent file", err)
	}
	if lock == nil {
		t.Fatal("LoadPluginLock() returned nil")
	}
	if lock.Version != LockfileVersion {
		t.Errorf("Version = %q, want %q", lock.Version, LockfileVersion)
	}
}

func TestPluginLock_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "plugins.lock.json")

	// Create and save
	lock := NewPluginLock(lockPath)
	lock.Add(LockedPlugin{
		Name:          "test-plugin",
		Version:       "1.0.0",
		InstalledFrom: "./test-plugin",
		Source:        "local",
		Checksum:      "abc123",
		Enabled:       true,
	})

	if err := lock.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatal("Lockfile not created")
	}

	// Load and verify
	loaded, err := LoadPluginLock(lockPath)
	if err != nil {
		t.Fatalf("LoadPluginLock() error = %v", err)
	}

	plugin, exists := loaded.Get("test-plugin")
	if !exists {
		t.Fatal("Plugin not found after load")
	}
	if plugin.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", plugin.Version, "1.0.0")
	}
	if plugin.Source != "local" {
		t.Errorf("Source = %q, want %q", plugin.Source, "local")
	}
}

func TestPluginLock_Add(t *testing.T) {
	lock := NewPluginLock("")

	plugin := LockedPlugin{
		Name:          "my-plugin",
		Version:       "2.0.0",
		InstalledFrom: "github.com/user/repo",
		Source:        "github",
		Enabled:       true,
	}

	lock.Add(plugin)

	if !lock.Has("my-plugin") {
		t.Error("Plugin not added")
	}

	got, exists := lock.Get("my-plugin")
	if !exists {
		t.Fatal("Get() returned false for existing plugin")
	}
	if got.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", got.Version, "2.0.0")
	}
	if got.InstalledAt.IsZero() {
		t.Error("InstalledAt should be set automatically")
	}
}

func TestPluginLock_AddWithTimestamp(t *testing.T) {
	lock := NewPluginLock("")

	customTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	plugin := LockedPlugin{
		Name:        "timed-plugin",
		Version:     "1.0.0",
		InstalledAt: customTime,
	}

	lock.Add(plugin)

	got, _ := lock.Get("timed-plugin")
	if !got.InstalledAt.Equal(customTime) {
		t.Errorf("InstalledAt = %v, want %v", got.InstalledAt, customTime)
	}
}

func TestPluginLock_Remove(t *testing.T) {
	lock := NewPluginLock("")
	lock.Add(LockedPlugin{Name: "plugin-to-remove", Version: "1.0.0"})

	if !lock.Has("plugin-to-remove") {
		t.Fatal("Plugin should exist before removal")
	}

	removed := lock.Remove("plugin-to-remove")
	if !removed {
		t.Error("Remove() returned false for existing plugin")
	}

	if lock.Has("plugin-to-remove") {
		t.Error("Plugin should not exist after removal")
	}

	// Remove non-existent
	removed = lock.Remove("non-existent")
	if removed {
		t.Error("Remove() returned true for non-existent plugin")
	}
}

func TestPluginLock_List(t *testing.T) {
	lock := NewPluginLock("")
	lock.Add(LockedPlugin{Name: "plugin-a", Version: "1.0.0"})
	lock.Add(LockedPlugin{Name: "plugin-b", Version: "2.0.0"})
	lock.Add(LockedPlugin{Name: "plugin-c", Version: "3.0.0"})

	plugins := lock.List()
	if len(plugins) != 3 {
		t.Errorf("List() returned %d plugins, want 3", len(plugins))
	}

	names := make(map[string]bool)
	for _, p := range plugins {
		names[p.Name] = true
	}

	for _, name := range []string{"plugin-a", "plugin-b", "plugin-c"} {
		if !names[name] {
			t.Errorf("List() missing plugin %q", name)
		}
	}
}

func TestPluginLock_SetEnabled(t *testing.T) {
	lock := NewPluginLock("")
	lock.Add(LockedPlugin{Name: "toggle-plugin", Version: "1.0.0", Enabled: true})

	// Disable
	if !lock.SetEnabled("toggle-plugin", false) {
		t.Error("SetEnabled() returned false for existing plugin")
	}

	plugin, _ := lock.Get("toggle-plugin")
	if plugin.Enabled {
		t.Error("Plugin should be disabled")
	}

	// Enable
	lock.SetEnabled("toggle-plugin", true)
	plugin, _ = lock.Get("toggle-plugin")
	if !plugin.Enabled {
		t.Error("Plugin should be enabled")
	}

	// Non-existent
	if lock.SetEnabled("non-existent", true) {
		t.Error("SetEnabled() returned true for non-existent plugin")
	}
}

func TestPluginLock_IsOutdated(t *testing.T) {
	lock := NewPluginLock("")
	lock.Add(LockedPlugin{Name: "versioned-plugin", Version: "1.0.0"})

	tests := []struct {
		available string
		outdated  bool
	}{
		{"2.0.0", true},
		{"1.1.0", true},
		{"1.0.1", true},
		{"1.0.0", false},
		{"0.9.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.available, func(t *testing.T) {
			outdated, err := lock.IsOutdated("versioned-plugin", tt.available)
			if err != nil {
				t.Fatalf("IsOutdated() error = %v", err)
			}
			if outdated != tt.outdated {
				t.Errorf("IsOutdated() = %v, want %v", outdated, tt.outdated)
			}
		})
	}
}

func TestPluginLock_IsOutdated_NotFound(t *testing.T) {
	lock := NewPluginLock("")

	_, err := lock.IsOutdated("non-existent", "1.0.0")
	if err == nil {
		t.Error("IsOutdated() should error for non-existent plugin")
	}
}

func TestPluginLock_GetOutdated(t *testing.T) {
	lock := NewPluginLock("")
	lock.Add(LockedPlugin{Name: "plugin-a", Version: "1.0.0"})
	lock.Add(LockedPlugin{Name: "plugin-b", Version: "2.0.0"})
	lock.Add(LockedPlugin{Name: "plugin-c", Version: "1.5.0"})

	availableVersions := map[string]string{
		"plugin-a": "2.0.0", // outdated
		"plugin-b": "2.0.0", // up to date
		"plugin-c": "1.0.0", // newer than available (shouldn't happen)
	}

	outdated, err := lock.GetOutdated(availableVersions)
	if err != nil {
		t.Fatalf("GetOutdated() error = %v", err)
	}

	if len(outdated) != 1 {
		t.Errorf("GetOutdated() returned %d plugins, want 1", len(outdated))
	}

	if len(outdated) > 0 && outdated[0].Name != "plugin-a" {
		t.Errorf("GetOutdated() returned %q, want plugin-a", outdated[0].Name)
	}
}

func TestPluginLock_GetDependents(t *testing.T) {
	lock := NewPluginLock("")
	lock.Add(LockedPlugin{Name: "core-lib", Version: "1.0.0"})
	lock.Add(LockedPlugin{Name: "plugin-a", Version: "1.0.0", Dependencies: []string{"core-lib"}})
	lock.Add(LockedPlugin{Name: "plugin-b", Version: "1.0.0", Dependencies: []string{"core-lib", "other"}})
	lock.Add(LockedPlugin{Name: "plugin-c", Version: "1.0.0", Dependencies: []string{"other"}})

	dependents := lock.GetDependents("core-lib")
	if len(dependents) != 2 {
		t.Errorf("GetDependents() returned %d plugins, want 2", len(dependents))
	}

	names := make(map[string]bool)
	for _, p := range dependents {
		names[p.Name] = true
	}

	if !names["plugin-a"] || !names["plugin-b"] {
		t.Error("GetDependents() missing expected dependents")
	}
}

func TestPluginLock_GetMissingDependencies(t *testing.T) {
	lock := NewPluginLock("")
	lock.Add(LockedPlugin{Name: "core-lib", Version: "1.0.0"})
	lock.Add(LockedPlugin{
		Name:         "dependent",
		Version:      "1.0.0",
		Dependencies: []string{"core-lib", "missing-lib", "another-missing"},
	})

	missing := lock.GetMissingDependencies("dependent")
	if len(missing) != 2 {
		t.Errorf("GetMissingDependencies() returned %d, want 2", len(missing))
	}

	missingMap := make(map[string]bool)
	for _, m := range missing {
		missingMap[m] = true
	}

	if !missingMap["missing-lib"] || !missingMap["another-missing"] {
		t.Error("GetMissingDependencies() missing expected dependencies")
	}
}

func TestPluginLock_GetMissingDependencies_NotFound(t *testing.T) {
	lock := NewPluginLock("")

	missing := lock.GetMissingDependencies("non-existent")
	if missing != nil {
		t.Errorf("GetMissingDependencies() = %v, want nil for non-existent plugin", missing)
	}
}

func TestPluginLock_Clean(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real plugin directory
	existingPluginDir := filepath.Join(tmpDir, "plugins", "existing-plugin")
	if err := os.MkdirAll(existingPluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin dir: %v", err)
	}

	lock := NewPluginLock("")
	lock.Add(LockedPlugin{Name: "existing-plugin", Version: "1.0.0"})
	lock.Add(LockedPlugin{Name: "missing-plugin", Version: "1.0.0"})

	pluginDirs := []string{filepath.Join(tmpDir, "plugins")}
	removed := lock.Clean(pluginDirs)

	if len(removed) != 1 {
		t.Errorf("Clean() removed %d plugins, want 1", len(removed))
	}

	if len(removed) > 0 && removed[0] != "missing-plugin" {
		t.Errorf("Clean() removed %q, want missing-plugin", removed[0])
	}

	if !lock.Has("existing-plugin") {
		t.Error("existing-plugin should still be in lockfile")
	}
	if lock.Has("missing-plugin") {
		t.Error("missing-plugin should be removed from lockfile")
	}
}

func TestPluginLock_Merge(t *testing.T) {
	lock1 := NewPluginLock("")
	lock1.Add(LockedPlugin{Name: "plugin-a", Version: "1.0.0"})
	lock1.Add(LockedPlugin{Name: "shared", Version: "1.0.0"})

	lock2 := NewPluginLock("")
	lock2.Add(LockedPlugin{Name: "plugin-b", Version: "2.0.0"})
	lock2.Add(LockedPlugin{Name: "shared", Version: "2.0.0"})

	// Merge without force
	lock1.Merge(lock2, false)

	if !lock1.Has("plugin-a") {
		t.Error("plugin-a should be preserved")
	}
	if !lock1.Has("plugin-b") {
		t.Error("plugin-b should be added")
	}

	shared, _ := lock1.Get("shared")
	if shared.Version != "1.0.0" {
		t.Errorf("shared Version = %q, want 1.0.0 (preserve existing)", shared.Version)
	}

	// Merge with force
	lock1.Merge(lock2, true)
	shared, _ = lock1.Get("shared")
	if shared.Version != "2.0.0" {
		t.Errorf("shared Version = %q, want 2.0.0 (force update)", shared.Version)
	}
}

func TestComputeChecksum(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a plugin directory with manifest
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin dir: %v", err)
	}

	manifestContent := `name: test-plugin
version: "1.0.0"
type: provider
`
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	checksum, err := ComputeChecksum(pluginDir)
	if err != nil {
		t.Fatalf("ComputeChecksum() error = %v", err)
	}

	if checksum == "" {
		t.Error("ComputeChecksum() returned empty string")
	}

	// Verify same content gives same checksum
	checksum2, _ := ComputeChecksum(pluginDir)
	if checksum != checksum2 {
		t.Error("ComputeChecksum() should be deterministic")
	}

	// Modify content and verify checksum changes
	if err := os.WriteFile(manifestPath, []byte(manifestContent+"# modified\n"), 0644); err != nil {
		t.Fatalf("Failed to modify manifest: %v", err)
	}

	checksum3, _ := ComputeChecksum(pluginDir)
	if checksum == checksum3 {
		t.Error("ComputeChecksum() should change when content changes")
	}
}

func TestPluginLock_VerifyChecksum(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a plugin directory
	pluginDir := filepath.Join(tmpDir, "verified-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin dir: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	if err := os.WriteFile(manifestPath, []byte("name: verified-plugin\n"), 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	checksum, _ := ComputeChecksum(pluginDir)

	lock := NewPluginLock("")
	lock.Add(LockedPlugin{
		Name:     "verified-plugin",
		Version:  "1.0.0",
		Checksum: checksum,
	})

	// Verify valid checksum
	valid, err := lock.VerifyChecksum("verified-plugin", pluginDir)
	if err != nil {
		t.Fatalf("VerifyChecksum() error = %v", err)
	}
	if !valid {
		t.Error("VerifyChecksum() should return true for matching checksum")
	}

	// Modify file and verify checksum fails
	if err := os.WriteFile(manifestPath, []byte("name: verified-plugin\n# modified\n"), 0644); err != nil {
		t.Fatalf("Failed to modify manifest: %v", err)
	}

	valid, err = lock.VerifyChecksum("verified-plugin", pluginDir)
	if err != nil {
		t.Fatalf("VerifyChecksum() error = %v", err)
	}
	if valid {
		t.Error("VerifyChecksum() should return false for mismatched checksum")
	}
}

func TestPluginLock_UpdateChecksum(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a plugin directory
	pluginDir := filepath.Join(tmpDir, "update-checksum-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin dir: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	if err := os.WriteFile(manifestPath, []byte("name: update-checksum-plugin\n"), 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	lock := NewPluginLock("")
	lock.Add(LockedPlugin{
		Name:     "update-checksum-plugin",
		Version:  "1.0.0",
		Checksum: "old-checksum",
	})

	if err := lock.UpdateChecksum("update-checksum-plugin", pluginDir); err != nil {
		t.Fatalf("UpdateChecksum() error = %v", err)
	}

	plugin, _ := lock.Get("update-checksum-plugin")
	if plugin.Checksum == "old-checksum" {
		t.Error("UpdateChecksum() should have updated the checksum")
	}

	// Verify new checksum is valid
	valid, _ := lock.VerifyChecksum("update-checksum-plugin", pluginDir)
	if !valid {
		t.Error("UpdateChecksum() should set a valid checksum")
	}
}

func TestPluginLock_ConcurrentAccess(t *testing.T) {
	lock := NewPluginLock("")

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			lock.Add(LockedPlugin{
				Name:    "concurrent-plugin",
				Version: "1.0.0",
			})
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should still work correctly
	if !lock.Has("concurrent-plugin") {
		t.Error("Concurrent writes should succeed")
	}
}

func TestDefaultLockfilePath(t *testing.T) {
	path := DefaultLockfilePath()
	if path == "" {
		t.Error("DefaultLockfilePath() returned empty string")
	}
	if !filepath.IsAbs(path) && path != ".specular/plugins.lock.json" {
		t.Errorf("DefaultLockfilePath() returned non-absolute path: %s", path)
	}
}

func TestLoadPluginLock_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "invalid.lock.json")

	// Write invalid JSON
	if err := os.WriteFile(lockPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	_, err := LoadPluginLock(lockPath)
	if err == nil {
		t.Error("LoadPluginLock() should error for invalid JSON")
	}
}

func TestPluginLock_VerifyChecksum_EmptyChecksum(t *testing.T) {
	lock := NewPluginLock("")
	lock.Add(LockedPlugin{
		Name:     "no-checksum",
		Version:  "1.0.0",
		Checksum: "", // Empty checksum
	})

	// Should return true when no checksum is recorded
	valid, err := lock.VerifyChecksum("no-checksum", "/some/path")
	if err != nil {
		t.Fatalf("VerifyChecksum() error = %v", err)
	}
	if !valid {
		t.Error("VerifyChecksum() should return true when no checksum is recorded")
	}
}

func TestPluginLock_UpdateChecksum_NotFound(t *testing.T) {
	lock := NewPluginLock("")

	err := lock.UpdateChecksum("non-existent", "/some/path")
	if err == nil {
		t.Error("UpdateChecksum() should error for non-existent plugin")
	}
}

func TestPluginLock_VerifyChecksum_NotFound(t *testing.T) {
	lock := NewPluginLock("")

	_, err := lock.VerifyChecksum("non-existent", "/some/path")
	if err == nil {
		t.Error("VerifyChecksum() should error for non-existent plugin")
	}
}
