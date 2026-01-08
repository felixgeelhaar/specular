package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/plugin"
	"github.com/felixgeelhaar/specular/internal/ux"
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
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed plugins",
	Long:    `List all installed plugins and their current status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmdCtx, err := NewCommandContext(cmd)
		if err != nil {
			return fmt.Errorf("failed to create command context: %w", err)
		}

		manager := plugin.NewManager(plugin.DefaultManagerConfig())

		// Discover plugins
		if err := manager.Discover(); err != nil {
			return fmt.Errorf("failed to discover plugins: %w", err)
		}

		plugins := manager.List()

		// JSON/YAML output
		if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
			formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
				NoColor: cmdCtx.NoColor,
			})
			if err != nil {
				return err
			}
			return formatter.Format(plugins)
		}

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
	Use:     "info <plugin-name>",
	Aliases: []string{"show"},
	Short:   "Show plugin information",
	Long:    `Show detailed information about a specific plugin.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmdCtx, err := NewCommandContext(cmd)
		if err != nil {
			return fmt.Errorf("failed to create command context: %w", err)
		}

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

		// JSON/YAML output
		if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
			formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
				NoColor: cmdCtx.NoColor,
			})
			if err != nil {
				return err
			}
			return formatter.Format(p)
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

var (
	pluginCreateType   string
	pluginCreateAuthor string
)

var pluginCreateCmd = &cobra.Command{
	Use:     "create <name>",
	Aliases: []string{"init", "new"},
	Short:   "Create a new plugin scaffold",
	Long: `Create a new plugin directory with manifest and entrypoint template.

This generates a basic plugin structure that you can customize:
  - plugin.yaml: Plugin manifest with metadata
  - entrypoint.sh: Shell script entrypoint (or .py for Python)
  - README.md: Basic documentation

Examples:
  # Create a provider plugin
  specular plugin create my-provider --type provider

  # Create a notifier plugin with author info
  specular plugin create slack-notifier --type notifier --author "Your Name"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]

		// Validate plugin type
		validTypes := []string{"provider", "validator", "formatter", "hook", "notifier"}
		typeValid := false
		for _, t := range validTypes {
			if pluginCreateType == t {
				typeValid = true
				break
			}
		}
		if !typeValid {
			return fmt.Errorf("invalid plugin type: %s (valid: %v)", pluginCreateType, validTypes)
		}

		// Create plugin directory
		pluginDir := filepath.Join(".", pluginName)
		if _, err := os.Stat(pluginDir); err == nil {
			return fmt.Errorf("directory already exists: %s", pluginDir)
		}

		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Create plugin.yaml manifest
		manifest := fmt.Sprintf(`# Specular Plugin Manifest
name: %s
version: "0.1.0"
type: %s
description: "A Specular %s plugin"
author: "%s"
license: "MIT"
entrypoint: "./entrypoint.sh"

# Capabilities this plugin provides
capabilities:
  - %s

# Configuration options (optional)
config: []
`, pluginName, pluginCreateType, pluginCreateType, pluginCreateAuthor, pluginCreateType)

		if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644); err != nil {
			return fmt.Errorf("failed to create manifest: %w", err)
		}

		// Create entrypoint script
		entrypoint := fmt.Sprintf(`#!/bin/bash
# Specular Plugin: %s
# Type: %s
#
# This plugin receives JSON input on stdin and outputs JSON on stdout.
# Input format: { "action": "...", "data": {...} }
# Output format: { "success": true, "result": {...} } or { "success": false, "error": "..." }

set -e

# Read input
INPUT=$(cat)
ACTION=$(echo "$INPUT" | jq -r '.action // "execute"')

case "$ACTION" in
  "health")
    # Health check
    echo '{"success": true, "result": {"status": "healthy", "version": "0.1.0"}}'
    ;;
  "execute")
    # Main execution logic
    # TODO: Implement your plugin logic here
    echo '{"success": true, "result": {"message": "Plugin executed successfully"}}'
    ;;
  *)
    echo "{\"success\": false, \"error\": \"Unknown action: $ACTION\"}"
    exit 1
    ;;
esac
`, pluginName, pluginCreateType)

		entrypointPath := filepath.Join(pluginDir, "entrypoint.sh")
		if err := os.WriteFile(entrypointPath, []byte(entrypoint), 0755); err != nil {
			return fmt.Errorf("failed to create entrypoint: %w", err)
		}

		// Create README.md
		readme := fmt.Sprintf(`# %s

A Specular %s plugin.

## Installation

`+"```bash"+`
specular plugin install ./%s
`+"```"+`

## Usage

After installation, this plugin will be available for use with Specular.

## Configuration

Edit `+"`plugin.yaml`"+` to configure the plugin options.

## Development

1. Edit `+"`entrypoint.sh`"+` to implement your plugin logic
2. Test locally: `+"`echo '{}' | ./entrypoint.sh`"+`
3. Install: `+"`specular plugin install ./%s`"+`

## License

MIT
`, pluginName, pluginCreateType, pluginName, pluginName)

		if err := os.WriteFile(filepath.Join(pluginDir, "README.md"), []byte(readme), 0644); err != nil {
			return fmt.Errorf("failed to create README: %w", err)
		}

		fmt.Printf("✓ Created plugin scaffold: %s\n", pluginDir)
		fmt.Printf("\nFiles created:\n")
		fmt.Printf("  %s/plugin.yaml    - Plugin manifest\n", pluginName)
		fmt.Printf("  %s/entrypoint.sh  - Plugin entrypoint script\n", pluginName)
		fmt.Printf("  %s/README.md      - Documentation\n", pluginName)
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Edit %s/entrypoint.sh to implement your plugin logic\n", pluginName)
		fmt.Printf("  2. Test: echo '{}' | ./%s/entrypoint.sh\n", pluginName)
		fmt.Printf("  3. Install: specular plugin install ./%s\n", pluginName)

		return nil
	},
}

// GetPluginTypes returns valid plugin types for shell completion
func GetPluginTypes() []string {
	return []string{"provider", "validator", "formatter", "hook", "notifier"}
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
	pluginCmd.AddCommand(pluginCreateCmd)

	// Flags for uninstall command
	pluginUninstallCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	// Flags for create command
	pluginCreateCmd.Flags().StringVar(&pluginCreateType, "type", "provider", "Plugin type (provider, validator, formatter, hook, notifier)")
	pluginCreateCmd.Flags().StringVar(&pluginCreateAuthor, "author", "", "Plugin author name")
}
