package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/specular/internal/quickstart"
	"github.com/felixgeelhaar/specular/internal/templates"
)

var (
	quickstartVerify   bool
	quickstartDemo     bool
	quickstartProvider string
	quickstartForce    bool
	quickstartTemplate string
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Get started with Specular in under 5 minutes",
	Long: `Automatically configure Specular with smart defaults.

This command detects your environment and creates a minimal working
configuration with zero mandatory prompts.

Detection order:
  1. Environment variables (ANTHROPIC_API_KEY, OPENAI_API_KEY, GEMINI_API_KEY)
  2. Local providers (Ollama running on localhost:11434)
  3. Claude CLI (if installed)

Available templates:
  rest-api   - RESTful API with CRUD, auth, and health endpoints
  cli-tool   - Command-line application with subcommands and config
  web-app    - Full-stack web application with frontend and backend
  library    - Reusable Go library with clean API and testing

Examples:
  specular quickstart              # Auto-detect and configure
  specular quickstart --verify     # Include verification test
  specular quickstart --demo       # Run demo after setup
  specular quickstart --provider anthropic  # Force specific provider
  specular quickstart --template rest-api   # Create spec from template
  specular quickstart --force      # Overwrite existing config`,
	RunE: runQuickstart,
}

func init() {
	quickstartCmd.Flags().BoolVar(&quickstartVerify, "verify", false, "run verification test after setup")
	quickstartCmd.Flags().BoolVar(&quickstartDemo, "demo", false, "run a demo goal after setup")
	quickstartCmd.Flags().StringVar(&quickstartProvider, "provider", "", "force specific provider (anthropic, openai, gemini, ollama)")
	quickstartCmd.Flags().BoolVarP(&quickstartForce, "force", "f", false, "overwrite existing configuration")
	quickstartCmd.Flags().StringVarP(&quickstartTemplate, "template", "t", "", "create spec from template (rest-api, cli-tool, web-app, library)")

	rootCmd.AddCommand(quickstartCmd)
}

func runQuickstart(cmd *cobra.Command, args []string) error {
	start := time.Now()

	fmt.Println()
	fmt.Println("  ╔═══════════════════════════════════════╗")
	fmt.Println("  ║       SPECULAR Quickstart             ║")
	fmt.Println("  ╚═══════════════════════════════════════╝")
	fmt.Println()

	// Check for existing configuration
	if quickstart.ConfigExists() && !quickstartForce {
		fmt.Println("  ⚠️  Existing configuration detected in .specular/")
		fmt.Println()
		fmt.Println("  Use --force to overwrite existing configuration,")
		fmt.Println("  or use 'specular init' for interactive setup.")
		fmt.Println()
		return nil
	}

	// Backup existing config if force
	if quickstart.ConfigExists() && quickstartForce {
		backupDir, err := quickstart.BackupExistingConfig()
		if err != nil {
			return fmt.Errorf("failed to backup existing config: %w", err)
		}
		fmt.Printf("  📦 Backed up existing config to %s\n", backupDir)
	}

	fmt.Println("  Detecting environment...")

	// Detect environment
	var detection *quickstart.DetectionResult
	var provider *quickstart.ProviderSelection
	var err error

	if quickstartProvider != "" {
		// User specified a provider
		provider, err = quickstart.DetectProviderByName(quickstartProvider)
		if err != nil {
			fmt.Printf("    ❌ Provider %s: %v\n", quickstartProvider, err)
			fmt.Println()
			return nil
		}

		// Still detect docker status
		detection, _ = quickstart.DetectEnvironment()
		detection.Provider = provider
	} else {
		// Auto-detect
		detection, err = quickstart.DetectEnvironment()
		if err != nil {
			if noProviderErr, ok := err.(*quickstart.NoProviderError); ok {
				fmt.Println()
				fmt.Println("  ❌ No AI provider found")
				fmt.Println()
				fmt.Printf("  %s\n", noProviderErr.Suggestion)
				fmt.Println()
				fmt.Println("  Quick setup options:")
				fmt.Println("    • Set ANTHROPIC_API_KEY environment variable")
				fmt.Println("    • Set OPENAI_API_KEY environment variable")
				fmt.Println("    • Install Ollama: https://ollama.com")
				fmt.Println()
				return nil
			}
			return fmt.Errorf("environment detection failed: %w", err)
		}
	}

	// Display detection results
	displayDetectionResults(detection)

	fmt.Println()
	fmt.Println("  Creating configuration...")

	// Generate minimal config
	files, err := quickstart.GenerateMinimalConfig(detection.Provider, detection.Docker)
	if err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}

	fmt.Printf("    ✓ Created %s\n", files.RouterPath)
	fmt.Printf("    ✓ Created %s\n", files.PolicyPath)
	fmt.Printf("    ✓ Created %s\n", files.SettingsPath)

	// Handle template if specified
	if quickstartTemplate != "" {
		fmt.Println()
		fmt.Printf("  Creating spec from template '%s'...\n", quickstartTemplate)

		registry, err := templates.NewRegistry()
		if err != nil {
			return fmt.Errorf("failed to load templates: %w", err)
		}

		tmpl, err := registry.Get(quickstartTemplate)
		if err != nil {
			fmt.Printf("    ❌ Template not found: %s\n", quickstartTemplate)
			fmt.Println()
			fmt.Println("  Available templates:")
			for _, t := range registry.List() {
				fmt.Printf("    • %s - %s\n", t.ID, t.Description)
			}
			fmt.Println()
			return nil
		}

		// Convert template to ProductSpec
		productSpec := tmpl.ToProductSpec("")

		// Write spec to .specular/spec.yaml
		specPath := filepath.Join(".specular", "spec.yaml")
		specData, err := yaml.Marshal(productSpec)
		if err != nil {
			return fmt.Errorf("failed to marshal spec: %w", err)
		}

		if err := os.WriteFile(specPath, specData, 0600); err != nil {
			return fmt.Errorf("failed to write spec: %w", err)
		}

		fmt.Printf("    ✓ Created %s (from %s template)\n", specPath, tmpl.Name)
	}

	// Optional verification
	if quickstartVerify {
		fmt.Println()
		fmt.Println("  Running verification test...")

		result, err := quickstart.VerifyProvider(detection.Provider)
		if err != nil {
			fmt.Printf("    ⚠️  Verification error: %v\n", err)
		} else if result.Success {
			fmt.Printf("    ✓ Provider response: %s\n", truncateString(result.Response, 50))
			fmt.Printf("    ✓ Latency: %v\n", result.Latency.Round(time.Millisecond))
		} else {
			fmt.Printf("    ⚠️  Verification failed: %s\n", result.Error)
			fmt.Println("    (Configuration was created - provider may still work)")
		}
	}

	// Summary
	elapsed := time.Since(start)
	fmt.Println()
	fmt.Printf("  ═══════════════════════════════════════\n")
	fmt.Printf("  Setup complete in %.1fs!\n", elapsed.Seconds())
	fmt.Printf("  ═══════════════════════════════════════\n")
	fmt.Println()
	fmt.Println("  You're ready to use Specular!")
	fmt.Println()
	fmt.Println("  Quick start:")
	fmt.Println("    specular spec new --tui   # Create specification")
	fmt.Println("    specular plan create      # Generate execution plan")
	fmt.Println("    specular build run        # Execute the plan")
	fmt.Println("    specular drift            # Check for implementation drift")
	fmt.Println()
	fmt.Println("  Useful commands:")
	fmt.Println("    specular doctor           # Check system health")
	fmt.Println("    specular status           # Show project status")
	fmt.Println("    specular auto -h          # See auto mode options")
	fmt.Println()

	// Optional demo
	if quickstartDemo {
		fmt.Println("  Running demo...")
		fmt.Println()
		if err := runQuickstartDemo(); err != nil {
			fmt.Printf("  Demo encountered an issue: %v\n", err)
			fmt.Println("  This doesn't affect your setup - you're still ready to go!")
			fmt.Println()
		}
	}

	return nil
}

// runQuickstartDemo demonstrates the Specular workflow
func runQuickstartDemo() error {
	fmt.Println("  ┌─────────────────────────────────────────────────┐")
	fmt.Println("  │           Specular Workflow Demo                │")
	fmt.Println("  └─────────────────────────────────────────────────┘")
	fmt.Println()

	// Step 1: Show the typical workflow
	fmt.Println("  The typical Specular workflow:")
	fmt.Println()
	fmt.Println("  1️⃣  specular spec new --tui")
	fmt.Println("      → Capture requirements and generate a structured specification")
	fmt.Println("      → Validate and lock the spec for planning")
	fmt.Println()
	fmt.Println("  2️⃣  specular plan create")
	fmt.Println("      → Generate an execution plan from the locked spec")
	fmt.Println()
	fmt.Println("  3️⃣  specular build run")
	fmt.Println("      → Execute the plan in the configured runtime")
	fmt.Println()

	fmt.Println("  4️⃣  specular drift")
	fmt.Println("      → Compares spec/plan to implementation")
	fmt.Println("      → Reports any drift or gaps")
	fmt.Println()

	fmt.Println("  5️⃣  specular doctor")
	fmt.Println("      → Checks system health")
	fmt.Println("      → Validates configuration")
	fmt.Println()

	// Step 2: Run doctor as a live demo
	fmt.Println("  Running 'specular doctor' to verify your setup...")
	fmt.Println()

	// Execute doctor command
	doctorCmd.SetArgs([]string{})
	if err := doctorCmd.Execute(); err != nil {
		return fmt.Errorf("doctor check failed: %w", err)
	}

	fmt.Println()
	fmt.Println("  ✓ Demo complete! Try running:")
	fmt.Println("    specular auto \"Create a hello world REST API\"")
	fmt.Println()

	return nil
}

// displayDetectionResults shows the detected environment
func displayDetectionResults(detection *quickstart.DetectionResult) {
	// Docker status
	if detection.Docker.Available {
		fmt.Printf("    ✓ Docker:    %s %s (running)\n",
			detection.Docker.Runtime,
			detection.Docker.Version)
	} else if detection.Docker.Warning != "" {
		fmt.Printf("    ⚠️  Docker:    %s\n", detection.Docker.Warning)
	} else {
		fmt.Println("    ⚠️  Docker:    Not detected")
	}

	// Provider status
	if detection.Provider != nil {
		fmt.Printf("    ✓ Provider:  %s (%s)\n",
			detection.Provider.Name,
			detection.Provider.Reason)
	}
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
