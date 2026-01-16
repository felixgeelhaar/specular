// Package cache provides cache management for Specular CLI
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// CacheType represents different types of cached content
type CacheType string

const (
	// CacheTypeModels represents cached AI model artifacts
	CacheTypeModels CacheType = "models"
	// CacheTypeBundles represents cached build bundles
	CacheTypeBundles CacheType = "bundles"
	// CacheTypeTraces represents cached execution traces
	CacheTypeTraces CacheType = "traces"
	// CacheTypeAll represents all cache types
	CacheTypeAll CacheType = "all"
)

// CacheEntry represents a single cached item
type CacheEntry struct {
	Path       string    `json:"path"`
	Type       CacheType `json:"type"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
	AccessedAt time.Time `json:"accessed_at"`
	Name       string    `json:"name"`
}

// CacheInfo provides summary information about the cache
type CacheInfo struct {
	TotalSize  int64        `json:"total_size"`
	EntryCount int          `json:"entry_count"`
	Location   string       `json:"location"`
	Types      []CacheStats `json:"types"`
}

// CacheStats provides statistics for a specific cache type
type CacheStats struct {
	Type       CacheType `json:"type"`
	Size       int64     `json:"size"`
	EntryCount int       `json:"entry_count"`
}

// Manager handles cache operations
type Manager struct {
	cacheDir string
	maxSize  int64 // Maximum cache size in bytes (0 = unlimited)
}

// ManagerOption configures the cache manager
type ManagerOption func(*Manager)

// WithMaxSize sets the maximum cache size
func WithMaxSize(size int64) ManagerOption {
	return func(m *Manager) {
		m.maxSize = size
	}
}

// WithCacheDir sets a custom cache directory
func WithCacheDir(dir string) ManagerOption {
	return func(m *Manager) {
		m.cacheDir = dir
	}
}

// NewManager creates a new cache manager
func NewManager(opts ...ManagerOption) (*Manager, error) {
	// Default cache directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	m := &Manager{
		cacheDir: filepath.Join(homeDir, ".specular", "cache"),
		maxSize:  0, // unlimited by default
	}

	for _, opt := range opts {
		opt(m)
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(m.cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return m, nil
}

// GetInfo returns cache statistics
func (m *Manager) GetInfo() (*CacheInfo, error) {
	info := &CacheInfo{
		Location: m.cacheDir,
		Types:    []CacheStats{},
	}

	// Calculate stats for each type
	types := []CacheType{CacheTypeModels, CacheTypeBundles, CacheTypeTraces}
	for _, t := range types {
		stats, err := m.getTypeStats(t)
		if err != nil {
			// Skip if directory doesn't exist
			continue
		}
		info.Types = append(info.Types, *stats)
		info.TotalSize += stats.Size
		info.EntryCount += stats.EntryCount
	}

	return info, nil
}

// getTypeStats calculates stats for a specific cache type
func (m *Manager) getTypeStats(t CacheType) (*CacheStats, error) {
	typeDir := filepath.Join(m.cacheDir, string(t))

	stats := &CacheStats{
		Type: t,
	}

	err := filepath.Walk(typeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			stats.Size += info.Size()
			stats.EntryCount++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// List returns all cache entries, optionally filtered by type
func (m *Manager) List(cacheType CacheType) ([]CacheEntry, error) {
	var entries []CacheEntry

	types := []CacheType{cacheType}
	if cacheType == CacheTypeAll {
		types = []CacheType{CacheTypeModels, CacheTypeBundles, CacheTypeTraces}
	}

	for _, t := range types {
		typeDir := filepath.Join(m.cacheDir, string(t))

		err := filepath.Walk(typeDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if !info.IsDir() {
				entry := CacheEntry{
					Path:       path,
					Type:       t,
					Size:       info.Size(),
					CreatedAt:  info.ModTime(),
					AccessedAt: info.ModTime(), // Use ModTime as proxy
					Name:       filepath.Base(path),
				}
				entries = append(entries, entry)
			}
			return nil
		})
		if err != nil {
			continue // Skip errors
		}
	}

	// Sort by size (largest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Size > entries[j].Size
	})

	return entries, nil
}

// Clear removes cached items
func (m *Manager) Clear(cacheType CacheType) (*ClearResult, error) {
	result := &ClearResult{}

	types := []CacheType{cacheType}
	if cacheType == CacheTypeAll {
		types = []CacheType{CacheTypeModels, CacheTypeBundles, CacheTypeTraces}
	}

	for _, t := range types {
		typeDir := filepath.Join(m.cacheDir, string(t))

		// Get size before deletion
		entries, _ := m.List(t)
		for _, e := range entries {
			result.BytesCleared += e.Size
			result.FilesCleared++
		}

		// Remove directory contents
		if err := os.RemoveAll(typeDir); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to clear %s: %v", t, err))
		}

		// Recreate empty directory
		os.MkdirAll(typeDir, 0755)
	}

	return result, nil
}

// ClearResult contains the results of a clear operation
type ClearResult struct {
	BytesCleared int64    `json:"bytes_cleared"`
	FilesCleared int      `json:"files_cleared"`
	Errors       []string `json:"errors,omitempty"`
}

// Prune removes old cache entries to stay within size limits
func (m *Manager) Prune(maxAge time.Duration) (*PruneResult, error) {
	result := &PruneResult{}
	cutoff := time.Now().Add(-maxAge)

	entries, err := m.List(CacheTypeAll)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.CreatedAt.Before(cutoff) {
			if err := os.Remove(entry.Path); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to remove %s: %v", entry.Path, err))
				continue
			}
			result.BytesPruned += entry.Size
			result.FilesPruned++
		}
	}

	return result, nil
}

// PruneResult contains the results of a prune operation
type PruneResult struct {
	BytesPruned int64    `json:"bytes_pruned"`
	FilesPruned int      `json:"files_pruned"`
	Errors      []string `json:"errors,omitempty"`
}

// GetCacheDir returns the cache directory
func (m *Manager) GetCacheDir() string {
	return m.cacheDir
}

// GetMaxSize returns the maximum cache size
func (m *Manager) GetMaxSize() int64 {
	return m.maxSize
}

// SetMaxSize sets the maximum cache size
func (m *Manager) SetMaxSize(size int64) {
	m.maxSize = size
}

// LoadConfig loads cache configuration from a JSON file
func LoadConfig(path string) (*CacheConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &CacheConfig{}, nil
		}
		return nil, err
	}

	var config CacheConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig saves cache configuration to a JSON file
func SaveConfig(path string, config *CacheConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	MaxSize  int64  `json:"max_size"`
	Location string `json:"location,omitempty"`
}

// FormatSize formats a size in bytes to human-readable format
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
