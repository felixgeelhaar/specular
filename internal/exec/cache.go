package exec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ImageCache manages Docker image caching for faster execution
type ImageCache struct {
	CacheDir    string
	MaxAge      time.Duration
	mu          sync.Mutex
	imageStates map[string]*ImageState
}

// ImageState tracks the state of a cached image
type ImageState struct {
	Image     string    `json:"image"`
	Digest    string    `json:"digest"`
	CachedAt  time.Time `json:"cached_at"`
	LastUsed  time.Time `json:"last_used"`
	PullTime  int64     `json:"pull_time_ms"`
	SizeBytes int64     `json:"size_bytes"`
}

// CacheManifest stores metadata about cached images
type CacheManifest struct {
	Version   string                 `json:"version"`
	Images    map[string]*ImageState `json:"images"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// NewImageCache creates a new image cache manager
func NewImageCache(cacheDir string, maxAge time.Duration) *ImageCache {
	return &ImageCache{
		CacheDir:    cacheDir,
		MaxAge:      maxAge,
		imageStates: make(map[string]*ImageState),
	}
}

// LoadManifest loads the cache manifest from disk
func (c *ImageCache) LoadManifest() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	manifestPath := filepath.Join(c.CacheDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No manifest exists yet, start fresh
			c.imageStates = make(map[string]*ImageState)
			return nil
		}
		return fmt.Errorf("read manifest: %w", err)
	}

	var manifest CacheManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("unmarshal manifest: %w", err)
	}

	c.imageStates = manifest.Images
	return nil
}

// SaveManifest saves the cache manifest to disk
func (c *ImageCache) SaveManifest() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := os.MkdirAll(c.CacheDir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	manifest := CacheManifest{
		Version:   "1.0",
		Images:    c.imageStates,
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	manifestPath := filepath.Join(c.CacheDir, "manifest.json")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

// EnsureImage ensures an image is available, using cache when possible
func (c *ImageCache) EnsureImage(image string, verbose bool) error {
	c.mu.Lock()
	state, exists := c.imageStates[image]
	c.mu.Unlock()

	// Check if image exists locally
	localExists, err := ImageExists(image)
	if err != nil {
		return fmt.Errorf("check image exists: %w", err)
	}

	// If image exists locally and is cached, use it
	if localExists {
		if exists && time.Since(state.CachedAt) < c.MaxAge {
			if verbose {
				fmt.Printf("  ✓ Using cached image: %s (age: %s)\n",
					image, time.Since(state.CachedAt).Round(time.Second))
			}

			// Update last used timestamp
			c.mu.Lock()
			state.LastUsed = time.Now()
			c.mu.Unlock()
			c.SaveManifest()

			return nil
		}
	}

	// Pull the image
	if verbose {
		fmt.Printf("  ⬇ Pulling image: %s...\n", image)
	}

	startTime := time.Now()
	if err := PullImage(image); err != nil {
		return fmt.Errorf("pull image: %w", err)
	}
	pullDuration := time.Since(startTime)

	// Get image digest and size
	digest, size, err := GetImageInfo(image)
	if err != nil {
		// Don't fail on metadata errors, just log warning
		if verbose {
			fmt.Printf("  ⚠ Warning: could not get image info: %v\n", err)
		}
	}

	// Update cache state
	c.mu.Lock()
	c.imageStates[image] = &ImageState{
		Image:     image,
		Digest:    digest,
		CachedAt:  time.Now(),
		LastUsed:  time.Now(),
		PullTime:  pullDuration.Milliseconds(),
		SizeBytes: size,
	}
	c.mu.Unlock()

	// Save manifest
	if err := c.SaveManifest(); err != nil {
		if verbose {
			fmt.Printf("  ⚠ Warning: failed to save manifest: %v\n", err)
		}
	}

	if verbose {
		fmt.Printf("  ✓ Pulled %s in %s (%.2f MB)\n",
			image, pullDuration.Round(time.Millisecond), float64(size)/(1024*1024))
	}

	return nil
}

// PrewarmImages pulls multiple images in parallel
func (c *ImageCache) PrewarmImages(images []string, concurrency int, verbose bool) error {
	if len(images) == 0 {
		return nil
	}

	// Remove duplicates
	uniqueImages := make([]string, 0, len(images))
	seen := make(map[string]bool)
	for _, img := range images {
		if !seen[img] {
			uniqueImages = append(uniqueImages, img)
			seen[img] = true
		}
	}

	if verbose {
		fmt.Printf("Pre-warming %d Docker images (concurrency: %d)...\n",
			len(uniqueImages), concurrency)
	}

	// Create worker pool
	imageChan := make(chan string, len(uniqueImages))
	errChan := make(chan error, len(uniqueImages))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for image := range imageChan {
				if err := c.EnsureImage(image, verbose); err != nil {
					errChan <- fmt.Errorf("%s: %w", image, err)
				}
			}
		}()
	}

	// Queue images
	for _, image := range uniqueImages {
		imageChan <- image
	}
	close(imageChan)

	// Wait for completion
	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to prewarm %d images: %v", len(errors), errors[0])
	}

	if verbose {
		fmt.Printf("✓ Successfully pre-warmed %d images\n", len(uniqueImages))
	}

	return nil
}

// PruneCache removes old or unused cached images
func (c *ImageCache) PruneCache(maxAge time.Duration, verbose bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if verbose {
		fmt.Println("Pruning old cached images...")
	}

	pruned := 0
	for image, state := range c.imageStates {
		age := time.Since(state.LastUsed)
		if age > maxAge {
			if verbose {
				fmt.Printf("  🗑 Removing %s (unused for %s)\n",
					image, age.Round(time.Hour))
			}

			// Remove from Docker daemon
			cmd := exec.Command("docker", "rmi", image)
			if err := cmd.Run(); err != nil {
				if verbose {
					fmt.Printf("  ⚠ Failed to remove %s: %v\n", image, err)
				}
			}

			// Remove from cache state
			delete(c.imageStates, image)
			pruned++
		}
	}

	if verbose && pruned > 0 {
		fmt.Printf("✓ Pruned %d images\n", pruned)
	} else if verbose {
		fmt.Println("✓ No images to prune")
	}

	return c.SaveManifest()
}

// ExportImages exports cached images to tar files for CI/CD caching
func (c *ImageCache) ExportImages(images []string, outputDir string, verbose bool) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	if verbose {
		fmt.Printf("Exporting %d images to %s...\n", len(images), outputDir)
	}

	for _, image := range images {
		// Generate cache key from image name
		cacheKey := generateCacheKey(image)
		tarPath := filepath.Join(outputDir, cacheKey+".tar")

		if verbose {
			fmt.Printf("  💾 Exporting %s...\n", image)
		}

		// Export to tar
		cmd := exec.Command("docker", "save", "-o", tarPath, image)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("export %s: %s", image, stderr.String())
		}

		// Get file size
		stat, _ := os.Stat(tarPath)
		if verbose && stat != nil {
			fmt.Printf("  ✓ Exported %s (%.2f MB)\n",
				image, float64(stat.Size())/(1024*1024))
		}
	}

	if verbose {
		fmt.Printf("✓ Exported %d images\n", len(images))
	}

	return nil
}

// ImportImages imports cached images from tar files
func (c *ImageCache) ImportImages(inputDir string, verbose bool) error {
	files, err := os.ReadDir(inputDir)
	if err != nil {
		if os.IsNotExist(err) {
			if verbose {
				fmt.Println("No cache directory found, skipping import")
			}
			return nil
		}
		return fmt.Errorf("read cache dir: %w", err)
	}

	tarFiles := make([]string, 0)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".tar") {
			tarFiles = append(tarFiles, filepath.Join(inputDir, file.Name()))
		}
	}

	if len(tarFiles) == 0 {
		if verbose {
			fmt.Println("No cached images found")
		}
		return nil
	}

	if verbose {
		fmt.Printf("Importing %d cached images...\n", len(tarFiles))
	}

	for _, tarPath := range tarFiles {
		if verbose {
			fmt.Printf("  📦 Importing %s...\n", filepath.Base(tarPath))
		}

		cmd := exec.Command("docker", "load", "-i", tarPath)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			if verbose {
				fmt.Printf("  ⚠ Failed to import %s: %s\n",
					filepath.Base(tarPath), stderr.String())
			}
			continue
		}

		// Extract image name from docker load output
		output := stdout.String()
		if verbose && strings.Contains(output, "Loaded image") {
			fmt.Printf("  ✓ %s\n", strings.TrimSpace(output))
		}
	}

	if verbose {
		fmt.Printf("✓ Import complete\n")
	}

	return nil
}

// GetStats returns cache statistics
func (c *ImageCache) GetStats() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	totalSize := int64(0)
	oldestCache := time.Now()
	newestCache := time.Time{}

	for _, state := range c.imageStates {
		totalSize += state.SizeBytes
		if state.CachedAt.Before(oldestCache) {
			oldestCache = state.CachedAt
		}
		if state.CachedAt.After(newestCache) {
			newestCache = state.CachedAt
		}
	}

	return map[string]interface{}{
		"total_images":  len(c.imageStates),
		"total_size_mb": float64(totalSize) / (1024 * 1024),
		"oldest_cache":  oldestCache,
		"newest_cache":  newestCache,
		"cache_dir":     c.CacheDir,
	}
}

// GetImageInfo retrieves digest and size for an image
func GetImageInfo(image string) (digest string, size int64, err error) {
	cmd := exec.Command("docker", "image", "inspect", image,
		"--format", "{{.Id}}|{{.Size}}")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", 0, fmt.Errorf("inspect image: %s", stderr.String())
	}

	parts := strings.Split(strings.TrimSpace(stdout.String()), "|")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("unexpected inspect output format")
	}

	digest = parts[0]
	fmt.Sscanf(parts[1], "%d", &size)

	return digest, size, nil
}

// generateCacheKey creates a deterministic cache key from image name
func generateCacheKey(image string) string {
	// Normalize image name (replace special chars with dashes)
	normalized := strings.ReplaceAll(image, "/", "-")
	normalized = strings.ReplaceAll(normalized, ":", "-")

	// Add hash for uniqueness
	hash := sha256.Sum256([]byte(image))
	hashStr := hex.EncodeToString(hash[:])[:8]

	return fmt.Sprintf("image-%s-%s", normalized, hashStr)
}

// GetRequiredImages extracts all Docker images needed for a plan
func GetRequiredImages(tasks []struct{ Skill string }) []string {
	imageMap := make(map[string]bool)

	for _, task := range tasks {
		var image string
		switch task.Skill {
		case "go-backend":
			image = "golang:1.22"
		case "ui-react":
			image = "node:20"
		case "infra":
			image = "alpine:latest"
		case "database":
			image = "postgres:15"
		case "testing":
			image = "golang:1.22"
		default:
			image = "alpine:latest"
		}
		imageMap[image] = true
	}

	// Convert to sorted slice for consistency
	images := make([]string, 0, len(imageMap))
	for img := range imageMap {
		images = append(images, img)
	}
	sort.Strings(images)

	return images
}
