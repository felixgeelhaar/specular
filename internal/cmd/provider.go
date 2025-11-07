package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/spf13/cobra"
)

const (
	defaultProviderConfigPath = ".specular/providers.yaml"
	exampleProviderConfigPath = ".specular/providers.yaml.example"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage AI providers",
	Long: `Manage AI providers that ai-dev can use for various tasks.
Providers can be local models (ollama), cloud APIs (OpenAI, Anthropic), or custom implementations.`,
}

var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured providers",
	Long:  `List all configured providers and their current status (enabled/disabled, loaded/not loaded).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath = defaultProviderConfigPath
		}

		// Check if config file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			fmt.Printf("No provider configuration found at %s\n", configPath)
			fmt.Printf("Run 'ai-dev provider init' to create one from the example.\n")
			return nil
		}

		// Load config
		config, err := provider.LoadProvidersConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load provider config: %w", err)
		}

		// Print providers table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tENABLED\tSOURCE\tVERSION")
		fmt.Fprintln(w, "----\t----\t-------\t------\t-------")

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

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				p.Name, p.Type, enabled, source, version)
		}

		w.Flush()

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

var providerHealthCmd = &cobra.Command{
	Use:   "health [provider-name]",
	Short: "Check provider health",
	Long:  `Check the health status of providers. If no provider name is specified, checks all enabled providers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath = defaultProviderConfigPath
		}

		// Check if config file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("no provider configuration found at %s", configPath)
		}

		// Load registry from config
		registry, err := provider.LoadRegistryFromConfig(configPath)
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
		fmt.Fprintln(w, "PROVIDER\tSTATUS\tMESSAGE")
		fmt.Fprintln(w, "--------\t------\t-------")

		for _, name := range providersToCheck {
			prov, err := registry.Get(name)
			if err != nil {
				fmt.Fprintf(w, "%s\t❌ ERROR\t%v\n", name, err)
				continue
			}

			if err := prov.Health(ctx); err != nil {
				fmt.Fprintf(w, "%s\t❌ UNHEALTHY\t%v\n", name, err)
			} else {
				info := prov.GetInfo()
				fmt.Fprintf(w, "%s\t✅ HEALTHY\t%s\n", name, info.Description)
			}
		}

		w.Flush()

		return nil
	},
}

var providerInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize provider configuration",
	Long: `Initialize provider configuration by copying the example file.
This creates a providers.yaml file from providers.yaml.example with default settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

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

		// Ensure .specular directory exists
		if err := os.MkdirAll(filepath.Dir(defaultProviderConfigPath), 0755); err != nil {
			return fmt.Errorf("failed to create .specular directory: %w", err)
		}

		// Write to target file
		if err := os.WriteFile(defaultProviderConfigPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write provider config: %w", err)
		}

		fmt.Printf("✓ Created provider configuration at %s\n", defaultProviderConfigPath)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit .specular/providers.yaml to enable desired providers")
		fmt.Println("  2. Set any required API keys as environment variables")
		fmt.Println("  3. Run 'ai-dev provider health' to check provider status")

		return nil
	},
}

func init() {
	// Add provider command to root
	rootCmd.AddCommand(providerCmd)

	// Add subcommands
	providerCmd.AddCommand(providerListCmd)
	providerCmd.AddCommand(providerHealthCmd)
	providerCmd.AddCommand(providerInitCmd)

	// Flags for list command
	providerListCmd.Flags().String("config", "", "Path to provider config file (default: .specular/providers.yaml)")

	// Flags for health command
	providerHealthCmd.Flags().String("config", "", "Path to provider config file (default: .specular/providers.yaml)")

	// Flags for init command
	providerInitCmd.Flags().Bool("force", false, "Overwrite existing provider config")
}
