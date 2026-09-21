package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/attestation"
)

func TestFromProvenanceMapsFields(t *testing.T) {
	t.Parallel()
	p := attestation.Provenance{
		Harness:        "claude-code",
		Governed:       true,
		WorktreePath:   "/tmp/wt/auth",
		WorktreeBranch: "specular/auth",
		WorktreeName:   "auth",
		GitRepo:        "https://github.com/example/repo.git",
		GitCommit:      "abcdef0123456789deadbeef",
		GitBranch:      "main",
		GitDirty:       true,
	}
	doc := FromProvenance("auth", p)
	if doc.Schema != Schema || doc.Version != Version {
		t.Fatalf("schema=%s version=%s", doc.Schema, doc.Version)
	}
	if doc.Session != "auth" || doc.Harness != "claude-code" || !doc.Governed {
		t.Fatalf("%+v", doc)
	}
	if doc.Worktree == nil || doc.Worktree.Path != "/tmp/wt/auth" {
		t.Fatalf("worktree=%+v", doc.Worktree)
	}
	if doc.Git == nil || doc.Git.Commit != "abcdef0123456789deadbeef" || !doc.Git.Dirty {
		t.Fatalf("git=%+v", doc.Git)
	}
	raw, err := doc.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var round Document
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatal(err)
	}
	if round.Schema != Schema || round.Harness != "claude-code" {
		t.Fatalf("%+v", round)
	}
}

func TestFromProvenanceOmitsEmptyWorktreeAndGit(t *testing.T) {
	t.Parallel()
	doc := FromProvenance("x", attestation.Provenance{Harness: "codex-cli"})
	if doc.Worktree != nil || doc.Git != nil {
		t.Fatalf("expected omit empty nested objects: %+v", doc)
	}
	if doc.Governed {
		t.Fatal("governed should be false/omit-friendly")
	}
}

func TestLoadSessionAndLatest(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}

	older := `{
		"version":"1.0",
		"workflowId":"session-old",
		"provenance":{
			"hostname":"h","platform":"linux","arch":"amd64",
			"specularVersion":"1.0.0","profile":"ci",
			"harness":"old-harness","models":[]
		},
		"planHash":"","outputHash":"",
		"signedAt":"2026-01-01T00:00:00Z","signedBy":"t","signature":"x","publicKey":"y"
	}`
	newer := `{
		"version":"1.0",
		"workflowId":"session-auth",
		"provenance":{
			"hostname":"h","platform":"linux","arch":"amd64",
			"specularVersion":"1.0.0","profile":"ci",
			"harness":"claude-code","governed":true,
			"worktreePath":"/tmp/wt/auth","worktreeBranch":"specular/auth","worktreeName":"auth",
			"gitCommit":"abc123def456","gitBranch":"specular/auth",
			"models":[]
		},
		"planHash":"","outputHash":"",
		"signedAt":"2026-01-02T00:00:00Z","signedBy":"t","signature":"x","publicKey":"y"
	}`
	oldPath := filepath.Join(dir, "old.attestation.json")
	newPath := filepath.Join(dir, "auth.attestation.json")
	if err := os.WriteFile(oldPath, []byte(older), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte(newer), 0o600); err != nil {
		t.Fatal(err)
	}
	// Ensure auth is newer by mtime.
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(oldPath, past, past); err != nil {
		t.Fatal(err)
	}

	doc, err := LoadSession(root, "auth")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Session != "auth" || doc.Harness != "claude-code" || !doc.Governed {
		t.Fatalf("%+v", doc)
	}
	if doc.Source != ".specular/sessions/auth.attestation.json" {
		t.Fatalf("source=%s", doc.Source)
	}

	latest, err := LoadLatest(root)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Session != "auth" {
		t.Fatalf("latest session=%s", latest.Session)
	}
}

func TestLoadLatestEmpty(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := LoadLatest(root); err == nil {
		t.Fatal("expected error")
	}
	if err := os.MkdirAll(SessionsPath(root), 0o750); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLatest(root); err == nil {
		t.Fatal("expected empty-dir error")
	}
}

func TestFormatHuman(t *testing.T) {
	t.Parallel()
	doc := &Document{
		Schema:   Schema,
		Version:  Version,
		Session:  "auth",
		Harness:  "claude-code",
		Governed: true,
		Worktree: &Worktree{Path: "/tmp/wt/auth", Branch: "specular/auth", Name: "auth"},
		Git:      &Git{Commit: "abcdef0123456789", Branch: "specular/auth", Dirty: true},
		Source:   ".specular/sessions/auth.attestation.json",
	}
	out := FormatHuman(doc)
	for _, want := range []string{
		"Agent Provenance (" + Schema + ")",
		"Session      auth",
		"Harness      claude-code",
		"Governed     true",
		"Worktree     /tmp/wt/auth (specular/auth) [auth]",
		"Git          abcdef012345 on specular/auth (dirty)",
		"Source       .specular/sessions/auth.attestation.json",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestNewProtocolRef(t *testing.T) {
	t.Parallel()
	ref := NewProtocolRef([]string{"b", "a"})
	if ref.Schema != Schema || ref.Version != Version {
		t.Fatalf("%+v", ref)
	}
	if len(ref.Sessions) != 2 || ref.Sessions[0] != "b" {
		t.Fatalf("%v", ref.Sessions)
	}
	// Mutation of input must not affect ref.
	in := []string{"x"}
	ref2 := NewProtocolRef(in)
	in[0] = "mutated"
	if ref2.Sessions[0] != "x" {
		t.Fatal("expected defensive copy")
	}
}

func TestFromAttestationNil(t *testing.T) {
	t.Parallel()
	doc := FromAttestation("s", nil)
	if doc.Schema != Schema || doc.Session != "s" {
		t.Fatalf("%+v", doc)
	}
}
