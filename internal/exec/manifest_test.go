package exec

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateManifest(t *testing.T) {
	step := Step{
		ID:      "test-task-1",
		Runner:  "docker",
		Image:   "golang:1.22",
		Cmd:     []string{"go", "version"},
		Workdir: "/workspace",
		Env: map[string]string{
			"FOO": "bar",
		},
	}

	result := &Result{
		ExitCode: 0,
		Stdout:   "go version go1.22.0 linux/amd64",
		Stderr:   "",
		Duration: 1 * time.Second,
	}

	manifest := CreateManifest(step, result)

	if manifest.StepID != step.ID {
		t.Errorf("CreateManifest() StepID = %v, want %v", manifest.StepID, step.ID)
	}
	if manifest.Runner != step.Runner {
		t.Errorf("CreateManifest() Runner = %v, want %v", manifest.Runner, step.Runner)
	}
	if manifest.Image != step.Image {
		t.Errorf("CreateManifest() Image = %v, want %v", manifest.Image, step.Image)
	}
	if manifest.ExitCode != result.ExitCode {
		t.Errorf("CreateManifest() ExitCode = %v, want %v", manifest.ExitCode, result.ExitCode)
	}
	if manifest.Duration != result.Duration.String() {
		t.Errorf("CreateManifest() Duration = %v, want %v", manifest.Duration, result.Duration.String())
	}
	if len(manifest.Command) != len(step.Cmd) {
		t.Errorf("CreateManifest() Command length = %v, want %v", len(manifest.Command), len(step.Cmd))
	}
	if manifest.Env["FOO"] != "bar" {
		t.Errorf("CreateManifest() Env[FOO] = %v, want %v", manifest.Env["FOO"], "bar")
	}
}

func TestSaveManifest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Create temporary directory
		tmpDir := t.TempDir()

		manifest := &RunManifest{
			Timestamp:    time.Now(),
			StepID:       "test-task-1",
			Runner:       "docker",
			Image:        "golang:1.22",
			Command:      []string{"go", "version"},
			ExitCode:     0,
			Duration:     "1s",
			InputHashes:  map[string]string{"input.go": "abc123"},
			OutputHashes: map[string]string{"output.bin": "def456"},
		}

		err := SaveManifest(manifest, tmpDir)
		if err != nil {
			t.Fatalf("SaveManifest() error = %v", err)
		}

		// Check if file was created
		files, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatalf("ReadDir() error = %v", err)
		}

		if len(files) != 1 {
			t.Errorf("SaveManifest() created %d files, want 1", len(files))
		}

		// Verify filename format
		filename := files[0].Name()
		if !contains(filename, manifest.StepID) {
			t.Errorf("SaveManifest() filename %q does not contain step ID %q", filename, manifest.StepID)
		}
		if !contains(filename, ".json") {
			t.Errorf("SaveManifest() filename %q does not have .json extension", filename)
		}
	})

	t.Run("write error - read-only directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a read-only directory by creating a file in the path
		// where we want to write the manifest
		readOnlyFile := filepath.Join(tmpDir, "readonly")
		if err := os.WriteFile(readOnlyFile, []byte("readonly"), 0444); err != nil {
			t.Fatalf("Setup failed: WriteFile() error = %v", err)
		}

		manifest := &RunManifest{
			Timestamp: time.Now(),
			StepID:    "test-task-1",
		}

		// Try to save manifest to a subdirectory of the readonly file (will fail)
		invalidDir := filepath.Join(readOnlyFile, "subdir")
		err := SaveManifest(manifest, invalidDir)
		if err == nil {
			t.Error("SaveManifest() expected error for invalid directory, got nil")
		}
	})
}

func TestHashFile(t *testing.T) {
	// Create temporary file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("Hello, World!")

	err := os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Hash the file
	hash, err := HashFile(testFile)
	if err != nil {
		t.Fatalf("HashFile() error = %v", err)
	}

	// Verify hash is not empty
	if hash == "" {
		t.Error("HashFile() returned empty hash")
	}

	// Verify hash is consistent
	hash2, err := HashFile(testFile)
	if err != nil {
		t.Fatalf("HashFile() second call error = %v", err)
	}

	if hash != hash2 {
		t.Errorf("HashFile() inconsistent: first=%q, second=%q", hash, hash2)
	}

	// Verify hash length (SHA-256 produces 64 hex characters)
	if len(hash) != 64 {
		t.Errorf("HashFile() hash length = %d, want 64", len(hash))
	}
}

func TestHashFileNotFound(t *testing.T) {
	_, err := HashFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("HashFile() expected error for nonexistent file, got nil")
	}
}

func TestAddInputHash(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "input.txt")
		content := []byte("input data")

		err := os.WriteFile(testFile, content, 0644)
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		manifest := &RunManifest{
			InputHashes: make(map[string]string),
		}

		err = manifest.AddInputHash("input.txt", testFile)
		if err != nil {
			t.Fatalf("AddInputHash() error = %v", err)
		}

		hash, exists := manifest.InputHashes["input.txt"]
		if !exists {
			t.Error("AddInputHash() did not add hash to manifest")
		}
		if hash == "" {
			t.Error("AddInputHash() added empty hash")
		}
		if len(hash) != 64 {
			t.Errorf("AddInputHash() hash length = %d, want 64", len(hash))
		}
	})

	t.Run("file not found", func(t *testing.T) {
		manifest := &RunManifest{
			InputHashes: make(map[string]string),
		}

		err := manifest.AddInputHash("missing.txt", "/nonexistent/file.txt")
		if err == nil {
			t.Error("AddInputHash() expected error for nonexistent file, got nil")
		}
	})
}

func TestAddOutputHash(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "output.bin")
		content := []byte("output data")

		err := os.WriteFile(testFile, content, 0644)
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		manifest := &RunManifest{
			OutputHashes: make(map[string]string),
		}

		err = manifest.AddOutputHash("output.bin", testFile)
		if err != nil {
			t.Fatalf("AddOutputHash() error = %v", err)
		}

		hash, exists := manifest.OutputHashes["output.bin"]
		if !exists {
			t.Error("AddOutputHash() did not add hash to manifest")
		}
		if hash == "" {
			t.Error("AddOutputHash() added empty hash")
		}
		if len(hash) != 64 {
			t.Errorf("AddOutputHash() hash length = %d, want 64", len(hash))
		}
	})

	t.Run("file not found", func(t *testing.T) {
		manifest := &RunManifest{
			OutputHashes: make(map[string]string),
		}

		err := manifest.AddOutputHash("missing.bin", "/nonexistent/output.bin")
		if err == nil {
			t.Error("AddOutputHash() expected error for nonexistent file, got nil")
		}
	})
}
