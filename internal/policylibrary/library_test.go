package policylibrary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListAndGet(t *testing.T) {
	entries, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 3 {
		t.Fatalf("want ≥3 seeds, got %d", len(entries))
	}
	seen := map[string]bool{}
	for _, e := range entries {
		if seen[e.ID] {
			t.Fatalf("duplicate id %q", e.ID)
		}
		seen[e.ID] = true
		if err := Validate(e); err != nil {
			t.Fatalf("validate %s: %v", e.ID, err)
		}
		got, getErr := Get(e.ID)
		if getErr != nil {
			t.Fatalf("Get(%s): %v", e.ID, getErr)
		}
		if got.Framework == "" || got.Control == "" {
			t.Fatalf("%s missing framework/control", e.ID)
		}
		if len(got.Raw) == 0 {
			t.Fatalf("%s: empty Raw", e.ID)
		}
	}
	if !seen["soc2-cc8.1"] {
		t.Fatal("missing soc2-cc8.1 seed")
	}
	if !seen["pci-dss-6.4.5"] {
		t.Fatal("missing pci-dss-6.4.5 seed")
	}
}

func TestGetUnknown(t *testing.T) {
	_, err := Get("does-not-exist")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown id") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestInstallRefuseAndForce(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "soc2-cc8.1.yaml")
	if err := Install("soc2-cc8.1", dest, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "CC8.1") {
		t.Fatalf("installed content missing control: %s", data)
	}
	if err := Install("soc2-cc8.1", dest, false); err == nil {
		t.Fatal("expected refuse without --force")
	}
	if err := Install("soc2-cc8.1", dest, true); err != nil {
		t.Fatalf("force overwrite: %v", err)
	}
}

func TestDefaultInstallPath(t *testing.T) {
	got := DefaultInstallPath("/proj", "soc2-cc8.1")
	want := filepath.Join("/proj", ".specular", "policies", "soc2-cc8.1.yaml")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestValidateRejectsEmptyArtifacts(t *testing.T) {
	err := Validate(Entry{ID: "x", Framework: "f", Control: "c", Title: "t"})
	if err == nil {
		t.Fatal("expected error")
	}
}
