package spec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

func TestFileSpecRepository_SaveLoad(t *testing.T) {
	repo := NewFileSpecRepository()
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "spec.yaml")

	featureID, err := types.NewFeatureID("feature-one")
	if err != nil {
		t.Fatalf("unexpected error creating feature ID: %v", err)
	}

	spec := &ProductSpec{
		Product:    "Test Product",
		Goals:      []string{"Deliver MVP"},
		Acceptance: []string{"Spec passes validation"},
		Features: []Feature{
			{
				ID:       featureID,
				Title:    "Awesome feature",
				Desc:     "Make things awesome",
				Priority: types.PriorityP1,
				Success:  []string{"Feature works end-to-end"},
				Trace:    []string{"trace-1"},
				API: []API{
					{
						Method:   "GET",
						Path:     "/health",
						Request:  "",
						Response: "ok",
					},
				},
			},
		},
		Milestones: []Milestone{
			{
				ID:         "m1",
				Name:       "Launch",
				FeatureIDs: []types.FeatureID{featureID},
			},
		},
	}

	if err := repo.Save(spec, specPath); err != nil {
		t.Fatalf("failed to save spec: %v", err)
	}

	loaded, err := repo.Load(specPath)
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	if loaded.Product != spec.Product {
		t.Errorf("expected product %q, got %q", spec.Product, loaded.Product)
	}
	if len(loaded.Features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(loaded.Features))
	}
	if loaded.Features[0].Title != spec.Features[0].Title {
		t.Errorf("unexpected feature title: %q", loaded.Features[0].Title)
	}
	if len(loaded.Milestones) != 1 {
		t.Fatalf("expected 1 milestone, got %d", len(loaded.Milestones))
	}
}

func TestFileSpecRepository_LoadInvalidYAML(t *testing.T) {
	repo := NewFileSpecRepository()
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(specPath, []byte("{invalid"), 0o600); err != nil {
		t.Fatalf("failed to write invalid file: %v", err)
	}

	if _, err := repo.Load(specPath); err == nil {
		t.Fatal("expected error loading invalid spec, got nil")
	}
}
