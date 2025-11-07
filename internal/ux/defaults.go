package ux

import (
	"fmt"
	"os"
	"path/filepath"
)

// PathDefaults provides smart defaults for common file paths
type PathDefaults struct {
	SpecularDir string
}

// NewPathDefaults creates a new PathDefaults with sensible defaults
func NewPathDefaults() *PathDefaults {
	return &PathDefaults{
		SpecularDir: ".specular",
	}
}

// SpecFile returns the default path to spec.yaml, checking if it exists
func (pd *PathDefaults) SpecFile() string {
	path := filepath.Join(pd.SpecularDir, "spec.yaml")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return path // Return path anyway for creation
}

// SpecLockFile returns the default path to spec.lock.json
func (pd *PathDefaults) SpecLockFile() string {
	return filepath.Join(pd.SpecularDir, "spec.lock.json")
}

// PlanFile returns the default path to plan.json
func (pd *PathDefaults) PlanFile() string {
	return "plan.json"
}

// PolicyFile returns the default path to policy.yaml
func (pd *PathDefaults) PolicyFile() string {
	path := filepath.Join(pd.SpecularDir, "policy.yaml")
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
	return filepath.Join(pd.SpecularDir, "providers.yaml")
}

// RouterFile returns the default path to router.yaml
func (pd *PathDefaults) RouterFile() string {
	return filepath.Join(pd.SpecularDir, "router.yaml")
}

// CheckpointDir returns the default checkpoint directory
func (pd *PathDefaults) CheckpointDir() string {
	return filepath.Join(pd.SpecularDir, "checkpoints")
}

// ManifestDir returns the default run manifest directory
func (pd *PathDefaults) ManifestDir() string {
	return filepath.Join(pd.SpecularDir, "runs")
}

// CacheDir returns the default cache directory
func (pd *PathDefaults) CacheDir() string {
	return filepath.Join(pd.SpecularDir, "cache")
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
		return "Create a spec with 'specular interview' or 'specular spec generate'"
	}

	if os.IsNotExist(hasLock) {
		return "Lock your spec with 'specular spec lock'"
	}

	if os.IsNotExist(hasPlan) {
		return "Generate a plan with 'specular plan'"
	}

	return "Execute your plan with 'specular build'"
}
