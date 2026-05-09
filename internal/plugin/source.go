package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SourceType represents the type of plugin source
type SourceType string

const (
	// SourceTypeLocal is a local filesystem path
	SourceTypeLocal SourceType = "local"
	// SourceTypeGitHub is a GitHub repository
	SourceTypeGitHub SourceType = "github"
	// SourceTypeRegistry is a plugin registry
	SourceTypeRegistry SourceType = "registry"
)

// PluginSource represents a parsed plugin source location
type PluginSource struct {
	// Type is the source type (local, github, registry)
	Type SourceType `json:"type"`
	// Raw is the original source string
	Raw string `json:"raw"`
	// Path is the local filesystem path (for local sources)
	Path string `json:"path,omitempty"`
	// Owner is the GitHub owner/org (for GitHub sources)
	Owner string `json:"owner,omitempty"`
	// Repo is the GitHub repository name (for GitHub sources)
	Repo string `json:"repo,omitempty"`
	// Version is the version constraint or ref (e.g., "v1.0.0", "main")
	Version string `json:"version,omitempty"`
	// Name is the plugin name (for registry sources)
	Name string `json:"name,omitempty"`
	// Subpath is an optional path within the repository
	Subpath string `json:"subpath,omitempty"`
}

// GitHub URL patterns
var (
	// githubFullURL matches https://github.com/owner/repo or git@github.com:owner/repo
	githubFullURL = regexp.MustCompile(`^(?:https?://)?github\.com[/:]([^/]+)/([^/@.]+)(?:\.git)?(?:@([^/]+))?(?:/(.+))?$`)
	// githubShorthand matches github.com/owner/repo[@version][/path]
	githubShorthand = regexp.MustCompile(`^github\.com/([^/]+)/([^/@.]+)(?:\.git)?(?:@([^/]+))?(?:/(.+))?$`)
	// registryPattern matches registry:name[@version]
	registryPattern = regexp.MustCompile(`^registry:([^@]+)(?:@(.+))?$`)
	// gitRefPattern matches safe git refs for branch/tag/sha inputs.
	gitRefPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
)

// ParseSource parses a plugin source string into a PluginSource
// Supported formats:
//   - Local: ./path, ../path, /absolute/path, relative/path
//   - GitHub: github.com/owner/repo, github.com/owner/repo@v1.0.0
//   - GitHub with subpath: github.com/owner/repo@v1.0.0/plugins/my-plugin
//   - Registry: registry:plugin-name, registry:plugin-name@1.0.0
func ParseSource(source string) (*PluginSource, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, fmt.Errorf("empty source string")
	}

	// Check for registry prefix
	if strings.HasPrefix(source, "registry:") {
		return parseRegistrySource(source)
	}

	// Check for GitHub patterns
	if strings.Contains(source, "github.com") {
		return parseGitHubSource(source)
	}

	// Treat as local path
	return parseLocalSource(source)
}

// parseLocalSource parses a local filesystem path
func parseLocalSource(source string) (*PluginSource, error) {
	// Clean and resolve the path
	path := source

	// Handle relative paths
	if !filepath.IsAbs(path) {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path: %w", err)
		}
	}

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", source)
		}
		return nil, fmt.Errorf("failed to access path: %w", err)
	}

	// Must be a directory
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", source)
	}

	return &PluginSource{
		Type: SourceTypeLocal,
		Raw:  source,
		Path: path,
	}, nil
}

// parseGitHubSource parses a GitHub repository URL
func parseGitHubSource(source string) (*PluginSource, error) {
	// Try full URL pattern first
	if matches := githubFullURL.FindStringSubmatch(source); matches != nil {
		src := &PluginSource{
			Type:    SourceTypeGitHub,
			Raw:     source,
			Owner:   matches[1],
			Repo:    matches[2],
			Version: matches[3],
			Subpath: matches[4],
		}
		if err := validateGitHubSourceFields(src); err != nil {
			return nil, err
		}
		return src, nil
	}

	// Try shorthand pattern
	if matches := githubShorthand.FindStringSubmatch(source); matches != nil {
		src := &PluginSource{
			Type:    SourceTypeGitHub,
			Raw:     source,
			Owner:   matches[1],
			Repo:    matches[2],
			Version: matches[3],
			Subpath: matches[4],
		}
		if err := validateGitHubSourceFields(src); err != nil {
			return nil, err
		}
		return src, nil
	}

	return nil, fmt.Errorf("invalid GitHub source format: %s", source)
}

func validateGitHubSourceFields(src *PluginSource) error {
	if src == nil {
		return fmt.Errorf("invalid GitHub source")
	}

	if src.Version != "" {
		if strings.HasPrefix(src.Version, "-") {
			return fmt.Errorf("invalid GitHub version/ref: must not start with '-' (%s)", src.Version)
		}
		if strings.Contains(src.Version, "..") || strings.Contains(src.Version, "\\") {
			return fmt.Errorf("invalid GitHub version/ref: %s", src.Version)
		}
		if !gitRefPattern.MatchString(src.Version) {
			return fmt.Errorf("invalid GitHub version/ref characters: %s", src.Version)
		}
	}

	if src.Subpath != "" {
		cleanSubpath := filepath.Clean(src.Subpath)
		if filepath.IsAbs(cleanSubpath) || cleanSubpath == "." || strings.HasPrefix(cleanSubpath, "..") || strings.Contains(cleanSubpath, ".."+string(filepath.Separator)) {
			return fmt.Errorf("invalid GitHub subpath: %s", src.Subpath)
		}
		src.Subpath = cleanSubpath
	}

	return nil
}

// parseRegistrySource parses a registry plugin reference
func parseRegistrySource(source string) (*PluginSource, error) {
	matches := registryPattern.FindStringSubmatch(source)
	if matches == nil {
		return nil, fmt.Errorf("invalid registry source format: %s", source)
	}

	return &PluginSource{
		Type:    SourceTypeRegistry,
		Raw:     source,
		Name:    matches[1],
		Version: matches[2],
	}, nil
}

// String returns a human-readable representation of the source
func (s *PluginSource) String() string {
	switch s.Type {
	case SourceTypeLocal:
		return s.Path
	case SourceTypeGitHub:
		str := fmt.Sprintf("github.com/%s/%s", s.Owner, s.Repo)
		if s.Version != "" {
			str += "@" + s.Version
		}
		if s.Subpath != "" {
			str += "/" + s.Subpath
		}
		return str
	case SourceTypeRegistry:
		str := "registry:" + s.Name
		if s.Version != "" {
			str += "@" + s.Version
		}
		return str
	default:
		return s.Raw
	}
}

// GitHubURL returns the full GitHub URL for GitHub sources
func (s *PluginSource) GitHubURL() string {
	if s.Type != SourceTypeGitHub {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s", s.Owner, s.Repo)
}

// GitHubCloneURL returns the clone URL for GitHub sources
func (s *PluginSource) GitHubCloneURL() string {
	if s.Type != SourceTypeGitHub {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s.git", s.Owner, s.Repo)
}

// GitHubTarballURL returns the tarball download URL for a specific ref
func (s *PluginSource) GitHubTarballURL() string {
	if s.Type != SourceTypeGitHub {
		return ""
	}
	ref := s.Version
	if ref == "" {
		ref = "main"
	}
	return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.tar.gz", s.Owner, s.Repo, ref)
}

// GitHubTagTarballURL returns the tarball URL for a tag
func (s *PluginSource) GitHubTagTarballURL() string {
	if s.Type != SourceTypeGitHub || s.Version == "" {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.tar.gz", s.Owner, s.Repo, s.Version)
}

// GitHubReleaseTarballURL returns the tarball URL from GitHub releases
func (s *PluginSource) GitHubReleaseTarballURL() string {
	if s.Type != SourceTypeGitHub || s.Version == "" {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s-%s.tar.gz",
		s.Owner, s.Repo, s.Version, s.Repo, s.Version)
}

// IsVersioned returns true if the source has a version constraint
func (s *PluginSource) IsVersioned() bool {
	return s.Version != ""
}

// IsLocal returns true if this is a local source
func (s *PluginSource) IsLocal() bool {
	return s.Type == SourceTypeLocal
}

// IsRemote returns true if this is a remote source (GitHub or registry)
func (s *PluginSource) IsRemote() bool {
	return s.Type == SourceTypeGitHub || s.Type == SourceTypeRegistry
}

// GetPluginName extracts the plugin name from the source
func (s *PluginSource) GetPluginName() string {
	switch s.Type {
	case SourceTypeLocal:
		// Use the directory name
		return filepath.Base(s.Path)
	case SourceTypeGitHub:
		// Use repo name or subpath if present
		if s.Subpath != "" {
			return filepath.Base(s.Subpath)
		}
		return s.Repo
	case SourceTypeRegistry:
		return s.Name
	default:
		return ""
	}
}

// WithVersion returns a new PluginSource with the specified version
func (s *PluginSource) WithVersion(version string) *PluginSource {
	copy := *s
	copy.Version = version
	return &copy
}

// ValidateVersion checks if the version string is valid for this source type
func (s *PluginSource) ValidateVersion() error {
	if s.Version == "" {
		return nil // No version is valid
	}

	switch s.Type {
	case SourceTypeLocal:
		// Local sources don't use versions
		return fmt.Errorf("local sources don't support version specifiers")
	case SourceTypeGitHub:
		// GitHub versions can be tags, branches, or commit SHAs
		// All are valid as strings
		return nil
	case SourceTypeRegistry:
		// Registry versions should be semver
		_, err := ParseVersion(s.Version)
		if err != nil {
			// Try parsing as constraint
			_, err = ParseConstraint(s.Version)
			if err != nil {
				return fmt.Errorf("invalid version for registry source: %s", s.Version)
			}
		}
		return nil
	default:
		return nil
	}
}

// MustParseSource parses a source and panics on error (for testing)
func MustParseSource(source string) *PluginSource {
	s, err := ParseSource(source)
	if err != nil {
		panic(err)
	}
	return s
}
