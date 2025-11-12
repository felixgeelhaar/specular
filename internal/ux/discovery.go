package ux

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DiscoverSpecularDir searches for .specular directory in multiple locations
// Priority: current dir -> parent dirs -> git root -> home dir
func DiscoverSpecularDir() (string, error) {
	// 1. Check current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	specularPath := filepath.Join(cwd, ".specular")
	if _, err := os.Stat(specularPath); err == nil {
		return specularPath, nil
	}

	// 2. Search parent directories (up to git root or filesystem root)
	dir := cwd
	for {
		specularPath = filepath.Join(dir, ".specular")
		if _, err := os.Stat(specularPath); err == nil {
			return specularPath, nil
		}

		// Check if we're at git root
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			// We're at git root but no .specular found yet
			// Keep searching up one more level in case it's in a parent workspace
			parent := filepath.Dir(dir)
			if parent == dir {
				// At filesystem root
				break
			}
			dir = parent
			continue
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	// 3. Try git root explicitly
	if gitRoot, err := getGitRoot(); err == nil {
		specularPath = filepath.Join(gitRoot, ".specular")
		if _, err := os.Stat(specularPath); err == nil {
			return specularPath, nil
		}
	}

	// 4. Fallback to current directory (will be created if needed)
	return filepath.Join(cwd, ".specular"), nil
}

// DiscoverConfigFile searches for a config file in multiple locations
func DiscoverConfigFile(filename string) (string, error) {
	// Try these locations in order:
	// 1. .specular/<filename>
	// 2. ./<filename>
	// 3. Parent directories up to git root
	// 4. ~/.specular/<filename>

	// 1. Check .specular directory
	specularDir, err := DiscoverSpecularDir()
	if err == nil {
		configPath := filepath.Join(specularDir, filename)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	// 2. Check current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(cwd, filename)
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	// 3. Search parent directories
	dir := cwd
	for {
		configPath = filepath.Join(dir, filename)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// Stop at git root
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			break
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	// 4. Check home directory .specular
	if homeDir, err := os.UserHomeDir(); err == nil {
		configPath = filepath.Join(homeDir, ".specular", filename)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	// Not found - return expected location in .specular dir
	if specularDir != "" {
		return filepath.Join(specularDir, filename), nil
	}

	return filepath.Join(cwd, ".specular", filename), nil
}

// getGitRoot returns the git repository root directory
func getGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// EnsureSpecularDir ensures the .specular directory exists
func EnsureSpecularDir() error {
	specularDir, err := DiscoverSpecularDir()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	if _, err := os.Stat(specularDir); os.IsNotExist(err) {
		if err := os.MkdirAll(specularDir, 0755); err != nil {
			return err
		}
	}

	// Create subdirectories
	subdirs := []string{"checkpoints", "runs", "cache", "logs"}
	for _, subdir := range subdirs {
		path := filepath.Join(specularDir, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	return nil
}

// PathDefaultsWithDiscovery creates PathDefaults using auto-discovery
type PathDefaultsWithDiscovery struct {
	*PathDefaults
	discoveredDir string
}

// NewPathDefaultsWithDiscovery creates PathDefaults with auto-discovered .specular directory
func NewPathDefaultsWithDiscovery() (*PathDefaultsWithDiscovery, error) {
	dir, err := DiscoverSpecularDir()
	if err != nil {
		// Fallback to default
		return &PathDefaultsWithDiscovery{
			PathDefaults:  NewPathDefaults(),
			discoveredDir: ".specular",
		}, nil
	}

	return &PathDefaultsWithDiscovery{
		PathDefaults: &PathDefaults{
			SpecularDir: dir,
		},
		discoveredDir: dir,
	}, nil
}

// DiscoveredDir returns the auto-discovered .specular directory path
func (pd *PathDefaultsWithDiscovery) DiscoveredDir() string {
	return pd.discoveredDir
}

// IsDiscovered returns true if .specular directory was found (vs created)
func (pd *PathDefaultsWithDiscovery) IsDiscovered() bool {
	_, err := os.Stat(pd.discoveredDir)
	return err == nil
}
