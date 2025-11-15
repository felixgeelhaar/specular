package plan

import (
	"context"
	"fmt"
	"testing"

	"github.com/felixgeelhaar/specular/internal/domain"
	"github.com/felixgeelhaar/specular/internal/spec"
)

// createTestSpec creates a ProductSpec with the specified number of features
func createTestSpec(numFeatures int) (*spec.ProductSpec, *spec.SpecLock) {
	features := make([]spec.Feature, numFeatures)
	lockedFeatures := make(map[domain.FeatureID]spec.LockedFeature)

	for i := 0; i < numFeatures; i++ {
		featureID := domain.FeatureID(fmt.Sprintf("feat-%03d", i+1))

		// Alternate priorities to create dependency chains
		var priority domain.Priority
		switch i % 3 {
		case 0:
			priority = "P0"
		case 1:
			priority = "P1"
		default:
			priority = "P2"
		}

		features[i] = spec.Feature{
			ID:       featureID,
			Title:    fmt.Sprintf("Feature %d", i+1),
			Desc:     fmt.Sprintf("Description for feature %d", i+1),
			Priority: priority,
			API: []spec.API{
				{Method: "POST", Path: fmt.Sprintf("/api/feature-%d", i+1)},
			},
			Success: []string{fmt.Sprintf("Feature %d works", i+1)},
			Trace:   []string{fmt.Sprintf("PRD-%03d", i+1)},
		}

		lockedFeatures[featureID] = spec.LockedFeature{
			Hash: fmt.Sprintf("hash%03d", i+1),
		}
	}

	testSpec := &spec.ProductSpec{
		Product:  "Benchmark Test Product",
		Features: features,
	}

	testLock := &spec.SpecLock{
		Version:  "1.0",
		Features: lockedFeatures,
	}

	return testSpec, testLock
}

// BenchmarkGenerate_SmallSpec benchmarks plan generation with 3 features
func BenchmarkGenerate_SmallSpec(b *testing.B) {
	testSpec, testLock := createTestSpec(3)

	opts := GenerateOptions{
		SpecLock:           testLock,
		EstimateComplexity: false,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(ctx, testSpec, opts)
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
	}
}

// BenchmarkGenerate_SmallSpec_WithComplexity benchmarks with complexity estimation
func BenchmarkGenerate_SmallSpec_WithComplexity(b *testing.B) {
	testSpec, testLock := createTestSpec(3)

	opts := GenerateOptions{
		SpecLock:           testLock,
		EstimateComplexity: true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(ctx, testSpec, opts)
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
	}
}

// BenchmarkGenerate_MediumSpec benchmarks plan generation with 10 features
func BenchmarkGenerate_MediumSpec(b *testing.B) {
	testSpec, testLock := createTestSpec(10)

	opts := GenerateOptions{
		SpecLock:           testLock,
		EstimateComplexity: false,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(ctx, testSpec, opts)
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
	}
}

// BenchmarkGenerate_MediumSpec_WithComplexity benchmarks with complexity estimation
func BenchmarkGenerate_MediumSpec_WithComplexity(b *testing.B) {
	testSpec, testLock := createTestSpec(10)

	opts := GenerateOptions{
		SpecLock:           testLock,
		EstimateComplexity: true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(ctx, testSpec, opts)
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
	}
}

// BenchmarkGenerate_LargeSpec benchmarks plan generation with 50 features
func BenchmarkGenerate_LargeSpec(b *testing.B) {
	testSpec, testLock := createTestSpec(50)

	opts := GenerateOptions{
		SpecLock:           testLock,
		EstimateComplexity: false,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(ctx, testSpec, opts)
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
	}
}

// BenchmarkGenerate_LargeSpec_WithComplexity benchmarks with complexity estimation
func BenchmarkGenerate_LargeSpec_WithComplexity(b *testing.B) {
	testSpec, testLock := createTestSpec(50)

	opts := GenerateOptions{
		SpecLock:           testLock,
		EstimateComplexity: true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(ctx, testSpec, opts)
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
	}
}

// BenchmarkGenerate_VeryLargeSpec benchmarks plan generation with 100 features
func BenchmarkGenerate_VeryLargeSpec(b *testing.B) {
	testSpec, testLock := createTestSpec(100)

	opts := GenerateOptions{
		SpecLock:           testLock,
		EstimateComplexity: false,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(ctx, testSpec, opts)
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
	}
}

// BenchmarkGenerate_VeryLargeSpec_WithComplexity benchmarks with complexity estimation
func BenchmarkGenerate_VeryLargeSpec_WithComplexity(b *testing.B) {
	testSpec, testLock := createTestSpec(100)

	opts := GenerateOptions{
		SpecLock:           testLock,
		EstimateComplexity: true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(ctx, testSpec, opts)
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
	}
}

// BenchmarkGenerate_AllP0 benchmarks with all P0 features (no dependencies)
func BenchmarkGenerate_AllP0(b *testing.B) {
	testSpec, testLock := createTestSpec(20)

	// Set all features to P0
	for i := range testSpec.Features {
		testSpec.Features[i].Priority = "P0"
	}

	opts := GenerateOptions{
		SpecLock:           testLock,
		EstimateComplexity: false,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(ctx, testSpec, opts)
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
	}
}

// BenchmarkGenerate_DeepDependencies benchmarks with deep dependency chains
func BenchmarkGenerate_DeepDependencies(b *testing.B) {
	testSpec, testLock := createTestSpec(20)

	// Create deep dependency chain: P0, then all P1, then all P2
	for i := range testSpec.Features {
		if i < 1 {
			testSpec.Features[i].Priority = "P0"
		} else if i < 10 {
			testSpec.Features[i].Priority = "P1"
		} else {
			testSpec.Features[i].Priority = "P2"
		}
	}

	opts := GenerateOptions{
		SpecLock:           testLock,
		EstimateComplexity: false,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(ctx, testSpec, opts)
		if err != nil {
			b.Fatalf("Generate() error = %v", err)
		}
	}
}

// BenchmarkGenerate_Parallel benchmarks parallel plan generation
func BenchmarkGenerate_Parallel(b *testing.B) {
	testSpec, testLock := createTestSpec(10)

	opts := GenerateOptions{
		SpecLock:           testLock,
		EstimateComplexity: true,
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := Generate(ctx, testSpec, opts)
			if err != nil {
				b.Fatalf("Generate() error = %v", err)
			}
		}
	})
}
