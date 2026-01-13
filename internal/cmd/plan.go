package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/internal/telemetry"
	"github.com/felixgeelhaar/specular/internal/tui"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var planCmd = &cobra.Command{
	Use:     "plan",
	Aliases: []string{"p"},
	Short:   "Manage execution plans",
	Long: `Generate, review, and manage execution plans from specifications.

Use 'specular plan create' to generate a new plan from a specification.
Use 'specular plan review' to interactively review a plan.
Use 'specular plan visualize' to visualize plan as graph.
Use 'specular plan validate' to validate plan structure.
Use 'specular plan explain' to understand routing decisions.

Typical flow:
  spec lock -> plan create -> build run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var planCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create execution plan from spec",
	Long: `Create a task DAG (Directed Acyclic Graph) from a specification.
The plan includes task dependencies, priorities, skill requirements, and
expected hashes for drift detection.

You can optionally create a plan for a specific feature using --feature.`,
	RunE: runPlanCreate,
}

var planReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Interactively review execution plan",
	Long: `Launch an interactive terminal UI to review the execution plan.

The TUI allows you to:
- View task dependencies as a graph
- Inspect task details and requirements
- Modify task priorities
- Approve or reject the plan`,
	RunE: runPlanReview,
}

var planExplainCmd = &cobra.Command{
	Use:   "explain [step]",
	Short: "Explain routing decisions for plan step",
	Long: `Explain the reasoning behind routing decisions for a specific plan step.

Shows:
- Why a particular model was selected
- Skill requirements that influenced the decision
- Cost and latency considerations
- Alternative models that were considered`,
	RunE: runPlanExplain,
}

var planVisualizeCmd = &cobra.Command{
	Use:   "visualize",
	Short: "Visualize execution plan as graph",
	Long: `Visualize the execution plan as a dependency graph.

Shows:
- Task dependencies and relationships
- Execution order and parallelization opportunities
- Critical path through the plan
- Task priorities and estimated complexity`,
	RunE: runPlanVisualize,
}

var planValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate plan structure and dependencies",
	Long: `Validate the execution plan for structural correctness.

Checks:
- No circular dependencies
- All task dependencies exist
- Valid task priorities
- Proper skill assignments
- Estimated complexity values`,
	RunE: runPlanValidate,
}

func runPlanCreate(cmd *cobra.Command, args []string) error {
	// Start distributed tracing span for plan create command
	ctx, span := telemetry.StartCommandSpan(cmd.Context(), "plan.create")
	defer span.End()

	startTime := time.Now()

	defaults := ux.NewPathDefaults()
	specPath := cmd.Flags().Lookup("in").Value.String()
	lockPath := cmd.Flags().Lookup("lock").Value.String()
	out := cmd.Flags().Lookup("out").Value.String()
	estimate := cmd.Flags().Lookup("estimate").Value.String() == "true"
	featureID := cmd.Flags().Lookup("feature").Value.String()

	// Check for TUI mode
	useTUI := false
	if flag := cmd.Flags().Lookup("tui"); flag != nil {
		useTUI = flag.Value.String() == "true"
	}

	// Use smart defaults if not changed
	if !cmd.Flags().Changed("in") {
		specPath = defaults.SpecFile()
	}
	if !cmd.Flags().Changed("lock") {
		lockPath = defaults.SpecLockFile()
	}
	if !cmd.Flags().Changed("out") {
		out = defaults.PlanFile()
	}

	// If TUI mode is enabled, run with BuildTUI wrapper (reusing build TUI for plan creation)
	if useTUI {
		tuiConfig := tui.BuildTUIConfig{
			SpecFile:  specPath,
			OutputDir: ".",
			Format:    "json",
			Validate:  true,
			ShowDiff:  false,
		}

		runner := tui.NewBuildTUIRunner(tuiConfig, nil)
		return runner.Run(func(buildTUI *tui.BuildTUI) error {
			// Load and generate plan within TUI context
			buildTUI.OnLoadSpec(true, specPath, nil)

			// Load spec
			s, err := spec.LoadSpec(specPath)
			if err != nil {
				buildTUI.OnLoadSpec(false, specPath, err)
				return fmt.Errorf("failed to load spec: %w", err)
			}
			buildTUI.OnLoadSpec(false, specPath, nil)

			// Validate spec
			buildTUI.OnValidate(true, 0, 0, nil)

			// Load SpecLock
			lock, err := spec.LoadSpecLock(lockPath)
			if err != nil {
				buildTUI.OnValidate(false, 0, 1, err)
				return fmt.Errorf("failed to load SpecLock: %w", err)
			}
			buildTUI.OnValidate(false, 0, 0, nil)

			// Resolve dependencies
			buildTUI.OnResolve(true, 0, nil)

			// Generate plan
			opts := plan.GenerateOptions{
				SpecLock:           lock,
				EstimateComplexity: estimate,
			}

			p, err := plan.Generate(ctx, s, opts)
			if err != nil {
				buildTUI.OnResolve(false, 0, err)
				return fmt.Errorf("failed to generate plan: %w", err)
			}
			buildTUI.OnResolve(false, len(p.Tasks), nil)

			// Generate output
			buildTUI.OnGenerate(true, "json", nil)

			// Save plan
			if err := plan.SavePlan(p, out); err != nil {
				buildTUI.OnGenerate(false, "json", err)
				return fmt.Errorf("failed to save plan: %w", err)
			}
			buildTUI.OnGenerate(false, "json", nil)

			// Write complete
			buildTUI.OnWriteComplete(1, 0, nil)
			buildTUI.Log(tui.LogLevelInfo, fmt.Sprintf("Generated plan with %d tasks", len(p.Tasks)))

			return nil
		})
	}

	// Record span attributes
	span.SetAttributes(
		attribute.String("spec_file", specPath),
		attribute.String("lock_file", lockPath),
		attribute.String("plan_file", out),
		attribute.Bool("estimate_complexity", estimate),
	)
	if featureID != "" {
		span.SetAttributes(attribute.String("feature_id", featureID))
	}

	// Validate required files with helpful errors
	if err := ux.ValidateRequiredFile(specPath, "Spec file", "specular spec new"); err != nil {
		telemetry.RecordError(span, err)
		return ux.EnhanceError(err)
	}
	if err := ux.ValidateRequiredFile(lockPath, "SpecLock file", "specular spec lock"); err != nil {
		telemetry.RecordError(span, err)
		return ux.EnhanceError(err)
	}

	// Load spec
	s, err := spec.LoadSpec(specPath)
	if err != nil {
		telemetry.RecordError(span, err)
		return ux.FormatError(err, "loading spec file")
	}

	// Load SpecLock
	lock, err := spec.LoadSpecLock(lockPath)
	if err != nil {
		telemetry.RecordError(span, err)
		return ux.FormatError(err, "loading SpecLock file")
	}

	// Generate plan
	opts := plan.GenerateOptions{
		SpecLock:           lock,
		EstimateComplexity: estimate,
	}

	// If feature flag is set, filter to specific feature
	if featureID != "" {
		// Verify feature exists and filter spec
		found := false
		var filteredFeatures []spec.Feature
		for _, f := range s.Features {
			if string(f.ID) == featureID {
				found = true
				filteredFeatures = append(filteredFeatures, f)
				break
			}
		}
		if !found {
			return fmt.Errorf("feature '%s' not found in spec", featureID)
		}

		fmt.Printf("Generating plan for feature: %s\n", featureID)
		// Create filtered spec with only the requested feature
		s = &spec.ProductSpec{
			Product:       s.Product,
			Goals:         s.Goals,
			Features:      filteredFeatures,
			NonFunctional: s.NonFunctional,
			Acceptance:    s.Acceptance,
			Milestones:    s.Milestones,
		}
	}

	p, err := plan.Generate(ctx, s, opts)
	if err != nil {
		telemetry.RecordError(span, err)
		return ux.FormatError(err, "generating plan")
	}

	// Save plan
	if saveErr := plan.SavePlan(p, out); saveErr != nil {
		telemetry.RecordError(span, saveErr)
		return ux.FormatError(saveErr, "saving plan file")
	}

	fmt.Printf("✓ Generated plan with %d tasks\n", len(p.Tasks))
	for _, task := range p.Tasks {
		deps := "none"
		if len(task.DependsOn) > 0 {
			deps = fmt.Sprintf("%d dependencies", len(task.DependsOn))
		}
		fmt.Printf("  %s [%s] %s - %s (%s)\n",
			task.ID, task.Priority, task.FeatureID, task.Skill, deps)
	}

	if summary := planProviderSummary(); summary != "" {
		fmt.Println()
		fmt.Println(summary)
	}

	fmt.Println("\nNext steps:")
	if featureID != "" {
		fmt.Printf("  1. Review plan: specular plan review\n")
		fmt.Printf("  2. Execute feature: specular build run --plan %s\n", out)
	} else {
		fmt.Printf("  1. Review plan: specular plan review\n")
		fmt.Printf("  2. Execute plan: specular build run\n")
	}

	// Record success with metrics
	duration := time.Since(startTime)
	telemetry.RecordSuccess(span,
		attribute.Int("tasks_count", len(p.Tasks)),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)

	return nil
}

func runPlanReview(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	planPath := cmd.Flags().Lookup("plan").Value.String()

	nonInteractive := cmd.Flags().Lookup("non-interactive").Value.String() == "true"
	autoApprove := cmd.Flags().Lookup("auto-approve").Value.String() == "true"
	format := cmd.Flags().Lookup("format").Value.String()

	if format != "text" && format != "json" {
		return fmt.Errorf("unsupported format %s (supported: text, json)", format)
	}

	if autoApprove && !nonInteractive {
		return fmt.Errorf("--auto-approve makes sense only with --non-interactive")
	}

	// Use smart default if not changed
	if !cmd.Flags().Changed("plan") {
		planPath = defaults.PlanFile()
	}

	// Validate plan file exists
	if err := ux.ValidateRequiredFile(planPath, "Plan file", "specular plan create"); err != nil {
		return ux.EnhanceError(err)
	}

	// Load plan
	p, err := plan.LoadPlan(planPath)
	if err != nil {
		return ux.FormatError(err, "loading plan file")
	}

	if nonInteractive {
		nextSteps := []string{"Execute plan: specular build run"}
		if !autoApprove {
			nextSteps = append(nextSteps, "Approve plan: specular plan review --non-interactive --auto-approve")
		}

		if err := printPlanSummary(planPath, p, format, nextSteps, autoApprove); err != nil {
			return err
		}

		if format == "text" {
			if autoApprove {
				fmt.Printf("\n✓ Plan approved\n\n")
				fmt.Println("Next steps:")
				fmt.Printf("  1. Execute plan: specular build run\n")
			} else {
				fmt.Println("\nNext steps:")
				fmt.Printf("  1. Execute plan: specular build run\n")
				fmt.Printf("  2. Approve plan: specular plan review --non-interactive --auto-approve\n")
			}
		}
		return nil
	}

	fmt.Printf("=== Plan Review (TUI) ===\n")
	fmt.Printf("Plan: %s (%d tasks)\n\n", planPath, len(p.Tasks))

	// Launch TUI for plan review
	result, err := tui.RunPlanReview(p)
	if err != nil {
		return ux.FormatError(err, "running plan review TUI")
	}

	// Show result
	if result.Approved {
		fmt.Printf("\n✓ Plan approved\n")
		fmt.Println("\nNext steps:")
		fmt.Printf("  1. Execute plan: specular build run\n")
	} else {
		fmt.Printf("\n✗ Plan rejected\n")
		if result.Reason != "" {
			fmt.Printf("  Reason: %s\n", result.Reason)
		}
		fmt.Println("\nNext steps:")
		fmt.Printf("  1. Modify spec: specular spec edit\n")
		fmt.Printf("  2. Regenerate plan: specular plan create\n")
	}

	return nil
}

type planTaskSummary struct {
	ID        string   `json:"id"`
	Feature   string   `json:"feature"`
	Skill     string   `json:"skill"`
	Priority  string   `json:"priority"`
	ModelHint string   `json:"model_hint"`
	DependsOn []string `json:"depends_on"`
}

type planSummary struct {
	Plan      string            `json:"plan"`
	TaskCount int               `json:"task_count"`
	Tasks     []planTaskSummary `json:"tasks"`
	NextSteps []string          `json:"next_steps,omitempty"`
	Approved  bool              `json:"approved"`
}

func buildPlanSummary(planPath string, p *plan.Plan) planSummary {
	tasks := make([]planTaskSummary, 0, len(p.Tasks))
	for _, task := range p.Tasks {
		deps := make([]string, 0, len(task.DependsOn))
		for _, dep := range task.DependsOn {
			deps = append(deps, string(dep))
		}
		if len(deps) == 0 {
			deps = append(deps, "none")
		}
		tasks = append(tasks, planTaskSummary{
			ID:        task.ID.String(),
			Feature:   string(task.FeatureID),
			Skill:     task.Skill,
			Priority:  string(task.Priority),
			ModelHint: task.ModelHint,
			DependsOn: deps,
		})
	}

	return planSummary{
		Plan:      planPath,
		TaskCount: len(p.Tasks),
		Tasks:     tasks,
	}
}

func printPlanSummary(planPath string, p *plan.Plan, format string, nextSteps []string, approved bool) error {
	summary := buildPlanSummary(planPath, p)
	summary.NextSteps = nextSteps
	summary.Approved = approved

	switch format {
	case "json":
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format plan summary as JSON: %w", err)
		}
		fmt.Println(string(data))
	default: // text
		printPlanSummaryText(summary)
	}

	return nil
}

func printPlanSummaryText(summary planSummary) {
	fmt.Printf("=== Plan Summary ===\n\n")
	fmt.Printf("Plan: %s (%d tasks)\n\n", summary.Plan, summary.TaskCount)
	fmt.Println("Tasks:")
	for _, task := range summary.Tasks {
		deps := strings.Join(task.DependsOn, ", ")
		fmt.Printf("  • %s: feature=%s, skill=%s, priority=%s, model_hint=%s, deps=%s\n",
			task.ID, task.Feature, task.Skill, task.Priority, task.ModelHint, deps)
	}
}

func runPlanExplain(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	planPath := cmd.Flags().Lookup("plan").Value.String()

	// Use smart default if not changed
	if !cmd.Flags().Changed("plan") {
		planPath = defaults.PlanFile()
	}

	// Validate plan file exists
	if err := ux.ValidateRequiredFile(planPath, "Plan file", "specular plan create"); err != nil {
		return ux.EnhanceError(err)
	}

	// Require step argument
	if len(args) == 0 {
		return fmt.Errorf("step ID is required\n\nUsage: specular plan explain <step-id>")
	}
	stepID := args[0]

	// Load plan
	p, err := plan.LoadPlan(planPath)
	if err != nil {
		return ux.FormatError(err, "loading plan file")
	}

	// Find the task
	var task *plan.Task
	for i := range p.Tasks {
		if string(p.Tasks[i].ID) == stepID {
			task = &p.Tasks[i]
			break
		}
	}

	if task == nil {
		return fmt.Errorf("task '%s' not found in plan", stepID)
	}

	// Explain the routing decision
	fmt.Printf("=== Plan Step Explanation ===\n\n")
	fmt.Printf("Task ID: %s\n", task.ID)
	fmt.Printf("Feature: %s\n", task.FeatureID)
	fmt.Printf("Skill: %s\n", task.Skill)
	fmt.Printf("Priority: %s\n", task.Priority)
	fmt.Printf("Model Hint: %s\n", task.ModelHint)
	fmt.Printf("Estimated Complexity: %d\n\n", task.Estimate)

	fmt.Printf("Routing Decision:\n")
	fmt.Printf("  Model selected based on:\n")
	fmt.Printf("    • Skill requirement: %s\n", task.Skill)
	fmt.Printf("    • Model hint: %s\n", task.ModelHint)
	fmt.Printf("    • Task priority: %s\n", task.Priority)
	fmt.Println()

	// Show dependencies
	if len(task.DependsOn) > 0 {
		fmt.Printf("Dependencies (%d):\n", len(task.DependsOn))
		for _, depID := range task.DependsOn {
			fmt.Printf("  • %s\n", depID)
		}
		fmt.Println()
	} else {
		fmt.Printf("Dependencies: none\n\n")
	}

	fmt.Printf("Expected Hash: %s\n", task.ExpectedHash)
	fmt.Printf("  (used for drift detection)\n")

	return nil
}

func runPlanVisualize(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	planPath := cmd.Flags().Lookup("plan").Value.String()

	// Use smart default if not changed
	if !cmd.Flags().Changed("plan") {
		planPath = defaults.PlanFile()
	}

	// Validate plan file exists
	if err := ux.ValidateRequiredFile(planPath, "Plan file", "specular plan create"); err != nil {
		return ux.EnhanceError(err)
	}

	// Load plan
	p, err := plan.LoadPlan(planPath)
	if err != nil {
		return ux.FormatError(err, "loading plan file")
	}

	fmt.Printf("=== Plan Visualization ===\n\n")
	fmt.Printf("Plan: %s (%d tasks)\n\n", planPath, len(p.Tasks))

	// ASCII graph visualization
	fmt.Println("Task Dependency Graph:")
	fmt.Println()

	// Group tasks by priority
	priorityGroups := make(map[string][]plan.Task)
	for _, task := range p.Tasks {
		priorityGroups[string(task.Priority)] = append(priorityGroups[string(task.Priority)], task)
	}

	// Display by priority level
	for _, priority := range []string{"P0", "P1", "P2", "P3"} {
		tasks := priorityGroups[priority]
		if len(tasks) == 0 {
			continue
		}

		fmt.Printf("[%s] Priority Tasks:\n", priority)
		for _, task := range tasks {
			deps := "none"
			if len(task.DependsOn) > 0 {
				// Convert []TaskID to []string
				depsStrs := make([]string, len(task.DependsOn))
				for i, depID := range task.DependsOn {
					depsStrs[i] = string(depID)
				}
				deps = strings.Join(depsStrs, ", ")
			}
			fmt.Printf("  • %s (%s) - depends on: %s\n",
				task.ID, task.Skill, deps)
		}
		fmt.Println()
	}

	fmt.Println("Next steps:")
	fmt.Printf("  1. Validate plan: specular plan validate\n")
	fmt.Printf("  2. Review plan: specular plan review\n")
	fmt.Printf("  3. Execute plan: specular build run\n")

	return nil
}

func runPlanValidate(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	planPath := cmd.Flags().Lookup("plan").Value.String()

	// Use smart default if not changed
	if !cmd.Flags().Changed("plan") {
		planPath = defaults.PlanFile()
	}

	// Validate plan file exists
	if err := ux.ValidateRequiredFile(planPath, "Plan file", "specular plan create"); err != nil {
		return ux.EnhanceError(err)
	}

	// Load plan
	p, err := plan.LoadPlan(planPath)
	if err != nil {
		return ux.FormatError(err, "loading plan file")
	}

	fmt.Printf("Validating plan: %s\n\n", planPath)

	validationErrors := 0

	// Check 1: Circular dependencies
	fmt.Printf("Checking for circular dependencies... ")
	if hasCircularDeps := checkCircularDependencies(p); hasCircularDeps {
		fmt.Printf("❌ FAILED\n")
		validationErrors++
	} else {
		fmt.Printf("✓ OK\n")
	}

	// Check 2: Missing dependencies
	fmt.Printf("Checking for missing dependencies... ")
	taskIDs := make(map[string]bool)
	for _, task := range p.Tasks {
		taskIDs[string(task.ID)] = true
	}

	missingDeps := false
	for _, task := range p.Tasks {
		for _, depID := range task.DependsOn {
			if !taskIDs[string(depID)] {
				if !missingDeps {
					fmt.Printf("❌ FAILED\n")
					missingDeps = true
					validationErrors++
				}
				fmt.Printf("  Task %s depends on missing task: %s\n", task.ID, depID)
			}
		}
	}
	if !missingDeps {
		fmt.Printf("✓ OK\n")
	}

	// Check 3: Valid priorities
	fmt.Printf("Checking task priorities... ")
	invalidPriorities := false
	for _, task := range p.Tasks {
		priority := string(task.Priority)
		if priority != "P0" && priority != "P1" && priority != "P2" && priority != "P3" {
			if !invalidPriorities {
				fmt.Printf("❌ FAILED\n")
				invalidPriorities = true
				validationErrors++
			}
			fmt.Printf("  Task %s has invalid priority: %s\n", task.ID, priority)
		}
	}
	if !invalidPriorities {
		fmt.Printf("✓ OK\n")
	}

	// Summary
	fmt.Println()
	if validationErrors == 0 {
		fmt.Printf("✓ Plan is valid (%d tasks, %d checks passed)\n", len(p.Tasks), 3)
		fmt.Println("\nNext steps:")
		fmt.Printf("  1. Review plan: specular plan review\n")
		fmt.Printf("  2. Execute plan: specular build run\n")
	} else {
		fmt.Printf("❌ Plan has %d validation error(s)\n", validationErrors)
		fmt.Println("\nRecommendations:")
		fmt.Printf("  1. Fix validation errors\n")
		fmt.Printf("  2. Regenerate plan: specular plan create\n")
		return fmt.Errorf("plan validation failed with %d error(s)", validationErrors)
	}

	return nil
}

// checkCircularDependencies checks if there are circular dependencies in the plan
func checkCircularDependencies(p *plan.Plan) bool {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, task := range p.Tasks {
		// Convert []TaskID to []string
		deps := make([]string, len(task.DependsOn))
		for i, depID := range task.DependsOn {
			deps[i] = string(depID)
		}
		graph[string(task.ID)] = deps
	}

	// Track visited and recursion stack
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	// DFS to detect cycle
	var hasCycle func(string) bool
	hasCycle = func(taskID string) bool {
		visited[taskID] = true
		recStack[taskID] = true

		for _, dep := range graph[taskID] {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[taskID] = false
		return false
	}

	// Check all tasks
	for _, task := range p.Tasks {
		taskID := string(task.ID)
		if !visited[taskID] {
			if hasCycle(taskID) {
				return true
			}
		}
	}

	return false
}

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.AddCommand(planCreateCmd)
	planCmd.AddCommand(planReviewCmd)
	planCmd.AddCommand(planExplainCmd)
	planCmd.AddCommand(planVisualizeCmd)
	planCmd.AddCommand(planValidateCmd)

	// plan create flags
	planCreateCmd.Flags().StringP("in", "i", ".specular/spec.yaml", "Input spec file")
	planCreateCmd.Flags().String("lock", ".specular/spec.lock.json", "Input SpecLock file")
	planCreateCmd.Flags().StringP("out", "o", "plan.json", "Output plan file")
	planCreateCmd.Flags().Bool("estimate", true, "Estimate task complexity")
	planCreateCmd.Flags().String("feature", "", "Generate plan for specific feature ID")
	planCreateCmd.Flags().Bool("tui", false, "Run with interactive TUI mode")

	// plan review flags
	planReviewCmd.Flags().String("plan", "plan.json", "Plan file to review")
	planReviewCmd.Flags().Bool("non-interactive", false, "Summarize plan without launching TUI")
	planReviewCmd.Flags().Bool("auto-approve", false, "Automatically approve plan (requires --non-interactive)")
	planReviewCmd.Flags().String("format", "text", "Output format for non-interactive summary (text,json)")

	// plan explain flags
	planExplainCmd.Flags().String("plan", "plan.json", "Plan file to explain")

	// plan visualize flags
	planVisualizeCmd.Flags().String("plan", "plan.json", "Plan file to visualize")

	// plan validate flags
	planValidateCmd.Flags().String("plan", "plan.json", "Plan file to validate")
}

func planProviderSummary() string {
	defaults := ux.NewPathDefaults()
	config, err := provider.LoadProvidersConfig(defaults.ProvidersFile())
	if err != nil {
		return ""
	}

	var enabled []string
	for _, p := range config.Providers {
		if !p.Enabled {
			continue
		}
		if desc := provider.DescriptorByName(p.Name); desc != nil {
			enabled = append(enabled, fmt.Sprintf("%s (trust: %s)", providerDisplayName(p.Name), desc.TrustLevel))
			continue
		}
		enabled = append(enabled, providerDisplayName(p.Name))
	}

	if len(enabled) == 0 {
		return ""
	}

	return fmt.Sprintf("Providers in use: %s", strings.Join(enabled, ", "))
}
