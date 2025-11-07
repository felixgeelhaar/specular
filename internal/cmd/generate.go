package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/internal/router"
)

var generateCmd = &cobra.Command{
	Use:   "generate [prompt]",
	Short: "Generate AI content using configured providers",
	Long: `Generate AI content using the configured providers and intelligent model selection.
The router will automatically select the best model based on complexity, budget, and availability.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		providerConfigPath := cmd.Flags().Lookup("provider-config").Value.String()
		routerConfigPath := cmd.Flags().Lookup("router-config").Value.String()
		modelHint := cmd.Flags().Lookup("model-hint").Value.String()
		systemPrompt := cmd.Flags().Lookup("system").Value.String()
		complexityStr := cmd.Flags().Lookup("complexity").Value.String()
		complexity, _ := strconv.Atoi(complexityStr) //nolint:errcheck // Has default value
		priority := cmd.Flags().Lookup("priority").Value.String()
		temperatureStr := cmd.Flags().Lookup("temperature").Value.String()
		temperature, _ := strconv.ParseFloat(temperatureStr, 64) //nolint:errcheck // Has default value
		maxTokensStr := cmd.Flags().Lookup("max-tokens").Value.String()
		maxTokens, _ := strconv.Atoi(maxTokensStr) //nolint:errcheck // Has default value
		stream := cmd.Flags().Lookup("stream").Value.String() == "true"
		verbose := cmd.Flags().Lookup("verbose").Value.String() == "true"

		// Use default paths if not specified
		if providerConfigPath == "" {
			providerConfigPath = defaultProviderConfigPath
		}

		// Check if provider config exists
		if _, err := os.Stat(providerConfigPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Provider configuration not found at %s\n", providerConfigPath)
			fmt.Fprintln(os.Stderr, "Run 'ai-dev provider init' to create one.")
			return fmt.Errorf("provider config not found")
		}

		// Load providers
		if verbose {
			fmt.Fprintf(os.Stderr, "Loading providers from %s...\n", providerConfigPath)
		}

		registry, err := provider.LoadRegistryFromConfig(providerConfigPath)
		if err != nil {
			return fmt.Errorf("failed to load providers: %w", err)
		}

		if verbose {
			providerNames := registry.List()
			fmt.Fprintf(os.Stderr, "Loaded %d provider(s): %s\n", len(providerNames), strings.Join(providerNames, ", "))
		}

		// Load or create router config
		var routerConfig *router.RouterConfig
		if routerConfigPath != "" {
			routerConfig, err = router.LoadConfig(routerConfigPath)
			if err != nil {
				return fmt.Errorf("failed to load router config: %w", err)
			}
		} else {
			// Load provider config to get strategy settings
			providerConfig, loadErr := provider.LoadProvidersConfig(providerConfigPath)
			if loadErr != nil {
				return fmt.Errorf("failed to load provider config: %w", loadErr)
			}

			// Create router config from provider strategy
			routerConfig = &router.RouterConfig{
				BudgetUSD:    providerConfig.Strategy.Budget.MaxCostPerDay,
				MaxLatencyMs: providerConfig.Strategy.Performance.MaxLatencyMs,
				PreferCheap:  providerConfig.Strategy.Performance.PreferCheap,
			}

			// Set defaults if not specified
			if routerConfig.BudgetUSD == 0 {
				routerConfig.BudgetUSD = 20.0
			}
			if routerConfig.MaxLatencyMs == 0 {
				routerConfig.MaxLatencyMs = 60000
			}
		}

		// Create router with providers
		r, err := router.NewRouterWithProviders(routerConfig, registry)
		if err != nil {
			return fmt.Errorf("failed to create router: %w", err)
		}

		if verbose {
			budget := r.GetBudget()
			fmt.Fprintf(os.Stderr, "Router initialized: budget=$%.2f, max_latency=%dms\n",
				budget.LimitUSD, routerConfig.MaxLatencyMs)
		}

		// Join args as prompt
		prompt := strings.Join(args, " ")

		// Estimate context size (rough estimate: ~4 chars per token)
		estimatedTokens := len(prompt) / 4
		if len(systemPrompt) > 0 {
			estimatedTokens += len(systemPrompt) / 4
		}

		// Build generate request
		req := router.GenerateRequest{
			Prompt:       prompt,
			SystemPrompt: systemPrompt,
			ModelHint:    modelHint,
			Complexity:   complexity,
			Priority:     priority,
			Temperature:  temperature,
			MaxTokens:    maxTokens,
			ContextSize:  estimatedTokens,
			TaskID:       fmt.Sprintf("cli-generate-%d", time.Now().Unix()),
		}

		// Generate
		ctx := context.Background()

		if stream {
			return runStreamingGenerate(ctx, r, req, verbose)
		} else {
			return runGenerate(ctx, r, req, verbose)
		}
	},
}

func runGenerate(ctx context.Context, r *router.Router, req router.GenerateRequest, verbose bool) error {
	startTime := time.Now()

	if verbose {
		fmt.Fprintln(os.Stderr, "\nGenerating...")
	}

	resp, err := r.Generate(ctx, req)
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	if resp.Error != "" {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", resp.Error)
	}

	// Print response
	fmt.Println(resp.Content)

	// Print metadata if verbose
	if verbose {
		elapsed := time.Since(startTime)
		fmt.Fprintln(os.Stderr, "\n"+strings.Repeat("-", 60))
		fmt.Fprintf(os.Stderr, "Model:          %s (%s)\n", resp.Model, resp.Provider)
		fmt.Fprintf(os.Stderr, "Tokens:         %d (in: %d, out: %d)\n",
			resp.TokensUsed, resp.InputTokens, resp.OutputTokens)
		fmt.Fprintf(os.Stderr, "Cost:           $%.6f\n", resp.CostUSD)
		fmt.Fprintf(os.Stderr, "Latency:        %v (total: %v)\n", resp.Latency, elapsed)
		fmt.Fprintf(os.Stderr, "Selection:      %s\n", resp.SelectionReason)
		if resp.FinishReason != "" {
			fmt.Fprintf(os.Stderr, "Finish Reason:  %s\n", resp.FinishReason)
		}

		// Print budget status
		budget := r.GetBudget()
		fmt.Fprintf(os.Stderr, "\nBudget:         $%.2f spent, $%.2f remaining (%.1f%% used)\n",
			budget.SpentUSD, budget.RemainingUSD,
			(budget.SpentUSD/budget.LimitUSD)*100)
	}

	return nil
}

func runStreamingGenerate(ctx context.Context, r *router.Router, req router.GenerateRequest, verbose bool) error {
	startTime := time.Now()

	if verbose {
		fmt.Fprintln(os.Stderr, "\nStreaming...")
	}

	chunkChan, err := r.Stream(ctx, req)
	if err != nil {
		return fmt.Errorf("streaming failed: %w", err)
	}

	var totalContent string
	var lastChunk *router.StreamChunk

	for chunk := range chunkChan {
		if chunk.Error != nil {
			return fmt.Errorf("stream error: %w", chunk.Error)
		}

		// Print the delta (incremental text)
		if chunk.Delta != "" {
			fmt.Print(chunk.Delta)
			os.Stdout.Sync() //nolint:errcheck,gosec // Flush output immediately for streaming effect
		}

		totalContent = chunk.Content
		lastChunk = &chunk
	}

	// Ensure we end with a newline
	if !strings.HasSuffix(totalContent, "\n") {
		fmt.Println()
	}

	// Print metadata if verbose
	if verbose && lastChunk != nil && lastChunk.Done {
		elapsed := time.Since(startTime)
		fmt.Fprintln(os.Stderr, "\n"+strings.Repeat("-", 60))
		fmt.Fprintf(os.Stderr, "Streaming completed in %v\n", elapsed)
		fmt.Fprintf(os.Stderr, "Total content length: %d characters\n", len(totalContent))

		// Print budget status
		budget := r.GetBudget()
		fmt.Fprintf(os.Stderr, "\nBudget:         $%.2f spent, $%.2f remaining (%.1f%% used)\n",
			budget.SpentUSD, budget.RemainingUSD,
			(budget.SpentUSD/budget.LimitUSD)*100)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(generateCmd)

	// Configuration flags
	generateCmd.Flags().String("provider-config", "", "Path to provider config (default: .specular/providers.yaml)")
	generateCmd.Flags().String("router-config", "", "Path to router config (optional)")

	// Model selection hints
	generateCmd.Flags().String("model-hint", "", "Model hint (codegen, agentic, fast, cheap, long-context)")
	generateCmd.Flags().Int("complexity", 5, "Task complexity (1-10)")
	generateCmd.Flags().String("priority", "P1", "Task priority (P0, P1, P2)")

	// Generation parameters
	generateCmd.Flags().String("system", "", "System prompt")
	generateCmd.Flags().Float64("temperature", 0.7, "Temperature (0.0-1.0)")
	generateCmd.Flags().Int("max-tokens", 0, "Max tokens to generate (0 = provider default)")

	// Output options
	generateCmd.Flags().Bool("stream", false, "Enable streaming output")
	generateCmd.Flags().Bool("verbose", false, "Show detailed metadata")
}
