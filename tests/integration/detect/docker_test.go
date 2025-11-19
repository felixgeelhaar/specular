//go:build integration
// +build integration

package detect_test

import (
	"os/exec"
	"testing"

	"github.com/felixgeelhaar/specular/internal/detect"
)

// TestDetectDocker tests Docker detection with real Docker installation
func TestDetectDocker(t *testing.T) {
	// Check if Docker is available in test environment
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available in test environment")
	}

	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	// Verify detection worked
	if !ctx.Docker.Available {
		t.Error("Docker should be detected as available")
	}

	// Version should be populated
	if ctx.Docker.Version == "" {
		t.Error("Docker version should be detected")
	}

	// Runtime should be set to "docker" if Docker is available
	if ctx.Docker.Available && ctx.Runtime != "docker" {
		t.Errorf("Runtime = %s, want docker", ctx.Runtime)
	}

	t.Logf("Detected Docker version: %s", ctx.Docker.Version)
	t.Logf("Docker running: %v", ctx.Docker.Running)
}

// TestDetectDockerFields tests all Docker-related fields
func TestDetectDockerFields(t *testing.T) {
	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	// Log all Docker fields for debugging
	t.Logf("Docker.Available: %v", ctx.Docker.Available)
	t.Logf("Docker.Version: %s", ctx.Docker.Version)
	t.Logf("Docker.Running: %v", ctx.Docker.Running)
	t.Logf("Context.Runtime: %s", ctx.Runtime)

	// If Docker is available, runtime should be "docker" unless Podman takes precedence
	if ctx.Docker.Available && ctx.Runtime == "" && !ctx.Podman.Available {
		t.Error("Runtime should be set to 'docker' when Docker is available")
	}
}

// TestDetectDockerVersion tests that version parsing works correctly
func TestDetectDockerVersion(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available in test environment")
	}

	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	if !ctx.Docker.Available {
		t.Skip("Docker binary is not available")
	}

	// Version should be non-empty if Available is true
	if ctx.Docker.Version == "" {
		t.Error("Docker version should be populated when Available is true")
	}

	t.Logf("Docker version: %s", ctx.Docker.Version)
}
