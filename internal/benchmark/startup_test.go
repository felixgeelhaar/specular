package benchmark

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMeasureStartup(t *testing.T) {
	// Build a test binary
	tmpDir := t.TempDir()
	binary := filepath.Join(tmpDir, "test-binary")

	// Use a simple go program for testing
	cmd := exec.Command("go", "build", "-o", binary, "-ldflags=-s -w", "../../cmd/specular")
	cmd.Dir = filepath.Join("..", "..")
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to build test binary: %v", err)
	}

	result, err := MeasureStartup(binary, []string{"version"}, 5)
	if err != nil {
		t.Fatalf("MeasureStartup failed: %v", err)
	}

	if result.Iterations != 5 {
		t.Errorf("Expected 5 iterations, got %d", result.Iterations)
	}

	if result.Avg <= 0 {
		t.Error("Expected positive average duration")
	}

	if result.Min > result.Max {
		t.Error("Min should be <= Max")
	}

	if result.P50 < result.Min || result.P50 > result.Max {
		t.Error("P50 should be between Min and Max")
	}
}

func TestGetBinaryInfo(t *testing.T) {
	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-file")
	if err := os.WriteFile(testFile, make([]byte, 1024), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := GetBinaryInfo(testFile)
	if err != nil {
		t.Fatalf("GetBinaryInfo failed: %v", err)
	}

	if info.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", info.Size)
	}

	if info.SizeHuman != "1.0 KB" {
		t.Errorf("Expected '1.0 KB', got '%s'", info.SizeHuman)
	}
}

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{10485760, "10.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := humanBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("humanBytes(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}

func BenchmarkStartupVersion(b *testing.B) {
	// Build binary once
	tmpDir := b.TempDir()
	binary := filepath.Join(tmpDir, "specular")

	cmd := exec.Command("go", "build", "-o", binary, "-ldflags=-s -w", "../../cmd/specular")
	cmd.Dir = filepath.Join("..", "..")
	if err := cmd.Run(); err != nil {
		b.Skipf("Failed to build binary: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binary, "version")
		cmd.Run()
	}
}

func BenchmarkStartupHelp(b *testing.B) {
	tmpDir := b.TempDir()
	binary := filepath.Join(tmpDir, "specular")

	cmd := exec.Command("go", "build", "-o", binary, "-ldflags=-s -w", "../../cmd/specular")
	cmd.Dir = filepath.Join("..", "..")
	if err := cmd.Run(); err != nil {
		b.Skipf("Failed to build binary: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binary, "--help")
		cmd.Run()
	}
}
