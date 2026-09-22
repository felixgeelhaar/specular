package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IntegrableHarness lists harnesses that support `session integrate`
// (native agent hook install into the repo).
var IntegrableHarness = []string{
	"claude-code",
	"claude",
	"cursor",
	"cursor-agent",
}

// IntegrateOptions configures native agent hook installation.
type IntegrateOptions struct {
	// Harness selects the coding-agent hook target (e.g. claude-code, cursor).
	Harness string
	// Root is the repository root to write into (required).
	Root string
	// DryRun reports planned writes without touching the filesystem.
	DryRun bool
	// Force overwrites an existing Specular hook script.
	Force bool
}

// IntegrateFileAction describes what happened (or would happen) to a path.
type IntegrateFileAction string

const (
	// IntegrateWrite creates a new file.
	IntegrateWrite IntegrateFileAction = "write"
	// IntegrateMerge updates an existing settings file while preserving other keys.
	IntegrateMerge IntegrateFileAction = "merge"
	// IntegrateSkip leaves an existing Specular hook unchanged.
	IntegrateSkip IntegrateFileAction = "skip"
	// IntegrateDryRun records a planned change without writing.
	IntegrateDryRun IntegrateFileAction = "dry-run"
)

// IntegrateFile is one planned or applied filesystem change.
type IntegrateFile struct {
	Path    string              `json:"path"`
	Action  IntegrateFileAction `json:"action"`
	Content string              `json:"content,omitempty"`
}

// IntegrateResult summarizes native hook installation.
type IntegrateResult struct {
	Harness   string          `json:"harness"`
	Root      string          `json:"root"`
	DryRun    bool            `json:"dryRun"`
	Files     []IntegrateFile `json:"files"`
	NextSteps []string        `json:"nextSteps,omitempty"`
}

const (
	claudeSettingsRel  = ".claude/settings.json"
	claudeHookRel      = ".claude/hooks/specular-session-stop.sh"
	cursorHooksJSONRel = ".cursor/hooks.json"
	cursorHookRel      = ".cursor/hooks/specular-session-stop.sh"
	specularHookMarker = "specular-session-stop.sh"
)

// IsIntegrableHarness reports whether integrate supports the harness label.
func IsIntegrableHarness(h string) bool {
	switch normalizeHarness(h) {
	case "claude-code", "claude", "cursor", "cursor-agent":
		return true
	default:
		return false
	}
}

// canonicalIntegrableHarness maps aliases to the label recorded in results.
func canonicalIntegrableHarness(h string) string {
	switch normalizeHarness(h) {
	case "claude", "claude-code":
		return "claude-code"
	case "cursor", "cursor-agent":
		return "cursor"
	default:
		return normalizeHarness(h)
	}
}

// Integrate writes a minimal native agent hook for the given harness so
// Stop/completion can call specular session attest + specular gate using
// existing session harness labels and attestation provenance.
func Integrate(opts IntegrateOptions) (*IntegrateResult, error) {
	root := strings.TrimSpace(opts.Root)
	if root == "" {
		return nil, fmt.Errorf("session integrate: root is required")
	}
	abs, absErr := filepath.Abs(root)
	if absErr != nil {
		return nil, fmt.Errorf("session integrate: resolve root: %w", absErr)
	}
	harness := normalizeHarness(opts.Harness)
	if !IsIntegrableHarness(harness) {
		return nil, fmt.Errorf("session integrate: unsupported harness %q (supported: %s)",
			opts.Harness, strings.Join(IntegrableHarness, ", "))
	}
	harness = canonicalIntegrableHarness(harness)

	res := &IntegrateResult{
		Harness: harness,
		Root:    abs,
		DryRun:  opts.DryRun,
	}

	switch harness {
	case "claude-code":
		res.NextSteps = []string{
			"Commit .claude/hooks/ and .claude/settings.json when ready to share with the team",
			"Start a managed session: specular session start --harness claude-code --governed \"…\"",
			"session start exports SPECULAR_SESSION_ID / SPECULAR_SESSION_HARNESS for the Stop hook",
			"On Stop, the hook runs: specular session attest <id> then specular gate (advisory)",
		}
		return planAndApplyClaude(res, opts)
	case "cursor":
		res.NextSteps = []string{
			"Commit .cursor/hooks/ and .cursor/hooks.json when ready to share with the team",
			"Ensure Cursor loads project hooks (trusted workspace; Settings → Hooks)",
			"session start exports SPECULAR_SESSION_ID / SPECULAR_SESSION_HARNESS — export them into the Cursor agent environment when using a managed session",
			"On Agent stop, the hook runs: specular session attest <id> then specular gate (advisory)",
		}
		return planAndApplyCursor(res, opts)
	default:
		return nil, fmt.Errorf("session integrate: unsupported harness %q", harness)
	}
}

func planAndApplyClaude(res *IntegrateResult, opts IntegrateOptions) (*IntegrateResult, error) {
	hookBody := claudeStopHookScript()
	hookPath := filepath.Join(res.Root, claudeHookRel)
	settingsPath := filepath.Join(res.Root, claudeSettingsRel)

	hookAction, hookContent, hookErr := planHookScript(hookPath, hookBody, opts.DryRun, opts.Force)
	if hookErr != nil {
		return nil, hookErr
	}
	res.Files = append(res.Files, IntegrateFile{
		Path:    filepath.ToSlash(claudeHookRel),
		Action:  hookAction,
		Content: maybeContent(opts.DryRun, hookContent),
	})

	settingsBody, settingsAction, settingsErr := planClaudeSettings(settingsPath, opts.DryRun)
	if settingsErr != nil {
		return nil, settingsErr
	}
	res.Files = append(res.Files, IntegrateFile{
		Path:    filepath.ToSlash(claudeSettingsRel),
		Action:  settingsAction,
		Content: maybeContent(opts.DryRun, settingsBody),
	})

	if opts.DryRun {
		return res, nil
	}
	if err := applyIntegrateWrites(hookPath, hookBody, hookAction, settingsPath, settingsBody, settingsAction); err != nil {
		return nil, err
	}
	return res, nil
}

func planAndApplyCursor(res *IntegrateResult, opts IntegrateOptions) (*IntegrateResult, error) {
	hookBody := cursorStopHookScript()
	hookPath := filepath.Join(res.Root, cursorHookRel)
	hooksJSONPath := filepath.Join(res.Root, cursorHooksJSONRel)

	hookAction, hookContent, hookErr := planHookScript(hookPath, hookBody, opts.DryRun, opts.Force)
	if hookErr != nil {
		return nil, hookErr
	}
	res.Files = append(res.Files, IntegrateFile{
		Path:    filepath.ToSlash(cursorHookRel),
		Action:  hookAction,
		Content: maybeContent(opts.DryRun, hookContent),
	})

	hooksBody, hooksAction, hooksErr := planCursorHooksJSON(hooksJSONPath, opts.DryRun)
	if hooksErr != nil {
		return nil, hooksErr
	}
	res.Files = append(res.Files, IntegrateFile{
		Path:    filepath.ToSlash(cursorHooksJSONRel),
		Action:  hooksAction,
		Content: maybeContent(opts.DryRun, hooksBody),
	})

	if opts.DryRun {
		return res, nil
	}
	if err := applyIntegrateWrites(hookPath, hookBody, hookAction, hooksJSONPath, hooksBody, hooksAction); err != nil {
		return nil, err
	}
	return res, nil
}

func applyIntegrateWrites(hookPath, hookBody string, hookAction IntegrateFileAction, settingsPath, settingsBody string, settingsAction IntegrateFileAction) error {
	if hookAction == IntegrateWrite {
		if mkdirErr := os.MkdirAll(filepath.Dir(hookPath), 0o750); mkdirErr != nil {
			return fmt.Errorf("session integrate: create hooks dir: %w", mkdirErr)
		}
		if writeErr := os.WriteFile(hookPath, []byte(hookBody), 0o600); writeErr != nil {
			return fmt.Errorf("session integrate: write hook: %w", writeErr)
		}
		// Owner-executable Stop hook; gosec G302 flags any mode with +x.
		if chmodErr := os.Chmod(hookPath, 0o700); chmodErr != nil { //nolint:gosec // G302: shell hook must be executable by owner
			return fmt.Errorf("session integrate: chmod hook: %w", chmodErr)
		}
	}

	if settingsAction == IntegrateWrite || settingsAction == IntegrateMerge {
		if mkdirErr := os.MkdirAll(filepath.Dir(settingsPath), 0o750); mkdirErr != nil {
			return fmt.Errorf("session integrate: create config dir: %w", mkdirErr)
		}
		if writeErr := os.WriteFile(settingsPath, []byte(settingsBody), 0o600); writeErr != nil {
			return fmt.Errorf("session integrate: write settings: %w", writeErr)
		}
	}
	return nil
}

func maybeContent(dryRun bool, content string) string {
	if dryRun {
		return content
	}
	return ""
}

func planHookScript(path, body string, dryRun, force bool) (IntegrateFileAction, string, error) {
	_, err := os.Stat(path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return "", "", fmt.Errorf("session integrate: stat hook: %w", err)
	}
	if dryRun {
		return IntegrateDryRun, body, nil
	}
	if exists && !force {
		return IntegrateSkip, body, nil
	}
	return IntegrateWrite, body, nil
}

func planClaudeSettings(path string, dryRun bool) (string, IntegrateFileAction, error) {
	existing := map[string]interface{}{}
	action := IntegrateWrite
	raw, readErr := os.ReadFile(path)
	if readErr == nil {
		if len(strings.TrimSpace(string(raw))) > 0 {
			if unmarshalErr := json.Unmarshal(raw, &existing); unmarshalErr != nil {
				return "", "", fmt.Errorf("session integrate: parse %s: %w", claudeSettingsRel, unmarshalErr)
			}
			action = IntegrateMerge
		}
	} else if !os.IsNotExist(readErr) {
		return "", "", fmt.Errorf("session integrate: read settings: %w", readErr)
	}

	merged, changed, mergeErr := mergeClaudeStopHook(existing)
	if mergeErr != nil {
		return "", "", mergeErr
	}
	out, marshalErr := json.MarshalIndent(merged, "", "  ")
	if marshalErr != nil {
		return "", "", fmt.Errorf("session integrate: marshal settings: %w", marshalErr)
	}
	outStr := string(out) + "\n"

	if dryRun {
		return outStr, IntegrateDryRun, nil
	}
	if action == IntegrateMerge && !changed {
		return outStr, IntegrateSkip, nil
	}
	return outStr, action, nil
}

func planCursorHooksJSON(path string, dryRun bool) (string, IntegrateFileAction, error) {
	existing := map[string]interface{}{}
	action := IntegrateWrite
	raw, readErr := os.ReadFile(path)
	if readErr == nil {
		if len(strings.TrimSpace(string(raw))) > 0 {
			if unmarshalErr := json.Unmarshal(raw, &existing); unmarshalErr != nil {
				return "", "", fmt.Errorf("session integrate: parse %s: %w", cursorHooksJSONRel, unmarshalErr)
			}
			action = IntegrateMerge
		}
	} else if !os.IsNotExist(readErr) {
		return "", "", fmt.Errorf("session integrate: read hooks.json: %w", readErr)
	}

	merged, changed, mergeErr := mergeCursorStopHook(existing)
	if mergeErr != nil {
		return "", "", mergeErr
	}
	out, marshalErr := json.MarshalIndent(merged, "", "  ")
	if marshalErr != nil {
		return "", "", fmt.Errorf("session integrate: marshal hooks.json: %w", marshalErr)
	}
	outStr := string(out) + "\n"

	if dryRun {
		return outStr, IntegrateDryRun, nil
	}
	if action == IntegrateMerge && !changed {
		return outStr, IntegrateSkip, nil
	}
	return outStr, action, nil
}

// mergeClaudeStopHook ensures hooks.Stop includes the Specular command hook.
// Returns the merged document and whether it changed.
func mergeClaudeStopHook(doc map[string]interface{}) (map[string]interface{}, bool, error) {
	if doc == nil {
		doc = map[string]interface{}{}
	}
	hooksObj, _ := doc["hooks"].(map[string]interface{})
	if hooksObj == nil {
		hooksObj = map[string]interface{}{}
	}

	stopList, _ := hooksObj["Stop"].([]interface{})
	if stopList == nil {
		// JSON numbers decode as float64; empty slice typed as []interface{}.
		if raw, ok := hooksObj["Stop"]; ok && raw != nil {
			return nil, false, fmt.Errorf("session integrate: hooks.Stop must be an array")
		}
		stopList = []interface{}{}
	}

	if stopHookPresent(stopList) {
		doc["hooks"] = hooksObj
		hooksObj["Stop"] = stopList
		return doc, false, nil
	}

	entry := map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": "$CLAUDE_PROJECT_DIR/.claude/hooks/specular-session-stop.sh",
			},
		},
	}
	stopList = append(stopList, entry)
	hooksObj["Stop"] = stopList
	doc["hooks"] = hooksObj
	return doc, true, nil
}

// mergeCursorStopHook ensures hooks.stop includes the Specular command hook.
// Cursor project hooks live in .cursor/hooks.json (version + hooks map).
func mergeCursorStopHook(doc map[string]interface{}) (map[string]interface{}, bool, error) {
	if doc == nil {
		doc = map[string]interface{}{}
	}
	changed := false
	if _, hasVersion := doc["version"]; !hasVersion {
		doc["version"] = 1
		changed = true
	}

	hooksObj, _ := doc["hooks"].(map[string]interface{})
	if hooksObj == nil {
		hooksObj = map[string]interface{}{}
		changed = true
	}

	stopList, _ := hooksObj["stop"].([]interface{})
	if stopList == nil {
		if raw, ok := hooksObj["stop"]; ok && raw != nil {
			return nil, false, fmt.Errorf("session integrate: hooks.stop must be an array")
		}
		stopList = []interface{}{}
	}

	if cursorStopHookPresent(stopList) {
		doc["hooks"] = hooksObj
		hooksObj["stop"] = stopList
		return doc, changed, nil
	}

	entry := map[string]interface{}{
		"command": ".cursor/hooks/specular-session-stop.sh",
	}
	stopList = append(stopList, entry)
	hooksObj["stop"] = stopList
	doc["hooks"] = hooksObj
	return doc, true, nil
}

func stopHookPresent(stopList []interface{}) bool {
	for _, item := range stopList {
		group, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		inner, _ := group["hooks"].([]interface{})
		for _, h := range inner {
			hm, isMap := h.(map[string]interface{})
			if !isMap {
				continue
			}
			cmd, _ := hm["command"].(string)
			if strings.Contains(cmd, specularHookMarker) {
				return true
			}
		}
	}
	return false
}

func cursorStopHookPresent(stopList []interface{}) bool {
	for _, item := range stopList {
		hm, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		cmd, _ := hm["command"].(string)
		if strings.Contains(cmd, specularHookMarker) {
			return true
		}
	}
	return false
}

func claudeStopHookScript() string {
	return `#!/usr/bin/env bash
# Specular native Stop hook for Claude Code (PRODUCT_INTENT P1 #4).
# Installed by: specular session integrate claude-code
# Calls existing session attest + gate surfaces — no new protocol.
set -euo pipefail

# Claude Code feeds Stop event JSON on stdin.
input="$(cat || true)"
if command -v jq >/dev/null 2>&1 && [[ -n "$input" ]]; then
  if [[ "$(echo "$input" | jq -r '.stop_hook_active // false')" == "true" ]]; then
    exit 0
  fi
fi

ROOT="${CLAUDE_PROJECT_DIR:-}"
if [[ -z "$ROOT" ]]; then
  ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
fi
cd "$ROOT"

if ! command -v specular >/dev/null 2>&1; then
  echo "specular: CLI not on PATH; skip attest/gate" >&2
  exit 0
fi

SESSION_ID="${SPECULAR_SESSION_ID:-}"
if [[ -z "$SESSION_ID" && -d .specular/sessions ]] && command -v jq >/dev/null 2>&1; then
  # Prefer the newest session record labeled claude-code / claude (portable mtime).
  newest_mtime=0
  for path in .specular/sessions/*.json; do
    [[ -f "$path" ]] || continue
    case "$path" in *.attestation.json) continue ;; esac
    harness="$(jq -r '.harness // empty' "$path" 2>/dev/null || true)"
    case "$harness" in claude-code|claude) ;; *) continue ;; esac
    mtime="$(stat -c %Y "$path" 2>/dev/null || stat -f %m "$path" 2>/dev/null || echo 0)"
    if [[ "$mtime" -ge "$newest_mtime" ]]; then
      newest_mtime="$mtime"
      SESSION_ID="$(jq -r '.id // empty' "$path" 2>/dev/null || true)"
    fi
  done
fi

if [[ -n "$SESSION_ID" ]]; then
  echo "specular: attesting session ${SESSION_ID} (harness provenance)" >&2
  specular session attest "$SESSION_ID" || echo "specular: attest failed (advisory)" >&2
else
  echo "specular: no SPECULAR_SESSION_ID / claude-code session; skip attest" >&2
fi

echo "specular: running gate (advisory — does not block Stop)" >&2
specular gate || echo "specular: gate exited non-zero (advisory)" >&2
exit 0
`
}

func cursorStopHookScript() string {
	return `#!/usr/bin/env bash
# Specular native stop hook for Cursor (PRODUCT_INTENT P1 #4).
# Installed by: specular session integrate cursor
# Calls existing session attest + gate surfaces — no new protocol.
# Project hooks run from the repo root; register via .cursor/hooks.json.
set -euo pipefail

# Cursor feeds stop-event JSON on stdin; drain it (advisory — never block stop).
cat >/dev/null || true

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

if ! command -v specular >/dev/null 2>&1; then
  echo "specular: CLI not on PATH; skip attest/gate" >&2
  printf '%s\n' '{}'
  exit 0
fi

SESSION_ID="${SPECULAR_SESSION_ID:-}"
if [[ -z "$SESSION_ID" && -d .specular/sessions ]] && command -v jq >/dev/null 2>&1; then
  # Prefer the newest session record labeled cursor / cursor-agent (portable mtime).
  newest_mtime=0
  for path in .specular/sessions/*.json; do
    [[ -f "$path" ]] || continue
    case "$path" in *.attestation.json) continue ;; esac
    harness="$(jq -r '.harness // empty' "$path" 2>/dev/null || true)"
    case "$harness" in cursor|cursor-agent) ;; *) continue ;; esac
    mtime="$(stat -c %Y "$path" 2>/dev/null || stat -f %m "$path" 2>/dev/null || echo 0)"
    if [[ "$mtime" -ge "$newest_mtime" ]]; then
      newest_mtime="$mtime"
      SESSION_ID="$(jq -r '.id // empty' "$path" 2>/dev/null || true)"
    fi
  done
fi

if [[ -n "$SESSION_ID" ]]; then
  echo "specular: attesting session ${SESSION_ID} (harness provenance)" >&2
  specular session attest "$SESSION_ID" || echo "specular: attest failed (advisory)" >&2
else
  echo "specular: no SPECULAR_SESSION_ID / cursor session; skip attest" >&2
fi

echo "specular: running gate (advisory — does not block stop)" >&2
specular gate || echo "specular: gate exited non-zero (advisory)" >&2

# Cursor stop hooks may consume JSON on stdout; emit empty object (no follow-up).
printf '%s\n' '{}'
exit 0
`
}
