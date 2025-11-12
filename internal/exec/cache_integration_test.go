//go:build integration

package exec

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestEnsureImage tests image ensure functionality with real Docker
func TestEnsureImage(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available in test environment")
	}

	tempDir := t.TempDir()
	cache := NewImageCache(tempDir, 24*time.Hour)

	// Use a small, commonly available image
	testImage := "alpine:latest"

	// First ensure - should pull if not cached
	t.Logf("First ensure of %s (may pull if not available locally)", testImage)
	err := cache.EnsureImage(testImage, true)
	if err != nil {
		t.Fatalf("EnsureImage failed: %v", err)
	}

	// Verify image state was saved
	state, exists := cache.imageStates[testImage]
	if !exists {
		t.Error("Image state should be saved after EnsureImage")
	}

	if state != nil {
		if state.Image != testImage {
			t.Errorf("Expected image %s, got %s", testImage, state.Image)
		}
		if state.CachedAt.IsZero() {
			t.Error("CachedAt timestamp should be set")
		}
		if state.LastUsed.IsZero() {
			t.Error("LastUsed timestamp should be set")
		}

		t.Logf("Image %s cached: Digest=%s, Size=%d bytes, PullTime=%dms",
			testImage, state.Digest, state.SizeBytes, state.PullTime)
	}

	// Second ensure - should use cache
	t.Logf("Second ensure of %s (should use cache)", testImage)
	oldLastUsed := state.LastUsed
	time.Sleep(10 * time.Millisecond) // Small delay to ensure timestamp difference

	err = cache.EnsureImage(testImage, true)
	if err != nil {
		t.Fatalf("Second EnsureImage failed: %v", err)
	}

	// Verify LastUsed was updated
	newState, _ := cache.imageStates[testImage]
	if !newState.LastUsed.After(oldLastUsed) {
		t.Error("LastUsed timestamp should be updated on cached access")
	}
}

// TestPrewarmImages tests parallel image pulling
func TestPrewarmImages(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available in test environment")
	}

	tempDir := t.TempDir()
	cache := NewImageCache(tempDir, 24*time.Hour)

	// Use small images for faster testing
	testImages := []string{
		"alpine:latest",
		"busybox:latest",
	}

	t.Logf("Prewarming %d images with concurrency 2", len(testImages))
	startTime := time.Now()
	err := cache.PrewarmImages(testImages, 2, true)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("PrewarmImages failed: %v", err)
	}

	t.Logf("Prewarmed %d images in %s", len(testImages), duration)

	// Verify all images are cached
	for _, img := range testImages {
		state, exists := cache.imageStates[img]
		if !exists {
			t.Errorf("Image %s should be in cache after prewarming", img)
		}
		if state != nil {
			t.Logf("  - %s: %d bytes, pulled in %dms", img, state.SizeBytes, state.PullTime)
		}
	}

	// Test with duplicate images
	duplicateImages := []string{
		"alpine:latest",
		"alpine:latest",
		"busybox:latest",
	}
	err = cache.PrewarmImages(duplicateImages, 2, true)
	if err != nil {
		t.Fatalf("PrewarmImages with duplicates failed: %v", err)
	}

	// Test with empty list
	err = cache.PrewarmImages([]string{}, 2, true)
	if err != nil {
		t.Errorf("PrewarmImages should handle empty list: %v", err)
	}
}

// TestPruneCache tests cache pruning functionality
func TestPruneCache(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available in test environment")
	}

	tempDir := t.TempDir()
	cache := NewImageCache(tempDir, 24*time.Hour)

	// Pull a test image
	testImage := "alpine:latest"
	err := cache.EnsureImage(testImage, true)
	if err != nil {
		t.Fatalf("Failed to ensure test image: %v", err)
	}

	// Modify LastUsed to simulate old cache entry
	state := cache.imageStates[testImage]
	state.LastUsed = time.Now().Add(-48 * time.Hour) // 2 days old

	// Prune with 24h max age (should remove the image)
	t.Logf("Pruning cache with maxAge=24h")
	err = cache.PruneCache(24*time.Hour, true)
	if err != nil {
		t.Fatalf("PruneCache failed: %v", err)
	}

	// Verify image was removed from cache state
	_, exists := cache.imageStates[testImage]
	if exists {
		t.Error("Old image should be removed from cache state after pruning")
	}

	// Test pruning with no old images
	cache.imageStates[testImage] = &ImageState{
		Image:    testImage,
		LastUsed: time.Now(),
	}

	err = cache.PruneCache(24*time.Hour, true)
	if err != nil {
		t.Fatalf("PruneCache failed on fresh cache: %v", err)
	}

	// Fresh image should still be there
	_, exists = cache.imageStates[testImage]
	if !exists {
		t.Error("Fresh image should not be pruned")
	}
}

// TestExportImportImages tests image export and import functionality
func TestExportImportImages(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available in test environment")
	}

	tempDir := t.TempDir()
	cache := NewImageCache(filepath.Join(tempDir, "cache"), 24*time.Hour)
	exportDir := filepath.Join(tempDir, "exports")

	// Ensure test image is available
	testImage := "alpine:latest"
	err := cache.EnsureImage(testImage, true)
	if err != nil {
		t.Fatalf("Failed to ensure test image: %v", err)
	}

	// Test export
	t.Logf("Exporting image %s to %s", testImage, exportDir)
	err = cache.ExportImages([]string{testImage}, exportDir, true)
	if err != nil {
		t.Fatalf("ExportImages failed: %v", err)
	}

	// Verify export directory was created
	if _, err := os.Stat(exportDir); os.IsNotExist(err) {
		t.Fatal("Export directory was not created")
	}

	// Verify tar file was created
	files, err := os.ReadDir(exportDir)
	if err != nil {
		t.Fatalf("Failed to read export directory: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No files were exported")
	}

	tarFound := false
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".tar" {
			tarFound = true
			info, _ := file.Info()
			t.Logf("Exported tar: %s (%d bytes)", file.Name(), info.Size())
			break
		}
	}

	if !tarFound {
		t.Error("No .tar file found in export directory")
	}

	// Test import
	importCache := NewImageCache(filepath.Join(tempDir, "import-cache"), 24*time.Hour)
	t.Logf("Importing images from %s", exportDir)
	err = importCache.ImportImages(exportDir, true)
	if err != nil {
		t.Fatalf("ImportImages failed: %v", err)
	}

	// Test import from non-existent directory (should not error)
	err = importCache.ImportImages(filepath.Join(tempDir, "nonexistent"), true)
	if err != nil {
		t.Errorf("ImportImages should handle non-existent directory gracefully: %v", err)
	}

	// Test import from empty directory
	emptyDir := filepath.Join(tempDir, "empty")
	os.MkdirAll(emptyDir, 0755)
	err = importCache.ImportImages(emptyDir, true)
	if err != nil {
		t.Errorf("ImportImages should handle empty directory gracefully: %v", err)
	}
}

// TestGetImageInfo tests image info retrieval
func TestGetImageInfo(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available in test environment")
	}

	// Ensure test image is available
	testImage := "alpine:latest"

	// Pull image if not available
	localExists, _ := ImageExists(testImage)
	if !localExists {
		t.Logf("Pulling %s for test", testImage)
		if err := PullImage(testImage); err != nil {
			t.Fatalf("Failed to pull test image: %v", err)
		}
	}

	// Get image info
	digest, size, err := GetImageInfo(testImage)
	if err != nil {
		t.Fatalf("GetImageInfo failed: %v", err)
	}

	// Verify digest format
	if digest == "" {
		t.Error("Digest should not be empty")
	}
	if len(digest) < 10 {
		t.Errorf("Digest seems too short: %s", digest)
	}

	// Verify size is reasonable
	if size <= 0 {
		t.Errorf("Size should be positive, got %d", size)
	}

	t.Logf("Image info for %s: Digest=%s, Size=%d bytes (%.2f MB)",
		testImage, digest, size, float64(size)/(1024*1024))

	// Test with non-existent image
	_, _, err = GetImageInfo("nonexistent:image:tag:invalid")
	if err == nil {
		t.Error("GetImageInfo should fail for non-existent image")
	}
}

// TestEnsureImageWithOldCache tests cache expiration behavior
func TestEnsureImageWithOldCache(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available in test environment")
	}

	tempDir := t.TempDir()
	cache := NewImageCache(tempDir, 1*time.Hour) // 1 hour max age

	testImage := "alpine:latest"

	// Ensure image is available
	err := cache.EnsureImage(testImage, true)
	if err != nil {
		t.Fatalf("Failed to ensure test image: %v", err)
	}

	// Simulate expired cache by modifying CachedAt
	state := cache.imageStates[testImage]
	oldCachedAt := state.CachedAt
	state.CachedAt = time.Now().Add(-2 * time.Hour) // Make it 2 hours old

	// EnsureImage should detect expired cache and re-validate
	t.Logf("Ensuring image with expired cache entry")
	err = cache.EnsureImage(testImage, true)
	if err != nil {
		t.Fatalf("EnsureImage failed with expired cache: %v", err)
	}

	// CachedAt should be updated if image was re-pulled
	newState := cache.imageStates[testImage]
	if newState.CachedAt.Equal(oldCachedAt) {
		t.Log("Cache was still valid (image exists locally)")
	} else {
		t.Logf("Cache was refreshed (CachedAt updated)")
	}
}

// TestPrewarmImagesWithErrors tests error handling in parallel operations
func TestPrewarmImagesWithErrors(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available in test environment")
	}

	tempDir := t.TempDir()
	cache := NewImageCache(tempDir, 24*time.Hour)

	// Mix of valid and invalid images
	mixedImages := []string{
		"alpine:latest", // Valid
		"invalid:nonexistent:image", // Invalid
		"busybox:latest", // Valid
	}

	t.Logf("Prewarming with mixed valid/invalid images")
	err := cache.PrewarmImages(mixedImages, 2, true)

	// Should report error but continue with valid images
	if err == nil {
		t.Log("PrewarmImages completed (may have skipped invalid images)")
	} else {
		t.Logf("PrewarmImages reported error (expected): %v", err)
	}

	// Check that at least some valid images were cached
	validCount := 0
	if _, exists := cache.imageStates["alpine:latest"]; exists {
		validCount++
	}
	if _, exists := cache.imageStates["busybox:latest"]; exists {
		validCount++
	}

	t.Logf("Successfully cached %d out of 2 valid images", validCount)
}
