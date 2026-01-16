package ux

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/specular/internal/safeutil"
)

// PathDefaults provides smart defaults for common file paths
type PathDefaults struct {
	SpecularDir string
}

func (pd *PathDefaults) safeJoin(segment string) string {
	joined := filepath.Join(pd.SpecularDir, segment)
	if pd.SpecularDir == "" {
		return joined
	}
	if _, err := safeutil.JoinInsideBase(pd.SpecularDir, segment); err == nil {
		return joined
	}
	return joined
}

// NewPathDefaults creates a new PathDefaults with sensible defaults
func NewPathDefaults() *PathDefaults {
	return &PathDefaults{
		SpecularDir: ".specular",
	}
}

// SpecFile returns the default path to spec.yaml, checking if it exists
func (pd *PathDefaults) SpecFile() string {
	path := pd.safeJoin("spec.yaml")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return path // Return path anyway for creation
}

// SpecLockFile returns the default path to spec.lock.json
func (pd *PathDefaults) SpecLockFile() string {
	return pd.safeJoin("spec.lock.json")
}

// PlanFile returns the default path to plan.json
func (pd *PathDefaults) PlanFile() string {
	return pd.safeJoin("plan.json")
}

// PolicyFile returns the default path to policy.yaml
func (pd *PathDefaults) PolicyFile() string {
	path := pd.safeJoin("policy.yaml")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	// Fallback to old location for backward compatibility
	oldPath := ".aidv/policy.yaml"
	if _, err := os.Stat(oldPath); err == nil {
		return oldPath
	}
	return path
}

// ProvidersFile returns the default path to providers.yaml
func (pd *PathDefaults) ProvidersFile() string {
	return pd.safeJoin("providers.yaml")
}

// RoutingFile returns the default path to routing.yaml
func (pd *PathDefaults) RoutingFile() string {
	return pd.safeJoin("routing.yaml")
}

// CheckpointDir returns the default checkpoint directory
func (pd *PathDefaults) CheckpointDir() string {
	return pd.safeJoin("checkpoints")
}

// ManifestDir returns the default run manifest directory
func (pd *PathDefaults) ManifestDir() string {
	return pd.safeJoin("runs")
}

// CacheDir returns the default cache directory
func (pd *PathDefaults) CacheDir() string {
	return pd.safeJoin("cache")
}

// ValidateSpecularSetup checks if the .specular directory is initialized
func (pd *PathDefaults) ValidateSpecularSetup() error {
	if _, err := os.Stat(pd.SpecularDir); os.IsNotExist(err) {
		return fmt.Errorf(".specular directory not found. Run 'specular init' to set up your project")
	}
	return nil
}

// ValidateRequiredFile checks if a required file exists and provides helpful error
func ValidateRequiredFile(path string, fileType string, creationCommand string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("%s not found at: %s\n\nRun '%s' to create it", fileType, path, creationCommand)
	} else if err != nil {
		return fmt.Errorf("error accessing %s: %w", path, err)
	}
	return nil
}

// SuggestNextSteps provides contextual next steps based on what exists
func SuggestNextSteps() string {
	defaults := NewPathDefaults()

	_, hasSpecular := os.Stat(defaults.SpecularDir)
	_, hasSpec := os.Stat(defaults.SpecFile())
	_, hasLock := os.Stat(defaults.SpecLockFile())
	_, hasPlan := os.Stat(defaults.PlanFile())

	if os.IsNotExist(hasSpecular) {
		return "Run 'specular init' to set up your project"
	}

	if os.IsNotExist(hasSpec) {
		return "Create a spec with 'specular spec new' or 'specular spec new --from PRD.md'"
	}

	if os.IsNotExist(hasLock) {
		return "Lock your spec with 'specular spec lock'"
	}

	if os.IsNotExist(hasPlan) {
		return "Generate a plan with 'specular plan create'"
	}

	return "Execute your plan with 'specular build run'"
}

// ConfigPaths holds all discovered configuration file paths for a project.
// Use DiscoverAllConfigs() to populate this struct with auto-discovered paths.
type ConfigPaths struct {
	// ProjectRoot is the root directory of the project (contains .git or .specular)
	ProjectRoot string
	// SpecularDir is the path to the .specular directory
	SpecularDir string
	// SpecFile is the path to spec.yaml
	SpecFile string
	// LockFile is the path to spec.lock.json
	LockFile string
	// PlanFile is the path to plan.json
	PlanFile string
	// RoutingFile is the path to routing.yaml
	RoutingFile string
	// PolicyFile is the path to policy.yaml
	PolicyFile string
	// ProvidersFile is the path to providers.yaml
	ProvidersFile string
	// CheckpointDir is the path to the checkpoints directory
	CheckpointDir string
	// CacheDir is the path to the cache directory
	CacheDir string
}

// Exists returns true if the given path exists
func (cp *ConfigPaths) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// HasSpec returns true if a spec file exists
func (cp *ConfigPaths) HasSpec() bool {
	return cp.Exists(cp.SpecFile)
}

// HasLock returns true if a lock file exists
func (cp *ConfigPaths) HasLock() bool {
	return cp.Exists(cp.LockFile)
}

// HasRouting returns true if a routing config exists
func (cp *ConfigPaths) HasRouting() bool {
	return cp.Exists(cp.RoutingFile)
}

// HasPolicy returns true if a policy config exists
func (cp *ConfigPaths) HasPolicy() bool {
	return cp.Exists(cp.PolicyFile)
}

// HasProviders returns true if a providers config exists
func (cp *ConfigPaths) HasProviders() bool {
	return cp.Exists(cp.ProvidersFile)
}

// IsInitialized returns true if the project has been initialized
func (cp *ConfigPaths) IsInitialized() bool {
	return cp.Exists(cp.SpecularDir)
}

// DiscoverProjectRoot finds the project root directory.
// It searches for .git or .specular directories starting from cwd
// and walking up the directory tree.
func DiscoverProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot get current directory: %w", err)
	}

	dir := cwd
	for {
		// Check for .specular directory (Specular project root)
		specularDir := filepath.Join(dir, ".specular")
		if _, err := os.Stat(specularDir); err == nil {
			return dir, nil
		}

		// Check for .git directory (Git repository root)
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root, use current working directory
			return cwd, nil
		}
		dir = parent
	}
}

// DiscoverAllConfigs discovers all configuration file paths for the project.
// It auto-detects the project root and .specular directory, then resolves
// all config file paths relative to that location.
func DiscoverAllConfigs() (*ConfigPaths, error) {
	projectRoot, err := DiscoverProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to discover project root: %w", err)
	}

	specularDir, joinErr := safeutil.JoinInsideBase(projectRoot, ".specular")
	if joinErr != nil {
		specularDir = filepath.Join(projectRoot, ".specular")
	}

	// Use PathDefaults to get consistent file paths
	pd := &PathDefaults{SpecularDir: specularDir}

	return &ConfigPaths{
		ProjectRoot:   projectRoot,
		SpecularDir:   specularDir,
		SpecFile:      pd.SpecFile(),
		LockFile:      pd.SpecLockFile(),
		PlanFile:      filepath.Join(projectRoot, pd.PlanFile()),
		RoutingFile:   pd.RoutingFile(),
		PolicyFile:    pd.PolicyFile(),
		ProvidersFile: pd.ProvidersFile(),
		CheckpointDir: pd.CheckpointDir(),
		CacheDir:      pd.CacheDir(),
	}, nil
}

// MustDiscoverAllConfigs is like DiscoverAllConfigs but returns defaults on error
func MustDiscoverAllConfigs() *ConfigPaths {
	configs, err := DiscoverAllConfigs()
	if err != nil {
		// Return defaults from current directory
		pd := NewPathDefaults()
		cwd, _ := os.Getwd()
		return &ConfigPaths{
			ProjectRoot:   cwd,
			SpecularDir:   pd.SpecularDir,
			SpecFile:      pd.SpecFile(),
			LockFile:      pd.SpecLockFile(),
			PlanFile:      pd.PlanFile(),
			RoutingFile:   pd.RoutingFile(),
			PolicyFile:    pd.PolicyFile(),
			ProvidersFile: pd.ProvidersFile(),
			CheckpointDir: pd.CheckpointDir(),
			CacheDir:      pd.CacheDir(),
		}
	}
	return configs
}
