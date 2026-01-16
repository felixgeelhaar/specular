package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/detect"
	"github.com/felixgeelhaar/specular/internal/docgen"
	"github.com/felixgeelhaar/specular/internal/provider"
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
	Use:     "init [directory]",
	Aliases: []string{"i", "new"},
	Short:   "Initialize a new specular project with smart context detection",
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

// setupTargetDirectory resolves and validates the target directory
func setupTargetDirectory(args []string) (string, string, error) {
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return "", "", ux.FormatError(err, "resolving directory path")
	}

	specDir := filepath.Join(absDir, ".specular")

	// Check if .specular directory already exists
	if _, statErr := os.Stat(specDir); statErr == nil && !initForce {
		return "", "", fmt.Errorf(".specular directory already exists at %s\nUse --force to overwrite existing files", specDir)
	}

	return absDir, specDir, nil
}

// detectProjectContext performs context detection or returns empty context
func detectProjectContext() *detect.Context {
	if initNoDetect {
		fmt.Println("ℹ  Skipping context detection (--no-detect)")
		return &detect.Context{}
	}

	fmt.Println("🔍 Detecting project context...")
	ctx, err := detect.DetectAll()
	if err != nil {
		fmt.Printf("⚠  Context detection failed: %v\n", err)
		fmt.Println("   Continuing with manual configuration...")
		return &detect.Context{}
	}

	recordDetectionEvents(ctx)
	printDetectionSummary(ctx)
	return ctx
}

func recordDetectionEvents(ctx *detect.Context) {
	for name, info := range ctx.Providers {
		provider.RecordProviderDetected(name, info.EnvVar, info.Version, info.Available)
	}
}

// buildInitConfig creates the initialization configuration
func buildInitConfig(absDir, specDir string, ctx *detect.Context) *InitConfig {
	return &InitConfig{
		TargetDir:        absDir,
		SpecDir:          specDir,
		Context:          ctx,
		Template:         initTemplate,
		ProviderStrategy: determineProviderStrategy(ctx),
		Governance:       initGovernance,
		MCPEnabled:       determineMCPEnabled(ctx),
		Timestamp:        time.Now(),
	}
}

// executeInit performs the actual initialization steps
func executeInit(config *InitConfig) error {
	// Create .specular directory
	if err := os.MkdirAll(config.SpecDir, 0750); err != nil {
		return ux.FormatError(err, "creating .specular directory")
	}
	fmt.Printf("\n✓ Created .specular directory at %s\n", config.SpecDir)

	// Generate and write configuration files
	if err := generateConfigFiles(config); err != nil {
		return ux.FormatError(err, "generating configuration files")
	}

	// Interactive provider setup (if not using --yes)
	if initProviderSetup && !initYes && initProviders == "" {
		if err := runSmartProviderSetup(config.SpecDir, config.Context); err != nil {
			fmt.Printf("⚠  Provider setup skipped: %v\n", err)
			fmt.Println("   You can manually edit .specular/routing.yaml to configure providers")
		}
	}

	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	// Extract command context (for consistency with other commands)
	// Currently not used, but establishes pattern for future use
	_, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	// Handle --workflow flag (shorthand for 'init workflow <id>')
	if workflowFlag != "" {
		return RunWorkflowFromFlag(cmd, workflowFlag)
	}

	// Setup target directory
	absDir, specDir, err := setupTargetDirectory(args)
	if err != nil {
		return err
	}

	// Display header
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           Specular Project Initialization                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Detect project context
	ctx := detectProjectContext()

	// Build configuration
	config := buildInitConfig(absDir, specDir, ctx)

	// Preview changes in dry-run mode
	if initDryRun {
		return previewChanges(config)
	}

	// Confirm before writing (unless --yes or --force)
	if !initYes && !initForce {
		if !confirmInitialization(config) {
			fmt.Println("\nInitialization cancelled.")
			return nil
		}
	}

	// Execute initialization
	if initErr := executeInit(config); initErr != nil {
		return initErr
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
	printProviderSummary(ctx)

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

func printProviderSummary(ctx *detect.Context) {
	if ctx == nil {
		fmt.Println("  ○ AI Providers: none detected")
		return
	}

	aggregates := provider.AggregatesFromDetection(ctx)
	if len(aggregates) == 0 {
		fmt.Println("  ○ AI Providers: none detected")
		return
	}

	sort.Slice(aggregates, func(i, j int) bool {
		return aggregates[i].Descriptor.Name < aggregates[j].Descriptor.Name
	})

	fmt.Println("  AI Providers:")
	available := false
	for _, agg := range aggregates {
		info := ctx.Providers[agg.Descriptor.Name]
		display := providerDisplayName(agg.Descriptor.Name)
		status := "○"
		if info.Available {
			status = "✓"
			available = true
		}
		details := []string{
			fmt.Sprintf("%s", strings.ToUpper(string(agg.Descriptor.Type))),
		}
		if info.Version != "" {
			details = append(details, fmt.Sprintf("version %s", info.Version))
		}
		if info.EnvVar != "" {
			details = append(details, fmt.Sprintf("env %s", info.EnvVar))
		}

		fmt.Printf("    %s %s (%s)\n", status, display, strings.Join(details, ", "))
		if info.EnvVar != "" && !info.EnvSet {
			fmt.Printf("       ⚠ Missing: %s\n", info.EnvVar)
		}
		if agg.Status.VisibleReason != "" {
			fmt.Printf("       %s\n", agg.Status.VisibleReason)
		}
	}

	if !available {
		fmt.Println("    ○ No AI providers are available")
	} else {
		recommended := ctx.GetRecommendedProviders()
		if len(recommended) > 0 {
			fmt.Printf("    ⚡ Recommended: %s\n", strings.Join(recommended, ", "))
			if insight := providerStrategyInsight(ctx); insight != "" {
				fmt.Printf("       %s\n", insight)
				fmt.Println("       (These recommendations become the enabled providers in .specular/providers.yaml once initialization finishes.)")
			}
		}
	}
}

func providerDisplayName(name string) string {
	switch name {
	case "claude-code":
		return "Claude Code CLI"
	case "claude-cli":
		return "Claude CLI"
	case "gemini-cli":
		return "Gemini CLI"
	case "codex-cli":
		return "Codex CLI"
	case "gemini":
		return "Google Gemini API"
	case "anthropic":
		return "Anthropic Claude API"
	case "openai":
		return "OpenAI API"
	case "ollama":
		return "Ollama"
	default:
		return strings.Title(name)
	}
}

func filterRouterProviders(recommended []string) []string {
	if len(recommended) == 0 {
		return nil
	}

	supported := map[string]struct{}{
		"ollama":    {},
		"openai":    {},
		"anthropic": {},
		"gemini":    {},
	}

	var result []string
	seen := map[string]struct{}{}
	for _, name := range recommended {
		if _, ok := supported[name]; !ok {
			continue
		}
		if _, added := seen[name]; added {
			continue
		}
		result = append(result, name)
		seen[name] = struct{}{}
	}

	return result
}

func containsString(slice []string, item string) bool {
	for _, val := range slice {
		if val == item {
			return true
		}
	}
	return false
}

// checkExplicitProviderFlags returns strategy if explicit flags are set
func checkExplicitProviderFlags() (string, bool) {
	if initLocal {
		return "local", true
	}
	if initCloud {
		return "cloud", true
	}
	if initProviders != "" {
		return "explicit", true
	}
	return "", false
}

// analyzeProviderAvailability determines what types of providers are available
func analyzeProviderAvailability(providers map[string]detect.ProviderInfo) (hasLocal, hasCloud bool) {
	for name, info := range providers {
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
	return hasLocal, hasCloud
}

func determineProviderStrategy(ctx *detect.Context) string {
	// Explicit flags take precedence
	if strategy, found := checkExplicitProviderFlags(); found {
		return strategy
	}

	// Auto-detect based on context
	if len(ctx.Providers) == 0 {
		return "manual"
	}

	// Check what's available
	hasLocal, hasCloud := analyzeProviderAvailability(ctx.Providers)

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
	fmt.Printf("  Strategy:   %s providers (%s)\n", config.ProviderStrategy, describeProviderStrategy(config.ProviderStrategy))
	fmt.Printf("  Governance: %s (%s)\n", config.Governance, describeGovernance(config.Governance))
	fmt.Printf("  MCP:        %v\n", config.MCPEnabled)
	fmt.Println()
	fmt.Print("Proceed with initialization? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n') //nolint:errcheck // Interactive prompt, empty response on error is acceptable
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
	fmt.Printf("  📄 %s/routing.yaml\n", filepath.Base(config.SpecDir))
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
	// Generate routing.yaml
	routerContent := generateRouterYAML(config)
	if err := os.WriteFile(filepath.Join(config.SpecDir, "routing.yaml"), []byte(routerContent), 0600); err != nil {
		return err
	}
	fmt.Println("✓ Created routing.yaml")

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

	// Generate provider configuration
	if err := createProviderConfig(config.SpecDir, config.Context); err != nil {
		return err
	}

	// Generate governance docs
	if err := generateGovernanceDocs(config); err != nil {
		return err
	}

	return nil
}

func createProviderConfig(specDir string, ctx *detect.Context) error {
	exampleConfig := provider.DefaultProvidersConfig()
	examplePath := filepath.Join(specDir, "providers.yaml.example")
	if err := provider.SaveProvidersConfigExample(exampleConfig, examplePath); err != nil {
		return fmt.Errorf("saving provider example: %w", err)
	}
	fmt.Println("✓ Created providers.yaml.example")

	config := provider.DefaultProvidersConfig()
	if ctx != nil {
		provider.ApplyRecommendedProviders(config, ctx.GetRecommendedProviders())
	}

	targetPath := filepath.Join(specDir, "providers.yaml")
	if err := provider.SaveProvidersConfig(config, targetPath); err != nil {
		return fmt.Errorf("saving provider config: %w", err)
	}

	fmt.Println("✓ Created providers.yaml")
	return nil
}

func generateGovernanceDocs(config *InitConfig) error {
	docCtx := buildDocContext(config)
	docsDir := filepath.Join(config.SpecDir, "docs")
	if err := docgen.WriteDocs(docsDir, docCtx); err != nil {
		return fmt.Errorf("writing governance docs: %w", err)
	}

	fmt.Println("✓ Created governance docs")
	return nil
}

func buildDocContext(config *InitConfig) docgen.DocContext {
	projectName := filepath.Base(config.TargetDir)
	recommended := []string{}
	if config.Context != nil {
		for _, name := range config.Context.GetRecommendedProviders() {
			if desc := provider.DescriptorByName(name); desc != nil {
				recommended = append(recommended, fmt.Sprintf("%s (%s)", providerDisplayName(name), desc.TrustLevel))
				continue
			}
			recommended = append(recommended, providerDisplayName(name))
		}
	}

	docProviders := make([]docgen.DocProvider, 0, len(provider.Descriptors()))
	for _, desc := range provider.Descriptors() {
		capabilities := strings.Join(desc.Capabilities, ", ")
		hints := docgen.FormatDetectionHints(desc.Hints)
		docProviders = append(docProviders, docgen.DocProvider{
			Name:         providerDisplayName(desc.Name),
			Source:       desc.Source,
			TrustLevel:   string(desc.TrustLevel),
			Description:  desc.Description,
			Capabilities: capabilities,
			Hints:        hints,
		})
	}

	features := []string{
		"Document governance level, provider strategy, and recommended providers",
		"Outline the success criteria for governance-first spec generation",
		"Highlight provider capabilities and detection hints for transparency",
	}

	return docgen.DocContext{
		ProjectName:                 projectName,
		Timestamp:                   config.Timestamp,
		Governance:                  config.Governance,
		GovernanceDescription:       describeGovernance(config.Governance),
		ProviderStrategy:            config.ProviderStrategy,
		ProviderStrategyDescription: describeProviderStrategy(config.ProviderStrategy),
		RecommendedProviders:        recommended,
		Providers:                   docProviders,
		Features:                    features,
	}
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

# Budget configuration (in USD)
budget_usd: 50.0

# Performance constraints
max_latency_ms: 60000
prefer_cheap: false

# Fallback and retry configuration
fallback_model: ollama/llama3.2
enable_fallback: true
max_retries: 3
retry_backoff_ms: 1000
retry_max_backoff_ms: 30000

# Context handling
enable_context_validation: true
auto_truncate: false
truncation_strategy: oldest

# Provider configuration
providers:
  - name: ollama
    enabled: %s
    base_url: http://localhost:11434
    models:
      codegen: llama3.2
      fast: llama3.2

  - name: openai
    enabled: %s
    api_key: ${OPENAI_API_KEY}
    models:
      codegen: gpt-4o
      long-context: gpt-4-turbo
      cheap: gpt-4o-mini
      fast: gpt-3.5-turbo

  - name: anthropic
    enabled: %s
    api_key: ${ANTHROPIC_API_KEY}
    models:
      codegen: claude-sonnet-4
      long-context: claude-sonnet-4
      cheap: claude-haiku-3.5
      fast: claude-haiku-3.5

  - name: gemini
    enabled: %s
    api_key: ${GEMINI_API_KEY}
    models:
      codegen: gemini-2.0-flash
      fast: gemini-2.0-flash
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
# Date: %s%s
#
# SCHEMA REFERENCE:
# -----------------
# product: (required) Product name as a string
# goals: (optional) High-level goals for the product
# features: (required) List of feature specifications
#   - id: (required) Unique feature ID (e.g., feat-001)
#   - title: (required) Short feature title
#   - desc: (required) Detailed feature description
#   - priority: (required) P0 (critical), P1 (high), P2 (medium)
#   - success: (required) List of success criteria
#   - trace: (optional) Traceability links (e.g., REQ-001)
#   - refs: (optional) External references/docs
#   - api: (optional) API definitions
#     - method: HTTP method (GET, POST, etc.)
#     - path: API endpoint path
# non_functional: (optional) Non-functional requirements
#   - performance: Performance requirements
#   - security: Security requirements
#   - scalability: Scalability requirements
# acceptance: (optional) Overall acceptance criteria
# milestones: (optional) Development milestones

product: %s

goals:
  - Build a fully functional product with core features
  - Ensure code quality through testing and reviews

features:
  - id: feat-001
    title: Example Feature
    desc: |
      This is an example feature specification.
      Replace with your actual feature descriptions.
    priority: P1
    success:
      - Feature works as described
      - All tests pass
      - Documentation is updated
    trace:
      - REQ-001 Example Requirement

# Uncomment and customize as needed:
# non_functional:
#   performance:
#     - Response time < 200ms for 95th percentile
#   security:
#     - All data encrypted at rest and in transit
#
# acceptance:
#   - All P0 and P1 features complete
#   - Code coverage > 80%%
#
# milestones:
#   - id: mvp
#     name: Minimum Viable Product
#     feature_ids: [feat-001]
#     target_date: "2024-Q2"
`, config.Timestamp.Format("2006-01-02"), detectedLangs, projectName)
}

func generateWebAppSpec(config *InitConfig) string {
	projectName := filepath.Base(config.TargetDir)
	return fmt.Sprintf(`# Specular Product Specification - Web Application
# Template: web-app
# Date: %s

product: %s

goals:
  - Build a modern, responsive web application
  - Ensure accessibility and performance best practices

features:
  - id: feat-frontend
    title: Frontend Setup
    desc: |
      Initialize frontend framework and build system with
      hot reload support for rapid development.
    priority: P0
    success:
      - Development server runs successfully
      - Build system configured
      - Hot reload working
    trace:
      - REQ-001 Frontend Infrastructure

  - id: feat-backend
    title: Backend API
    desc: |
      RESTful API backend service with authentication
      and database connectivity.
    priority: P0
    success:
      - API endpoints respond correctly
      - Database connection established
      - Authentication working
    trace:
      - REQ-002 Backend Infrastructure

  - id: feat-components
    title: UI Components
    desc: |
      Reusable UI component library with accessibility
      support and documentation.
    priority: P1
    success:
      - Component library created
      - Storybook documentation
      - Accessible components
    trace:
      - REQ-003 UI Library
`, config.Timestamp.Format("2006-01-02"), projectName)
}

func generateAPIServiceSpec(config *InitConfig) string {
	projectName := filepath.Base(config.TargetDir)
	return fmt.Sprintf(`# Specular Product Specification - API Service
# Template: api-service
# Date: %s

product: %s

goals:
  - Build a robust, well-documented API service
  - Ensure security and performance best practices

features:
  - id: feat-api
    title: REST API Endpoints
    desc: |
      Core RESTful API endpoints with OpenAPI specification
      and input validation.
    priority: P0
    success:
      - OpenAPI specification complete
      - All endpoints functional
      - Input validation working
    trace:
      - REQ-001 API Endpoints

  - id: feat-auth
    title: API Authentication
    desc: |
      JWT-based authentication with token generation,
      protected endpoint access, and refresh token support.
    priority: P0
    success:
      - JWT tokens generated
      - Protected endpoints secured
      - Refresh tokens working
    trace:
      - REQ-002 Authentication

  - id: feat-database
    title: Database Layer
    desc: |
      Database schema and migrations with optimized
      CRUD operations and proper indexing.
    priority: P0
    success:
      - Schema migrations working
      - CRUD operations complete
      - Indexes optimized
    trace:
      - REQ-003 Data Persistence
`, config.Timestamp.Format("2006-01-02"), projectName)
}

func generateCLIToolSpec(config *InitConfig) string {
	projectName := filepath.Base(config.TargetDir)
	return fmt.Sprintf(`# Specular Product Specification - CLI Tool
# Template: cli-tool
# Date: %s

product: %s

goals:
  - Build a user-friendly command-line tool
  - Provide comprehensive help and documentation

features:
  - id: feat-commands
    title: Command Structure
    desc: |
      CLI command hierarchy with flags, subcommands,
      and comprehensive help text.
    priority: P0
    success:
      - Commands parse correctly
      - Help text complete
      - Flags validated
    trace:
      - REQ-001 CLI Interface

  - id: feat-config
    title: Configuration System
    desc: |
      Config file and environment variable support
      with sensible defaults.
    priority: P1
    success:
      - Config file loading works
      - Environment variables override
      - Sensible defaults
    trace:
      - REQ-002 Configuration

  - id: feat-output
    title: Output Formatting
    desc: |
      Multiple output formats (text, JSON, YAML)
      for different use cases.
    priority: P1
    success:
      - Text output formatted
      - JSON output valid
      - YAML output correct
    trace:
      - REQ-003 Output Formats
`, config.Timestamp.Format("2006-01-02"), projectName)
}

func generateMicroserviceSpec(config *InitConfig) string {
	projectName := filepath.Base(config.TargetDir)
	return fmt.Sprintf(`# Specular Product Specification - Microservice
# Template: microservice
# Date: %s

product: %s

goals:
  - Build a resilient, scalable microservice
  - Ensure proper observability and health monitoring

features:
  - id: feat-api
    title: Service API
    desc: |
      gRPC or REST API endpoints with validated
      service contracts and health checks.
    priority: P0
    success:
      - API endpoints defined
      - Service contract validated
      - Health checks implemented
    trace:
      - REQ-001 Service Interface

  - id: feat-messaging
    title: Message Queue Integration
    desc: |
      Event-driven communication with message queue
      publishing and consumption.
    priority: P1
    success:
      - Message queue connected
      - Events published
      - Event handlers working
    trace:
      - REQ-002 Event Messaging

  - id: feat-observability
    title: Observability
    desc: |
      Logging, metrics, and distributed tracing
      for production monitoring.
    priority: P0
    success:
      - Structured logging configured
      - Metrics exported
      - Distributed tracing enabled
    trace:
      - REQ-003 Observability
`, config.Timestamp.Format("2006-01-02"), projectName)
}

func generateDataPipelineSpec(config *InitConfig) string {
	projectName := filepath.Base(config.TargetDir)
	return fmt.Sprintf(`# Specular Product Specification - Data Pipeline
# Template: data-pipeline
# Date: %s
#
# SCHEMA REFERENCE:
# -----------------
# product: (required) Product name as a string
# goals: (optional) High-level goals for the product
# features: (required) List of feature specifications
#   - id: (required) Unique feature ID (e.g., feat-001)
#   - title: (required) Short feature title
#   - desc: (required) Detailed feature description
#   - priority: (required) P0 (critical), P1 (high), P2 (medium)
#   - success: (required) List of success criteria
#   - trace: (optional) Traceability links (e.g., REQ-001)
#   - refs: (optional) External references
#   - api: (optional) API specifications
# non_functional: (optional) Non-functional requirements
# acceptance: (optional) Overall acceptance criteria
# milestones: (optional) Project milestones

product: %s

goals:
  - Build reliable data pipeline infrastructure
  - Enable efficient data ingestion and transformation
  - Ensure data quality and integrity

features:
  - id: feat-001
    title: Data Ingestion
    desc: |
      Ingest data from various sources including databases,
      APIs, file systems, and streaming sources.
    priority: P0
    success:
      - Data sources connected and configured
      - Ingestion scheduling operational
      - Error handling and retry logic robust
    trace:
      - REQ-001 Data Sources Integration

  - id: feat-002
    title: Data Transformation
    desc: |
      Transform and clean data according to business rules
      and data quality requirements.
    priority: P0
    success:
      - Transformation rules applied correctly
      - Data validation complete
      - Quality checks passing
    trace:
      - REQ-002 Data Processing Rules

  - id: feat-003
    title: Data Storage
    desc: |
      Store processed data in appropriate storage systems
      with proper partitioning and retention policies.
    priority: P0
    success:
      - Storage configured and accessible
      - Data partitioned efficiently
      - Retention policies enforced
    trace:
      - REQ-003 Data Persistence

  - id: feat-004
    title: Pipeline Monitoring
    desc: |
      Monitor pipeline health, data quality metrics,
      and processing performance.
    priority: P1
    success:
      - Metrics dashboards operational
      - Alerting configured
      - Data lineage tracked
    trace:
      - REQ-004 Observability
`, config.Timestamp.Format("2006-01-02"), projectName)
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
		quoted[i] = fmt.Sprintf("%q", item)
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
	if len(ctx.Providers) == 0 {
		return fmt.Errorf("no providers detected")
	}

	recommended := ctx.GetRecommendedProviders()
	if len(recommended) == 0 {
		return fmt.Errorf("no providers detected")
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Smart Provider Setup")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("Recommended providers: %s\n", strings.Join(recommended, ", "))

	routerTargets := filterRouterProviders(recommended)
	if len(routerTargets) == 0 {
		fmt.Println("\nNote: None of the recommended providers are managed by routing.yaml. You can edit routing.yaml later to enable additional providers.")
		return nil
	}

	fmt.Println()
	fmt.Print("Enable recommended router providers? [Y/n]: ")
	response, _ := reader.ReadString('\n') //nolint:errcheck // Interactive prompt, empty response on error is acceptable
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" || response == "y" || response == "yes" {
		routerPath := filepath.Join(specDir, "routing.yaml")
		for _, provider := range routerTargets {
			if err := enableProvider(routerPath, provider); err != nil {
				fmt.Printf("⚠  Failed to enable %s: %v\n", provider, err)
			}
		}
	}

	if len(recommended) != len(routerTargets) {
		var extra []string
		for _, name := range recommended {
			if !containsString(routerTargets, name) {
				extra = append(extra, providerDisplayName(name))
			}
		}
		if len(extra) > 0 {
			fmt.Printf("\nAdditional detected providers (%s) are not listed in routing.yaml and must be configured manually if needed.\n", strings.Join(extra, ", "))
		}
	}

	return nil
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

	choice, _ := reader.ReadString('\n') //nolint:errcheck // Interactive prompt, empty response on error is acceptable
	choice = strings.TrimSpace(choice)

	routerDir, err := sanitizeSpecDir(specDir)
	if err != nil {
		return err
	}
	routerPath := filepath.Join(routerDir, "routing.yaml")

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

var _ = runProviderSetup

func enableProvider(routerPath string, providerName string) error {
	// Read routing.yaml
	content, err := os.ReadFile(routerPath)
	if err != nil {
		return fmt.Errorf("failed to read routing.yaml: %w", err)
	}

	contentStr := string(content)

	// Find and replace "enabled: false" with "enabled: true" for the provider
	searchPattern := fmt.Sprintf("- name: %s", providerName)
	providerIndex := strings.Index(contentStr, searchPattern)
	if providerIndex == -1 {
		return fmt.Errorf("provider %s not found in routing.yaml", providerName)
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
	if writeErr := os.WriteFile(routerPath, []byte(contentStr), 0600); writeErr != nil {
		return fmt.Errorf("failed to update routing.yaml: %w", writeErr)
	}

	fmt.Printf("✓ Enabled provider: %s\n", providerName)
	return nil
}

func sanitizeSpecDir(specDir string) (string, error) {
	if specDir == "" {
		specDir = "."
	}
	cleanDir := filepath.Clean(specDir)
	absDir, err := filepath.Abs(cleanDir)
	if err != nil {
		return "", fmt.Errorf("invalid spec directory %q: %w", specDir, err)
	}
	return absDir, nil
}

func printSmartSuccessMessage(config *InitConfig) {
	projectName := filepath.Base(config.TargetDir)

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  ✨ Project Initialized Successfully!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Configuration files created:")
	fmt.Println("  • .specular/routing.yaml    - AI provider routing")
	fmt.Println("  • .specular/policy.yaml    - Security policies")
	fmt.Println("  • .specular/spec.yaml      - Product specification")
	fmt.Println("  • .specular/settings.json  - Project settings")
	fmt.Println("  • .specular/providers.yaml - Provider configuration")
	fmt.Println("  • .specular/docs          - Governance source documents (PRD, Vision, Roadmap, TDD)")
	fmt.Println("  • .specular/providers.yaml.example - Provider configuration example")
	fmt.Println()

	if config.Template != "" {
		fmt.Printf("Template: %s\n", config.Template)
	}
	fmt.Printf("Provider Strategy: %s (%s)\n", config.ProviderStrategy, describeProviderStrategy(config.ProviderStrategy))
	if rationale := describeStrategyRationale(config.Context, config.ProviderStrategy); rationale != "" {
		fmt.Printf("Strategy rationale: %s\n", rationale)
	}
	fmt.Printf("Governance Level: %s (%s)\n", config.Governance, describeGovernance(config.Governance))
	fmt.Printf("Provider configuration: %s (edit to adjust which descriptors are enabled)\n", filepath.Join(config.SpecDir, "providers.yaml"))
	fmt.Println()

	fmt.Println("Next steps:")
	fmt.Println()
	fmt.Println("  1. Check your system health:")
	fmt.Println("     $ specular doctor")
	fmt.Println()
	fmt.Println("  2. Review your configuration:")
	fmt.Println("     $ cat .specular/routing.yaml")
	fmt.Println("     $ specular route show")
	fmt.Println()
	fmt.Println("  3. Create your spec (interactive):")
	fmt.Println("     $ specular spec new --tui")
	fmt.Println()
	fmt.Println("  4. Or edit the spec template:")
	fmt.Println("     $ vim .specular/spec.yaml")
	fmt.Println()
	fmt.Println("  5. Generate a plan:")
	fmt.Println("     $ specular plan create")
	fmt.Println()
	fmt.Println("  6. Execute your plan:")
	fmt.Println("     $ specular build run")
	fmt.Println()

	fmt.Printf("Project: %s\n", projectName)
	fmt.Println("Documentation: https://github.com/felixgeelhaar/specular")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

func describeProviderStrategy(strategy string) string {
	switch strategy {
	case "local":
		return "Prefer local providers (Ollama/CLI)"
	case "cloud":
		return "Prefer cloud APIs (OpenAI, Anthropic, Gemini)"
	case "hybrid":
		return "Mix of local and cloud providers detected"
	case "explicit":
		return "Using the explicit provider list you requested"
	case "manual":
		return "Manual provider selection"
	default:
		return "Custom strategy"
	}
}

func describeGovernance(level string) string {
	if val, ok := ux.GovernanceLevels.FindByValue(level); ok && val.Description != "" {
		return val.Description
	}
	return "Governance level"
}

func describeStrategyRationale(ctx *detect.Context, strategy string) string {
	if ctx == nil {
		return ""
	}

	hasLocal, hasCloud := analyzeProviderAvailability(ctx.Providers)
	switch strategy {
	case "local":
		if hasLocal && !hasCloud {
			return "Local providers (Ollama/CLI wrappers) were detected while cloud APIs lacked keys, so the local stack is preferred."
		}
		if hasLocal && hasCloud {
			return "Both local and cloud providers are available, but the local stack is preferred (via --local or defaults)."
		}
		return "Local strategy enforced explicitly."
	case "cloud":
		keys := cloudAPIProvidersWithKeys(ctx)
		if len(keys) > 0 {
			return fmt.Sprintf("Cloud API keys detected (%s); the cloud providers are enabled.", strings.Join(keys, ", "))
		}
		return "Cloud strategy enforced explicitly."
	case "hybrid":
		if hasLocal && hasCloud {
			return "A hybrid mix of local models and cloud APIs was detected, so both types stay available."
		}
		return "Hybrid strategy requested but one provider type was missing; adjust .specular/providers.yaml if needed."
	case "explicit":
		return "You provided an explicit provider list via --providers."
	case "manual":
		return "No providers were auto-detected or requested; add providers to .specular/providers.yaml manually or run 'specular provider add'."
	default:
		return ""
	}
}

func cloudAPIProvidersWithKeys(ctx *detect.Context) []string {
	if ctx == nil {
		return nil
	}
	var names []string
	for _, name := range []string{"openai", "anthropic", "gemini"} {
		if info, ok := ctx.Providers[name]; ok && info.Available && info.EnvSet {
			names = append(names, providerDisplayName(name))
		}
	}
	return names
}
