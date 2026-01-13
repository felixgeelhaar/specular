package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

var (
	installForce   bool
	installUpgrade bool
	installVersion string
)

var pluginInstallCmd = &cobra.Command{
	Use:   "install <source>",
	Short: "Install a plugin",
	Long: `Install a plugin from a source.

Sources can be:
  - Local directory path (e.g., ./my-plugin)
  - GitHub repository URL (e.g., github.com/user/repo)
  - GitHub with version (e.g., github.com/user/repo@v1.0.0)
  - Plugin registry name (e.g., registry:plugin-name@1.0.0)

Examples:
  specular plugin install ./my-plugin
  specular plugin install github.com/felixgeelhaar/specular-slack-notifier
  specular plugin install github.com/felixgeelhaar/specular-slack-notifier@v1.2.0
  specular plugin install github.com/user/monorepo@main/plugins/my-plugin`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]

		manager := plugin.NewManager(plugin.DefaultManagerConfig())

		opts := plugin.InstallOptions{
			Force:   installForce,
			Upgrade: installUpgrade,
			Version: installVersion,
		}

		if err := manager.InstallWithOptions(source, opts); err != nil {
			return fmt.Errorf("failed to install plugin: %w", err)
		}

		return nil
	},
}

var pluginUpdateCmd = &cobra.Command{
	Use:   "update [plugin-name]",
	Short: "Update installed plugins",
	Long: `Update one or all installed plugins to their latest versions.

If a plugin name is provided, only that plugin is updated.
If no arguments are given, all plugins are updated.

Version can be specified with @version suffix:
  specular plugin update slack-notifier@v2.0.0

Examples:
  specular plugin update                    # Update all plugins
  specular plugin update slack-notifier     # Update specific plugin
  specular plugin update slack-notifier@v2.0.0  # Update to specific version`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := plugin.NewManager(plugin.DefaultManagerConfig())

		if err := manager.Discover(); err != nil {
			return fmt.Errorf("failed to discover plugins: %w", err)
		}

		if len(args) == 0 {
			// Update all plugins
			fmt.Println("Updating all plugins...")
			results, err := manager.UpdateAll()
			if err != nil {
				return fmt.Errorf("failed to update plugins: %w", err)
			}

			var updated, failed int
			for _, r := range results {
				if r.Error != nil {
					fmt.Printf("✗ %s: %v\n", r.Name, r.Error)
					failed++
				} else if r.Updated {
					fmt.Printf("✓ %s: %s → %s\n", r.Name, r.OldVersion, r.NewVersion)
					updated++
				} else {
					fmt.Printf("  %s: already at latest (%s)\n", r.Name, r.OldVersion)
				}
			}

			fmt.Printf("\nUpdated %d plugin(s), %d failed\n", updated, failed)
			return nil
		}

		// Update specific plugin
		pluginSpec := args[0]
		name := pluginSpec
		version := ""

		// Parse version from plugin@version format
		if idx := len(pluginSpec) - 1; idx > 0 {
			for i := len(pluginSpec) - 1; i >= 0; i-- {
				if pluginSpec[i] == '@' {
					name = pluginSpec[:i]
					version = pluginSpec[i+1:]
					break
				}
			}
		}

		result, err := manager.Update(name, version)
		if err != nil {
			return fmt.Errorf("failed to update plugin: %w", err)
		}

		if result.Updated {
			fmt.Printf("✓ Updated %s: %s → %s\n", name, result.OldVersion, result.NewVersion)
		} else {
			fmt.Printf("  %s: already at latest (%s)\n", name, result.OldVersion)
		}

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
	pluginCreateLang   string
)

var pluginCreateCmd = &cobra.Command{
	Use:     "create <name>",
	Aliases: []string{"init", "new"},
	Short:   "Create a new plugin scaffold",
	Long: `Create a new plugin directory with manifest and entrypoint template.

This generates a plugin structure in your chosen language:
  - plugin.yaml: Plugin manifest with metadata
  - Language-specific entrypoint (main.go, main.py, index.js, or entrypoint.sh)
  - Build/dependency files (go.mod, requirements.txt, package.json)

Supported languages: go, python, node, shell

Examples:
  # Create a Go provider plugin
  specular plugin create my-provider --type provider --lang go

  # Create a Python notifier plugin
  specular plugin create slack-notifier --type notifier --lang python

  # Create a Node.js validator plugin
  specular plugin create json-validator --type validator --lang node

  # Create a shell hook plugin (default)
  specular plugin create pre-commit-hook --type hook`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]

		// Validate plugin type
		validTypes := GetPluginTypes()
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

		// Validate language
		validLangs := plugin.GetSupportedLanguages()
		langValid := false
		for _, l := range validLangs {
			if pluginCreateLang == l {
				langValid = true
				break
			}
		}
		if !langValid {
			return fmt.Errorf("invalid language: %s (valid: %v)", pluginCreateLang, validLangs)
		}

		// Create plugin directory
		pluginDir := filepath.Join(".", pluginName)
		if _, err := os.Stat(pluginDir); err == nil {
			return fmt.Errorf("directory already exists: %s", pluginDir)
		}

		// Use scaffold to create plugin
		cfg := plugin.ScaffoldConfig{
			Name:    pluginName,
			Type:    pluginCreateType,
			Lang:    pluginCreateLang,
			Author:  pluginCreateAuthor,
			Version: "0.1.0",
		}

		if err := plugin.Scaffold(pluginDir, cfg); err != nil {
			return fmt.Errorf("failed to scaffold plugin: %w", err)
		}

		fmt.Printf("✓ Created %s plugin: %s\n\n", pluginCreateLang, pluginDir)
		plugin.PrintNextSteps(pluginName, pluginCreateLang)

		return nil
	},
}

var (
	searchType   string
	searchLimit  int
	registryURL  string
	clearCache   bool
)

var pluginSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the plugin registry",
	Long: `Search for plugins in the Specular plugin registry.

If no query is provided, lists all available plugins.
Results are sorted by relevance and popularity.

Examples:
  specular plugin search                    # List all plugins
  specular plugin search slack              # Search by name/keyword
  specular plugin search --type notifier    # Filter by type
  specular plugin search notifications      # Search by description`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmdCtx, err := NewCommandContext(cmd)
		if err != nil {
			return fmt.Errorf("failed to create command context: %w", err)
		}

		// Initialize registry
		opts := []plugin.RegistryOption{}
		if registryURL != "" {
			opts = append(opts, plugin.WithRegistryURL(registryURL))
		}
		registry := plugin.NewRegistry(opts...)

		if clearCache {
			if err := registry.ClearCache(); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Warning: failed to clear cache: %v\n", err)
			}
		}

		// Perform search
		query := ""
		if len(args) > 0 {
			query = args[0]
		}

		var results []plugin.SearchResult

		if searchType != "" {
			// Search by type
			pluginType := plugin.PluginType(searchType)
			results, err = registry.SearchByType(pluginType)
			if err != nil {
				return fmt.Errorf("failed to search registry: %w", err)
			}
			// If query also provided, filter results
			if query != "" {
				var filtered []plugin.SearchResult
				queryLower := strings.ToLower(query)
				for _, r := range results {
					if strings.Contains(strings.ToLower(r.Plugin.Name), queryLower) ||
						strings.Contains(strings.ToLower(r.Plugin.Description), queryLower) {
						filtered = append(filtered, r)
					}
				}
				results = filtered
			}
		} else {
			results, err = registry.Search(query)
			if err != nil {
				return fmt.Errorf("failed to search registry: %w", err)
			}
		}

		// Apply limit
		if searchLimit > 0 && len(results) > searchLimit {
			results = results[:searchLimit]
		}

		// JSON/YAML output
		if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
			formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
				NoColor: cmdCtx.NoColor,
			})
			if err != nil {
				return err
			}
			return formatter.Format(results)
		}

		if len(results) == 0 {
			fmt.Println("No plugins found.")
			return nil
		}

		// Print results table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tTYPE\tDOWNLOADS\tDESCRIPTION") //nolint:errcheck
		fmt.Fprintln(w, "----\t-------\t----\t---------\t-----------") //nolint:errcheck

		for _, r := range results {
			description := r.Plugin.Description
			if len(description) > 50 {
				description = description[:47] + "..."
			}

			deprecatedMark := ""
			if r.Plugin.Deprecated {
				deprecatedMark = " [DEPRECATED]"
			}

			fmt.Fprintf(w, "%s%s\t%s\t%s\t%d\t%s\n", //nolint:errcheck
				r.Plugin.Name,
				deprecatedMark,
				r.Plugin.Latest,
				r.Plugin.Type,
				r.Plugin.Downloads,
				description)
		}

		w.Flush() //#nosec G104 -- Tabwriter flush errors not critical

		fmt.Printf("\nFound %d plugin(s). Use 'specular plugin registry-info <name>' for details.\n", len(results))

		return nil
	},
}

var pluginRegistryInfoCmd = &cobra.Command{
	Use:     "registry-info <plugin-name>",
	Aliases: []string{"reg-info"},
	Short:   "Show registry information about a plugin",
	Long: `Show detailed registry information about a plugin.

This shows the plugin's registry entry including:
  - All available versions
  - Author and license information
  - Download statistics
  - Keywords and dependencies

Examples:
  specular plugin registry-info slack-notifier`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmdCtx, err := NewCommandContext(cmd)
		if err != nil {
			return fmt.Errorf("failed to create command context: %w", err)
		}

		pluginName := args[0]

		// Initialize registry
		opts := []plugin.RegistryOption{}
		if registryURL != "" {
			opts = append(opts, plugin.WithRegistryURL(registryURL))
		}
		registry := plugin.NewRegistry(opts...)

		// Get plugin info
		info, err := registry.GetPluginInfo(pluginName)
		if err != nil {
			return fmt.Errorf("failed to get plugin info: %w", err)
		}

		// JSON/YAML output
		if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
			formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
				NoColor: cmdCtx.NoColor,
			})
			if err != nil {
				return err
			}
			return formatter.Format(info)
		}

		// Print plugin details
		fmt.Printf("Name:        %s\n", info.Name)
		fmt.Printf("Type:        %s\n", info.Type)
		fmt.Printf("Latest:      %s\n", info.Latest)
		fmt.Printf("Author:      %s\n", info.Author)
		if info.License != "" {
			fmt.Printf("License:     %s\n", info.License)
		}
		if info.Repository != "" {
			fmt.Printf("Repository:  %s\n", info.Repository)
		}
		if info.Homepage != "" {
			fmt.Printf("Homepage:    %s\n", info.Homepage)
		}
		fmt.Printf("Downloads:   %d\n", info.Downloads)
		fmt.Printf("Stars:       %d\n", info.Stars)

		if info.Description != "" {
			fmt.Printf("\nDescription:\n  %s\n", info.Description)
		}

		if len(info.Keywords) > 0 {
			fmt.Printf("\nKeywords:\n  %s\n", strings.Join(info.Keywords, ", "))
		}

		if info.Deprecated {
			fmt.Printf("\n⚠️  This plugin is DEPRECATED\n")
		}

		if len(info.Versions) > 0 {
			fmt.Println("\nAvailable Versions:")
			// Show up to 10 most recent versions
			count := len(info.Versions)
			if count > 10 {
				count = 10
			}
			for i := 0; i < count; i++ {
				v := info.Versions[i]
				marker := ""
				if v == info.Latest {
					marker = " (latest)"
				}
				fmt.Printf("  - %s%s\n", v, marker)
			}
			if len(info.Versions) > 10 {
				fmt.Printf("  ... and %d more\n", len(info.Versions)-10)
			}
		}

		fmt.Printf("\nInstall:\n  specular plugin install registry:%s\n", info.Name)
		fmt.Printf("  specular plugin install registry:%s@%s\n", info.Name, info.Latest)

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
	pluginCmd.AddCommand(pluginUpdateCmd)
	pluginCmd.AddCommand(pluginCreateCmd)
	pluginCmd.AddCommand(pluginSearchCmd)
	pluginCmd.AddCommand(pluginRegistryInfoCmd)

	// Flags for install command
	pluginInstallCmd.Flags().BoolVarP(&installForce, "force", "f", false, "Overwrite existing plugin")
	pluginInstallCmd.Flags().BoolVarP(&installUpgrade, "upgrade", "u", false, "Upgrade existing plugin")
	pluginInstallCmd.Flags().StringVar(&installVersion, "version", "", "Specific version to install")

	// Flags for uninstall command
	pluginUninstallCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	// Flags for create command
	pluginCreateCmd.Flags().StringVar(&pluginCreateType, "type", "provider", "Plugin type (provider, validator, formatter, hook, notifier)")
	pluginCreateCmd.Flags().StringVar(&pluginCreateAuthor, "author", "", "Plugin author name")
	pluginCreateCmd.Flags().StringVar(&pluginCreateLang, "lang", "shell", "Template language (go, python, node, shell)")

	// Flags for search command
	pluginSearchCmd.Flags().StringVar(&searchType, "type", "", "Filter by plugin type (provider, validator, formatter, hook, notifier)")
	pluginSearchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 0, "Limit number of results (0 = unlimited)")
	pluginSearchCmd.Flags().StringVar(&registryURL, "registry", "", "Custom registry URL")
	pluginSearchCmd.Flags().BoolVar(&clearCache, "clear-cache", false, "Clear registry cache before searching")

	// Flags for registry-info command
	pluginRegistryInfoCmd.Flags().StringVar(&registryURL, "registry", "", "Custom registry URL")
}
