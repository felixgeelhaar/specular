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

// Start creates an isolated worktree (unless skipped) and launches specular auto.
func (m *Manager) Start(ctx context.Context, opts StartOptions) (*Record, error) {
	rec, prepErr := m.prepareRecord(ctx, opts)
	if prepErr != nil {
		return nil, prepErr
	}
	if saveErr := m.store.Save(rec); saveErr != nil {
		return nil, saveErr
	}

	binary, binErr := resolveBinary(opts.Binary)
	if binErr != nil {
		return nil, binErr
	}
	args := buildAutoArgs(opts, rec)

	logFile, logErr := os.OpenFile(rec.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if logErr != nil {
		return nil, fmt.Errorf("session: open log: %w", logErr)
	}

	cmd, cmdErr := safeutil.SafeCommand(ctx, binary, args...)
	if cmdErr != nil {
		_ = logFile.Close()
		return m.failRecord(rec, cmdErr)
	}
	cmd.Dir = m.repoRoot
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

	if opts.Detach {
		go m.reapDetached(*rec, cmd, logFile)
		return rec, nil
	}
	return m.waitForeground(rec, cmd, logFile)
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
	harness := opts.Harness
	if harness == "" {
		harness = "specular-auto"
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
	defer func() { _ = logFile.Close() }()
	waitErr := cmd.Wait()
	latest, loadErr := m.store.Load(seed.ID)
	if loadErr != nil || latest.Status == StatusStopped {
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
	if rec.PID <= 0 || processAlive(rec.PID) {
		return nil
	}
	rec.Status = StatusFailed
	if rec.Error == "" {
		rec.Error = "process exited"
	}
	rec.PID = 0
	return m.store.Save(rec)
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

// ParsePID is a small helper for tests.
func ParsePID(s string) (int, error) {
	return strconv.Atoi(s)
}
