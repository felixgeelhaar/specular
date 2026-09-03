package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// KnownHarness lists supported coding-agent harnesses Specular can launch
// as first-class parallel sessions (the competitive inner loop).
var KnownHarness = []string{
	"specular-auto",
	"claude-code",
	"claude",
	"codex",
	"codex-cli",
	"gemini",
	"gemini-cli",
}

// LaunchPlan is the resolved process Specular will start for a session.
type LaunchPlan struct {
	// Binary is passed to safeutil.SafeCommand (PATH name or absolute path).
	Binary string
	// Args are the command arguments (already free of null bytes by construction).
	Args []string
	// WorkDir is the process working directory (worktree path when isolated).
	WorkDir string
	// Kind classifies the launch for provenance ("specular-auto" | "native").
	Kind string
}

// ResolveLaunch picks the binary + args for a harness.
// opts.Binary, when set, always overrides the executable (tests / wrappers).
func ResolveLaunch(opts StartOptions, rec *Record) (LaunchPlan, error) {
	if rec == nil {
		return LaunchPlan{}, fmt.Errorf("session: nil record")
	}
	harness := normalizeHarness(rec.Harness)
	workDir := rec.WorktreePath
	if workDir == "" {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return LaunchPlan{}, cwdErr
		}
		workDir = cwd
	}

	plan := LaunchPlan{WorkDir: workDir, Kind: "native"}

	switch harness {
	case "specular-auto":
		plan.Kind = "specular-auto"
		bin, binErr := resolveBinary(opts.Binary)
		if binErr != nil {
			return LaunchPlan{}, binErr
		}
		plan.Binary = bin
		plan.Args = buildAutoArgs(opts, rec)
		if !opts.SkipWorktree {
			if root, rootErr := findGitRoot(workDir); rootErr == nil {
				plan.WorkDir = root
			}
		}
		return plan, nil

	case "claude-code", "claude":
		bin, binErr := harnessBinary(opts.Binary, "claude")
		if binErr != nil {
			return LaunchPlan{}, binErr
		}
		plan.Binary = bin
		// Agentic non-interactive run inside an isolated worktree.
		plan.Args = []string{
			"--print",
			"--permission-mode", "acceptEdits",
			"--dangerously-skip-permissions",
			"--output-format", "text",
		}
		plan.Args = append(plan.Args, opts.ExtraArgs...)
		plan.Args = append(plan.Args, rec.Goal)
		return plan, nil

	case "codex", "codex-cli":
		bin, binErr := harnessBinary(opts.Binary, "codex")
		if binErr != nil {
			return LaunchPlan{}, binErr
		}
		plan.Binary = bin
		plan.Args = []string{"exec", "--full-auto"}
		plan.Args = append(plan.Args, opts.ExtraArgs...)
		plan.Args = append(plan.Args, rec.Goal)
		return plan, nil

	case "gemini", "gemini-cli":
		bin, binErr := harnessBinary(opts.Binary, "gemini")
		if binErr != nil {
			return LaunchPlan{}, binErr
		}
		plan.Binary = bin
		plan.Args = []string{"--prompt"}
		plan.Args = append(plan.Args, opts.ExtraArgs...)
		plan.Args = append(plan.Args, rec.Goal)
		return plan, nil

	default:
		return LaunchPlan{}, fmt.Errorf("session: unknown harness %q (supported: %s)",
			rec.Harness, strings.Join(KnownHarness, ", "))
	}
}

func harnessBinary(override, defaultName string) (string, error) {
	if override == "" {
		return defaultName, nil
	}
	if !filepath.IsAbs(override) {
		return "", fmt.Errorf("session: binary override must be absolute")
	}
	return override, nil
}

func normalizeHarness(h string) string {
	h = strings.TrimSpace(strings.ToLower(h))
	if h == "" {
		return "specular-auto"
	}
	return h
}

func findGitRoot(start string) (string, error) {
	dir := start
	for {
		if st, err := os.Stat(filepath.Join(dir, ".git")); err == nil && (st.IsDir() || st.Mode().IsRegular()) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository")
		}
		dir = parent
	}
}

// IsNativeHarness reports whether the harness launches an external coding agent.
func IsNativeHarness(h string) bool {
	switch normalizeHarness(h) {
	case "claude-code", "claude", "codex", "codex-cli", "gemini", "gemini-cli":
		return true
	default:
		return false
	}
}

func isKnownHarness(h string) bool {
	h = normalizeHarness(h)
	for _, known := range KnownHarness {
		if h == known {
			return true
		}
	}
	return false
}
