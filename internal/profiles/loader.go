package profiles

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed builtin/*.yaml
var builtinProfiles embed.FS

// Loader handles loading and resolving profiles from various sources.
type Loader struct {
	// projectDir is the project directory (for ./auto.profiles.yaml)
	projectDir string

	// userDir is the user config directory (for ~/.specular/auto.profiles.yaml)
	userDir string

	// cache stores loaded profiles
	cache map[string]*Profile
}

// NewLoader creates a new profile loader.
func NewLoader() *Loader {
	homeDir, _ := os.UserHomeDir()
	userDir := filepath.Join(homeDir, ".specular")

	return &Loader{
		projectDir: ".",
		userDir:    userDir,
		cache:      make(map[string]*Profile),
	}
}

// SetProjectDir sets the project directory for project-level profiles.
func (l *Loader) SetProjectDir(dir string) {
	l.projectDir = dir
}

// Load loads a profile by name, resolving from multiple sources.
//
// Resolution order (highest to lowest precedence):
// 1. Project-level profile (./auto.profiles.yaml)
// 2. User-level profile (~/.specular/auto.profiles.yaml)
// 3. Built-in profile (embedded in binary)
//
// If the profile is not found in any source, returns an error.
func (l *Loader) Load(name string) (*Profile, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", l.projectDir, name)
	if cached, ok := l.cache[cacheKey]; ok {
		return cached, nil
	}

	// Start with built-in profile as base
	base, err := l.loadBuiltin(name)
	if err != nil {
		return nil, fmt.Errorf("profile %q not found in built-in profiles: %w", name, err)
	}

	// Layer user-level profile
	if userProfile, err := l.loadUser(name); err == nil {
		base = base.Merge(userProfile)
	}

	// Layer project-level profile (highest precedence)
	if projectProfile, err := l.loadProject(name); err == nil {
		base = base.Merge(projectProfile)
	}

	// Validate final profile
	if err := base.Validate(); err != nil {
		return nil, fmt.Errorf("invalid profile %q: %w", name, err)
	}

	// Cache and return
	l.cache[cacheKey] = base
	return base, nil
}

// LoadFromFile loads profiles from a specific file.
func (l *Loader) LoadFromFile(path string, name string) (*Profile, error) {
	collection, err := l.parseYAMLFile(path)
	if err != nil {
		return nil, err
	}

	profile, ok := collection.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in %s", name, path)
	}

	profile.Name = name
	if err := profile.Validate(); err != nil {
		return nil, fmt.Errorf("invalid profile %q in %s: %w", name, path, err)
	}

	return &profile, nil
}

// List returns available profile names from all sources.
func (l *Loader) List() ([]string, error) {
	profiles := make(map[string]bool)

	// Built-in profiles
	builtinNames, err := l.listBuiltin()
	if err != nil {
		return nil, err
	}
	for _, name := range builtinNames {
		profiles[name] = true
	}

	// User-level profiles
	if userNames, err := l.listUser(); err == nil {
		for _, name := range userNames {
			profiles[name] = true
		}
	}

	// Project-level profiles
	if projectNames, err := l.listProject(); err == nil {
		for _, name := range projectNames {
			profiles[name] = true
		}
	}

	// Convert to sorted slice
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}

	return names, nil
}

// loadBuiltin loads a built-in profile from embedded files.
func (l *Loader) loadBuiltin(name string) (*Profile, error) {
	data, err := builtinProfiles.ReadFile(fmt.Sprintf("builtin/%s.yaml", name))
	if err != nil {
		return nil, fmt.Errorf("built-in profile not found: %w", err)
	}

	var collection ProfileCollection
	if err := yaml.Unmarshal(data, &collection); err != nil {
		return nil, fmt.Errorf("failed to parse built-in profile: %w", err)
	}

	profile, ok := collection.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in built-in file", name)
	}

	profile.Name = name
	return &profile, nil
}

// loadUser loads a user-level profile from ~/.specular/auto.profiles.yaml.
func (l *Loader) loadUser(name string) (*Profile, error) {
	path := filepath.Join(l.userDir, "auto.profiles.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("user profile file not found")
	}

	return l.LoadFromFile(path, name)
}

// loadProject loads a project-level profile from ./auto.profiles.yaml.
func (l *Loader) loadProject(name string) (*Profile, error) {
	path := filepath.Join(l.projectDir, "auto.profiles.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("project profile file not found")
	}

	return l.LoadFromFile(path, name)
}

// listBuiltin returns names of built-in profiles.
func (l *Loader) listBuiltin() ([]string, error) {
	entries, err := builtinProfiles.ReadDir("builtin")
	if err != nil {
		return nil, fmt.Errorf("failed to read built-in profiles: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			name := strings.TrimSuffix(entry.Name(), ".yaml")
			names = append(names, name)
		}
	}

	return names, nil
}

// listUser returns names of user-level profiles.
func (l *Loader) listUser() ([]string, error) {
	path := filepath.Join(l.userDir, "auto.profiles.yaml")
	collection, err := l.parseYAMLFile(path)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(collection.Profiles))
	for name := range collection.Profiles {
		names = append(names, name)
	}

	return names, nil
}

// listProject returns names of project-level profiles.
func (l *Loader) listProject() ([]string, error) {
	path := filepath.Join(l.projectDir, "auto.profiles.yaml")
	collection, err := l.parseYAMLFile(path)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(collection.Profiles))
	for name := range collection.Profiles {
		names = append(names, name)
	}

	return names, nil
}

// parseYAMLFile parses a YAML file into a ProfileCollection.
func (l *Loader) parseYAMLFile(path string) (*ProfileCollection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile file: %w", err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	var collection ProfileCollection
	if err := yaml.Unmarshal([]byte(expanded), &collection); err != nil {
		return nil, fmt.Errorf("failed to parse profile file: %w", err)
	}

	// Validate schema version
	if collection.Schema != "" && !strings.HasPrefix(collection.Schema, "specular.auto.profiles/v") {
		return nil, fmt.Errorf("unsupported schema version: %s", collection.Schema)
	}

	return &collection, nil
}

// GetDefault returns the default profile.
func (l *Loader) GetDefault() (*Profile, error) {
	return l.Load("default")
}

// MergeWithCLIFlags merges a profile with CLI flag overrides.
// CLI flags take precedence over profile settings.
func MergeWithCLIFlags(profile *Profile, flags *CLIFlags) *Profile {
	merged := *profile // Copy profile

	// Override approval settings
	if flags.RequireApproval != nil {
		merged.Approvals.Interactive = *flags.RequireApproval
		if *flags.RequireApproval {
			merged.Approvals.Mode = ApprovalModeAll
		} else {
			merged.Approvals.Mode = ApprovalModeNone
		}
	}

	// Override safety settings
	if flags.MaxSteps != nil {
		merged.Safety.MaxSteps = *flags.MaxSteps
	}
	if flags.Timeout != nil {
		merged.Safety.Timeout = *flags.Timeout
	}
	if flags.MaxCostUSD != nil {
		merged.Safety.MaxCostUSD = *flags.MaxCostUSD
	}
	if flags.MaxCostPerTask != nil {
		merged.Safety.MaxCostPerTask = *flags.MaxCostPerTask
	}
	if flags.MaxRetries != nil {
		merged.Safety.MaxRetries = *flags.MaxRetries
	}

	// Override execution settings
	if flags.Trace != nil {
		merged.Execution.TraceLogging = *flags.Trace
	}
	if flags.SavePatches != nil {
		merged.Execution.SavePatches = *flags.SavePatches
	}
	if flags.JSONOutput != nil {
		merged.Execution.JSONOutput = *flags.JSONOutput
	}
	if flags.EnableTUI != nil {
		merged.Execution.EnableTUI = *flags.EnableTUI
	}

	return &merged
}

// CLIFlags represents CLI flag overrides for profile settings.
type CLIFlags struct {
	// Approval overrides
	RequireApproval *bool

	// Safety overrides
	MaxSteps       *int
	Timeout        *time.Duration
	MaxCostUSD     *float64
	MaxCostPerTask *float64
	MaxRetries     *int

	// Execution overrides
	Trace       *bool
	SavePatches *bool
	JSONOutput  *bool
	EnableTUI   *bool
}
