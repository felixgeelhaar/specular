package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/specular/internal/detect"
)

// formatBrownfieldBoard renders the PRODUCT_INTENT §24 brownfield init board:
// Detected / Existing controls / Recommended Specular baseline.
func formatBrownfieldBoard(root string, ctx *detect.Context) string {
	if root == "" {
		root = "."
	}
	var b strings.Builder
	b.WriteString("Detected\n")
	writeDetectedLines(&b, ctx)
	b.WriteString("Existing controls\n")
	writeExistingControls(&b, root, ctx)
	b.WriteString("Recommended Specular baseline\n")
	b.WriteString(formatRecommendedBaseline(root))
	return b.String()
}

func writeDetectedLines(b *strings.Builder, ctx *detect.Context) {
	if ctx == nil {
		b.WriteString("  ○ (no detection — use --no-detect was set or detection failed)\n")
		return
	}
	n := 0
	if ctx.Git.Initialized {
		line := "  ✓ Git"
		if ctx.Git.Branch != "" {
			line += " (" + ctx.Git.Branch + ")"
		}
		b.WriteString(line + "\n")
		n++
	}
	if ctx.CI.Detected && ctx.CI.Name != "" {
		fmt.Fprintf(b, "  ✓ CI: %s\n", displayCIName(ctx.CI.Name))
		n++
	}
	if ctx.Docker.Available {
		fmt.Fprintf(b, "  ✓ Docker%s\n", versionSuffix(ctx.Docker.Version))
		n++
	} else if ctx.Podman.Available {
		fmt.Fprintf(b, "  ✓ Podman%s\n", versionSuffix(ctx.Podman.Version))
		n++
	}
	for _, lang := range ctx.Languages {
		fmt.Fprintf(b, "  ✓ %s\n", lang)
		n++
	}
	for _, fw := range ctx.Frameworks {
		fmt.Fprintf(b, "  ✓ %s\n", fw)
		n++
	}
	if ctx.Providers != nil {
		for _, name := range []string{"claude-code", "codex-cli", "gemini-cli", "ollama", "anthropic", "openai", "gemini"} {
			info, ok := ctx.Providers[name]
			if !ok || !info.Available {
				continue
			}
			fmt.Fprintf(b, "  ✓ %s\n", displayProvider(name))
			n++
		}
	}
	if n == 0 {
		b.WriteString("  ○ nothing detected yet — Specular will use safe defaults\n")
	}
}

func writeExistingControls(b *strings.Builder, root string, ctx *detect.Context) {
	n := 0
	if pathExists(filepath.Join(root, "CODEOWNERS")) ||
		pathExists(filepath.Join(root, ".github", "CODEOWNERS")) ||
		pathExists(filepath.Join(root, "docs", "CODEOWNERS")) {
		b.WriteString("  ✓ CODEOWNERS\n")
		n++
	}
	if hasGitHubWorkflows(root) {
		b.WriteString("  ✓ GitHub Actions workflows\n")
		n++
	}
	if pathExists(filepath.Join(root, ".github", "dependabot.yml")) ||
		pathExists(filepath.Join(root, ".github", "dependabot.yaml")) {
		b.WriteString("  ✓ Dependabot\n")
		n++
	}
	if hasGolangCI(root) {
		b.WriteString("  ✓ golangci-lint\n")
		n++
	}
	if ctx != nil && ctx.CI.Detected {
		b.WriteString("  ✓ CI checks\n")
		n++
	}
	if n == 0 {
		b.WriteString("  ○ no existing control files detected\n")
	}
}

func formatRecommendedBaseline(root string) string {
	approvals := "advisory"
	if pathExists(filepath.Join(root, "CODEOWNERS")) ||
		pathExists(filepath.Join(root, ".github", "CODEOWNERS")) ||
		pathExists(filepath.Join(root, "docs", "CODEOWNERS")) {
		approvals = "existing CODEOWNERS"
	}
	return strings.Join([]string{
		"  provenance      advisory",
		"  intent drift    advisory",
		"  scope drift     advisory",
		"  dependencies    blocking",
		"  secrets         blocking",
		"  approvals       " + approvals,
	}, "\n") + "\n"
}

func displayCIName(name string) string {
	switch strings.ToLower(name) {
	case "github", "github-actions", "github_actions":
		return "GitHub Actions"
	case "gitlab", "gitlab-ci":
		return "GitLab CI"
	default:
		return name
	}
}

func displayProvider(name string) string {
	switch name {
	case "claude-code":
		return "Claude Code"
	case "codex-cli":
		return "Codex CLI"
	case "gemini-cli":
		return "Gemini CLI"
	default:
		return name
	}
}

func versionSuffix(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return ": " + v
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasGitHubWorkflows(root string) bool {
	dir := filepath.Join(root, ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			return true
		}
	}
	return false
}

func hasGolangCI(root string) bool {
	candidates := []string{
		".golangci.yml",
		".golangci.yaml",
		".golangci.toml",
		"golangci.yml",
		"golangci.yaml",
	}
	for _, c := range candidates {
		if pathExists(filepath.Join(root, c)) {
			return true
		}
	}
	return false
}
