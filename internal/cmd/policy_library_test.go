package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPolicyLibraryRegistered(t *testing.T) {
	var libraryCmdFound bool
	for _, c := range policyCmd.Commands() {
		if c.Name() != "library" {
			continue
		}
		libraryCmdFound = true
		subs := map[string]bool{"list": false, "show": false, "install": false}
		for _, s := range c.Commands() {
			if _, ok := subs[s.Name()]; ok {
				subs[s.Name()] = true
			}
		}
		for name, ok := range subs {
			if !ok {
				t.Errorf("policy library %s missing", name)
			}
		}
	}
	if !libraryCmdFound {
		t.Fatal("policy library not registered")
	}
}

func TestPolicySubcommandsIncludesLibrary(t *testing.T) {
	found := false
	for _, c := range policyCmd.Commands() {
		if c.Name() == "library" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected library among policy subcommands")
	}
}

func TestRunPolicyLibraryList(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := runPolicyLibraryList(policyLibraryListCmd, nil)
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
}

func TestRunPolicyLibraryInstall(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	if err := runPolicyLibraryInstall(policyLibraryInstallCmd, []string{"soc2-cc8.1"}); err != nil {
		t.Fatalf("install: %v", err)
	}
	dest := filepath.Join(dir, ".specular", "policies", "soc2-cc8.1.yaml")
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("missing install: %v", err)
	}
	if err := runPolicyLibraryInstall(policyLibraryInstallCmd, []string{"soc2-cc8.1"}); err == nil {
		t.Fatal("expected refuse without --force")
	}
	_ = policyLibraryInstallCmd.Flags().Set("force", "true")
	defer func() { _ = policyLibraryInstallCmd.Flags().Set("force", "false") }()
	if err := runPolicyLibraryInstall(policyLibraryInstallCmd, []string{"soc2-cc8.1"}); err != nil {
		t.Fatalf("force: %v", err)
	}
}

func TestRunPolicyLibraryShow(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := runPolicyLibraryShow(policyLibraryShowCmd, []string{"eu-ai-act-art17"})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if runErr != nil {
		t.Fatal(runErr)
	}
	if !strings.Contains(buf.String(), "Article 17") {
		t.Fatalf("show missing Article 17: %s", buf.String())
	}
}
