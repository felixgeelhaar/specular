package drift

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/internal/spec"
)

// createTestOpenAPISpec creates a test OpenAPI spec file
func createTestOpenAPISpec(t *testing.T, dir string) string {
	specContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /api/users:
    get:
      summary: List users
      responses:
        '200':
          description: Success
    post:
      summary: Create user
      responses:
        '201':
          description: Created
  /api/users/{id}:
    get:
      summary: Get user
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
    put:
      summary: Update user
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
    delete:
      summary: Delete user
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '204':
          description: Deleted
  /api/products:
    get:
      summary: List products
      responses:
        '200':
          description: Success
`

	specPath := filepath.Join(dir, "openapi.yaml")
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("Failed to create test OpenAPI spec: %v", err)
	}

	return specPath
}

func TestNewOpenAPIValidator(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := createTestOpenAPISpec(t, tmpDir)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid spec",
			path:    specPath,
			wantErr: false,
		},
		{
			name:    "missing spec",
			path:    filepath.Join(tmpDir, "missing.yaml"),
			wantErr: true,
		},
		{
			name:    "invalid spec",
			path:    filepath.Join(tmpDir, "invalid.yaml"),
			wantErr: true,
		},
	}

	// Create invalid spec
	invalidSpec := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidSpec, []byte("invalid: yaml: content:"), 0644); err != nil {
		t.Fatalf("Failed to create invalid spec: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewOpenAPIValidator(tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenAPIValidator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && validator == nil {
				t.Error("NewOpenAPIValidator() returned nil validator")
			}

			if !tt.wantErr && validator.spec == nil {
				t.Error("NewOpenAPIValidator() validator has nil spec")
			}
		})
	}
}

func TestValidateEndpoints(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := createTestOpenAPISpec(t, tmpDir)

	validator, err := NewOpenAPIValidator(specPath)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name          string
		features      []spec.Feature
		wantFindings  int
		wantCodes     []string
	}{
		{
			name: "all endpoints exist",
			features: []spec.Feature{
				{
					ID: "feat-001",
					API: []spec.API{
						{Path: "/api/users", Method: "GET"},
						{Path: "/api/users", Method: "POST"},
					},
				},
			},
			wantFindings: 0,
			wantCodes:    []string{},
		},
		{
			name: "missing path",
			features: []spec.Feature{
				{
					ID: "feat-001",
					API: []spec.API{
						{Path: "/api/missing", Method: "GET"},
					},
				},
			},
			wantFindings: 1,
			wantCodes:    []string{"MISSING_API_PATH"},
		},
		{
			name: "missing method",
			features: []spec.Feature{
				{
					ID: "feat-001",
					API: []spec.API{
						{Path: "/api/users", Method: "PATCH"},
					},
				},
			},
			wantFindings: 1,
			wantCodes:    []string{"MISSING_API_METHOD"},
		},
		{
			name: "path with parameters",
			features: []spec.Feature{
				{
					ID: "feat-001",
					API: []spec.API{
						{Path: "/api/users/{id}", Method: "GET"},
						{Path: "/api/users/{userId}", Method: "PUT"}, // Different param name
					},
				},
			},
			wantFindings: 0,
			wantCodes:    []string{},
		},
		{
			name: "no APIs",
			features: []spec.Feature{
				{
					ID:  "feat-001",
					API: []spec.API{},
				},
			},
			wantFindings: 0,
			wantCodes:    []string{},
		},
		{
			name: "multiple missing endpoints",
			features: []spec.Feature{
				{
					ID: "feat-001",
					API: []spec.API{
						{Path: "/api/missing1", Method: "GET"},
						{Path: "/api/missing2", Method: "POST"},
					},
				},
			},
			wantFindings: 2,
			wantCodes:    []string{"MISSING_API_PATH", "MISSING_API_PATH"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := validator.ValidateEndpoints(tt.features)

			if len(findings) != tt.wantFindings {
				t.Errorf("ValidateEndpoints() found %d findings, want %d", len(findings), tt.wantFindings)
				for _, f := range findings {
					t.Logf("  Finding: %s - %s", f.Code, f.Message)
				}
			}

			// Check specific finding codes
			foundCodes := make(map[string]int)
			for _, f := range findings {
				foundCodes[f.Code]++
			}

			wantCodeCounts := make(map[string]int)
			for _, code := range tt.wantCodes {
				wantCodeCounts[code]++
			}

			for code, wantCount := range wantCodeCounts {
				if gotCount := foundCodes[code]; gotCount != wantCount {
					t.Errorf("ValidateEndpoints() found %d occurrences of %s, want %d", gotCount, code, wantCount)
				}
			}
		})
	}
}

func TestGetEndpointSummary(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := createTestOpenAPISpec(t, tmpDir)

	validator, err := NewOpenAPIValidator(specPath)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	summary := validator.GetEndpointSummary()

	expectedPaths := map[string][]string{
		"/api/users":       {"GET", "POST"},
		"/api/users/{id}":  {"GET", "PUT", "DELETE"},
		"/api/products":    {"GET"},
	}

	for path, expectedMethods := range expectedPaths {
		methods, exists := summary[path]
		if !exists {
			t.Errorf("GetEndpointSummary() missing path: %s", path)
			continue
		}

		if len(methods) != len(expectedMethods) {
			t.Errorf("GetEndpointSummary() path %s has %d methods, want %d", path, len(methods), len(expectedMethods))
		}

		// Check each expected method exists
		methodSet := make(map[string]bool)
		for _, m := range methods {
			methodSet[m] = true
		}

		for _, expectedMethod := range expectedMethods {
			if !methodSet[expectedMethod] {
				t.Errorf("GetEndpointSummary() path %s missing method: %s", path, expectedMethod)
			}
		}
	}

	// Check total number of paths
	if len(summary) != len(expectedPaths) {
		t.Errorf("GetEndpointSummary() found %d paths, want %d", len(summary), len(expectedPaths))
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		want  string
	}{
		{
			name: "simple path",
			path: "/api/users",
			want: "/api/users",
		},
		{
			name: "path without leading slash",
			path: "api/users",
			want: "/api/users",
		},
		{
			name: "path with trailing slash",
			path: "/api/users/",
			want: "/api/users",
		},
		{
			name: "root path",
			path: "/",
			want: "/",
		},
		{
			name: "path with query params",
			path: "/api/users?limit=10",
			want: "/api/users",
		},
		{
			name: "path with multiple query params",
			path: "/api/users?limit=10&offset=5",
			want: "/api/users",
		},
		{
			name: "path with parameters",
			path: "/api/users/{id}",
			want: "/api/users/{id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePath(tt.path)
			if got != tt.want {
				t.Errorf("normalizePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAPISpec(t *testing.T) {
	tmpDir := t.TempDir()
	_ = createTestOpenAPISpec(t, tmpDir) // Create test spec
	relativeSpecPath := "openapi.yaml"

	tests := []struct {
		name         string
		specPath     string
		projectRoot  string
		features     []spec.Feature
		wantFindings int
		wantCodes    []string
	}{
		{
			name:        "valid spec with matching endpoints",
			specPath:    relativeSpecPath,
			projectRoot: tmpDir,
			features: []spec.Feature{
				{
					ID: "feat-001",
					API: []spec.API{
						{Path: "/api/users", Method: "GET"},
					},
				},
			},
			wantFindings: 0,
			wantCodes:    []string{},
		},
		{
			name:        "missing spec file",
			specPath:    "missing.yaml",
			projectRoot: tmpDir,
			features: []spec.Feature{
				{
					ID: "feat-001",
					API: []spec.API{
						{Path: "/api/users", Method: "GET"},
					},
				},
			},
			wantFindings: 1,
			wantCodes:    []string{"MISSING_API_SPEC"},
		},
		{
			name:        "invalid spec file",
			specPath:    "invalid.yaml",
			projectRoot: tmpDir,
			features: []spec.Feature{
				{
					ID: "feat-001",
					API: []spec.API{
						{Path: "/api/users", Method: "GET"},
					},
				},
			},
			wantFindings: 1,
			wantCodes:    []string{"INVALID_API_SPEC"},
		},
		{
			name:        "missing endpoint",
			specPath:    relativeSpecPath,
			projectRoot: tmpDir,
			features: []spec.Feature{
				{
					ID: "feat-001",
					API: []spec.API{
						{Path: "/api/missing", Method: "GET"},
					},
				},
			},
			wantFindings: 1,
			wantCodes:    []string{"MISSING_API_PATH"},
		},
	}

	// Create invalid spec
	invalidSpec := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidSpec, []byte("invalid: yaml: content:"), 0644); err != nil {
		t.Fatalf("Failed to create invalid spec: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := ValidateAPISpec(tt.specPath, tt.projectRoot, tt.features)

			if len(findings) != tt.wantFindings {
				t.Errorf("ValidateAPISpec() found %d findings, want %d", len(findings), tt.wantFindings)
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
					t.Errorf("ValidateAPISpec() missing expected finding code: %s", wantCode)
				}
			}
		})
	}
}

func TestFindPathWithParams(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := createTestOpenAPISpec(t, tmpDir)

	validator, err := NewOpenAPIValidator(specPath)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		wantMatch bool
	}{
		{
			name:      "exact match",
			path:      "/api/users/{id}",
			wantMatch: true,
		},
		{
			name:      "different param name",
			path:      "/api/users/{userId}",
			wantMatch: true,
		},
		{
			name:      "no match - wrong path",
			path:      "/api/missing/{id}",
			wantMatch: false,
		},
		{
			name:      "no match - different segments",
			path:      "/api/users/extra/{id}",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathItem := validator.findPathWithParams(tt.path)
			gotMatch := pathItem != nil

			if gotMatch != tt.wantMatch {
				t.Errorf("findPathWithParams() match = %v, want %v", gotMatch, tt.wantMatch)
			}
		})
	}
}

func TestValidateEndpoints_AllHTTPMethods(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create OpenAPI spec with all HTTP methods
	specContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /api/resource:
    get:
      summary: Get resource
      responses:
        '200':
          description: Success
    post:
      summary: Create resource
      responses:
        '201':
          description: Created
    put:
      summary: Update resource
      responses:
        '200':
          description: Updated
    patch:
      summary: Partial update
      responses:
        '200':
          description: Updated
    delete:
      summary: Delete resource
      responses:
        '204':
          description: Deleted
    head:
      summary: Get headers
      responses:
        '200':
          description: Success
    options:
      summary: Get options
      responses:
        '200':
          description: Success
`
	specPath := filepath.Join(tmpDir, "openapi.yaml")
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("Failed to write spec: %v", err)
	}

	validator, err := NewOpenAPIValidator(specPath)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Test all HTTP methods
	features := []spec.Feature{
		{
			ID: "feat-all-methods",
			API: []spec.API{
				{Path: "/api/resource", Method: "GET"},
				{Path: "/api/resource", Method: "POST"},
				{Path: "/api/resource", Method: "PUT"},
				{Path: "/api/resource", Method: "PATCH"},
				{Path: "/api/resource", Method: "DELETE"},
				{Path: "/api/resource", Method: "HEAD"},
				{Path: "/api/resource", Method: "OPTIONS"},
			},
		},
	}

	findings := validator.ValidateEndpoints(features)
	if len(findings) != 0 {
		t.Errorf("ValidateEndpoints() found %d findings, want 0", len(findings))
		for _, f := range findings {
			t.Logf("Finding: %s", f.Message)
		}
	}
}

func TestValidateEndpoints_UnsupportedMethod(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := createTestOpenAPISpec(t, tmpDir)

	validator, err := NewOpenAPIValidator(specPath)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Test unsupported HTTP method
	features := []spec.Feature{
		{
			ID: "feat-unsupported",
			API: []spec.API{
				{Path: "/api/users", Method: "CONNECT"}, // Unsupported method
			},
		},
	}

	findings := validator.ValidateEndpoints(features)
	if len(findings) == 0 {
		t.Error("ValidateEndpoints() expected findings for unsupported method, got 0")
	}
}
