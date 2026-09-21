package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestParseManifestDependsOn(t *testing.T) {
	t.Parallel()
	data := []byte(`
- name: impl
  goal: implement
- name: review
  goal: review
  dependsOn: [impl]
`)
	entries, err := ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries[1].DependsOn) != 1 || entries[1].DependsOn[0] != "impl" {
		t.Fatalf("%+v", entries[1])
	}
}

func TestParseManifestRejectsUnknownAndCycleDeps(t *testing.T) {
	t.Parallel()
	if _, err := ParseManifest([]byte(`[{"name":"a","goal":"1","dependsOn":["missing"]}]`)); err == nil {
		t.Fatal("expected unknown dep error")
	}
	if _, err := ParseManifest([]byte(`[
		{"name":"a","goal":"1","dependsOn":["b"]},
		{"name":"b","goal":"2","dependsOn":["a"]}
	]`)); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
	if _, err := ParseManifest([]byte(`[{"name":"a","goal":"1","dependsOn":["a"]}]`)); err == nil {
		t.Fatal("expected self-dep error")
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

func TestStartManyDependencyChain(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	ok := writeExitStub(t, 0)
	entries := []ManifestEntry{
		{Name: "impl", Goal: "implement", Harness: "specular-auto"},
		{Name: "review", Goal: "review", Harness: "specular-auto", DependsOn: []string{"impl"}},
	}
	started, err := mgr.StartMany(context.Background(), entries, StartOptions{
		Binary: ok, Detach: true, NoApproval: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(started) != 2 {
		t.Fatalf("started=%d", len(started))
	}
	impl, _ := mgr.Get("impl")
	if impl.Status != StatusCompleted {
		t.Fatalf("impl status=%s", impl.Status)
	}
	_, _ = mgr.Wait(context.Background(), []string{"review"}, WaitOptions{Timeout: 5 * time.Second})
	review, _ := mgr.Get("review")
	if review.Status != StatusCompleted {
		t.Fatalf("review status=%s", review.Status)
	}
}

func TestStartManyMultiParentJoin(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	ok := writeExitStub(t, 0)
	entries := []ManifestEntry{
		{Name: "a", Goal: "a", Harness: "specular-auto"},
		{Name: "b", Goal: "b", Harness: "specular-auto"},
		{Name: "join", Goal: "join", Harness: "specular-auto", DependsOn: []string{"a", "b"}},
	}
	started, err := mgr.StartMany(context.Background(), entries, StartOptions{
		Binary: ok, Detach: true, NoApproval: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(started) != 3 {
		t.Fatalf("started=%d", len(started))
	}
	_, _ = mgr.Wait(context.Background(), []string{"join"}, WaitOptions{Timeout: 5 * time.Second})
	join, _ := mgr.Get("join")
	if join.Status != StatusCompleted {
		t.Fatalf("join=%s", join.Status)
	}
}

func TestStartManyAbortsOnParentFailure(t *testing.T) {
	repo := initTempRepo(t)
	mgr, err := NewManager(repo)
	if err != nil {
		t.Fatal(err)
	}
	fail := writeExitStub(t, 1)
	entries := []ManifestEntry{
		{Name: "bad", Goal: "fail", Harness: "specular-auto"},
		{Name: "child", Goal: "child", Harness: "specular-auto", DependsOn: []string{"bad"}},
	}
	started, err := mgr.StartMany(context.Background(), entries, StartOptions{
		Binary: fail, Detach: true, NoApproval: true,
	})
	if err == nil {
		t.Fatal("expected dependency wait error")
	}
	if len(started) != 1 || started[0].ID != "bad" {
		t.Fatalf("started=%v", started)
	}
	child, getErr := mgr.Get("child")
	if getErr != nil {
		t.Fatal(getErr)
	}
	if child.Status != StatusFailed && child.Status != StatusQueued {
		t.Fatalf("child status=%s want failed/queued", child.Status)
	}
}
