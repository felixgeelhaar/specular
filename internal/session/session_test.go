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

func TestStartWithWorktree(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)

	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal:       "create worktree session",
		Name:       "wt-sess",
		Detach:     false,
		NoApproval: true,
		Binary:     stub,
	})
	if err != nil {
		t.Fatalf("Start: %v (log=%s)", err, mustRead(t, rec.LogPath))
	}
	if rec.WorktreePath == "" {
		t.Fatal("expected worktree path")
	}
	if _, err := os.Stat(rec.WorktreePath); err != nil {
		t.Fatalf("worktree missing: %v", err)
	}
	if rec.Status != StatusCompleted {
		t.Fatalf("status=%s err=%s", rec.Status, rec.Error)
	}
}

func TestWaitCompletedAndFailed(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}

	okStub := writeExitStub(t, 0)
	failStub := writeExitStub(t, 1)

	okRec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "ok", Name: "wait-ok", Harness: "specular-auto",
		Detach: true, NoApproval: true, SkipWorktree: true, Binary: okStub,
	})
	if err != nil {
		t.Fatal(err)
	}
	failRec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "fail", Name: "wait-fail", Harness: "specular-auto",
		Detach: true, NoApproval: true, SkipWorktree: true, Binary: failStub,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = mgr.Wait(ctx, []string{okRec.ID}, WaitOptions{})
	if err != nil {
		t.Fatalf("wait ok: %v", err)
	}

	_, err = mgr.Wait(ctx, []string{failRec.ID}, WaitOptions{})
	if err == nil {
		t.Fatal("expected failure from failed session")
	}
}

func TestWaitTimeout(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeSleepStub(t)
	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "slow", Name: "wait-timeout", Harness: "specular-auto",
		Detach: true, NoApproval: true, SkipWorktree: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = mgr.Stop(rec.ID) }()

	_, err = mgr.Wait(context.Background(), []string{rec.ID}, WaitOptions{
		Timeout:  200 * time.Millisecond,
		Interval: 50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout")
	}
}

func TestWaitAny(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	fast := writeExitStub(t, 0)
	slow := writeSleepStub(t)

	a, err := mgr.Start(context.Background(), StartOptions{
		Goal: "a", Name: "wait-any-a", Harness: "specular-auto",
		Detach: true, NoApproval: true, SkipWorktree: true, Binary: fast,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := mgr.Start(context.Background(), StartOptions{
		Goal: "b", Name: "wait-any-b", Harness: "specular-auto",
		Detach: true, NoApproval: true, SkipWorktree: true, Binary: slow,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = mgr.Stop(b.ID) }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done, err := mgr.Wait(ctx, []string{a.ID, b.ID}, WaitOptions{Any: true, Interval: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if len(done) != 1 || done[0].ID != a.ID {
		t.Fatalf("done=%+v", done)
	}
}

func TestRestartSwitchesHarness(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)

	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "first", Name: "restart-me", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	wt := rec.WorktreePath
	if wt == "" {
		t.Fatal("expected worktree")
	}

	// Pretend claude is available via absolute binary override.
	restarted, err := mgr.Restart(context.Background(), rec.ID, RestartOptions{
		Harness:    "claude-code",
		Goal:       "second pass",
		Detach:     false,
		NoApproval: true,
		Binary:     stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	if restarted.Harness != "claude-code" {
		t.Fatalf("harness=%s", restarted.Harness)
	}
	if restarted.Goal != "second pass" {
		t.Fatalf("goal=%s", restarted.Goal)
	}
	if restarted.WorktreePath != wt {
		t.Fatalf("worktree changed: %s vs %s", restarted.WorktreePath, wt)
	}
	if restarted.Status != StatusCompleted {
		t.Fatalf("status=%s", restarted.Status)
	}
}

func TestRestartRequiresForceWhenRunning(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeSleepStub(t)
	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "run", Name: "force-me", Harness: "specular-auto",
		Detach: true, NoApproval: true, SkipWorktree: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = mgr.Stop(rec.ID) }()

	_, err = mgr.Restart(context.Background(), rec.ID, RestartOptions{
		Detach: true, NoApproval: true, Binary: stub,
	})
	if err == nil {
		t.Fatal("expected error without --force")
	}

	restarted, err := mgr.Restart(context.Background(), rec.ID, RestartOptions{
		Force: true, Detach: true, NoApproval: true, Binary: writeExitStub(t, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if restarted.Status != StatusWorking && restarted.Status != StatusCompleted {
		t.Fatalf("status=%s", restarted.Status)
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
