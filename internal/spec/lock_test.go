package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateSpecLock(t *testing.T) {
	tests := []struct {
		name     string
		spec     ProductSpec
		version  string
		validate func(*testing.T, *SpecLock)
	}{
		{
			name: "spec with multiple features",
			spec: ProductSpec{
				Product: "TestProduct",
				Features: []Feature{
					{
						ID:       "feat-001",
						Title:    "Feature One",
						Desc:     "First feature",
						Priority: "P0",
						Success:  []string{"Success 1"},
						Trace:    []string{"PRD-001"},
					},
					{
						ID:       "feat-002",
						Title:    "Feature Two",
						Desc:     "Second feature",
						Priority: "P1",
						Success:  []string{"Success 2"},
						Trace:    []string{"PRD-002"},
					},
				},
			},
			version: "1.0.0",
			validate: func(t *testing.T, lock *SpecLock) {
				if lock.Version != "1.0.0" {
					t.Errorf("Version = %v, want 1.0.0", lock.Version)
				}
				if len(lock.Features) != 2 {
					t.Errorf("Features length = %d, want 2", len(lock.Features))
				}

				// Check feat-001
				if locked, ok := lock.Features["feat-001"]; !ok {
					t.Error("feat-001 not found in lock")
				} else {
					if locked.Hash == "" {
						t.Error("feat-001 Hash is empty")
					}
					if len(locked.Hash) != 64 {
						t.Errorf("feat-001 Hash length = %d, want 64", len(locked.Hash))
					}
					if locked.OpenAPIPath != ".specular/openapi/feat-001.yaml" {
						t.Errorf("feat-001 OpenAPIPath = %v, want .specular/openapi/feat-001.yaml", locked.OpenAPIPath)
					}
					if len(locked.TestPaths) != 1 {
						t.Errorf("feat-001 TestPaths length = %d, want 1", len(locked.TestPaths))
					}
					if locked.TestPaths[0] != ".specular/tests/feat-001_test.go" {
						t.Errorf("feat-001 TestPath = %v, want .specular/tests/feat-001_test.go", locked.TestPaths[0])
					}
				}

				// Check feat-002
				if locked, ok := lock.Features["feat-002"]; !ok {
					t.Error("feat-002 not found in lock")
				} else {
					if locked.Hash == "" {
						t.Error("feat-002 Hash is empty")
					}
				}
			},
		},
		{
			name: "spec with no features",
			spec: ProductSpec{
				Product:  "EmptyProduct",
				Features: []Feature{},
			},
			version: "0.1.0",
			validate: func(t *testing.T, lock *SpecLock) {
				if lock.Version != "0.1.0" {
					t.Errorf("Version = %v, want 0.1.0", lock.Version)
				}
				if len(lock.Features) != 0 {
					t.Errorf("Features length = %d, want 0", len(lock.Features))
				}
			},
		},
		{
			name: "spec with feature with API",
			spec: ProductSpec{
				Product: "APIProduct",
				Features: []Feature{
					{
						ID:       "feat-api",
						Title:    "API Feature",
						Desc:     "Feature with API",
						Priority: "P0",
						API: []API{
							{
								Method:   "GET",
								Path:     "/api/test",
								Request:  "TestRequest",
								Response: "TestResponse",
							},
						},
						Success: []string{"API works"},
						Trace:   []string{"PRD-API"},
					},
				},
			},
			version: "2.0.0",
			validate: func(t *testing.T, lock *SpecLock) {
				if len(lock.Features) != 1 {
					t.Errorf("Features length = %d, want 1", len(lock.Features))
				}
				if locked, ok := lock.Features["feat-api"]; !ok {
					t.Error("feat-api not found in lock")
				} else {
					if locked.Hash == "" {
						t.Error("feat-api Hash is empty")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lock, err := GenerateSpecLock(tt.spec, tt.version)
			if err != nil {
				t.Fatalf("GenerateSpecLock() error = %v", err)
			}

			if lock == nil {
				t.Fatal("GenerateSpecLock() returned nil lock")
			}

			if tt.validate != nil {
				tt.validate(t, lock)
			}
		})
	}
}

func TestGenerateSpecLock_HashDeterminism(t *testing.T) {
	spec := ProductSpec{
		Product: "TestProduct",
		Features: []Feature{
			{
				ID:       "feat-001",
				Title:    "Test Feature",
				Desc:     "Test description",
				Priority: "P0",
				Success:  []string{"Success"},
				Trace:    []string{"PRD-001"},
			},
		},
	}

	// Generate lock twice
	lock1, err := GenerateSpecLock(spec, "1.0.0")
	if err != nil {
		t.Fatalf("GenerateSpecLock() first call error = %v", err)
	}

	lock2, err := GenerateSpecLock(spec, "1.0.0")
	if err != nil {
		t.Fatalf("GenerateSpecLock() second call error = %v", err)
	}

	// Hashes should be identical
	hash1 := lock1.Features["feat-001"].Hash
	hash2 := lock2.Features["feat-001"].Hash

	if hash1 != hash2 {
		t.Errorf("GenerateSpecLock() not deterministic: %s != %s", hash1, hash2)
	}
}

func TestSaveSpecLock(t *testing.T) {
	tests := []struct {
		name    string
		lock    *SpecLock
		wantErr bool
	}{
		{
			name: "complete lock",
			lock: &SpecLock{
				Version: "1.0.0",
				Features: map[string]LockedFeature{
					"feat-001": {
						Hash:        "abc123",
						OpenAPIPath: ".specular/openapi/feat-001.yaml",
						TestPaths:   []string{".specular/tests/feat-001_test.go"},
					},
					"feat-002": {
						Hash:        "def456",
						OpenAPIPath: ".specular/openapi/feat-002.yaml",
						TestPaths:   []string{".specular/tests/feat-002_test.go"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty lock",
			lock: &SpecLock{
				Version:  "0.1.0",
				Features: make(map[string]LockedFeature),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			lockFile := filepath.Join(tmpDir, "spec.lock.json")

			err := SaveSpecLock(tt.lock, lockFile)

			if tt.wantErr {
				if err == nil {
					t.Error("SaveSpecLock() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("SaveSpecLock() unexpected error = %v", err)
			}

			// Verify file was created
			if _, err := os.Stat(lockFile); os.IsNotExist(err) {
				t.Error("SaveSpecLock() did not create file")
			}

			// Verify JSON format
			data, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read saved lock file: %v", err)
			}

			var loaded SpecLock
			if err := json.Unmarshal(data, &loaded); err != nil {
				t.Errorf("SaveSpecLock() did not create valid JSON: %v", err)
			}

			// Verify content
			if loaded.Version != tt.lock.Version {
				t.Errorf("Loaded Version = %v, want %v", loaded.Version, tt.lock.Version)
			}
			if len(loaded.Features) != len(tt.lock.Features) {
				t.Errorf("Loaded Features length = %d, want %d", len(loaded.Features), len(tt.lock.Features))
			}
		})
	}
}

func TestSaveSpecLock_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	deepPath := filepath.Join(tmpDir, "level1", "level2", "spec.lock.json")

	lock := &SpecLock{
		Version: "1.0.0",
		Features: map[string]LockedFeature{
			"feat-001": {
				Hash:        "abc123",
				OpenAPIPath: ".specular/openapi/feat-001.yaml",
				TestPaths:   []string{".specular/tests/feat-001_test.go"},
			},
		},
	}

	err := SaveSpecLock(lock, deepPath)
	if err != nil {
		t.Fatalf("SaveSpecLock() failed to create nested directories: %v", err)
	}

	// Verify all directories were created
	if _, err := os.Stat(filepath.Dir(deepPath)); os.IsNotExist(err) {
		t.Error("SaveSpecLock() did not create nested directories")
	}

	// Verify file exists
	if _, err := os.Stat(deepPath); os.IsNotExist(err) {
		t.Error("SaveSpecLock() did not create file in nested directories")
	}
}

func TestLoadSpecLock(t *testing.T) {
	tests := []struct {
		name        string
		lockContent string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *SpecLock)
	}{
		{
			name: "valid complete lock",
			lockContent: `{
  "version": "1.0.0",
  "features": {
    "feat-001": {
      "hash": "abc123def456",
      "openapi_path": ".specular/openapi/feat-001.yaml",
      "test_paths": [
        ".specular/tests/feat-001_test.go"
      ]
    },
    "feat-002": {
      "hash": "789xyz",
      "openapi_path": ".specular/openapi/feat-002.yaml",
      "test_paths": [
        ".specular/tests/feat-002_test.go",
        ".specular/tests/feat-002_integration_test.go"
      ]
    }
  }
}`,
			wantErr: false,
			validate: func(t *testing.T, lock *SpecLock) {
				if lock.Version != "1.0.0" {
					t.Errorf("Version = %v, want 1.0.0", lock.Version)
				}
				if len(lock.Features) != 2 {
					t.Errorf("Features length = %d, want 2", len(lock.Features))
				}

				if locked, ok := lock.Features["feat-001"]; !ok {
					t.Error("feat-001 not found")
				} else {
					if locked.Hash != "abc123def456" {
						t.Errorf("feat-001 Hash = %v, want abc123def456", locked.Hash)
					}
					if locked.OpenAPIPath != ".specular/openapi/feat-001.yaml" {
						t.Errorf("feat-001 OpenAPIPath = %v", locked.OpenAPIPath)
					}
					if len(locked.TestPaths) != 1 {
						t.Errorf("feat-001 TestPaths length = %d, want 1", len(locked.TestPaths))
					}
				}

				if locked, ok := lock.Features["feat-002"]; !ok {
					t.Error("feat-002 not found")
				} else {
					if len(locked.TestPaths) != 2 {
						t.Errorf("feat-002 TestPaths length = %d, want 2", len(locked.TestPaths))
					}
				}
			},
		},
		{
			name: "minimal lock",
			lockContent: `{
  "version": "0.1.0",
  "features": {}
}`,
			wantErr: false,
			validate: func(t *testing.T, lock *SpecLock) {
				if lock.Version != "0.1.0" {
					t.Errorf("Version = %v, want 0.1.0", lock.Version)
				}
				if len(lock.Features) != 0 {
					t.Errorf("Features length = %d, want 0", len(lock.Features))
				}
			},
		},
		{
			name:        "invalid json",
			lockContent: `{invalid json`,
			wantErr:     true,
			errContains: "unmarshal spec lock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			lockFile := filepath.Join(tmpDir, "spec.lock.json")

			err := os.WriteFile(lockFile, []byte(tt.lockContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test lock file: %v", err)
			}

			lock, err := LoadSpecLock(lockFile)

			if tt.wantErr {
				if err == nil {
					t.Error("LoadSpecLock() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("LoadSpecLock() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadSpecLock() unexpected error = %v", err)
			}

			if lock == nil {
				t.Fatal("LoadSpecLock() returned nil lock")
			}

			if tt.validate != nil {
				tt.validate(t, lock)
			}
		})
	}
}

func TestLoadSpecLock_FileNotFound(t *testing.T) {
	_, err := LoadSpecLock("/nonexistent/path/spec.lock.json")
	if err == nil {
		t.Error("LoadSpecLock() expected error for nonexistent file, got nil")
	}
	if !contains(err.Error(), "read spec lock") {
		t.Errorf("LoadSpecLock() error = %v, want error containing 'read spec lock'", err)
	}
}

func TestSpecLock_RoundTrip(t *testing.T) {
	// Create a spec
	spec := ProductSpec{
		Product: "TestProduct",
		Features: []Feature{
			{
				ID:       "feat-001",
				Title:    "Feature One",
				Desc:     "First feature",
				Priority: "P0",
				Success:  []string{"Success"},
				Trace:    []string{"PRD-001"},
			},
		},
	}

	// Generate lock
	lock1, err := GenerateSpecLock(spec, "1.0.0")
	if err != nil {
		t.Fatalf("GenerateSpecLock() error = %v", err)
	}

	// Save lock
	tmpDir := t.TempDir()
	lockFile := filepath.Join(tmpDir, "spec.lock.json")
	err = SaveSpecLock(lock1, lockFile)
	if err != nil {
		t.Fatalf("SaveSpecLock() error = %v", err)
	}

	// Load lock
	lock2, err := LoadSpecLock(lockFile)
	if err != nil {
		t.Fatalf("LoadSpecLock() error = %v", err)
	}

	// Verify round-trip
	if lock2.Version != lock1.Version {
		t.Errorf("Round-trip Version = %v, want %v", lock2.Version, lock1.Version)
	}

	if len(lock2.Features) != len(lock1.Features) {
		t.Errorf("Round-trip Features length = %d, want %d", len(lock2.Features), len(lock1.Features))
	}

	hash1 := lock1.Features["feat-001"].Hash
	hash2 := lock2.Features["feat-001"].Hash
	if hash1 != hash2 {
		t.Errorf("Round-trip Hash = %v, want %v", hash2, hash1)
	}
}


func TestSaveSpecLock_WriteError(t *testing.T) {
	lock := &SpecLock{
		Version: "1.0.0",
		Features: map[string]LockedFeature{
			"feat-001": {
				Hash:        "test-hash",
				OpenAPIPath: ".specular/openapi/feat-001.yaml",
				TestPaths:   []string{".specular/tests/feat-001_test.go"},
			},
		},
	}

	// Try to write to an invalid path
	err := SaveSpecLock(lock, "/nonexistent/directory/spec.lock")
	if err == nil {
		t.Error("SaveSpecLock() expected error for invalid path, got nil")
	}
}

func TestSaveSpecLock_ReadOnlyFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "spec.lock.json")

	// Create a read-only file at the target path
	if err := os.WriteFile(lockPath, []byte("readonly content"), 0444); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	lock := &SpecLock{
		Version: "1.0.0",
		Features: map[string]LockedFeature{
			"feat-001": {
				Hash:        "test-hash",
				OpenAPIPath: ".specular/openapi/feat-001.yaml",
				TestPaths:   []string{".specular/tests/feat-001_test.go"},
			},
		},
	}

	// Try to overwrite read-only file
	err := SaveSpecLock(lock, lockPath)
	if err == nil {
		t.Error("SaveSpecLock() expected error when writing to read-only file, got nil")
	}
	if !contains(err.Error(), "write spec lock") {
		t.Errorf("SaveSpecLock() error = %v, want error containing 'write spec lock'", err)
	}
}
