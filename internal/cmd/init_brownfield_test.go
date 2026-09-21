package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/detect"
)

func TestFormatBrownfieldBoard(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".github", "workflows", "ci.yml"), []byte("name: ci\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".github", "CODEOWNERS"), []byte("* @owners\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".golangci.yml"), []byte("run:\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx := &detect.Context{
		Languages: []string{"Go"},
		Git:       detect.GitContext{Initialized: true, Branch: "main"},
		CI:        detect.CIInfo{Detected: true, Name: "github"},
		Providers: map[string]detect.ProviderInfo{
			"claude-code": {Available: true},
		},
		Docker: detect.ContainerRuntime{Available: true, Version: "24.0"},
	}

	got := formatBrownfieldBoard(root, ctx)
	for _, want := range []string{
		"Detected",
		"✓ Git (main)",
		"✓ CI: GitHub Actions",
		"✓ Go",
		"✓ Claude Code",
		"✓ Docker: 24.0",
		"Existing controls",
		"✓ CODEOWNERS",
		"✓ GitHub Actions workflows",
		"✓ golangci-lint",
		"Recommended Specular baseline",
		"provenance      advisory",
		"secrets         blocking",
		"approvals       existing CODEOWNERS",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("board missing %q:\n%s", want, got)
		}
	}
}

func TestFormatRecommendedBaselineWithoutCODEOWNERS(t *testing.T) {
	t.Parallel()
	got := formatRecommendedBaseline(t.TempDir())
	if !strings.Contains(got, "approvals       advisory") {
		t.Fatalf("got %q", got)
	}
}

func TestFormatBrownfieldBoardEmpty(t *testing.T) {
	t.Parallel()
	got := formatBrownfieldBoard(t.TempDir(), &detect.Context{})
	for _, want := range []string{
		"Detected",
		"nothing detected yet",
		"Existing controls",
		"no existing control files",
		"Recommended Specular baseline",
		"approvals       advisory",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("board missing %q:\n%s", want, got)
		}
	}
}
