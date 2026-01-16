package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/detect"
	"github.com/felixgeelhaar/specular/internal/docgen"
	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/internal/safeutil"
	"github.com/felixgeelhaar/specular/internal/tui"
	"github.com/felixgeelhaar/specular/internal/ux"
)

const (
	defaultProviderConfigPath = ".specular/providers.yaml"
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
		configPath, err := resolveProviderConfigPath(cmd)
		if err != nil {
			return err
		}

		config, created, examplePath, recommended, err := loadProviderConfigWithBootstrap(configPath)
		if err != nil {
			return err
		}

		if created {
			fmt.Printf("✓ Created provider configuration at %s\n", configPath)
			if len(recommended) > 0 {
				fmt.Printf("  Recommended providers enabled: %s\n", strings.Join(recommended, ", "))
			}
			if examplePath != "" {
				fmt.Printf("  Example config available at %s\n", examplePath)
			}
			fmt.Println()
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
	Use:   "doctor [provider-name]",
	Short: "Check provider health and configuration",
	Long:  `Check the health status of providers. If no provider name is specified, checks all enabled providers. Use --quick to skip detailed health checks and list provider status only.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		quick := cmd.Flags().Lookup("quick").Value.String() == "true"

		configPath, err := resolveProviderConfigPath(cmd)
		if err != nil {
			return err
		}

		_, created, examplePath, recommended, err := loadProviderConfigWithBootstrap(configPath)
		if err != nil {
			return err
		}

		if created {
			fmt.Printf("✓ Created provider configuration at %s\n", configPath)
			if len(recommended) > 0 {
				fmt.Printf("  Recommended providers enabled: %s\n", strings.Join(recommended, ", "))
			}
			if examplePath != "" {
				fmt.Printf("  Example config available at %s\n", examplePath)
			}
			fmt.Println()
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

			if quick {
				info := prov.GetInfo()
				fmt.Fprintf(w, "%s\t✅ HEALTHY\t%s (quick check)\n", name, info.Description) //nolint:errcheck
				provider.RecordProviderHealth(name, true, "quick check")
				continue
			}

			if healthErr := prov.Health(ctx); healthErr != nil {
				fmt.Fprintf(w, "%s\t❌ UNHEALTHY\t%v\n", name, healthErr) //nolint:errcheck
				provider.RecordProviderHealth(name, false, healthErr.Error())
			} else {
				info := prov.GetInfo()
				fmt.Fprintf(w, "%s\t✅ HEALTHY\t%s\n", name, info.Description) //nolint:errcheck
				provider.RecordProviderHealth(name, true, info.Description)
			}
		}

		w.Flush() //#nosec G104 -- Tabwriter flush errors not critical

		return nil
	},
}

var providerInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize provider configuration",
	Long: `Generate a providers.yaml file from the bundled descriptor catalog with sensible defaults.

Use --recommendations to preview recommended providers without writing any files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force := cmd.Flags().Lookup("force").Value.String() == "true"
		showRecommendations := cmd.Flags().Lookup("recommendations").Value.String() == "true"

		ctx := detectProviderContext()
		recommended := ctx.GetRecommendedProviders()

		// If --recommendations flag is set, just show recommendations and exit
		if showRecommendations {
			printProviderRecommendations(ctx, recommended)
			return nil
		}

		// Check if target file already exists
		if _, err := os.Stat(defaultProviderConfigPath); err == nil && !force {
			return fmt.Errorf("provider config already exists at %s (use --force to overwrite)", defaultProviderConfigPath)
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

		exampleConfig := provider.DefaultProvidersConfig()
		examplePath := filepath.Join(specularDir, "providers.yaml.example")
		if err := provider.SaveProvidersConfigExample(exampleConfig, examplePath); err != nil {
			return fmt.Errorf("failed to write provider example: %w", err)
		}

		config := provider.ConfigFromRecommended(recommended)

		// Write to providers.yaml in discovered directory
		targetPath := filepath.Join(specularDir, "providers.yaml")
		if writeErr := provider.SaveProvidersConfig(config, targetPath); writeErr != nil {
			return fmt.Errorf("failed to write provider config: %w", writeErr)
		}

		fmt.Printf("✓ Created provider configuration at %s\n", targetPath)
		fmt.Printf("✓ Created provider example at %s\n", examplePath)
		if len(recommended) > 0 {
			fmt.Printf("⚡ Enabled recommended providers: %s\n", strings.Join(recommended, ", "))
		}
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit .specular/providers.yaml to enable desired providers")
		fmt.Println("  2. Review the example at .specular/providers.yaml.example for reference")
		fmt.Println("  3. Set any required API keys as environment variables")
		fmt.Println("  4. Run 'specular provider doctor' to check provider status")

		return nil
	},
}

// printProviderRecommendations prints an annotated list of recommended providers
func printProviderRecommendations(ctx *detect.Context, recommended []string) {
	fmt.Println("Provider Recommendations")
	fmt.Println("========================")
	fmt.Println()

	if len(recommended) == 0 {
		fmt.Println("No providers were detected or recommended for your environment.")
		fmt.Println("\nAvailable provider types:")
		for _, desc := range provider.Descriptors() {
			fmt.Printf("  • %s (%s)\n", desc.Name, desc.Description)
		}
		fmt.Println("\nRun 'specular provider add <provider-name>' to manually add a provider.")
		return
	}

	fmt.Println("Based on your environment, the following providers are recommended:")
	fmt.Println()

	for i, name := range recommended {
		fmt.Printf("%d. %s\n", i+1, providerDisplayName(name))

		// Get descriptor for detailed info
		if desc := provider.DescriptorByName(name); desc != nil {
			fmt.Printf("   Type: %s | Source: %s | Trust: %s\n", desc.Type, desc.Source, desc.TrustLevel)
			if desc.Description != "" {
				fmt.Printf("   Description: %s\n", desc.Description)
			}
			if len(desc.Capabilities) > 0 {
				fmt.Printf("   Capabilities: %s\n", strings.Join(desc.Capabilities, ", "))
			}
			if hints := docgen.FormatDetectionHints(desc.Hints); hints != "" {
				fmt.Printf("   Detection: %s\n", hints)
			}
		}

		// Show detection status from context
		if info, exists := ctx.Providers[name]; exists {
			if info.Detected {
				fmt.Printf("   Status: ✓ Detected")
				if info.Version != "" {
					fmt.Printf(" (version %s)", info.Version)
				}
				fmt.Println()
			}
			if info.EnvKeySet {
				fmt.Printf("   API Key: ✓ Environment variable is set\n")
			}
		}
		fmt.Println()
	}

	fmt.Println("To apply these recommendations:")
	fmt.Println("  specular provider init")
	fmt.Println()
	fmt.Println("To add a specific provider manually:")
	fmt.Println("  specular provider add <provider-name>")
}

var providerAddCmd = &cobra.Command{
	Use:   "add [provider-name]",
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
	Args: cobra.MaximumNArgs(1),
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
	var providerName string
	if len(args) > 0 {
		providerName = args[0]
	} else {
		interactiveCfg := ux.NewInteractiveConfig()
		if !ux.ShouldPrompt(interactiveCfg) {
			return fmt.Errorf("provider name is required when running in non-interactive mode")
		}
		name, err := promptForProviderSelection()
		if err != nil {
			return err
		}
		providerName = name
	}
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

	// Let the user adjust defaults interactively when possible
	if err := configureProviderInteractively(providerName, newProvider); err != nil {
		return fmt.Errorf("configuring provider %s: %w", providerName, err)
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

func resolveProviderConfigPath(cmd *cobra.Command) (string, error) {
	configPath := cmd.Flags().Lookup("config").Value.String()
	if configPath != "" {
		return configPath, nil
	}

	discoveredPath, err := ux.DiscoverConfigFile("providers.yaml")
	if err == nil {
		if _, statErr := os.Stat(discoveredPath); statErr == nil {
			return discoveredPath, nil
		}
	}

	return defaultProviderConfigPath, nil
}

func loadProviderConfigWithBootstrap(configPath string) (*provider.ProvidersConfig, bool, string, []string, error) {
	config, err := provider.LoadProvidersConfig(configPath)
	if err == nil {
		return config, false, "", nil, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, false, "", nil, fmt.Errorf("failed to load provider config: %w", err)
	}

	ctx := detectProviderContext()
	recommended := ctx.GetRecommendedProviders()
	config, err = provider.WriteProvidersConfigFromDescriptors(configPath, recommended)
	if err != nil {
		return nil, false, "", nil, fmt.Errorf("failed to create provider config: %w", err)
	}

	exampleDir := filepath.Dir(configPath)
	examplePath, joinErr := safeutil.JoinInsideBase(exampleDir, "providers.yaml.example")
	if joinErr != nil {
		fmt.Printf("⚠️  Failed to secure provider example path: %v\n", joinErr)
		examplePath = filepath.Join(exampleDir, "providers.yaml.example")
	}
	if exampleErr := provider.SaveProvidersConfigExample(config, examplePath); exampleErr != nil {
		fmt.Printf("⚠️  Failed to write provider example: %v\n", exampleErr)
	}

	return config, true, examplePath, recommended, nil
}

func detectProviderContext() *detect.Context {
	ctx, err := detect.DetectAll()
	if err != nil {
		fmt.Printf("⚠  Provider detection failed: %v\n", err)
		fmt.Println("   Proceeding with the default provider catalog.")
		return &detect.Context{Providers: make(map[string]detect.ProviderInfo)}
	}

	printDetectionSummary(ctx)
	return ctx
}

// generateProviderConfigForAdd creates a provider config for the add command
func generateProviderConfigForAdd(providerName string) *provider.ProviderConfig {
	if desc := provider.DescriptorByName(providerName); desc != nil {
		cfg := desc.ToProviderConfig()
		return &cfg
	}
	return nil
}

type configFieldKind int

const (
	fieldKindString configFieldKind = iota
	fieldKindPath
	fieldKindEnv
)

type interactiveField struct {
	key          string
	prompt       string
	kind         configFieldKind
	defaultValue string
}

var interactiveProviderFields = map[string][]interactiveField{
	"ollama": {
		{key: "path", prompt: "Path to Ollama CLI binary", kind: fieldKindPath, defaultValue: "ollama"},
		{key: "base_url", prompt: "Base URL for the Ollama HTTP endpoint", kind: fieldKindString, defaultValue: "http://localhost:11434"},
	},
	"claude-code": {
		{key: "path", prompt: "Path to Claude Code wrapper", kind: fieldKindPath},
	},
	"gemini-cli": {
		{key: "path", prompt: "Path to Gemini CLI wrapper", kind: fieldKindPath},
	},
	"codex-cli": {
		{key: "path", prompt: "Path to Codex CLI wrapper", kind: fieldKindPath},
	},
	"copilot-cli": {
		{key: "path", prompt: "Path to GitHub Copilot CLI executable", kind: fieldKindPath},
	},
	"anthropic": {
		{key: "api_key", prompt: "Environment variable for Anthropic API key", kind: fieldKindEnv, defaultValue: "ANTHROPIC_API_KEY"},
	},
	"openai": {
		{key: "api_key", prompt: "Environment variable for OpenAI API key", kind: fieldKindEnv, defaultValue: "OPENAI_API_KEY"},
	},
}

func configureProviderInteractively(providerName string, cfg *provider.ProviderConfig) error {
	interactiveCfg := ux.NewInteractiveConfig()
	shouldPrompt := ux.ShouldPrompt(interactiveCfg)

	applyDefaultEnable(cfg)
	if !shouldPrompt {
		return nil
	}

	if desc := provider.DescriptorByName(providerName); desc != nil {
		fmt.Printf("\nProvider %s metadata:\n", providerDisplayName(providerName))
		fmt.Printf("  Source: %s | Trust: %s\n", desc.Source, desc.TrustLevel)
		if desc.Description != "" {
			fmt.Printf("  Description: %s\n", desc.Description)
		}
		if hints := docgen.FormatDetectionHints(desc.Hints); hints != "" {
			fmt.Printf("  %s\n", hints)
		}
		if len(desc.Capabilities) > 0 {
			fmt.Printf("  Capabilities: %s\n", strings.Join(desc.Capabilities, ", "))
		}
	}

	fmt.Printf("\nConfiguring provider %s (%s)\n", providerName, cfg.Type)

	if fields, ok := interactiveProviderFields[providerName]; ok {
		for _, field := range fields {
			if err := promptProviderField(cfg, field); err != nil {
				return err
			}
		}
	}

	enabled, err := tui.PromptForConfirmation(fmt.Sprintf("Enable %s now?", providerDisplayName(providerName)), cfg.Enabled)
	if err != nil {
		return fmt.Errorf("confirmation prompt failed: %w", err)
	}
	cfg.Enabled = enabled

	return nil
}

func applyDefaultEnable(cfg *provider.ProviderConfig) {
	if cfg.Type == provider.ProviderTypeCLI {
		cfg.Enabled = true
	}
}

func promptProviderField(cfg *provider.ProviderConfig, field interactiveField) error {
	defaultValue := fieldDefaultValue(cfg, field)

	switch field.kind {
	case fieldKindPath:
		cfg.Config[field.key] = ux.PromptForPath(field.prompt, defaultValue)
	case fieldKindString:
		value, err := promptString(field.prompt, defaultValue)
		if err != nil {
			return err
		}
		if strings.TrimSpace(value) == "" {
			value = defaultValue
		}
		cfg.Config[field.key] = value
	case fieldKindEnv:
		envValue, err := promptString(field.prompt, defaultValue)
		if err != nil {
			return err
		}
		if strings.TrimSpace(envValue) == "" {
			envValue = defaultValue
		}
		envValue = strings.TrimSpace(envValue)
		cfg.Config[field.key] = fmt.Sprintf("${%s}", envValue)
		cfg.Enabled = provider.IsEnvVarSet(envValue)
	}

	return nil
}

func fieldDefaultValue(cfg *provider.ProviderConfig, field interactiveField) string {
	if raw, ok := cfg.Config[field.key]; ok && raw != nil {
		value := fmt.Sprintf("%v", raw)
		if field.kind == fieldKindEnv {
			return envVarFromValue(value)
		}
		return value
	}

	return field.defaultValue
}

func promptString(message, defaultValue string) (string, error) {
	return tui.PromptForString(tui.Prompt{
		Message:     message,
		Default:     defaultValue,
		Placeholder: message,
	})
}

func promptForProviderSelection() (string, error) {
	descs := provider.Descriptors()
	if len(descs) == 0 {
		return "", fmt.Errorf("no provider descriptors registered")
	}

	fmt.Println("\nAvailable providers:")
	for i, desc := range descs {
		index := i + 1
		fmt.Printf("  %d) %s (%s) - %s\n", index, desc.Name, desc.Source, desc.Description)
		if hints := docgen.FormatDetectionHints(desc.Hints); hints != "" {
			fmt.Printf("       %s\n", hints)
		}
		if len(desc.Capabilities) > 0 {
			fmt.Printf("       Capabilities: %s\n", strings.Join(desc.Capabilities, ", "))
		}
	}

	choice, err := promptString("Choose provider number", "1")
	if err != nil {
		return "", fmt.Errorf("failed to read provider choice: %w", err)
	}

	idx, err := strconv.Atoi(strings.TrimSpace(choice))
	if err != nil {
		return "", fmt.Errorf("invalid selection: %w", err)
	}

	if idx <= 0 || idx > len(descs) {
		return "", fmt.Errorf("selection %d is out of range", idx)
	}

	return descs[idx-1].Name, nil
}

func envVarFromValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		return strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
	}
	return value
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
	providerDoctorCmd.Flags().Bool("quick", false, "Skip provider health checks and return quick summary")

	// Flags for init command
	providerInitCmd.Flags().Bool("force", false, "Overwrite existing provider config")
	providerInitCmd.Flags().Bool("recommendations", false, "Show recommended providers without writing files")

	// Flags for add command
	providerAddCmd.Flags().String("config", "", "Path to provider config file (default: .specular/providers.yaml)")

	// Flags for remove command
	providerRemoveCmd.Flags().String("config", "", "Path to provider config file (default: .specular/providers.yaml)")
}
