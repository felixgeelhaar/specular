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
	for _, h := range []string{"claude-code", "claude", "Claude-Code"} {
		if !IsIntegrableHarness(h) {
			t.Fatalf("expected integrable: %s", h)
		}
	}
	for _, h := range []string{"codex", "gemini", "specular-auto", "aider", ""} {
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
	_, err := Integrate(IntegrateOptions{Harness: "codex", Root: t.TempDir()})
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
