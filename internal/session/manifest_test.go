package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifestYAMLArray(t *testing.T) {
	t.Parallel()
	data := []byte(`
- name: auth
  harness: claude-code
  goal: Harden JWT
- name: ratelimit
  harness: codex
  goal: Add rate limiting
`)
	entries, err := ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("len=%d", len(entries))
	}
	if entries[0].Name != "auth" || entries[0].Harness != "claude-code" {
		t.Fatalf("%+v", entries[0])
	}
}

func TestParseManifestWrappedJSON(t *testing.T) {
	t.Parallel()
	data := []byte(`{"sessions":[{"goal":"one"},{"name":"two","goal":"two goal","profile":"ci"}]}`)
	entries, err := ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[1].Name != "two" {
		t.Fatalf("%+v", entries)
	}
}

func TestParseManifestRejectsDuplicatesAndEmpty(t *testing.T) {
	t.Parallel()
	if _, err := ParseManifest([]byte(`[{"name":"a","goal":"1"},{"name":"a","goal":"2"}]`)); err == nil {
		t.Fatal("expected duplicate error")
	}
	if _, err := ParseManifest([]byte(`[{"name":"a"}]`)); err == nil {
		t.Fatal("expected missing goal error")
	}
	if _, err := ParseManifest([]byte(`[]`)); err == nil {
		t.Fatal("expected empty error")
	}
}

func TestStartMany(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	stub := writeExitStub(t, 0)
	entries := []ManifestEntry{
		{Name: "fleet-a", Goal: "goal a", Harness: "specular-auto"},
		{Name: "fleet-b", Goal: "goal b", Harness: "specular-auto"},
	}
	started, err := mgr.StartMany(context.Background(), entries, StartOptions{
		Binary:     stub,
		Detach:     true,
		NoApproval: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(started) != 2 {
		t.Fatalf("started=%d", len(started))
	}
	for _, rec := range started {
		if _, err := mgr.Get(rec.ID); err != nil {
			t.Fatalf("get %s: %v", rec.ID, err)
		}
		if rec.WorktreePath == "" {
			t.Fatalf("%s missing worktree", rec.ID)
		}
		if _, err := os.Stat(filepath.Join(mgr.Store().Dir(), rec.ID+".json")); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = mgr.Wait(context.Background(), []string{"fleet-a", "fleet-b"}, WaitOptions{})
}
