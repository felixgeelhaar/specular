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
	"strconv"
	"strings"
	"time"

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
	// Binary overrides the specular executable path (tests).
	Binary string
	// ExtraArgs are appended to the auto invocation.
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
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("session: resolve root: %w", err)
	}
	dir := filepath.Join(abs, DefaultRelativeDir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("session: create store: %w", err)
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
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	path := s.path(rec.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("session: write: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("session: rename: %w", err)
	}
	return nil
}

// Load reads a session by ID.
func (s *Store) Load(id string) (*Record, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil, fmt.Errorf("session: load %s: %w", id, err)
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("session: decode %s: %w", id, err)
	}
	return &rec, nil
}

// List returns all session records, newest first.
func (s *Store) List() ([]Record, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Record
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		rec, err := s.Load(id)
		if err != nil {
			continue
		}
		out = append(out, *rec)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.After(out[i].CreatedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
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
	store, err := NewStore(repoRoot)
	if err != nil {
		return nil, err
	}
	wt, err := worktree.NewManager(repoRoot)
	if err != nil {
		return nil, err
	}
	return &Manager{store: store, worktrees: wt, repoRoot: wt.RepoRoot()}, nil
}

// Store exposes the underlying store (for tests / CLI).
func (m *Manager) Store() *Store { return m.store }

// Start creates an isolated worktree (unless skipped) and launches specular auto.
func (m *Manager) Start(ctx context.Context, opts StartOptions) (*Record, error) {
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
	}

	if !opts.SkipWorktree {
		info, err := m.ensureWorktree(ctx, name)
		if err != nil {
			return nil, err
		}
		rec.WorktreeName = info.Name
		rec.WorktreePath = info.Path
		rec.WorktreeBranch = info.Branch
	}

	logPath := filepath.Join(m.store.Dir(), name+".log")
	rec.LogPath = logPath

	if err := m.store.Save(rec); err != nil {
		return nil, err
	}

	binary := opts.Binary
	if binary == "" {
		var err error
		binary, err = os.Executable()
		if err != nil {
			binary = os.Args[0]
		}
	}

	args := []string{
		"auto",
		"--profile", profile,
		"--harness", harness,
		"--json",
	}
	if opts.NoApproval || opts.Detach {
		args = append(args, "--no-approval")
	}
	if !opts.SkipWorktree {
		args = append(args, "--worktree", name)
	}
	args = append(args, opts.ExtraArgs...)
	args = append(args, goal)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("session: open log: %w", err)
	}

	cmd := exec.CommandContext(ctx, binary, args...) // #nosec G204 -- binary is self or test override
	cmd.Dir = m.repoRoot
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	configureProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		rec.Status = StatusFailed
		rec.Error = err.Error()
		_ = m.store.Save(rec)
		return rec, fmt.Errorf("session: start auto: %w", err)
	}

	rec.PID = cmd.Process.Pid
	if err := m.store.Save(rec); err != nil {
		_ = logFile.Close()
		return rec, err
	}

	if opts.Detach {
		go func(r Record, c *exec.Cmd, f *os.File) {
			defer f.Close()
			waitErr := c.Wait()
			latest, loadErr := m.store.Load(r.ID)
			if loadErr != nil {
				return
			}
			if latest.Status == StatusStopped {
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
		}(*rec, cmd, logFile)
		return rec, nil
	}

	defer logFile.Close()
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
	list, err := m.worktrees.List(ctx)
	if err == nil {
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
	rec, err := m.store.Load(id)
	if err != nil {
		return nil, err
	}
	_ = m.Refresh(rec)
	return rec, nil
}

// List returns refreshed session records.
func (m *Manager) List() ([]Record, error) {
	list, err := m.store.List()
	if err != nil {
		return nil, err
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
	if rec.Status == StatusCompleted || rec.Status == StatusFailed || rec.Status == StatusStopped {
		return nil
	}
	if rec.PID <= 0 {
		return nil
	}
	if !processAlive(rec.PID) {
		if rec.Status == StatusWorking || rec.Status == StatusIdle || rec.Status == StatusWaiting {
			rec.Status = StatusFailed
			if rec.Error == "" {
				rec.Error = "process exited"
			}
			rec.PID = 0
			return m.store.Save(rec)
		}
	}
	return nil
}

// Stop terminates a running session process.
func (m *Manager) Stop(id string) (*Record, error) {
	rec, err := m.store.Load(id)
	if err != nil {
		return nil, err
	}
	if rec.PID > 0 && processAlive(rec.PID) {
		if err := killProcess(rec.PID); err != nil {
			return rec, fmt.Errorf("session: stop pid %d: %w", rec.PID, err)
		}
	}
	rec.Status = StatusStopped
	rec.PID = 0
	if err := m.store.Save(rec); err != nil {
		return rec, err
	}
	return rec, nil
}

// ParsePID is a small helper for tests.
func ParsePID(s string) (int, error) {
	return strconv.Atoi(s)
}
