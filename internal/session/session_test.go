package session

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/attestation"
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

func TestWaitTimeoutStop(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeSleepStub(t)
	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "slow", Name: "wait-stop", Harness: "specular-auto",
		Detach: true, NoApproval: true, SkipWorktree: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}

	done, err := mgr.Wait(context.Background(), []string{rec.ID}, WaitOptions{
		Timeout:       200 * time.Millisecond,
		Interval:      50 * time.Millisecond,
		StopOnTimeout: true,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "stopped:") {
		t.Fatalf("error=%v", err)
	}
	if len(done) != 1 || done[0].Status != StatusStopped {
		t.Fatalf("done=%+v", done)
	}
	got, getErr := mgr.Get(rec.ID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if got.Status != StatusStopped || (got.PID > 0 && processAlive(got.PID)) {
		t.Fatalf("expected stopped process: %+v", got)
	}
}

func TestStopAll(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeSleepStub(t)
	a, err := mgr.Start(context.Background(), StartOptions{
		Goal: "a", Name: "stop-all-a", Harness: "specular-auto",
		Detach: true, NoApproval: true, SkipWorktree: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := mgr.Start(context.Background(), StartOptions{
		Goal: "b", Name: "stop-all-b", Harness: "specular-auto",
		Detach: true, NoApproval: true, SkipWorktree: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	stopped, err := mgr.StopAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(stopped) < 2 {
		t.Fatalf("stopped=%+v", stopped)
	}
	for _, id := range []string{a.ID, b.ID} {
		got, getErr := mgr.Get(id)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if got.Status != StatusStopped {
			t.Fatalf("%s status=%s", id, got.Status)
		}
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

func TestRestartPreservesNoGovernedWithPolicy(t *testing.T) {
	repo := initTempRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, ".specular"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".specular", "policy.yaml"), []byte("routing:\n  deny_tools: [web_search]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)
	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "ungoverned", Name: "restart-nogov", Harness: "claude-code",
		Detach: false, NoApproval: true, Binary: stub, NoGoverned: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Governed {
		t.Fatal("expected start with NoGoverned to stay ungoverned")
	}
	// Inherit: restart must not re-enable auto-govern from policy.yaml.
	restarted, err := mgr.Restart(context.Background(), rec.ID, RestartOptions{
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	if restarted.Governed {
		t.Fatal("inherit restart should keep ungoverned despite policy.yaml")
	}
	// Explicit --no-governed after a governed start.
	gov, err := mgr.Start(context.Background(), StartOptions{
		Goal: "gov", Name: "restart-gov", Harness: "codex",
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !gov.Governed {
		t.Fatal("expected auto-governed from policy")
	}
	optOut, err := mgr.Restart(context.Background(), gov.ID, RestartOptions{
		Detach: false, NoApproval: true, Binary: stub,
		UseNoGoverned: true, NoGoverned: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if optOut.Governed {
		t.Fatal("--no-governed should win on restart")
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

func TestRemoveAndPrune(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)

	done, err := mgr.Start(context.Background(), StartOptions{
		Goal: "cleanup me", Name: "rm-me", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	wt := done.WorktreePath
	if wt == "" {
		t.Fatal("expected worktree")
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("worktree missing before remove: %v", err)
	}

	// Running session should require --force.
	live, err := mgr.Start(context.Background(), StartOptions{
		Goal: "live", Name: "still-running", Harness: "specular-auto",
		Detach: true, NoApproval: true, SkipWorktree: true, Binary: writeSleepStub(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = mgr.Stop(live.ID) }()

	if _, err := mgr.Remove(context.Background(), live.ID, RemoveOptions{}); err == nil {
		t.Fatal("expected error removing running session without force")
	}

	snap, err := mgr.Remove(context.Background(), done.ID, RemoveOptions{DeleteBranch: true})
	if err != nil {
		t.Fatal(err)
	}
	if snap.ID != "rm-me" {
		t.Fatalf("id=%s", snap.ID)
	}
	if _, err := mgr.Get("rm-me"); err == nil {
		t.Fatal("expected session gone after remove")
	}
	if _, err := os.Stat(filepath.Join(mgr.Store().Dir(), "rm-me.json")); !os.IsNotExist(err) {
		t.Fatalf("json should be gone: %v", err)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Fatalf("worktree should be gone: %v", err)
	}

	// Prune terminal sessions, keep worktrees for a second completed one.
	keepWT, err := mgr.Start(context.Background(), StartOptions{
		Goal: "prune keep wt", Name: "prune-me", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	pruneWT := keepWT.WorktreePath

	removed, err := mgr.Prune(context.Background(), PruneOptions{KeepWorktree: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) < 1 {
		t.Fatalf("expected pruned sessions, got %d", len(removed))
	}
	if _, err := mgr.Get("prune-me"); err == nil {
		t.Fatal("expected prune-me gone")
	}
	if _, err := os.Stat(pruneWT); err != nil {
		t.Fatalf("worktree should remain with KeepWorktree: %v", err)
	}

	// Active session must survive prune.
	if _, err := mgr.Get(live.ID); err != nil {
		t.Fatalf("active session should remain: %v", err)
	}
}

func TestSessionDiff(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)

	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "diff me", Name: "diff-a", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.WorktreePath == "" {
		t.Fatal("expected worktree")
	}

	changed := filepath.Join(rec.WorktreePath, "feature.txt")
	if err := os.WriteFile(changed, []byte("hello from session\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := mgr.Diff(context.Background(), rec.ID, DiffOptions{NameOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Base == "" {
		t.Fatal("expected base")
	}
	if !strings.Contains(res.Output, "feature.txt") {
		t.Fatalf("expected feature.txt in diff output, got %q", res.Output)
	}

	other, err := mgr.Start(context.Background(), StartOptions{
		Goal: "other", Name: "diff-b", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	otherFile := filepath.Join(other.WorktreePath, "other.txt")
	if err := os.WriteFile(otherFile, []byte("other\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, other.WorktreePath, "git", "add", "other.txt")
	run(t, other.WorktreePath, "git", "commit", "-m", "other change")

	run(t, rec.WorktreePath, "git", "add", "feature.txt")
	run(t, rec.WorktreePath, "git", "commit", "-m", "feature change")

	cross, err := mgr.Diff(context.Background(), rec.ID, DiffOptions{
		Against: other.ID,
		Stat:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cross.AgainstID != other.ID {
		t.Fatalf("against=%s", cross.AgainstID)
	}
	if strings.TrimSpace(cross.Output) == "" {
		t.Fatal("expected non-empty cross-session diff")
	}
}

func TestSessionExec(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)
	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "exec me", Name: "exec-a", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.WorktreePath == "" {
		t.Fatal("expected worktree")
	}

	marker := filepath.Join(rec.WorktreePath, "from-exec.txt")
	res, err := mgr.Exec(context.Background(), rec.ID, []string{"sh", "-c", "echo hi > from-exec.txt"}, ExecOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 || res.SessionID != rec.ID {
		t.Fatalf("%+v", res)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("command did not run in worktree: %v", err)
	}

	fail, err := mgr.Exec(context.Background(), rec.ID, []string{"sh", "-c", "exit 7"}, ExecOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if fail.ExitCode != 7 {
		t.Fatalf("exit=%d want 7", fail.ExitCode)
	}

	if _, err := mgr.Exec(context.Background(), rec.ID, nil, ExecOptions{}); err == nil {
		t.Fatal("expected empty argv error")
	}

	noWT, err := mgr.Start(context.Background(), StartOptions{
		Goal: "no wt", Name: "exec-nowt", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub, SkipWorktree: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Exec(context.Background(), noWT.ID, []string{"true"}, ExecOptions{}); err == nil {
		t.Fatal("expected no-worktree error")
	}
}

func TestSessionCommit(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)
	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "commit me please", Name: "commit-a", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := mgr.Commit(context.Background(), rec.ID, CommitOptions{}); err == nil {
		t.Fatal("expected nothing-to-commit error")
	}

	tracked := filepath.Join(rec.WorktreePath, "README.md")
	if err := os.WriteFile(tracked, []byte("updated by session\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := mgr.Commit(context.Background(), rec.ID, CommitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.SHA == "" || !strings.Contains(res.Message, "commit-a") {
		t.Fatalf("%+v", res)
	}

	untracked := filepath.Join(rec.WorktreePath, "new-file.txt")
	if err := os.WriteFile(untracked, []byte("brand new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Commit(context.Background(), rec.ID, CommitOptions{}); err == nil {
		t.Fatal("expected nothing-to-commit without --all for untracked-only")
	}
	allRes, err := mgr.Commit(context.Background(), rec.ID, CommitOptions{All: true, Message: "add new-file"})
	if err != nil {
		t.Fatal(err)
	}
	if allRes.SHA == res.SHA {
		t.Fatal("expected new commit SHA after --all")
	}

	noWT, err := mgr.Start(context.Background(), StartOptions{
		Goal: "no wt", Name: "commit-nowt", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub, SkipWorktree: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Commit(context.Background(), noWT.ID, CommitOptions{AllowEmpty: true}); err == nil {
		t.Fatal("expected no-worktree error")
	}
}

func TestSessionPush(t *testing.T) {
	repo := initTempRepo(t)
	bare := t.TempDir()
	run(t, bare, "git", "init", "--bare")
	run(t, repo, "git", "remote", "add", "origin", bare)

	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)
	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "push me", Name: "push-a", Harness: "claude-code",
		Detach: false, NoApproval: true, Binary: stub, Governed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rec.WorktreePath, "README.md"), []byte("pushed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Commit(context.Background(), rec.ID, CommitOptions{Message: "ready to push"}); err != nil {
		t.Fatal(err)
	}
	res, err := mgr.Push(context.Background(), rec.ID, PushOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Remote != "origin" || res.Branch == "" || res.SHA == "" {
		t.Fatalf("%+v", res)
	}
	// Verify bare remote received the branch.
	cmd := exec.Command("git", "branch", "--list", rec.WorktreeBranch)
	cmd.Dir = bare
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list remote branches: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), rec.WorktreeBranch) {
		t.Fatalf("branch missing on remote: %q", out)
	}

	noWT, err := mgr.Start(context.Background(), StartOptions{
		Goal: "no wt", Name: "push-nowt", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub, SkipWorktree: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Push(context.Background(), noWT.ID, PushOptions{}); err == nil {
		t.Fatal("expected no-worktree error")
	}
}

func TestSessionSync(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)
	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "sync me", Name: "sync-a", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Session-side change.
	sessFile := filepath.Join(rec.WorktreePath, "session.txt")
	if err := os.WriteFile(sessFile, []byte("from session\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Commit(context.Background(), rec.ID, CommitOptions{All: true, Message: "session change"}); err != nil {
		t.Fatal(err)
	}

	// Advance base in the primary checkout.
	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("from base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, repo, "git", "add", "base.txt")
	run(t, repo, "git", "commit", "-m", "base advance")

	res, err := mgr.Sync(context.Background(), rec.ID, SyncOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.BeforeSHA == "" || res.AfterSHA == "" || res.Strategy != "rebase" {
		t.Fatalf("%+v", res)
	}
	if res.AfterSHA == res.BeforeSHA {
		t.Fatal("expected HEAD to move after sync onto advanced base")
	}
	// Session commit should still be reachable and base.txt present after rebase.
	if _, err := os.Stat(filepath.Join(rec.WorktreePath, "base.txt")); err != nil {
		t.Fatalf("base.txt missing after sync: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rec.WorktreePath, "session.txt")); err != nil {
		t.Fatalf("session.txt missing after sync: %v", err)
	}

	// Dirty refusal.
	if err := os.WriteFile(filepath.Join(rec.WorktreePath, "dirty.txt"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Sync(context.Background(), rec.ID, SyncOptions{}); err == nil {
		t.Fatal("expected dirty error")
	}
	if _, err := mgr.Sync(context.Background(), rec.ID, SyncOptions{Autostash: true}); err != nil {
		t.Fatalf("autostash sync: %v", err)
	}

	noWT, err := mgr.Start(context.Background(), StartOptions{
		Goal: "no wt", Name: "sync-nowt", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub, SkipWorktree: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Sync(context.Background(), noWT.ID, SyncOptions{}); err == nil {
		t.Fatal("expected no-worktree error")
	}
}

func TestSessionSyncConflict(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)
	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "conflict", Name: "sync-conflict", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rec.WorktreePath, "README.md"), []byte("session edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Commit(context.Background(), rec.ID, CommitOptions{Message: "session readme"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, repo, "git", "add", "README.md")
	run(t, repo, "git", "commit", "-m", "base readme")

	res, err := mgr.Sync(context.Background(), rec.ID, SyncOptions{})
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if res == nil {
		t.Fatal("expected result with conflict metadata")
	}
	// After abort, worktree should not be mid-rebase.
	cmd := exec.Command("git", "rev-parse", "--git-path", "rebase-merge")
	cmd.Dir = rec.WorktreePath
	out, _ := cmd.Output()
	rebasePath := strings.TrimSpace(string(out))
	if rebasePath != "" {
		if !filepath.IsAbs(rebasePath) {
			rebasePath = filepath.Join(rec.WorktreePath, rebasePath)
		}
		if st, stErr := os.Stat(rebasePath); stErr == nil && st.IsDir() {
			t.Fatal("rebase should have been aborted")
		}
	}
}

func TestSessionMerge(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)

	// Fast-forward: session tip is a linear descendant of main.
	ffRec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "ff land", Name: "merge-ff", Harness: "claude-code",
		Detach: false, NoApproval: true, Binary: stub, Governed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ffRec.WorktreePath, "ff.txt"), []byte("ff\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Commit(context.Background(), ffRec.ID, CommitOptions{All: true, Message: "ff tip"}); err != nil {
		t.Fatal(err)
	}
	ff, err := mgr.Merge(context.Background(), ffRec.ID, MergeOptions{FFOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if ff.Strategy != "ff-only" || ff.AfterSHA == ff.BeforeSHA {
		t.Fatalf("%+v", ff)
	}
	if _, err := os.Stat(filepath.Join(repo, "ff.txt")); err != nil {
		t.Fatalf("ff.txt missing after ff-only: %v", err)
	}

	// No-ff: force a merge commit even when FF would work.
	nfRec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "noff land", Name: "merge-noff", Harness: "codex",
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nfRec.WorktreePath, "noff.txt"), []byte("noff\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Commit(context.Background(), nfRec.ID, CommitOptions{All: true, Message: "noff tip"}); err != nil {
		t.Fatal(err)
	}
	nf, err := mgr.Merge(context.Background(), nfRec.ID, MergeOptions{NoFF: true})
	if err != nil {
		t.Fatal(err)
	}
	if nf.Strategy != "no-ff" || nf.AfterSHA == nf.BeforeSHA {
		t.Fatalf("%+v", nf)
	}
	if _, err := os.Stat(filepath.Join(repo, "noff.txt")); err != nil {
		t.Fatalf("noff.txt missing after no-ff: %v", err)
	}
	// Merge commit has two parents.
	cmd := exec.Command("git", "rev-list", "--parents", "-n", "1", "HEAD")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rev-list: %v\n%s", err, out)
	}
	if len(strings.Fields(strings.TrimSpace(string(out)))) < 3 {
		t.Fatalf("expected merge commit with 2 parents, got %q", out)
	}

	noWT, err := mgr.Start(context.Background(), StartOptions{
		Goal: "no wt", Name: "merge-nowt", Harness: "specular-auto",
		Detach: false, NoApproval: true, Binary: stub, SkipWorktree: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Merge(context.Background(), noWT.ID, MergeOptions{}); err == nil {
		t.Fatal("expected no-worktree error")
	}
}

func TestSessionAttest(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)
	rec, err := mgr.Start(context.Background(), StartOptions{
		Goal: "attest me", Name: "attest-claude", Harness: "claude-code",
		Detach: false, NoApproval: true, Binary: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := mgr.Attest(context.Background(), rec.ID, AttestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Path == "" || res.Harness != "claude-code" {
		t.Fatalf("%+v", res)
	}
	data, readErr := os.ReadFile(res.Path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	att, parseErr := attestation.FromJSON(data)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	if att.Provenance.Harness != "claude-code" {
		t.Fatalf("harness=%s", att.Provenance.Harness)
	}
	if att.Provenance.WorktreePath == "" || att.Provenance.WorktreeBranch == "" {
		t.Fatalf("missing worktree provenance: %+v", att.Provenance)
	}
	if att.Signature == "" || att.PublicKey == "" {
		t.Fatal("expected signed attestation")
	}
	verifier := attestation.NewStandardVerifier()
	if err := verifier.Verify(att); err != nil {
		t.Fatalf("verify: %v", err)
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
