// Package session manages parallel Specular agent sessions across the
// authoring (inner) and governance (outer) loops.
//
// A session is a tracked auto-mode run, typically isolated in a Git worktree,
// with harness provenance that flows into attestations. This is Specular's
// CLI-native answer to agentic session managers: goal → isolated worktree →
// tracked parallel runs → the same drift/policy gate.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/safeutil"
	"github.com/felixgeelhaar/specular/internal/worktree"
)

const (
	// DefaultRelativeDir is where session records live under the repo root.
	DefaultRelativeDir = ".specular/sessions"

	// StatusWorking means the agent process is actively executing.
	StatusWorking = "working"
	// StatusIdle means the process is alive but waiting for input (reserved).
	StatusIdle = "idle"
	// StatusWaiting means the process is blocked on approval / policy.
	StatusWaiting = "waiting"
	// StatusCompleted means the run finished successfully.
	StatusCompleted = "completed"
	// StatusFailed means the run exited non-zero.
	StatusFailed = "failed"
	// StatusStopped means the run was stopped by the user.
	StatusStopped = "stopped"
)

// Record is the persisted identity of a managed session.
type Record struct {
	ID             string    `json:"id"`
	Goal           string    `json:"goal"`
	WorktreeName   string    `json:"worktreeName,omitempty"`
	WorktreePath   string    `json:"worktreePath,omitempty"`
	WorktreeBranch string    `json:"worktreeBranch,omitempty"`
	Harness        string    `json:"harness"`
	Profile        string    `json:"profile,omitempty"`
	CheckpointID   string    `json:"checkpointId,omitempty"`
	PID            int       `json:"pid,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	LogPath        string    `json:"logPath,omitempty"`
	Error          string    `json:"error,omitempty"`
	ExitCode       *int      `json:"exitCode,omitempty"`
}

// StartOptions configures a new parallel session.
type StartOptions struct {
	// Goal is the natural-language objective passed to specular auto.
	Goal string
	// Name becomes the session/worktree slug. Generated if empty.
	Name string
	// Harness is recorded in attestation provenance.
	Harness string
	// Profile selects an auto profile (default, ci, strict, …).
	Profile string
	// NoApproval skips the interactive approval gate (typical for detached runs).
	NoApproval bool
	// Detach starts auto in the background.
	Detach bool
	// Binary overrides the specular executable path (tests only).
	// Must be an absolute path to a trusted binary.
	Binary string
	// ExtraArgs are appended to the auto invocation (validated for null bytes).
	ExtraArgs []string
	// SkipWorktree runs in the current checkout (not recommended for parallel).
	SkipWorktree bool
}

// Store persists session records as JSON files.
type Store struct {
	dir string
}

// NewStore creates a session store under repoRoot/.specular/sessions.
func NewStore(repoRoot string) (*Store, error) {
	abs, absErr := filepath.Abs(repoRoot)
	if absErr != nil {
		return nil, fmt.Errorf("session: resolve root: %w", absErr)
	}
	dir := filepath.Join(abs, DefaultRelativeDir)
	if mkdirErr := os.MkdirAll(dir, 0o750); mkdirErr != nil {
		return nil, fmt.Errorf("session: create store: %w", mkdirErr)
	}
	return &Store{dir: dir}, nil
}

// Dir returns the store directory.
func (s *Store) Dir() string { return s.dir }

// Save writes a session record atomically.
func (s *Store) Save(rec *Record) error {
	if rec == nil || rec.ID == "" {
		return fmt.Errorf("session: empty record")
	}
	rec.UpdatedAt = time.Now().UTC()
	data, marshalErr := json.MarshalIndent(rec, "", "  ")
	if marshalErr != nil {
		return marshalErr
	}
	path := s.path(rec.ID)
	tmp := path + ".tmp"
	if writeErr := os.WriteFile(tmp, data, 0o600); writeErr != nil {
		return fmt.Errorf("session: write: %w", writeErr)
	}
	if renameErr := os.Rename(tmp, path); renameErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("session: rename: %w", renameErr)
	}
	return nil
}

// Load reads a session by ID.
func (s *Store) Load(id string) (*Record, error) {
	data, readErr := os.ReadFile(s.path(id))
	if readErr != nil {
		return nil, fmt.Errorf("session: load %s: %w", id, readErr)
	}
	var rec Record
	if decodeErr := json.Unmarshal(data, &rec); decodeErr != nil {
		return nil, fmt.Errorf("session: decode %s: %w", id, decodeErr)
	}
	return &rec, nil
}

// List returns all session records, newest first.
func (s *Store) List() ([]Record, error) {
	entries, readErr := os.ReadDir(s.dir)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return nil, nil
		}
		return nil, readErr
	}
	var out []Record
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		rec, loadErr := s.Load(id)
		if loadErr != nil {
			continue
		}
		out = append(out, *rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func (s *Store) exitPath(id string) string {
	return filepath.Join(s.dir, id+".exit")
}

func (s *Store) logPath(id string) string {
	return filepath.Join(s.dir, id+".log")
}

// Delete removes the session JSON record plus log and exit sidecars.
func (s *Store) Delete(id string) error {
	if id == "" {
		return fmt.Errorf("session: empty id")
	}
	if err := os.Remove(s.path(id)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("session: delete %s: %w", id, err)
	}
	_ = os.Remove(s.exitPath(id))
	_ = os.Remove(s.logPath(id))
	return nil
}

// Manager orchestrates session lifecycle over worktrees + specular auto.
type Manager struct {
	store     *Store
	worktrees *worktree.Manager
	repoRoot  string
}

// NewManager builds a session manager for a git repository.
func NewManager(repoRoot string) (*Manager, error) {
	store, storeErr := NewStore(repoRoot)
	if storeErr != nil {
		return nil, storeErr
	}
	wt, wtErr := worktree.NewManager(repoRoot)
	if wtErr != nil {
		return nil, wtErr
	}
	return &Manager{store: store, worktrees: wt, repoRoot: wt.RepoRoot()}, nil
}

// Store exposes the underlying store (for tests / CLI).
func (m *Manager) Store() *Store { return m.store }

// Start creates an isolated worktree (unless skipped) and launches the
// selected harness (specular-auto, claude-code, codex, or gemini).
func (m *Manager) Start(ctx context.Context, opts StartOptions) (*Record, error) {
	rec, prepErr := m.prepareRecord(ctx, opts)
	if prepErr != nil {
		return nil, prepErr
	}
	if saveErr := m.store.Save(rec); saveErr != nil {
		return nil, saveErr
	}

	plan, planErr := ResolveLaunch(opts, rec)
	if planErr != nil {
		return m.failRecord(rec, planErr)
	}

	if opts.Detach {
		return m.startDetached(rec, plan)
	}
	return m.startForeground(ctx, rec, plan)
}

func (m *Manager) startForeground(ctx context.Context, rec *Record, plan LaunchPlan) (*Record, error) {
	logFile, logErr := os.OpenFile(rec.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if logErr != nil {
		return nil, fmt.Errorf("session: open log: %w", logErr)
	}

	cmd, cmdErr := safeutil.SafeCommand(ctx, plan.Binary, plan.Args...)
	if cmdErr != nil {
		_ = logFile.Close()
		return m.failRecord(rec, cmdErr)
	}
	cmd.Dir = plan.WorkDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	configureProcessGroup(cmd)

	if startErr := cmd.Start(); startErr != nil {
		_ = logFile.Close()
		return m.failRecord(rec, startErr)
	}

	rec.PID = cmd.Process.Pid
	if saveErr := m.store.Save(rec); saveErr != nil {
		_ = logFile.Close()
		return rec, saveErr
	}
	return m.waitForeground(rec, cmd, logFile)
}

// startDetached launches the harness under a shell wrapper that survives the
// CLI process exiting. The wrapper writes <id>.exit so Wait/Refresh can read
// the real exit code after the parent is gone. Uses context.Background so
// cobra canceling the start command does not SIGKILL the child.
func (m *Manager) startDetached(rec *Record, plan LaunchPlan) (*Record, error) {
	probe, probeErr := safeutil.SafeCommand(context.Background(), plan.Binary, plan.Args...)
	if probeErr != nil {
		return m.failRecord(rec, probeErr)
	}
	bin := probe.Path
	args := probe.Args[1:]

	exitPath := m.store.exitPath(rec.ID)
	_ = os.Remove(exitPath)
	logFile, logErr := os.OpenFile(rec.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if logErr != nil {
		return nil, fmt.Errorf("session: open log: %w", logErr)
	}
	_ = logFile.Close()

	script := `"$SPECULAR_SESSION_BIN" "$@" >>"$SPECULAR_SESSION_LOG" 2>&1; code=$?; echo "$code" > "$SPECULAR_SESSION_EXIT"; exit "$code"`
	cmdArgs := append([]string{"-c", script, "specular-session"}, args...)
	cmd := exec.Command("/bin/sh", cmdArgs...) // #nosec G204 -- binary/args validated via SafeCommand
	cmd.Dir = plan.WorkDir
	cmd.Env = append(os.Environ(),
		"SPECULAR_SESSION_BIN="+bin,
		"SPECULAR_SESSION_LOG="+rec.LogPath,
		"SPECULAR_SESSION_EXIT="+exitPath,
	)
	configureProcessGroup(cmd)

	if startErr := cmd.Start(); startErr != nil {
		return m.failRecord(rec, startErr)
	}

	rec.PID = cmd.Process.Pid
	if saveErr := m.store.Save(rec); saveErr != nil {
		return rec, saveErr
	}
	// Best-effort in-process reaper when the CLI stays alive (e.g. tests).
	go m.reapDetached(*rec, cmd, nil)
	return rec, nil
}

// Fork duplicates a session's worktree onto a new branch/name without starting
// a process. Pass StartOptions.Detach/Goal via a subsequent Start, or use
// CLI `session fork --start`.
func (m *Manager) Fork(ctx context.Context, sourceID, newName string) (*Record, error) {
	src, loadErr := m.Get(sourceID)
	if loadErr != nil {
		return nil, loadErr
	}
	if newName == "" {
		newName = src.ID + "-fork"
	}
	info, wtErr := m.worktrees.Create(ctx, worktree.Options{
		Name: newName,
		Base: src.WorktreeBranch,
	})
	if wtErr != nil {
		// Fall back to HEAD if source branch is gone.
		info, wtErr = m.worktrees.Create(ctx, worktree.Options{Name: newName})
		if wtErr != nil {
			return nil, wtErr
		}
	}
	rec := &Record{
		ID:             newName,
		Goal:           src.Goal,
		Harness:        src.Harness,
		Profile:        src.Profile,
		WorktreeName:   info.Name,
		WorktreePath:   info.Path,
		WorktreeBranch: info.Branch,
		Status:         StatusIdle,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		LogPath:        filepath.Join(m.store.Dir(), newName+".log"),
	}
	if saveErr := m.store.Save(rec); saveErr != nil {
		return nil, saveErr
	}
	return rec, nil
}

func (m *Manager) prepareRecord(ctx context.Context, opts StartOptions) (*Record, error) {
	goal := strings.TrimSpace(opts.Goal)
	if goal == "" {
		return nil, fmt.Errorf("session: goal is required")
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = fmt.Sprintf("sess-%d", time.Now().Unix())
	}
	harness := normalizeHarness(opts.Harness)
	if !isKnownHarness(harness) {
		return nil, fmt.Errorf("session: unknown harness %q (supported: %s)",
			opts.Harness, strings.Join(KnownHarness, ", "))
	}
	profile := opts.Profile
	if profile == "" {
		profile = "ci"
	}

	rec := &Record{
		ID:        name,
		Goal:      goal,
		Harness:   harness,
		Profile:   profile,
		Status:    StatusWorking,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		LogPath:   filepath.Join(m.store.Dir(), name+".log"),
	}

	if opts.SkipWorktree {
		return rec, nil
	}
	info, wtErr := m.ensureWorktree(ctx, name)
	if wtErr != nil {
		return nil, wtErr
	}
	rec.WorktreeName = info.Name
	rec.WorktreePath = info.Path
	rec.WorktreeBranch = info.Branch
	return rec, nil
}

func buildAutoArgs(opts StartOptions, rec *Record) []string {
	args := []string{
		"auto",
		"--profile", rec.Profile,
		"--harness", rec.Harness,
		"--json",
	}
	if opts.NoApproval || opts.Detach {
		args = append(args, "--no-approval")
	}
	if !opts.SkipWorktree {
		args = append(args, "--worktree", rec.ID)
	}
	args = append(args, opts.ExtraArgs...)
	args = append(args, rec.Goal)
	return args
}

func resolveBinary(override string) (string, error) {
	if override != "" {
		if !filepath.IsAbs(override) {
			return "", fmt.Errorf("session: binary override must be absolute")
		}
		return override, nil
	}
	exe, exeErr := os.Executable()
	if exeErr != nil {
		return "", fmt.Errorf("session: resolve executable: %w", exeErr)
	}
	return exe, nil
}

func (m *Manager) failRecord(rec *Record, cause error) (*Record, error) {
	rec.Status = StatusFailed
	rec.Error = cause.Error()
	_ = m.store.Save(rec)
	return rec, fmt.Errorf("session: start auto: %w", cause)
}

func (m *Manager) reapDetached(seed Record, cmd *exec.Cmd, logFile *os.File) {
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
	}
	waitErr := cmd.Wait()
	latest, loadErr := m.store.Load(seed.ID)
	if loadErr != nil || latest.Status == StatusStopped {
		return
	}
	// Prefer the detach wrapper's exit sidecar when present.
	if m.applyExitFile(latest) {
		_ = m.store.Save(latest)
		return
	}
	code := 0
	if waitErr != nil {
		latest.Status = StatusFailed
		latest.Error = waitErr.Error()
		if ee, ok := waitErr.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			code = 1
		}
	} else {
		latest.Status = StatusCompleted
		latest.Error = ""
	}
	latest.ExitCode = &code
	latest.PID = 0
	_ = m.store.Save(latest)
}

func (m *Manager) waitForeground(rec *Record, cmd *exec.Cmd, logFile *os.File) (*Record, error) {
	defer func() { _ = logFile.Close() }()
	waitErr := cmd.Wait()
	code := 0
	if waitErr != nil {
		rec.Status = StatusFailed
		rec.Error = waitErr.Error()
		if ee, ok := waitErr.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			code = 1
		}
	} else {
		rec.Status = StatusCompleted
	}
	rec.ExitCode = &code
	rec.PID = 0
	_ = m.store.Save(rec)
	return rec, waitErr
}

func (m *Manager) ensureWorktree(ctx context.Context, name string) (*worktree.Info, error) {
	list, listErr := m.worktrees.List(ctx)
	if listErr == nil {
		for i := range list {
			if list[i].Managed && (list[i].Name == name || list[i].Branch == worktree.BranchPrefix+name) {
				return &list[i], nil
			}
		}
	}
	return m.worktrees.Create(ctx, worktree.Options{Name: name})
}

// Get loads and refreshes a session's live status.
func (m *Manager) Get(id string) (*Record, error) {
	rec, loadErr := m.store.Load(id)
	if loadErr != nil {
		return nil, loadErr
	}
	_ = m.Refresh(rec)
	return rec, nil
}

// List returns refreshed session records.
func (m *Manager) List() ([]Record, error) {
	list, listErr := m.store.List()
	if listErr != nil {
		return nil, listErr
	}
	for i := range list {
		_ = m.Refresh(&list[i])
	}
	return list, nil
}

// Refresh updates Status based on whether the PID is still alive.
func (m *Manager) Refresh(rec *Record) error {
	if rec == nil {
		return nil
	}
	switch rec.Status {
	case StatusCompleted, StatusFailed, StatusStopped:
		return nil
	}
	if rec.PID > 0 && processAlive(rec.PID) {
		return nil
	}

	// Process exited (or never had a PID). Prefer exit-code sidecar from the
	// detached wrapper, then the in-process reaper's store write.
	if m.applyExitFile(rec) {
		return m.store.Save(rec)
	}

	latest, loadErr := m.store.Load(rec.ID)
	if loadErr == nil {
		if IsTerminal(latest.Status) {
			*rec = *latest
			return nil
		}
		if time.Since(latest.UpdatedAt) < 5*time.Second {
			*rec = *latest
			return nil
		}
		*rec = *latest
	}

	if rec.PID <= 0 {
		return nil
	}
	rec.Status = StatusFailed
	if rec.Error == "" {
		rec.Error = "process exited"
	}
	rec.PID = 0
	return m.store.Save(rec)
}

func (m *Manager) applyExitFile(rec *Record) bool {
	b, err := os.ReadFile(m.store.exitPath(rec.ID))
	if err != nil {
		return false
	}
	code, convErr := strconv.Atoi(strings.TrimSpace(string(b)))
	if convErr != nil {
		return false
	}
	rec.ExitCode = &code
	rec.PID = 0
	if code == 0 {
		rec.Status = StatusCompleted
		rec.Error = ""
	} else {
		rec.Status = StatusFailed
		if rec.Error == "" {
			rec.Error = fmt.Sprintf("exit code %d", code)
		}
	}
	return true
}

// Stop terminates a running session process.
func (m *Manager) Stop(id string) (*Record, error) {
	rec, loadErr := m.store.Load(id)
	if loadErr != nil {
		return nil, loadErr
	}
	if rec.PID > 0 && processAlive(rec.PID) {
		if killErr := killProcess(rec.PID); killErr != nil {
			return rec, fmt.Errorf("session: stop pid %d: %w", rec.PID, killErr)
		}
	}
	rec.Status = StatusStopped
	rec.PID = 0
	if saveErr := m.store.Save(rec); saveErr != nil {
		return rec, saveErr
	}
	return rec, nil
}

// RemoveOptions configures Manager.Remove.
type RemoveOptions struct {
	// Force stops a still-running session before removal.
	Force bool
	// KeepWorktree leaves the Git worktree and branch in place.
	KeepWorktree bool
	// DeleteBranch deletes the managed worktree branch (ignored when KeepWorktree).
	DeleteBranch bool
}

// Remove stops (optional) and deletes a session record, logs, exit sidecar,
// and by default the associated worktree — closing the start→wait→cleanup loop.
func (m *Manager) Remove(ctx context.Context, id string, opts RemoveOptions) (*Record, error) {
	rec, getErr := m.Get(id)
	if getErr != nil {
		return nil, getErr
	}
	running := !IsTerminal(rec.Status) || (rec.PID > 0 && processAlive(rec.PID))
	if running {
		if !opts.Force {
			return nil, fmt.Errorf("session: %s is still %s; pass --force to stop and remove", id, rec.Status)
		}
		if _, stopErr := m.Stop(id); stopErr != nil {
			return nil, stopErr
		}
		// Reload after stop so the returned snapshot is accurate.
		if latest, loadErr := m.store.Load(id); loadErr == nil {
			rec = latest
		}
	}

	if !opts.KeepWorktree && rec.WorktreePath != "" {
		if rmErr := m.worktrees.Remove(ctx, rec.WorktreePath, opts.DeleteBranch); rmErr != nil {
			return rec, fmt.Errorf("session: remove worktree: %w", rmErr)
		}
	}

	snapshot := *rec
	if delErr := m.store.Delete(id); delErr != nil {
		return &snapshot, delErr
	}
	return &snapshot, nil
}

// PruneOptions configures Manager.Prune.
type PruneOptions struct {
	// OlderThan keeps terminal sessions newer than this age (0 = prune all terminal).
	OlderThan time.Duration
	// KeepWorktree leaves Git worktrees in place.
	KeepWorktree bool
	// DeleteBranch deletes managed worktree branches when removing worktrees.
	DeleteBranch bool
}

// Prune removes terminal sessions (completed/failed/stopped/idle), optionally
// filtered by age. Returns the removed records.
func (m *Manager) Prune(ctx context.Context, opts PruneOptions) ([]Record, error) {
	list, listErr := m.List()
	if listErr != nil {
		return nil, listErr
	}
	var cutoff time.Time
	if opts.OlderThan > 0 {
		cutoff = time.Now().UTC().Add(-opts.OlderThan)
	}
	var removed []Record
	for _, rec := range list {
		if !IsTerminal(rec.Status) {
			continue
		}
		if !cutoff.IsZero() && rec.UpdatedAt.After(cutoff) {
			continue
		}
		got, rmErr := m.Remove(ctx, rec.ID, RemoveOptions{
			KeepWorktree: opts.KeepWorktree,
			DeleteBranch: opts.DeleteBranch,
		})
		if rmErr != nil {
			return removed, rmErr
		}
		if got != nil {
			removed = append(removed, *got)
		}
	}
	return removed, nil
}

// WaitOptions configures Manager.Wait.
type WaitOptions struct {
	// Timeout bounds how long to wait. Zero means no deadline (context only).
	Timeout time.Duration
	// Interval is the poll period (default 500ms).
	Interval time.Duration
	// Any returns as soon as one target reaches a terminal status.
	Any bool
}

// IsTerminal reports whether status is a finished session state.
func IsTerminal(status string) bool {
	switch status {
	case StatusCompleted, StatusFailed, StatusStopped, StatusIdle:
		return true
	default:
		return false
	}
}

// Wait blocks until the named sessions finish (or all active sessions when
// ids is empty). Returns the final records. An error is returned on timeout
// or when any waited session ends in failed/stopped (completed and idle are OK).
func (m *Manager) Wait(ctx context.Context, ids []string, opts WaitOptions) ([]Record, error) {
	if opts.Interval <= 0 {
		opts.Interval = 500 * time.Millisecond
	}
	targets, resolveErr := m.resolveWaitTargets(ids)
	if resolveErr != nil {
		return nil, resolveErr
	}
	if len(targets) == 0 {
		return nil, nil
	}

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	for {
		done, pending, pollErr := m.pollWait(targets, opts.Any)
		if pollErr != nil {
			return done, pollErr
		}
		if opts.Any && len(done) > 0 {
			return done, waitOutcomeError(done)
		}
		if len(pending) == 0 {
			return done, waitOutcomeError(done)
		}
		select {
		case <-ctx.Done():
			return done, fmt.Errorf("session: wait timed out; still running: %s", strings.Join(pending, ", "))
		case <-ticker.C:
		}
	}
}

func (m *Manager) resolveWaitTargets(ids []string) ([]string, error) {
	if len(ids) > 0 {
		for _, id := range ids {
			if _, err := m.store.Load(id); err != nil {
				return nil, err
			}
		}
		return append([]string(nil), ids...), nil
	}
	list, listErr := m.List()
	if listErr != nil {
		return nil, listErr
	}
	var active []string
	for _, rec := range list {
		if !IsTerminal(rec.Status) {
			active = append(active, rec.ID)
		}
	}
	return active, nil
}

func (m *Manager) pollWait(ids []string, any bool) (done []Record, pending []string, err error) {
	for _, id := range ids {
		rec, getErr := m.Get(id)
		if getErr != nil {
			return done, pending, getErr
		}
		if IsTerminal(rec.Status) {
			done = append(done, *rec)
			if any {
				return done, nil, nil
			}
			continue
		}
		pending = append(pending, id)
	}
	return done, pending, nil
}

func waitOutcomeError(recs []Record) error {
	var failed []string
	for _, rec := range recs {
		switch rec.Status {
		case StatusFailed, StatusStopped:
			failed = append(failed, fmt.Sprintf("%s(%s)", rec.ID, rec.Status))
		}
	}
	if len(failed) == 0 {
		return nil
	}
	return fmt.Errorf("session: wait finished with failures: %s", strings.Join(failed, ", "))
}

// RestartOptions configures Manager.Restart (same worktree, optional new harness).
type RestartOptions struct {
	Harness    string
	Goal       string
	Profile    string
	Force      bool
	Detach     bool
	NoApproval bool
	Binary     string
	ExtraArgs  []string
}

// Restart stops (optional) and re-launches a session in its existing worktree,
// optionally switching harness or goal — the CLI analogue of Xirp agent swap.
func (m *Manager) Restart(ctx context.Context, id string, opts RestartOptions) (*Record, error) {
	rec, loadErr := m.Get(id)
	if loadErr != nil {
		return nil, loadErr
	}
	if rec.PID > 0 && processAlive(rec.PID) {
		if !opts.Force {
			return nil, fmt.Errorf("session: %s still running (pid %d); pass --force to stop it", id, rec.PID)
		}
		if _, stopErr := m.Stop(id); stopErr != nil {
			return nil, stopErr
		}
	}

	goal := strings.TrimSpace(opts.Goal)
	if goal == "" {
		goal = rec.Goal
	}
	harness := opts.Harness
	if harness == "" {
		harness = rec.Harness
	}
	profile := opts.Profile
	if profile == "" {
		profile = rec.Profile
	}

	return m.Start(ctx, StartOptions{
		Goal:         goal,
		Name:         id,
		Harness:      harness,
		Profile:      profile,
		Detach:       opts.Detach,
		NoApproval:   opts.NoApproval || opts.Detach,
		Binary:       opts.Binary,
		ExtraArgs:    opts.ExtraArgs,
		SkipWorktree: rec.WorktreePath == "",
	})
}

// DiffOptions configures Manager.Diff.
type DiffOptions struct {
	// Base is the git ref to compare against (default: main/master/HEAD).
	Base string
	// Against is another session ID; when set, compares the two session HEADs.
	Against string
	// Stat requests a --stat summary (default when neither NameOnly nor Patch).
	Stat bool
	// NameOnly lists changed paths only.
	NameOnly bool
	// Patch requests a full unified diff.
	Patch bool
}

// DiffResult is the git diff for a session worktree (or between two sessions).
type DiffResult struct {
	SessionID    string `json:"sessionId"`
	AgainstID    string `json:"againstId,omitempty"`
	Base         string `json:"base,omitempty"`
	WorktreePath string `json:"worktreePath,omitempty"`
	Branch       string `json:"branch,omitempty"`
	Output       string `json:"output"`
}

// Diff shows changes in a session worktree versus a base ref, or versus
// another session's HEAD — the scriptable analogue of Xirp's per-session
// Git changes panel.
func (m *Manager) Diff(ctx context.Context, id string, opts DiffOptions) (*DiffResult, error) {
	rec, getErr := m.Get(id)
	if getErr != nil {
		return nil, getErr
	}
	if rec.WorktreePath == "" {
		return nil, fmt.Errorf("session: %s has no worktree (started with --no-worktree?)", id)
	}

	stat := opts.Stat
	nameOnly := opts.NameOnly
	if !opts.Patch && !opts.Stat && !opts.NameOnly {
		stat = true
	}
	if opts.Patch {
		stat = false
		nameOnly = false
	}

	result := &DiffResult{
		SessionID:    rec.ID,
		WorktreePath: rec.WorktreePath,
		Branch:       rec.WorktreeBranch,
	}

	if opts.Against != "" {
		other, otherErr := m.Get(opts.Against)
		if otherErr != nil {
			return nil, otherErr
		}
		if other.WorktreePath == "" {
			return nil, fmt.Errorf("session: %s has no worktree", opts.Against)
		}
		left, leftErr := m.worktrees.HeadSHA(ctx, rec.WorktreePath)
		if leftErr != nil {
			return nil, fmt.Errorf("session: resolve %s HEAD: %w", id, leftErr)
		}
		right, rightErr := m.worktrees.HeadSHA(ctx, other.WorktreePath)
		if rightErr != nil {
			return nil, fmt.Errorf("session: resolve %s HEAD: %w", opts.Against, rightErr)
		}
		out, diffErr := m.worktrees.Diff(ctx, worktree.DiffOptions{
			Base:        left,
			AgainstHead: right,
			Stat:        stat,
			NameOnly:    nameOnly,
		})
		if diffErr != nil {
			return nil, fmt.Errorf("session: diff %s..%s: %w", id, opts.Against, diffErr)
		}
		result.AgainstID = other.ID
		result.Base = left
		result.Output = out
		return result, nil
	}

	base := opts.Base
	if base == "" {
		var baseErr error
		base, baseErr = m.worktrees.DefaultBase(ctx)
		if baseErr != nil {
			return nil, baseErr
		}
	}
	out, diffErr := m.worktrees.Diff(ctx, worktree.DiffOptions{
		WorkDir:  rec.WorktreePath,
		Base:     base,
		Stat:     stat,
		NameOnly: nameOnly,
	})
	if diffErr != nil {
		return nil, fmt.Errorf("session: diff %s: %w", id, diffErr)
	}
	if untracked, uErr := m.worktrees.UntrackedFiles(ctx, rec.WorktreePath); uErr == nil && len(untracked) > 0 {
		out = appendUntracked(out, untracked, nameOnly, stat)
	}
	result.Base = base
	result.Output = out
	return result, nil
}

func appendUntracked(out string, files []string, nameOnly, stat bool) string {
	if nameOnly {
		var b strings.Builder
		b.WriteString(out)
		if out != "" && !strings.HasSuffix(out, "\n") {
			b.WriteByte('\n')
		}
		for _, f := range files {
			b.WriteString(f)
			b.WriteByte('\n')
		}
		return b.String()
	}
	var b strings.Builder
	b.WriteString(out)
	if out != "" && !strings.HasSuffix(out, "\n") {
		b.WriteByte('\n')
	}
	if stat || strings.TrimSpace(out) == "" {
		b.WriteString(" Untracked files:\n")
		for _, f := range files {
			b.WriteString("  ")
			b.WriteString(f)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// ParsePID is a small helper for tests.
func ParsePID(s string) (int, error) {
	return strconv.Atoi(s)
}
