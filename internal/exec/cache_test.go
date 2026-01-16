package exec

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestImageCache_SaveLoadManifest(t *testing.T) {
	dir := t.TempDir()
	cache := NewImageCache(dir, time.Hour)
	now := time.Now().UTC()

	cache.imageStates["busybox:latest"] = &ImageState{
		Image:     "busybox:latest",
		Digest:    "sha256:123",
		CachedAt:  now,
		LastUsed:  now,
		PullTime:  42,
		SizeBytes: 1024,
	}

	if err := cache.SaveManifest(); err != nil {
		t.Fatalf("SaveManifest failed: %v", err)
	}

	other := NewImageCache(dir, time.Hour)
	if err := other.LoadManifest(); err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if got := len(other.imageStates); got != 1 {
		t.Fatalf("expected 1 cached image, got %d", got)
	}

	state, ok := other.imageStates["busybox:latest"]
	if !ok {
		t.Fatalf("cached state missing for busybox:latest")
	}
	if state.Digest != "sha256:123" {
		t.Errorf("expected digest sha256:123, got %s", state.Digest)
	}

	stats := cache.GetStats()
	if total, ok := stats["total_images"].(int); !ok || total != 1 {
		t.Errorf("GetStats returned unexpected total_images: %v", stats["total_images"])
	}
}

func TestImageCache_LoadManifestInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	cache := NewImageCache(dir, time.Hour)
	if err := cache.LoadManifest(); err == nil {
		t.Fatal("expected error loading invalid manifest, got nil")
	}
}

func TestGenerateCacheKeyDeterministic(t *testing.T) {
	key1 := generateCacheKey("registry.example.com/my/image:tag")
	key2 := generateCacheKey("registry.example.com/my/image:tag")
	if key1 != key2 {
		t.Fatalf("expected deterministic cache keys, got %s and %s", key1, key2)
	}
	if !strings.Contains(key1, "registry.example.com-my-image-tag") {
		t.Errorf("unexpected cache key format: %s", key1)
	}
}

func TestGetRequiredImagesDeduplicated(t *testing.T) {
	tasks := []struct{ Skill string }{
		{Skill: "go-backend"},
		{Skill: "infra"},
		{Skill: "testing"},
	}
	expected := []string{"golang:1.22", "alpine:latest"}
	images := GetRequiredImages(tasks)
	if len(images) != len(expected) {
		t.Fatalf("expected %d images, got %d", len(expected), len(images))
	}

	imageSet := make(map[string]struct{})
	for _, img := range images {
		imageSet[img] = struct{}{}
	}
	for _, want := range expected {
		if _, ok := imageSet[want]; !ok {
			t.Errorf("expected image %s in results", want)
		}
	}
	if reflect.DeepEqual(images, expected) {
		t.Logf("image order matches expected, ok")
	}
}
