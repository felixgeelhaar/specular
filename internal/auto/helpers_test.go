package auto

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/internal/domain"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
	"gopkg.in/yaml.v3"
)

// TestSaveOutputFiles tests the saveOutputFiles helper method
func TestSaveOutputFiles(t *testing.T) {
	// Create a temporary directory for test output
	tmpDir := t.TempDir()

	// Create test data
	productSpec := &spec.ProductSpec{
		Product: "TestProduct",
		Features: []spec.Feature{
			{
				ID:    "feat-1",
				Title: "Test Feature",
				Desc:  "Test description",
			},
		},
	}

	specLock := &spec.SpecLock{
		Version: "1.0.0",
		Features: map[domain.FeatureID]spec.LockedFeature{
			"feat-1": {
				Hash: "abc123",
			},
		},
	}

	execPlan := &plan.Plan{
		Tasks: []plan.Task{
			{
				ID:        "task-1",
				FeatureID: "feat-1",
				Skill:     "test-skill",
			},
		},
	}

	actionPlan := CreateDefaultActionPlan("Test goal", "default")

	// Create orchestrator with output directory
	config := DefaultConfig()
	config.OutputDir = tmpDir
	o := &Orchestrator{
		config: config,
	}

	// Call saveOutputFiles
	err := o.saveOutputFiles(productSpec, specLock, execPlan, actionPlan)
	if err != nil {
		t.Fatalf("saveOutputFiles failed: %v", err)
	}

	// Verify spec.yaml was created
	specPath := filepath.Join(tmpDir, "spec.yaml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Error("spec.yaml was not created")
	}

	// Verify spec.lock.json was created
	lockPath := filepath.Join(tmpDir, "spec.lock.json")
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("spec.lock.json was not created")
	}

	// Verify plan.json was created
	planPath := filepath.Join(tmpDir, "plan.json")
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		t.Error("plan.json was not created")
	}

	// Verify action-plan.json was created
	actionPlanPath := filepath.Join(tmpDir, "action-plan.json")
	if _, err := os.Stat(actionPlanPath); os.IsNotExist(err) {
		t.Error("action-plan.json was not created")
	}

	// Verify spec.yaml content
	specData, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read spec.yaml: %v", err)
	}

	var loadedSpec spec.ProductSpec
	if err := yaml.Unmarshal(specData, &loadedSpec); err != nil {
		t.Fatalf("Failed to unmarshal spec.yaml: %v", err)
	}

	if loadedSpec.Product != productSpec.Product {
		t.Errorf("Spec product = %s, want %s", loadedSpec.Product, productSpec.Product)
	}

	// Verify plan.json content
	planData, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("Failed to read plan.json: %v", err)
	}

	var loadedPlan plan.Plan
	if err := json.Unmarshal(planData, &loadedPlan); err != nil {
		t.Fatalf("Failed to unmarshal plan.json: %v", err)
	}

	if len(loadedPlan.Tasks) != len(execPlan.Tasks) {
		t.Errorf("Plan tasks count = %d, want %d", len(loadedPlan.Tasks), len(execPlan.Tasks))
	}
}

// TestSaveOutputFiles_InvalidDirectory tests saveOutputFiles with invalid directory
func TestSaveOutputFiles_InvalidDirectory(t *testing.T) {
	productSpec := &spec.ProductSpec{Product: "Test"}
	specLock := &spec.SpecLock{Version: "1.0.0"}
	execPlan := &plan.Plan{}
	actionPlan := CreateDefaultActionPlan("Test", "default")

	config := DefaultConfig()
	config.OutputDir = "/invalid/path/that/does/not/exist/and/cannot/be/created"
	o := &Orchestrator{
		config: config,
	}

	err := o.saveOutputFiles(productSpec, specLock, execPlan, actionPlan)
	if err == nil {
		t.Error("Expected error for invalid directory, got nil")
	}
}

// TestSaveOutputFiles_EmptyDirectory tests saveOutputFiles creates directory
func TestSaveOutputFiles_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "new_output_dir")

	productSpec := &spec.ProductSpec{Product: "Test"}
	specLock := &spec.SpecLock{Version: "1.0.0"}
	execPlan := &plan.Plan{}
	actionPlan := CreateDefaultActionPlan("Test", "default")

	config := DefaultConfig()
	config.OutputDir = outputDir
	o := &Orchestrator{
		config: config,
	}

	err := o.saveOutputFiles(productSpec, specLock, execPlan, actionPlan)
	if err != nil {
		t.Fatalf("saveOutputFiles failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("Output directory was not created")
	}
}

// TestGenerateSpecLock tests the generateSpecLock helper method
func TestGenerateSpecLock(t *testing.T) {
	productSpec := &spec.ProductSpec{
		Product: "TestProduct",
		Features: []spec.Feature{
			{
				ID:    "feat-1",
				Title: "Feature 1",
				Desc:  "Description 1",
			},
			{
				ID:    "feat-2",
				Title: "Feature 2",
				Desc:  "Description 2",
			},
		},
	}

	o := &Orchestrator{}
	specLock, err := o.generateSpecLock(productSpec)

	if err != nil {
		t.Fatalf("generateSpecLock failed: %v", err)
	}

	if specLock == nil {
		t.Fatal("generateSpecLock returned nil lock")
	}

	if specLock.Version != "1.0.0" {
		t.Errorf("SpecLock version = %s, want 1.0.0", specLock.Version)
	}

	if len(specLock.Features) != len(productSpec.Features) {
		t.Errorf("SpecLock features count = %d, want %d", len(specLock.Features), len(productSpec.Features))
	}

	// Verify each feature has a hash
	for _, feature := range productSpec.Features {
		lockedFeature, exists := specLock.Features[feature.ID]
		if !exists {
			t.Errorf("Feature %s not found in SpecLock", feature.ID)
			continue
		}
		if lockedFeature.Hash == "" {
			t.Errorf("Feature %s hash is empty", feature.ID)
		}
	}
}

// TestGenerateSpecLock_EmptySpec tests generateSpecLock with empty spec
func TestGenerateSpecLock_EmptySpec(t *testing.T) {
	productSpec := &spec.ProductSpec{
		Product:  "EmptyProduct",
		Features: []spec.Feature{},
	}

	o := &Orchestrator{}
	specLock, err := o.generateSpecLock(productSpec)

	if err != nil {
		t.Fatalf("generateSpecLock failed: %v", err)
	}

	if specLock == nil {
		t.Fatal("generateSpecLock returned nil lock")
	}

	if len(specLock.Features) != 0 {
		t.Errorf("SpecLock features count = %d, want 0", len(specLock.Features))
	}
}

// TestGeneratePlan tests the generatePlan helper method
func TestGeneratePlan(t *testing.T) {
	productSpec := &spec.ProductSpec{
		Product: "TestProduct",
		Features: []spec.Feature{
			{
				ID:       "feat-1",
				Title:    "Feature 1",
				Desc:     "Description 1",
				Priority: "P0",
			},
		},
	}

	specLock := &spec.SpecLock{
		Version: "1.0.0",
		Features: map[domain.FeatureID]spec.LockedFeature{
			"feat-1": {
				Hash: "abc123",
			},
		},
	}

	o := &Orchestrator{}
	ctx := context.Background()

	execPlan, err := o.generatePlan(ctx, productSpec, specLock)

	if err != nil {
		t.Fatalf("generatePlan failed: %v", err)
	}

	if execPlan == nil {
		t.Fatal("generatePlan returned nil plan")
	}

	if len(execPlan.Tasks) == 0 {
		t.Error("generatePlan returned empty task list")
	}

	// Verify tasks were created for the feature
	foundFeatureTask := false
	for _, task := range execPlan.Tasks {
		if task.FeatureID == "feat-1" {
			foundFeatureTask = true
			break
		}
	}

	if !foundFeatureTask {
		t.Error("generatePlan did not create tasks for feat-1")
	}
}

// TestGeneratePlan_MultipleFeatures tests generatePlan with multiple features
func TestGeneratePlan_MultipleFeatures(t *testing.T) {
	productSpec := &spec.ProductSpec{
		Product: "TestProduct",
		Features: []spec.Feature{
			{
				ID:       "feat-1",
				Title:    "Feature 1",
				Desc:     "Description 1",
				Priority: "P0",
			},
			{
				ID:       "feat-2",
				Title:    "Feature 2",
				Desc:     "Description 2",
				Priority: "P1",
			},
		},
	}

	specLock := &spec.SpecLock{
		Version: "1.0.0",
		Features: map[domain.FeatureID]spec.LockedFeature{
			"feat-1": {Hash: "abc123"},
			"feat-2": {Hash: "def456"},
		},
	}

	o := &Orchestrator{}
	ctx := context.Background()

	execPlan, err := o.generatePlan(ctx, productSpec, specLock)

	if err != nil {
		t.Fatalf("generatePlan failed: %v", err)
	}

	if len(execPlan.Tasks) == 0 {
		t.Error("generatePlan returned empty task list")
	}

	// Verify tasks were created for both features
	featureTaskCount := make(map[domain.FeatureID]int)
	for _, task := range execPlan.Tasks {
		featureTaskCount[task.FeatureID]++
	}

	if featureTaskCount["feat-1"] == 0 {
		t.Error("No tasks created for feat-1")
	}
	if featureTaskCount["feat-2"] == 0 {
		t.Error("No tasks created for feat-2")
	}
}

// TestGeneratePlan_EmptySpec tests generatePlan with empty feature list
func TestGeneratePlan_EmptySpec(t *testing.T) {
	productSpec := &spec.ProductSpec{
		Product:  "EmptyProduct",
		Features: []spec.Feature{},
	}

	specLock := &spec.SpecLock{
		Version:  "1.0.0",
		Features: map[domain.FeatureID]spec.LockedFeature{},
	}

	o := &Orchestrator{}
	ctx := context.Background()

	execPlan, err := o.generatePlan(ctx, productSpec, specLock)

	if err != nil {
		t.Fatalf("generatePlan failed: %v", err)
	}

	if execPlan == nil {
		t.Fatal("generatePlan returned nil plan")
	}

	// Empty spec should produce empty plan
	if len(execPlan.Tasks) != 0 {
		t.Errorf("Expected 0 tasks for empty spec, got %d", len(execPlan.Tasks))
	}
}
