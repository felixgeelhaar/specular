package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsIntegrableHarness(t *testing.T) {
	t.Parallel()
	for _, h := range []string{
		"claude-code", "claude", "Claude-Code",
		"cursor", "cursor-agent", "Cursor",
		"codex", "codex-cli", "Codex",
		"gemini", "gemini-cli", "Gemini",
	} {
		if !IsIntegrableHarness(h) {
			t.Fatalf("expected integrable: %s", h)
		}
	}
	for _, h := range []string{"specular-auto", "aider", ""} {
		if IsIntegrableHarness(h) {
			t.Fatalf("expected not integrable: %s", h)
		}
	}
}

func TestIntegrateDryRunClaudeCode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res, err := Integrate(IntegrateOptions{
		Harness: "claude-code",
		Root:    dir,
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun || res.Harness != "claude-code" {
		t.Fatalf("%+v", res)
	}
	if len(res.Files) != 2 {
		t.Fatalf("files=%d", len(res.Files))
	}
	for _, f := range res.Files {
		if f.Action != IntegrateDryRun {
			t.Fatalf("action=%s path=%s", f.Action, f.Path)
		}
		if f.Content == "" {
			t.Fatalf("dry-run missing content for %s", f.Path)
		}
		if _, err := os.Stat(filepath.Join(dir, f.Path)); !os.IsNotExist(err) {
			t.Fatalf("dry-run must not write %s: %v", f.Path, err)
		}
	}
	hook := res.Files[0].Content
	if !strings.Contains(hook, "specular session attest") || !strings.Contains(hook, "specular gate") {
		t.Fatalf("hook missing attest/gate:\n%s", hook)
	}
	if !strings.Contains(hook, "SPECULAR_SESSION_ID") {
		t.Fatalf("hook missing session id env:\n%s", hook)
	}
	if !strings.Contains(hook, "Specular hook mode: advisory") {
		t.Fatalf("expected advisory mode marker:\n%s", hook)
	}
	if strings.Contains(hook, "--require-attested") {
		t.Fatalf("advisory must not require attested:\n%s", hook)
	}
	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(res.Files[1].Content), &settings); err != nil {
		t.Fatal(err)
	}
	hooks := settings["hooks"].(map[string]interface{})
	stop := hooks["Stop"].([]interface{})
	if len(stop) != 1 {
		t.Fatalf("stop=%v", stop)
	}
}

func TestIntegrateEnforceFailClosed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res, err := Integrate(IntegrateOptions{
		Harness: "claude-code",
		Root:    dir,
		DryRun:  true,
		Enforce: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Enforce {
		t.Fatal("expected Enforce on result")
	}
	hook := res.Files[0].Content
	for _, want := range []string{
		"Specular hook mode: enforce",
		"--require-attested",
		"--require-protocol",
		"enforce — blocks Stop on DENY",
	} {
		if !strings.Contains(hook, want) {
			t.Fatalf("missing %q in:\n%s", want, hook)
		}
	}
	if strings.Contains(hook, "does not block") || strings.Contains(hook, "attest failed (advisory)") {
		t.Fatalf("enforce must not swallow failures:\n%s", hook)
	}
	joined := strings.Join(res.NextSteps, "\n")
	if !strings.Contains(joined, "enforce (fail-closed)") {
		t.Fatalf("nextSteps=%v", res.NextSteps)
	}

	// Force rewrite from advisory → enforce.
	if _, err := Integrate(IntegrateOptions{Harness: "cursor", Root: dir}); err != nil {
		t.Fatal(err)
	}
	adv, err := os.ReadFile(filepath.Join(dir, cursorHookRel))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(adv), "hook mode: advisory") {
		t.Fatalf("expected advisory install first:\n%s", adv)
	}
	if _, err := Integrate(IntegrateOptions{Harness: "cursor", Root: dir, Enforce: true, Force: true}); err != nil {
		t.Fatal(err)
	}
	enf, err := os.ReadFile(filepath.Join(dir, cursorHookRel))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(enf), "hook mode: enforce") || !strings.Contains(string(enf), "--require-attested") {
		t.Fatalf("force rewrite to enforce failed:\n%s", enf)
	}
}

func TestIntegrateWritesAndIdempotentMerge(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	res, err := Integrate(IntegrateOptions{Harness: "claude", Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.Harness != "claude-code" {
		t.Fatalf("canonical harness=%s", res.Harness)
	}
	hookPath := filepath.Join(dir, claudeHookRel)
	settingsPath := filepath.Join(dir, claudeSettingsRel)
	if st, err := os.Stat(hookPath); err != nil || st.Mode()&0o100 == 0 {
		t.Fatalf("hook not executable: %v %#o", err, st.Mode())
	}
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), specularHookMarker) {
		t.Fatalf("settings missing hook: %s", raw)
	}

	// Second run without --force skips hook; settings already present → skip.
	res2, err := Integrate(IntegrateOptions{Harness: "claude-code", Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]IntegrateFileAction{}
	for _, f := range res2.Files {
		actions[f.Path] = f.Action
	}
	if actions[claudeHookRel] != IntegrateSkip {
		t.Fatalf("hook action=%s", actions[claudeHookRel])
	}
	if actions[claudeSettingsRel] != IntegrateSkip {
		t.Fatalf("settings action=%s", actions[claudeSettingsRel])
	}
}

func TestIntegrateMergesExistingSettings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(settingsDir, 0o750); err != nil {
		t.Fatal(err)
	}
	existing := `{
  "permissions": {"allow": ["Bash"]},
  "hooks": {
    "PostToolUse": [{"matcher": "Edit", "hooks": [{"type": "command", "command": "echo hi"}]}]
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, claudeSettingsRel), []byte(existing), 0o640); err != nil {
		t.Fatal(err)
	}

	res, err := Integrate(IntegrateOptions{Harness: "claude-code", Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	var settingsAction IntegrateFileAction
	for _, f := range res.Files {
		if f.Path == claudeSettingsRel {
			settingsAction = f.Action
		}
	}
	if settingsAction != IntegrateMerge {
		t.Fatalf("settings action=%s", settingsAction)
	}

	raw, err := os.ReadFile(filepath.Join(dir, claudeSettingsRel))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["permissions"]; !ok {
		t.Fatalf("lost permissions: %s", raw)
	}
	hooks := doc["hooks"].(map[string]interface{})
	if _, ok := hooks["PostToolUse"]; !ok {
		t.Fatalf("lost PostToolUse: %s", raw)
	}
	stop := hooks["Stop"].([]interface{})
	if len(stop) != 1 {
		t.Fatalf("stop=%v", stop)
	}
	if !strings.Contains(string(raw), specularHookMarker) {
		t.Fatalf("missing specular hook: %s", raw)
	}
}

func TestIntegrateUnsupportedHarness(t *testing.T) {
	t.Parallel()
	_, err := Integrate(IntegrateOptions{Harness: "aider", Root: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "unsupported harness") {
		t.Fatalf("err=%v", err)
	}
}

func TestIntegrateRequiresRoot(t *testing.T) {
	t.Parallel()
	_, err := Integrate(IntegrateOptions{Harness: "claude-code"})
	if err == nil || !strings.Contains(err.Error(), "root is required") {
		t.Fatalf("err=%v", err)
	}
}

func TestSessionProvenanceEnv(t *testing.T) {
	t.Parallel()
	env := sessionProvenanceEnv(&Record{ID: "auth", Harness: "claude-code"})
	joined := strings.Join(env, " ")
	if !strings.Contains(joined, "SPECULAR_SESSION_ID=auth") {
		t.Fatalf("%v", env)
	}
	if !strings.Contains(joined, "SPECULAR_SESSION_HARNESS=claude-code") {
		t.Fatalf("%v", env)
	}
	if sessionProvenanceEnv(nil) != nil {
		t.Fatal("nil record")
	}
}

func TestMergeClaudeStopHookIdempotent(t *testing.T) {
	t.Parallel()
	doc := map[string]interface{}{}
	merged, changed, err := mergeClaudeStopHook(doc)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	_, changed2, err := mergeClaudeStopHook(merged)
	if err != nil || changed2 {
		t.Fatalf("second merge should be noop: changed=%v err=%v", changed2, err)
	}
}

func TestIntegrateDryRunCursor(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res, err := Integrate(IntegrateOptions{
		Harness: "cursor",
		Root:    dir,
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun || res.Harness != "cursor" {
		t.Fatalf("%+v", res)
	}
	if len(res.Files) != 2 {
		t.Fatalf("files=%d", len(res.Files))
	}
	for _, f := range res.Files {
		if f.Action != IntegrateDryRun {
			t.Fatalf("action=%s path=%s", f.Action, f.Path)
		}
		if f.Content == "" {
			t.Fatalf("dry-run missing content for %s", f.Path)
		}
		if _, err := os.Stat(filepath.Join(dir, f.Path)); !os.IsNotExist(err) {
			t.Fatalf("dry-run must not write %s: %v", f.Path, err)
		}
	}
	hook := res.Files[0].Content
	if !strings.Contains(hook, "specular session attest") || !strings.Contains(hook, "specular gate") {
		t.Fatalf("hook missing attest/gate:\n%s", hook)
	}
	if !strings.Contains(hook, "SPECULAR_SESSION_ID") {
		t.Fatalf("hook missing session id env:\n%s", hook)
	}
	if !strings.Contains(hook, "cursor|cursor-agent") {
		t.Fatalf("hook missing cursor harness fallback:\n%s", hook)
	}
	var hooksJSON map[string]interface{}
	if err := json.Unmarshal([]byte(res.Files[1].Content), &hooksJSON); err != nil {
		t.Fatal(err)
	}
	if v, _ := hooksJSON["version"].(float64); v != 1 {
		t.Fatalf("version=%v", hooksJSON["version"])
	}
	hooks := hooksJSON["hooks"].(map[string]interface{})
	stop := hooks["stop"].([]interface{})
	if len(stop) != 1 {
		t.Fatalf("stop=%v", stop)
	}
	entry := stop[0].(map[string]interface{})
	cmd, _ := entry["command"].(string)
	if !strings.Contains(cmd, specularHookMarker) {
		t.Fatalf("command=%s", cmd)
	}
}

func TestIntegrateWritesAndIdempotentCursor(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	res, err := Integrate(IntegrateOptions{Harness: "cursor-agent", Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.Harness != "cursor" {
		t.Fatalf("canonical harness=%s", res.Harness)
	}
	hookPath := filepath.Join(dir, cursorHookRel)
	hooksJSONPath := filepath.Join(dir, cursorHooksJSONRel)
	if st, err := os.Stat(hookPath); err != nil || st.Mode()&0o100 == 0 {
		t.Fatalf("hook not executable: %v %#o", err, st.Mode())
	}
	raw, err := os.ReadFile(hooksJSONPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), specularHookMarker) {
		t.Fatalf("hooks.json missing hook: %s", raw)
	}
	if !strings.Contains(string(raw), `"version"`) {
		t.Fatalf("hooks.json missing version: %s", raw)
	}

	// Second run without --force skips hook; hooks.json already present → skip.
	res2, err := Integrate(IntegrateOptions{Harness: "cursor", Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]IntegrateFileAction{}
	for _, f := range res2.Files {
		actions[f.Path] = f.Action
	}
	if actions[cursorHookRel] != IntegrateSkip {
		t.Fatalf("hook action=%s", actions[cursorHookRel])
	}
	if actions[cursorHooksJSONRel] != IntegrateSkip {
		t.Fatalf("hooks.json action=%s", actions[cursorHooksJSONRel])
	}
}

func TestIntegrateMergesExistingCursorHooksJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".cursor")
	if err := os.MkdirAll(hooksDir, 0o750); err != nil {
		t.Fatal(err)
	}
	existing := `{
  "version": 1,
  "hooks": {
    "afterFileEdit": [{"command": ".cursor/hooks/format.sh"}]
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, cursorHooksJSONRel), []byte(existing), 0o640); err != nil {
		t.Fatal(err)
	}

	res, err := Integrate(IntegrateOptions{Harness: "cursor", Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	var hooksAction IntegrateFileAction
	for _, f := range res.Files {
		if f.Path == cursorHooksJSONRel {
			hooksAction = f.Action
		}
	}
	if hooksAction != IntegrateMerge {
		t.Fatalf("hooks.json action=%s", hooksAction)
	}

	raw, err := os.ReadFile(filepath.Join(dir, cursorHooksJSONRel))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	hooks := doc["hooks"].(map[string]interface{})
	if _, ok := hooks["afterFileEdit"]; !ok {
		t.Fatalf("lost afterFileEdit: %s", raw)
	}
	stop := hooks["stop"].([]interface{})
	if len(stop) != 1 {
		t.Fatalf("stop=%v", stop)
	}
	if !strings.Contains(string(raw), specularHookMarker) {
		t.Fatalf("missing specular hook: %s", raw)
	}
}

func TestMergeCursorStopHookIdempotent(t *testing.T) {
	t.Parallel()
	doc := map[string]interface{}{}
	merged, changed, err := mergeCursorStopHook(doc)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	_, changed2, err := mergeCursorStopHook(merged)
	if err != nil || changed2 {
		t.Fatalf("second merge should be noop: changed=%v err=%v", changed2, err)
	}
}

func TestIntegrateCodexWritesHookAndJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res, err := Integrate(IntegrateOptions{Harness: "codex-cli", Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.Harness != "codex" {
		t.Fatalf("harness=%s", res.Harness)
	}
	hookPath := filepath.Join(dir, codexHookRel)
	hooksJSONPath := filepath.Join(dir, codexHooksJSONRel)
	if _, err := os.Stat(hookPath); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(hooksJSONPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	hooks := doc["hooks"].(map[string]interface{})
	stop := hooks["Stop"].([]interface{})
	if len(stop) != 1 {
		t.Fatalf("Stop=%v", stop)
	}
	hookBody, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hookBody), "codex|codex-cli") {
		t.Fatalf("hook missing harness fallback:\n%s", hookBody)
	}
	if !strings.Contains(string(hookBody), "specular session attest") {
		t.Fatalf("hook missing attest:\n%s", hookBody)
	}

	res2, err := Integrate(IntegrateOptions{Harness: "codex", Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]IntegrateFileAction{}
	for _, f := range res2.Files {
		actions[f.Path] = f.Action
	}
	if actions[codexHookRel] != IntegrateSkip || actions[codexHooksJSONRel] != IntegrateSkip {
		t.Fatalf("actions=%v", actions)
	}
}

func TestIntegrateGeminiWritesHookAndSettings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res, err := Integrate(IntegrateOptions{Harness: "gemini-cli", Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.Harness != "gemini" {
		t.Fatalf("harness=%s", res.Harness)
	}
	hookPath := filepath.Join(dir, geminiHookRel)
	settingsPath := filepath.Join(dir, geminiSettingsRel)
	if _, err := os.Stat(hookPath); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	cfg := doc["hooksConfig"].(map[string]interface{})
	if enabled, _ := cfg["enabled"].(bool); !enabled {
		t.Fatalf("hooksConfig.enabled=%v", cfg["enabled"])
	}
	hooks := doc["hooks"].(map[string]interface{})
	end := hooks["SessionEnd"].([]interface{})
	if len(end) != 1 {
		t.Fatalf("SessionEnd=%v", end)
	}
	hookBody, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hookBody), "gemini|gemini-cli") {
		t.Fatalf("hook missing harness fallback:\n%s", hookBody)
	}
	if !strings.Contains(string(hookBody), `printf '%s\n' '{}'`) {
		t.Fatalf("gemini hook must emit JSON on stdout:\n%s", hookBody)
	}
}

func TestMergeCodexAndGeminiIdempotent(t *testing.T) {
	t.Parallel()
	codexDoc := map[string]interface{}{}
	merged, changed, err := mergeCodexStopHook(codexDoc)
	if err != nil || !changed {
		t.Fatalf("codex changed=%v err=%v", changed, err)
	}
	_, changed2, err := mergeCodexStopHook(merged)
	if err != nil || changed2 {
		t.Fatalf("codex second merge: changed=%v err=%v", changed2, err)
	}

	gemDoc := map[string]interface{}{}
	mergedG, changedG, err := mergeGeminiSessionEndHook(gemDoc)
	if err != nil || !changedG {
		t.Fatalf("gemini changed=%v err=%v", changedG, err)
	}
	_, changedG2, err := mergeGeminiSessionEndHook(mergedG)
	if err != nil || changedG2 {
		t.Fatalf("gemini second merge: changed=%v err=%v", changedG2, err)
	}
}

func TestIntegrateMergesExistingGeminiSettings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".gemini"), 0o750); err != nil {
		t.Fatal(err)
	}
	existing := `{
  "general": {"vimMode": true},
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "command": "echo hi"}]}]
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, geminiSettingsRel), []byte(existing), 0o640); err != nil {
		t.Fatal(err)
	}
	res, err := Integrate(IntegrateOptions{Harness: "gemini", Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	var action IntegrateFileAction
	for _, f := range res.Files {
		if f.Path == geminiSettingsRel {
			action = f.Action
		}
	}
	if action != IntegrateMerge {
		t.Fatalf("action=%s", action)
	}
	raw, err := os.ReadFile(filepath.Join(dir, geminiSettingsRel))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["general"]; !ok {
		t.Fatalf("lost general: %s", raw)
	}
	hooks := doc["hooks"].(map[string]interface{})
	if _, ok := hooks["SessionStart"]; !ok {
		t.Fatalf("lost SessionStart: %s", raw)
	}
	if _, ok := hooks["SessionEnd"]; !ok {
		t.Fatalf("missing SessionEnd: %s", raw)
	}
}
