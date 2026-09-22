package policylibrary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckMissingArtifacts(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	rep, err := Check(root, "soc2-cc8.1")
	if err != nil {
		t.Fatal(err)
	}
	if rep.OK || rep.Missing == 0 {
		t.Fatalf("expected missing artifacts: %+v", rep)
	}
	if rep.ID != "soc2-cc8.1" || rep.Framework == "" {
		t.Fatalf("%+v", rep)
	}
	text := FormatCheckHuman(rep)
	if !strings.Contains(text, "FAIL") || !strings.Contains(text, "does not certify") {
		t.Fatalf("%s", text)
	}
}

func TestCheckPresentArtifacts(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sess := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(sess, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sess, "auth.attestation.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "drift.sarif"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".specular", "approvals"), 0o750); err != nil {
		t.Fatal(err)
	}
	rep, err := Check(root, "soc2-cc8.1")
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK || rep.Missing != 0 || rep.Present != 3 {
		t.Fatalf("%+v", rep)
	}
	text := FormatCheckHuman(rep)
	if !strings.Contains(text, "PASS") {
		t.Fatalf("%s", text)
	}
}

func TestCheckUnknownPack(t *testing.T) {
	t.Parallel()
	_, err := Check(t.TempDir(), "no-such-pack")
	if err == nil {
		t.Fatal("expected error")
	}
}
