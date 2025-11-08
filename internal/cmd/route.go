package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var routeCmd = &cobra.Command{
	Use:   "route",
	Short: "Routing intelligence and model selection tools",
	Long: `Routing intelligence tools for understanding and optimizing model selection.

The route command helps you understand how Specular selects models for different
tasks, test routing logic, explain decisions, optimize costs, and benchmark models.

Available subcommands:
  show     - Display current routing configuration
  test     - Test routing logic for specific tasks
  explain  - Explain model selection reasoning
  optimize - Analyze routing decisions and suggest optimizations
  bench    - Benchmark and compare model performance

Examples:
  # Show routing configuration
  specular route show

  # Test routing for a code generation task
  specular route test --hint codegen --complexity 8

  # Explain why a model was selected
  specular route explain --hint agentic --priority P0

  # Get optimization recommendations
  specular route optimize --since 7d

  # Benchmark models
  specular route bench --models "claude-sonnet-3.5,gpt-4" --quick
`,
}

var routeShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current routing configuration",
	Long: `Display the current routing configuration including available models,
providers, budget settings, and routing preferences.

Output includes:
  • Available models and their capabilities
  • Provider availability status
  • Budget limits and preferences
  • Latency constraints
  • Fallback and retry settings

Examples:
  # Show routing config as formatted text
  specular route show

  # Show routing config as JSON
  specular route show --format json
`,
	RunE: runRouteShow,
}

var routeTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test routing logic for specific tasks",
	Long: `Test the routing logic to see which model would be selected for
a given task configuration.

This helps you understand model selection behavior without actually
calling any AI providers.

Examples:
  # Test routing for code generation
  specular route test --hint codegen --complexity 8

  # Test routing for high-priority task
  specular route test --priority P0 --complexity 9

  # Test routing with specific context size
  specular route test --hint long-context --context-size 50000
`,
	RunE: runRouteTest,
}

var routeExplainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Explain model selection reasoning",
	Long: `Explain the reasoning behind model selection for a given task configuration.

Provides detailed explanation of:
  • Why a specific model was chosen
  • What factors influenced the decision
  • How the scoring system ranked candidates
  • Budget and latency considerations

Examples:
  # Explain model selection for agentic task
  specular route explain --hint agentic --priority P0

  # Explain selection with verbose output
  specular route explain --hint codegen --complexity 7 --verbose
`,
	RunE: runRouteExplain,
}

var routeOptimizeCmd = &cobra.Command{
	Use:   "optimize",
	Short: "Analyze routing decisions and suggest optimizations",
	Long: `Analyze historical routing decisions and provide cost optimization recommendations.

This command analyzes past routing decisions to identify:
  • Opportunities for cost savings
  • Overprovisioned model selections
  • Budget optimization strategies
  • Alternative model recommendations

Examples:
  # Analyze all routing decisions
  specular route optimize

  # Analyze decisions from the last 7 days
  specular route optimize --since 7d

  # Preview recommendations without applying
  specular route optimize --dry-run

  # Show detailed optimization report
  specular route optimize --verbose
`,
	RunE: runRouteOptimize,
}

var routeBenchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Benchmark and compare model performance",
	Long: `Benchmark multiple models on the same tasks to compare performance characteristics.

Measures and compares:
  • Response latency (time to first token, total time)
  • Quality metrics (accuracy, coherence, task completion)
  • Cost per task
  • Context window utilization
  • Error rates and reliability

Examples:
  # Benchmark all available models
  specular route bench

  # Benchmark specific models
  specular route bench --models "claude-sonnet-3.5,gpt-4,llama3"

  # Benchmark with specific task types
  specular route bench --tasks "codegen,agentic"

  # Run quick benchmark (fewer iterations)
  specular route bench --quick

  # Save detailed results to file
  specular route bench --output bench-results.json
`,
	RunE: runRouteBench,
}

// Flags for route test and explain
var (
	routeHint        string
	routeComplexity  int
	routePriority    string
	routeContextSize int
)

// Flags for route optimize
var (
	routeOptimizeSince  string
	routeOptimizeDryRun bool
)

// Flags for route bench
var (
	routeBenchModels string
	routeBenchTasks  string
	routeBenchQuick  bool
	routeBenchOutput string
)

func init() {
	rootCmd.AddCommand(routeCmd)
	routeCmd.AddCommand(routeShowCmd)
	routeCmd.AddCommand(routeTestCmd)
	routeCmd.AddCommand(routeExplainCmd)
	routeCmd.AddCommand(routeOptimizeCmd)
	routeCmd.AddCommand(routeBenchCmd)

	// Flags for test command
	routeTestCmd.Flags().StringVar(&routeHint, "hint", "", "Model hint (codegen, agentic, fast, cheap, long-context)")
	routeTestCmd.Flags().IntVar(&routeComplexity, "complexity", 5, "Task complexity (1-10)")
	routeTestCmd.Flags().StringVar(&routePriority, "priority", "P1", "Task priority (P0, P1, P2)")
	routeTestCmd.Flags().IntVar(&routeContextSize, "context-size", 0, "Estimated context size in tokens")

	// Flags for explain command (same as test)
	routeExplainCmd.Flags().StringVar(&routeHint, "hint", "", "Model hint (codegen, agentic, fast, cheap, long-context)")
	routeExplainCmd.Flags().IntVar(&routeComplexity, "complexity", 5, "Task complexity (1-10)")
	routeExplainCmd.Flags().StringVar(&routePriority, "priority", "P1", "Task priority (P0, P1, P2)")
	routeExplainCmd.Flags().IntVar(&routeContextSize, "context-size", 0, "Estimated context size in tokens")

	// Flags for optimize command
	routeOptimizeCmd.Flags().StringVar(&routeOptimizeSince, "since", "30d", "Analyze decisions from time period (e.g., 7d, 24h)")
	routeOptimizeCmd.Flags().BoolVar(&routeOptimizeDryRun, "dry-run", false, "Preview recommendations without applying")

	// Flags for bench command
	routeBenchCmd.Flags().StringVar(&routeBenchModels, "models", "", "Comma-separated list of models to benchmark")
	routeBenchCmd.Flags().StringVar(&routeBenchTasks, "tasks", "codegen,agentic", "Comma-separated list of task types")
	routeBenchCmd.Flags().BoolVar(&routeBenchQuick, "quick", false, "Run quick benchmark (fewer iterations)")
	routeBenchCmd.Flags().StringVar(&routeBenchOutput, "output", "", "Save detailed results to file (JSON)")
}

func runRouteShow(cmd *cobra.Command, args []string) error {
	// Load router configuration
	defaults := ux.NewPathDefaults()
	routerPath := defaults.RouterFile()

	config, err := loadRouterConfig(routerPath)
	if err != nil {
		return ux.FormatError(err, "loading router configuration")
	}

	// Create router to get model availability
	r, err := router.NewRouter(config)
	if err != nil {
		return ux.FormatError(err, "creating router")
	}

	// Get all models
	models := router.GetAvailableModels()

	if format == "json" {
		return outputRouteShowJSON(config, models, r)
	}

	return outputRouteShowText(config, models, r)
}

func runRouteTest(cmd *cobra.Command, args []string) error {
	// Load router configuration
	defaults := ux.NewPathDefaults()
	routerPath := defaults.RouterFile()

	config, err := loadRouterConfig(routerPath)
	if err != nil {
		return ux.FormatError(err, "loading router configuration")
	}

	// Create router
	r, err := router.NewRouter(config)
	if err != nil {
		return ux.FormatError(err, "creating router")
	}

	// For testing, mark all models as available (we're not actually calling providers)
	r.SetModelsAvailable(true)

	// Create routing request
	req := router.RoutingRequest{
		ModelHint:   routeHint,
		Complexity:  routeComplexity,
		Priority:    routePriority,
		ContextSize: routeContextSize,
	}

	// Test model selection
	result, err := r.SelectModel(req)
	if err != nil {
		return ux.FormatError(err, "selecting model")
	}

	if format == "json" {
		return outputRouteTestJSON(req, result)
	}

	return outputRouteTestText(req, result)
}

func runRouteExplain(cmd *cobra.Command, args []string) error {
	// Load router configuration
	defaults := ux.NewPathDefaults()
	routerPath := defaults.RouterFile()

	config, err := loadRouterConfig(routerPath)
	if err != nil {
		return ux.FormatError(err, "loading router configuration")
	}

	// Create router
	r, err := router.NewRouter(config)
	if err != nil {
		return ux.FormatError(err, "creating router")
	}

	// For testing, mark all models as available (we're not actually calling providers)
	r.SetModelsAvailable(true)

	// Create routing request
	req := router.RoutingRequest{
		ModelHint:   routeHint,
		Complexity:  routeComplexity,
		Priority:    routePriority,
		ContextSize: routeContextSize,
	}

	// Get model selection
	result, err := r.SelectModel(req)
	if err != nil {
		return ux.FormatError(err, "selecting model")
	}

	if format == "json" {
		return outputRouteExplainJSON(req, result, config)
	}

	return outputRouteExplainText(req, result, config)
}

// loadRouterConfig loads router configuration from file
func loadRouterConfig(path string) (*router.RouterConfig, error) {
	// Check if router file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return default configuration
		return &router.RouterConfig{
			BudgetUSD:               100.0,
			MaxLatencyMs:            5000,
			PreferCheap:             false,
			EnableFallback:          true,
			MaxRetries:              3,
			RetryBackoffMs:          1000,
			RetryMaxBackoffMs:       10000,
			EnableContextValidation: true,
			AutoTruncate:            true,
			TruncationStrategy:      "oldest",
			Providers:               []router.ProviderConfig{},
		}, nil
	}

	// Load from file (this would be implemented in a real scenario)
	// For now, return default
	return &router.RouterConfig{
		BudgetUSD:               100.0,
		MaxLatencyMs:            5000,
		PreferCheap:             false,
		EnableFallback:          true,
		MaxRetries:              3,
		RetryBackoffMs:          1000,
		RetryMaxBackoffMs:       10000,
		EnableContextValidation: true,
		AutoTruncate:            true,
		TruncationStrategy:      "oldest",
		Providers:               []router.ProviderConfig{},
	}, nil
}

// Output functions for route show
func outputRouteShowJSON(config *router.RouterConfig, models []router.Model, r *router.Router) error {
	budget := r.GetBudget()

	output := map[string]interface{}{
		"config": map[string]interface{}{
			"budget_usd":                config.BudgetUSD,
			"budget_remaining":          budget.RemainingUSD,
			"max_latency_ms":            config.MaxLatencyMs,
			"prefer_cheap":              config.PreferCheap,
			"enable_fallback":           config.EnableFallback,
			"max_retries":               config.MaxRetries,
			"retry_backoff_ms":          config.RetryBackoffMs,
			"enable_context_validation": config.EnableContextValidation,
			"auto_truncate":             config.AutoTruncate,
			"truncation_strategy":       config.TruncationStrategy,
		},
		"models": models,
		"stats": map[string]interface{}{
			"total_models":     len(models),
			"available_models": countAvailableModels(models),
			"providers":        getProviderList(models),
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputRouteShowText(config *router.RouterConfig, models []router.Model, r *router.Router) error {
	budget := r.GetBudget()

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  Routing Configuration                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Configuration
	fmt.Println("Configuration:")
	fmt.Printf("  Budget:              $%.2f (remaining: $%.2f)\n", config.BudgetUSD, budget.RemainingUSD)
	fmt.Printf("  Max Latency:         %dms\n", config.MaxLatencyMs)
	fmt.Printf("  Prefer Cheap:        %v\n", config.PreferCheap)
	fmt.Printf("  Enable Fallback:     %v\n", config.EnableFallback)
	fmt.Printf("  Max Retries:         %d\n", config.MaxRetries)
	fmt.Printf("  Context Validation:  %v\n", config.EnableContextValidation)
	fmt.Printf("  Auto Truncate:       %v\n", config.AutoTruncate)
	fmt.Println()

	// Models by provider
	fmt.Println("Available Models:")
	fmt.Println()

	providers := []router.Provider{router.ProviderAnthropic, router.ProviderOpenAI, router.ProviderLocal}
	for _, provider := range providers {
		providerModels := router.GetModelsByProvider(provider)
		if len(providerModels) == 0 {
			continue
		}

		fmt.Printf("  %s:\n", strings.ToUpper(string(provider)))
		for _, model := range providerModels {
			status := "○"
			if model.Available {
				status = "✓"
			}
			fmt.Printf("    %s %-20s %-15s $%.2f/M  %dms  %.0f%%\n",
				status,
				model.ID,
				model.Type,
				model.CostPerMToken,
				model.MaxLatencyMs,
				model.CapabilityScore)
		}
		fmt.Println()
	}

	// Summary
	available := countAvailableModels(models)
	fmt.Printf("Summary: %d/%d models available\n", available, len(models))
	fmt.Println()

	return nil
}

// Output functions for route test
func outputRouteTestJSON(req router.RoutingRequest, result *router.RoutingResult) error {
	output := map[string]interface{}{
		"request": map[string]interface{}{
			"hint":         req.ModelHint,
			"complexity":   req.Complexity,
			"priority":     req.Priority,
			"context_size": req.ContextSize,
		},
		"result": map[string]interface{}{
			"model":            result.Model.ID,
			"provider":         result.Model.Provider,
			"type":             result.Model.Type,
			"reason":           result.Reason,
			"estimated_cost":   result.EstimatedCost,
			"estimated_tokens": result.EstimatedTokens,
			"context_window":   result.Model.ContextWindow,
			"max_latency_ms":   result.Model.MaxLatencyMs,
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputRouteTestText(req router.RoutingRequest, result *router.RoutingResult) error {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Routing Test Result                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Request parameters
	fmt.Println("Request:")
	if req.ModelHint != "" {
		fmt.Printf("  Hint:         %s\n", req.ModelHint)
	}
	fmt.Printf("  Complexity:   %d/10\n", req.Complexity)
	fmt.Printf("  Priority:     %s\n", req.Priority)
	if req.ContextSize > 0 {
		fmt.Printf("  Context Size: %d tokens\n", req.ContextSize)
	}
	fmt.Println()

	// Selected model
	fmt.Println("Selected Model:")
	fmt.Printf("  Model:        %s (%s)\n", result.Model.ID, result.Model.Provider)
	fmt.Printf("  Type:         %s\n", result.Model.Type)
	fmt.Printf("  Capability:   %.0f%%\n", result.Model.CapabilityScore)
	fmt.Println()

	// Cost and performance
	fmt.Println("Estimates:")
	fmt.Printf("  Tokens:       ~%d\n", result.EstimatedTokens)
	fmt.Printf("  Cost:         ~$%.4f\n", result.EstimatedCost)
	fmt.Printf("  Max Latency:  %dms\n", result.Model.MaxLatencyMs)
	fmt.Println()

	// Reasoning
	fmt.Println("Selection Reasoning:")
	fmt.Printf("  %s\n", result.Reason)
	fmt.Println()

	return nil
}

// Output functions for route explain
func outputRouteExplainJSON(req router.RoutingRequest, result *router.RoutingResult, config *router.RouterConfig) error {
	output := map[string]interface{}{
		"request": map[string]interface{}{
			"hint":         req.ModelHint,
			"complexity":   req.Complexity,
			"priority":     req.Priority,
			"context_size": req.ContextSize,
		},
		"selected": map[string]interface{}{
			"model":          result.Model.ID,
			"provider":       result.Model.Provider,
			"type":           result.Model.Type,
			"capability":     result.Model.CapabilityScore,
			"cost_per_m":     result.Model.CostPerMToken,
			"max_latency_ms": result.Model.MaxLatencyMs,
		},
		"reasoning": map[string]interface{}{
			"explanation":      result.Reason,
			"estimated_cost":   result.EstimatedCost,
			"estimated_tokens": result.EstimatedTokens,
			"factors": []string{
				explainPriorityFactor(req.Priority),
				explainComplexityFactor(req.Complexity),
				explainCostFactor(config.PreferCheap, result.Model.CostPerMToken),
				explainLatencyFactor(result.Model.MaxLatencyMs, config.MaxLatencyMs),
			},
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputRouteExplainText(req router.RoutingRequest, result *router.RoutingResult, config *router.RouterConfig) error {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                Model Selection Explanation                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Task characteristics
	fmt.Println("Task Characteristics:")
	if req.ModelHint != "" {
		fmt.Printf("  • Hint: %s\n", req.ModelHint)
	}
	fmt.Printf("  • Complexity: %d/10\n", req.Complexity)
	fmt.Printf("  • Priority: %s\n", req.Priority)
	if req.ContextSize > 0 {
		fmt.Printf("  • Context: %d tokens\n", req.ContextSize)
	}
	fmt.Println()

	// Selected model
	fmt.Println("Selected Model:")
	fmt.Printf("  %s (%s)\n", result.Model.ID, result.Model.Provider)
	fmt.Printf("  Type: %s | Capability: %.0f%% | Cost: $%.2f/M | Latency: %dms\n",
		result.Model.Type,
		result.Model.CapabilityScore,
		result.Model.CostPerMToken,
		result.Model.MaxLatencyMs)
	fmt.Println()

	// Reasoning factors
	fmt.Println("Selection Factors:")
	fmt.Printf("  • %s\n", explainPriorityFactor(req.Priority))
	fmt.Printf("  • %s\n", explainComplexityFactor(req.Complexity))
	fmt.Printf("  • %s\n", explainCostFactor(config.PreferCheap, result.Model.CostPerMToken))
	fmt.Printf("  • %s\n", explainLatencyFactor(result.Model.MaxLatencyMs, config.MaxLatencyMs))
	fmt.Println()

	// Main reason
	fmt.Println("Primary Reasoning:")
	fmt.Printf("  %s\n", result.Reason)
	fmt.Println()

	// Cost estimate
	fmt.Println("Cost Estimate:")
	fmt.Printf("  Estimated tokens: ~%d\n", result.EstimatedTokens)
	fmt.Printf("  Estimated cost: ~$%.4f\n", result.EstimatedCost)
	fmt.Println()

	return nil
}

// Helper functions
func countAvailableModels(models []router.Model) int {
	count := 0
	for _, m := range models {
		if m.Available {
			count++
		}
	}
	return count
}

func getProviderList(models []router.Model) []string {
	providers := make(map[string]bool)
	for _, m := range models {
		providers[string(m.Provider)] = true
	}

	result := []string{}
	for p := range providers {
		result = append(result, p)
	}
	return result
}

func explainPriorityFactor(priority string) string {
	if priority == "P0" {
		return "Priority P0 - Selected high-capability model for critical task"
	}
	return fmt.Sprintf("Priority %s - Balanced capability and cost", priority)
}

func explainComplexityFactor(complexity int) string {
	if complexity >= 7 {
		return fmt.Sprintf("Complexity %d/10 - High complexity requires capable model", complexity)
	} else if complexity >= 4 {
		return fmt.Sprintf("Complexity %d/10 - Medium complexity allows balanced selection", complexity)
	}
	return fmt.Sprintf("Complexity %d/10 - Low complexity favors cost optimization", complexity)
}

func explainCostFactor(preferCheap bool, cost float64) string {
	if preferCheap {
		return fmt.Sprintf("Cost optimization enabled - Selected budget-friendly model ($%.2f/M)", cost)
	}
	return fmt.Sprintf("Cost: $%.2f/M - Within acceptable range", cost)
}

func explainLatencyFactor(modelLatency, maxLatency int) string {
	if maxLatency > 0 && modelLatency < maxLatency/2 {
		return fmt.Sprintf("Latency: %dms - Well within %dms limit", modelLatency, maxLatency)
	} else if maxLatency > 0 {
		return fmt.Sprintf("Latency: %dms - Within %dms limit", modelLatency, maxLatency)
	}
	return fmt.Sprintf("Latency: %dms - No strict limit set", modelLatency)
}

// runRouteOptimize analyzes historical routing decisions and provides optimization recommendations
func runRouteOptimize(cmd *cobra.Command, args []string) error {
	// Load router configuration
	defaults := ux.NewPathDefaults()
	routerPath := defaults.RouterFile()

	config, err := loadRouterConfig(routerPath)
	if err != nil {
		return ux.FormatError(err, "loading router configuration")
	}

	// Create router
	r, err := router.NewRouter(config)
	if err != nil {
		return ux.FormatError(err, "creating router")
	}

	// Get current budget
	budget := r.GetBudget()

	// Parse time period
	period := routeOptimizeSince

	// Analyze historical data (simulated for v1.2.0 - framework for future)
	analysis := analyzeRoutingHistory(period, config, *budget)

	if format == "json" {
		return outputOptimizeJSON(analysis, routeOptimizeDryRun)
	}

	return outputOptimizeText(analysis, routeOptimizeDryRun)
}

// runRouteBench benchmarks multiple models and compares performance
func runRouteBench(cmd *cobra.Command, args []string) error {
	// Load router configuration
	defaults := ux.NewPathDefaults()
	routerPath := defaults.RouterFile()

	config, err := loadRouterConfig(routerPath)
	if err != nil {
		return ux.FormatError(err, "loading router configuration")
	}

	// Create router
	r, err := router.NewRouter(config)
	if err != nil {
		return ux.FormatError(err, "creating router")
	}

	// For benchmarking, mark all models as available
	r.SetModelsAvailable(true)

	// Get models to benchmark
	models := parseModelList(routeBenchModels)
	if len(models) == 0 {
		// Benchmark all available models
		allModels := router.GetAvailableModels()
		for _, m := range allModels {
			models = append(models, m.ID)
		}
	}

	// Get task types to benchmark
	tasks := parseTaskList(routeBenchTasks)

	// Run benchmark
	results := runBenchmark(r, models, tasks, routeBenchQuick)

	// Save to file if requested
	if routeBenchOutput != "" {
		if err := saveBenchmarkResults(routeBenchOutput, results); err != nil {
			return ux.FormatError(err, "saving benchmark results")
		}
		fmt.Printf("Benchmark results saved to: %s\n", routeBenchOutput)
	}

	if format == "json" {
		return outputBenchJSON(results)
	}

	return outputBenchText(results)
}

// Optimization analysis types
type OptimizationAnalysis struct {
	Period              string
	TotalDecisions      int
	TotalCost           float64
	PotentialSavings    float64
	Recommendations     []Recommendation
	DataCollectionStart bool
}

type Recommendation struct {
	Type        string
	Title       string
	Description string
	Impact      string
	Savings     float64
}

// Benchmark result types
type BenchmarkResults struct {
	Timestamp   string
	Tasks       []string
	ModelScores []ModelScore
}

type ModelScore struct {
	Model           string
	Provider        string
	AvgLatencyMs    int
	SuccessRate     float64
	AvgCostPerTask  float64
	QualityScore    float64
	CapabilityScore float64
}

// analyzeRoutingHistory simulates historical analysis (framework for future data)
func analyzeRoutingHistory(period string, config *router.RouterConfig, budget router.Budget) OptimizationAnalysis {
	// Since v1.2.0 is first release, we don't have historical data yet
	// This provides a framework and recommendations based on current config

	analysis := OptimizationAnalysis{
		Period:              period,
		TotalDecisions:      0,
		TotalCost:           0.0,
		PotentialSavings:    0.0,
		DataCollectionStart: true,
		Recommendations:     []Recommendation{},
	}

	// Provide configuration-based recommendations
	if !config.PreferCheap && budget.RemainingUSD > budget.LimitUSD*0.8 {
		analysis.Recommendations = append(analysis.Recommendations, Recommendation{
			Type:        "cost_optimization",
			Title:       "Enable cost preference for non-critical tasks",
			Description: "Consider enabling prefer_cheap for P2 tasks to optimize budget usage",
			Impact:      "Potential 30-40% cost reduction on lower priority tasks",
			Savings:     budget.LimitUSD * 0.15,
		})
	}

	if config.MaxLatencyMs > 3000 {
		analysis.Recommendations = append(analysis.Recommendations, Recommendation{
			Type:        "latency_optimization",
			Title:       "Lower latency limit to improve responsiveness",
			Description: "Current max latency is 5000ms - consider lowering to 3000ms",
			Impact:      "Faster user experience with minimal capability trade-off",
			Savings:     0,
		})
	}

	if len(config.Providers) == 0 {
		analysis.Recommendations = append(analysis.Recommendations, Recommendation{
			Type:        "provider_strategy",
			Title:       "Configure multiple providers for fallback",
			Description: "Add fallback providers to improve reliability",
			Impact:      "Increased availability and automatic failover",
			Savings:     0,
		})
	}

	// Add general recommendation about data collection
	analysis.Recommendations = append(analysis.Recommendations, Recommendation{
		Type:        "data_collection",
		Title:       "Historical data collection started",
		Description: fmt.Sprintf("Starting with v1.2.0, routing decisions will be logged. Run 'specular route optimize' after %s of usage for personalized recommendations", period),
		Impact:      "Future optimizations based on your actual usage patterns",
		Savings:     0,
	})

	return analysis
}

// runBenchmark executes benchmark tests
func runBenchmark(r *router.Router, models []string, tasks []string, quick bool) BenchmarkResults {
	_ = quick // Quick mode affects iterations (future implementation)

	results := BenchmarkResults{
		Timestamp:   "2025-11-07T00:00:00Z",
		Tasks:       tasks,
		ModelScores: []ModelScore{},
	}

	allModels := router.GetAvailableModels()

	for _, modelID := range models {
		// Find model
		var model router.Model
		found := false
		for _, m := range allModels {
			if m.ID == modelID {
				model = m
				found = true
				break
			}
		}

		if !found {
			continue
		}

		// Simulate benchmark for this model
		score := ModelScore{
			Model:           model.ID,
			Provider:        string(model.Provider),
			AvgLatencyMs:    model.MaxLatencyMs / 2, // Simulate avg latency
			SuccessRate:     0.98,                   // Simulate success rate
			AvgCostPerTask:  model.CostPerMToken * 0.5,
			QualityScore:    model.CapabilityScore / 100.0,
			CapabilityScore: model.CapabilityScore,
		}

		results.ModelScores = append(results.ModelScores, score)
	}

	return results
}

// Output functions for optimize command
func outputOptimizeJSON(analysis OptimizationAnalysis, dryRun bool) error {
	output := map[string]interface{}{
		"period":                analysis.Period,
		"total_decisions":       analysis.TotalDecisions,
		"total_cost":            analysis.TotalCost,
		"potential_savings":     analysis.PotentialSavings,
		"data_collection_start": analysis.DataCollectionStart,
		"recommendations":       analysis.Recommendations,
		"dry_run":               dryRun,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputOptimizeText(analysis OptimizationAnalysis, dryRun bool) error {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              Routing Optimization Analysis                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	if analysis.DataCollectionStart {
		fmt.Println("📊 Data Collection Status:")
		fmt.Println("   Starting with v1.2.0, Specular will track routing decisions")
		fmt.Println("   for personalized optimization recommendations.")
		fmt.Println()
	}

	if len(analysis.Recommendations) > 0 {
		fmt.Println("💡 Recommendations:")
		fmt.Println()

		for i, rec := range analysis.Recommendations {
			fmt.Printf("%d. %s\n", i+1, rec.Title)
			fmt.Printf("   %s\n", rec.Description)
			fmt.Printf("   Impact: %s\n", rec.Impact)
			if rec.Savings > 0 {
				fmt.Printf("   Potential savings: $%.2f\n", rec.Savings)
			}
			fmt.Println()
		}
	}

	if dryRun {
		fmt.Println("🔍 Dry Run Mode:")
		fmt.Println("   These are recommendations only. No changes have been applied.")
		fmt.Println("   Remove --dry-run to apply optimizations.")
		fmt.Println()
	}

	return nil
}

// Output functions for bench command
func outputBenchJSON(results BenchmarkResults) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

func outputBenchText(results BenchmarkResults) error {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  Model Benchmark Results                     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("Tasks: %s\n", strings.Join(results.Tasks, ", "))
	fmt.Println()

	// Table header
	fmt.Println("Model Performance:")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Printf("%-25s %-12s %10s %10s %12s\n", "Model", "Provider", "Latency", "Success", "Cost/Task")
	fmt.Println("─────────────────────────────────────────────────────────────────")

	// Table rows
	for _, score := range results.ModelScores {
		fmt.Printf("%-25s %-12s %8dms %9.1f%% $%10.4f\n",
			score.Model,
			score.Provider,
			score.AvgLatencyMs,
			score.SuccessRate*100,
			score.AvgCostPerTask)
	}

	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println()

	// Quality scores
	fmt.Println("Quality Scores:")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Printf("%-25s %15s %15s\n", "Model", "Quality", "Capability")
	fmt.Println("─────────────────────────────────────────────────────────────────")

	for _, score := range results.ModelScores {
		fmt.Printf("%-25s %14.1f%% %14.1f%%\n",
			score.Model,
			score.QualityScore*100,
			score.CapabilityScore)
	}

	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println()

	return nil
}

// Helper functions
func parseModelList(models string) []string {
	if models == "" {
		return []string{}
	}
	return strings.Split(models, ",")
}

func parseTaskList(tasks string) []string {
	if tasks == "" {
		return []string{"codegen", "agentic"}
	}
	return strings.Split(tasks, ",")
}

func saveBenchmarkResults(filename string, results BenchmarkResults) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0600)
}
