package ux

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPathDefaults(t *testing.T) {
	defaults := NewPathDefaults()

	if defaults == nil {
		t.Fatal("NewPathDefaults() returned nil")
	}

	if defaults.SpecularDir != ".specular" {
		t.Errorf("SpecularDir = %s, want .specular", defaults.SpecularDir)
	}
}

func TestPathDefaults_SpecFile(t *testing.T) {
	defaults := NewPathDefaults()
	specFile := defaults.SpecFile()

	expected := filepath.Join(".specular", "spec.yaml")
	if specFile != expected {
		t.Errorf("SpecFile() = %s, want %s", specFile, expected)
	}
}

func TestPathDefaults_SpecLockFile(t *testing.T) {
	defaults := NewPathDefaults()
	lockFile := defaults.SpecLockFile()

	expected := filepath.Join(".specular", "spec.lock.json")
	if lockFile != expected {
		t.Errorf("SpecLockFile() = %s, want %s", lockFile, expected)
	}
}

func TestPathDefaults_PlanFile(t *testing.T) {
	defaults := NewPathDefaults()
	planFile := defaults.PlanFile()

	expected := "plan.json"
	if planFile != expected {
		t.Errorf("PlanFile() = %s, want %s", planFile, expected)
	}
}

func TestPathDefaults_PolicyFile(t *testing.T) {
	defaults := NewPathDefaults()
	policyFile := defaults.PolicyFile()

	expected := filepath.Join(".specular", "policy.yaml")
	if policyFile != expected {
		t.Errorf("PolicyFile() = %s, want %s", policyFile, expected)
	}
}

func TestPathDefaults_ProvidersFile(t *testing.T) {
	defaults := NewPathDefaults()
	providersFile := defaults.ProvidersFile()

	expected := filepath.Join(".specular", "providers.yaml")
	if providersFile != expected {
		t.Errorf("ProvidersFile() = %s, want %s", providersFile, expected)
	}
}

func TestPathDefaults_RouterFile(t *testing.T) {
	defaults := NewPathDefaults()
	routerFile := defaults.RouterFile()

	expected := filepath.Join(".specular", "router.yaml")
	if routerFile != expected {
		t.Errorf("RouterFile() = %s, want %s", routerFile, expected)
	}
}

func TestPathDefaults_CheckpointDir(t *testing.T) {
	defaults := NewPathDefaults()
	checkpointDir := defaults.CheckpointDir()

	expected := filepath.Join(".specular", "checkpoints")
	if checkpointDir != expected {
		t.Errorf("CheckpointDir() = %s, want %s", checkpointDir, expected)
	}
}

func TestPathDefaults_ManifestDir(t *testing.T) {
	defaults := NewPathDefaults()
	manifestDir := defaults.ManifestDir()

	expected := filepath.Join(".specular", "runs")
	if manifestDir != expected {
		t.Errorf("ManifestDir() = %s, want %s", manifestDir, expected)
	}
}

func TestPathDefaults_CacheDir(t *testing.T) {
	defaults := NewPathDefaults()
	cacheDir := defaults.CacheDir()

	expected := filepath.Join(".specular", "cache")
	if cacheDir != expected {
		t.Errorf("CacheDir() = %s, want %s", cacheDir, expected)
	}
}

func TestPathDefaults_ValidateSpecularSetup_Missing(t *testing.T) {
	// Create a temporary directory without .specular
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	defaults := NewPathDefaults()
	err = defaults.ValidateSpecularSetup()
	if err == nil {
		t.Error("ValidateSpecularSetup() should return error when .specular is missing")
	}
}

func TestPathDefaults_ValidateSpecularSetup_Exists(t *testing.T) {
	// Create a temporary directory with .specular
	tmpDir := t.TempDir()
	specularDir := filepath.Join(tmpDir, ".specular")
	if err := os.MkdirAll(specularDir, 0755); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	defaults := NewPathDefaults()
	err = defaults.ValidateSpecularSetup()
	if err != nil {
		t.Errorf("ValidateSpecularSetup() should not return error when .specular exists: %v", err)
	}
}

func TestValidateRequiredFile_FileExists(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Validate it exists
	err = ValidateRequiredFile(tmpFile.Name(), "test file", "create it")
	if err != nil {
		t.Errorf("ValidateRequiredFile() failed for existing file: %v", err)
	}
}

func TestValidateRequiredFile_FileMissing(t *testing.T) {
	// Validate non-existent file
	err := ValidateRequiredFile("/tmp/nonexistent-file-12345.txt", "test file", "create it")
	if err == nil {
		t.Error("ValidateRequiredFile() should return error for missing file")
	}

	// Check error message contains helpful info
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Error message should not be empty")
	}
}

func TestSuggestNextSteps_NoSpecular(t *testing.T) {
	// Create a temporary directory without .specular
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	suggestion := SuggestNextSteps()
	if suggestion != "Run 'specular init' to set up your project" {
		t.Errorf("SuggestNextSteps() = %q, want init suggestion", suggestion)
	}
}

func TestSuggestNextSteps_NoSpec(t *testing.T) {
	// Create .specular directory but no spec file
	tmpDir := t.TempDir()
	specularDir := filepath.Join(tmpDir, ".specular")
	if err := os.MkdirAll(specularDir, 0755); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	suggestion := SuggestNextSteps()
	if suggestion != "Create a spec with 'specular interview' or 'specular spec generate'" {
		t.Errorf("SuggestNextSteps() = %q, want spec creation suggestion", suggestion)
	}
}

func TestSuggestNextSteps_NoLock(t *testing.T) {
	// Create .specular and spec file but no lock file
	tmpDir := t.TempDir()
	specularDir := filepath.Join(tmpDir, ".specular")
	if err := os.MkdirAll(specularDir, 0755); err != nil {
		t.Fatal(err)
	}

	specFile := filepath.Join(specularDir, "spec.yaml")
	if err := os.WriteFile(specFile, []byte("version: 1.0"), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	suggestion := SuggestNextSteps()
	if suggestion != "Lock your spec with 'specular spec lock'" {
		t.Errorf("SuggestNextSteps() = %q, want lock suggestion", suggestion)
	}
}

func TestSuggestNextSteps_NoPlan(t *testing.T) {
	// Create .specular, spec, and lock but no plan
	tmpDir := t.TempDir()
	specularDir := filepath.Join(tmpDir, ".specular")
	if err := os.MkdirAll(specularDir, 0755); err != nil {
		t.Fatal(err)
	}

	specFile := filepath.Join(specularDir, "spec.yaml")
	if err := os.WriteFile(specFile, []byte("version: 1.0"), 0644); err != nil {
		t.Fatal(err)
	}

	lockFile := filepath.Join(specularDir, "spec.lock.json")
	if err := os.WriteFile(lockFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	suggestion := SuggestNextSteps()
	if suggestion != "Generate a plan with 'specular plan'" {
		t.Errorf("SuggestNextSteps() = %q, want plan suggestion", suggestion)
	}
}

func TestSuggestNextSteps_AllExists(t *testing.T) {
	// Create everything
	tmpDir := t.TempDir()
	specularDir := filepath.Join(tmpDir, ".specular")
	if err := os.MkdirAll(specularDir, 0755); err != nil {
		t.Fatal(err)
	}

	specFile := filepath.Join(specularDir, "spec.yaml")
	if err := os.WriteFile(specFile, []byte("version: 1.0"), 0644); err != nil {
		t.Fatal(err)
	}

	lockFile := filepath.Join(specularDir, "spec.lock.json")
	if err := os.WriteFile(lockFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	planFile := filepath.Join(tmpDir, "plan.json")
	if err := os.WriteFile(planFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	suggestion := SuggestNextSteps()
	if suggestion != "Execute your plan with 'specular build'" {
		t.Errorf("SuggestNextSteps() = %q, want build suggestion", suggestion)
	}
}
