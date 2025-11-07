package exec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewImageCache(t *testing.T) {
	cache := NewImageCache("/tmp/test-cache", 24*time.Hour)
	if cache == nil {
		t.Fatal("Expected cache to be created")
	}
	if cache.CacheDir != "/tmp/test-cache" {
		t.Errorf("Expected cache dir /tmp/test-cache, got %s", cache.CacheDir)
	}
	if cache.MaxAge != 24*time.Hour {
		t.Errorf("Expected max age 24h, got %v", cache.MaxAge)
	}
	if cache.imageStates == nil {
		t.Error("Expected imageStates map to be initialized")
	}
}

func TestImageCacheManifest(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewImageCache(tempDir, 24*time.Hour)

	// Test saving manifest
	cache.imageStates["alpine:latest"] = &ImageState{
		Image:     "alpine:latest",
		Digest:    "sha256:abc123",
		CachedAt:  time.Now(),
		LastUsed:  time.Now(),
		PullTime:  1000,
		SizeBytes: 5242880,
	}

	err := cache.SaveManifest()
	if err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}

	// Verify manifest file exists
	manifestPath := filepath.Join(tempDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Fatal("Manifest file was not created")
	}

	// Test loading manifest
	newCache := NewImageCache(tempDir, 24*time.Hour)
	err = newCache.LoadManifest()
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	state, exists := newCache.imageStates["alpine:latest"]
	if !exists {
		t.Fatal("Expected alpine:latest in loaded manifest")
	}
	if state.Image != "alpine:latest" {
		t.Errorf("Expected image alpine:latest, got %s", state.Image)
	}
	if state.Digest != "sha256:abc123" {
		t.Errorf("Expected digest sha256:abc123, got %s", state.Digest)
	}
}

func TestLoadManifestNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewImageCache(tempDir, 24*time.Hour)

	// Should not error when manifest doesn't exist
	err := cache.LoadManifest()
	if err != nil {
		t.Errorf("Expected no error when manifest doesn't exist, got: %v", err)
	}

	// Should have empty map
	if len(cache.imageStates) != 0 {
		t.Errorf("Expected empty imageStates, got %d entries", len(cache.imageStates))
	}
}

func TestLoadManifestCorrupted(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")

	// Write invalid JSON
	err := os.WriteFile(manifestPath, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted manifest: %v", err)
	}

	cache := NewImageCache(tempDir, 24*time.Hour)
	err = cache.LoadManifest()
	if err == nil {
		t.Error("Expected error when loading corrupted manifest")
	}
}

func TestGetCacheStats(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewImageCache(tempDir, 24*time.Hour)

	// Add some test images
	now := time.Now()
	cache.imageStates["alpine:latest"] = &ImageState{
		Image:     "alpine:latest",
		CachedAt:  now.Add(-2 * time.Hour),
		SizeBytes: 5242880,
	}
	cache.imageStates["golang:1.22"] = &ImageState{
		Image:     "golang:1.22",
		CachedAt:  now,
		SizeBytes: 800000000,
	}

	stats := cache.GetStats()
	if stats["total_images"] != 2 {
		t.Errorf("Expected 2 images, got %v", stats["total_images"])
	}

	sizeMB := stats["total_size_mb"].(float64)
	expectedMB := float64(5242880+800000000) / (1024 * 1024)
	if sizeMB < expectedMB-1 || sizeMB > expectedMB+1 {
		t.Errorf("Expected size ~%.2f MB, got %.2f MB", expectedMB, sizeMB)
	}

	if stats["cache_dir"] != tempDir {
		t.Errorf("Expected cache_dir %s, got %v", tempDir, stats["cache_dir"])
	}
}

func TestGenerateCacheKey(t *testing.T) {
	tests := []struct {
		image    string
		expected string // Just check it's not empty and deterministic
	}{
		{"alpine:latest", ""},
		{"golang:1.22", ""},
		{"gcr.io/project/image:tag", ""},
	}

	for _, tt := range tests {
		key1 := generateCacheKey(tt.image)
		key2 := generateCacheKey(tt.image)

		if key1 == "" {
			t.Errorf("Expected non-empty cache key for %s", tt.image)
		}

		if key1 != key2 {
			t.Errorf("Cache key should be deterministic for %s: %s != %s", tt.image, key1, key2)
		}

		// Check format
		if len(key1) < 10 {
			t.Errorf("Cache key seems too short for %s: %s", tt.image, key1)
		}
	}
}

func TestGetRequiredImages(t *testing.T) {
	tests := []struct {
		name     string
		tasks    []struct{ Skill string }
		expected []string
	}{
		{
			name: "go backend tasks",
			tasks: []struct{ Skill string }{
				{Skill: "go-backend"},
				{Skill: "testing"},
			},
			expected: []string{"golang:1.22"},
		},
		{
			name: "mixed tasks",
			tasks: []struct{ Skill string }{
				{Skill: "go-backend"},
				{Skill: "ui-react"},
				{Skill: "database"},
			},
			expected: []string{"golang:1.22", "node:20", "postgres:15"},
		},
		{
			name: "unknown skill",
			tasks: []struct{ Skill string }{
				{Skill: "unknown"},
			},
			expected: []string{"alpine:latest"},
		},
		{
			name:     "empty tasks",
			tasks:    []struct{ Skill string }{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images := GetRequiredImages(tt.tasks)

			if len(images) != len(tt.expected) {
				t.Errorf("Expected %d images, got %d: %v", len(tt.expected), len(images), images)
			}

			for _, exp := range tt.expected {
				found := false
				for _, img := range images {
					if img == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected image %s not found in %v", exp, images)
				}
			}
		})
	}
}

func TestImageCacheStatsEmptyCache(t *testing.T) {
	cache := NewImageCache(t.TempDir(), 24*time.Hour)
	stats := cache.GetStats()

	if stats["total_images"] != 0 {
		t.Errorf("Expected 0 images, got %v", stats["total_images"])
	}

	sizeMB := stats["total_size_mb"].(float64)
	if sizeMB != 0 {
		t.Errorf("Expected 0 MB, got %.2f MB", sizeMB)
	}
}

func TestImageStateJSON(t *testing.T) {
	now := time.Now()
	state := &ImageState{
		Image:     "alpine:latest",
		Digest:    "sha256:test",
		CachedAt:  now,
		LastUsed:  now,
		PullTime:  1234,
		SizeBytes: 5000000,
	}

	// Marshal to JSON
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal ImageState: %v", err)
	}

	// Unmarshal back
	var loaded ImageState
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		t.Fatalf("Failed to unmarshal ImageState: %v", err)
	}

	if loaded.Image != state.Image {
		t.Errorf("Image mismatch: %s != %s", loaded.Image, state.Image)
	}
	if loaded.Digest != state.Digest {
		t.Errorf("Digest mismatch: %s != %s", loaded.Digest, state.Digest)
	}
	if loaded.PullTime != state.PullTime {
		t.Errorf("PullTime mismatch: %d != %d", loaded.PullTime, state.PullTime)
	}
	if loaded.SizeBytes != state.SizeBytes {
		t.Errorf("SizeBytes mismatch: %d != %d", loaded.SizeBytes, state.SizeBytes)
	}
}

func TestCacheManifestJSON(t *testing.T) {
	now := time.Now()
	manifest := CacheManifest{
		Version: "1.0",
		Images: map[string]*ImageState{
			"alpine:latest": {
				Image:     "alpine:latest",
				Digest:    "sha256:test",
				CachedAt:  now,
				LastUsed:  now,
				PullTime:  1000,
				SizeBytes: 5000000,
			},
		},
		UpdatedAt: now,
	}

	// Marshal to JSON
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal CacheManifest: %v", err)
	}

	// Unmarshal back
	var loaded CacheManifest
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		t.Fatalf("Failed to unmarshal CacheManifest: %v", err)
	}

	if loaded.Version != manifest.Version {
		t.Errorf("Version mismatch: %s != %s", loaded.Version, manifest.Version)
	}

	state, exists := loaded.Images["alpine:latest"]
	if !exists {
		t.Fatal("alpine:latest not found in loaded manifest")
	}
	if state.Image != "alpine:latest" {
		t.Errorf("Image mismatch: %s != alpine:latest", state.Image)
	}
}

func TestSaveManifestCreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "subdir", "cache")
	cache := NewImageCache(cacheDir, 24*time.Hour)

	cache.imageStates["test:latest"] = &ImageState{
		Image:    "test:latest",
		CachedAt: time.Now(),
	}

	err := cache.SaveManifest()
	if err != nil {
		t.Fatalf("SaveManifest should create directories: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("Cache directory was not created")
	}

	// Verify manifest exists
	manifestPath := filepath.Join(cacheDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("Manifest file was not created")
	}
}
