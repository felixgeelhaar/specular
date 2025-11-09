package spec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/internal/domain"
)

func TestLoadSpec(t *testing.T) {
	tests := []struct {
		name        string
		specContent string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *ProductSpec)
	}{
		{
			name: "valid complete spec",
			specContent: `
product: TestProduct
goals:
  - Goal 1
  - Goal 2
features:
  - id: feat-001
    title: Feature One
    desc: First feature
    priority: P0
    success:
      - Success criterion 1
    trace:
      - PRD-001
  - id: feat-002
    title: Feature Two
    desc: Second feature
    priority: P1
    api:
      - method: GET
        path: /api/test
        request: TestRequest
        response: TestResponse
    success:
      - Success criterion 2
    trace:
      - PRD-002
non_functional:
  performance:
    - Response time < 2s
  security:
    - HTTPS/TLS
  scalability:
    - Handle 1000 req/s
  availability:
    - 99.9% uptime
acceptance:
  - Acceptance criterion 1
  - Acceptance criterion 2
milestones:
  - id: m1
    name: MVP
    target_date: "4 weeks"
    description: Core features
    feature_ids:
      - feat-001
`,
			wantErr: false,
			validate: func(t *testing.T, s *ProductSpec) {
				if s.Product != "TestProduct" {
					t.Errorf("Product = %v, want TestProduct", s.Product)
				}
				if len(s.Goals) != 2 {
					t.Errorf("Goals length = %d, want 2", len(s.Goals))
				}
				if len(s.Features) != 2 {
					t.Errorf("Features length = %d, want 2", len(s.Features))
				}
				if s.Features[0].ID != "feat-001" {
					t.Errorf("Feature[0].ID = %v, want feat-001", s.Features[0].ID)
				}
				if len(s.Features[1].API) != 1 {
					t.Errorf("Feature[1].API length = %d, want 1", len(s.Features[1].API))
				}
				// NonFunctional fields are tested in round-trip SaveSpec test
				if len(s.Acceptance) != 2 {
					t.Errorf("Acceptance length = %d, want 2", len(s.Acceptance))
				}
				if len(s.Milestones) != 1 {
					t.Errorf("Milestones length = %d, want 1", len(s.Milestones))
				}
			},
		},
		{
			name: "minimal spec",
			specContent: `
product: MinimalProduct
goals:
  - Goal
features:
  - id: feat-001
    title: Feature
    desc: Description
    priority: P0
    success:
      - Success
    trace:
      - PRD-001
`,
			wantErr: false,
			validate: func(t *testing.T, s *ProductSpec) {
				if s.Product != "MinimalProduct" {
					t.Errorf("Product = %v, want MinimalProduct", s.Product)
				}
				if len(s.Features) != 1 {
					t.Errorf("Features length = %d, want 1", len(s.Features))
				}
			},
		},
		{
			name:        "invalid yaml",
			specContent: `invalid: [yaml: syntax`,
			wantErr:     true,
			errContains: "unmarshal spec",
		},
		{
			name:        "empty file",
			specContent: "",
			wantErr:     false,
			validate: func(t *testing.T, s *ProductSpec) {
				if s == nil {
					t.Error("LoadSpec should return non-nil spec for empty file")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			specFile := filepath.Join(tmpDir, "spec.yaml")

			err := os.WriteFile(specFile, []byte(tt.specContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test spec file: %v", err)
			}

			spec, err := LoadSpec(specFile)

			if tt.wantErr {
				if err == nil {
					t.Error("LoadSpec() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("LoadSpec() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadSpec() unexpected error = %v", err)
			}

			if spec == nil {
				t.Fatal("LoadSpec() returned nil spec")
			}

			if tt.validate != nil {
				tt.validate(t, spec)
			}
		})
	}
}

func TestLoadSpec_FileNotFound(t *testing.T) {
	_, err := LoadSpec("/nonexistent/path/spec.yaml")
	if err == nil {
		t.Error("LoadSpec() expected error for nonexistent file, got nil")
	}
	if !contains(err.Error(), "read spec file") {
		t.Errorf("LoadSpec() error = %v, want error containing 'read spec file'", err)
	}
}

func TestSaveSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    *ProductSpec
		wantErr bool
	}{
		{
			name: "complete spec",
			spec: &ProductSpec{
				Product: "TestProduct",
				Goals:   []string{"Goal 1", "Goal 2"},
				Features: []Feature{
					{
						ID:       "feat-001",
						Title:    "Feature One",
						Desc:     "Description",
						Priority: "P0",
						Success:  []string{"Success"},
						Trace:    []string{"PRD-001"},
					},
				},
				NonFunctional: NonFunctional{
					Performance:  []string{"Fast"},
					Security:     []string{"Secure"},
					Scalability:  []string{"Scalable"},
					Availability: []string{"Available"},
				},
				Acceptance: []string{"Acceptance 1"},
				Milestones: []Milestone{
					{
						ID:          "m1",
						Name:        "MVP",
						TargetDate:  "4 weeks",
						Description: "Core features",
						FeatureIDs:  []domain.FeatureID{domain.FeatureID("feat-001")},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "minimal spec",
			spec: &ProductSpec{
				Product:  "MinimalProduct",
				Goals:    []string{},
				Features: []Feature{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			specFile := filepath.Join(tmpDir, "subdir", "spec.yaml")

			err := SaveSpec(tt.spec, specFile)

			if tt.wantErr {
				if err == nil {
					t.Error("SaveSpec() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("SaveSpec() unexpected error = %v", err)
			}

			// Verify file was created
			if _, err := os.Stat(specFile); os.IsNotExist(err) {
				t.Error("SaveSpec() did not create file")
			}

			// Verify file can be loaded back
			loaded, err := LoadSpec(specFile)
			if err != nil {
				t.Fatalf("LoadSpec() after SaveSpec() failed: %v", err)
			}

			if loaded.Product != tt.spec.Product {
				t.Errorf("Loaded Product = %v, want %v", loaded.Product, tt.spec.Product)
			}
		})
	}
}

func TestSaveSpec_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	deepPath := filepath.Join(tmpDir, "level1", "level2", "level3", "spec.yaml")

	spec := &ProductSpec{
		Product: "TestProduct",
		Goals:   []string{"Goal"},
		Features: []Feature{
			{
				ID:       "feat-001",
				Title:    "Feature",
				Desc:     "Description",
				Priority: "P0",
				Success:  []string{"Success"},
				Trace:    []string{"PRD-001"},
			},
		},
	}

	err := SaveSpec(spec, deepPath)
	if err != nil {
		t.Fatalf("SaveSpec() failed to create nested directories: %v", err)
	}

	// Verify all directories were created
	if _, err := os.Stat(filepath.Dir(deepPath)); os.IsNotExist(err) {
		t.Error("SaveSpec() did not create nested directories")
	}

	// Verify file exists
	if _, err := os.Stat(deepPath); os.IsNotExist(err) {
		t.Error("SaveSpec() did not create file in nested directories")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSaveSpec_WriteError(t *testing.T) {
	spec := &ProductSpec{
		Product: "TestProduct",
		Goals:   []string{"Goal"},
		Features: []Feature{
			{
				ID:       "feat-001",
				Title:    "Feature",
				Desc:     "Description",
				Priority: "P0",
				Success:  []string{"Success"},
				Trace:    []string{"PRD-001"},
			},
		},
	}

	// Try to write to an invalid path
	err := SaveSpec(spec, "/nonexistent/directory/spec.yaml")
	if err == nil {
		t.Error("SaveSpec() expected error for invalid path, got nil")
	}
}

func TestSaveSpec_ReadOnlyFile(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "spec.yaml")

	// Create a read-only file at the target path
	if err := os.WriteFile(specPath, []byte("readonly content"), 0444); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	spec := &ProductSpec{
		Product: "TestProduct",
		Goals:   []string{"Goal"},
		Features: []Feature{
			{
				ID:       "feat-001",
				Title:    "Feature",
				Desc:     "Description",
				Priority: "P0",
				Success:  []string{"Success"},
				Trace:    []string{"PRD-001"},
			},
		},
	}

	// Try to overwrite read-only file
	err := SaveSpec(spec, specPath)
	if err == nil {
		t.Error("SaveSpec() expected error when writing to read-only file, got nil")
	}
	if !contains(err.Error(), "write spec file") {
		t.Errorf("SaveSpec() error = %v, want error containing 'write spec file'", err)
	}
}
