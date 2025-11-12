package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/detect"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Detect and display environment setup",
	Long: `Detect environment setup including installed models, API keys, Docker, and other dependencies.

This command helps you understand what AI providers and tools are available in your environment.

Checks include:
  • Container runtime (Docker/Podman) availability
  • AI providers (Ollama, OpenAI, Anthropic, Google Gemini)
  • API key configuration
  • Programming languages and frameworks
  • Git repository status
  • CI/CD environment detection

Examples:
  # Display environment in default text format
  specular context

  # Output as JSON for scripting
  specular context --format json

  # Output as YAML
  specular context --format yaml
`,
	RunE: runContext,
}

func init() {
	rootCmd.AddCommand(contextCmd)
}

func runContext(cmd *cobra.Command, args []string) error {
	// Extract command context
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	// Detect all environment context
	ctx, err := detect.DetectAll()
	if err != nil {
		return ux.FormatError(err, "detecting environment context")
	}

	// Format and output the context
	return outputContext(cmdCtx, ctx)
}

func outputContext(cmdCtx *CommandContext, ctx *detect.Context) error {
	// For JSON and YAML, use the formatter
	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if err != nil {
			return err
		}
		return formatter.Format(ctx)
	}

	// For text format, use the built-in Summary method
	fmt.Println(ctx.Summary())

	// Add recommendations
	recommended := ctx.GetRecommendedProviders()
	if len(recommended) > 0 {
		fmt.Println("\nRecommended Providers:")
		for _, provider := range recommended {
			fmt.Printf("  • %s\n", provider)
		}
	}

	// Add warnings if no providers are available
	hasProvider := false
	for _, info := range ctx.Providers {
		if info.Available {
			hasProvider = true
			break
		}
	}

	if !hasProvider {
		fmt.Println("\n⚠️  Warning: No AI providers detected")
		fmt.Println("   Install Ollama (https://ollama.ai) or set API keys:")
		fmt.Println("     • OPENAI_API_KEY for OpenAI")
		fmt.Println("     • ANTHROPIC_API_KEY for Anthropic Claude")
		fmt.Println("     • GEMINI_API_KEY for Google Gemini")
	}

	// Add warning if no container runtime
	if ctx.Runtime == "" {
		fmt.Println("\n⚠️  Warning: No container runtime detected")
		fmt.Println("   Install Docker (https://docker.com) or Podman")
	}

	return nil
}

// ContextReport creates a simplified report structure for external use
type ContextReport struct {
	Runtime   RuntimeInfo              `json:"runtime"`
	Providers map[string]ProviderState `json:"providers"`
	Languages []string                 `json:"languages,omitempty"`
	Git       *GitInfo                 `json:"git,omitempty"`
	CI        *CIInfo                  `json:"ci,omitempty"`
}

type RuntimeInfo struct {
	Type      string `json:"type"`      // "docker", "podman", or ""
	Version   string `json:"version"`   // Runtime version
	Available bool   `json:"available"` // Is runtime available
	Running   bool   `json:"running"`   // Is daemon running
}

type ProviderState struct {
	Available bool   `json:"available"`        // Is provider available
	Type      string `json:"type"`             // "cli", "api", "local"
	Version   string `json:"version"`          // Provider version
	APIKey    bool   `json:"api_key_set"`      // Is API key set
	KeyName   string `json:"key_name"`         // Environment variable name
}

type GitInfo struct {
	Initialized bool   `json:"initialized"`
	Branch      string `json:"branch,omitempty"`
	Dirty       bool   `json:"dirty"`
	Uncommitted int    `json:"uncommitted"`
}

type CIInfo struct {
	Detected bool   `json:"detected"`
	Name     string `json:"name,omitempty"`
}

// convertToReport converts detect.Context to ContextReport
func convertToReport(ctx *detect.Context) *ContextReport {
	report := &ContextReport{
		Providers: make(map[string]ProviderState),
	}

	// Runtime info
	if ctx.Runtime == "docker" {
		report.Runtime = RuntimeInfo{
			Type:      "docker",
			Version:   ctx.Docker.Version,
			Available: ctx.Docker.Available,
			Running:   ctx.Docker.Running,
		}
	} else if ctx.Runtime == "podman" {
		report.Runtime = RuntimeInfo{
			Type:      "podman",
			Version:   ctx.Podman.Version,
			Available: ctx.Podman.Available,
			Running:   ctx.Podman.Running,
		}
	} else {
		report.Runtime = RuntimeInfo{
			Type:      "",
			Available: false,
			Running:   false,
		}
	}

	// Provider info
	for name, info := range ctx.Providers {
		report.Providers[name] = ProviderState{
			Available: info.Available,
			Type:      info.Type,
			Version:   info.Version,
			APIKey:    info.EnvSet,
			KeyName:   info.EnvVar,
		}
	}

	// Languages
	report.Languages = ctx.Languages

	// Git info (if initialized)
	if ctx.Git.Initialized {
		report.Git = &GitInfo{
			Initialized: true,
			Branch:      ctx.Git.Branch,
			Dirty:       ctx.Git.Dirty,
			Uncommitted: ctx.Git.Uncommitted,
		}
	}

	// CI info (if detected)
	if ctx.CI.Detected {
		report.CI = &CIInfo{
			Detected: true,
			Name:     ctx.CI.Name,
		}
	}

	return report
}
