package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/internal/router"
)

var routeCmd = &cobra.Command{
	Use:   "route",
	Short: "Manage AI model routing and provider selection",
	Long: `View routing decisions, list available models and providers, and override provider selection.

The route command helps you understand and control how Specular selects AI models for tasks.

Subcommands:
  list      List all available models and providers with costs
  override  Override provider selection for the current session
  explain   Explain routing logic and model selection decisions

Examples:
  specular route list
  specular route override anthropic
  specular route explain codegen`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// routeListCmd lists all available models and providers
var routeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available models and providers with costs",
	Long: `List all AI models and providers configured in Specular with their costs,
capabilities, and availability status.

Examples:
  specular route list                # List all models
  specular route list --available    # List only available models
  specular route list --provider anthropic  # List models for specific provider`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		showAvailable, _ := cmd.Flags().GetBool("available")
		filterProvider, _ := cmd.Flags().GetString("provider")

		// Load provider registry
		providerConfigPath := ".specular/providers.yaml"
		registry, err := provider.LoadRegistryWithAutoDiscovery(providerConfigPath)
		if err != nil {
			// If config doesn't exist, use empty registry
			registry = provider.NewRegistry()
		}

		// Get provider names
		providerNames := registry.List()

		// Create router to get models with availability
		routerConfig := &router.RouterConfig{
			BudgetUSD:    1000.0,
			MaxLatencyMs: 60000,
		}
		r, err := router.NewRouterWithProviders(routerConfig, registry)
		if err != nil {
			return fmt.Errorf("failed to create router: %w", err)
		}

		// Get all models
		models := router.GetAvailableModels()

		// Filter models
		var filteredModels []router.Model
		for _, m := range models {
			// Filter by availability if requested
			if showAvailable && !m.Available {
				continue
			}

			// Filter by provider if requested
			if filterProvider != "" && string(m.Provider) != filterProvider {
				continue
			}

			filteredModels = append(filteredModels, m)
		}

		if len(filteredModels) == 0 {
			fmt.Println("No models found matching criteria")
			return nil
		}

		// Display header
		fmt.Println("=== Available AI Models ===")
		fmt.Println()

		// Group by provider
		providerGroups := make(map[router.Provider][]router.Model)
		for _, m := range filteredModels {
			providerGroups[m.Provider] = append(providerGroups[m.Provider], m)
		}

		// Display each provider group
		for _, prov := range []router.Provider{router.ProviderAnthropic, router.ProviderOpenAI, router.ProviderLocal} {
			models := providerGroups[prov]
			if len(models) == 0 {
				continue
			}

			// Provider header
			providerLoaded := false
			for _, name := range providerNames {
				if name == string(prov) {
					providerLoaded = true
					break
				}
			}

			statusIcon := "✅"
			if !providerLoaded {
				statusIcon = "❌"
			}

			fmt.Printf("%s Provider: %s\n", statusIcon, prov)
			if !providerLoaded {
				fmt.Printf("   (Not configured - see '.specular/providers.yaml')\n")
			}
			fmt.Println()

			// Display models for this provider
			for _, m := range models {
				availIcon := "✅"
				if !m.Available {
					availIcon = "❌"
				}

				fmt.Printf("  %s %s\n", availIcon, m.ID)
				fmt.Printf("     Name: %s\n", m.Name)
				fmt.Printf("     Type: %s\n", m.Type)
				fmt.Printf("     Context: %d tokens\n", m.ContextWindow)
				fmt.Printf("     Cost: $%.2f per million tokens\n", m.CostPerMToken)
				fmt.Printf("     Latency: ~%dms\n", m.MaxLatencyMs)
				fmt.Printf("     Capability: %.0f/100\n", m.CapabilityScore)
				fmt.Println()
			}
		}

		// Display budget info
		budget := r.GetBudget()
		fmt.Println("Router Budget:")
		fmt.Printf("  Limit: $%.2f\n", budget.LimitUSD)
		fmt.Printf("  Spent: $%.4f\n", budget.SpentUSD)
		fmt.Printf("  Remaining: $%.4f\n", budget.RemainingUSD)

		return nil
	},
}

// routeOverrideCmd overrides provider selection
var routeOverrideCmd = &cobra.Command{
	Use:   "override <provider>",
	Short: "Override provider selection for current session",
	Long: `Override the default provider selection logic to use a specific provider.

This sets an environment variable for the current session that forces Specular
to use only models from the specified provider.

Valid providers: anthropic, openai, local

Examples:
  specular route override anthropic    # Use only Anthropic models
  specular route override openai       # Use only OpenAI models
  specular route override local        # Use only local models (Ollama)

To clear the override:
  unset SPECULAR_PROVIDER_OVERRIDE`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerName := strings.ToLower(args[0])

		// Validate provider
		validProviders := map[string]bool{
			"anthropic": true,
			"openai":    true,
			"local":     true,
		}

		if !validProviders[providerName] {
			return fmt.Errorf("invalid provider '%s'. Valid providers: anthropic, openai, local", providerName)
		}

		// Check if provider is configured
		providerConfigPath := ".specular/providers.yaml"
		registry, err := provider.LoadRegistryWithAutoDiscovery(providerConfigPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not load provider config: %v\n", err)
		} else {
			providerNames := registry.List()
			providerLoaded := false
			for _, name := range providerNames {
				if name == providerName {
					providerLoaded = true
					break
				}
			}

			if !providerLoaded {
				fmt.Fprintf(os.Stderr, "⚠️  Warning: Provider '%s' is not configured in .specular/providers.yaml\n", providerName)
				fmt.Fprintf(os.Stderr, "   The override will be set, but no models will be available.\n\n")
			}
		}

		// Display instruction for setting environment variable
		fmt.Printf("To override provider selection to '%s', set this environment variable:\n\n", providerName)
		fmt.Printf("  export SPECULAR_PROVIDER_OVERRIDE=%s\n\n", providerName)
		fmt.Println("This will force all routing decisions to use only models from this provider.")
		fmt.Println()
		fmt.Println("To clear the override:")
		fmt.Println("  unset SPECULAR_PROVIDER_OVERRIDE")
		fmt.Println()
		fmt.Println("Note: The override applies to the current shell session only.")

		return nil
	},
}

// routeExplainCmd explains routing logic
var routeExplainCmd = &cobra.Command{
	Use:   "explain <task-type>",
	Short: "Explain routing logic for a task type",
	Long: `Explain how Specular would route a specific type of task to an AI model.

This helps you understand the routing decision logic without actually executing a task.

Valid task types:
  codegen       Code generation tasks
  long-context  Tasks requiring large context windows
  agentic       Multi-step reasoning tasks
  fast          Quick, low-latency tasks
  cheap         Budget-friendly tasks

Examples:
  specular route explain codegen       # Explain routing for code generation
  specular route explain agentic       # Explain routing for agentic tasks
  specular route explain fast          # Explain routing for fast tasks`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskType := strings.ToLower(args[0])

		// Validate task type
		validTypes := map[string]string{
			"codegen":      "Code generation specialists",
			"long-context": "Models with large context windows",
			"agentic":      "Multi-step reasoning models",
			"fast":         "Low-latency models",
			"cheap":        "Budget-friendly models",
		}

		description, valid := validTypes[taskType]
		if !valid {
			return fmt.Errorf("invalid task type '%s'. Valid types: codegen, long-context, agentic, fast, cheap", taskType)
		}

		// Load provider registry
		providerConfigPath := ".specular/providers.yaml"
		registry, err := provider.LoadRegistryWithAutoDiscovery(providerConfigPath)
		if err != nil {
			registry = provider.NewRegistry()
		}

		// Create router
		routerConfig := &router.RouterConfig{
			BudgetUSD:    1000.0,
			MaxLatencyMs: 60000,
			PreferCheap:  false,
		}
		r, err := router.NewRouterWithProviders(routerConfig, registry)
		if err != nil {
			return fmt.Errorf("failed to create router: %w", err)
		}

		// Create routing request
		req := router.RoutingRequest{
			ModelHint:   taskType,
			Complexity:  5, // Medium complexity
			Priority:    "P1",
			ContextSize: 4000,
		}

		// Get routing result
		result, err := r.SelectModel(context.Background(), req)
		if err != nil {
			return fmt.Errorf("routing failed: %w", err)
		}

		// Display explanation
		fmt.Printf("=== Routing Explanation: %s ===\n", taskType)
		fmt.Println()
		fmt.Printf("Task Type: %s\n", description)
		fmt.Println()

		fmt.Println("Selected Model:")
		fmt.Printf("  ID: %s\n", result.Model.ID)
		fmt.Printf("  Provider: %s\n", result.Model.Provider)
		fmt.Printf("  Name: %s\n", result.Model.Name)
		fmt.Printf("  Type: %s\n", result.Model.Type)
		fmt.Println()

		fmt.Println("Selection Reason:")
		fmt.Printf("  %s\n", result.Reason)
		fmt.Println()

		fmt.Println("Cost Estimate:")
		fmt.Printf("  Estimated tokens: %d\n", result.EstimatedTokens)
		fmt.Printf("  Estimated cost: $%.4f\n", result.EstimatedCost)
		fmt.Println()

		fmt.Println("Model Details:")
		fmt.Printf("  Context window: %d tokens\n", result.Model.ContextWindow)
		fmt.Printf("  Cost per Mtok: $%.2f\n", result.Model.CostPerMToken)
		fmt.Printf("  Max latency: %dms\n", result.Model.MaxLatencyMs)
		fmt.Printf("  Capability score: %.0f/100\n", result.Model.CapabilityScore)
		fmt.Println()

		fmt.Println("Alternative Models:")
		// Get models of the same type
		models := router.GetAvailableModels()
		var alternatives []router.Model
		for _, m := range models {
			if string(m.Type) == taskType && m.ID != result.Model.ID && m.Available {
				alternatives = append(alternatives, m)
			}
		}

		if len(alternatives) == 0 {
			fmt.Println("  No alternative models available for this task type")
		} else {
			for _, m := range alternatives {
				fmt.Printf("  • %s (%s) - $%.2f/Mtok, %dms latency\n",
					m.ID, m.Provider, m.CostPerMToken, m.MaxLatencyMs)
			}
		}

		return nil
	},
}

func init() {
	// Add subcommands
	routeCmd.AddCommand(routeListCmd)
	routeCmd.AddCommand(routeOverrideCmd)
	routeCmd.AddCommand(routeExplainCmd)

	// Flags for route list
	routeListCmd.Flags().Bool("available", false, "Show only available models")
	routeListCmd.Flags().String("provider", "", "Filter by provider (anthropic, openai, local)")

	rootCmd.AddCommand(routeCmd)
}
