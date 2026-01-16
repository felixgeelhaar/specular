package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "policy.yaml")

	expected := DefaultPolicy()
	expected.Execution.Docker.Network = "bridge"

	if err := SavePolicy(expected, policyPath); err != nil {
		t.Fatalf("SavePolicy() failed: %v", err)
	}

	loaded, err := LoadPolicy(policyPath)
	if err != nil {
		t.Fatalf("LoadPolicy() error = %v", err)
	}

	if loaded.Execution.Docker.Network != "bridge" {
		t.Errorf("Execution.Docker.Network = %q, want %q", loaded.Execution.Docker.Network, "bridge")
	}
	if !loaded.Tests.RequirePass {
		t.Errorf("Tests.RequirePass expected true")
	}
	if loaded.Tests.MinCoverage != expected.Tests.MinCoverage {
		t.Errorf("Tests.MinCoverage = %f, want %f", loaded.Tests.MinCoverage, expected.Tests.MinCoverage)
	}
}

func TestLoadPolicyInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "broken.yaml")

	if err := os.WriteFile(policyPath, []byte("{invalid"), 0o600); err != nil {
		t.Fatalf("failed to write invalid policy: %v", err)
	}

	if _, err := LoadPolicy(policyPath); err == nil {
		t.Fatal("expected error loading invalid policy, got nil")
	}
}
