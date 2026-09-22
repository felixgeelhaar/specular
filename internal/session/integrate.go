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
	"codex",
	"codex-cli",
	"gemini",
	"gemini-cli",
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
	// Enforce installs fail-closed Stop/SessionEnd hooks (Level 3):
	// attest + gate --require-attested --require-protocol; non-zero on DENY.
	Enforce bool
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
	Enforce   bool            `json:"enforce,omitempty"`
	Files     []IntegrateFile `json:"files"`
	NextSteps []string        `json:"nextSteps,omitempty"`
}

const (
	claudeSettingsRel  = ".claude/settings.json"
	claudeHookRel      = ".claude/hooks/specular-session-stop.sh"
	cursorHooksJSONRel = ".cursor/hooks.json"
	cursorHookRel      = ".cursor/hooks/specular-session-stop.sh"
	codexHooksJSONRel  = ".codex/hooks.json"
	codexHookRel       = ".codex/hooks/specular-session-stop.sh"
	geminiSettingsRel  = ".gemini/settings.json"
	geminiHookRel      = ".gemini/hooks/specular-session-stop.sh"
	specularHookMarker = "specular-session-stop.sh"
)

// IsIntegrableHarness reports whether integrate supports the harness label.
func IsIntegrableHarness(h string) bool {
	switch normalizeHarness(h) {
	case "claude-code", "claude", "cursor", "cursor-agent",
		"codex", "codex-cli", "gemini", "gemini-cli":
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
	case "codex", "codex-cli":
		return "codex"
	case "gemini", "gemini-cli":
		return "gemini"
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
		Enforce: opts.Enforce,
	}

	hookMode := "advisory"
	if opts.Enforce {
		hookMode = "enforce (fail-closed)"
	}

	switch harness {
	case "claude-code":
		res.NextSteps = []string{
			"Commit .claude/hooks/ and .claude/settings.json when ready to share with the team",
			"Start a managed session: specular session start --harness claude-code --governed \"…\"",
			"session start exports SPECULAR_SESSION_ID / SPECULAR_SESSION_HARNESS for the Stop hook",
			fmt.Sprintf("On Stop, the hook runs: specular session attest <id> then specular gate (%s)", hookMode),
		}
		return planAndApplyClaude(res, opts)
	case "cursor":
		res.NextSteps = []string{
			"Commit .cursor/hooks/ and .cursor/hooks.json when ready to share with the team",
			"Ensure Cursor loads project hooks (trusted workspace; Settings → Hooks)",
			"session start exports SPECULAR_SESSION_ID / SPECULAR_SESSION_HARNESS — export them into the Cursor agent environment when using a managed session",
			fmt.Sprintf("On Agent stop, the hook runs: specular session attest <id> then specular gate (%s)", hookMode),
		}
		return planAndApplyCursor(res, opts)
	case "codex":
		res.NextSteps = []string{
			"Commit .codex/hooks/ and .codex/hooks.json when ready to share with the team",
			"Trust the project .codex/ layer and review the Stop hook via Codex /hooks",
			"Start a managed session: specular session start --harness codex --governed \"…\"",
			fmt.Sprintf("On Stop, the hook runs: specular session attest <id> then specular gate (%s)", hookMode),
		}
		return planAndApplyCodex(res, opts)
	case "gemini":
		res.NextSteps = []string{
			"Commit .gemini/hooks/ and .gemini/settings.json when ready to share with the team",
			"Ensure hooksConfig.enabled is true (integrate sets this when merging settings)",
			"Start a managed session: specular session start --harness gemini --governed \"…\"",
			fmt.Sprintf("On SessionEnd, the hook runs: specular session attest <id> then specular gate (%s)", hookMode),
		}
		return planAndApplyGemini(res, opts)
	default:
		return nil, fmt.Errorf("session integrate: unsupported harness %q", harness)
	}
}

func planAndApplyClaude(res *IntegrateResult, opts IntegrateOptions) (*IntegrateResult, error) {
	return planAndApplyPair(res, opts, claudeHookRel, claudeSettingsRel, claudeStopHookScript(opts.Enforce), planClaudeSettings)
}

func planAndApplyCursor(res *IntegrateResult, opts IntegrateOptions) (*IntegrateResult, error) {
	return planAndApplyPair(res, opts, cursorHookRel, cursorHooksJSONRel, cursorStopHookScript(opts.Enforce), planCursorHooksJSON)
}

func planAndApplyCodex(res *IntegrateResult, opts IntegrateOptions) (*IntegrateResult, error) {
	return planAndApplyPair(res, opts, codexHookRel, codexHooksJSONRel, codexStopHookScript(opts.Enforce), planCodexHooksJSON)
}

func planAndApplyGemini(res *IntegrateResult, opts IntegrateOptions) (*IntegrateResult, error) {
	return planAndApplyPair(res, opts, geminiHookRel, geminiSettingsRel, geminiSessionEndHookScript(opts.Enforce), planGeminiSettings)
}

func planAndApplyPair(
	res *IntegrateResult,
	opts IntegrateOptions,
	hookRel, configRel, hookBody string,
	planConfig func(path string, dryRun bool) (string, IntegrateFileAction, error),
) (*IntegrateResult, error) {
	hookPath := filepath.Join(res.Root, hookRel)
	configPath := filepath.Join(res.Root, configRel)

	hookAction, hookContent, hookErr := planHookScript(hookPath, hookBody, opts.DryRun, opts.Force)
	if hookErr != nil {
		return nil, hookErr
	}
	res.Files = append(res.Files, IntegrateFile{
		Path:    filepath.ToSlash(hookRel),
		Action:  hookAction,
		Content: maybeContent(opts.DryRun, hookContent),
	})

	configBody, configAction, configErr := planConfig(configPath, opts.DryRun)
	if configErr != nil {
		return nil, configErr
	}
	res.Files = append(res.Files, IntegrateFile{
		Path:    filepath.ToSlash(configRel),
		Action:  configAction,
		Content: maybeContent(opts.DryRun, configBody),
	})

	if opts.DryRun {
		return res, nil
	}
	if err := applyIntegrateWrites(hookPath, hookBody, hookAction, configPath, configBody, configAction); err != nil {
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

type jsonMergeFn func(map[string]interface{}) (map[string]interface{}, bool, error)

func planClaudeSettings(path string, dryRun bool) (string, IntegrateFileAction, error) {
	return planJSONConfig(path, claudeSettingsRel, "settings", dryRun, mergeClaudeStopHook)
}

func planCursorHooksJSON(path string, dryRun bool) (string, IntegrateFileAction, error) {
	return planJSONConfig(path, cursorHooksJSONRel, "hooks.json", dryRun, mergeCursorStopHook)
}

func planCodexHooksJSON(path string, dryRun bool) (string, IntegrateFileAction, error) {
	return planJSONConfig(path, codexHooksJSONRel, "hooks.json", dryRun, mergeCodexStopHook)
}

func planGeminiSettings(path string, dryRun bool) (string, IntegrateFileAction, error) {
	return planJSONConfig(path, geminiSettingsRel, "settings", dryRun, mergeGeminiSessionEndHook)
}

func planJSONConfig(path, relLabel, kind string, dryRun bool, merge jsonMergeFn) (string, IntegrateFileAction, error) {
	existing := map[string]interface{}{}
	action := IntegrateWrite
	raw, readErr := os.ReadFile(path)
	if readErr == nil {
		if len(strings.TrimSpace(string(raw))) > 0 {
			if unmarshalErr := json.Unmarshal(raw, &existing); unmarshalErr != nil {
				return "", "", fmt.Errorf("session integrate: parse %s: %w", relLabel, unmarshalErr)
			}
			action = IntegrateMerge
		}
	} else if !os.IsNotExist(readErr) {
		return "", "", fmt.Errorf("session integrate: read %s: %w", kind, readErr)
	}

	merged, changed, mergeErr := merge(existing)
	if mergeErr != nil {
		return "", "", mergeErr
	}
	out, marshalErr := json.MarshalIndent(merged, "", "  ")
	if marshalErr != nil {
		return "", "", fmt.Errorf("session integrate: marshal %s: %w", kind, marshalErr)
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

// mergeCodexStopHook ensures hooks.Stop includes the Specular command hook.
// Codex project hooks live in .codex/hooks.json (Claude-style nested Stop).
func mergeCodexStopHook(doc map[string]interface{}) (map[string]interface{}, bool, error) {
	if doc == nil {
		doc = map[string]interface{}{}
	}
	hooksObj, _ := doc["hooks"].(map[string]interface{})
	if hooksObj == nil {
		hooksObj = map[string]interface{}{}
	}

	stopList, _ := hooksObj["Stop"].([]interface{})
	if stopList == nil {
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
				"command": "$(git rev-parse --show-toplevel)/.codex/hooks/specular-session-stop.sh",
				"timeout": 60,
			},
		},
	}
	stopList = append(stopList, entry)
	hooksObj["Stop"] = stopList
	doc["hooks"] = hooksObj
	return doc, true, nil
}

// mergeGeminiSessionEndHook ensures hooks.SessionEnd includes Specular and
// enables hooksConfig.enabled on project .gemini/settings.json.
func mergeGeminiSessionEndHook(doc map[string]interface{}) (map[string]interface{}, bool, error) {
	if doc == nil {
		doc = map[string]interface{}{}
	}
	changed := false

	hooksCfg, _ := doc["hooksConfig"].(map[string]interface{})
	if hooksCfg == nil {
		hooksCfg = map[string]interface{}{}
	}
	if enabled, ok := hooksCfg["enabled"].(bool); !ok || !enabled {
		hooksCfg["enabled"] = true
		doc["hooksConfig"] = hooksCfg
		changed = true
	} else {
		doc["hooksConfig"] = hooksCfg
	}

	hooksObj, _ := doc["hooks"].(map[string]interface{})
	if hooksObj == nil {
		hooksObj = map[string]interface{}{}
		changed = true
	}

	endList, _ := hooksObj["SessionEnd"].([]interface{})
	if endList == nil {
		if raw, ok := hooksObj["SessionEnd"]; ok && raw != nil {
			return nil, false, fmt.Errorf("session integrate: hooks.SessionEnd must be an array")
		}
		endList = []interface{}{}
	}

	if stopHookPresent(endList) {
		doc["hooks"] = hooksObj
		hooksObj["SessionEnd"] = endList
		return doc, changed, nil
	}

	entry := map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"name":    "specular-session-stop",
				"type":    "command",
				"command": "$PWD/.gemini/hooks/specular-session-stop.sh",
			},
		},
	}
	endList = append(endList, entry)
	hooksObj["SessionEnd"] = endList
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

func hookModeComment(enforce bool) string {
	if enforce {
		return "# Specular hook mode: enforce"
	}
	return "# Specular hook mode: advisory"
}

func hookMissingCLIExit(enforce, emitJSON bool) string {
	var b strings.Builder
	b.WriteString("  echo \"specular: CLI not on PATH;")
	if enforce {
		b.WriteString(" enforce requires specular\" >&2\n")
	} else {
		b.WriteString(" skip attest/gate\" >&2\n")
	}
	if emitJSON {
		b.WriteString("  printf '%s\\n' '{}'\n")
	}
	if enforce {
		b.WriteString("  exit 1\n")
	} else {
		b.WriteString("  exit 0\n")
	}
	return b.String()
}

func hookAttestGateBlock(eventLabel string, enforce bool) string {
	if enforce {
		return fmt.Sprintf(`if [[ -z "$SESSION_ID" ]]; then
  echo "specular: no session id; enforce requires attest" >&2
  exit 1
fi
echo "specular: attesting session ${SESSION_ID} (harness provenance)" >&2
specular session attest "$SESSION_ID"
echo "specular: running gate (enforce — blocks %s on DENY)" >&2
specular gate --require-attested --require-protocol
`, eventLabel)
	}
	return fmt.Sprintf(`if [[ -n "$SESSION_ID" ]]; then
  echo "specular: attesting session ${SESSION_ID} (harness provenance)" >&2
  specular session attest "$SESSION_ID" || echo "specular: attest failed (advisory)" >&2
else
  echo "specular: no SPECULAR_SESSION_ID / matching session; skip attest" >&2
fi

echo "specular: running gate (advisory — does not block %s)" >&2
specular gate || echo "specular: gate exited non-zero (advisory)" >&2
`, eventLabel)
}

func claudeStopHookScript(enforce bool) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
# Specular native Stop hook for Claude Code (PRODUCT_INTENT P1 #4).
# Installed by: specular session integrate claude-code
# Calls existing session attest + gate surfaces — no new protocol.
%s
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
%sfi

SESSION_ID="${SPECULAR_SESSION_ID:-}"
if [[ -z "$SESSION_ID" && -d .specular/sessions ]] && command -v jq >/dev/null 2>&1; then
  # Prefer the newest session record labeled claude-code / claude (portable mtime).
  newest_mtime=0
  for path in .specular/sessions/*.json; do
    [[ -f "$path" ]] || continue
    case "$path" in *.attestation.json) continue ;; esac
    harness="$(jq -r '.harness // empty' "$path" 2>/dev/null || true)"
    case "$harness" in claude-code|claude) ;; *) continue ;; esac
    mtime="$(stat -c %%Y "$path" 2>/dev/null || stat -f %%m "$path" 2>/dev/null || echo 0)"
    if [[ "$mtime" -ge "$newest_mtime" ]]; then
      newest_mtime="$mtime"
      SESSION_ID="$(jq -r '.id // empty' "$path" 2>/dev/null || true)"
    fi
  done
fi

%s
`, hookModeComment(enforce), hookMissingCLIExit(enforce, false), hookAttestGateBlock("Stop", enforce))
}

func cursorStopHookScript(enforce bool) string {
	tail := hookAttestGateBlock("stop", enforce)
	if enforce {
		tail += "\n# Cursor stop hooks may consume JSON on stdout; emit empty object (no follow-up).\nprintf '%s\\n' '{}'\n"
	} else {
		tail += "\n# Cursor stop hooks may consume JSON on stdout; emit empty object (no follow-up).\nprintf '%s\\n' '{}'\nexit 0\n"
	}
	return fmt.Sprintf(`#!/usr/bin/env bash
# Specular native stop hook for Cursor (PRODUCT_INTENT P1 #4).
# Installed by: specular session integrate cursor
# Calls existing session attest + gate surfaces — no new protocol.
# Project hooks run from the repo root; register via .cursor/hooks.json.
%s
set -euo pipefail

# Cursor feeds stop-event JSON on stdin; drain it.
cat >/dev/null || true

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

if ! command -v specular >/dev/null 2>&1; then
%sfi

SESSION_ID="${SPECULAR_SESSION_ID:-}"
if [[ -z "$SESSION_ID" && -d .specular/sessions ]] && command -v jq >/dev/null 2>&1; then
  # Prefer the newest session record labeled cursor / cursor-agent (portable mtime).
  newest_mtime=0
  for path in .specular/sessions/*.json; do
    [[ -f "$path" ]] || continue
    case "$path" in *.attestation.json) continue ;; esac
    harness="$(jq -r '.harness // empty' "$path" 2>/dev/null || true)"
    case "$harness" in cursor|cursor-agent) ;; *) continue ;; esac
    mtime="$(stat -c %%Y "$path" 2>/dev/null || stat -f %%m "$path" 2>/dev/null || echo 0)"
    if [[ "$mtime" -ge "$newest_mtime" ]]; then
      newest_mtime="$mtime"
      SESSION_ID="$(jq -r '.id // empty' "$path" 2>/dev/null || true)"
    fi
  done
fi

%s`, hookModeComment(enforce), hookMissingCLIExit(enforce, true), tail)
}

func codexStopHookScript(enforce bool) string {
	tail := hookAttestGateBlock("Stop", enforce)
	if !enforce {
		tail += "exit 0\n"
	}
	return fmt.Sprintf(`#!/usr/bin/env bash
# Specular native Stop hook for Codex (PRODUCT_INTENT P1 #4).
# Installed by: specular session integrate codex
# Calls existing session attest + gate surfaces — no new protocol.
# Register via .codex/hooks.json (hooks.Stop). Prefer Stop over SessionEnd
# so attest+gate have enough timeout budget.
%s
set -euo pipefail

# Codex feeds Stop event JSON on stdin; drain.
cat >/dev/null || true

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

if ! command -v specular >/dev/null 2>&1; then
%sfi

SESSION_ID="${SPECULAR_SESSION_ID:-}"
if [[ -z "$SESSION_ID" && -d .specular/sessions ]] && command -v jq >/dev/null 2>&1; then
  # Prefer the newest session record labeled codex / codex-cli (portable mtime).
  newest_mtime=0
  for path in .specular/sessions/*.json; do
    [[ -f "$path" ]] || continue
    case "$path" in *.attestation.json) continue ;; esac
    harness="$(jq -r '.harness // empty' "$path" 2>/dev/null || true)"
    case "$harness" in codex|codex-cli) ;; *) continue ;; esac
    mtime="$(stat -c %%Y "$path" 2>/dev/null || stat -f %%m "$path" 2>/dev/null || echo 0)"
    if [[ "$mtime" -ge "$newest_mtime" ]]; then
      newest_mtime="$mtime"
      SESSION_ID="$(jq -r '.id // empty' "$path" 2>/dev/null || true)"
    fi
  done
fi

%s`, hookModeComment(enforce), hookMissingCLIExit(enforce, false), tail)
}

func geminiSessionEndHookScript(enforce bool) string {
	tail := hookAttestGateBlock("SessionEnd", enforce)
	if enforce {
		tail += "\n# Gemini SessionEnd requires JSON-only stdout; logs go to stderr.\nprintf '%s\\n' '{}'\n"
	} else {
		tail += "\n# Gemini SessionEnd is best-effort; emit empty JSON object on stdout.\nprintf '%s\\n' '{}'\nexit 0\n"
	}
	return fmt.Sprintf(`#!/usr/bin/env bash
# Specular native SessionEnd hook for Gemini CLI (PRODUCT_INTENT P1 #4).
# Installed by: specular session integrate gemini
# Calls existing session attest + gate surfaces — no new protocol.
# Gemini hooks require JSON-only stdout; logs go to stderr.
%s
set -euo pipefail

# SessionEnd feeds JSON on stdin; drain it.
cat >/dev/null || true

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

if ! command -v specular >/dev/null 2>&1; then
%sfi

SESSION_ID="${SPECULAR_SESSION_ID:-}"
if [[ -z "$SESSION_ID" && -d .specular/sessions ]] && command -v jq >/dev/null 2>&1; then
  # Prefer the newest session record labeled gemini / gemini-cli (portable mtime).
  newest_mtime=0
  for path in .specular/sessions/*.json; do
    [[ -f "$path" ]] || continue
    case "$path" in *.attestation.json) continue ;; esac
    harness="$(jq -r '.harness // empty' "$path" 2>/dev/null || true)"
    case "$harness" in gemini|gemini-cli) ;; *) continue ;; esac
    mtime="$(stat -c %%Y "$path" 2>/dev/null || stat -f %%m "$path" 2>/dev/null || echo 0)"
    if [[ "$mtime" -ge "$newest_mtime" ]]; then
      newest_mtime="$mtime"
      SESSION_ID="$(jq -r '.id // empty' "$path" 2>/dev/null || true)"
    fi
  done
fi

%s`, hookModeComment(enforce), hookMissingCLIExit(enforce, true), tail)
}
