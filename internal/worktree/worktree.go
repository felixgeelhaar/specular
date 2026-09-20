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

// DiffOptions configures Manager.Diff.
type DiffOptions struct {
	// WorkDir is the directory to run git in (typically a session worktree).
	WorkDir string
	// Base is the ref/tree to compare against. Empty uses DefaultBase.
	Base string
	// AgainstHead, when set, compares BaseHead..AgainstHead from the repo root
	// instead of a worktree working-tree diff.
	AgainstHead string
	// Stat requests --stat output.
	Stat bool
	// NameOnly requests --name-only output.
	NameOnly bool
}

// DefaultBase picks a sensible comparison ref: main, master, or HEAD.
func (m *Manager) DefaultBase(ctx context.Context) (string, error) {
	for _, candidate := range []string{"main", "master", "HEAD"} {
		if _, err := revParse(ctx, m.repoRoot, "--verify", candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("worktree: no default base ref found")
}

// HeadSHA resolves HEAD in dir to a full commit SHA.
func (m *Manager) HeadSHA(ctx context.Context, dir string) (string, error) {
	if dir == "" {
		dir = m.repoRoot
	}
	return revParse(ctx, dir, "HEAD")
}

// Diff returns git diff output for a worktree (or between two commits).
func (m *Manager) Diff(ctx context.Context, opts DiffOptions) (string, error) {
	args := []string{"diff", "--no-ext-diff"}
	if opts.NameOnly {
		args = append(args, "--name-only")
	} else if opts.Stat {
		args = append(args, "--stat")
	}

	if opts.AgainstHead != "" {
		left := opts.Base
		if left == "" {
			return "", fmt.Errorf("worktree: base head required for against-diff")
		}
		args = append(args, left, opts.AgainstHead)
		return runGitOutput(ctx, m.repoRoot, args...)
	}

	workDir := opts.WorkDir
	if workDir == "" {
		workDir = m.repoRoot
	}
	base := opts.Base
	if base == "" {
		var baseErr error
		base, baseErr = m.DefaultBase(ctx)
		if baseErr != nil {
			return "", baseErr
		}
	}
	args = append(args, base)
	return runGitOutput(ctx, workDir, args...)
}

// UntrackedFiles lists untracked (and not ignored) paths under dir.
func (m *Manager) UntrackedFiles(ctx context.Context, dir string) ([]string, error) {
	if dir == "" {
		dir = m.repoRoot
	}
	out, err := runGitOutput(ctx, dir, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// CommitOptions configures Manager.Commit in a worktree.
type CommitOptions struct {
	// WorkDir is the git working directory (session worktree).
	WorkDir string
	// Message is the commit message (required).
	Message string
	// All stages untracked files as well (git add -A). Without All, only
	// tracked modifications are staged (git add -u).
	All bool
	// AllowEmpty creates a commit even when the index has no changes.
	AllowEmpty bool
}

// CommitResult is the outcome of a worktree commit.
type CommitResult struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
}

// Dirty reports whether dir has staged, unstaged, or (optionally) untracked changes.
func (m *Manager) Dirty(ctx context.Context, dir string, includeUntracked bool) (bool, error) {
	if dir == "" {
		dir = m.repoRoot
	}
	return isDirty(ctx, dir, includeUntracked)
}

// Commit stages and commits changes in WorkDir. Returns the new HEAD SHA.
func (m *Manager) Commit(ctx context.Context, opts CommitOptions) (*CommitResult, error) {
	dir := opts.WorkDir
	if dir == "" {
		dir = m.repoRoot
	}
	msg := strings.TrimSpace(opts.Message)
	if msg == "" {
		return nil, fmt.Errorf("worktree: commit message is required")
	}

	dirty, dirtyErr := m.Dirty(ctx, dir, opts.All)
	if dirtyErr != nil {
		return nil, dirtyErr
	}
	if !dirty && !opts.AllowEmpty {
		return nil, fmt.Errorf("worktree: nothing to commit")
	}

	if dirty {
		addArgs := []string{"add", "-u"}
		if opts.All {
			addArgs = []string{"add", "-A"}
		}
		if addErr := runGit(ctx, dir, addArgs...); addErr != nil {
			return nil, fmt.Errorf("worktree: git add: %w", addErr)
		}
	}

	commitArgs := []string{"commit", "-m", msg}
	if opts.AllowEmpty {
		commitArgs = append(commitArgs, "--allow-empty")
	}
	if commitErr := runGit(ctx, dir, commitArgs...); commitErr != nil {
		return nil, fmt.Errorf("worktree: git commit: %w", commitErr)
	}
	sha, shaErr := m.HeadSHA(ctx, dir)
	if shaErr != nil {
		return nil, shaErr
	}
	return &CommitResult{SHA: sha, Message: msg}, nil
}

// PushOptions configures Manager.Push.
type PushOptions struct {
	WorkDir string
	// Remote defaults to "origin".
	Remote string
	// Branch is the local branch to push. Empty uses the current branch.
	Branch string
	// SetUpstream passes --set-upstream.
	SetUpstream bool
}

// PushResult is the outcome of pushing a worktree branch.
type PushResult struct {
	Remote string
	Branch string
	SHA    string
}

// Push publishes the worktree branch to the remote.
func (m *Manager) Push(ctx context.Context, opts PushOptions) (*PushResult, error) {
	dir := opts.WorkDir
	if dir == "" {
		dir = m.repoRoot
	}
	remote := strings.TrimSpace(opts.Remote)
	if remote == "" {
		remote = "origin"
	}
	branch := strings.TrimSpace(opts.Branch)
	if branch == "" {
		cur, err := revParse(ctx, dir, "--abbrev-ref", "HEAD")
		if err != nil {
			return nil, fmt.Errorf("worktree: resolve branch: %w", err)
		}
		if cur == "" || cur == "HEAD" {
			return nil, fmt.Errorf("worktree: detached HEAD; cannot push without --branch")
		}
		branch = cur
	}
	args := []string{"push"}
	if opts.SetUpstream {
		args = append(args, "-u")
	}
	args = append(args, remote, branch)
	if err := runGit(ctx, dir, args...); err != nil {
		return nil, fmt.Errorf("worktree: git push: %w", err)
	}
	sha, shaErr := m.HeadSHA(ctx, dir)
	if shaErr != nil {
		return nil, shaErr
	}
	return &PushResult{Remote: remote, Branch: branch, SHA: sha}, nil
}

// MergeOptions configures Manager.Merge — land a source branch into a target
// branch at the repository primary checkout (not a session worktree).
type MergeOptions struct {
	// SourceBranch is the branch to merge in (typically a session worktree branch).
	SourceBranch string
	// Into is the target branch. Empty uses DefaultBase (main/master/HEAD).
	Into string
	// Message overrides the merge commit message (ignored for pure fast-forwards).
	Message string
	// FFOnly requires a fast-forward merge.
	FFOnly bool
	// NoFF always creates a merge commit.
	NoFF bool
}

// MergeResult is the outcome of landing a branch into the primary checkout.
type MergeResult struct {
	Into      string   `json:"into"`
	Source    string   `json:"source"`
	Strategy  string   `json:"strategy"`
	BeforeSHA string   `json:"beforeSha"`
	AfterSHA  string   `json:"afterSha"`
	Conflicts []string `json:"conflicts,omitempty"`
}

// Merge lands SourceBranch into Into at the repository root.
// Requires a clean primary working tree. On conflict, aborts and returns Conflicts.
func (m *Manager) Merge(ctx context.Context, opts MergeOptions) (*MergeResult, error) {
	source := strings.TrimSpace(opts.SourceBranch)
	if source == "" {
		return nil, fmt.Errorf("worktree: merge source branch is required")
	}
	if opts.FFOnly && opts.NoFF {
		return nil, fmt.Errorf("worktree: --ff-only and --no-ff are mutually exclusive")
	}
	into, intoErr := m.resolveMergeInto(ctx, opts.Into)
	if intoErr != nil {
		return nil, intoErr
	}
	if checkoutErr := m.ensureOnBranch(ctx, into); checkoutErr != nil {
		return nil, checkoutErr
	}
	before, beforeErr := m.HeadSHA(ctx, m.repoRoot)
	if beforeErr != nil {
		return nil, beforeErr
	}
	strategy := mergeStrategy(opts.FFOnly, opts.NoFF)
	res := &MergeResult{Into: into, Source: source, Strategy: strategy, BeforeSHA: before}
	if mergeErr := runMergeOp(ctx, m.repoRoot, source, opts); mergeErr != nil {
		res.Conflicts = conflictedPaths(ctx, m.repoRoot)
		_ = runGit(ctx, m.repoRoot, "merge", "--abort")
		res.AfterSHA, _ = m.HeadSHA(ctx, m.repoRoot)
		return res, fmt.Errorf("worktree: merge %s into %s failed: %w", source, into, mergeErr)
	}
	after, afterErr := m.HeadSHA(ctx, m.repoRoot)
	if afterErr != nil {
		return res, afterErr
	}
	res.AfterSHA = after
	return res, nil
}

func (m *Manager) resolveMergeInto(ctx context.Context, into string) (string, error) {
	into = strings.TrimSpace(into)
	if into != "" {
		if _, err := revParse(ctx, m.repoRoot, "--verify", into); err != nil {
			return "", fmt.Errorf("worktree: unknown into ref %q: %w", into, err)
		}
		return into, nil
	}
	return m.DefaultBase(ctx)
}

func (m *Manager) ensureOnBranch(ctx context.Context, branch string) error {
	cur, err := revParse(ctx, m.repoRoot, "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("worktree: resolve HEAD: %w", err)
	}
	dirty, dirtyErr := isDirty(ctx, m.repoRoot, false) // tracked only; ignore .specular/ etc.
	if dirtyErr != nil {
		return dirtyErr
	}
	if dirty {
		if cur == branch {
			return fmt.Errorf("worktree: primary checkout is dirty; commit or stash before merge")
		}
		return fmt.Errorf("worktree: primary checkout is dirty; commit or stash before checking out %s", branch)
	}
	if cur == branch {
		return nil
	}
	if err := runGit(ctx, m.repoRoot, "checkout", branch); err != nil {
		return fmt.Errorf("worktree: checkout %s: %w", branch, err)
	}
	return nil
}

func mergeStrategy(ffOnly, noFF bool) string {
	switch {
	case ffOnly:
		return "ff-only"
	case noFF:
		return "no-ff"
	default:
		return "merge"
	}
}

func runMergeOp(ctx context.Context, dir, source string, opts MergeOptions) error {
	args := []string{"merge"}
	if opts.FFOnly {
		args = append(args, "--ff-only")
	}
	if opts.NoFF {
		args = append(args, "--no-ff")
	}
	if msg := strings.TrimSpace(opts.Message); msg != "" {
		args = append(args, "-m", msg)
	} else {
		args = append(args, "--no-edit")
	}
	args = append(args, source)
	return runGit(ctx, dir, args...)
}

// SyncOptions configures Manager.Sync in a worktree.
type SyncOptions struct {
	// WorkDir is the git working directory (session worktree).
	WorkDir string
	// Onto is the ref to rebase/merge onto. Empty uses DefaultBase.
	Onto string
	// Merge uses merge instead of rebase.
	Merge bool
	// Autostash stashes dirty changes before sync and pops afterward.
	Autostash bool
}

// SyncResult is the outcome of syncing a worktree onto a base ref.
type SyncResult struct {
	Onto      string   `json:"onto"`
	Strategy  string   `json:"strategy"`
	BeforeSHA string   `json:"beforeSha"`
	AfterSHA  string   `json:"afterSha"`
	Conflicts []string `json:"conflicts,omitempty"`
	Stashed   bool     `json:"stashed,omitempty"`
}

// Sync rebases (default) or merges WorkDir onto Onto.
// On conflict, aborts the in-progress operation and returns Conflicts.
func (m *Manager) Sync(ctx context.Context, opts SyncOptions) (*SyncResult, error) {
	dir, onto, strategy, err := m.resolveSyncTarget(ctx, opts)
	if err != nil {
		return nil, err
	}
	before, beforeErr := m.HeadSHA(ctx, dir)
	if beforeErr != nil {
		return nil, beforeErr
	}
	res := &SyncResult{Onto: onto, Strategy: strategy, BeforeSHA: before}

	stashed, stashErr := prepareSyncStash(ctx, dir, opts.Autostash)
	if stashErr != nil {
		return nil, stashErr
	}
	res.Stashed = stashed

	if syncErr := runSyncOp(ctx, dir, onto, opts.Merge); syncErr != nil {
		return m.failSync(ctx, dir, res, opts.Merge, syncErr)
	}
	if res.Stashed {
		if popErr := runGit(ctx, dir, "stash", "pop"); popErr != nil {
			res.Conflicts = conflictedPaths(ctx, dir)
			res.AfterSHA, _ = m.HeadSHA(ctx, dir)
			return res, fmt.Errorf("worktree: stash pop after sync: %w", popErr)
		}
	}
	after, afterErr := m.HeadSHA(ctx, dir)
	if afterErr != nil {
		return res, afterErr
	}
	res.AfterSHA = after
	return res, nil
}

func (m *Manager) resolveSyncTarget(ctx context.Context, opts SyncOptions) (dir, onto, strategy string, err error) {
	dir = opts.WorkDir
	if dir == "" {
		dir = m.repoRoot
	}
	onto = strings.TrimSpace(opts.Onto)
	if onto == "" {
		onto, err = m.DefaultBase(ctx)
		if err != nil {
			return "", "", "", err
		}
	}
	strategy = "rebase"
	if opts.Merge {
		strategy = "merge"
	}
	return dir, onto, strategy, nil
}

func prepareSyncStash(ctx context.Context, dir string, autostash bool) (bool, error) {
	dirty, dirtyErr := isDirty(ctx, dir, true)
	if dirtyErr != nil {
		return false, dirtyErr
	}
	if !dirty {
		return false, nil
	}
	if !autostash {
		return false, fmt.Errorf("worktree: dirty working tree (commit first, or pass --autostash)")
	}
	if stashErr := runGit(ctx, dir, "stash", "push", "-u", "-m", "specular session sync"); stashErr != nil {
		return false, fmt.Errorf("worktree: stash: %w", stashErr)
	}
	return true, nil
}

func isDirty(ctx context.Context, dir string, includeUntracked bool) (bool, error) {
	args := []string{"status", "--porcelain"}
	if !includeUntracked {
		args = append(args, "--untracked-files=no")
	}
	out, err := runGitOutput(ctx, dir, args...)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func runSyncOp(ctx context.Context, dir, onto string, merge bool) error {
	if merge {
		return runGit(ctx, dir, "merge", "--no-edit", onto)
	}
	return runGit(ctx, dir, "rebase", onto)
}

func (m *Manager) failSync(ctx context.Context, dir string, res *SyncResult, merge bool, syncErr error) (*SyncResult, error) {
	res.Conflicts = conflictedPaths(ctx, dir)
	_ = abortSync(ctx, dir, merge)
	if res.Stashed {
		_ = runGit(ctx, dir, "stash", "pop")
	}
	res.AfterSHA, _ = m.HeadSHA(ctx, dir)
	return res, fmt.Errorf("worktree: %s onto %s failed: %w", res.Strategy, res.Onto, syncErr)
}

func abortSync(ctx context.Context, dir string, merge bool) error {
	if merge {
		return runGit(ctx, dir, "merge", "--abort")
	}
	return runGit(ctx, dir, "rebase", "--abort")
}

func conflictedPaths(ctx context.Context, dir string) []string {
	out, err := runGitOutput(ctx, dir, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil
	}
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths
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
