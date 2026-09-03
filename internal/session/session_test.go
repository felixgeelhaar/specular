package session

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestStoreSaveLoadList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	rec := &Record{
		ID:        "alpha",
		Goal:      "add health check",
		Harness:   "specular-auto",
		Status:    StatusWorking,
		CreatedAt: time.Now().UTC().Add(-time.Minute),
	}
	if err := store.Save(rec); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Goal != rec.Goal || loaded.Harness != rec.Harness {
		t.Fatalf("loaded mismatch: %+v", loaded)
	}

	rec2 := &Record{
		ID:        "beta",
		Goal:      "later",
		Harness:   "claude-code",
		Status:    StatusCompleted,
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Save(rec2); err != nil {
		t.Fatal(err)
	}

	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len=%d", len(list))
	}
	if list[0].ID != "beta" {
		t.Fatalf("expected newest first, got %s", list[0].ID)
	}
}

func TestStartDetachStop(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}

	// Use a long-running stub binary instead of real specular auto.
	stub := writeSleepStub(t)

	ctx := context.Background()
	rec, err := mgr.Start(ctx, StartOptions{
		Goal:         "noop goal for session manager",
		Name:         "parallel-a",
		Harness:      "specular-auto",
		Detach:       true,
		NoApproval:   true,
		SkipWorktree: true, // avoid needing full worktree for stub args
		Binary:       stub,
		ExtraArgs:    []string{"--duration", "30"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if rec.PID <= 0 {
		t.Fatal("expected pid")
	}
	if rec.Status != StatusWorking {
		t.Fatalf("status=%s", rec.Status)
	}
	if rec.Harness != "specular-auto" {
		t.Fatalf("harness=%s", rec.Harness)
	}

	// Give the stub a moment to start.
	time.Sleep(100 * time.Millisecond)
	if !processAlive(rec.PID) {
		t.Fatalf("stub pid %d not alive; log=%s", rec.PID, mustRead(t, rec.LogPath))
	}

	stopped, err := mgr.Stop(rec.ID)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if stopped.Status != StatusStopped {
		t.Fatalf("status=%s", stopped.Status)
	}

	time.Sleep(100 * time.Millisecond)
	if processAlive(rec.PID) {
		t.Fatal("expected process stopped")
	}
}

func TestFork(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)
	src, err := mgr.Start(context.Background(), StartOptions{
		Goal:       "source session",
		Name:       "src-sess",
		Detach:     false,
		NoApproval: true,
		Binary:     stub,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	forked, err := mgr.Fork(context.Background(), src.ID, "src-fork")
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if forked.ID != "src-fork" {
		t.Fatalf("id=%s", forked.ID)
	}
	if forked.Status != StatusIdle {
		t.Fatalf("status=%s", forked.Status)
	}
	if forked.WorktreePath == "" || forked.WorktreePath == src.WorktreePath {
		t.Fatalf("expected distinct worktree, got %s vs %s", forked.WorktreePath, src.WorktreePath)
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
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

func writeSleepStub(t *testing.T) string {
	t.Helper()
	// A tiny Go program would need compile time; use shell sleep via a script.
	path := filepath.Join(t.TempDir(), "stub.sh")
	script := "#!/bin/sh\n# ignore args; sleep until killed\nsleep 60\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeExitStub(t *testing.T, code int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stub.sh")
	script := "#!/bin/sh\nexit " + strconv.Itoa(code) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}
