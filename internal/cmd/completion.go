package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/templates/workflows"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for specular commands.

Shell completion enables tab-completion for commands, flags, and arguments.
Install the completion script for your shell to enable this feature.

Bash:
  # Linux
  specular completion bash > /etc/bash_completion.d/specular

  # macOS with Homebrew
  specular completion bash > $(brew --prefix)/etc/bash_completion.d/specular

Zsh:
  # If shell completion is not already enabled, add this to ~/.zshrc:
  autoload -Uz compinit && compinit

  # Generate and install completion
  specular completion zsh > "${fpath[1]}/_specular"

  # Or for Oh My Zsh:
  specular completion zsh > ~/.oh-my-zsh/completions/_specular

Fish:
  specular completion fish > ~/.config/fish/completions/specular.fish

PowerShell:
  specular completion powershell > specular.ps1
  # Add to $PROFILE: . /path/to/specular.ps1
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)

	// Register dynamic completions for various flags and arguments
	registerDynamicCompletions()
}

// registerDynamicCompletions sets up dynamic completion functions for flags
func registerDynamicCompletions() {
	// Workflow template completion
	registerWorkflowCompletion()

	// Provider completion
	registerProviderCompletion()

	// Config file completion
	registerConfigFileCompletion()

	// Scenario completion
	registerScenarioCompletion()

	// Format completion
	registerFormatCompletion()

	// Governance level completion
	registerGovernanceCompletion()
}

// registerWorkflowCompletion adds completion for workflow template IDs
func registerWorkflowCompletion() {
	// For init workflow command
	if initWorkflowCmd != nil {
		initWorkflowCmd.ValidArgsFunction = workflowTemplateCompletion
	}

	// For --workflow flag on init
	if initCmd != nil {
		if err := initCmd.RegisterFlagCompletionFunc("workflow", workflowTemplateCompletion); err != nil {
			// Flag may not exist, ignore error
		}
	}
}

// workflowTemplateCompletion provides completion for workflow template IDs
func workflowTemplateCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	registry, err := workflows.NewRegistry()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	templates := registry.List()
	for _, tmpl := range templates {
		if strings.HasPrefix(tmpl.ID, toComplete) {
			// Format: id\tdescription
			completions = append(completions, fmt.Sprintf("%s\t%s", tmpl.ID, tmpl.Description))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// registerProviderCompletion adds completion for provider names
func registerProviderCompletion() {
	providers := []string{
		"ollama\tLocal AI provider",
		"openai\tOpenAI API",
		"anthropic\tAnthropic Claude API",
		"gemini\tGoogle Gemini API",
	}

	// For --providers flag on init
	if initCmd != nil {
		if err := initCmd.RegisterFlagCompletionFunc("providers", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return filterCompletions(providers, toComplete), cobra.ShellCompDirectiveNoFileComp
		}); err != nil {
			// Flag may not exist, ignore error
		}
	}
}

// registerConfigFileCompletion adds completion for config file paths
func registerConfigFileCompletion() {
	configPatterns := []string{
		".specular/providers.yaml",
		".specular/routing.yaml",
		".specular/policy.yaml",
		".specular/spec.yaml",
		".specular/slo.yaml",
		".specular/config.yaml",
	}

	// For config validate command
	if configValidateCmd != nil {
		configValidateCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			var completions []string
			for _, pattern := range configPatterns {
				if strings.HasPrefix(pattern, toComplete) {
					completions = append(completions, pattern)
				}
			}
			// Also allow file completion
			return completions, cobra.ShellCompDirectiveDefault
		}
	}

	// For config fix command
	if configFixCmd != nil {
		configFixCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			var completions []string
			for _, pattern := range configPatterns {
				if strings.HasPrefix(pattern, toComplete) {
					completions = append(completions, pattern)
				}
			}
			return completions, cobra.ShellCompDirectiveDefault
		}
	}
}

// registerScenarioCompletion adds completion for test scenarios
func registerScenarioCompletion() {
	scenarios := []string{
		"smoke\tQuick validation tests",
		"integration\tIntegration tests",
		"security\tSecurity scanning tests",
		"performance\tPerformance benchmarks",
	}

	// Register for any commands that have --scenario flag
	for _, cmd := range rootCmd.Commands() {
		if cmd.Flags().Lookup("scenario") != nil {
			if err := cmd.RegisterFlagCompletionFunc("scenario", func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				return filterCompletions(scenarios, toComplete), cobra.ShellCompDirectiveNoFileComp
			}); err != nil {
				// Ignore error
			}
		}
	}
}

// registerFormatCompletion adds completion for output format
func registerFormatCompletion() {
	formats := []string{
		"text\tHuman-readable text output",
		"json\tJSON output for scripting",
		"yaml\tYAML output",
		"sarif\tSARIF format for CI tools",
	}

	// Register for root command --format flag
	if err := rootCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return filterCompletions(formats, toComplete), cobra.ShellCompDirectiveNoFileComp
	}); err != nil {
		// Ignore error
	}

	// Register for config validate --format flag
	if configValidateCmd != nil {
		if err := configValidateCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return filterCompletions(formats, toComplete), cobra.ShellCompDirectiveNoFileComp
		}); err != nil {
			// Ignore error
		}
	}
}

// registerGovernanceCompletion adds completion for governance levels
func registerGovernanceCompletion() {
	levels := []string{
		"L2\tStandard governance (default)",
		"L3\tEnhanced governance with approvals",
		"L4\tFull governance with audit trail",
	}

	// Register for init --governance flag
	if initCmd != nil {
		if err := initCmd.RegisterFlagCompletionFunc("governance", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return filterCompletions(levels, toComplete), cobra.ShellCompDirectiveNoFileComp
		}); err != nil {
			// Ignore error
		}
	}
}

// filterCompletions filters completion options by prefix
func filterCompletions(options []string, prefix string) []string {
	if prefix == "" {
		return options
	}

	var filtered []string
	for _, opt := range options {
		// Options may be in format "value\tdescription"
		value := strings.Split(opt, "\t")[0]
		if strings.HasPrefix(value, prefix) {
			filtered = append(filtered, opt)
		}
	}
	return filtered
}

// CompletionShells returns the list of supported shells
func CompletionShells() []string {
	return []string{"bash", "zsh", "fish", "powershell"}
}
