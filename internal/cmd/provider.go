package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/internal/ux"
)

const (
	defaultProviderConfigPath = ".specular/providers.yaml"
	exampleProviderConfigPath = ".specular/providers.yaml.example"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage AI providers",
	Long: `Manage AI providers that specular can use for various tasks.
Providers can be local models (ollama), cloud APIs (OpenAI, Anthropic), or custom implementations.`,
}

var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured providers",
	Long:  `List all configured providers and their current status (enabled/disabled, loaded/not loaded).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := cmd.Flags().Lookup("config").Value.String()
		if configPath == "" {
			// Try to discover providers.yaml in multiple locations
			discoveredPath, discoverErr := ux.DiscoverConfigFile("providers.yaml")
			if discoverErr == nil {
				if _, statErr := os.Stat(discoveredPath); statErr == nil {
					configPath = discoveredPath
				}
			}
			// Fall back to default if discovery didn't find existing file
			if configPath == "" {
				configPath = defaultProviderConfigPath
			}
		}

		// Check if config file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			fmt.Printf("No provider configuration found at %s\n", configPath)
			fmt.Printf("Run 'specular provider init' to create one.\n")
			fmt.Printf("Tip: Specular will auto-discover providers if you have ollama installed or API keys set.\n")
			return nil
		}

		// Load config
		config, err := provider.LoadProvidersConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load provider config: %w", err)
		}

		// Print providers table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tENABLED\tSOURCE\tVERSION") //nolint:errcheck
		fmt.Fprintln(w, "----\t----\t-------\t------\t-------") //nolint:errcheck

		for _, p := range config.Providers {
			enabled := "no"
			if p.Enabled {
				enabled = "yes"
			}

			source := p.Source
			if source == "" {
				source = "-"
			}

			version := p.Version
			if version == "" {
				version = "-"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", //nolint:errcheck
				p.Name, p.Type, enabled, source, version)
		}

		w.Flush() //#nosec G104 -- Tabwriter flush errors not critical

		// Print strategy info
		if config.Strategy.Budget.MaxCostPerDay > 0 || config.Strategy.Budget.MaxCostPerRequest > 0 {
			fmt.Println("\nBudget Constraints:")
			if config.Strategy.Budget.MaxCostPerDay > 0 {
				fmt.Printf("  Max cost per day: $%.2f\n", config.Strategy.Budget.MaxCostPerDay)
			}
			if config.Strategy.Budget.MaxCostPerRequest > 0 {
				fmt.Printf("  Max cost per request: $%.2f\n", config.Strategy.Budget.MaxCostPerRequest)
			}
		}

		if len(config.Strategy.Preference) > 0 {
			fmt.Println("\nProvider Preference Order:")
			for i, name := range config.Strategy.Preference {
				fmt.Printf("  %d. %s\n", i+1, name)
			}
		}

		return nil
	},
}

var providerDoctorCmd = &cobra.Command{
	Use:     "doctor [provider-name]",
	Aliases: []string{"health"}, // Keep health for backward compatibility
	Short:   "Check provider health and configuration",
	Long:    `Check the health status of providers. If no provider name is specified, checks all enabled providers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := cmd.Flags().Lookup("config").Value.String()
		if configPath == "" {
			// Try to discover providers.yaml in multiple locations
			discoveredPath, discoverErr := ux.DiscoverConfigFile("providers.yaml")
			if discoverErr == nil {
				if _, statErr := os.Stat(discoveredPath); statErr == nil {
					configPath = discoveredPath
				}
			}
			// Fall back to default if discovery didn't find existing file
			if configPath == "" {
				configPath = defaultProviderConfigPath
			}
		}

		// Load registry with auto-discovery (will try config first, then auto-discover)
		registry, err := provider.LoadRegistryWithAutoDiscovery(configPath)
		if err != nil {
			return fmt.Errorf("failed to load providers: %w", err)
		}

		// Check if specific provider requested
		var providersToCheck []string
		if len(args) > 0 {
			providersToCheck = args
		} else {
			providersToCheck = registry.List()
		}

		if len(providersToCheck) == 0 {
			fmt.Println("No providers loaded.")
			return nil
		}

		// Check health of each provider
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "PROVIDER\tSTATUS\tMESSAGE") //nolint:errcheck
		fmt.Fprintln(w, "--------\t------\t-------") //nolint:errcheck

		for _, name := range providersToCheck {
			prov, getErr := registry.Get(name)
			if err != nil {
				fmt.Fprintf(w, "%s\t❌ ERROR\t%v\n", name, getErr) //nolint:errcheck
				continue
			}

			if healthErr := prov.Health(ctx); healthErr != nil {
				fmt.Fprintf(w, "%s\t❌ UNHEALTHY\t%v\n", name, healthErr) //nolint:errcheck
			} else {
				info := prov.GetInfo()
				fmt.Fprintf(w, "%s\t✅ HEALTHY\t%s\n", name, info.Description) //nolint:errcheck
			}
		}

		w.Flush() //#nosec G104 -- Tabwriter flush errors not critical

		return nil
	},
}

var providerInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize provider configuration",
	Long: `Initialize provider configuration by copying the example file.
This creates a providers.yaml file from providers.yaml.example with default settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force := cmd.Flags().Lookup("force").Value.String() == "true"

		// Check if example file exists
		if _, err := os.Stat(exampleProviderConfigPath); os.IsNotExist(err) {
			return fmt.Errorf("example config file not found at %s", exampleProviderConfigPath)
		}

		// Check if target file already exists
		if _, err := os.Stat(defaultProviderConfigPath); err == nil && !force {
			return fmt.Errorf("provider config already exists at %s (use --force to overwrite)", defaultProviderConfigPath)
		}

		// Read example file
		data, err := os.ReadFile(exampleProviderConfigPath)
		if err != nil {
			return fmt.Errorf("failed to read example config: %w", err)
		}

		// Ensure .specular directory exists (with all subdirectories)
		if mkdirErr := ux.EnsureSpecularDir(); mkdirErr != nil {
			return fmt.Errorf("failed to create .specular directory: %w", mkdirErr)
		}

		// Get the discovered .specular directory
		specularDir, discoverErr := ux.DiscoverSpecularDir()
		if discoverErr != nil {
			return fmt.Errorf("failed to discover .specular directory: %w", discoverErr)
		}

		// Write to providers.yaml in discovered directory
		targetPath := filepath.Join(specularDir, "providers.yaml")
		if writeErr := os.WriteFile(targetPath, data, 0o600); writeErr != nil {
			return fmt.Errorf("failed to write provider config: %w", writeErr)
		}

		fmt.Printf("✓ Created provider configuration at %s\n", targetPath)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit .specular/providers.yaml to enable desired providers")
		fmt.Println("  2. Set any required API keys as environment variables")
		fmt.Println("  3. Run 'specular provider health' to check provider status")

		return nil
	},
}

var providerAddCmd = &cobra.Command{
	Use:   "add <provider-name>",
	Short: "Add a provider to configuration",
	Long: `Add a new provider to the providers.yaml configuration.

Supported providers:
  - ollama (local models)
  - anthropic (Claude API)
  - openai (GPT API)
  - claude-code (Claude Code CLI)
  - gemini-cli (Gemini CLI)
  - codex-cli (Codex CLI)
  - copilot-cli (GitHub Copilot)`,
	Args: cobra.ExactArgs(1),
	RunE: runProviderAdd,
}

var providerRemoveCmd = &cobra.Command{
	Use:     "remove <provider-name>",
	Aliases: []string{"rm"},
	Short:   "Remove a provider from configuration",
	Long:    `Remove a provider from the providers.yaml configuration.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runProviderRemove,
}

func runProviderAdd(cmd *cobra.Command, args []string) error {
	providerName := args[0]
	configPath := cmd.Flags().Lookup("config").Value.String()
	if configPath == "" {
		configPath = defaultProviderConfigPath
	}

	// Load existing config or create new one
	var config *provider.ProvidersConfig
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create new config with defaults
		config = provider.DefaultProvidersConfig()
		config.Providers = []provider.ProviderConfig{} // Clear default providers
	} else {
		var loadErr error
		config, loadErr = provider.LoadProvidersConfig(configPath)
		if loadErr != nil {
			return fmt.Errorf("failed to load provider config: %w", loadErr)
		}
	}

	// Check if provider already exists
	for _, p := range config.Providers {
		if p.Name == providerName {
			return fmt.Errorf("provider %s already exists in configuration", providerName)
		}
	}

	// Generate provider config based on name
	newProvider := generateProviderConfigForAdd(providerName)
	if newProvider == nil {
		return fmt.Errorf("unknown provider: %s", providerName)
	}

	// Add to config
	config.Providers = append(config.Providers, *newProvider)

	// Save config
	if err := provider.SaveProvidersConfig(config, configPath); err != nil {
		return fmt.Errorf("failed to save provider config: %w", err)
	}

	fmt.Printf("✓ Added provider: %s\n", providerName)
	fmt.Printf("  Type: %s\n", newProvider.Type)
	fmt.Printf("  Enabled: %v\n", newProvider.Enabled)
	if !newProvider.Enabled {
		fmt.Printf("\nTo enable, edit %s and set enabled: true\n", configPath)
	}

	return nil
}

func runProviderRemove(cmd *cobra.Command, args []string) error {
	providerName := args[0]
	configPath := cmd.Flags().Lookup("config").Value.String()
	if configPath == "" {
		configPath = defaultProviderConfigPath
	}

	// Load config
	config, err := provider.LoadProvidersConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load provider config: %w", err)
	}

	// Find and remove provider
	found := false
	newProviders := []provider.ProviderConfig{}
	for _, p := range config.Providers {
		if p.Name == providerName {
			found = true
			continue // Skip this provider
		}
		newProviders = append(newProviders, p)
	}

	if !found {
		return fmt.Errorf("provider %s not found in configuration", providerName)
	}

	// Update config
	config.Providers = newProviders

	// Save config
	if err := provider.SaveProvidersConfig(config, configPath); err != nil {
		return fmt.Errorf("failed to save provider config: %w", err)
	}

	fmt.Printf("✓ Removed provider: %s\n", providerName)

	return nil
}

// generateProviderConfigForAdd creates a provider config for the add command
func generateProviderConfigForAdd(providerName string) *provider.ProviderConfig {
	// Reuse the existing generateProviderConfig function from provider package
	// But set enabled based on availability
	switch providerName {
	case "ollama":
		return &provider.ProviderConfig{
			Name:    "ollama",
			Type:    provider.ProviderTypeCLI,
			Enabled: false, // User must configure
			Source:  "local",
			Config: map[string]interface{}{
				"path":     "ollama",
				"base_url": "http://localhost:11434",
			},
			Models: map[string]string{
				"fast":         "llama3.3:70b",
				"codegen":      "qwen2.5-coder:7b",
				"cheap":        "llama3.2",
				"long-context": "llama3.3:70b",
			},
		}
	case "anthropic":
		return &provider.ProviderConfig{
			Name:    "anthropic",
			Type:    provider.ProviderTypeAPI,
			Enabled: provider.IsEnvVarSet("ANTHROPIC_API_KEY"),
			Source:  "api",
			Config: map[string]interface{}{
				"api_key":  "${ANTHROPIC_API_KEY}",
				"base_url": "https://api.anthropic.com",
			},
			Models: map[string]string{
				"fast":         "claude-haiku-4-5-20251015",
				"codegen":      "claude-sonnet-4-5-20250929",
				"agentic":      "claude-opus-4-1-20250805",
				"long-context": "claude-sonnet-4-5-20250929",
				"cheap":        "claude-haiku-4-5-20251015",
			},
		}
	case "openai":
		return &provider.ProviderConfig{
			Name:    "openai",
			Type:    provider.ProviderTypeAPI,
			Enabled: provider.IsEnvVarSet("OPENAI_API_KEY"),
			Source:  "api",
			Config: map[string]interface{}{
				"api_key":  "${OPENAI_API_KEY}",
				"base_url": "https://api.openai.com/v1",
			},
			Models: map[string]string{
				"fast":         "gpt-5-mini",
				"codegen":      "gpt-5",
				"cheap":        "gpt-5-nano",
				"long-context": "gpt-5",
				"agentic":      "gpt-5",
			},
		}
	case "claude-code":
		wrapperPath, _ := filepath.Abs("providers/claude-code/claude-code-provider")
		return &provider.ProviderConfig{
			Name:    "claude-code",
			Type:    provider.ProviderTypeCLI,
			Enabled: false,
			Source:  "local",
			Config: map[string]interface{}{
				"path": wrapperPath,
			},
			Models: map[string]string{
				"fast":         "claude-haiku-4-5-20251015",
				"codegen":      "claude-sonnet-4-5-20250929",
				"agentic":      "claude-opus-4-1-20250805",
				"long-context": "claude-sonnet-4-5-20250929",
				"cheap":        "claude-haiku-4-5-20251015",
			},
		}
	case "gemini-cli":
		wrapperPath, _ := filepath.Abs("providers/gemini-cli/gemini-cli-provider")
		return &provider.ProviderConfig{
			Name:    "gemini-cli",
			Type:    provider.ProviderTypeCLI,
			Enabled: false,
			Source:  "local",
			Config: map[string]interface{}{
				"path": wrapperPath,
			},
			Models: map[string]string{
				"fast":         "gemini-2.0-flash-exp",
				"codegen":      "gemini-exp-1206",
				"agentic":      "gemini-exp-1206",
				"long-context": "gemini-exp-1206",
				"cheap":        "gemini-2.0-flash-exp",
			},
		}
	case "copilot-cli":
		return &provider.ProviderConfig{
			Name:    "copilot-cli",
			Type:    provider.ProviderTypeCLI,
			Enabled: false,
			Source:  "local",
			Config: map[string]interface{}{
				"path": "copilot",
			},
			Models: map[string]string{
				"fast":         "copilot",
				"codegen":      "copilot",
				"agentic":      "copilot",
				"long-context": "copilot",
				"cheap":        "copilot",
			},
		}
	case "codex-cli":
		wrapperPath, _ := filepath.Abs("providers/codex-cli/codex-cli-provider")
		return &provider.ProviderConfig{
			Name:    "codex-cli",
			Type:    provider.ProviderTypeCLI,
			Enabled: false,
			Source:  "local",
			Config: map[string]interface{}{
				"path": wrapperPath,
			},
			Models: map[string]string{
				"fast":         "codex",
				"codegen":      "codex",
				"agentic":      "codex",
				"long-context": "codex",
				"cheap":        "codex",
			},
		}
	}

	return nil
}

func init() {
	// Add provider command to root
	rootCmd.AddCommand(providerCmd)

	// Add subcommands
	providerCmd.AddCommand(providerListCmd)
	providerCmd.AddCommand(providerDoctorCmd)
	providerCmd.AddCommand(providerInitCmd)
	providerCmd.AddCommand(providerAddCmd)
	providerCmd.AddCommand(providerRemoveCmd)

	// Flags for list command
	providerListCmd.Flags().String("config", "", "Path to provider config file (default: .specular/providers.yaml)")

	// Flags for doctor command
	providerDoctorCmd.Flags().String("config", "", "Path to provider config file (default: .specular/providers.yaml)")

	// Flags for init command
	providerInitCmd.Flags().Bool("force", false, "Overwrite existing provider config")

	// Flags for add command
	providerAddCmd.Flags().String("config", "", "Path to provider config file (default: .specular/providers.yaml)")

	// Flags for remove command
	providerRemoveCmd.Flags().String("config", "", "Path to provider config file (default: .specular/providers.yaml)")
}
