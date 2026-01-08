package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := NewManager(WithCacheDir(tmpDir))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if mgr.GetCacheDir() != tmpDir {
		t.Errorf("GetCacheDir() = %s, want %s", mgr.GetCacheDir(), tmpDir)
	}
}

func TestNewManagerWithMaxSize(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := NewManager(WithCacheDir(tmpDir), WithMaxSize(1024*1024))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if mgr.GetMaxSize() != 1024*1024 {
		t.Errorf("GetMaxSize() = %d, want %d", mgr.GetMaxSize(), 1024*1024)
	}
}

func TestManagerGetInfo(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := NewManager(WithCacheDir(tmpDir))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	info, err := mgr.GetInfo()
	if err != nil {
		t.Fatalf("GetInfo() error = %v", err)
	}

	if info.Location != tmpDir {
		t.Errorf("Location = %s, want %s", info.Location, tmpDir)
	}

	// Empty cache should have zero size
	if info.TotalSize != 0 {
		t.Errorf("TotalSize = %d, want 0", info.TotalSize)
	}
}

func TestManagerList(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	modelsDir := filepath.Join(tmpDir, "models")
	os.MkdirAll(modelsDir, 0755)
	os.WriteFile(filepath.Join(modelsDir, "test-model.bin"), []byte("test data"), 0644)

	mgr, err := NewManager(WithCacheDir(tmpDir))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	entries, err := mgr.List(CacheTypeAll)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("List() returned %d entries, want 1", len(entries))
	}

	if len(entries) > 0 && entries[0].Name != "test-model.bin" {
		t.Errorf("Entry name = %s, want test-model.bin", entries[0].Name)
	}
}

func TestManagerListByType(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files in different type directories
	modelsDir := filepath.Join(tmpDir, "models")
	bundlesDir := filepath.Join(tmpDir, "bundles")
	os.MkdirAll(modelsDir, 0755)
	os.MkdirAll(bundlesDir, 0755)
	os.WriteFile(filepath.Join(modelsDir, "model.bin"), []byte("model"), 0644)
	os.WriteFile(filepath.Join(bundlesDir, "bundle.zip"), []byte("bundle"), 0644)

	mgr, err := NewManager(WithCacheDir(tmpDir))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// List only models
	entries, err := mgr.List(CacheTypeModels)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("List(models) returned %d entries, want 1", len(entries))
	}

	// List only bundles
	entries, err = mgr.List(CacheTypeBundles)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("List(bundles) returned %d entries, want 1", len(entries))
	}

	// List all
	entries, err = mgr.List(CacheTypeAll)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("List(all) returned %d entries, want 2", len(entries))
	}
}

func TestManagerClear(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	modelsDir := filepath.Join(tmpDir, "models")
	os.MkdirAll(modelsDir, 0755)
	os.WriteFile(filepath.Join(modelsDir, "test-model.bin"), []byte("test data"), 0644)

	mgr, err := NewManager(WithCacheDir(tmpDir))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	result, err := mgr.Clear(CacheTypeAll)
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	if result.FilesCleared != 1 {
		t.Errorf("FilesCleared = %d, want 1", result.FilesCleared)
	}

	// Verify files are gone
	entries, _ := mgr.List(CacheTypeAll)
	if len(entries) != 0 {
		t.Errorf("After clear, List() returned %d entries, want 0", len(entries))
	}
}

func TestManagerClearByType(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files in different type directories
	modelsDir := filepath.Join(tmpDir, "models")
	bundlesDir := filepath.Join(tmpDir, "bundles")
	os.MkdirAll(modelsDir, 0755)
	os.MkdirAll(bundlesDir, 0755)
	os.WriteFile(filepath.Join(modelsDir, "model.bin"), []byte("model"), 0644)
	os.WriteFile(filepath.Join(bundlesDir, "bundle.zip"), []byte("bundle"), 0644)

	mgr, err := NewManager(WithCacheDir(tmpDir))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Clear only models
	_, err = mgr.Clear(CacheTypeModels)
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Models should be empty
	entries, _ := mgr.List(CacheTypeModels)
	if len(entries) != 0 {
		t.Errorf("After clearing models, List(models) = %d, want 0", len(entries))
	}

	// Bundles should still exist
	entries, _ = mgr.List(CacheTypeBundles)
	if len(entries) != 1 {
		t.Errorf("After clearing models, List(bundles) = %d, want 1", len(entries))
	}
}

func TestManagerPrune(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	modelsDir := filepath.Join(tmpDir, "models")
	os.MkdirAll(modelsDir, 0755)
	filePath := filepath.Join(modelsDir, "test-model.bin")
	os.WriteFile(filePath, []byte("test data"), 0644)

	// Set modification time to 10 days ago
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	os.Chtimes(filePath, oldTime, oldTime)

	mgr, err := NewManager(WithCacheDir(tmpDir))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Prune entries older than 7 days
	result, err := mgr.Prune(7 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}

	if result.FilesPruned != 1 {
		t.Errorf("FilesPruned = %d, want 1", result.FilesPruned)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatSize(%d) = %s, want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestCacheTypes(t *testing.T) {
	types := []CacheType{CacheTypeModels, CacheTypeBundles, CacheTypeTraces, CacheTypeAll}

	for _, ct := range types {
		if string(ct) == "" {
			t.Errorf("CacheType should not be empty string")
		}
	}

	// Check expected values
	if CacheTypeModels != "models" {
		t.Errorf("CacheTypeModels = %s, want models", CacheTypeModels)
	}
	if CacheTypeBundles != "bundles" {
		t.Errorf("CacheTypeBundles = %s, want bundles", CacheTypeBundles)
	}
	if CacheTypeTraces != "traces" {
		t.Errorf("CacheTypeTraces = %s, want traces", CacheTypeTraces)
	}
}

func TestCacheConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "cache-config.json")

	// Save config
	config := &CacheConfig{
		MaxSize:  1024 * 1024 * 100, // 100 MB
		Location: "/custom/cache",
	}

	err := SaveConfig(configPath, config)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Load config
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.MaxSize != config.MaxSize {
		t.Errorf("MaxSize = %d, want %d", loaded.MaxSize, config.MaxSize)
	}
	if loaded.Location != config.Location {
		t.Errorf("Location = %s, want %s", loaded.Location, config.Location)
	}
}

func TestLoadConfigNotExist(t *testing.T) {
	config, err := LoadConfig("/nonexistent/path")
	if err != nil {
		t.Fatalf("LoadConfig() should not error for missing file, got %v", err)
	}

	if config == nil {
		t.Error("LoadConfig() should return empty config for missing file")
	}
}

func TestCacheEntryStructure(t *testing.T) {
	entry := CacheEntry{
		Path:       "/cache/models/test.bin",
		Type:       CacheTypeModels,
		Size:       1024,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		Name:       "test.bin",
	}

	if entry.Path == "" {
		t.Error("Path should not be empty")
	}
	if entry.Type != CacheTypeModels {
		t.Error("Type should be models")
	}
	if entry.Size != 1024 {
		t.Error("Size should be 1024")
	}
}
