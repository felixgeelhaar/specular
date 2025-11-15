package drift

import (
	"fmt"
	"testing"

	"github.com/felixgeelhaar/specular/internal/domain"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
)

// createTestData creates a SpecLock and Plan with the specified number of features
func createTestData(numFeatures int, driftRatio float64) (*spec.SpecLock, *plan.Plan) {
	lock := &spec.SpecLock{
		Version:  "1.0",
		Features: make(map[domain.FeatureID]spec.LockedFeature),
	}

	tasks := make([]plan.Task, numFeatures)

	for i := 0; i < numFeatures; i++ {
		featureID := domain.FeatureID(fmt.Sprintf("feat-%03d", i+1))
		correctHash := fmt.Sprintf("hash%03d", i+1)

		// Add to lock
		lock.Features[featureID] = spec.LockedFeature{
			Hash:        correctHash,
			OpenAPIPath: fmt.Sprintf(".specular/openapi/%s.yaml", featureID),
			TestPaths:   []string{fmt.Sprintf(".specular/tests/%s_test.go", featureID)},
		}

		// Determine task hash based on drift ratio
		taskHash := correctHash
		if float64(i) < float64(numFeatures)*driftRatio {
			taskHash = fmt.Sprintf("wrong%03d", i+1) // Intentional mismatch
		}

		// Add to plan
		tasks[i] = plan.Task{
			ID:           domain.TaskID(fmt.Sprintf("task-%03d", i+1)),
			FeatureID:    featureID,
			ExpectedHash: taskHash,
			DependsOn:    []domain.TaskID{},
			Skill:        "go-backend",
			Priority:     domain.Priority("P0"),
		}
	}

	p := &plan.Plan{
		Tasks: tasks,
	}

	return lock, p
}

// BenchmarkDetectPlanDrift_SmallNoDrift benchmarks drift detection with 3 features, no drift
func BenchmarkDetectPlanDrift_SmallNoDrift(b *testing.B) {
	lock, p := createTestData(3, 0.0) // 0% drift

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings := DetectPlanDrift(lock, p)
		if len(findings) != 0 {
			b.Fatalf("Expected 0 findings, got %d", len(findings))
		}
	}
}

// BenchmarkDetectPlanDrift_SmallWithDrift benchmarks with 3 features, 33% drift
func BenchmarkDetectPlanDrift_SmallWithDrift(b *testing.B) {
	lock, p := createTestData(3, 0.33) // 33% drift (1 feature)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings := DetectPlanDrift(lock, p)
		if len(findings) != 1 {
			b.Fatalf("Expected 1 finding, got %d", len(findings))
		}
	}
}

// BenchmarkDetectPlanDrift_MediumNoDrift benchmarks with 10 features, no drift
func BenchmarkDetectPlanDrift_MediumNoDrift(b *testing.B) {
	lock, p := createTestData(10, 0.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings := DetectPlanDrift(lock, p)
		if len(findings) != 0 {
			b.Fatalf("Expected 0 findings, got %d", len(findings))
		}
	}
}

// BenchmarkDetectPlanDrift_MediumWithDrift benchmarks with 10 features, 30% drift
func BenchmarkDetectPlanDrift_MediumWithDrift(b *testing.B) {
	lock, p := createTestData(10, 0.30) // 30% drift (3 features)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings := DetectPlanDrift(lock, p)
		if len(findings) != 3 {
			b.Fatalf("Expected 3 findings, got %d", len(findings))
		}
	}
}

// BenchmarkDetectPlanDrift_LargeNoDrift benchmarks with 50 features, no drift
func BenchmarkDetectPlanDrift_LargeNoDrift(b *testing.B) {
	lock, p := createTestData(50, 0.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings := DetectPlanDrift(lock, p)
		if len(findings) != 0 {
			b.Fatalf("Expected 0 findings, got %d", len(findings))
		}
	}
}

// BenchmarkDetectPlanDrift_LargeWithDrift benchmarks with 50 features, 20% drift
func BenchmarkDetectPlanDrift_LargeWithDrift(b *testing.B) {
	lock, p := createTestData(50, 0.20) // 20% drift (10 features)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings := DetectPlanDrift(lock, p)
		if len(findings) != 10 {
			b.Fatalf("Expected 10 findings, got %d", len(findings))
		}
	}
}

// BenchmarkDetectPlanDrift_VeryLargeNoDrift benchmarks with 100 features, no drift
func BenchmarkDetectPlanDrift_VeryLargeNoDrift(b *testing.B) {
	lock, p := createTestData(100, 0.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings := DetectPlanDrift(lock, p)
		if len(findings) != 0 {
			b.Fatalf("Expected 0 findings, got %d", len(findings))
		}
	}
}

// BenchmarkDetectPlanDrift_VeryLargeWithDrift benchmarks with 100 features, 25% drift
func BenchmarkDetectPlanDrift_VeryLargeWithDrift(b *testing.B) {
	lock, p := createTestData(100, 0.25) // 25% drift (25 features)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings := DetectPlanDrift(lock, p)
		if len(findings) != 25 {
			b.Fatalf("Expected 25 findings, got %d", len(findings))
		}
	}
}

// BenchmarkDetectPlanDrift_HighDriftRatio benchmarks with 20 features, 80% drift
func BenchmarkDetectPlanDrift_HighDriftRatio(b *testing.B) {
	lock, p := createTestData(20, 0.80) // 80% drift (16 features)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings := DetectPlanDrift(lock, p)
		if len(findings) != 16 {
			b.Fatalf("Expected 16 findings, got %d", len(findings))
		}
	}
}

// BenchmarkDetectPlanDrift_MissingTasks benchmarks drift detection with missing tasks
func BenchmarkDetectPlanDrift_MissingTasks(b *testing.B) {
	lock := &spec.SpecLock{
		Version: "1.0",
		Features: map[domain.FeatureID]spec.LockedFeature{
			domain.FeatureID("feat-001"): {Hash: "abc123"},
			domain.FeatureID("feat-002"): {Hash: "def456"},
			domain.FeatureID("feat-003"): {Hash: "ghi789"},
			domain.FeatureID("feat-004"): {Hash: "jkl012"},
			domain.FeatureID("feat-005"): {Hash: "mno345"},
		},
	}

	// Plan only covers 2 out of 5 features
	p := &plan.Plan{
		Tasks: []plan.Task{
			{
				ID:           domain.TaskID("task-001"),
				FeatureID:    domain.FeatureID("feat-001"),
				ExpectedHash: "abc123",
			},
			{
				ID:           domain.TaskID("task-002"),
				FeatureID:    domain.FeatureID("feat-002"),
				ExpectedHash: "def456",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findings := DetectPlanDrift(lock, p)
		if len(findings) != 3 { // 3 missing tasks
			b.Fatalf("Expected 3 findings, got %d", len(findings))
		}
	}
}

// BenchmarkGenerateReport benchmarks report generation with multiple drift types
func BenchmarkGenerateReport(b *testing.B) {
	// Create sample findings for each drift type
	planDrift := []Finding{
		{Code: "HASH_MISMATCH", FeatureID: "feat-001", Severity: "error", Message: "Hash mismatch"},
		{Code: "MISSING_TASK", FeatureID: "feat-002", Severity: "warning", Message: "Missing task"},
	}

	codeDrift := []Finding{
		{Code: "MISSING_IMPL", FeatureID: "feat-003", Severity: "error", Message: "Missing implementation"},
		{Code: "EXTRA_CODE", FeatureID: "feat-004", Severity: "info", Message: "Extra code"},
	}

	infraDrift := []Finding{
		{Code: "CONFIG_MISMATCH", FeatureID: "feat-005", Severity: "warning", Message: "Config mismatch"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report := GenerateReport(planDrift, codeDrift, infraDrift)
		if report.Summary.TotalFindings != 5 {
			b.Fatalf("Expected 5 total findings, got %d", report.Summary.TotalFindings)
		}
		if report.Summary.Errors != 2 {
			b.Fatalf("Expected 2 errors, got %d", report.Summary.Errors)
		}
		if report.Summary.Warnings != 2 {
			b.Fatalf("Expected 2 warnings, got %d", report.Summary.Warnings)
		}
		if report.Summary.Info != 1 {
			b.Fatalf("Expected 1 info, got %d", report.Summary.Info)
		}
	}
}

// BenchmarkDetectPlanDrift_Parallel benchmarks parallel drift detection
func BenchmarkDetectPlanDrift_Parallel(b *testing.B) {
	lock, p := createTestData(20, 0.25)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			findings := DetectPlanDrift(lock, p)
			if len(findings) != 5 {
				b.Fatalf("Expected 5 findings, got %d", len(findings))
			}
		}
	})
}
