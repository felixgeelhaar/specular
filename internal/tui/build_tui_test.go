package tui

import (
	"bytes"
	"errors"
	"testing"
)

func TestNewBuildTUI(t *testing.T) {
	config := BuildTUIConfig{
		SpecFile:  "spec.yaml",
		OutputDir: "./out",
		Format:    "yaml",
		Validate:  true,
		ShowDiff:  false,
	}

	var buf bytes.Buffer
	tui := NewBuildTUI(config, &buf)

	if tui == nil {
		t.Fatal("NewBuildTUI returned nil")
	}
	if tui.adapter == nil {
		t.Error("Adapter should not be nil")
	}
	if tui.config.SpecFile != "spec.yaml" {
		t.Errorf("Expected spec file 'spec.yaml', got %q", tui.config.SpecFile)
	}
}

func TestNewBuildTUI_NilOutput(t *testing.T) {
	config := BuildTUIConfig{
		SpecFile: "spec.yaml",
	}

	// Should default to os.Stdout
	tui := NewBuildTUI(config, nil)
	if tui == nil {
		t.Fatal("NewBuildTUI returned nil")
	}
}

func TestBuildPhases(t *testing.T) {
	expectedPhases := []string{
		"Load spec file",
		"Validate spec",
		"Resolve dependencies",
		"Generate output",
		"Write files",
	}

	if len(BuildPhases) != len(expectedPhases) {
		t.Errorf("Expected %d phases, got %d", len(expectedPhases), len(BuildPhases))
	}

	for i, phase := range expectedPhases {
		if BuildPhases[i] != phase {
			t.Errorf("Phase %d: expected %q, got %q", i, phase, BuildPhases[i])
		}
	}
}

func TestBuildTUI_OnLoadSpec(t *testing.T) {
	config := BuildTUIConfig{
		SpecFile: "spec.yaml",
	}

	var buf bytes.Buffer
	tui := NewBuildTUI(config, &buf)

	// Test start
	tui.OnLoadSpec(true, "spec.yaml", nil)
	// Test complete
	tui.OnLoadSpec(false, "spec.yaml", nil)
	// Test error
	tui.OnLoadSpec(false, "spec.yaml", errors.New("file not found"))

	// No panic means success
}

func TestBuildTUI_OnValidate(t *testing.T) {
	config := BuildTUIConfig{
		SpecFile: "spec.yaml",
	}

	var buf bytes.Buffer
	tui := NewBuildTUI(config, &buf)

	// Test start
	tui.OnValidate(true, 0, 0, nil)
	// Test complete with warnings
	tui.OnValidate(false, 2, 0, nil)
	// Test complete without warnings
	tui.OnValidate(false, 0, 0, nil)
	// Test error
	tui.OnValidate(false, 0, 1, errors.New("validation failed"))
}

func TestBuildTUI_OnResolve(t *testing.T) {
	config := BuildTUIConfig{
		SpecFile: "spec.yaml",
	}

	var buf bytes.Buffer
	tui := NewBuildTUI(config, &buf)

	// Test start
	tui.OnResolve(true, 0, nil)
	// Test complete
	tui.OnResolve(false, 5, nil)
	// Test error
	tui.OnResolve(false, 0, errors.New("resolution failed"))
}

func TestBuildTUI_OnGenerate(t *testing.T) {
	config := BuildTUIConfig{
		SpecFile: "spec.yaml",
		Format:   "yaml",
	}

	var buf bytes.Buffer
	tui := NewBuildTUI(config, &buf)

	// Test start
	tui.OnGenerate(true, "yaml", nil)
	// Test complete
	tui.OnGenerate(false, "yaml", nil)
	// Test error
	tui.OnGenerate(false, "yaml", errors.New("generation failed"))
}

func TestBuildTUI_OnWriteFile(t *testing.T) {
	config := BuildTUIConfig{
		SpecFile:  "spec.yaml",
		OutputDir: "./out",
	}

	var buf bytes.Buffer
	tui := NewBuildTUI(config, &buf)

	// Test success
	tui.OnWriteFile("./out/spec.yaml", 1024, nil)
	// Test error
	tui.OnWriteFile("./out/spec.yaml", 0, errors.New("write failed"))
}

func TestBuildTUI_OnWriteComplete(t *testing.T) {
	config := BuildTUIConfig{
		SpecFile: "spec.yaml",
	}

	var buf bytes.Buffer
	tui := NewBuildTUI(config, &buf)

	// Test success
	tui.OnWriteComplete(3, 4096, nil)
	// Test error
	tui.OnWriteComplete(0, 0, errors.New("write failed"))
}

func TestBuildTUI_Log(t *testing.T) {
	config := BuildTUIConfig{
		SpecFile: "spec.yaml",
	}

	var buf bytes.Buffer
	tui := NewBuildTUI(config, &buf)

	// Should not panic
	tui.Log(LogLevelInfo, "Test message")
	tui.Log(LogLevelWarn, "Warning message")
	tui.Log(LogLevelError, "Error message")
}

func TestBuildTUI_UpdateMetric(t *testing.T) {
	config := BuildTUIConfig{
		SpecFile: "spec.yaml",
	}

	var buf bytes.Buffer
	tui := NewBuildTUI(config, &buf)

	// Should not panic
	tui.UpdateMetric("files", 5)
	tui.UpdateMetric("size", 1024)
}

func TestBuildTUIRunner(t *testing.T) {
	config := BuildTUIConfig{
		SpecFile: "spec.yaml",
	}

	var buf bytes.Buffer
	runner := NewBuildTUIRunner(config, &buf)

	if runner == nil {
		t.Fatal("NewBuildTUIRunner returned nil")
	}
	if runner.tui == nil {
		t.Error("Runner tui should not be nil")
	}
}

func TestBuildTUIConfig(t *testing.T) {
	config := BuildTUIConfig{
		SpecFile:  "spec.yaml",
		OutputDir: "./output",
		Format:    "yaml",
		Validate:  true,
		ShowDiff:  true,
	}

	if config.SpecFile != "spec.yaml" {
		t.Errorf("Expected SpecFile 'spec.yaml', got %q", config.SpecFile)
	}
	if config.OutputDir != "./output" {
		t.Errorf("Expected OutputDir './output', got %q", config.OutputDir)
	}
	if config.Format != "yaml" {
		t.Errorf("Expected Format 'yaml', got %q", config.Format)
	}
	if !config.Validate {
		t.Error("Expected Validate to be true")
	}
	if !config.ShowDiff {
		t.Error("Expected ShowDiff to be true")
	}
}
