package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPolicyPackRegistered(t *testing.T) {
	var packCmdFound bool
	for _, c := range policyCmd.Commands() {
		if c.Name() != "pack" {
			continue
		}
		packCmdFound = true
		subs := map[string]bool{"list": false, "show": false, "apply": false, "check": false}
		for _, s := range c.Commands() {
			if _, ok := subs[s.Name()]; ok {
				subs[s.Name()] = true
			}
		}
		for name, ok := range subs {
			if !ok {
				t.Errorf("policy pack %s missing", name)
			}
		}
	}
	if !packCmdFound {
		t.Fatal("policy pack not registered")
	}
}

func TestRunPolicyPackList(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := runPolicyPackList(policyPackListCmd, nil)
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if runErr != nil {
		t.Fatal(runErr)
	}
	out := buf.String()
	if !strings.Contains(out, "soc2-cc8.1") {
		t.Fatalf("list missing soc2-cc8.1: %s", out)
	}
	if !strings.Contains(out, "TITLE") || !strings.Contains(out, "SUMMARY") {
		t.Fatalf("list missing title/summary columns: %s", out)
	}
	if !strings.Contains(out, "Change Management") {
		t.Fatalf("list missing title text: %s", out)
	}
}

func TestRunPolicyPackShowHumanAndJSON(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := runPolicyPackShow(policyPackShowCmd, []string{"soc2-cc8.1"})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if runErr != nil {
		t.Fatal(runErr)
	}
	out := buf.String()
	if !strings.Contains(out, "Change Management") {
		t.Fatalf("human show missing title: %s", out)
	}
	if !strings.Contains(out, "Artifacts:") {
		t.Fatalf("human show missing artifacts: %s", out)
	}

	_ = policyPackShowCmd.Flags().Set("json", "true")
	defer func() { _ = policyPackShowCmd.Flags().Set("json", "false") }()

	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w2
	runErr = runPolicyPackShow(policyPackShowCmd, []string{"soc2-cc8.1"})
	_ = w2.Close()
	os.Stdout = old
	buf.Reset()
	_, _ = io.Copy(&buf, r2)
	if runErr != nil {
		t.Fatal(runErr)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, buf.String())
	}
	if parsed["id"] != "soc2-cc8.1" {
		t.Fatalf("json id: %v", parsed["id"])
	}
	if parsed["title"] != "Change Management" {
		t.Fatalf("json title: %v", parsed["title"])
	}
	sum, _ := parsed["summary"].(string)
	if sum == "" {
		t.Fatal("json missing summary")
	}
}

func TestRunPolicyPackApplyDryRun(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	_ = policyPackApplyCmd.Flags().Set("dry-run", "true")
	defer func() { _ = policyPackApplyCmd.Flags().Set("dry-run", "false") }()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := runPolicyPackApply(policyPackApplyCmd, []string{"soc2-cc8.1"})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if runErr != nil {
		t.Fatal(runErr)
	}
	out := buf.String()
	if !strings.Contains(out, "Dry-run") {
		t.Fatalf("expected dry-run message: %s", out)
	}
	dest := filepath.Join(dir, ".specular", "policies", "soc2-cc8.1.yaml")
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not write file: %v", err)
	}

	_ = policyPackApplyCmd.Flags().Set("dry-run", "false")
	if err := runPolicyPackApply(policyPackApplyCmd, []string{"soc2-cc8.1"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("missing apply output: %v", err)
	}
}

func TestRunPolicyPackCheckMissingAndPresent(t *testing.T) {
	root := t.TempDir()
	_ = policyPackCheckCmd.Flags().Set("project-root", root)
	defer func() { _ = policyPackCheckCmd.Flags().Set("project-root", "") }()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := runPolicyPackCheck(policyPackCheckCmd, []string{"soc2-cc8.1"})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if runErr == nil {
		t.Fatal("expected check failure when artifacts missing")
	}
	if !strings.Contains(buf.String(), "FAIL") {
		t.Fatalf("%s", buf.String())
	}

	sess := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(sess, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sess, "a.attestation.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "drift.sarif"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".specular", "approvals"), 0o750); err != nil {
		t.Fatal(err)
	}

	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w2
	runErr = runPolicyPackCheck(policyPackCheckCmd, []string{"soc2-cc8.1"})
	_ = w2.Close()
	os.Stdout = old
	buf.Reset()
	_, _ = io.Copy(&buf, r2)
	if runErr != nil {
		t.Fatalf("expected pass: %v\n%s", runErr, buf.String())
	}
	if !strings.Contains(buf.String(), "PASS") {
		t.Fatalf("%s", buf.String())
	}
}
