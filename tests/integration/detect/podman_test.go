// +build integration

package detect_test

import (
	"os/exec"
	"testing"

	"github.com/felixgeelhaar/specular/internal/detect"
)

// TestDetectPodman tests Podman detection with real Podman installation
func TestDetectPodman(t *testing.T) {
	// Check if Podman is available in test environment
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("Podman not available in test environment")
	}

	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	// Verify detection worked
	if !ctx.Podman.Available {
		t.Error("Podman should be detected as available")
	}

	// Version should be populated
	if ctx.Podman.Version == "" {
		t.Error("Podman version should be detected")
	}

	// If Podman is available and Docker is not, Runtime should be "podman"
	if ctx.Podman.Available && !ctx.Docker.Available && ctx.Runtime != "podman" {
		t.Errorf("Runtime = %s, want podman", ctx.Runtime)
	}

	t.Logf("Detected Podman version: %s", ctx.Podman.Version)
	t.Logf("Podman running: %v", ctx.Podman.Running)
}

// TestDetectPodmanFields tests all Podman-related fields
func TestDetectPodmanFields(t *testing.T) {
	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	// Log all Podman fields for debugging
	t.Logf("Podman.Available: %v", ctx.Podman.Available)
	t.Logf("Podman.Version: %s", ctx.Podman.Version)
	t.Logf("Podman.Running: %v", ctx.Podman.Running)
	t.Logf("Context.Runtime: %s", ctx.Runtime)

	// If Podman is available, check runtime selection logic
	if ctx.Podman.Available {
		// If Docker is also available, Docker takes precedence
		if ctx.Docker.Available && ctx.Runtime != "docker" {
			t.Error("Runtime should be 'docker' when both Docker and Podman are available (Docker takes precedence)")
		}
		// If only Podman is available, it should be selected
		if !ctx.Docker.Available && ctx.Runtime != "podman" {
			t.Error("Runtime should be 'podman' when only Podman is available")
		}
	}
}

// TestDetectPodmanVersion tests that version parsing works correctly
func TestDetectPodmanVersion(t *testing.T) {
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("Podman not available in test environment")
	}

	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	if !ctx.Podman.Available {
		t.Skip("Podman binary is not available")
	}

	// Version should be non-empty if Available is true
	if ctx.Podman.Version == "" {
		t.Error("Podman version should be populated when Available is true")
	}

	// Version should follow semantic versioning pattern (loosely)
	// e.g., "4.7.2", "5.0.0", etc.
	if len(ctx.Podman.Version) < 3 {
		t.Errorf("Podman version seems invalid: %s", ctx.Podman.Version)
	}

	t.Logf("Podman version: %s", ctx.Podman.Version)
}

// TestDetectRuntimePriority tests that Docker is prioritized over Podman
func TestDetectRuntimePriority(t *testing.T) {
	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	// If both are available, Docker should be selected
	if ctx.Docker.Available && ctx.Podman.Available {
		if ctx.Runtime != "docker" {
			t.Error("Docker should be prioritized over Podman when both are available")
		}
		t.Log("Both Docker and Podman available - Docker correctly prioritized")
	}

	// If only Podman is available
	if !ctx.Docker.Available && ctx.Podman.Available {
		if ctx.Runtime != "podman" {
			t.Error("Podman should be selected when Docker is not available")
		}
		t.Log("Only Podman available - correctly selected")
	}

	// If only Docker is available
	if ctx.Docker.Available && !ctx.Podman.Available {
		if ctx.Runtime != "docker" {
			t.Error("Docker should be selected when Podman is not available")
		}
		t.Log("Only Docker available - correctly selected")
	}

	// If neither is available
	if !ctx.Docker.Available && !ctx.Podman.Available {
		if ctx.Runtime != "" {
			t.Errorf("Runtime should be empty when neither Docker nor Podman is available, got: %s", ctx.Runtime)
		}
		t.Log("Neither Docker nor Podman available - runtime correctly empty")
	}
}

// TestDetectPodmanNotAvailable tests behavior when Podman is not installed
func TestDetectPodmanNotAvailable(t *testing.T) {
	// This test documents expected behavior when podman is not installed
	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	if !ctx.Podman.Available {
		// When Podman is not available, all fields should be zero/empty
		if ctx.Podman.Version != "" {
			t.Errorf("Podman.Version should be empty when not available, got: %s", ctx.Podman.Version)
		}
		if ctx.Podman.Running {
			t.Error("Podman.Running should be false when not available")
		}
		t.Log("Podman not available - all fields correctly empty/false")
	} else {
		t.Log("Podman is available on this system")
	}
}
