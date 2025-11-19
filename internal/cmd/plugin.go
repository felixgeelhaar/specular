package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/plugin"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage Specular plugins",
	Long: `Manage plugins that extend Specular functionality.

Plugins can provide:
  - AI providers (custom model integrations)
  - Validators (policy validation)
  - Formatters (output formatting)
  - Hooks (event handlers)
  - Notifiers (notifications)`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long:  `List all installed plugins and their current status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := plugin.NewManager(plugin.DefaultManagerConfig())

		// Discover plugins
		if err := manager.Discover(); err != nil {
			return fmt.Errorf("failed to discover plugins: %w", err)
		}

		plugins := manager.List()

		if len(plugins) == 0 {
			fmt.Println("No plugins installed.")
			fmt.Println("\nPlugin directories searched:")
			config := plugin.DefaultManagerConfig()
			for _, dir := range config.PluginDirs {
				fmt.Printf("  - %s\n", dir)
			}
			fmt.Println("\nTo install a plugin, run: specular plugin install <source>")
			return nil
		}

		// Print plugins table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tTYPE\tSTATE\tDESCRIPTION") //nolint:errcheck
		fmt.Fprintln(w, "----\t-------\t----\t-----\t-----------") //nolint:errcheck

		for _, p := range plugins {
			description := p.Manifest.Description
			if len(description) > 40 {
				description = description[:37] + "..."
			}
			if p.State == plugin.PluginStateError {
				description = fmt.Sprintf("ERROR: %s", p.Error)
				if len(description) > 40 {
					description = description[:37] + "..."
				}
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", //nolint:errcheck
				p.Manifest.Name,
				p.Manifest.Version,
				p.Manifest.Type,
				p.State,
				description)
		}

		w.Flush() //#nosec G104 -- Tabwriter flush errors not critical

		return nil
	},
}

var pluginInfoCmd = &cobra.Command{
	Use:   "info <plugin-name>",
	Short: "Show plugin information",
	Long:  `Show detailed information about a specific plugin.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]

		manager := plugin.NewManager(plugin.DefaultManagerConfig())

		// Discover plugins
		if err := manager.Discover(); err != nil {
			return fmt.Errorf("failed to discover plugins: %w", err)
		}

		p, ok := manager.Get(pluginName)
		if !ok {
			return fmt.Errorf("plugin not found: %s", pluginName)
		}

		// Print plugin details
		fmt.Printf("Name:        %s\n", p.Manifest.Name)
		fmt.Printf("Version:     %s\n", p.Manifest.Version)
		fmt.Printf("Type:        %s\n", p.Manifest.Type)
		fmt.Printf("State:       %s\n", p.State)
		fmt.Printf("Path:        %s\n", p.Path)
		fmt.Printf("Loaded At:   %s\n", p.LoadedAt.Format(time.RFC3339))

		if p.Manifest.Description != "" {
			fmt.Printf("Description: %s\n", p.Manifest.Description)
		}
		if p.Manifest.Author != "" {
			fmt.Printf("Author:      %s\n", p.Manifest.Author)
		}
		if p.Manifest.License != "" {
			fmt.Printf("License:     %s\n", p.Manifest.License)
		}
		if p.Manifest.Homepage != "" {
			fmt.Printf("Homepage:    %s\n", p.Manifest.Homepage)
		}
		if p.Manifest.Entrypoint != "" {
			fmt.Printf("Entrypoint:  %s\n", p.Manifest.Entrypoint)
		}
		if p.Manifest.MinSpecularVersion != "" {
			fmt.Printf("Min Version: %s\n", p.Manifest.MinSpecularVersion)
		}

		if len(p.Manifest.Capabilities) > 0 {
			fmt.Println("\nCapabilities:")
			for _, cap := range p.Manifest.Capabilities {
				fmt.Printf("  - %s\n", cap)
			}
		}

		if len(p.Manifest.Config) > 0 {
			fmt.Println("\nConfiguration Options:")
			for _, cfg := range p.Manifest.Config {
				required := ""
				if cfg.Required {
					required = " (required)"
				}
				fmt.Printf("  %s: %s%s\n", cfg.Name, cfg.Type, required)
				if cfg.Description != "" {
					fmt.Printf("    %s\n", cfg.Description)
				}
			}
		}

		if p.State == plugin.PluginStateError {
			fmt.Printf("\nError: %s\n", p.Error)
		}

		return nil
	},
}

var pluginHealthCmd = &cobra.Command{
	Use:   "health <plugin-name>",
	Short: "Check plugin health",
	Long:  `Check the health status of a plugin.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]

		manager := plugin.NewManager(plugin.DefaultManagerConfig())

		// Discover plugins
		if err := manager.Discover(); err != nil {
			return fmt.Errorf("failed to discover plugins: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		health, err := manager.Health(ctx, pluginName)
		if err != nil {
			fmt.Printf("❌ Plugin %s is unhealthy: %v\n", pluginName, err)
			return nil
		}

		fmt.Printf("✅ Plugin %s is healthy\n", pluginName)
		fmt.Printf("   Status:  %s\n", health.Status)
		fmt.Printf("   Version: %s\n", health.Version)

		return nil
	},
}

var pluginEnableCmd = &cobra.Command{
	Use:   "enable <plugin-name>",
	Short: "Enable a plugin",
	Long:  `Enable an installed plugin so it can be used by Specular.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]

		manager := plugin.NewManager(plugin.DefaultManagerConfig())

		// Discover plugins
		if err := manager.Discover(); err != nil {
			return fmt.Errorf("failed to discover plugins: %w", err)
		}

		if err := manager.Enable(pluginName); err != nil {
			return fmt.Errorf("failed to enable plugin: %w", err)
		}

		fmt.Printf("✓ Plugin %s enabled\n", pluginName)
		return nil
	},
}

var pluginDisableCmd = &cobra.Command{
	Use:   "disable <plugin-name>",
	Short: "Disable a plugin",
	Long:  `Disable an installed plugin without removing it.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]

		manager := plugin.NewManager(plugin.DefaultManagerConfig())

		// Discover plugins
		if err := manager.Discover(); err != nil {
			return fmt.Errorf("failed to discover plugins: %w", err)
		}

		if err := manager.Disable(pluginName); err != nil {
			return fmt.Errorf("failed to disable plugin: %w", err)
		}

		fmt.Printf("✓ Plugin %s disabled\n", pluginName)
		return nil
	},
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install <source>",
	Short: "Install a plugin",
	Long: `Install a plugin from a source.

Sources can be:
  - Local directory path
  - GitHub repository URL (e.g., github.com/user/repo)
  - Plugin registry name (coming soon)

Examples:
  specular plugin install ./my-plugin
  specular plugin install github.com/felixgeelhaar/specular-slack-notifier`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]

		manager := plugin.NewManager(plugin.DefaultManagerConfig())

		if err := manager.Install(source); err != nil {
			return fmt.Errorf("failed to install plugin: %w", err)
		}

		fmt.Printf("✓ Plugin installed from %s\n", source)
		return nil
	},
}

var pluginUninstallCmd = &cobra.Command{
	Use:     "uninstall <plugin-name>",
	Aliases: []string{"remove", "rm"},
	Short:   "Uninstall a plugin",
	Long:    `Uninstall a plugin and remove its files.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]

		manager := plugin.NewManager(plugin.DefaultManagerConfig())

		// Discover plugins
		if err := manager.Discover(); err != nil {
			return fmt.Errorf("failed to discover plugins: %w", err)
		}

		// Confirm uninstall
		force := cmd.Flags().Lookup("force").Value.String() == "true"
		if !force {
			fmt.Printf("Are you sure you want to uninstall plugin '%s'? This will delete all plugin files.\n", pluginName)
			fmt.Print("Type 'yes' to confirm: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("Uninstall cancelled.")
				return nil
			}
		}

		if err := manager.Uninstall(pluginName); err != nil {
			return fmt.Errorf("failed to uninstall plugin: %w", err)
		}

		fmt.Printf("✓ Plugin %s uninstalled\n", pluginName)
		return nil
	},
}

func init() {
	// Add plugin command to root
	rootCmd.AddCommand(pluginCmd)

	// Add subcommands
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInfoCmd)
	pluginCmd.AddCommand(pluginHealthCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginUninstallCmd)

	// Flags for uninstall command
	pluginUninstallCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}
