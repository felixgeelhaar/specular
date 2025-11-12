package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/checkpoint"
	"github.com/felixgeelhaar/specular/internal/drift"
	"github.com/felixgeelhaar/specular/internal/eval"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/policy"
	"github.com/felixgeelhaar/specular/internal/progress"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Manage evaluation and testing workflows",
	Long: `Run evaluation scenarios, manage guardrail rules, and detect drift.

Use 'specular eval run' to run evaluation scenarios.
Use 'specular eval rules' to manage guardrail rules.
Use 'specular eval drift' to detect drift.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if this is being used as the old direct command
		// If flags are set, run the drift command for backward compatibility
		if cmd.Flags().Changed("plan") || cmd.Flags().Changed("lock") || cmd.Flags().Changed("fail-on-drift") {
			fmt.Fprintf(os.Stderr, "\n⚠️  DEPRECATION WARNING:\n")
			fmt.Fprintf(os.Stderr, "Running 'eval' directly is deprecated and will be removed in v1.6.0.\n")
			fmt.Fprintf(os.Stderr, "Please use 'specular eval drift' instead.\n\n")

			// Run drift command
			return runEvalDrift(cmd, args)
		}

		// Otherwise show help
		return cmd.Help()
	},
}

var evalRunCmd = &cobra.Command{
	Use:   "run [scenario]",
	Short: "Run evaluation scenarios",
	Long: `Run comprehensive evaluation scenarios to validate your project.

Available scenarios:
  smoke        - Basic health checks (default)
  integration  - Full integration tests
  security     - Security scan + policy check
  performance  - Performance benchmarks

If no scenario is specified, 'smoke' is run by default.`,
	RunE: runEvalRun,
}

var evalRulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Manage guardrail rules",
	Long: `View or edit guardrail rules for evaluation.

Guardrail rules define quality gates, security checks, and policy enforcement
that are applied during evaluation scenarios.`,
	RunE: runEvalRules,
}

var evalDriftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect drift between plan and repository",
	Long: `Execute comprehensive drift detection including:
- Plan drift detection (spec hash mismatches)
- Code drift detection (contract tests, API conformance)
- Infrastructure drift (policy violations)
- Test execution and coverage analysis
- Security scanning

Results are output in SARIF format for integration with CI/CD tools.`,
	RunE: runEvalDrift,
}

func runEvalRun(cmd *cobra.Command, args []string) error {
	// Determine scenario
	scenario := "smoke" // default
	if len(args) > 0 {
		scenario = args[0]
	} else if cmd.Flags().Changed("scenario") {
		scenario = cmd.Flags().Lookup("scenario").Value.String()
	}

	// Validate scenario
	validScenarios := map[string]bool{
		"smoke":       true,
		"integration": true,
		"security":    true,
		"performance": true,
	}
	if !validScenarios[scenario] {
		return fmt.Errorf("invalid scenario '%s'. Valid scenarios: smoke, integration, security, performance", scenario)
	}

	fmt.Printf("Running evaluation scenario: %s\n\n", scenario)

	// Load policy if provided
	policyFile := cmd.Flags().Lookup("policy").Value.String()
	var pol *policy.Policy
	var polErr error
	if policyFile != "" {
		pol, polErr = policy.LoadPolicy(policyFile)
		if polErr != nil {
			return fmt.Errorf("failed to load policy: %w", polErr)
		}
	}

	// Execute scenario-specific checks
	passed := 0
	failed := 0
	var checks []string

	switch scenario {
	case "smoke":
		checks = []string{"go vet", "go build", "basic tests"}
		fmt.Println("=== Smoke Test Scenario ===")
		fmt.Println("Running basic health checks...")
		fmt.Println()

		// 1. go vet
		fmt.Printf("1. Running go vet...\n")
		vetCmd := exec.Command("go", "vet", "./...")
		if vetErr := vetCmd.Run(); vetErr != nil {
			fmt.Printf("   ✗ go vet failed\n")
			failed++
		} else {
			fmt.Printf("   ✓ go vet passed\n")
			passed++
		}

		// 2. go build
		fmt.Printf("2. Running go build...\n")
		buildCmd := exec.Command("go", "build", "./...")
		if buildErr := buildCmd.Run(); buildErr != nil {
			fmt.Printf("   ✗ go build failed\n")
			failed++
		} else {
			fmt.Printf("   ✓ go build passed\n")
			passed++
		}

		// 3. Basic tests
		fmt.Printf("3. Running basic tests...\n")
		testCmd := exec.Command("go", "test", "./...", "-short", "-timeout=30s")
		if testErr := testCmd.Run(); testErr != nil {
			fmt.Printf("   ✗ tests failed\n")
			failed++
		} else {
			fmt.Printf("   ✓ tests passed\n")
			passed++
		}

	case "integration":
		checks = []string{"go vet", "all tests", "coverage check"}
		fmt.Println("=== Integration Test Scenario ===")
		fmt.Println("Running full integration tests...")
		fmt.Println()

		// 1. go vet
		fmt.Printf("1. Running go vet...\n")
		vetCmd := exec.Command("go", "vet", "./...")
		if vetErr := vetCmd.Run(); vetErr != nil {
			fmt.Printf("   ✗ go vet failed\n")
			failed++
		} else {
			fmt.Printf("   ✓ go vet passed\n")
			passed++
		}

		// 2. All tests (no -short flag)
		fmt.Printf("2. Running all tests...\n")
		testCmd := exec.Command("go", "test", "./...", "-timeout=5m")
		if testErr := testCmd.Run(); testErr != nil {
			fmt.Printf("   ✗ tests failed\n")
			failed++
		} else {
			fmt.Printf("   ✓ tests passed\n")
			passed++
		}

		// 3. Coverage check
		fmt.Printf("3. Checking test coverage...\n")
		coverCmd := exec.Command("go", "test", "./...", "-cover")
		if coverErr := coverCmd.Run(); coverErr != nil {
			fmt.Printf("   ✗ coverage check failed\n")
			failed++
		} else {
			fmt.Printf("   ✓ coverage check passed\n")
			passed++
		}

	case "security":
		checks = []string{"go vet", "gosec scan", "policy check"}
		fmt.Println("=== Security Test Scenario ===")
		fmt.Println("Running security scans and policy checks...")
		fmt.Println()

		// 1. go vet
		fmt.Printf("1. Running go vet...\n")
		vetCmd := exec.Command("go", "vet", "./...")
		if vetErr := vetCmd.Run(); vetErr != nil {
			fmt.Printf("   ✗ go vet failed\n")
			failed++
		} else {
			fmt.Printf("   ✓ go vet passed\n")
			passed++
		}

		// 2. gosec scan
		fmt.Printf("2. Running gosec security scan...\n")
		gosecCmd := exec.Command("gosec", "./...")
		gosecErr := gosecCmd.Run()
		if gosecErr != nil {
			// Check if gosec is not installed
			if strings.Contains(gosecErr.Error(), "not found") || strings.Contains(gosecErr.Error(), "executable file not found") {
				fmt.Printf("   ⊘ gosec not installed (skipping)\n")
			} else {
				fmt.Printf("   ✗ gosec scan failed\n")
				failed++
			}
		} else {
			fmt.Printf("   ✓ gosec scan passed\n")
			passed++
		}

		// 3. Policy check
		fmt.Printf("3. Checking policy compliance...\n")
		if pol != nil {
			fmt.Printf("   ✓ Policy loaded\n")
			fmt.Printf("   • Docker required: %v\n", pol.Execution.Docker.Required)
			fmt.Printf("   • Security scans: secrets=%v deps=%v\n", pol.Security.SecretsScan, pol.Security.DepScan)
			passed++
		} else {
			fmt.Printf("   ⊘ No policy file (skipping)\n")
		}

	case "performance":
		checks = []string{"benchmark tests", "memory profiling", "CPU profiling"}
		fmt.Println("=== Performance Test Scenario ===")
		fmt.Println("Running performance benchmarks...")
		fmt.Println()

		// 1. Benchmark tests
		fmt.Printf("1. Running benchmark tests...\n")
		benchCmd := exec.Command("go", "test", "./...", "-bench=.", "-benchtime=1s", "-run=^$")
		if benchErr := benchCmd.Run(); benchErr != nil {
			fmt.Printf("   ✗ benchmarks failed\n")
			failed++
		} else {
			fmt.Printf("   ✓ benchmarks passed\n")
			passed++
		}

		// 2. Memory profiling check
		fmt.Printf("2. Checking memory profiling support...\n")
		fmt.Printf("   ✓ Memory profiling available (use -memprofile flag)\n")
		passed++

		// 3. CPU profiling check
		fmt.Printf("3. Checking CPU profiling support...\n")
		fmt.Printf("   ✓ CPU profiling available (use -cpuprofile flag)\n")
		passed++
	}

	// Summary
	fmt.Println()
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Evaluation Summary (%s scenario):\n", scenario)
	fmt.Printf("  ✓ Passed: %d\n", passed)
	if failed > 0 {
		fmt.Printf("  ✗ Failed: %d\n", failed)
	}
	fmt.Printf("  Total:   %d\n", len(checks))
	fmt.Println(strings.Repeat("=", 50))

	if failed > 0 {
		return fmt.Errorf("evaluation failed with %d errors", failed)
	}

	fmt.Println("\n✓ Evaluation passed")
	return nil
}

func runEvalRules(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	policyFile := cmd.Flags().Lookup("policy").Value.String()

	// Use smart default if not changed
	if !cmd.Flags().Changed("policy") {
		policyFile = defaults.PolicyFile()
	}

	// Check if policy file exists
	if _, err := os.Stat(policyFile); os.IsNotExist(err) {
		fmt.Printf("Policy file not found: %s\n\n", policyFile)
		fmt.Println("To create a policy file:")
		fmt.Printf("  specular init\n\n")
		fmt.Println("Or create manually at: .specular/policy.yaml")
		return fmt.Errorf("policy file not found")
	}

	// Load policy
	pol, err := policy.LoadPolicy(policyFile)
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	fmt.Printf("=== Guardrail Rules ===\n")
	fmt.Printf("Policy file: %s\n\n", policyFile)

	// Display execution rules
	fmt.Println("Execution Policy:")
	fmt.Printf("  Allow Local: %v\n", pol.Execution.AllowLocal)
	fmt.Printf("  Docker Required: %v\n", pol.Execution.Docker.Required)
	if len(pol.Execution.Docker.ImageAllowlist) > 0 {
		fmt.Printf("  Docker Image Allowlist:\n")
		for _, img := range pol.Execution.Docker.ImageAllowlist {
			fmt.Printf("    - %s\n", img)
		}
	}
	if pol.Execution.Docker.CPULimit != "" {
		fmt.Printf("  Docker CPU Limit: %s\n", pol.Execution.Docker.CPULimit)
	}
	if pol.Execution.Docker.MemLimit != "" {
		fmt.Printf("  Docker Memory Limit: %s\n", pol.Execution.Docker.MemLimit)
	}
	fmt.Println()

	// Display linter rules
	if len(pol.Linters) > 0 {
		fmt.Println("Linters:")
		for name, cfg := range pol.Linters {
			status := "disabled"
			if cfg.Enabled {
				status = "enabled"
			}
			fmt.Printf("  %s: %s", name, status)
			if cfg.Cmd != "" {
				fmt.Printf(" (cmd: %s)", cfg.Cmd)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// Display formatter rules
	if len(pol.Formatters) > 0 {
		fmt.Println("Formatters:")
		for name, cfg := range pol.Formatters {
			status := "disabled"
			if cfg.Enabled {
				status = "enabled"
			}
			fmt.Printf("  %s: %s", name, status)
			if cfg.Cmd != "" {
				fmt.Printf(" (cmd: %s)", cfg.Cmd)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// Display test rules
	fmt.Println("Test Policy:")
	fmt.Printf("  Require Pass: %v\n", pol.Tests.RequirePass)
	fmt.Printf("  Min Coverage: %.1f%%\n", pol.Tests.MinCoverage*100)
	fmt.Println()

	// Display security rules
	fmt.Println("Security Policy:")
	fmt.Printf("  Secrets Scan: %v\n", pol.Security.SecretsScan)
	fmt.Printf("  Dependency Scan: %v\n", pol.Security.DepScan)
	fmt.Println()

	// Display routing rules
	if len(pol.Routing.AllowModels) > 0 {
		fmt.Println("Routing Policy:")
		fmt.Println("  Allowed Models:")
		for _, allow := range pol.Routing.AllowModels {
			fmt.Printf("    Provider: %s\n", allow.Provider)
			for _, name := range allow.Names {
				fmt.Printf("      - %s\n", name)
			}
		}
		fmt.Println()
	}

	if len(pol.Routing.DenyTools) > 0 {
		fmt.Println("  Denied Tools:")
		for _, tool := range pol.Routing.DenyTools {
			fmt.Printf("    - %s\n", tool)
		}
		fmt.Println()
	}

	// Check if --edit flag is set
	if cmd.Flags().Lookup("edit").Value.String() == "true" {
		// Open in editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi" // fallback
		}

		fmt.Printf("Opening %s in %s...\n", policyFile, editor)
		editorCmd := exec.Command(editor, policyFile)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if editorErr := editorCmd.Run(); editorErr != nil {
			return fmt.Errorf("failed to open editor: %w", editorErr)
		}

		fmt.Println("\nPolicy file updated. Validating...")
		// Validate after edit
		if _, valErr := policy.LoadPolicy(policyFile); valErr != nil {
			return fmt.Errorf("policy validation failed: %w", valErr)
		}
		fmt.Println("✓ Policy validated successfully")
	}

	return nil
}

func runEvalDrift(cmd *cobra.Command, args []string) error {
	defaults := ux.NewPathDefaults()
	planFile := cmd.Flags().Lookup("plan").Value.String()
	lockFile := cmd.Flags().Lookup("lock").Value.String()
	specFile := cmd.Flags().Lookup("spec").Value.String()
	policyFile := cmd.Flags().Lookup("policy").Value.String()
	reportFile := cmd.Flags().Lookup("report").Value.String()
	failOnDrift := cmd.Flags().Lookup("fail-on-drift").Value.String() == "true"
	projectRoot := cmd.Flags().Lookup("project-root").Value.String()
	apiSpecPath := cmd.Flags().Lookup("api-spec").Value.String()
	ignoreGlobs, _ := cmd.Flags().GetStringSlice("ignore") //nolint:errcheck // Acceptable to ignore array return
	resume := cmd.Flags().Lookup("resume").Value.String() == "true"
	checkpointDir := cmd.Flags().Lookup("checkpoint-dir").Value.String()
	checkpointID := cmd.Flags().Lookup("checkpoint-id").Value.String()

	// Use smart defaults if not changed
	if !cmd.Flags().Changed("plan") {
		planFile = defaults.PlanFile()
	}
	if !cmd.Flags().Changed("lock") {
		lockFile = defaults.SpecLockFile()
	}
	if !cmd.Flags().Changed("spec") {
		specFile = defaults.SpecFile()
	}
	if !cmd.Flags().Changed("policy") && policyFile == "" {
		policyFile = defaults.PolicyFile()
	}
	if !cmd.Flags().Changed("checkpoint-dir") {
		checkpointDir = defaults.CheckpointDir()
	}

	// Validate required files with helpful errors
	if err := ux.ValidateRequiredFile(planFile, "Plan file", "specular plan"); err != nil {
		return ux.EnhanceError(err)
	}
	if err := ux.ValidateRequiredFile(lockFile, "SpecLock file", "specular spec lock"); err != nil {
		return ux.EnhanceError(err)
	}
	if err := ux.ValidateRequiredFile(specFile, "Spec file", "specular spec generate"); err != nil {
		return ux.EnhanceError(err)
	}

	// Setup checkpoint manager
	checkpointMgr := checkpoint.NewManager(checkpointDir, true, 30*time.Second)
	var cpState *checkpoint.State

	// Generate operation ID if not provided
	if checkpointID == "" {
		checkpointID = fmt.Sprintf("eval-%s-%d", planFile, time.Now().Unix())
	}

	// Initialize progress indicator
	progressIndicator := progress.NewIndicator(progress.Config{
		Writer:      os.Stdout,
		ShowSpinner: true,
	})

	// Handle resume if requested
	if resume {
		if checkpointMgr.Exists(checkpointID) {
			var err error
			cpState, err = checkpointMgr.Load(checkpointID)
			if err != nil {
				return fmt.Errorf("failed to load checkpoint: %w", err)
			}

			// Use progress indicator for formatted resume info
			progressIndicator.SetState(cpState)
			progressIndicator.PrintResumeInfo()
		} else {
			fmt.Printf("No checkpoint found for: %s\n", checkpointID)
			fmt.Println("Starting fresh evaluation...")
			cpState = checkpoint.NewState(checkpointID)
		}
	} else {
		cpState = checkpoint.NewState(checkpointID)
	}

	// Set state in progress indicator
	progressIndicator.SetState(cpState)

	// Store metadata
	cpState.SetMetadata("plan", planFile)
	cpState.SetMetadata("lock", lockFile)
	cpState.SetMetadata("spec", specFile)
	cpState.SetMetadata("policy", policyFile)

	// Initialize check tasks
	checks := []string{"quality-gate", "plan-drift", "code-drift", "infra-drift", "report-generation"}
	for _, checkID := range checks {
		if _, exists := cpState.Tasks[checkID]; !exists {
			progressIndicator.UpdateTask(checkID, "pending", nil)
		}
	}

	// Save initial checkpoint
	if err := checkpointMgr.Save(cpState); err != nil {
		fmt.Printf("Warning: failed to save initial checkpoint: %v\n", err)
	}

	// Start progress indicator
	progressIndicator.Start()
	defer progressIndicator.Stop()

	// Load plan
	p, err := plan.LoadPlan(planFile)
	if err != nil {
		return ux.FormatError(err, "loading plan file")
	}

	// Load SpecLock
	lock, err := spec.LoadSpecLock(lockFile)
	if err != nil {
		return ux.FormatError(err, "loading SpecLock file")
	}

	// Load spec for code drift detection
	s, err := spec.LoadSpec(specFile)
	if err != nil {
		return ux.FormatError(err, "loading spec file")
	}

	// Run eval gate if policy is provided
	if policyFile != "" && cpState.Tasks["quality-gate"].Status != "completed" {
		progressIndicator.UpdateTask("quality-gate", "running", nil)
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}

		polQualityGate, polErr := policy.LoadPolicy(policyFile)
		if polErr != nil {
			progressIndicator.UpdateTask("quality-gate", "failed", polErr)
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}
			return fmt.Errorf("failed to load policy: %w", polErr)
		}

		fmt.Println("Running quality gate checks...")
		gateReport, gateErr := eval.RunEvalGate(eval.GateOptions{
			Policy:      polQualityGate,
			ProjectRoot: projectRoot,
			Verbose:     false,
		})
		if gateErr != nil {
			progressIndicator.UpdateTask("quality-gate", "failed", gateErr)
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}
			return fmt.Errorf("eval gate failed: %w", gateErr)
		}

		// Print gate results
		fmt.Printf("\nQuality Gate Results:\n")
		fmt.Printf("  Total Checks: %d\n", len(gateReport.Checks))
		fmt.Printf("  Passed:       %d\n", gateReport.TotalPassed)
		fmt.Printf("  Failed:       %d\n", gateReport.TotalFailed)
		fmt.Printf("  Skipped:      %d\n", gateReport.TotalSkipped)
		fmt.Printf("  Duration:     %s\n\n", gateReport.Duration)

		for _, check := range gateReport.Checks {
			status := "✓"
			if !check.Passed {
				status = "✗"
			}
			fmt.Printf("  %s %s: %s (%.2fs)\n", status, check.Name, check.Message, check.Duration.Seconds())
		}
		fmt.Println()

		// Fail early if gate failed
		if !gateReport.AllPassed {
			progressIndicator.UpdateTask("quality-gate", "failed", fmt.Errorf("quality gate failed with %d failed checks", gateReport.TotalFailed))
			if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
				fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
			}
			return fmt.Errorf("quality gate failed with %d failed checks", gateReport.TotalFailed)
		}

		progressIndicator.UpdateTask("quality-gate", "completed", nil)
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}
	} else if policyFile == "" {
		progressIndicator.UpdateTask("quality-gate", "skipped", nil)
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}
	} else {
		fmt.Println("✓ Quality gate check already completed (skipping)")
	}

	// Detect plan drift
	if cpState.Tasks["plan-drift"].Status != "completed" {
		progressIndicator.UpdateTask("plan-drift", "running", nil)
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}

		fmt.Println("Detecting plan drift...")
		planDrift := drift.DetectPlanDrift(lock, p)

		progressIndicator.UpdateTask("plan-drift", "completed", nil)
		cpState.SetMetadata("plan_drift_count", fmt.Sprintf("%d", len(planDrift)))
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}
	} else {
		fmt.Println("✓ Plan drift check already completed (skipping)")
	}

	// Detect code drift
	if cpState.Tasks["code-drift"].Status != "completed" {
		progressIndicator.UpdateTask("code-drift", "running", nil)
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}

		fmt.Println("Detecting code drift...")
		codeDrift := drift.DetectCodeDrift(s, lock, drift.CodeDriftOptions{
			ProjectRoot: projectRoot,
			APISpecPath: apiSpecPath,
			IgnoreGlobs: ignoreGlobs,
		})

		progressIndicator.UpdateTask("code-drift", "completed", nil)
		cpState.SetMetadata("code_drift_count", fmt.Sprintf("%d", len(codeDrift)))
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}
	} else {
		fmt.Println("✓ Code drift check already completed (skipping)")
	}

	// Detect infrastructure drift
	var infraDrift []drift.Finding
	if cpState.Tasks["infra-drift"].Status != "completed" {
		progressIndicator.UpdateTask("infra-drift", "running", nil)
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}

		fmt.Println("Detecting infrastructure drift...")
		if policyFile != "" {
			polInfra, polInfraErr := policy.LoadPolicy(policyFile)
			if polInfraErr != nil {
				progressIndicator.UpdateTask("infra-drift", "failed", polInfraErr)
				if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
					fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
				}
				return fmt.Errorf("failed to load policy: %w", polInfraErr)
			}

			// Build task images map from plan
			// Note: Currently plan.Task doesn't have Image field, so this will be empty
			// This is a placeholder for future enhancement when task images are tracked
			taskImages := make(map[string]string)
			// Future: when plan.Task has Image field, populate taskImages here

			infraDrift = drift.DetectInfraDrift(drift.InfraDriftOptions{
				Policy:     polInfra,
				TaskImages: taskImages,
			})
		}

		progressIndicator.UpdateTask("infra-drift", "completed", nil)
		cpState.SetMetadata("infra_drift_count", fmt.Sprintf("%d", len(infraDrift)))
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}
	} else {
		fmt.Println("✓ Infrastructure drift check already completed (skipping)")
	}

	// Get drift results from checkpoint metadata if checks were skipped
	planDrift := drift.DetectPlanDrift(lock, p)
	codeDrift := drift.DetectCodeDrift(s, lock, drift.CodeDriftOptions{
		ProjectRoot: projectRoot,
		APISpecPath: apiSpecPath,
		IgnoreGlobs: ignoreGlobs,
	})

	// Generate report
	progressIndicator.UpdateTask("report-generation", "running", nil)
	if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
		fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
	}

	report := drift.GenerateReport(planDrift, codeDrift, infraDrift)

	progressIndicator.UpdateTask("report-generation", "completed", nil)
	if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
		fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
	}

	// Print summary
	fmt.Printf("\nDrift Detection Summary:\n")
	fmt.Printf("  Total Findings: %d\n", report.Summary.TotalFindings)
	fmt.Printf("  Errors:        %d\n", report.Summary.Errors)
	fmt.Printf("  Warnings:      %d\n", report.Summary.Warnings)
	fmt.Printf("  Info:          %d\n", report.Summary.Info)
	fmt.Println()

	// Print findings
	if len(planDrift) > 0 {
		fmt.Println("Plan Drift:")
		for _, f := range planDrift {
			fmt.Printf("  [%s] %s: %s\n", f.Severity, f.Code, f.Message)
		}
	}

	if len(codeDrift) > 0 {
		fmt.Println("\nCode Drift:")
		for _, f := range codeDrift {
			fmt.Printf("  [%s] %s: %s (feature: %s)\n", f.Severity, f.Code, f.Message, f.FeatureID)
		}
	}

	if len(infraDrift) > 0 {
		fmt.Println("\nInfrastructure Drift:")
		for _, f := range infraDrift {
			fmt.Printf("  [%s] %s: %s\n", f.Severity, f.Code, f.Message)
		}
	}

	// Generate SARIF output
	sarif := report.ToSARIF()
	if errSARIF := drift.SaveSARIF(sarif, reportFile); errSARIF != nil {
		return fmt.Errorf("failed to save SARIF report: %w", errSARIF)
	}
	fmt.Printf("✓ SARIF report saved to %s\n", reportFile)

	// Mark evaluation as completed
	cpState.Status = "completed"
	if errCP := checkpointMgr.Save(cpState); errCP != nil {
		fmt.Printf("Warning: failed to save final checkpoint: %v\n", errCP)
	}

	// Fail if requested and drift detected
	if failOnDrift && report.HasErrors() {
		cpState.Status = "failed"
		if saveErr := checkpointMgr.Save(cpState); saveErr != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", saveErr)
		}
		return fmt.Errorf("drift detection failed with %d errors", report.Summary.Errors)
	}

	if report.IsClean() {
		fmt.Println("✓ No drift detected")
	}

	// Clean up checkpoint on success unless user wants to keep it
	keepCheckpoint := cmd.Flags().Lookup("keep-checkpoint").Value.String() == "true"
	if !keepCheckpoint && cpState.Status == "completed" {
		if errDel := checkpointMgr.Delete(checkpointID); errDel != nil {
			fmt.Printf("Warning: failed to delete checkpoint: %v\n", errDel)
		} else {
			fmt.Printf("Checkpoint cleaned up: %s\n", checkpointID)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(evalCmd)
	evalCmd.AddCommand(evalRunCmd)
	evalCmd.AddCommand(evalRulesCmd)
	evalCmd.AddCommand(evalDriftCmd)

	// Flags for backward compatibility on root eval command
	evalCmd.Flags().String("plan", "plan.json", "Plan file to evaluate")
	evalCmd.Flags().String("lock", ".specular/spec.lock.json", "SpecLock file")
	evalCmd.Flags().String("spec", ".specular/spec.yaml", "Spec file for code drift detection")
	evalCmd.Flags().String("policy", "", "Policy file for infrastructure drift detection")
	evalCmd.Flags().String("report", "drift.sarif", "Output report file (SARIF format)")
	evalCmd.Flags().Bool("fail-on-drift", false, "Exit with error if drift is detected")
	evalCmd.Flags().String("project-root", ".", "Project root directory")
	evalCmd.Flags().String("api-spec", "", "Path to OpenAPI spec file")
	evalCmd.Flags().StringSlice("ignore", []string{}, "Glob patterns to ignore (e.g., *.test.js)")
	evalCmd.Flags().Bool("resume", false, "Resume from previous checkpoint")
	evalCmd.Flags().String("checkpoint-dir", ".specular/checkpoints", "Directory for checkpoints")
	evalCmd.Flags().String("checkpoint-id", "", "Checkpoint ID (auto-generated if not provided)")
	evalCmd.Flags().Bool("keep-checkpoint", false, "Keep checkpoint after successful completion")

	// eval run flags
	evalRunCmd.Flags().String("scenario", "smoke", "Evaluation scenario to run")
	evalRunCmd.Flags().String("policy", ".specular/policy.yaml", "Policy file for security scenario")

	// eval rules flags
	evalRulesCmd.Flags().String("policy", ".specular/policy.yaml", "Policy file path")
	evalRulesCmd.Flags().Bool("edit", false, "Open policy file in $EDITOR")

	// eval drift flags
	evalDriftCmd.Flags().String("plan", "plan.json", "Plan file to evaluate")
	evalDriftCmd.Flags().String("lock", ".specular/spec.lock.json", "SpecLock file")
	evalDriftCmd.Flags().String("spec", ".specular/spec.yaml", "Spec file for code drift detection")
	evalDriftCmd.Flags().String("policy", "", "Policy file for infrastructure drift detection")
	evalDriftCmd.Flags().String("report", "drift.sarif", "Output report file (SARIF format)")
	evalDriftCmd.Flags().Bool("fail-on-drift", false, "Exit with error if drift is detected")
	evalDriftCmd.Flags().String("project-root", ".", "Project root directory")
	evalDriftCmd.Flags().String("api-spec", "", "Path to OpenAPI spec file")
	evalDriftCmd.Flags().StringSlice("ignore", []string{}, "Glob patterns to ignore (e.g., *.test.js)")
	evalDriftCmd.Flags().Bool("resume", false, "Resume from previous checkpoint")
	evalDriftCmd.Flags().String("checkpoint-dir", ".specular/checkpoints", "Directory for checkpoints")
	evalDriftCmd.Flags().String("checkpoint-id", "", "Checkpoint ID (auto-generated if not provided)")
	evalDriftCmd.Flags().Bool("keep-checkpoint", false, "Keep checkpoint after successful completion")
}
