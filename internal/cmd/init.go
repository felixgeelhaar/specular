package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/detect"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var (
	initForce         bool
	initProviderSetup bool
	initTemplate      string
	initLocal         bool
	initCloud         bool
	initGovernance    string
	initProviders     string
	initMCP           string
	initDryRun        bool
	initNoDetect      bool
	initYes           bool
)

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize a new specular project with smart context detection",
	Long: `Initialize a new specular project with smart context detection and configuration.

Automatically detects your environment (Docker, AI providers, languages, frameworks, Git, CI)
and generates optimized configuration files based on your project context.

Examples:
  # Initialize with automatic detection
  specular init

  # Initialize with specific template
  specular init --template web-app

  # Initialize preferring local providers (Ollama)
  specular init --local

  # Initialize preferring cloud providers (OpenAI, Anthropic)
  specular init --cloud

  # Initialize with specific providers
  specular init --providers ollama,anthropic

  # Preview changes without writing files
  specular init --dry-run

  # Skip auto-detection and use defaults
  specular init --no-detect

  # Auto-accept all prompts (non-interactive)
  specular init --yes

  # Force re-initialization
  specular init --force`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite existing configuration files")
	initCmd.Flags().BoolVar(&initProviderSetup, "provider-setup", true, "run interactive provider setup wizard")
	initCmd.Flags().StringVar(&initTemplate, "template", "", "project template (web-app, api-service, cli-tool, microservice, data-pipeline)")
	initCmd.Flags().BoolVar(&initLocal, "local", false, "prefer local AI providers (Ollama)")
	initCmd.Flags().BoolVar(&initCloud, "cloud", false, "prefer cloud AI providers (OpenAI, Anthropic, Gemini)")
	initCmd.Flags().StringVar(&initGovernance, "governance", "L2", "target governance level (L2, L3, L4)")
	initCmd.Flags().StringVar(&initProviders, "providers", "", "comma-separated list of providers to enable")
	initCmd.Flags().StringVar(&initMCP, "mcp", "auto", "MCP integration (enable, disable, auto)")
	initCmd.Flags().BoolVar(&initDryRun, "dry-run", false, "preview changes without writing files")
	initCmd.Flags().BoolVar(&initNoDetect, "no-detect", false, "skip automatic context detection")
	initCmd.Flags().BoolVar(&initYes, "yes", false, "auto-accept all prompts (non-interactive mode)")

	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	// Create absolute path
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return ux.FormatError(err, "resolving directory path")
	}

	specDir := filepath.Join(absDir, ".specular")

	// Check if .specular directory already exists
	if _, err := os.Stat(specDir); err == nil && !initForce {
		return fmt.Errorf(".specular directory already exists at %s\nUse --force to overwrite existing files", specDir)
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           Specular Project Initialization                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Context detection
	var ctx *detect.Context
	if !initNoDetect {
		fmt.Println("🔍 Detecting project context...")
		ctx, err = detect.DetectAll()
		if err != nil {
			fmt.Printf("⚠  Context detection failed: %v\n", err)
			fmt.Println("   Continuing with manual configuration...")
			ctx = &detect.Context{}
		} else {
			printDetectionSummary(ctx)
		}
	} else {
		ctx = &detect.Context{}
		fmt.Println("ℹ  Skipping context detection (--no-detect)")
	}

	// Determine provider strategy
	providerStrategy := determineProviderStrategy(ctx)

	// Generate configuration
	config := &InitConfig{
		TargetDir:        absDir,
		SpecDir:          specDir,
		Context:          ctx,
		Template:         initTemplate,
		ProviderStrategy: providerStrategy,
		Governance:       initGovernance,
		MCPEnabled:       determineMCPEnabled(ctx),
		Timestamp:        time.Now(),
	}

	// Preview changes
	if initDryRun {
		return previewChanges(config)
	}

	// Confirm before writing (unless --yes)
	if !initYes && !initForce {
		if !confirmInitialization(config) {
			fmt.Println("\nInitialization cancelled.")
			return nil
		}
	}

	// Create .specular directory
	if err := os.MkdirAll(specDir, 0750); err != nil {
		return ux.FormatError(err, "creating .specular directory")
	}

	fmt.Printf("\n✓ Created .specular directory at %s\n", specDir)

	// Generate and write configuration files
	if err := generateConfigFiles(config); err != nil {
		return ux.FormatError(err, "generating configuration files")
	}

	// Interactive provider setup (if not using --yes)
	if initProviderSetup && !initYes && initProviders == "" {
		if err := runSmartProviderSetup(specDir, ctx); err != nil {
			fmt.Printf("⚠  Provider setup skipped: %v\n", err)
			fmt.Println("   You can manually edit .specular/router.yaml to configure providers")
		}
	}

	// Print success message and next steps
	printSmartSuccessMessage(config)

	return nil
}

// InitConfig holds all configuration for initialization
type InitConfig struct {
	TargetDir        string
	SpecDir          string
	Context          *detect.Context
	Template         string
	ProviderStrategy string
	Governance       string
	MCPEnabled       bool
	Timestamp        time.Time
}

func printDetectionSummary(ctx *detect.Context) {
	fmt.Println()
	fmt.Println("Detected Environment:")

	// Container runtime
	if ctx.Docker.Available {
		fmt.Printf("  ✓ Docker: %s\n", ctx.Docker.Version)
	} else if ctx.Podman.Available {
		fmt.Printf("  ✓ Podman: %s\n", ctx.Podman.Version)
	} else {
		fmt.Println("  ○ No container runtime detected")
	}

	// AI Providers
	providers := ctx.GetRecommendedProviders()
	if len(providers) > 0 {
		fmt.Printf("  ✓ AI Providers: %s\n", strings.Join(providers, ", "))
	} else {
		fmt.Println("  ○ No AI providers detected")
	}

	// Languages/Frameworks
	if len(ctx.Languages) > 0 {
		fmt.Printf("  ✓ Languages: %s\n", strings.Join(ctx.Languages, ", "))
	}
	if len(ctx.Frameworks) > 0 {
		fmt.Printf("  ✓ Frameworks: %s\n", strings.Join(ctx.Frameworks, ", "))
	}

	// Git
	if ctx.Git.Initialized {
		fmt.Printf("  ✓ Git: branch %s\n", ctx.Git.Branch)
	}

	// CI
	if ctx.CI.Detected {
		fmt.Printf("  ✓ CI: %s\n", ctx.CI.Name)
	}

	fmt.Println()
}

func determineProviderStrategy(ctx *detect.Context) string {
	// Explicit flags take precedence
	if initLocal {
		return "local"
	}
	if initCloud {
		return "cloud"
	}
	if initProviders != "" {
		return "explicit"
	}

	// Auto-detect based on context
	if ctx.Providers == nil || len(ctx.Providers) == 0 {
		return "manual"
	}

	// Check what's available
	hasLocal := false
	hasCloud := false

	for name, info := range ctx.Providers {
		if !info.Available {
			continue
		}
		if name == "ollama" {
			hasLocal = true
		}
		if name == "anthropic" || name == "openai" || name == "gemini" {
			hasCloud = true
		}
	}

	if hasLocal && !hasCloud {
		return "local"
	}
	if hasCloud && !hasLocal {
		return "cloud"
	}
	if hasLocal && hasCloud {
		return "hybrid"
	}

	return "manual"
}

func determineMCPEnabled(ctx *detect.Context) bool {
	if initMCP == "enable" {
		return true
	}
	if initMCP == "disable" {
		return false
	}

	// Auto: disabled for now (no IDE detection yet)
	return false
}

func confirmInitialization(config *InitConfig) bool {
	fmt.Println()
	fmt.Println("Configuration Summary:")
	fmt.Printf("  Directory:  %s\n", config.TargetDir)
	if config.Template != "" {
		fmt.Printf("  Template:   %s\n", config.Template)
	}
	fmt.Printf("  Strategy:   %s providers\n", config.ProviderStrategy)
	fmt.Printf("  Governance: %s\n", config.Governance)
	fmt.Printf("  MCP:        %v\n", config.MCPEnabled)
	fmt.Println()
	fmt.Print("Proceed with initialization? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	return response == "" || response == "y" || response == "yes"
}

func previewChanges(config *InitConfig) error {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     Dry Run Preview                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Would create the following files:")
	fmt.Printf("  📄 %s/router.yaml\n", filepath.Base(config.SpecDir))
	fmt.Printf("  📄 %s/policy.yaml\n", filepath.Base(config.SpecDir))
	fmt.Printf("  📄 %s/spec.yaml\n", filepath.Base(config.SpecDir))
	fmt.Printf("  📄 %s/settings.json\n", filepath.Base(config.SpecDir))
	fmt.Println()
	fmt.Println("Configuration Summary:")
	fmt.Printf("  Provider Strategy: %s\n", config.ProviderStrategy)
	fmt.Printf("  Governance Level:  %s\n", config.Governance)
	fmt.Printf("  MCP Enabled:       %v\n", config.MCPEnabled)
	fmt.Println()
	fmt.Println("Run without --dry-run to create these files.")
	fmt.Println()

	return nil
}

func generateConfigFiles(config *InitConfig) error {
	// Generate router.yaml
	routerContent := generateRouterYAML(config)
	if err := os.WriteFile(filepath.Join(config.SpecDir, "router.yaml"), []byte(routerContent), 0600); err != nil {
		return err
	}
	fmt.Println("✓ Created router.yaml")

	// Generate policy.yaml
	policyContent := generatePolicyYAML(config)
	if err := os.WriteFile(filepath.Join(config.SpecDir, "policy.yaml"), []byte(policyContent), 0600); err != nil {
		return err
	}
	fmt.Println("✓ Created policy.yaml")

	// Generate spec.yaml template
	specContent := generateSpecYAML(config)
	if err := os.WriteFile(filepath.Join(config.SpecDir, "spec.yaml"), []byte(specContent), 0600); err != nil {
		return err
	}
	fmt.Println("✓ Created spec.yaml")

	// Generate settings.json
	settingsContent := generateSettingsJSON(config)
	if err := os.WriteFile(filepath.Join(config.SpecDir, "settings.json"), []byte(settingsContent), 0600); err != nil {
		return err
	}
	fmt.Println("✓ Created settings.json")

	return nil
}

func generateRouterYAML(config *InitConfig) string {
	// Determine which providers to enable based on strategy
	ollama := "false"
	openai := "false"
	anthropic := "false"
	gemini := "false"

	switch config.ProviderStrategy {
	case "local":
		ollama = "true"
	case "cloud":
		openai = "true"
		anthropic = "true"
		gemini = "true"
	case "hybrid":
		ollama = "true"
		openai = "true"
		anthropic = "true"
	case "explicit":
		providers := strings.Split(initProviders, ",")
		for _, p := range providers {
			p = strings.TrimSpace(p)
			switch p {
			case "ollama":
				ollama = "true"
			case "openai":
				openai = "true"
			case "anthropic":
				anthropic = "true"
			case "gemini":
				gemini = "true"
			}
		}
	}

	return fmt.Sprintf(`# Specular Router Configuration
# Generated by: specular init
# Date: %s

# Budget configuration
budget:
  max_cost: 100.0
  remaining: 100.0

# Performance constraints
max_latency_ms: 5000

# Cost preferences
prefer_cheap: false

# Fallback and retry configuration
enable_fallback: true
max_retries: 3

# Context handling
validate_context: true
auto_truncate: true

# Provider configuration
providers:
  - name: ollama
    enabled: %s
    type: local
    base_url: http://localhost:11434

  - name: openai
    enabled: %s
    type: api
    env_var: OPENAI_API_KEY

  - name: anthropic
    enabled: %s
    type: api
    env_var: ANTHROPIC_API_KEY

  - name: gemini
    enabled: %s
    type: api
    env_var: GEMINI_API_KEY
`, config.Timestamp.Format("2006-01-02 15:04:05"), ollama, openai, anthropic, gemini)
}

func generatePolicyYAML(config *InitConfig) string {
	// Adjust policies based on governance level
	allowInternet := "false"
	allowFilesystem := "limited"

	switch config.Governance {
	case "L2":
		allowInternet = "false"
		allowFilesystem = "limited"
	case "L3":
		allowInternet = "limited"
		allowFilesystem = "limited"
	case "L4":
		allowInternet = "true"
		allowFilesystem = "full"
	}

	return fmt.Sprintf(`# Specular Security Policy
# Generated by: specular init
# Governance Level: %s
# Date: %s

# Allowed Docker images
allowed_images:
  - "alpine:latest"
  - "node:20-alpine"
  - "python:3.11-alpine"
  - "golang:1.21-alpine"

# Network access policy
network:
  allow_internet: %s
  allowed_domains:
    - "github.com"
    - "npmjs.com"
    - "pypi.org"

# Filesystem access policy
filesystem:
  mode: %s  # limited, full
  allowed_paths:
    - "/app"
    - "/tmp"

# Resource limits
resources:
  max_memory_mb: 512
  max_cpu_percent: 50
  max_execution_time_minutes: 10

# Privacy settings
privacy:
  telemetry: false
  upload_code: false
  share_metrics: false
`, config.Governance, config.Timestamp.Format("2006-01-02 15:04:05"), allowInternet, allowFilesystem)
}

func generateSpecYAML(config *InitConfig) string {
	// Generate template-specific spec
	switch config.Template {
	case "web-app":
		return generateWebAppSpec(config)
	case "api-service":
		return generateAPIServiceSpec(config)
	case "cli-tool":
		return generateCLIToolSpec(config)
	case "microservice":
		return generateMicroserviceSpec(config)
	case "data-pipeline":
		return generateDataPipelineSpec(config)
	default:
		return generateDefaultSpec(config)
	}
}

func generateDefaultSpec(config *InitConfig) string {
	projectName := filepath.Base(config.TargetDir)
	detectedLangs := ""
	if config.Context != nil && len(config.Context.Languages) > 0 {
		detectedLangs = fmt.Sprintf("\n# Detected languages: %s", strings.Join(config.Context.Languages, ", "))
	}

	return fmt.Sprintf(`# Specular Product Specification
# Generated by: specular init
# Project: %s
# Date: %s%s

project:
  name: "%s"
  version: "0.1.0"
  description: "Product specification for %s"

features:
  - id: example-feature
    name: "Example Feature"
    description: "This is an example feature specification"
    priority: P1
    acceptance_criteria:
      - "Feature works as described"
      - "Tests pass"
      - "Documentation updated"
    tasks:
      - "Implement core functionality"
      - "Add tests"
      - "Update documentation"
`, projectName, config.Timestamp.Format("2006-01-02"), detectedLangs, projectName, projectName)
}

func generateWebAppSpec(config *InitConfig) string {
	projectName := filepath.Base(config.TargetDir)
	return fmt.Sprintf(`# Specular Product Specification - Web Application
# Project: %s
# Template: web-app
# Date: %s

project:
  name: "%s"
  version: "0.1.0"
  description: "Web application specification"
  type: "web-app"

features:
  - id: frontend-setup
    name: "Frontend Setup"
    description: "Initialize frontend framework and build system"
    priority: P0
    acceptance_criteria:
      - "Development server runs successfully"
      - "Build system configured"
      - "Hot reload working"

  - id: backend-api
    name: "Backend API"
    description: "RESTful API backend service"
    priority: P0
    acceptance_criteria:
      - "API endpoints respond correctly"
      - "Database connection established"
      - "Authentication working"

  - id: ui-components
    name: "UI Components"
    description: "Reusable UI component library"
    priority: P1
    acceptance_criteria:
      - "Component library created"
      - "Storybook documentation"
      - "Accessible components"
`, projectName, config.Timestamp.Format("2006-01-02"), projectName)
}

func generateAPIServiceSpec(config *InitConfig) string {
	projectName := filepath.Base(config.TargetDir)
	return fmt.Sprintf(`# Specular Product Specification - API Service
# Project: %s
# Template: api-service
# Date: %s

project:
  name: "%s"
  version: "0.1.0"
  description: "API service specification"
  type: "api-service"

features:
  - id: api-endpoints
    name: "REST API Endpoints"
    description: "Core RESTful API endpoints"
    priority: P0
    acceptance_criteria:
      - "OpenAPI specification complete"
      - "All endpoints functional"
      - "Input validation working"

  - id: authentication
    name: "API Authentication"
    description: "JWT-based authentication"
    priority: P0
    acceptance_criteria:
      - "JWT tokens generated"
      - "Protected endpoints secured"
      - "Refresh tokens working"

  - id: database
    name: "Database Layer"
    description: "Database schema and migrations"
    priority: P0
    acceptance_criteria:
      - "Schema migrations working"
      - "CRUD operations complete"
      - "Indexes optimized"
`, projectName, config.Timestamp.Format("2006-01-02"), projectName)
}

func generateCLIToolSpec(config *InitConfig) string {
	projectName := filepath.Base(config.TargetDir)
	return fmt.Sprintf(`# Specular Product Specification - CLI Tool
# Project: %s
# Template: cli-tool
# Date: %s

project:
  name: "%s"
  version: "0.1.0"
  description: "Command-line tool specification"
  type: "cli-tool"

features:
  - id: command-structure
    name: "Command Structure"
    description: "CLI command hierarchy and flags"
    priority: P0
    acceptance_criteria:
      - "Commands parse correctly"
      - "Help text complete"
      - "Flags validated"

  - id: configuration
    name: "Configuration System"
    description: "Config file and environment variable support"
    priority: P1
    acceptance_criteria:
      - "Config file loading works"
      - "Environment variables override"
      - "Sensible defaults"

  - id: output-formatting
    name: "Output Formatting"
    description: "Multiple output formats (text, JSON, YAML)"
    priority: P1
    acceptance_criteria:
      - "Text output formatted"
      - "JSON output valid"
      - "YAML output correct"
`, projectName, config.Timestamp.Format("2006-01-02"), projectName)
}

func generateMicroserviceSpec(config *InitConfig) string {
	projectName := filepath.Base(config.TargetDir)
	return fmt.Sprintf(`# Specular Product Specification - Microservice
# Project: %s
# Template: microservice
# Date: %s

project:
  name: "%s"
  version: "0.1.0"
  description: "Microservice specification"
  type: "microservice"

features:
  - id: service-api
    name: "Service API"
    description: "gRPC or REST API endpoints"
    priority: P0
    acceptance_criteria:
      - "API endpoints defined"
      - "Service contract validated"
      - "Health checks implemented"

  - id: messaging
    name: "Message Queue Integration"
    description: "Event-driven communication"
    priority: P1
    acceptance_criteria:
      - "Message queue connected"
      - "Events published"
      - "Event handlers working"

  - id: observability
    name: "Observability"
    description: "Logging, metrics, and tracing"
    priority: P0
    acceptance_criteria:
      - "Structured logging configured"
      - "Metrics exported"
      - "Distributed tracing enabled"
`, projectName, config.Timestamp.Format("2006-01-02"), projectName)
}

func generateDataPipelineSpec(config *InitConfig) string {
	projectName := filepath.Base(config.TargetDir)
	return fmt.Sprintf(`# Specular Product Specification - Data Pipeline
# Project: %s
# Template: data-pipeline
# Date: %s

project:
  name: "%s"
  version: "0.1.0"
  description: "Data pipeline specification"
  type: "data-pipeline"

features:
  - id: data-ingestion
    name: "Data Ingestion"
    description: "Ingest data from various sources"
    priority: P0
    acceptance_criteria:
      - "Data sources connected"
      - "Ingestion scheduled"
      - "Error handling robust"

  - id: data-transformation
    name: "Data Transformation"
    description: "Transform and clean data"
    priority: P0
    acceptance_criteria:
      - "Transformation rules applied"
      - "Data validation complete"
      - "Quality checks passing"

  - id: data-storage
    name: "Data Storage"
    description: "Store processed data"
    priority: P0
    acceptance_criteria:
      - "Storage configured"
      - "Data partitioned"
      - "Retention policies set"
`, projectName, config.Timestamp.Format("2006-01-02"), projectName)
}

func generateSettingsJSON(config *InitConfig) string {
	return fmt.Sprintf(`{
  "initialized": "%s",
  "version": "1.2.0",
  "template": "%s",
  "provider_strategy": "%s",
  "governance": "%s",
  "mcp_enabled": %v,
  "telemetry": false,
  "detected_context": {
    "docker": %v,
    "languages": %s,
    "frameworks": %s,
    "git": %v,
    "ci": "%s"
  }
}`,
		config.Timestamp.Format(time.RFC3339),
		config.Template,
		config.ProviderStrategy,
		config.Governance,
		config.MCPEnabled,
		config.Context != nil && config.Context.Docker.Available,
		formatJSONArray(config.Context, "languages"),
		formatJSONArray(config.Context, "frameworks"),
		config.Context != nil && config.Context.Git.Initialized,
		getCI(config.Context),
	)
}

func formatJSONArray(ctx *detect.Context, field string) string {
	if ctx == nil {
		return "[]"
	}

	var items []string
	switch field {
	case "languages":
		items = ctx.Languages
	case "frameworks":
		items = ctx.Frameworks
	}

	if len(items) == 0 {
		return "[]"
	}

	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf(`"%s"`, item)
	}

	return "[" + strings.Join(quoted, ", ") + "]"
}

func getCI(ctx *detect.Context) string {
	if ctx == nil || !ctx.CI.Detected {
		return ""
	}
	return ctx.CI.Name
}

func runSmartProviderSetup(specDir string, ctx *detect.Context) error {
	if ctx.Providers == nil || len(ctx.Providers) == 0 {
		return fmt.Errorf("no providers detected")
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Smart Provider Setup")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Show detected providers
	fmt.Println("Detected Providers:")
	for name, info := range ctx.Providers {
		if info.Available {
			status := "✓"
			if info.EnvVar != "" && !info.EnvSet {
				status = "⚠"
			}
			fmt.Printf("  %s %s (%s)\n", status, name, info.Type)
			if info.EnvVar != "" && !info.EnvSet {
				fmt.Printf("     Missing: %s\n", info.EnvVar)
			}
		}
	}
	fmt.Println()

	// Recommend providers
	recommended := ctx.GetRecommendedProviders()
	if len(recommended) > 0 {
		fmt.Printf("Recommended: %s\n", strings.Join(recommended, ", "))
		fmt.Println()
		fmt.Print("Use recommended providers? [Y/n]: ")

		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "" || response == "y" || response == "yes" {
			routerPath := filepath.Join(specDir, "router.yaml")
			for _, provider := range recommended {
				if err := enableProvider(routerPath, provider); err != nil {
					fmt.Printf("⚠  Failed to enable %s: %v\n", provider, err)
				}
			}
			return nil
		}
	}

	// Manual provider selection
	return runProviderSetup(specDir)
}

func runProviderSetup(specDir string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nWhich provider would you like to use?")
	fmt.Println("  1. ollama (local, free)")
	fmt.Println("  2. openai (cloud, API key required)")
	fmt.Println("  3. anthropic (cloud, API key required)")
	fmt.Println("  4. gemini (cloud, API key required)")
	fmt.Println("  5. skip")
	fmt.Print("\nChoice [1-5]: ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	routerPath := filepath.Join(specDir, "router.yaml")

	switch choice {
	case "1":
		return enableProvider(routerPath, "ollama")
	case "2":
		return enableProvider(routerPath, "openai")
	case "3":
		return enableProvider(routerPath, "anthropic")
	case "4":
		return enableProvider(routerPath, "gemini")
	case "5", "":
		return nil
	default:
		fmt.Printf("Invalid choice: %s\n", choice)
		return nil
	}
}

func enableProvider(routerPath string, providerName string) error {
	// Read router.yaml
	content, err := os.ReadFile(routerPath)
	if err != nil {
		return fmt.Errorf("failed to read router.yaml: %w", err)
	}

	contentStr := string(content)

	// Find and replace "enabled: false" with "enabled: true" for the provider
	searchPattern := fmt.Sprintf("- name: %s", providerName)
	providerIndex := strings.Index(contentStr, searchPattern)
	if providerIndex == -1 {
		return fmt.Errorf("provider %s not found in router.yaml", providerName)
	}

	// Find the next "enabled: false" after the provider name
	searchStart := providerIndex
	nextProvider := strings.Index(contentStr[searchStart+len(searchPattern):], "- name:")
	searchEnd := len(contentStr)
	if nextProvider != -1 {
		searchEnd = searchStart + len(searchPattern) + nextProvider
	}

	// Search within this provider's section
	providerSection := contentStr[searchStart:searchEnd]
	enabledPattern := "enabled: false"
	enabledIndex := strings.Index(providerSection, enabledPattern)

	if enabledIndex == -1 {
		// Already enabled
		fmt.Printf("✓ Provider %s is already enabled\n", providerName)
		return nil
	}

	// Replace in the full content
	absoluteIndex := searchStart + enabledIndex
	contentStr = contentStr[:absoluteIndex] + "enabled: true " + contentStr[absoluteIndex+len(enabledPattern):]

	// Write back
	if err := os.WriteFile(routerPath, []byte(contentStr), 0600); err != nil {
		return fmt.Errorf("failed to update router.yaml: %w", err)
	}

	fmt.Printf("✓ Enabled provider: %s\n", providerName)
	return nil
}

func printSmartSuccessMessage(config *InitConfig) {
	projectName := filepath.Base(config.TargetDir)

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  ✨ Project Initialized Successfully!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Configuration files created:")
	fmt.Println("  • .specular/router.yaml    - AI provider routing")
	fmt.Println("  • .specular/policy.yaml    - Security policies")
	fmt.Println("  • .specular/spec.yaml      - Product specification")
	fmt.Println("  • .specular/settings.json  - Project settings")
	fmt.Println()

	if config.Template != "" {
		fmt.Printf("Template: %s\n", config.Template)
	}
	fmt.Printf("Provider Strategy: %s\n", config.ProviderStrategy)
	fmt.Printf("Governance Level: %s\n", config.Governance)
	fmt.Println()

	fmt.Println("Next steps:")
	fmt.Println()
	fmt.Println("  1. Check your system health:")
	fmt.Println("     $ specular doctor")
	fmt.Println()
	fmt.Println("  2. Review your configuration:")
	fmt.Println("     $ cat .specular/router.yaml")
	fmt.Println("     $ specular route show")
	fmt.Println()
	fmt.Println("  3. Create your spec (interactive):")
	fmt.Println("     $ specular interview")
	fmt.Println()
	fmt.Println("  4. Or edit the spec template:")
	fmt.Println("     $ vim .specular/spec.yaml")
	fmt.Println()
	fmt.Println("  5. Generate a plan:")
	fmt.Println("     $ specular plan")
	fmt.Println()
	fmt.Println("  6. Execute your plan:")
	fmt.Println("     $ specular build")
	fmt.Println()

	fmt.Printf("Project: %s\n", projectName)
	fmt.Println("Documentation: https://github.com/felixgeelhaar/specular")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}
