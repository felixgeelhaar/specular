package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/internal/domain"
)

// createBenchmarkSpec creates a YAML spec file with the specified number of features
func createBenchmarkSpec(b *testing.B, numFeatures int) string {
	b.Helper()

	content := `product: BenchmarkProduct
goals:
  - Test performance
  - Measure scalability
features:
`

	for i := 0; i < numFeatures; i++ {
		priority := "P0"
		if i%3 == 1 {
			priority = "P1"
		} else if i%3 == 2 {
			priority = "P2"
		}

		featureContent := fmt.Sprintf(`  - id: feat-%03d
    title: Feature %d
    desc: Description for feature %d with some additional text to make it realistic
    priority: %s
    api:
      - method: GET
        path: /api/feature-%d
        request: Feature%dRequest
        response: Feature%dResponse
      - method: POST
        path: /api/feature-%d/create
        request: Create%dRequest
        response: Create%dResponse
    success:
      - Feature %d works correctly
      - Integration tests pass
      - Performance meets requirements
    trace:
      - PRD-%03d
      - EPIC-%03d
`, i+1, i+1, i+1, priority, i+1, i+1, i+1, i+1, i+1, i+1, i+1, i+1, i+1)

		content += featureContent
	}

	content += `non_functional:
  performance:
    - Response time < 2s
    - Throughput > 1000 req/s
  availability:
    - Uptime > 99.9%
    - Error rate < 0.1%
  security:
    - HTTPS only
    - Authentication required
    - Input validation
acceptance:
  - All features implemented according to specification
  - Integration tests passing
  - Performance benchmarks meet requirements
`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "bench-spec-*.yaml")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		b.Fatalf("Failed to write temp file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		b.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpFile.Name()
}

// BenchmarkLoadSpec_SmallSpec benchmarks loading a spec with 3 features
func BenchmarkLoadSpec_SmallSpec(b *testing.B) {
	specPath := createBenchmarkSpec(b, 3)
	defer os.Remove(specPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadSpec(specPath)
		if err != nil {
			b.Fatalf("LoadSpec() error = %v", err)
		}
	}
}

// BenchmarkLoadSpec_MediumSpec benchmarks loading a spec with 10 features
func BenchmarkLoadSpec_MediumSpec(b *testing.B) {
	specPath := createBenchmarkSpec(b, 10)
	defer os.Remove(specPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadSpec(specPath)
		if err != nil {
			b.Fatalf("LoadSpec() error = %v", err)
		}
	}
}

// BenchmarkLoadSpec_LargeSpec benchmarks loading a spec with 50 features
func BenchmarkLoadSpec_LargeSpec(b *testing.B) {
	specPath := createBenchmarkSpec(b, 50)
	defer os.Remove(specPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadSpec(specPath)
		if err != nil {
			b.Fatalf("LoadSpec() error = %v", err)
		}
	}
}

// BenchmarkLoadSpec_VeryLargeSpec benchmarks loading a spec with 100 features
func BenchmarkLoadSpec_VeryLargeSpec(b *testing.B) {
	specPath := createBenchmarkSpec(b, 100)
	defer os.Remove(specPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadSpec(specPath)
		if err != nil {
			b.Fatalf("LoadSpec() error = %v", err)
		}
	}
}

// BenchmarkLoadSpec_Parallel benchmarks parallel loading
func BenchmarkLoadSpec_Parallel(b *testing.B) {
	specPath := createBenchmarkSpec(b, 10)
	defer os.Remove(specPath)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := LoadSpec(specPath)
			if err != nil {
				b.Fatalf("LoadSpec() error = %v", err)
			}
		}
	})
}

// BenchmarkSaveSpec_SmallSpec benchmarks saving a spec with 3 features
func BenchmarkSaveSpec_SmallSpec(b *testing.B) {
	spec := createTestProductSpec(3)
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("spec-%d.yaml", i))
		err := SaveSpec(spec, path)
		if err != nil {
			b.Fatalf("SaveSpec() error = %v", err)
		}
	}
}

// BenchmarkSaveSpec_MediumSpec benchmarks saving a spec with 10 features
func BenchmarkSaveSpec_MediumSpec(b *testing.B) {
	spec := createTestProductSpec(10)
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("spec-%d.yaml", i))
		err := SaveSpec(spec, path)
		if err != nil {
			b.Fatalf("SaveSpec() error = %v", err)
		}
	}
}

// BenchmarkSaveSpec_LargeSpec benchmarks saving a spec with 50 features
func BenchmarkSaveSpec_LargeSpec(b *testing.B) {
	spec := createTestProductSpec(50)
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("spec-%d.yaml", i))
		err := SaveSpec(spec, path)
		if err != nil {
			b.Fatalf("SaveSpec() error = %v", err)
		}
	}
}

// BenchmarkLoadSpec_Cached benchmarks loading the same spec repeatedly (tests filesystem caching)
func BenchmarkLoadSpec_Cached(b *testing.B) {
	specPath := createBenchmarkSpec(b, 10)
	defer os.Remove(specPath)

	// Pre-load once to ensure file is in filesystem cache
	_, err := LoadSpec(specPath)
	if err != nil {
		b.Fatalf("Pre-load failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadSpec(specPath)
		if err != nil {
			b.Fatalf("LoadSpec() error = %v", err)
		}
	}
}

// createTestProductSpec creates a ProductSpec programmatically for Save benchmarks
func createTestProductSpec(numFeatures int) *ProductSpec {
	features := make([]Feature, numFeatures)

	for i := 0; i < numFeatures; i++ {
		priority := domain.Priority("P0")
		if i%3 == 1 {
			priority = domain.Priority("P1")
		} else if i%3 == 2 {
			priority = domain.Priority("P2")
		}

		features[i] = Feature{
			ID:       domain.FeatureID(fmt.Sprintf("feat-%03d", i+1)),
			Title:    fmt.Sprintf("Feature %d", i+1),
			Desc:     fmt.Sprintf("Description for feature %d with some additional text to make it realistic", i+1),
			Priority: priority,
			API: []API{
				{
					Method:   "GET",
					Path:     fmt.Sprintf("/api/feature-%d", i+1),
					Request:  fmt.Sprintf("Feature%dRequest", i+1),
					Response: fmt.Sprintf("Feature%dResponse", i+1),
				},
				{
					Method:   "POST",
					Path:     fmt.Sprintf("/api/feature-%d/create", i+1),
					Request:  fmt.Sprintf("Create%dRequest", i+1),
					Response: fmt.Sprintf("Create%dResponse", i+1),
				},
			},
			Success: []string{
				fmt.Sprintf("Feature %d works correctly", i+1),
				"Integration tests pass",
				"Performance meets requirements",
			},
			Trace: []string{
				fmt.Sprintf("PRD-%03d", i+1),
				fmt.Sprintf("EPIC-%03d", i+1),
			},
		}
	}

	return &ProductSpec{
		Product: "BenchmarkProduct",
		Goals: []string{
			"Test performance",
			"Measure scalability",
		},
		Features: features,
		NonFunctional: NonFunctional{
			Performance: []string{
				"Response time < 2s",
				"Throughput > 1000 req/s",
			},
			Availability: []string{
				"Uptime > 99.9%",
				"Error rate < 0.1%",
			},
			Security: []string{
				"HTTPS only",
				"Authentication required",
				"Input validation",
			},
		},
	}
}
