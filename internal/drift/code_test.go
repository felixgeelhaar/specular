package drift

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/pkg/specular/types"
	"github.com/felixgeelhaar/specular/internal/spec"
)

func TestDetectCodeDrift(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()

	// Create test files
	testFile := filepath.Join(tmpDir, "feature_test.go")
	if err := os.WriteFile(testFile, []byte("package main\nfunc TestFeature() {}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	traceFile := filepath.Join(tmpDir, "feature.go")
	if err := os.WriteFile(traceFile, []byte("package main\nfunc Feature() {}"), 0644); err != nil {
		t.Fatalf("Failed to create trace file: %v", err)
	}

	tests := []struct {
		name         string
		spec         *spec.ProductSpec
		lock         *spec.SpecLock
		opts         CodeDriftOptions
		wantFindings int
		wantCodes    []string
	}{
		{
			name: "no drift - all files present",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID:    "feat-001",
						Title: "Test Feature",
						Trace: []string{"feature.go"},
					},
				},
			},
			lock: &spec.SpecLock{
				Features: map[types.FeatureID]spec.LockedFeature{
					types.FeatureID("feat-001"): {
						TestPaths: []string{"feature_test.go"},
					},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 0,
			wantCodes:    []string{},
		},
		{
			name: "missing test file",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID:    "feat-001",
						Title: "Test Feature",
					},
				},
			},
			lock: &spec.SpecLock{
				Features: map[types.FeatureID]spec.LockedFeature{
					types.FeatureID("feat-001"): {
						TestPaths: []string{"missing_test.go"},
					},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 1,
			wantCodes:    []string{"MISSING_TEST"},
		},
		{
			name: "missing trace file",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID:    "feat-001",
						Title: "Test Feature",
						Trace: []string{"missing_trace.go"},
					},
				},
			},
			lock: &spec.SpecLock{
				Features: map[types.FeatureID]spec.LockedFeature{
					types.FeatureID("feat-001"): {},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 1,
			wantCodes:    []string{"MISSING_TRACE"},
		},
		{
			name: "P0 feature without tests",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID:       "feat-001",
						Title:    "Critical Feature",
						Priority: "P0",
					},
				},
			},
			lock: &spec.SpecLock{
				Features: map[types.FeatureID]spec.LockedFeature{
					types.FeatureID("feat-001"): {},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 1,
			wantCodes:    []string{"NO_TESTS"},
		},
		{
			name: "ignored trace file",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID:    "feat-001",
						Title: "Test Feature",
						Trace: []string{"feature.test.go"},
					},
				},
			},
			lock: &spec.SpecLock{
				Features: map[types.FeatureID]spec.LockedFeature{
					types.FeatureID("feat-001"): {},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
				IgnoreGlobs: []string{"*.test.go"},
			},
			wantFindings: 0,
			wantCodes:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := DetectCodeDrift(tt.spec, tt.lock, tt.opts)

			if len(findings) != tt.wantFindings {
				t.Errorf("DetectCodeDrift() found %d findings, want %d", len(findings), tt.wantFindings)
				for _, f := range findings {
					t.Logf("  Finding: %s - %s", f.Code, f.Message)
				}
			}

			// Check specific finding codes
			foundCodes := make(map[string]bool)
			for _, f := range findings {
				foundCodes[f.Code] = true
			}

			for _, wantCode := range tt.wantCodes {
				if !foundCodes[wantCode] {
					t.Errorf("DetectCodeDrift() missing expected finding code: %s", wantCode)
				}
			}
		})
	}
}

func TestCheckFileHashes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name         string
		spec         *spec.ProductSpec
		lock         *spec.SpecLock
		opts         CodeDriftOptions
		wantFindings int
		wantCode     string
	}{
		{
			name: "test file exists",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{ID: "feat-001"},
				},
			},
			lock: &spec.SpecLock{
				Features: map[types.FeatureID]spec.LockedFeature{
					types.FeatureID("feat-001"): {
						TestPaths: []string{"test.go"},
					},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 0,
		},
		{
			name: "test file missing",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{ID: "feat-001"},
				},
			},
			lock: &spec.SpecLock{
				Features: map[types.FeatureID]spec.LockedFeature{
					types.FeatureID("feat-001"): {
						TestPaths: []string{"missing.go"},
					},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 1,
			wantCode:     "MISSING_TEST",
		},
		{
			name: "trace file exists",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID:    "feat-001",
						Trace: []string{"test.go"},
					},
				},
			},
			lock: &spec.SpecLock{
				Features: map[types.FeatureID]spec.LockedFeature{
					types.FeatureID("feat-001"): {},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 0,
		},
		{
			name: "trace file missing",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID:    "feat-001",
						Trace: []string{"missing.go"},
					},
				},
			},
			lock: &spec.SpecLock{
				Features: map[types.FeatureID]spec.LockedFeature{
					types.FeatureID("feat-001"): {},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 1,
			wantCode:     "MISSING_TRACE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := checkFileHashes(tt.spec, tt.lock, tt.opts)

			if len(findings) != tt.wantFindings {
				t.Errorf("checkFileHashes() found %d findings, want %d", len(findings), tt.wantFindings)
			}

			if tt.wantCode != "" && len(findings) > 0 {
				if findings[0].Code != tt.wantCode {
					t.Errorf("checkFileHashes() code = %s, want %s", findings[0].Code, tt.wantCode)
				}
			}
		})
	}
}

func TestCheckAPIImplementations(t *testing.T) {
	tmpDir := t.TempDir()

	// Create API spec file with valid OpenAPI structure
	apiSpecPath := "api/openapi.yaml"
	fullPath := filepath.Join(tmpDir, apiSpecPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("Failed to create api directory: %v", err)
	}
	validSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /api/users:
    get:
      summary: Get users
      responses:
        '200':
          description: Success
`
	if err := os.WriteFile(fullPath, []byte(validSpec), 0644); err != nil {
		t.Fatalf("Failed to create API spec: %v", err)
	}

	tests := []struct {
		name         string
		spec         *spec.ProductSpec
		opts         CodeDriftOptions
		wantFindings int
		wantCode     string
	}{
		{
			name: "no APIs defined",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{ID: "feat-001"},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 0,
		},
		{
			name: "API spec exists",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID: "feat-001",
						API: []spec.API{
							{Path: "/api/users", Method: "GET"},
						},
					},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
				APISpecPath: apiSpecPath,
			},
			wantFindings: 0,
		},
		{
			name: "API spec missing",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID: "feat-001",
						API: []spec.API{
							{Path: "/api/users", Method: "GET"},
						},
					},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
				APISpecPath: "missing/openapi.yaml",
			},
			wantFindings: 1,
			wantCode:     "MISSING_API_SPEC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := checkAPIImplementations(tt.spec, tt.opts)

			if len(findings) != tt.wantFindings {
				t.Errorf("checkAPIImplementations() found %d findings, want %d", len(findings), tt.wantFindings)
			}

			if tt.wantCode != "" && len(findings) > 0 {
				if findings[0].Code != tt.wantCode {
					t.Errorf("checkAPIImplementations() code = %s, want %s", findings[0].Code, tt.wantCode)
				}
			}
		})
	}
}

func TestCheckTestCoverage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file matching feature name
	testFile := filepath.Join(tmpDir, "user_management_test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name         string
		spec         *spec.ProductSpec
		opts         CodeDriftOptions
		wantFindings int
		wantSeverity string
	}{
		{
			name: "P0 feature without tests",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID:       "feat-001",
						Title:    "Authentication",
						Priority: "P0",
					},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 1,
			wantSeverity: "error",
		},
		{
			name: "P1 feature without tests",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID:       "feat-001",
						Title:    "Notifications",
						Priority: "P1",
					},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 1,
			wantSeverity: "warning",
		},
		{
			name: "P2 feature without tests",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID:       "feat-001",
						Title:    "Theme",
						Priority: "P2",
					},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 0,
		},
		{
			name: "feature with matching test file",
			spec: &spec.ProductSpec{
				Features: []spec.Feature{
					{
						ID:       "feat-001",
						Title:    "User Management",
						Priority: "P0",
					},
				},
			},
			opts: CodeDriftOptions{
				ProjectRoot: tmpDir,
			},
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := checkTestCoverage(tt.spec, tt.opts)

			if len(findings) != tt.wantFindings {
				t.Errorf("checkTestCoverage() found %d findings, want %d", len(findings), tt.wantFindings)
			}

			if tt.wantSeverity != "" && len(findings) > 0 {
				if findings[0].Severity != tt.wantSeverity {
					t.Errorf("checkTestCoverage() severity = %s, want %s", findings[0].Severity, tt.wantSeverity)
				}
			}
		})
	}
}

func TestPathExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "file exists",
			path: testFile,
			want: true,
		},
		{
			name: "file does not exist",
			path: filepath.Join(tmpDir, "missing.txt"),
			want: false,
		},
		{
			name: "directory exists",
			path: tmpDir,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathExists(tt.path); got != tt.want {
				t.Errorf("pathExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content for hashing")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
		wantLen int
	}{
		{
			name:    "hash existing file",
			path:    testFile,
			wantErr: false,
			wantLen: 64, // SHA-256 hex string length
		},
		{
			name:    "hash non-existent file",
			path:    filepath.Join(tmpDir, "missing.txt"),
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hashFile(tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("hashFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(hash) != tt.wantLen {
				t.Errorf("hashFile() hash length = %d, want %d", len(hash), tt.wantLen)
			}

			// Test consistency - same file should produce same hash
			if !tt.wantErr {
				hash2, err := hashFile(tt.path)
				if err != nil {
					t.Errorf("hashFile() second call failed: %v", err)
				}
				if hash != hash2 {
					t.Errorf("hashFile() inconsistent hash: %s != %s", hash, hash2)
				}
			}
		})
	}
}

func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		ignoreGlobs []string
		want        bool
	}{
		{
			name:        "no ignore patterns",
			path:        "file.go",
			ignoreGlobs: []string{},
			want:        false,
		},
		{
			name:        "match test files",
			path:        "feature_test.go",
			ignoreGlobs: []string{"*_test.go"},
			want:        true,
		},
		{
			name:        "match vendor directory",
			path:        "vendor/package/file.go",
			ignoreGlobs: []string{"vendor"},
			want:        false, // Pattern matches basename only
		},
		{
			name:        "match multiple patterns",
			path:        "file.tmp",
			ignoreGlobs: []string{"*.log", "*.tmp", "*.bak"},
			want:        true,
		},
		{
			name:        "no match",
			path:        "main.go",
			ignoreGlobs: []string{"*_test.go", "*.md"},
			want:        false,
		},
		{
			name:        "match with path separator",
			path:        "internal/feature_test.go",
			ignoreGlobs: []string{"*_test.go"},
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldIgnore(tt.path, tt.ignoreGlobs); got != tt.want {
				t.Errorf("shouldIgnore() = %v, want %v", got, tt.want)
			}
		})
	}
}
