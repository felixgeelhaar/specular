// Package worktree manages Git worktree isolation for parallel agent sessions.
//
// Inspired by the parallel-session pattern popularized by agentic development
// environments (e.g. Spotify Xirp): each concurrent Specular auto/build session
// can run in its own checkout and branch so agents do not collide on the same
// working tree. Specular additionally records worktree provenance in
// attestations so the drift gate can attribute authorship to an isolated session.
package worktree

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/safeutil"
)

const (
	// DefaultRelativeDir is the default location for Specular-managed worktrees
	// relative to the repository root.
	DefaultRelativeDir = ".specular/worktrees"

	// BranchPrefix is applied to auto-generated branch names.
	BranchPrefix = "specular/"
)

var (
	// safeNamePattern allows alphanumeric, dash, underscore, and slash segments.
	safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)
)

// Info describes a managed worktree.
type Info struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Branch    string `json:"branch"`
	Head      string `json:"head,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	Bare      bool   `json:"bare,omitempty"`
	Detached  bool   `json:"detached,omitempty"`
	Managed   bool   `json:"managed"` // true if under .specular/worktrees
	RepoRoot  string `json:"repoRoot,omitempty"`
}

// Options controls worktree creation.
type Options struct {
	// Name is a human-readable session slug. Used for the directory and branch.
	// If empty, a timestamp-based name is generated.
	Name string

	// Branch overrides the branch name. Defaults to BranchPrefix + Name.
	Branch string

	// Base is the starting ref (branch, tag, or commit). Defaults to HEAD.
	Base string

	// ParentDir is where worktrees are created. Defaults to
	// <repoRoot>/.specular/worktrees.
	ParentDir string

	// Force overwrites an existing worktree directory if it is empty/orphaned.
	Force bool
}

// Manager creates and inspects Git worktrees for a repository.
type Manager struct {
	repoRoot string
}

// NewManager returns a Manager rooted at repoRoot. repoRoot must be inside a
// Git working tree (or the repository root itself).
func NewManager(repoRoot string) (*Manager, error) {
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("worktree: resolve path: %w", err)
	}
	root, err := findRepoRoot(abs)
	if err != nil {
		return nil, err
	}
	return &Manager{repoRoot: root}, nil
}

// RepoRoot returns the detected Git repository root.
func (m *Manager) RepoRoot() string {
	return m.repoRoot
}

// Create adds a new worktree with an isolated branch and returns its Info.
func (m *Manager) Create(ctx context.Context, opts Options) (*Info, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = fmt.Sprintf("session-%d", time.Now().Unix())
	}
	if err := validateName(name); err != nil {
		return nil, err
	}

	branch := strings.TrimSpace(opts.Branch)
	if branch == "" {
		branch = BranchPrefix + name
	}
	if err := validateName(branch); err != nil {
		return nil, fmt.Errorf("worktree: invalid branch: %w", err)
	}

	parent := opts.ParentDir
	if parent == "" {
		parent = filepath.Join(m.repoRoot, DefaultRelativeDir)
	}
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return nil, fmt.Errorf("worktree: create parent dir: %w", err)
	}

	path := filepath.Join(parent, sanitizeDirName(name))
	if _, statErr := os.Stat(path); statErr == nil {
		if !opts.Force {
			return nil, fmt.Errorf("worktree: path already exists: %s", path)
		}
		if rmErr := os.RemoveAll(path); rmErr != nil {
			return nil, fmt.Errorf("worktree: remove existing path: %w", rmErr)
		}
	}

	base := opts.Base
	if base == "" {
		base = "HEAD"
	}

	// Prefer creating a new branch from base: git worktree add -b <branch> <path> <base>
	args := []string{"worktree", "add", "-b", branch, path, base}
	if err := runGit(ctx, m.repoRoot, args...); err != nil {
		// If the branch already exists, attach to it instead of failing.
		if strings.Contains(err.Error(), "already exists") {
			args = []string{"worktree", "add", path, branch}
			if err2 := runGit(ctx, m.repoRoot, args...); err2 != nil {
				return nil, fmt.Errorf("worktree: add existing branch: %w", err2)
			}
		} else {
			return nil, fmt.Errorf("worktree: add: %w", err)
		}
	}

	info := &Info{
		Name:      name,
		Path:      path,
		Branch:    branch,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Managed:   true,
		RepoRoot:  m.repoRoot,
	}
	if head, err := revParse(ctx, path, "HEAD"); err == nil {
		info.Head = head
	}
	return info, nil
}

// List returns all worktrees known to Git for this repository.
func (m *Manager) List(ctx context.Context) ([]Info, error) {
	out, err := runGitOutput(ctx, m.repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("worktree: list: %w", err)
	}
	return parsePorcelain(out, m.repoRoot), nil
}

// Remove deletes a worktree by path. If deleteBranch is true and the branch
// matches BranchPrefix, the branch is also deleted.
func (m *Manager) Remove(ctx context.Context, path string, deleteBranch bool) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("worktree: resolve path: %w", err)
	}

	var branch string
	if deleteBranch {
		branch, _ = revParse(ctx, abs, "--abbrev-ref", "HEAD")
	}

	if removeErr := runGit(ctx, m.repoRoot, "worktree", "remove", "--force", abs); removeErr != nil {
		// Fall back to prune + directory removal for broken worktrees.
		_ = runGit(ctx, m.repoRoot, "worktree", "prune")
		if rmErr := os.RemoveAll(abs); rmErr != nil {
			return fmt.Errorf("worktree: remove: %w (also: %v)", removeErr, rmErr)
		}
	}

	if deleteBranch && branch != "" && strings.HasPrefix(branch, BranchPrefix) && branch != "HEAD" {
		_ = runGit(ctx, m.repoRoot, "branch", "-D", branch)
	}
	return nil
}

// findRepoRoot walks up from start until it finds a .git directory/file.
func findRepoRoot(start string) (string, error) {
	dir := start
	for {
		gitPath := filepath.Join(dir, ".git")
		if st, err := os.Stat(gitPath); err == nil && (st.IsDir() || st.Mode().IsRegular()) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("worktree: not a git repository (%s)", start)
		}
		dir = parent
	}
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("worktree: empty name")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("worktree: name must not contain '..'")
	}
	if !safeNamePattern.MatchString(name) {
		return fmt.Errorf("worktree: name %q contains invalid characters", name)
	}
	return nil
}

func sanitizeDirName(name string) string {
	return strings.ReplaceAll(name, "/", "-")
}

func runGit(ctx context.Context, dir string, args ...string) error {
	cmd, err := safeutil.SafeCommand(ctx, "git", args...)
	if err != nil {
		return err
	}
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func runGitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd, err := safeutil.SafeCommand(ctx, "git", args...)
	if err != nil {
		return "", err
	}
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func revParse(ctx context.Context, dir string, args ...string) (string, error) {
	full := append([]string{"rev-parse"}, args...)
	out, err := runGitOutput(ctx, dir, full...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func parsePorcelain(out, repoRoot string) []Info {
	managedPrefix := filepath.Join(repoRoot, DefaultRelativeDir)
	var results []Info
	var cur *Info

	flush := func() {
		if cur != nil && cur.Path != "" {
			cur.Managed = strings.HasPrefix(cur.Path, managedPrefix)
			cur.RepoRoot = repoRoot
			if cur.Name == "" {
				cur.Name = filepath.Base(cur.Path)
			}
			results = append(results, *cur)
		}
		cur = nil
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}
		if cur == nil {
			cur = &Info{}
		}
		switch {
		case strings.HasPrefix(line, "worktree "):
			cur.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			cur.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			cur.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			cur.Bare = true
		case line == "detached":
			cur.Detached = true
		}
	}
	flush()
	return results
}
