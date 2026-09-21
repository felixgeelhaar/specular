package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"session-1", false},
		{"feat/auth", false},
		{"../escape", true},
		{"", true},
		{"has space", true},
		{"ok_name.2", false},
	}
	for _, tc := range cases {
		err := validateName(tc.name)
		if tc.wantErr && err == nil {
			t.Errorf("validateName(%q) expected error", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("validateName(%q) unexpected error: %v", tc.name, err)
		}
	}
}

func TestCreateListRemove(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	ctx := context.Background()
	info, err := mgr.Create(ctx, Options{Name: "parallel-a"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if info.Path == "" {
		t.Fatal("expected non-empty path")
	}
	if info.Branch != BranchPrefix+"parallel-a" {
		t.Fatalf("branch = %q, want %q", info.Branch, BranchPrefix+"parallel-a")
	}
	if !info.Managed {
		t.Fatal("expected Managed=true")
	}
	if _, err := os.Stat(info.Path); err != nil {
		t.Fatalf("worktree path missing: %v", err)
	}

	// Second worktree in parallel
	info2, err := mgr.Create(ctx, Options{Name: "parallel-b"})
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}
	if info2.Path == info.Path {
		t.Fatal("expected distinct worktree paths")
	}

	list, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	managed := 0
	for _, w := range list {
		if w.Managed {
			managed++
		}
	}
	if managed < 2 {
		t.Fatalf("expected >=2 managed worktrees, got %d (total %d)", managed, len(list))
	}

	if err := mgr.Remove(ctx, info.Path, true); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(info.Path); !os.IsNotExist(err) {
		t.Fatalf("expected worktree removed, stat err=%v", err)
	}

	if err := mgr.Remove(ctx, info2.Path, true); err != nil {
		t.Fatalf("Remove second: %v", err)
	}
}

func TestCreateExistingBranch(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	ctx := context.Background()

	// Pre-create branch
	run(t, repo, "git", "branch", "specular/reuse-me")

	info, err := mgr.Create(ctx, Options{Name: "reuse-me", Branch: "specular/reuse-me"})
	if err != nil {
		t.Fatalf("Create with existing branch: %v", err)
	}
	defer func() { _ = mgr.Remove(ctx, info.Path, false) }()

	if info.Branch != "specular/reuse-me" {
		t.Fatalf("branch = %q", info.Branch)
	}
}

func TestNotAGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := NewManager(dir)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestParsePorcelain(t *testing.T) {
	t.Parallel()
	out := "" +
		"worktree /repo\n" +
		"HEAD abc\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"worktree /repo/.specular/worktrees/s1\n" +
		"HEAD def\n" +
		"branch refs/heads/specular/s1\n" +
		"\n"
	infos := parsePorcelain(out, "/repo")
	if len(infos) != 2 {
		t.Fatalf("got %d infos", len(infos))
	}
	if infos[0].Managed {
		t.Error("main checkout should not be managed")
	}
	if !infos[1].Managed {
		t.Error("specular worktree should be managed")
	}
	if infos[1].Branch != "specular/s1" {
		t.Errorf("branch = %q", infos[1].Branch)
	}
}

func initTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@example.com")
	run(t, dir, "git", "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "README.md")
	run(t, dir, "git", "commit", "-m", "init")
	return dir
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}
