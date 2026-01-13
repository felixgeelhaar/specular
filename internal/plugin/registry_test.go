package plugin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createTestIndex creates a sample registry index for testing
func createTestIndex() *RegistryIndex {
	return &RegistryIndex{
		Version: "1.0.0",
		Updated: time.Now(),
		Plugins: map[string]RegistryPlugin{
			"slack-notifier": {
				Name:        "slack-notifier",
				Description: "Send notifications to Slack channels",
				Author:      "Specular Team",
				Type:        PluginTypeNotifier,
				Repository:  "github.com/specular/slack-notifier",
				Homepage:    "https://github.com/specular/slack-notifier",
				License:     "MIT",
				Latest:      "1.2.0",
				Keywords:    []string{"slack", "notifications", "messaging"},
				Downloads:   10000,
				Stars:       150,
				Versions: map[string]RegistryVersion{
					"1.0.0": {
						Released:           time.Now().Add(-90 * 24 * time.Hour),
						MinSpecularVersion: "1.5.0",
						Checksum:           "abc123",
					},
					"1.1.0": {
						Released:           time.Now().Add(-30 * 24 * time.Hour),
						MinSpecularVersion: "1.5.0",
						Checksum:           "def456",
					},
					"1.2.0": {
						Released:           time.Now().Add(-7 * 24 * time.Hour),
						MinSpecularVersion: "1.6.0",
						Checksum:           "ghi789",
					},
				},
			},
			"discord-notifier": {
				Name:        "discord-notifier",
				Description: "Send notifications to Discord servers",
				Author:      "Community",
				Type:        PluginTypeNotifier,
				Repository:  "github.com/community/discord-notifier",
				License:     "Apache-2.0",
				Latest:      "2.0.0",
				Keywords:    []string{"discord", "notifications", "gaming"},
				Downloads:   5000,
				Stars:       80,
				Versions: map[string]RegistryVersion{
					"1.0.0": {
						Released: time.Now().Add(-60 * 24 * time.Hour),
						Checksum: "123abc",
					},
					"2.0.0": {
						Released:           time.Now().Add(-14 * 24 * time.Hour),
						MinSpecularVersion: "1.6.0",
						Checksum:           "456def",
					},
				},
			},
			"security-validator": {
				Name:        "security-validator",
				Description: "Validate security best practices in code",
				Author:      "Security Team",
				Type:        PluginTypeValidator,
				Repository:  "github.com/specular/security-validator",
				License:     "MIT",
				Latest:      "0.9.0",
				Keywords:    []string{"security", "validation", "owasp"},
				Downloads:   8000,
				Stars:       120,
				Versions: map[string]RegistryVersion{
					"0.9.0": {
						Released: time.Now().Add(-21 * 24 * time.Hour),
						Checksum: "sec789",
					},
				},
			},
			"old-plugin": {
				Name:               "old-plugin",
				Description:        "An old deprecated plugin",
				Author:             "Legacy Team",
				Type:               PluginTypeHook,
				Repository:         "github.com/legacy/old-plugin",
				Latest:             "1.0.0",
				Deprecated:         true,
				DeprecationMessage: "Use new-plugin instead",
				Downloads:          1000,
				Stars:              10,
				Versions: map[string]RegistryVersion{
					"1.0.0": {
						Released: time.Now().Add(-365 * 24 * time.Hour),
						Checksum: "old123",
					},
				},
			},
			"yanked-plugin": {
				Name:        "yanked-plugin",
				Description: "A plugin with yanked versions",
				Author:      "Test Author",
				Type:        PluginTypeFormatter,
				Repository:  "github.com/test/yanked-plugin",
				Latest:      "1.0.0",
				Downloads:   500,
				Versions: map[string]RegistryVersion{
					"1.0.0": {
						Released: time.Now().Add(-30 * 24 * time.Hour),
						Checksum: "yank100",
					},
					"1.1.0": {
						Released:     time.Now().Add(-7 * 24 * time.Hour),
						Checksum:     "yank110",
						Yanked:       true,
						YankedReason: "Critical security vulnerability",
					},
				},
			},
		},
	}
}

// createTestServer creates a mock HTTP server serving the test index
func createTestServer(index *RegistryIndex) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	}))
}

func TestRegistry_NewRegistry(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		r := NewRegistry()
		if r.URL != DefaultRegistryURL {
			t.Errorf("URL = %s, want %s", r.URL, DefaultRegistryURL)
		}
		if r.CacheTTL != DefaultCacheTTL {
			t.Errorf("CacheTTL = %v, want %v", r.CacheTTL, DefaultCacheTTL)
		}
	})

	t.Run("with custom URL", func(t *testing.T) {
		customURL := "https://custom.registry.com/index.json"
		r := NewRegistry(WithRegistryURL(customURL))
		if r.URL != customURL {
			t.Errorf("URL = %s, want %s", r.URL, customURL)
		}
	})

	t.Run("with custom TTL", func(t *testing.T) {
		customTTL := 30 * time.Minute
		r := NewRegistry(WithCacheTTL(customTTL))
		if r.CacheTTL != customTTL {
			t.Errorf("CacheTTL = %v, want %v", r.CacheTTL, customTTL)
		}
	})

	t.Run("with custom cache dir", func(t *testing.T) {
		customDir := "/tmp/test-cache"
		r := NewRegistry(WithCacheDir(customDir))
		if r.cacheDir != customDir {
			t.Errorf("cacheDir = %s, want %s", r.cacheDir, customDir)
		}
	})
}

func TestRegistry_Fetch(t *testing.T) {
	index := createTestIndex()
	server := createTestServer(index)
	defer server.Close()

	tmpDir := t.TempDir()
	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(tmpDir),
	)

	t.Run("successful fetch", func(t *testing.T) {
		err := r.Fetch()
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		if !r.IsCached() {
			t.Error("expected registry to be cached after fetch")
		}
	})

	t.Run("uses cache on subsequent fetch", func(t *testing.T) {
		// First fetch
		err := r.Fetch()
		if err != nil {
			t.Fatalf("First Fetch() error = %v", err)
		}

		// Second fetch should use cache
		err = r.Fetch()
		if err != nil {
			t.Fatalf("Second Fetch() error = %v", err)
		}
	})

	t.Run("cache file is created", func(t *testing.T) {
		cacheFile := filepath.Join(tmpDir, "registry.json")
		if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
			t.Error("cache file was not created")
		}
	})
}

func TestRegistry_Get(t *testing.T) {
	index := createTestIndex()
	server := createTestServer(index)
	defer server.Close()

	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(t.TempDir()),
	)

	t.Run("get existing plugin", func(t *testing.T) {
		plugin, err := r.Get("slack-notifier")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if plugin.Name != "slack-notifier" {
			t.Errorf("Name = %s, want slack-notifier", plugin.Name)
		}
		if plugin.Latest != "1.2.0" {
			t.Errorf("Latest = %s, want 1.2.0", plugin.Latest)
		}
	})

	t.Run("get non-existent plugin", func(t *testing.T) {
		_, err := r.Get("non-existent")
		if err == nil {
			t.Error("expected error for non-existent plugin")
		}
	})
}

func TestRegistry_GetVersion(t *testing.T) {
	index := createTestIndex()
	server := createTestServer(index)
	defer server.Close()

	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(t.TempDir()),
	)

	t.Run("get specific version", func(t *testing.T) {
		plugin, version, err := r.GetVersion("slack-notifier", "1.1.0")
		if err != nil {
			t.Fatalf("GetVersion() error = %v", err)
		}

		if plugin.Name != "slack-notifier" {
			t.Errorf("Name = %s, want slack-notifier", plugin.Name)
		}
		if version.Checksum != "def456" {
			t.Errorf("Checksum = %s, want def456", version.Checksum)
		}
	})

	t.Run("get latest when version is empty", func(t *testing.T) {
		plugin, version, err := r.GetVersion("slack-notifier", "")
		if err != nil {
			t.Fatalf("GetVersion() error = %v", err)
		}

		if version.Checksum != "ghi789" {
			t.Errorf("expected latest version checksum ghi789, got %s", version.Checksum)
		}
		if plugin.Latest != "1.2.0" {
			t.Errorf("Latest = %s, want 1.2.0", plugin.Latest)
		}
	})

	t.Run("get non-existent version", func(t *testing.T) {
		_, _, err := r.GetVersion("slack-notifier", "9.9.9")
		if err == nil {
			t.Error("expected error for non-existent version")
		}
	})
}

func TestRegistry_Search(t *testing.T) {
	index := createTestIndex()
	server := createTestServer(index)
	defer server.Close()

	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(t.TempDir()),
	)

	t.Run("search by name", func(t *testing.T) {
		results, err := r.Search("slack")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		if len(results) == 0 {
			t.Fatal("expected at least one result")
		}

		// First result should be slack-notifier (exact name match)
		if results[0].Plugin.Name != "slack-notifier" {
			t.Errorf("first result = %s, want slack-notifier", results[0].Plugin.Name)
		}
	})

	t.Run("search by keyword", func(t *testing.T) {
		results, err := r.Search("notifications")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		// Should find both notifiers
		if len(results) < 2 {
			t.Errorf("expected at least 2 results, got %d", len(results))
		}
	})

	t.Run("search by description", func(t *testing.T) {
		results, err := r.Search("security")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		found := false
		for _, r := range results {
			if r.Plugin.Name == "security-validator" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to find security-validator")
		}
	})

	t.Run("empty query returns all plugins", func(t *testing.T) {
		results, err := r.Search("")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		if len(results) != 5 {
			t.Errorf("expected 5 results, got %d", len(results))
		}
	})

	t.Run("deprecated plugins ranked lower", func(t *testing.T) {
		results, err := r.Search("plugin")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		// Find old-plugin in results
		var oldPluginIndex int
		for i, r := range results {
			if r.Plugin.Name == "old-plugin" {
				oldPluginIndex = i
				break
			}
		}

		// It should be ranked lower due to deprecation
		if oldPluginIndex < len(results)-2 {
			t.Errorf("deprecated plugin should be ranked lower, index = %d", oldPluginIndex)
		}
	})
}

func TestRegistry_SearchByType(t *testing.T) {
	index := createTestIndex()
	server := createTestServer(index)
	defer server.Close()

	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(t.TempDir()),
	)

	t.Run("search notifiers", func(t *testing.T) {
		results, err := r.SearchByType(PluginTypeNotifier)
		if err != nil {
			t.Fatalf("SearchByType() error = %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 notifiers, got %d", len(results))
		}

		for _, r := range results {
			if r.Plugin.Type != PluginTypeNotifier {
				t.Errorf("expected notifier type, got %s", r.Plugin.Type)
			}
		}
	})

	t.Run("search validators", func(t *testing.T) {
		results, err := r.SearchByType(PluginTypeValidator)
		if err != nil {
			t.Fatalf("SearchByType() error = %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 validator, got %d", len(results))
		}
	})
}

func TestRegistry_ResolveVersion(t *testing.T) {
	index := createTestIndex()
	server := createTestServer(index)
	defer server.Close()

	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(t.TempDir()),
	)

	tests := []struct {
		name       string
		plugin     string
		constraint string
		want       string
		wantErr    bool
	}{
		{
			name:       "empty constraint returns latest",
			plugin:     "slack-notifier",
			constraint: "",
			want:       "1.2.0",
		},
		{
			name:       "exact version",
			plugin:     "slack-notifier",
			constraint: "1.1.0",
			want:       "1.1.0",
		},
		{
			name:       "constraint >=1.0.0",
			plugin:     "slack-notifier",
			constraint: ">=1.0.0",
			want:       "1.2.0",
		},
		{
			name:       "constraint <1.2.0",
			plugin:     "slack-notifier",
			constraint: "<1.2.0",
			want:       "1.1.0",
		},
		{
			name:       "constraint ~1.1.0",
			plugin:     "slack-notifier",
			constraint: "~1.1.0",
			want:       "1.1.0",
		},
		{
			name:       "skips yanked versions",
			plugin:     "yanked-plugin",
			constraint: ">=1.0.0",
			want:       "1.0.0", // 1.1.0 is yanked
		},
		{
			name:       "non-existent plugin",
			plugin:     "non-existent",
			constraint: "1.0.0",
			wantErr:    true,
		},
		{
			name:       "unsatisfiable constraint",
			plugin:     "slack-notifier",
			constraint: ">99.0.0",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.ResolveVersion(tt.plugin, tt.constraint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveVersion() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestRegistry_GetPluginInfo(t *testing.T) {
	index := createTestIndex()
	server := createTestServer(index)
	defer server.Close()

	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(t.TempDir()),
	)

	t.Run("get plugin info", func(t *testing.T) {
		info, err := r.GetPluginInfo("slack-notifier")
		if err != nil {
			t.Fatalf("GetPluginInfo() error = %v", err)
		}

		if info.Name != "slack-notifier" {
			t.Errorf("Name = %s, want slack-notifier", info.Name)
		}
		if info.Latest != "1.2.0" {
			t.Errorf("Latest = %s, want 1.2.0", info.Latest)
		}
		if len(info.Versions) != 3 {
			t.Errorf("expected 3 versions, got %d", len(info.Versions))
		}
		// Versions should be sorted descending
		if info.Versions[0] != "1.2.0" {
			t.Errorf("first version should be 1.2.0, got %s", info.Versions[0])
		}
	})

	t.Run("non-existent plugin", func(t *testing.T) {
		_, err := r.GetPluginInfo("non-existent")
		if err == nil {
			t.Error("expected error for non-existent plugin")
		}
	})
}

func TestRegistry_CheckCompatibility(t *testing.T) {
	index := createTestIndex()
	server := createTestServer(index)
	defer server.Close()

	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(t.TempDir()),
	)

	tests := []struct {
		name       string
		plugin     string
		version    string
		cliVersion string
		wantErr    bool
	}{
		{
			name:       "compatible version",
			plugin:     "slack-notifier",
			version:    "1.2.0",
			cliVersion: "1.6.0",
			wantErr:    false,
		},
		{
			name:       "compatible higher CLI version",
			plugin:     "slack-notifier",
			version:    "1.2.0",
			cliVersion: "2.0.0",
			wantErr:    false,
		},
		{
			name:       "incompatible version",
			plugin:     "slack-notifier",
			version:    "1.2.0",
			cliVersion: "1.5.0",
			wantErr:    true,
		},
		{
			name:       "no minimum version required",
			plugin:     "yanked-plugin",
			version:    "1.0.0",
			cliVersion: "1.0.0",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.CheckCompatibility(tt.plugin, tt.version, tt.cliVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckCompatibility() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegistry_Cache(t *testing.T) {
	index := createTestIndex()
	server := createTestServer(index)
	defer server.Close()

	tmpDir := t.TempDir()
	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(tmpDir),
		WithCacheTTL(1*time.Hour),
	)

	t.Run("cache persists to disk", func(t *testing.T) {
		err := r.Fetch()
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		// Create new registry instance with same cache dir
		r2 := NewRegistry(
			WithRegistryURL(server.URL),
			WithCacheDir(tmpDir),
		)

		// Load cache
		err = r2.Fetch()
		if err != nil {
			t.Fatalf("Second Fetch() error = %v", err)
		}

		if !r2.IsCached() {
			t.Error("expected cache to be loaded from disk")
		}
	})

	t.Run("clear cache", func(t *testing.T) {
		err := r.Fetch()
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		err = r.ClearCache()
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("ClearCache() error = %v", err)
		}

		if r.IsCached() {
			t.Error("expected cache to be cleared")
		}
	})

	t.Run("cache age", func(t *testing.T) {
		r3 := NewRegistry(
			WithRegistryURL(server.URL),
			WithCacheDir(t.TempDir()),
		)

		err := r3.Fetch()
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		age := r3.GetCacheAge()
		if age > 1*time.Second {
			t.Errorf("cache age should be less than 1 second, got %v", age)
		}
	})
}

func TestRegistry_GetDownloadURL(t *testing.T) {
	index := createTestIndex()
	server := createTestServer(index)
	defer server.Close()

	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(t.TempDir()),
	)

	t.Run("construct URL from repository", func(t *testing.T) {
		url, err := r.GetDownloadURL("slack-notifier", "1.2.0")
		if err != nil {
			t.Fatalf("GetDownloadURL() error = %v", err)
		}

		expected := "https://github.com/specular/slack-notifier/archive/refs/tags/1.2.0.tar.gz"
		if url != expected {
			t.Errorf("URL = %s, want %s", url, expected)
		}
	})

	t.Run("non-existent plugin", func(t *testing.T) {
		_, err := r.GetDownloadURL("non-existent", "1.0.0")
		if err == nil {
			t.Error("expected error for non-existent plugin")
		}
	})
}

func TestRegistry_ListAll(t *testing.T) {
	index := createTestIndex()
	server := createTestServer(index)
	defer server.Close()

	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(t.TempDir()),
	)

	plugins, err := r.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}

	if len(plugins) != 5 {
		t.Errorf("expected 5 plugins, got %d", len(plugins))
	}

	// Should be sorted by name
	for i := 1; i < len(plugins); i++ {
		if plugins[i].Name < plugins[i-1].Name {
			t.Error("plugins should be sorted by name")
			break
		}
	}
}

func TestRegistry_FetchError(t *testing.T) {
	// Server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(t.TempDir()),
	)

	err := r.Fetch()
	if err == nil {
		t.Error("expected error for server error")
	}
}

func TestRegistry_FetchInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(t.TempDir()),
	)

	err := r.Fetch()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestRegistry_UsesStaleCache(t *testing.T) {
	index := createTestIndex()
	server := createTestServer(index)

	tmpDir := t.TempDir()
	r := NewRegistry(
		WithRegistryURL(server.URL),
		WithCacheDir(tmpDir),
		WithCacheTTL(1*time.Millisecond),
	)

	// First fetch populates cache
	err := r.Fetch()
	if err != nil {
		t.Fatalf("First Fetch() error = %v", err)
	}

	// Wait for cache to expire
	time.Sleep(10 * time.Millisecond)

	// Close server
	server.Close()

	// Should use stale cache
	err = r.Fetch()
	if err != nil {
		t.Fatalf("Fetch() with stale cache error = %v", err)
	}

	// Should still be able to search
	results, err := r.Search("slack")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("expected results from stale cache")
	}
}
