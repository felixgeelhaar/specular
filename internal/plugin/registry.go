package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DefaultRegistryURL is the default URL for the Specular plugin registry
const DefaultRegistryURL = "https://raw.githubusercontent.com/specular/specular-plugins/main/index.json"

// DefaultCacheTTL is the default time-to-live for the registry cache
const DefaultCacheTTL = 1 * time.Hour

// RegistryIndex represents the plugin registry index structure
type RegistryIndex struct {
	// Version is the schema version of the index
	Version string `json:"version"`
	// Updated is when the index was last updated
	Updated time.Time `json:"updated"`
	// Plugins is the map of plugin name to plugin entry
	Plugins map[string]RegistryPlugin `json:"plugins"`
}

// RegistryPlugin represents a plugin entry in the registry
type RegistryPlugin struct {
	// Name is the plugin name
	Name string `json:"name"`
	// Description is a short description of the plugin
	Description string `json:"description"`
	// Author is the plugin author
	Author string `json:"author"`
	// Type is the plugin type (provider, validator, formatter, hook, notifier)
	Type PluginType `json:"type"`
	// Repository is the source repository URL
	Repository string `json:"repository"`
	// Homepage is the plugin homepage URL
	Homepage string `json:"homepage,omitempty"`
	// License is the plugin license
	License string `json:"license,omitempty"`
	// Versions is a map of version to version info
	Versions map[string]RegistryVersion `json:"versions"`
	// Latest is the latest stable version
	Latest string `json:"latest"`
	// Keywords for search indexing
	Keywords []string `json:"keywords,omitempty"`
	// Downloads is the download count
	Downloads int64 `json:"downloads,omitempty"`
	// Stars is the star count
	Stars int `json:"stars,omitempty"`
	// IsDeprecated indicates if the plugin is deprecated
	Deprecated bool `json:"deprecated,omitempty"`
	// DeprecationMessage explains why the plugin is deprecated
	DeprecationMessage string `json:"deprecation_message,omitempty"`
}

// RegistryVersion represents a specific version of a plugin
type RegistryVersion struct {
	// Released is when this version was released
	Released time.Time `json:"released"`
	// MinSpecularVersion is the minimum required Specular CLI version
	MinSpecularVersion string `json:"min_specular_version,omitempty"`
	// Checksum is the SHA256 checksum of the release tarball
	Checksum string `json:"checksum,omitempty"`
	// DownloadURL is the direct download URL for this version
	DownloadURL string `json:"download_url,omitempty"`
	// Dependencies are required plugins for this version
	Dependencies []PluginDependency `json:"dependencies,omitempty"`
	// Yanked indicates if this version has been yanked
	Yanked bool `json:"yanked,omitempty"`
	// YankedReason explains why the version was yanked
	YankedReason string `json:"yanked_reason,omitempty"`
}

// RegistryCache stores cached registry data
type RegistryCache struct {
	// Index is the cached registry index
	Index *RegistryIndex `json:"index"`
	// FetchedAt is when the index was fetched
	FetchedAt time.Time `json:"fetched_at"`
	// URL is the registry URL this was fetched from
	URL string `json:"url"`
}

// Registry provides access to the plugin registry
type Registry struct {
	// URL is the registry URL
	URL string
	// CacheTTL is how long to cache the registry index
	CacheTTL time.Duration
	// cacheDir is where to store the cache file
	cacheDir string
	// httpClient is the HTTP client for fetching
	httpClient *http.Client
	// cache is the in-memory cache
	cache *RegistryCache
	// mu protects cache
	mu sync.RWMutex
}

// RegistryOption is a functional option for Registry
type RegistryOption func(*Registry)

// WithRegistryURL sets a custom registry URL
func WithRegistryURL(url string) RegistryOption {
	return func(r *Registry) {
		r.URL = url
	}
}

// WithCacheTTL sets a custom cache TTL
func WithCacheTTL(ttl time.Duration) RegistryOption {
	return func(r *Registry) {
		r.CacheTTL = ttl
	}
}

// WithCacheDir sets a custom cache directory
func WithCacheDir(dir string) RegistryOption {
	return func(r *Registry) {
		r.cacheDir = dir
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) RegistryOption {
	return func(r *Registry) {
		r.httpClient = client
	}
}

// NewRegistry creates a new registry client
func NewRegistry(opts ...RegistryOption) *Registry {
	r := &Registry{
		URL:      DefaultRegistryURL,
		CacheTTL: DefaultCacheTTL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(r)
	}

	// Set default cache directory if not specified
	if r.cacheDir == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			r.cacheDir = filepath.Join(home, ".specular", "cache")
		}
	}

	return r
}

// cacheFilePath returns the path to the cache file
func (r *Registry) cacheFilePath() string {
	return filepath.Join(r.cacheDir, "registry.json")
}

// loadCache loads the registry cache from disk
func (r *Registry) loadCache() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.cacheFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache file is OK
		}
		return fmt.Errorf("failed to read cache: %w", err)
	}

	var cache RegistryCache
	if err := json.Unmarshal(data, &cache); err != nil {
		// Invalid cache, ignore it
		return nil
	}

	// Only use cache if URL matches
	if cache.URL != r.URL {
		return nil
	}

	r.cache = &cache
	return nil
}

// saveCache saves the registry cache to disk
func (r *Registry) saveCache() error {
	r.mu.RLock()
	cache := r.cache
	r.mu.RUnlock()

	if cache == nil {
		return nil
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(r.cacheDir, 0750); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Write atomically
	tmpFile := r.cacheFilePath() + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	if err := os.Rename(tmpFile, r.cacheFilePath()); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	return nil
}

// isCacheValid checks if the cache is still valid
func (r *Registry) isCacheValid() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.cache == nil || r.cache.Index == nil {
		return false
	}

	return time.Since(r.cache.FetchedAt) < r.CacheTTL
}

// Fetch retrieves the registry index from the remote URL
func (r *Registry) Fetch() error {
	// Try to load disk cache first
	if r.cache == nil {
		_ = r.loadCache()
	}

	// Return cached version if valid
	if r.isCacheValid() {
		return nil
	}

	// Fetch from remote
	req, err := http.NewRequest("GET", r.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "specular-cli/1.0")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		// If we have a stale cache, use it
		r.mu.RLock()
		hasCache := r.cache != nil && r.cache.Index != nil
		r.mu.RUnlock()
		if hasCache {
			return nil
		}
		return fmt.Errorf("failed to fetch registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// If we have a stale cache, use it
		r.mu.RLock()
		hasCache := r.cache != nil && r.cache.Index != nil
		r.mu.RUnlock()
		if hasCache {
			return nil
		}
		return fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var index RegistryIndex
	if err := json.Unmarshal(body, &index); err != nil {
		return fmt.Errorf("failed to parse registry index: %w", err)
	}

	// Update cache
	r.mu.Lock()
	r.cache = &RegistryCache{
		Index:     &index,
		FetchedAt: time.Now(),
		URL:       r.URL,
	}
	r.mu.Unlock()

	// Save to disk
	_ = r.saveCache()

	return nil
}

// GetIndex returns the registry index, fetching if necessary
func (r *Registry) GetIndex() (*RegistryIndex, error) {
	if err := r.Fetch(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.cache == nil || r.cache.Index == nil {
		return nil, fmt.Errorf("registry index not available")
	}

	return r.cache.Index, nil
}

// Get retrieves a specific plugin from the registry
func (r *Registry) Get(name string) (*RegistryPlugin, error) {
	index, err := r.GetIndex()
	if err != nil {
		return nil, err
	}

	plugin, ok := index.Plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	return &plugin, nil
}

// GetVersion retrieves a specific version of a plugin
func (r *Registry) GetVersion(name, version string) (*RegistryPlugin, *RegistryVersion, error) {
	plugin, err := r.Get(name)
	if err != nil {
		return nil, nil, err
	}

	// If version is empty, use latest
	if version == "" {
		version = plugin.Latest
	}

	v, ok := plugin.Versions[version]
	if !ok {
		return nil, nil, fmt.Errorf("version %s not found for plugin %s", version, name)
	}

	return plugin, &v, nil
}

// SearchResult represents a search result
type SearchResult struct {
	Plugin    *RegistryPlugin
	Score     float64
	MatchType string // "name", "keyword", "description"
}

// calculatePluginScore calculates the search score and match type for a plugin
func calculatePluginScore(name string, p *RegistryPlugin, query string) (float64, string) {
	var score float64
	var matchType string

	// Check name match (highest priority)
	if strings.Contains(strings.ToLower(name), query) {
		score = 100
		if strings.EqualFold(name, query) {
			score = 200 // Exact match
		}
		matchType = "name"
	}

	// Check type match
	if score == 0 && strings.Contains(strings.ToLower(string(p.Type)), query) {
		score = 80
		matchType = "type"
	}

	// Check keyword match
	if score == 0 {
		for _, kw := range p.Keywords {
			if strings.Contains(strings.ToLower(kw), query) {
				score = 60
				matchType = "keyword"
				break
			}
		}
	}

	// Check description match
	if score == 0 && strings.Contains(strings.ToLower(p.Description), query) {
		score = 40
		matchType = "description"
	}

	// Check author match
	if score == 0 && strings.Contains(strings.ToLower(p.Author), query) {
		score = 30
		matchType = "author"
	}

	return score, matchType
}

// Search searches the registry for plugins matching the query
func (r *Registry) Search(query string) ([]SearchResult, error) {
	index, err := r.GetIndex()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		// Return all plugins sorted by downloads
		var results []SearchResult
		for name := range index.Plugins {
			p := index.Plugins[name]
			results = append(results, SearchResult{
				Plugin:    &p,
				Score:     1.0,
				MatchType: "all",
			})
		}
		sort.Slice(results, func(i, j int) bool {
			return results[i].Plugin.Downloads > results[j].Plugin.Downloads
		})
		return results, nil
	}

	var results []SearchResult

	for name := range index.Plugins {
		p := index.Plugins[name]
		score, matchType := calculatePluginScore(name, &p, query)

		if score > 0 {
			// Boost score based on downloads and stars
			score += float64(p.Downloads) / 10000
			score += float64(p.Stars) / 100

			// Penalize deprecated plugins
			if p.Deprecated {
				score *= 0.5
			}

			results = append(results, SearchResult{
				Plugin:    &p,
				Score:     score,
				MatchType: matchType,
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// SearchByType returns all plugins of a specific type
func (r *Registry) SearchByType(pluginType PluginType) ([]SearchResult, error) {
	index, err := r.GetIndex()
	if err != nil {
		return nil, err
	}

	var results []SearchResult

	for name := range index.Plugins {
		p := index.Plugins[name]
		if p.Type == pluginType {
			results = append(results, SearchResult{
				Plugin:    &p,
				Score:     float64(p.Downloads),
				MatchType: "type",
			})
		}
	}

	// Sort by downloads descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Plugin.Downloads > results[j].Plugin.Downloads
	})

	return results, nil
}

// ListAll returns all plugins in the registry
func (r *Registry) ListAll() ([]RegistryPlugin, error) {
	index, err := r.GetIndex()
	if err != nil {
		return nil, err
	}

	var plugins []RegistryPlugin
	for name := range index.Plugins {
		plugins = append(plugins, index.Plugins[name])
	}

	// Sort by name
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})

	return plugins, nil
}

// ResolveVersion resolves a version constraint to a specific version
func (r *Registry) ResolveVersion(name string, constraint string) (string, error) {
	plugin, err := r.Get(name)
	if err != nil {
		return "", err
	}

	// If no constraint, return latest
	if constraint == "" {
		return plugin.Latest, nil
	}

	// Parse constraint
	vc, err := ParseConstraint(constraint)
	if err != nil {
		// Try as exact version
		if _, ok := plugin.Versions[constraint]; ok {
			return constraint, nil
		}
		return "", fmt.Errorf("invalid version constraint: %s", constraint)
	}

	// Find best matching version
	var versions []*PluginVersion
	for v := range plugin.Versions {
		pv, err := ParseVersion(v)
		if err != nil {
			continue
		}
		vInfo := plugin.Versions[v]
		// Skip yanked versions
		if vInfo.Yanked {
			continue
		}
		if pv.Satisfies(vc) {
			versions = append(versions, pv)
		}
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no version of %s satisfies constraint %s", name, constraint)
	}

	// Sort versions descending and return highest
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Compare(versions[j]) > 0
	})

	return versions[0].String(), nil
}

// GetDownloadURL returns the download URL for a specific version
func (r *Registry) GetDownloadURL(name, version string) (string, error) {
	plugin, v, err := r.GetVersion(name, version)
	if err != nil {
		return "", err
	}

	if v.DownloadURL != "" {
		return v.DownloadURL, nil
	}

	// Construct URL from repository
	if plugin.Repository != "" {
		// Parse as GitHub source
		source, err := ParseSource(plugin.Repository)
		if err == nil && source.Type == SourceTypeGitHub {
			source.Version = version
			return source.GitHubTagTarballURL(), nil
		}
	}

	return "", fmt.Errorf("no download URL available for %s@%s", name, version)
}

// ClearCache clears the registry cache
func (r *Registry) ClearCache() error {
	r.mu.Lock()
	r.cache = nil
	r.mu.Unlock()

	return os.Remove(r.cacheFilePath())
}

// GetCacheAge returns how old the cache is
func (r *Registry) GetCacheAge() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.cache == nil {
		return 0
	}

	return time.Since(r.cache.FetchedAt)
}

// IsCached returns true if the registry is cached
func (r *Registry) IsCached() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.cache != nil && r.cache.Index != nil
}

// PluginInfo returns detailed information about a plugin formatted for display
type PluginInfo struct {
	Name        string
	Description string
	Author      string
	Type        PluginType
	Repository  string
	Homepage    string
	License     string
	Latest      string
	Downloads   int64
	Stars       int
	Deprecated  bool
	Versions    []string
	Keywords    []string
}

// GetPluginInfo returns detailed information about a plugin
func (r *Registry) GetPluginInfo(name string) (*PluginInfo, error) {
	plugin, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	// Get sorted version list
	var versions []string
	for v := range plugin.Versions {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool {
		v1, _ := ParseVersion(versions[i])
		v2, _ := ParseVersion(versions[j])
		if v1 == nil || v2 == nil {
			return versions[i] > versions[j]
		}
		return v1.Compare(v2) > 0
	})

	return &PluginInfo{
		Name:        plugin.Name,
		Description: plugin.Description,
		Author:      plugin.Author,
		Type:        plugin.Type,
		Repository:  plugin.Repository,
		Homepage:    plugin.Homepage,
		License:     plugin.License,
		Latest:      plugin.Latest,
		Downloads:   plugin.Downloads,
		Stars:       plugin.Stars,
		Deprecated:  plugin.Deprecated,
		Versions:    versions,
		Keywords:    plugin.Keywords,
	}, nil
}

// CheckCompatibility checks if a plugin version is compatible with the current CLI version
func (r *Registry) CheckCompatibility(name, version, cliVersion string) error {
	_, v, err := r.GetVersion(name, version)
	if err != nil {
		return err
	}

	if v.MinSpecularVersion == "" {
		return nil // No minimum version required
	}

	cliV, err := ParseVersion(cliVersion)
	if err != nil {
		return fmt.Errorf("invalid CLI version: %w", err)
	}

	minV, err := ParseVersion(v.MinSpecularVersion)
	if err != nil {
		return fmt.Errorf("invalid minimum version: %w", err)
	}

	if cliV.Compare(minV) < 0 {
		return fmt.Errorf("plugin %s@%s requires Specular CLI %s or higher (current: %s)",
			name, version, v.MinSpecularVersion, cliVersion)
	}

	return nil
}
