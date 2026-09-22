package provenance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/attestation"
)

func TestValidateOK(t *testing.T) {
	t.Parallel()
	doc := &Document{Schema: Schema, Version: Version, Session: "auth", Harness: "claude-code"}
	res := Validate(doc)
	if !res.OK || len(res.Errors) != 0 {
		t.Fatalf("%+v", res)
	}
}

func TestValidateFails(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		doc  *Document
		want string
	}{
		{"nil", nil, "nil"},
		{"bad schema", &Document{Schema: "other", Version: Version, Session: "s"}, "schema"},
		{"empty session", &Document{Schema: Schema, Version: Version}, "session"},
		{"bad version", &Document{Schema: Schema, Version: "9", Session: "s"}, "version"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := Validate(tc.doc)
			if res.OK {
				t.Fatal("expected fail")
			}
			joined := strings.Join(res.Errors, " ")
			if !strings.Contains(joined, tc.want) {
				t.Fatalf("errors=%v want substring %q", res.Errors, tc.want)
			}
		})
	}
}

func TestWriteBesideAndResolvePrefer(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	attPath := filepath.Join(dir, "auth.attestation.json")
	attJSON := `{
		"version":"1.0","workflowId":"session-auth",
		"provenance":{"hostname":"h","platform":"linux","arch":"amd64",
			"specularVersion":"1.0.0","profile":"ci","harness":"claude-code","models":[]},
		"planHash":"","outputHash":"",
		"signedAt":"2026-01-02T00:00:00Z","signedBy":"t","signature":"x","publicKey":"y"
	}`
	if err := os.WriteFile(attPath, []byte(attJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	att, err := attestation.FromJSON([]byte(attJSON))
	if err != nil {
		t.Fatal(err)
	}
	doc := FromAttestation("auth", att)
	out, err := WriteBesideAttestation(attPath, doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
	loaded, err := Resolve(root, "auth")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(loaded.Source, ".attestation.json") {
		t.Fatalf("source=%s", loaded.Source)
	}
	res := Validate(loaded)
	if !res.OK {
		t.Fatalf("%+v", res)
	}
	bound := ValidateBound(loaded, root)
	if !bound.OK {
		t.Fatalf("bound=%+v", bound)
	}
	if bound.Bound != "sibling" {
		t.Fatalf("Bound=%q", bound.Bound)
	}
	human := FormatVerifyHuman(bound)
	if !strings.Contains(human, "Bound        sibling attestation") {
		t.Fatalf("human:\n%s", human)
	}
}

func TestValidateBoundHarnessMismatch(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	attJSON := `{
		"version":"1.0","workflowId":"session-auth",
		"provenance":{"hostname":"h","platform":"linux","arch":"amd64",
			"specularVersion":"1.0.0","profile":"ci","harness":"claude-code","governed":true,"models":[]},
		"planHash":"","outputHash":"",
		"signedAt":"2026-01-02T00:00:00Z","signedBy":"t","signature":"x","publicKey":"y"
	}`
	if err := os.WriteFile(filepath.Join(dir, "auth.attestation.json"), []byte(attJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	prov := `{
		"schema":"specular.provenance/v1","version":"1","session":"auth",
		"harness":"codex","governed":false,
		"source":".specular/sessions/auth.attestation.json"
	}`
	if err := os.WriteFile(filepath.Join(dir, "auth.provenance.json"), []byte(prov), 0o600); err != nil {
		t.Fatal(err)
	}
	doc, err := Resolve(root, "auth")
	if err != nil {
		t.Fatal(err)
	}
	res := ValidateBound(doc, root)
	if res.OK {
		t.Fatal("expected bind failure")
	}
	joined := strings.Join(res.Errors, " ")
	if !strings.Contains(joined, "harness") || !strings.Contains(joined, "governed") {
		t.Fatalf("errors=%v", res.Errors)
	}
}

func TestValidateBoundMissingSibling(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	prov := `{"schema":"specular.provenance/v1","version":"1","session":"orphan","harness":"x"}`
	if err := os.WriteFile(filepath.Join(dir, "orphan.provenance.json"), []byte(prov), 0o600); err != nil {
		t.Fatal(err)
	}
	doc, err := LoadDocumentFile(filepath.Join(dir, "orphan.provenance.json"))
	if err != nil {
		t.Fatal(err)
	}
	res := ValidateBound(doc, root)
	if res.OK || !strings.Contains(strings.Join(res.Errors, " "), "sibling attestation missing") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveFallsBackToAttestation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	att := `{
		"version":"1.0","workflowId":"session-x",
		"provenance":{"hostname":"h","platform":"linux","arch":"amd64",
			"specularVersion":"1.0.0","profile":"ci","harness":"codex","models":[]},
		"planHash":"","outputHash":"",
		"signedAt":"2026-01-02T00:00:00Z","signedBy":"t","signature":"x","publicKey":"y"
	}`
	if err := os.WriteFile(filepath.Join(dir, "x.attestation.json"), []byte(att), 0o600); err != nil {
		t.Fatal(err)
	}
	doc, err := Resolve(root, "x")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Harness != "codex" || !strings.Contains(doc.Source, ".attestation.json") {
		t.Fatalf("%+v", doc)
	}
}
