package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/specular/internal/ux"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or edit Specular configuration",
	Long: `Manage Specular global configuration stored at ~/.specular/config.yaml

Configuration includes:
  • Default provider preferences
  • Global budget limits
  • Default output format
  • Logging settings
  • API keys and credentials

Examples:
  # View current configuration
  specular config view

  # Edit configuration in $EDITOR
  specular config edit

  # Get a specific value
  specular config get default_provider

  # Set a specific value
  specular config set default_provider ollama

  # Show configuration file path
  specular config path
`,
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "Display current configuration",
	Long:  `Display the current Specular configuration in the specified format.`,
	RunE:  runConfigView,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration in $EDITOR",
	Long:  `Open the configuration file in your default editor (from $EDITOR environment variable).`,
	RunE:  runConfigEdit,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a specific configuration value",
	Long:  `Retrieve the value of a specific configuration key using dot notation (e.g., providers.default).`,
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a specific configuration value",
	Long:  `Set the value of a specific configuration key using dot notation (e.g., providers.default ollama).`,
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	Long:  `Display the path to the global configuration file.`,
	RunE:  runConfigPath,
}

func init() {
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configPathCmd)

	rootCmd.AddCommand(configCmd)
}

// GlobalConfig represents the global Specular configuration
type GlobalConfig struct {
	Providers  ProviderDefaults `yaml:"providers,omitempty"`
	Defaults   CommandDefaults  `yaml:"defaults,omitempty"`
	Budget     BudgetLimits     `yaml:"budget,omitempty"`
	Logging    LoggingConfig    `yaml:"logging,omitempty"`
	Telemetry  TelemetryConfig  `yaml:"telemetry,omitempty"`
}

type ProviderDefaults struct {
	Default    string   `yaml:"default,omitempty"`
	Preference []string `yaml:"preference,omitempty"`
}

type CommandDefaults struct {
	Format      string `yaml:"format,omitempty"`      // "text", "json", "yaml"
	NoColor     bool   `yaml:"no_color,omitempty"`
	Verbose     bool   `yaml:"verbose,omitempty"`
	SpecularDir string `yaml:"specular_dir,omitempty"` // Default .specular
}

type BudgetLimits struct {
	MaxCostPerDay     float64 `yaml:"max_cost_per_day,omitempty"`
	MaxCostPerRequest float64 `yaml:"max_cost_per_request,omitempty"`
	MaxLatencyMs      int     `yaml:"max_latency_ms,omitempty"`
}

type LoggingConfig struct {
	Level      string `yaml:"level,omitempty"`       // "debug", "info", "warn", "error"
	EnableFile bool   `yaml:"enable_file,omitempty"` // Log to file
	LogDir     string `yaml:"log_dir,omitempty"`     // Default ~/.specular/logs
}

type TelemetryConfig struct {
	Enabled    bool `yaml:"enabled,omitempty"`
	ShareUsage bool `yaml:"share_usage,omitempty"`
}

// getConfigPath returns the path to the global configuration file
func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".specular")
	configFile := filepath.Join(configDir, "config.yaml")

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configFile, nil
}

// loadConfig loads the global configuration, creating default if it doesn't exist
func loadConfig() (*GlobalConfig, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// Create default config if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := defaultGlobalConfig()
		if err := saveConfig(defaultConfig, configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return defaultConfig, nil
	}

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config GlobalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// saveConfig saves the configuration to the file
func saveConfig(config *GlobalConfig, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// defaultGlobalConfig returns the default global configuration
func defaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		Providers: ProviderDefaults{
			Default:    "ollama",
			Preference: []string{"ollama", "anthropic", "openai", "gemini"},
		},
		Defaults: CommandDefaults{
			Format:      "text",
			NoColor:     false,
			Verbose:     false,
			SpecularDir: ".specular",
		},
		Budget: BudgetLimits{
			MaxCostPerDay:     20.0,
			MaxCostPerRequest: 1.0,
			MaxLatencyMs:      60000,
		},
		Logging: LoggingConfig{
			Level:      "info",
			EnableFile: true,
			LogDir:     "~/.specular/logs",
		},
		Telemetry: TelemetryConfig{
			Enabled:    false,
			ShareUsage: false,
		},
	}
}

func runConfigView(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	config, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	// Use formatter for JSON/YAML output
	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if err != nil {
			return err
		}
		return formatter.Format(config)
	}

	// Text output
	configPath, _ := getConfigPath()
	fmt.Printf("Configuration file: %s\n\n", configPath)

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return ux.FormatError(err, "getting config path")
	}

	// Ensure config exists
	if _, err := loadConfig(); err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Fallback to vi
	}

	// Open editor
	editorCmd := exec.Command(editor, configPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("failed to run editor: %w", err)
	}

	// Validate the edited config
	if _, err := loadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Configuration may contain errors: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please check and fix the configuration file.\n")
		return err
	}

	fmt.Println("✓ Configuration updated successfully")
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	config, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	value, err := getNestedValue(config, key)
	if err != nil {
		return fmt.Errorf("failed to get value: %w", err)
	}

	fmt.Println(value)
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	config, err := loadConfig()
	if err != nil {
		return ux.FormatError(err, "loading configuration")
	}

	if err := setNestedValue(config, key, value); err != nil {
		return fmt.Errorf("failed to set value: %w", err)
	}

	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	if err := saveConfig(config, configPath); err != nil {
		return ux.FormatError(err, "saving configuration")
	}

	fmt.Printf("✓ Set %s = %s\n", key, value)
	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return ux.FormatError(err, "getting config path")
	}

	fmt.Println(configPath)
	return nil
}

// getNestedValue retrieves a value from the config using dot notation
func getNestedValue(config *GlobalConfig, key string) (string, error) {
	parts := strings.Split(key, ".")

	// Simple key mapping for common values
	switch strings.Join(parts, ".") {
	case "providers.default":
		return config.Providers.Default, nil
	case "defaults.format":
		return config.Defaults.Format, nil
	case "defaults.no_color":
		return fmt.Sprintf("%t", config.Defaults.NoColor), nil
	case "defaults.verbose":
		return fmt.Sprintf("%t", config.Defaults.Verbose), nil
	case "defaults.specular_dir":
		return config.Defaults.SpecularDir, nil
	case "budget.max_cost_per_day":
		return fmt.Sprintf("%.2f", config.Budget.MaxCostPerDay), nil
	case "budget.max_cost_per_request":
		return fmt.Sprintf("%.2f", config.Budget.MaxCostPerRequest), nil
	case "budget.max_latency_ms":
		return fmt.Sprintf("%d", config.Budget.MaxLatencyMs), nil
	case "logging.level":
		return config.Logging.Level, nil
	case "logging.enable_file":
		return fmt.Sprintf("%t", config.Logging.EnableFile), nil
	case "logging.log_dir":
		return config.Logging.LogDir, nil
	case "telemetry.enabled":
		return fmt.Sprintf("%t", config.Telemetry.Enabled), nil
	case "telemetry.share_usage":
		return fmt.Sprintf("%t", config.Telemetry.ShareUsage), nil
	default:
		return "", fmt.Errorf("unknown configuration key: %s", key)
	}
}

// setNestedValue sets a value in the config using dot notation
func setNestedValue(config *GlobalConfig, key, value string) error {
	parts := strings.Split(key, ".")

	// Simple key mapping for common values
	switch strings.Join(parts, ".") {
	case "providers.default":
		config.Providers.Default = value
	case "defaults.format":
		config.Defaults.Format = value
	case "defaults.no_color":
		config.Defaults.NoColor = parseBool(value)
	case "defaults.verbose":
		config.Defaults.Verbose = parseBool(value)
	case "defaults.specular_dir":
		config.Defaults.SpecularDir = value
	case "budget.max_cost_per_day":
		if v, err := parseFloat(value); err == nil {
			config.Budget.MaxCostPerDay = v
		} else {
			return err
		}
	case "budget.max_cost_per_request":
		if v, err := parseFloat(value); err == nil {
			config.Budget.MaxCostPerRequest = v
		} else {
			return err
		}
	case "budget.max_latency_ms":
		if v, err := parseInt(value); err == nil {
			config.Budget.MaxLatencyMs = v
		} else {
			return err
		}
	case "logging.level":
		config.Logging.Level = value
	case "logging.enable_file":
		config.Logging.EnableFile = parseBool(value)
	case "logging.log_dir":
		config.Logging.LogDir = value
	case "telemetry.enabled":
		config.Telemetry.Enabled = parseBool(value)
	case "telemetry.share_usage":
		config.Telemetry.ShareUsage = parseBool(value)
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return nil
}

// Helper functions for parsing values
func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "yes" || s == "1"
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
