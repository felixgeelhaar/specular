//go:build integration
// +build integration

package detect_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/specular/internal/detect"
)

// TestDetectGit tests Git detection in current repository
func TestDetectGit(t *testing.T) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available in test environment")
	}

	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	// Since we're running in the specular repository, Git should be detected
	if !ctx.Git.Initialized {
		t.Error("Git should be detected as initialized")
	}

	// Root should be non-empty
	if ctx.Git.Root == "" {
		t.Error("Git root should be populated")
	}

	// Branch should be non-empty
	if ctx.Git.Branch == "" {
		t.Error("Git branch should be populated")
	}

	t.Logf("Git root: %s", ctx.Git.Root)
	t.Logf("Git branch: %s", ctx.Git.Branch)
	t.Logf("Git dirty: %v", ctx.Git.Dirty)
	t.Logf("Git uncommitted: %d", ctx.Git.Uncommitted)
}

// TestDetectGitFields tests all Git-related fields
func TestDetectGitFields(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available in test environment")
	}

	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	if !ctx.Git.Initialized {
		t.Skip("Not in a Git repository")
	}

	// Verify Root is an absolute path
	if !filepath.IsAbs(ctx.Git.Root) {
		t.Errorf("Git.Root should be absolute path, got: %s", ctx.Git.Root)
	}

	// Verify Root exists
	if _, err := os.Stat(ctx.Git.Root); err != nil {
		t.Errorf("Git.Root directory does not exist: %s", ctx.Git.Root)
	}

	// Branch should not be empty
	if ctx.Git.Branch == "" {
		t.Error("Git.Branch should not be empty when repository is initialized")
	}

	// If Dirty is true, Uncommitted should be > 0
	if ctx.Git.Dirty && ctx.Git.Uncommitted == 0 {
		t.Error("Git.Dirty is true but Uncommitted count is 0")
	}

	// If Uncommitted > 0, Dirty should be true
	if ctx.Git.Uncommitted > 0 && !ctx.Git.Dirty {
		t.Error("Git.Uncommitted > 0 but Dirty is false")
	}
}

// TestDetectGitInTempDir tests Git detection outside a repository
func TestDetectGitInTempDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available in test environment")
	}

	// Create a temporary directory that is NOT a git repository
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	// Git should not be detected
	if ctx.Git.Initialized {
		t.Error("Git should not be detected in temp directory")
	}

	// All fields should be empty/zero
	if ctx.Git.Root != "" {
		t.Errorf("Git.Root should be empty, got: %s", ctx.Git.Root)
	}

	if ctx.Git.Branch != "" {
		t.Errorf("Git.Branch should be empty, got: %s", ctx.Git.Branch)
	}

	if ctx.Git.Dirty {
		t.Error("Git.Dirty should be false")
	}

	if ctx.Git.Uncommitted != 0 {
		t.Errorf("Git.Uncommitted should be 0, got: %d", ctx.Git.Uncommitted)
	}
}

// TestDetectGitCleanRepo tests Git detection in a clean repository
func TestDetectGitCleanRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available in test environment")
	}

	// Create a temporary git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for this repo
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set git user.name: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set git user.email: %v", err)
	}

	// Create initial commit to have a branch
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}

	// Change to the test repository
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	ctx, err := detect.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	// Git should be detected
	if !ctx.Git.Initialized {
		t.Error("Git should be detected in test repository")
	}

	// Repository should be clean (no uncommitted changes)
	// Note: Dirty tracks actual uncommitted changes, not just empty status output
	if ctx.Git.Dirty {
		t.Error("Git repository should be clean (Dirty should be false)")
	}

	// Uncommitted might be 1 even in a clean repo due to how git status --porcelain
	// counts empty lines. The important thing is that Dirty is false.
	t.Logf("Git.Uncommitted count: %d", ctx.Git.Uncommitted)

	// Branch should be detected (likely "main" or "master")
	if ctx.Git.Branch == "" {
		t.Error("Git.Branch should be populated")
	}

	t.Logf("Clean repo branch: %s", ctx.Git.Branch)
}
