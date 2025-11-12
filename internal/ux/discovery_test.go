package ux

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSpecularDir(t *testing.T) {
	// Create a temporary test directory structure
	tmpDir := t.TempDir()

	// Create nested directories
	projectRoot := filepath.Join(tmpDir, "project")
	subDir := filepath.Join(projectRoot, "sub", "nested")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}

	// Create .specular directory in project root
	specularDir := filepath.Join(projectRoot, ".specular")
	if err := os.Mkdir(specularDir, 0755); err != nil {
		t.Fatalf("Failed to create .specular directory: %v", err)
	}

	// Change to nested directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test: Should find .specular in parent directory
	found, err := DiscoverSpecularDir()
	if err != nil {
		t.Fatalf("DiscoverSpecularDir failed: %v", err)
	}

	// Compare after resolving symlinks (macOS has /var -> /private/var)
	expectedResolved, _ := filepath.EvalSymlinks(specularDir)
	foundResolved, _ := filepath.EvalSymlinks(found)

	if foundResolved != expectedResolved {
		t.Errorf("Expected to find %s, got %s", expectedResolved, foundResolved)
	}
}

func TestDiscoverConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .specular directory
	specularDir := filepath.Join(tmpDir, ".specular")
	if err := os.Mkdir(specularDir, 0755); err != nil {
		t.Fatalf("Failed to create .specular directory: %v", err)
	}

	// Create a config file
	configFile := filepath.Join(specularDir, "test.yaml")
	if err := os.WriteFile(configFile, []byte("test: true"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Change to tmpDir
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test: Should find config file
	found, err := DiscoverConfigFile("test.yaml")
	if err != nil {
		t.Fatalf("DiscoverConfigFile failed: %v", err)
	}

	// Compare after resolving symlinks (macOS has /var -> /private/var)
	expectedResolved, _ := filepath.EvalSymlinks(configFile)
	foundResolved, _ := filepath.EvalSymlinks(found)

	if foundResolved != expectedResolved {
		t.Errorf("Expected to find %s, got %s", expectedResolved, foundResolved)
	}
}

func TestEnsureSpecularDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmpDir
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test: Should create .specular directory and subdirectories
	if err := EnsureSpecularDir(); err != nil {
		t.Fatalf("EnsureSpecularDir failed: %v", err)
	}

	// Check that .specular exists
	specularDir := filepath.Join(tmpDir, ".specular")
	if _, err := os.Stat(specularDir); os.IsNotExist(err) {
		t.Error(".specular directory was not created")
	}

	// Check subdirectories
	for _, subdir := range []string{"checkpoints", "runs", "cache", "logs"} {
		subdirPath := filepath.Join(specularDir, subdir)
		if _, err := os.Stat(subdirPath); os.IsNotExist(err) {
			t.Errorf("Subdirectory %s was not created", subdir)
		}
	}
}

func TestNewPathDefaultsWithDiscovery(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .specular directory
	specularDir := filepath.Join(tmpDir, ".specular")
	if err := os.Mkdir(specularDir, 0755); err != nil {
		t.Fatalf("Failed to create .specular directory: %v", err)
	}

	// Change to tmpDir
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test: Should discover .specular directory
	pd, err := NewPathDefaultsWithDiscovery()
	if err != nil {
		t.Fatalf("NewPathDefaultsWithDiscovery failed: %v", err)
	}

	// Compare after resolving symlinks (macOS has /var -> /private/var)
	expectedResolved, _ := filepath.EvalSymlinks(specularDir)
	foundResolved, _ := filepath.EvalSymlinks(pd.DiscoveredDir())

	if foundResolved != expectedResolved {
		t.Errorf("Expected discovered dir %s, got %s", expectedResolved, foundResolved)
	}

	if !pd.IsDiscovered() {
		t.Error("IsDiscovered should return true when .specular exists")
	}
}

func TestNewPathDefaultsWithDiscovery_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmpDir (no .specular directory)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test: Should use default when .specular not found
	pd, err := NewPathDefaultsWithDiscovery()
	if err != nil {
		t.Fatalf("NewPathDefaultsWithDiscovery failed: %v", err)
	}

	expectedDir := filepath.Join(tmpDir, ".specular")
	// Compare after resolving symlinks (macOS has /var -> /private/var)
	expectedResolved, _ := filepath.EvalSymlinks(expectedDir)
	foundResolved, _ := filepath.EvalSymlinks(pd.DiscoveredDir())

	if foundResolved != expectedResolved {
		t.Errorf("Expected default dir %s, got %s", expectedResolved, foundResolved)
	}

	// .specular doesn't exist yet, so IsDiscovered should return false
	if pd.IsDiscovered() {
		t.Error("IsDiscovered should return false when .specular doesn't exist")
	}
}
