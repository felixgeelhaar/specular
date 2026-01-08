package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/specular/internal/slo"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"c", "cfg"},
	Short:   "View or edit Specular configuration",
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
	Providers         ProviderDefaults        `yaml:"providers,omitempty"`
	Defaults          CommandDefaults         `yaml:"defaults,omitempty"`
	Budget            BudgetLimits            `yaml:"budget,omitempty"`
	Logging           LoggingConfig           `yaml:"logging,omitempty"`
	Telemetry         TelemetryConfig         `yaml:"telemetry,omitempty"`
	Vault             VaultConfig             `yaml:"vault,omitempty"`
	AWSSecretsManager AWSSecretsManagerConfig `yaml:"aws_secrets_manager,omitempty"`
	Observability     ObservabilityConfig     `yaml:"observability,omitempty"`
}

// VaultConfig holds HashiCorp Vault integration settings for secrets management
// and cryptographic signing operations.
type VaultConfig struct {
	// Enabled controls whether Vault integration is active
	Enabled bool `yaml:"enabled,omitempty"`

	// Address is the Vault server URL (e.g., "https://vault.example.com:8200")
	Address string `yaml:"address,omitempty"`

	// MountPath is the KV v2 secrets engine mount path (default: "secret")
	MountPath string `yaml:"mount_path,omitempty"`

	// Namespace is the Vault namespace (Enterprise feature, optional)
	Namespace string `yaml:"namespace,omitempty"`

	// SigningKeyPath is the path within Vault where the signing key is stored
	// Example: "specular/audit/signing-key"
	SigningKeyPath string `yaml:"signing_key_path,omitempty"`

	// SignerIdentity is the identity used for audit log signatures
	// Example: "system@specular.dev"
	SignerIdentity string `yaml:"signer_identity,omitempty"`

	// AutoGenerateKey will create a new signing key if one doesn't exist
	AutoGenerateKey bool `yaml:"auto_generate_key,omitempty"`

	// TLS configuration for Vault connection
	TLS VaultTLSConfig `yaml:"tls,omitempty"`
}

// VaultTLSConfig holds TLS/mTLS settings for Vault connection.
type VaultTLSConfig struct {
	// CACert is the path to the CA certificate file
	CACert string `yaml:"ca_cert,omitempty"`

	// CAPath is the path to a directory of CA certificates
	CAPath string `yaml:"ca_path,omitempty"`

	// ClientCert is the path to the client certificate (for mTLS)
	ClientCert string `yaml:"client_cert,omitempty"`

	// ClientKey is the path to the client private key (for mTLS)
	ClientKey string `yaml:"client_key,omitempty"`

	// ServerName is the server name for SNI verification
	ServerName string `yaml:"server_name,omitempty"`

	// InsecureSkipVerify disables certificate verification (NOT for production)
	InsecureSkipVerify bool `yaml:"insecure_skip_verify,omitempty"`
}

// AWSSecretsManagerConfig holds AWS Secrets Manager integration settings for
// secrets management and cryptographic signing operations.
type AWSSecretsManagerConfig struct {
	// Enabled controls whether AWS Secrets Manager integration is active
	Enabled bool `yaml:"enabled,omitempty"`

	// Region is the primary AWS region (e.g., "us-west-2")
	Region string `yaml:"region,omitempty"`

	// SecondaryRegion is the DR/failover region (optional)
	SecondaryRegion string `yaml:"secondary_region,omitempty"`

	// Profile is the AWS profile name from ~/.aws/credentials (optional)
	Profile string `yaml:"profile,omitempty"`

	// AssumeRoleARN is the ARN of a role to assume for cross-account access (optional)
	AssumeRoleARN string `yaml:"assume_role_arn,omitempty"`

	// SigningKeyName is the secret name where the signing key is stored
	// Example: "specular/audit/signing-key"
	SigningKeyName string `yaml:"signing_key_name,omitempty"`

	// SignerIdentity is the identity used for audit log signatures
	// Example: "system@specular.dev"
	SignerIdentity string `yaml:"signer_identity,omitempty"`

	// AutoGenerateKey will create a new signing key if one doesn't exist
	AutoGenerateKey bool `yaml:"auto_generate_key,omitempty"`

	// Endpoint is a custom endpoint URL for LocalStack testing (optional)
	Endpoint string `yaml:"endpoint,omitempty"`

	// Tags are applied to secrets when creating new keys
	Tags map[string]string `yaml:"tags,omitempty"`
}

// ProviderDefaults specifies the default and preferred AI providers for command
// execution in order of user preference.
type ProviderDefaults struct {
	Default    string   `yaml:"default,omitempty"`
	Preference []string `yaml:"preference,omitempty"`
}

// CommandDefaults provides default options for all Specular commands including
// output format, color settings, verbosity, and workspace directory location.
type CommandDefaults struct {
	Format      string `yaml:"format,omitempty"` // "text", "json", "yaml"
	NoColor     bool   `yaml:"no_color,omitempty"`
	Verbose     bool   `yaml:"verbose,omitempty"`
	SpecularDir string `yaml:"specular_dir,omitempty"` // Default .specular
}

// BudgetLimits enforces spending and performance constraints on AI provider
// usage, including daily costs, per-request costs, and latency thresholds.
type BudgetLimits struct {
	MaxCostPerDay     float64 `yaml:"max_cost_per_day,omitempty"`
	MaxCostPerRequest float64 `yaml:"max_cost_per_request,omitempty"`
	MaxLatencyMs      int     `yaml:"max_latency_ms,omitempty"`
}

// LoggingConfig defines logging behavior including verbosity level, file output
// options, and log directory location.
type LoggingConfig struct {
	Level      string `yaml:"level,omitempty"`       // "debug", "info", "warn", "error"
	EnableFile bool   `yaml:"enable_file,omitempty"` // Log to file
	LogDir     string `yaml:"log_dir,omitempty"`     // Default ~/.specular/logs
}

// TelemetryConfig controls telemetry collection and reporting behavior,
// including whether usage data is collected and sent to an external endpoint.
type TelemetryConfig struct {
	Enabled    bool    `yaml:"enabled,omitempty"`
	ShareUsage bool    `yaml:"share_usage,omitempty"`
	Endpoint   string  `yaml:"endpoint,omitempty"`
	SampleRate float64 `yaml:"sample_rate,omitempty"`
}

// ObservabilityConfig holds configuration for the observability stack including
// SLO tracking, alerting integrations, and log aggregation.
type ObservabilityConfig struct {
	// SLO configuration for Service Level Objectives tracking
	SLO SLOConfig `yaml:"slo,omitempty"`

	// Alerting configuration for alert routing
	Alerting AlertingConfig `yaml:"alerting,omitempty"`

	// LogExport configuration for log aggregation
	LogExport LogExportConfig `yaml:"log_export,omitempty"`
}

// SLOConfig holds Service Level Objective tracking configuration.
type SLOConfig struct {
	// Enabled controls whether SLO tracking is active
	Enabled bool `yaml:"enabled,omitempty"`

	// ConfigPath is the path to an external SLO definitions file
	ConfigPath string `yaml:"config_path,omitempty"`

	// DefaultWindow is the default SLO evaluation window (e.g., "30d")
	DefaultWindow slo.Duration `yaml:"default_window,omitempty"`

	// CacheTTL is how long to cache SLO status calculations
	CacheTTL slo.Duration `yaml:"cache_ttl,omitempty"`

	// SLOs are inline SLO definitions (alternative to ConfigPath)
	SLOs []*slo.SLO `yaml:"slos,omitempty"`
}

// AlertingConfig holds alert routing configuration for incident management.
type AlertingConfig struct {
	// Enabled controls whether alerting is active
	Enabled bool `yaml:"enabled,omitempty"`

	// DefaultSeverity is the default alert severity if not specified
	DefaultSeverity string `yaml:"default_severity,omitempty"`

	// PagerDuty configuration for PagerDuty integration
	PagerDuty PagerDutyConfig `yaml:"pagerduty,omitempty"`

	// Opsgenie configuration for Opsgenie integration
	Opsgenie OpsgenieConfig `yaml:"opsgenie,omitempty"`

	// Slack configuration for Slack webhook integration
	Slack SlackAlertConfig `yaml:"slack,omitempty"`

	// Webhook configuration for generic webhook integration
	Webhook WebhookAlertConfig `yaml:"webhook,omitempty"`
}

// PagerDutyConfig holds PagerDuty integration settings.
type PagerDutyConfig struct {
	// Enabled controls whether PagerDuty integration is active
	Enabled bool `yaml:"enabled,omitempty"`

	// RoutingKey is the PagerDuty routing key (integration key)
	RoutingKey string `yaml:"routing_key,omitempty"`

	// ServiceID is the PagerDuty service ID (optional)
	ServiceID string `yaml:"service_id,omitempty"`

	// URL is the PagerDuty Events API URL (defaults to v2 events API)
	URL string `yaml:"url,omitempty"`
}

// OpsgenieConfig holds Opsgenie integration settings.
type OpsgenieConfig struct {
	// Enabled controls whether Opsgenie integration is active
	Enabled bool `yaml:"enabled,omitempty"`

	// APIKey is the Opsgenie API key
	APIKey string `yaml:"api_key,omitempty"`

	// Region is the Opsgenie region ("us" or "eu")
	Region string `yaml:"region,omitempty"`

	// TeamID is the Opsgenie team ID (optional)
	TeamID string `yaml:"team_id,omitempty"`
}

// SlackAlertConfig holds Slack webhook integration settings.
type SlackAlertConfig struct {
	// Enabled controls whether Slack integration is active
	Enabled bool `yaml:"enabled,omitempty"`

	// WebhookURL is the Slack incoming webhook URL
	WebhookURL string `yaml:"webhook_url,omitempty"`

	// Channel is the default channel to send alerts (optional)
	Channel string `yaml:"channel,omitempty"`

	// Username is the bot username for alerts (optional)
	Username string `yaml:"username,omitempty"`

	// IconEmoji is the emoji icon for alert messages (optional)
	IconEmoji string `yaml:"icon_emoji,omitempty"`
}

// WebhookAlertConfig holds generic webhook integration settings.
type WebhookAlertConfig struct {
	// Enabled controls whether webhook integration is active
	Enabled bool `yaml:"enabled,omitempty"`

	// URL is the webhook endpoint URL
	URL string `yaml:"url,omitempty"`

	// Method is the HTTP method (defaults to POST)
	Method string `yaml:"method,omitempty"`

	// Headers are additional HTTP headers to include
	Headers map[string]string `yaml:"headers,omitempty"`

	// Timeout is the request timeout (defaults to 30s)
	Timeout time.Duration `yaml:"timeout,omitempty"`
}

// LogExportConfig holds log aggregation/export configuration.
type LogExportConfig struct {
	// Enabled controls whether log export is active
	Enabled bool `yaml:"enabled,omitempty"`

	// Type is the export type: "elk", "splunk", or "loki"
	Type string `yaml:"type,omitempty"`

	// Endpoint is the log aggregation endpoint URL
	Endpoint string `yaml:"endpoint,omitempty"`

	// APIKey is the API key for authentication (if required)
	APIKey string `yaml:"api_key,omitempty"`

	// Index is the index name for ELK or source for Splunk
	Index string `yaml:"index,omitempty"`

	// BatchSize is the number of log entries per batch
	BatchSize int `yaml:"batch_size,omitempty"`

	// FlushInterval is the interval between batch sends
	FlushInterval time.Duration `yaml:"flush_interval,omitempty"`

	// Labels are additional labels for Loki
	Labels map[string]string `yaml:"labels,omitempty"`
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

// loadConfig loads the global configuration, creating default if it doesn't exist.
// Environment variables override file-based configuration for Vault settings.
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
		// Apply environment overrides to default config
		applyVaultEnvOverrides(defaultConfig)
		applyAWSSecretsManagerEnvOverrides(defaultConfig)
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

	// Apply environment variable overrides
	applyVaultEnvOverrides(&config)
	applyAWSSecretsManagerEnvOverrides(&config)

	return &config, nil
}

// applyVaultEnvOverrides applies environment variable overrides for Vault configuration.
// This follows HashiCorp Vault conventions for environment variables:
//   - VAULT_ADDR: Vault server address
//   - VAULT_NAMESPACE: Vault namespace (Enterprise)
//   - VAULT_CACERT: Path to CA certificate
//   - VAULT_CAPATH: Path to CA certificate directory
//   - VAULT_CLIENT_CERT: Path to client certificate
//   - VAULT_CLIENT_KEY: Path to client private key
//   - VAULT_TLS_SERVER_NAME: TLS server name for SNI
//   - VAULT_SKIP_VERIFY: Skip TLS verification (not recommended)
//
// Specular-specific environment variables:
//   - SPECULAR_VAULT_ENABLED: Enable/disable Vault integration
//   - SPECULAR_VAULT_MOUNT_PATH: KV mount path
//   - SPECULAR_VAULT_SIGNING_KEY_PATH: Path to signing key
//   - SPECULAR_VAULT_SIGNER_IDENTITY: Signer identity
func applyVaultEnvOverrides(config *GlobalConfig) {
	// Standard Vault environment variables
	if addr := os.Getenv("VAULT_ADDR"); addr != "" {
		config.Vault.Address = addr
		// If VAULT_ADDR is set, implicitly enable Vault
		config.Vault.Enabled = true
	}

	if namespace := os.Getenv("VAULT_NAMESPACE"); namespace != "" {
		config.Vault.Namespace = namespace
	}

	// TLS configuration from environment
	if caCert := os.Getenv("VAULT_CACERT"); caCert != "" {
		config.Vault.TLS.CACert = caCert
	}

	if caPath := os.Getenv("VAULT_CAPATH"); caPath != "" {
		config.Vault.TLS.CAPath = caPath
	}

	if clientCert := os.Getenv("VAULT_CLIENT_CERT"); clientCert != "" {
		config.Vault.TLS.ClientCert = clientCert
	}

	if clientKey := os.Getenv("VAULT_CLIENT_KEY"); clientKey != "" {
		config.Vault.TLS.ClientKey = clientKey
	}

	if serverName := os.Getenv("VAULT_TLS_SERVER_NAME"); serverName != "" {
		config.Vault.TLS.ServerName = serverName
	}

	if skipVerify := os.Getenv("VAULT_SKIP_VERIFY"); skipVerify != "" {
		config.Vault.TLS.InsecureSkipVerify = parseBool(skipVerify)
	}

	// Specular-specific Vault environment variables
	if enabled := os.Getenv("SPECULAR_VAULT_ENABLED"); enabled != "" {
		config.Vault.Enabled = parseBool(enabled)
	}

	if mountPath := os.Getenv("SPECULAR_VAULT_MOUNT_PATH"); mountPath != "" {
		config.Vault.MountPath = mountPath
	}

	if keyPath := os.Getenv("SPECULAR_VAULT_SIGNING_KEY_PATH"); keyPath != "" {
		config.Vault.SigningKeyPath = keyPath
	}

	if identity := os.Getenv("SPECULAR_VAULT_SIGNER_IDENTITY"); identity != "" {
		config.Vault.SignerIdentity = identity
	}

	if autoGen := os.Getenv("SPECULAR_VAULT_AUTO_GENERATE_KEY"); autoGen != "" {
		config.Vault.AutoGenerateKey = parseBool(autoGen)
	}
}

// applyAWSSecretsManagerEnvOverrides applies environment variable overrides for
// AWS Secrets Manager configuration.
//
// AWS standard environment variables:
//   - AWS_REGION: AWS region
//   - AWS_PROFILE: AWS profile name
//   - AWS_ENDPOINT_URL: Custom endpoint (for LocalStack)
//
// Specular-specific environment variables:
//   - SPECULAR_AWS_SM_ENABLED: Enable/disable AWS Secrets Manager integration
//   - SPECULAR_AWS_SM_REGION: Primary AWS region
//   - SPECULAR_AWS_SM_SECONDARY_REGION: DR/failover region
//   - SPECULAR_AWS_SM_PROFILE: AWS profile name
//   - SPECULAR_AWS_SM_ASSUME_ROLE_ARN: Role ARN for cross-account access
//   - SPECULAR_AWS_SM_SIGNING_KEY_NAME: Secret name for signing key
//   - SPECULAR_AWS_SM_SIGNER_IDENTITY: Signer identity
//   - SPECULAR_AWS_SM_AUTO_GENERATE_KEY: Auto-generate signing key
//   - SPECULAR_AWS_SM_ENDPOINT: Custom endpoint URL
func applyAWSSecretsManagerEnvOverrides(config *GlobalConfig) {
	// AWS standard environment variables
	if region := os.Getenv("AWS_REGION"); region != "" {
		config.AWSSecretsManager.Region = region
	}

	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		config.AWSSecretsManager.Profile = profile
	}

	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		config.AWSSecretsManager.Endpoint = endpoint
	}

	// Specular-specific AWS Secrets Manager environment variables
	if enabled := os.Getenv("SPECULAR_AWS_SM_ENABLED"); enabled != "" {
		config.AWSSecretsManager.Enabled = parseBool(enabled)
	}

	if region := os.Getenv("SPECULAR_AWS_SM_REGION"); region != "" {
		config.AWSSecretsManager.Region = region
		// If region is explicitly set, implicitly enable AWS SM
		config.AWSSecretsManager.Enabled = true
	}

	if secondaryRegion := os.Getenv("SPECULAR_AWS_SM_SECONDARY_REGION"); secondaryRegion != "" {
		config.AWSSecretsManager.SecondaryRegion = secondaryRegion
	}

	if profile := os.Getenv("SPECULAR_AWS_SM_PROFILE"); profile != "" {
		config.AWSSecretsManager.Profile = profile
	}

	if roleARN := os.Getenv("SPECULAR_AWS_SM_ASSUME_ROLE_ARN"); roleARN != "" {
		config.AWSSecretsManager.AssumeRoleARN = roleARN
	}

	if keyName := os.Getenv("SPECULAR_AWS_SM_SIGNING_KEY_NAME"); keyName != "" {
		config.AWSSecretsManager.SigningKeyName = keyName
	}

	if identity := os.Getenv("SPECULAR_AWS_SM_SIGNER_IDENTITY"); identity != "" {
		config.AWSSecretsManager.SignerIdentity = identity
	}

	if autoGen := os.Getenv("SPECULAR_AWS_SM_AUTO_GENERATE_KEY"); autoGen != "" {
		config.AWSSecretsManager.AutoGenerateKey = parseBool(autoGen)
	}

	if endpoint := os.Getenv("SPECULAR_AWS_SM_ENDPOINT"); endpoint != "" {
		config.AWSSecretsManager.Endpoint = endpoint
	}
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
			Endpoint:   "",
			SampleRate: 1.0,
		},
		Vault: VaultConfig{
			Enabled:         false,
			Address:         "",
			MountPath:       "secret",
			Namespace:       "",
			SigningKeyPath:  "specular/audit/signing-key",
			SignerIdentity:  "",
			AutoGenerateKey: true,
			TLS:             VaultTLSConfig{},
		},
		AWSSecretsManager: AWSSecretsManagerConfig{
			Enabled:         false,
			Region:          "",
			SecondaryRegion: "",
			Profile:         "",
			AssumeRoleARN:   "",
			SigningKeyName:  "specular/audit/signing-key",
			SignerIdentity:  "",
			AutoGenerateKey: true,
			Endpoint:        "",
			Tags:            nil,
		},
		Observability: ObservabilityConfig{
			SLO: SLOConfig{
				Enabled:       false,
				ConfigPath:    "",
				DefaultWindow: slo.Duration(30 * 24 * time.Hour), // 30 days
				CacheTTL:      slo.Duration(1 * time.Minute),
				SLOs:          nil,
			},
			Alerting: AlertingConfig{
				Enabled:         false,
				DefaultSeverity: "warning",
				PagerDuty: PagerDutyConfig{
					Enabled: false,
					URL:     "https://events.pagerduty.com/v2/enqueue",
				},
				Opsgenie: OpsgenieConfig{
					Enabled: false,
					Region:  "us",
				},
				Slack: SlackAlertConfig{
					Enabled:   false,
					Username:  "Specular",
					IconEmoji: ":robot_face:",
				},
				Webhook: WebhookAlertConfig{
					Enabled: false,
					Method:  "POST",
					Timeout: 30 * time.Second,
				},
			},
			LogExport: LogExportConfig{
				Enabled:       false,
				Type:          "",
				Endpoint:      "",
				BatchSize:     100,
				FlushInterval: 10 * time.Second,
			},
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
	case "telemetry.endpoint":
		return config.Telemetry.Endpoint, nil
	case "telemetry.sample_rate":
		return fmt.Sprintf("%.2f", config.Telemetry.SampleRate), nil
	case "vault.enabled":
		return fmt.Sprintf("%t", config.Vault.Enabled), nil
	case "vault.address":
		return config.Vault.Address, nil
	case "vault.mount_path":
		return config.Vault.MountPath, nil
	case "vault.namespace":
		return config.Vault.Namespace, nil
	case "vault.signing_key_path":
		return config.Vault.SigningKeyPath, nil
	case "vault.signer_identity":
		return config.Vault.SignerIdentity, nil
	case "vault.auto_generate_key":
		return fmt.Sprintf("%t", config.Vault.AutoGenerateKey), nil
	case "vault.tls.ca_cert":
		return config.Vault.TLS.CACert, nil
	case "vault.tls.ca_path":
		return config.Vault.TLS.CAPath, nil
	case "vault.tls.client_cert":
		return config.Vault.TLS.ClientCert, nil
	case "vault.tls.client_key":
		return config.Vault.TLS.ClientKey, nil
	case "vault.tls.server_name":
		return config.Vault.TLS.ServerName, nil
	case "vault.tls.insecure_skip_verify":
		return fmt.Sprintf("%t", config.Vault.TLS.InsecureSkipVerify), nil
	case "aws_secrets_manager.enabled":
		return fmt.Sprintf("%t", config.AWSSecretsManager.Enabled), nil
	case "aws_secrets_manager.region":
		return config.AWSSecretsManager.Region, nil
	case "aws_secrets_manager.secondary_region":
		return config.AWSSecretsManager.SecondaryRegion, nil
	case "aws_secrets_manager.profile":
		return config.AWSSecretsManager.Profile, nil
	case "aws_secrets_manager.assume_role_arn":
		return config.AWSSecretsManager.AssumeRoleARN, nil
	case "aws_secrets_manager.signing_key_name":
		return config.AWSSecretsManager.SigningKeyName, nil
	case "aws_secrets_manager.signer_identity":
		return config.AWSSecretsManager.SignerIdentity, nil
	case "aws_secrets_manager.auto_generate_key":
		return fmt.Sprintf("%t", config.AWSSecretsManager.AutoGenerateKey), nil
	case "aws_secrets_manager.endpoint":
		return config.AWSSecretsManager.Endpoint, nil
	// Observability configuration
	case "observability.slo.enabled":
		return fmt.Sprintf("%t", config.Observability.SLO.Enabled), nil
	case "observability.slo.config_path":
		return config.Observability.SLO.ConfigPath, nil
	case "observability.slo.default_window":
		return config.Observability.SLO.DefaultWindow.String(), nil
	case "observability.slo.cache_ttl":
		return config.Observability.SLO.CacheTTL.String(), nil
	case "observability.alerting.enabled":
		return fmt.Sprintf("%t", config.Observability.Alerting.Enabled), nil
	case "observability.alerting.default_severity":
		return config.Observability.Alerting.DefaultSeverity, nil
	case "observability.alerting.pagerduty.enabled":
		return fmt.Sprintf("%t", config.Observability.Alerting.PagerDuty.Enabled), nil
	case "observability.alerting.pagerduty.routing_key":
		return config.Observability.Alerting.PagerDuty.RoutingKey, nil
	case "observability.alerting.pagerduty.url":
		return config.Observability.Alerting.PagerDuty.URL, nil
	case "observability.alerting.opsgenie.enabled":
		return fmt.Sprintf("%t", config.Observability.Alerting.Opsgenie.Enabled), nil
	case "observability.alerting.opsgenie.api_key":
		return config.Observability.Alerting.Opsgenie.APIKey, nil
	case "observability.alerting.opsgenie.region":
		return config.Observability.Alerting.Opsgenie.Region, nil
	case "observability.alerting.slack.enabled":
		return fmt.Sprintf("%t", config.Observability.Alerting.Slack.Enabled), nil
	case "observability.alerting.slack.webhook_url":
		return config.Observability.Alerting.Slack.WebhookURL, nil
	case "observability.alerting.slack.channel":
		return config.Observability.Alerting.Slack.Channel, nil
	case "observability.alerting.webhook.enabled":
		return fmt.Sprintf("%t", config.Observability.Alerting.Webhook.Enabled), nil
	case "observability.alerting.webhook.url":
		return config.Observability.Alerting.Webhook.URL, nil
	case "observability.alerting.webhook.method":
		return config.Observability.Alerting.Webhook.Method, nil
	case "observability.log_export.enabled":
		return fmt.Sprintf("%t", config.Observability.LogExport.Enabled), nil
	case "observability.log_export.type":
		return config.Observability.LogExport.Type, nil
	case "observability.log_export.endpoint":
		return config.Observability.LogExport.Endpoint, nil
	case "observability.log_export.api_key":
		return config.Observability.LogExport.APIKey, nil
	case "observability.log_export.index":
		return config.Observability.LogExport.Index, nil
	case "observability.log_export.batch_size":
		return fmt.Sprintf("%d", config.Observability.LogExport.BatchSize), nil
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
	case "telemetry.endpoint":
		config.Telemetry.Endpoint = value
	case "telemetry.sample_rate":
		if v, err := parseFloat(value); err == nil {
			config.Telemetry.SampleRate = v
		} else {
			return err
		}
	case "vault.enabled":
		config.Vault.Enabled = parseBool(value)
	case "vault.address":
		config.Vault.Address = value
	case "vault.mount_path":
		config.Vault.MountPath = value
	case "vault.namespace":
		config.Vault.Namespace = value
	case "vault.signing_key_path":
		config.Vault.SigningKeyPath = value
	case "vault.signer_identity":
		config.Vault.SignerIdentity = value
	case "vault.auto_generate_key":
		config.Vault.AutoGenerateKey = parseBool(value)
	case "vault.tls.ca_cert":
		config.Vault.TLS.CACert = value
	case "vault.tls.ca_path":
		config.Vault.TLS.CAPath = value
	case "vault.tls.client_cert":
		config.Vault.TLS.ClientCert = value
	case "vault.tls.client_key":
		config.Vault.TLS.ClientKey = value
	case "vault.tls.server_name":
		config.Vault.TLS.ServerName = value
	case "vault.tls.insecure_skip_verify":
		config.Vault.TLS.InsecureSkipVerify = parseBool(value)
	case "aws_secrets_manager.enabled":
		config.AWSSecretsManager.Enabled = parseBool(value)
	case "aws_secrets_manager.region":
		config.AWSSecretsManager.Region = value
	case "aws_secrets_manager.secondary_region":
		config.AWSSecretsManager.SecondaryRegion = value
	case "aws_secrets_manager.profile":
		config.AWSSecretsManager.Profile = value
	case "aws_secrets_manager.assume_role_arn":
		config.AWSSecretsManager.AssumeRoleARN = value
	case "aws_secrets_manager.signing_key_name":
		config.AWSSecretsManager.SigningKeyName = value
	case "aws_secrets_manager.signer_identity":
		config.AWSSecretsManager.SignerIdentity = value
	case "aws_secrets_manager.auto_generate_key":
		config.AWSSecretsManager.AutoGenerateKey = parseBool(value)
	case "aws_secrets_manager.endpoint":
		config.AWSSecretsManager.Endpoint = value
	// Observability configuration
	case "observability.slo.enabled":
		config.Observability.SLO.Enabled = parseBool(value)
	case "observability.slo.config_path":
		config.Observability.SLO.ConfigPath = value
	case "observability.slo.default_window":
		d, err := slo.ParseDuration(value)
		if err != nil {
			return err
		}
		config.Observability.SLO.DefaultWindow = slo.Duration(d)
	case "observability.slo.cache_ttl":
		d, err := slo.ParseDuration(value)
		if err != nil {
			return err
		}
		config.Observability.SLO.CacheTTL = slo.Duration(d)
	case "observability.alerting.enabled":
		config.Observability.Alerting.Enabled = parseBool(value)
	case "observability.alerting.default_severity":
		config.Observability.Alerting.DefaultSeverity = value
	case "observability.alerting.pagerduty.enabled":
		config.Observability.Alerting.PagerDuty.Enabled = parseBool(value)
	case "observability.alerting.pagerduty.routing_key":
		config.Observability.Alerting.PagerDuty.RoutingKey = value
	case "observability.alerting.pagerduty.url":
		config.Observability.Alerting.PagerDuty.URL = value
	case "observability.alerting.opsgenie.enabled":
		config.Observability.Alerting.Opsgenie.Enabled = parseBool(value)
	case "observability.alerting.opsgenie.api_key":
		config.Observability.Alerting.Opsgenie.APIKey = value
	case "observability.alerting.opsgenie.region":
		config.Observability.Alerting.Opsgenie.Region = value
	case "observability.alerting.slack.enabled":
		config.Observability.Alerting.Slack.Enabled = parseBool(value)
	case "observability.alerting.slack.webhook_url":
		config.Observability.Alerting.Slack.WebhookURL = value
	case "observability.alerting.slack.channel":
		config.Observability.Alerting.Slack.Channel = value
	case "observability.alerting.webhook.enabled":
		config.Observability.Alerting.Webhook.Enabled = parseBool(value)
	case "observability.alerting.webhook.url":
		config.Observability.Alerting.Webhook.URL = value
	case "observability.alerting.webhook.method":
		config.Observability.Alerting.Webhook.Method = value
	case "observability.log_export.enabled":
		config.Observability.LogExport.Enabled = parseBool(value)
	case "observability.log_export.type":
		config.Observability.LogExport.Type = value
	case "observability.log_export.endpoint":
		config.Observability.LogExport.Endpoint = value
	case "observability.log_export.api_key":
		config.Observability.LogExport.APIKey = value
	case "observability.log_export.index":
		config.Observability.LogExport.Index = value
	case "observability.log_export.batch_size":
		if v, err := parseInt(value); err == nil {
			config.Observability.LogExport.BatchSize = v
		} else {
			return err
		}
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
