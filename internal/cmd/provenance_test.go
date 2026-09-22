package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/provenance"
)

func TestProvenanceCommandsRegistered(t *testing.T) {
	// Not parallel: rootCmd.Find mutates shared cobra command state.
	for _, path := range [][]string{
		{"provenance"},
		{"provenance", "show"},
		{"provenance", "verify"},
	} {
		cmd, _, err := rootCmd.Find(path)
		if err != nil {
			t.Fatalf("%v: %v", path, err)
		}
		if cmd == nil {
			t.Fatalf("missing %v", path)
		}
	}
	show, _, err := rootCmd.Find([]string{"provenance", "show"})
	if err != nil {
		t.Fatal(err)
	}
	if show.Flags().Lookup("json") == nil {
		t.Fatal("missing --json on show")
	}
	verify, _, err := rootCmd.Find([]string{"provenance", "verify"})
	if err != nil {
		t.Fatal(err)
	}
	if verify.Flags().Lookup("json") == nil {
		t.Fatal("missing --json on verify")
	}
	if provenanceCmd.PersistentFlags().Lookup("project-root") == nil {
		t.Fatal("missing --project-root")
	}
}

func TestRunProvenanceShowHumanAndJSON(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	att := `{
		"version":"1.0",
		"workflowId":"session-auth",
		"provenance":{
			"hostname":"h","platform":"linux","arch":"amd64",
			"specularVersion":"1.0.0","profile":"ci",
			"harness":"claude-code","governed":true,
			"worktreePath":"/tmp/wt/auth","worktreeBranch":"specular/auth",
			"gitCommit":"abc123","gitBranch":"specular/auth",
			"models":[]
		},
		"planHash":"","outputHash":"",
		"signedAt":"2026-01-02T00:00:00Z","signedBy":"t","signature":"x","publicKey":"y"
	}`
	if err := os.WriteFile(filepath.Join(dir, "auth.attestation.json"), []byte(att), 0o600); err != nil {
		t.Fatal(err)
	}

	_ = provenanceCmd.PersistentFlags().Set("project-root", root)
	defer func() { _ = provenanceCmd.PersistentFlags().Set("project-root", "") }()
	_ = provenanceShowCmd.Flags().Set("json", "false")

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := runProvenanceShow(provenanceShowCmd, nil)
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if runErr != nil {
		t.Fatal(runErr)
	}
	human := buf.String()
	if !strings.Contains(human, "Agent Provenance ("+provenance.Schema+")") {
		t.Fatalf("human:\n%s", human)
	}
	if !strings.Contains(human, "Harness      claude-code") {
		t.Fatalf("human:\n%s", human)
	}

	_ = provenanceShowCmd.Flags().Set("json", "true")
	defer func() { _ = provenanceShowCmd.Flags().Set("json", "false") }()

	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w2
	runErr = runProvenanceShow(provenanceShowCmd, []string{"auth"})
	_ = w2.Close()
	os.Stdout = old
	buf.Reset()
	_, _ = io.Copy(&buf, r2)
	if runErr != nil {
		t.Fatal(runErr)
	}
	var doc provenance.Document
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("json: %v\n%s", err, buf.String())
	}
	if doc.Schema != provenance.Schema || doc.Session != "auth" || doc.Harness != "claude-code" {
		t.Fatalf("%+v", doc)
	}
	if !doc.Governed || doc.Worktree == nil || doc.Git == nil {
		t.Fatalf("%+v", doc)
	}
}

func TestRunProvenanceVerifyOKAndFail(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	good := `{
		"schema":"specular.provenance/v1","version":"1","session":"auth","harness":"claude-code"
	}`
	if err := os.WriteFile(filepath.Join(dir, "auth.provenance.json"), []byte(good), 0o600); err != nil {
		t.Fatal(err)
	}
	// Newest attestation so Resolve("") finds auth
	att := `{
		"version":"1.0","workflowId":"session-auth",
		"provenance":{"hostname":"h","platform":"linux","arch":"amd64",
			"specularVersion":"1.0.0","profile":"ci","harness":"claude-code","models":[]},
		"planHash":"","outputHash":"",
		"signedAt":"2026-01-02T00:00:00Z","signedBy":"t","signature":"x","publicKey":"y"
	}`
	if err := os.WriteFile(filepath.Join(dir, "auth.attestation.json"), []byte(att), 0o600); err != nil {
		t.Fatal(err)
	}

	_ = provenanceCmd.PersistentFlags().Set("project-root", root)
	defer func() { _ = provenanceCmd.PersistentFlags().Set("project-root", "") }()
	_ = provenanceVerifyCmd.Flags().Set("json", "false")

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := runProvenanceVerify(provenanceVerifyCmd, []string{"auth"})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if runErr != nil {
		t.Fatal(runErr)
	}
	if !strings.Contains(buf.String(), "VERIFY: OK") {
		t.Fatalf("%s", buf.String())
	}

	bad := `{"schema":"nope","version":"1","session":""}`
	if err := os.WriteFile(filepath.Join(dir, "bad.provenance.json"), []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runProvenanceVerify(provenanceVerifyCmd, []string{filepath.Join(dir, "bad.provenance.json")}); err == nil {
		t.Fatal("expected verify failure")
	}
}

func TestRunProvenanceShowMissing(t *testing.T) {
	root := t.TempDir()
	_ = provenanceCmd.PersistentFlags().Set("project-root", root)
	defer func() { _ = provenanceCmd.PersistentFlags().Set("project-root", "") }()
	if err := runProvenanceShow(provenanceShowCmd, nil); err == nil {
		t.Fatal("expected error")
	}
}
