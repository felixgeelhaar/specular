package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestExecuteInitWritesActivationMarker drives executeInit end-to-end against
// a tempdir target and asserts the activation marker file (the cross-session
// time-to-first-success carrier) was persisted with the expected schema. The
// individual metric instruments are exercised by the telemetry package unit
// tests; this test guards the production-call-site contract that runInit
// delegates marker creation correctly.
func TestExecuteInitWritesActivationMarker(t *testing.T) {
	target := t.TempDir()
	specDir := filepath.Join(target, ".specular")

	// Save and restore the init flags this test mutates.
	origForce, origGovernance, origNoDetect, origProviderSetup, origYes := initForce, initGovernance, initNoDetect, initProviderSetup, initYes
	initForce = true
	initGovernance = "L2"
	initNoDetect = true
	initProviderSetup = false
	initYes = true
	t.Cleanup(func() {
		initForce, initGovernance, initNoDetect, initProviderSetup, initYes = origForce, origGovernance, origNoDetect, origProviderSetup, origYes
	})

	// Drive through rootCmd so the persistent flags (verbose, quiet, format,
	// no-color) NewCommandContext expects are registered.
	rootCmd.SetArgs([]string{"init", "--yes", "--no-detect", "--governance", "L2", target})
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("rootCmd execute init: %v", err)
	}

	marker, err := readActivationMarker(specDir)
	if err != nil {
		t.Fatalf("readActivationMarker: %v", err)
	}
	if marker.Version != activationMarkerVersion {
		t.Errorf("marker.Version = %d, want %d", marker.Version, activationMarkerVersion)
	}
	if marker.StartedAt.IsZero() {
		t.Errorf("marker.StartedAt must be set")
	}
	if marker.InitCompleteAt.IsZero() {
		t.Errorf("marker.InitCompleteAt must be set after successful init")
	}
	if marker.InitCompleteAt.Before(marker.StartedAt) {
		t.Errorf("marker.InitCompleteAt (%v) must be >= StartedAt (%v)",
			marker.InitCompleteAt, marker.StartedAt)
	}

	// Ownership sibling must exist so subsequent recordFirstSuccessIfPending
	// calls treat the directory as owned.
	if _, err := os.Stat(filepath.Join(specDir, activationOwnershipSibling)); err != nil {
		t.Errorf("ownership sibling missing after init: %v", err)
	}
}
