# Agent Mode Implementation Roadmap

**Created:** 2025-01-10
**Status:** Planning
**ADR:** See [ADR 0007](adr/0007-autonomous-agent-mode.md)

## Overview

This document provides a detailed, phased implementation plan for transforming Specular from a manual CLI tool into an autonomous agent system while preserving manual control capabilities.

**Goal:** Enable users to run `specular auto --goal "Build a REST API"` and get working, tested code with minimal intervention.

---

## Prerequisites

### Existing Infrastructure (✅ Complete)
- ✅ AI Provider System (`internal/provider`)
- ✅ Model Router (`internal/router`)
- ✅ Workflow Orchestration (`internal/workflow`)
- ✅ Docker Sandbox Execution (`internal/exec`)
- ✅ Policy Enforcement (`internal/policy`)
- ✅ Interactive TUI (`internal/tui`)
- ✅ Structured Error System (`internal/errors`)
- ✅ Checkpoint/Resume (`internal/checkpoint`)
- ✅ Drift Detection (`internal/drift`)

### Dependencies to Add
- [ ] File system watcher: `github.com/fsnotify/fsnotify`
- [ ] Retry library: `github.com/avast/retry-go`
- [ ] Progress bars: Already have `github.com/charmbracelet/bubbletea`

---

## Phase 1: Auto Mode Foundation

**Timeline:** 2-3 weeks
**Effort:** ~80-100 hours
**Goal:** Single command workflow with approval gates

### 1.1 Package Structure

Create new package `internal/auto`:

```
internal/auto/
├── auto.go           # Main orchestrator
├── config.go         # Configuration types
├── goal_parser.go    # Natural language → spec
├── approval.go       # Approval gate UI
├── executor.go       # Plan execution with progress
├── budget.go         # Cost tracking and limits
└── auto_test.go      # Comprehensive tests
```

### 1.2 Core Types

```go
// internal/auto/config.go
package auto

import (
    "time"
    "github.com/felixgeelhaar/specular/internal/policy"
)

// Config defines auto mode settings
type Config struct {
    // User's goal in natural language
    Goal string `yaml:"goal"`

    // Approval settings
    RequireApproval bool `yaml:"require_approval"`

    // Budget constraints
    MaxCostUSD      float64       `yaml:"max_cost_usd"`
    MaxCostPerTask  float64       `yaml:"max_cost_per_task"`

    // Retry settings
    MaxRetries      int           `yaml:"max_retries"`
    RetryDelay      time.Duration `yaml:"retry_delay"`

    // Timeout settings
    TimeoutMinutes  int           `yaml:"timeout_minutes"`
    TaskTimeout     time.Duration `yaml:"task_timeout"`

    // Policy enforcement
    PolicyPath      string        `yaml:"policy_path"`

    // Behavior flags
    FallbackToManual bool         `yaml:"fallback_to_manual"`
    Verbose          bool         `yaml:"verbose"`
    DryRun           bool         `yaml:"dry_run"`
}

// Result contains the outcome of auto mode execution
type Result struct {
    Success       bool
    Spec          *spec.ProductSpec
    SpecLock      *spec.SpecLock
    Plan          *plan.Plan
    EvalResult    *eval.GateReport
    DriftFindings []drift.Finding
    TotalCost     float64
    Duration      time.Duration
    TasksExecuted int
    TasksFailed   int
    Errors        []error
}
```

### 1.3 Goal Parser

Convert natural language goals into structured specifications:

```go
// internal/auto/goal_parser.go
package auto

import (
    "context"
    "fmt"
    "github.com/felixgeelhaar/specular/internal/router"
    "github.com/felixgeelhaar/specular/internal/spec"
)

type GoalParser struct {
    router *router.Router
}

func NewGoalParser(r *router.Router) *GoalParser {
    return &GoalParser{router: r}
}

// ParseGoal converts a natural language goal into a ProductSpec
func (p *GoalParser) ParseGoal(ctx context.Context, goal string) (*spec.ProductSpec, error) {
    systemPrompt := `You are a software specification expert. Convert the user's goal into a structured YAML specification following this format:

name: <project-name>
description: <brief-description>
version: 1.0.0
metadata:
  author: AI Generated
  created: <timestamp>

features:
  - id: <feature-id>
    title: <feature-title>
    description: <detailed-description>
    priority: P0|P1|P2
    category: <api|ui|data|infra>
    acceptance_criteria:
      - <criterion-1>
      - <criterion-2>

Focus on:
1. Clear, testable acceptance criteria
2. Proper priority assignment (P0 = critical, P1 = important, P2 = nice-to-have)
3. Logical feature breakdown
4. Realistic scope

Return ONLY the YAML, no explanations.`

    req := router.GenerateRequest{
        Prompt:       goal,
        SystemPrompt: systemPrompt,
        ModelHint:    "agentic",
        Complexity:   7,
        Priority:     "P0",
        Temperature:  0.3, // Lower temperature for structured output
        MaxTokens:    2000,
        TaskID:       domain.TaskID("goal-parse"),
    }

    resp, err := p.router.Generate(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("generate spec: %w", err)
    }

    // Parse YAML response into ProductSpec
    loader := spec.NewLoader()
    productSpec, err := loader.LoadFromString(resp.Content)
    if err != nil {
        return nil, fmt.Errorf("parse generated spec: %w", err)
    }

    return productSpec, nil
}
```

### 1.4 Approval Gate UI

```go
// internal/auto/approval.go
package auto

import (
    "fmt"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/felixgeelhaar/specular/internal/plan"
)

type approvalModel struct {
    plan     *plan.Plan
    approved bool
    quitting bool
}

func ShowApprovalGate(p *plan.Plan) (bool, error) {
    model := approvalModel{plan: p}
    program := tea.NewProgram(model)

    finalModel, err := program.Run()
    if err != nil {
        return false, fmt.Errorf("run approval UI: %w", err)
    }

    return finalModel.(approvalModel).approved, nil
}

func (m approvalModel) Init() tea.Cmd {
    return nil
}

func (m approvalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "y", "Y":
            m.approved = true
            m.quitting = true
            return m, tea.Quit
        case "n", "N", "q", "ctrl+c":
            m.approved = false
            m.quitting = true
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m approvalModel) View() string {
    if m.quitting {
        return ""
    }

    style := lipgloss.NewStyle().
        Foreground(lipgloss.Color("86")).
        Bold(true)

    s := style.Render("📋 Generated Execution Plan") + "\n\n"

    s += fmt.Sprintf("Total Tasks: %d\n", len(m.plan.Tasks))
    s += fmt.Sprintf("Estimated Duration: ~%d minutes\n\n", len(m.plan.Tasks)*5)

    // Show task breakdown by priority
    p0, p1, p2 := 0, 0, 0
    for _, task := range m.plan.Tasks {
        switch task.Priority {
        case domain.PriorityP0:
            p0++
        case domain.PriorityP1:
            p1++
        case domain.PriorityP2:
            p2++
        }
    }

    s += fmt.Sprintf("  P0 (Critical): %d tasks\n", p0)
    s += fmt.Sprintf("  P1 (Important): %d tasks\n", p1)
    s += fmt.Sprintf("  P2 (Nice-to-have): %d tasks\n\n", p2)

    // Show first 5 tasks
    s += "First 5 tasks:\n"
    for i, task := range m.plan.Tasks {
        if i >= 5 {
            break
        }
        s += fmt.Sprintf("  %d. [%s] %s\n", i+1, task.Priority, task.Title)
    }

    s += "\n"
    s += style.Render("Approve and execute?") + " (y/n): "

    return s
}
```

### 1.5 Main Auto Command

```go
// internal/cmd/auto.go
package cmd

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/spf13/cobra"
    "github.com/felixgeelhaar/specular/internal/auto"
    "github.com/felixgeelhaar/specular/internal/provider"
    "github.com/felixgeelhaar/specular/internal/router"
)

var autoCmd = &cobra.Command{
    Use:   "auto --goal \"<natural-language-goal>\"",
    Short: "Autonomous workflow from goal to working code",
    Long: `Run the complete development workflow autonomously:
  1. Convert natural language goal to specification
  2. Generate and lock specification
  3. Create execution plan
  4. Request approval (if enabled)
  5. Execute plan with automatic retry
  6. Run evaluation gate
  7. Detect and handle drift

Example:
  specular auto --goal "Build a REST API for user management"
  specular auto --goal "Create a CLI tool for project scaffolding" --no-approval`,
    RunE: runAutoMode,
}

func runAutoMode(cmd *cobra.Command, args []string) error {
    // Parse flags
    goal, _ := cmd.Flags().GetString("goal")
    requireApproval, _ := cmd.Flags().GetBool("approve-each-step")
    maxCost, _ := cmd.Flags().GetFloat64("max-cost")
    maxRetries, _ := cmd.Flags().GetInt("max-retries")
    timeoutMin, _ := cmd.Flags().GetInt("timeout")
    policyPath, _ := cmd.Flags().GetString("policy")
    verbose, _ := cmd.Flags().GetBool("verbose")
    dryRun, _ := cmd.Flags().GetBool("dry-run")

    if goal == "" {
        return fmt.Errorf("--goal flag is required")
    }

    // Build config
    config := auto.Config{
        Goal:             goal,
        RequireApproval:  requireApproval,
        MaxCostUSD:       maxCost,
        MaxRetries:       maxRetries,
        TimeoutMinutes:   timeoutMin,
        PolicyPath:       policyPath,
        Verbose:          verbose,
        DryRun:           dryRun,
        FallbackToManual: true,
    }

    // Load providers and create router
    registry, err := provider.LoadRegistryFromConfig(".specular/providers.yaml")
    if err != nil {
        return fmt.Errorf("load providers: %w", err)
    }

    routerConfig, err := router.LoadConfig(".specular/router.yaml")
    if err != nil {
        // Use defaults
        routerConfig = &router.RouterConfig{
            BudgetUSD:    20.0,
            MaxLatencyMs: 60000,
        }
    }

    r, err := router.NewRouterWithProviders(routerConfig, registry)
    if err != nil {
        return fmt.Errorf("create router: %w", err)
    }

    // Create orchestrator
    orchestrator := auto.NewOrchestrator(r, config)

    // Run autonomous workflow
    ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMin)*time.Minute)
    defer cancel()

    fmt.Printf("🤖 Starting autonomous workflow...\n")
    fmt.Printf("Goal: %s\n\n", goal)

    result, err := orchestrator.Execute(ctx)
    if err != nil {
        return fmt.Errorf("auto mode failed: %w", err)
    }

    // Display results
    displayAutoResult(result, verbose)

    if !result.Success {
        os.Exit(1)
    }

    return nil
}

func init() {
    rootCmd.AddCommand(autoCmd)

    autoCmd.Flags().String("goal", "", "Natural language project goal (required)")
    autoCmd.Flags().Bool("approve-each-step", true, "Request approval before execution")
    autoCmd.Flags().Float64("max-cost", 5.0, "Maximum cost in USD for this run")
    autoCmd.Flags().Int("max-retries", 3, "Maximum retry attempts for failed tasks")
    autoCmd.Flags().Int("timeout", 30, "Maximum workflow duration in minutes")
    autoCmd.Flags().String("policy", ".specular/policy.yaml", "Policy file path")
    autoCmd.Flags().Bool("verbose", false, "Show detailed progress")
    autoCmd.Flags().Bool("dry-run", false, "Validate without executing")

    autoCmd.MarkFlagRequired("goal")
}
```

### 1.6 Orchestrator Implementation

```go
// internal/auto/auto.go
package auto

import (
    "context"
    "fmt"
    "time"

    "github.com/felixgeelhaar/specular/internal/plan"
    "github.com/felixgeelhaar/specular/internal/router"
    "github.com/felixgeelhaar/specular/internal/spec"
    "github.com/felixgeelhaar/specular/internal/workflow"
)

type Orchestrator struct {
    router *router.Router
    config Config
    parser *GoalParser
}

func NewOrchestrator(r *router.Router, config Config) *Orchestrator {
    return &Orchestrator{
        router: r,
        config: config,
        parser: NewGoalParser(r),
    }
}

func (o *Orchestrator) Execute(ctx context.Context) (*Result, error) {
    start := time.Now()
    result := &Result{
        Success: false,
        Errors:  []error{},
    }

    // Step 1: Parse goal into spec
    fmt.Println("🤖 Generating specification from goal...")
    productSpec, err := o.parser.ParseGoal(ctx, o.config.Goal)
    if err != nil {
        return nil, fmt.Errorf("parse goal: %w", err)
    }
    result.Spec = productSpec
    fmt.Printf("✅ Generated spec with %d features\n\n", len(productSpec.Features))

    // Step 2: Generate spec lock
    fmt.Println("🔒 Locking specification...")
    specLock, err := spec.GenerateSpecLock(productSpec)
    if err != nil {
        return nil, fmt.Errorf("generate spec lock: %w", err)
    }
    result.SpecLock = specLock
    fmt.Printf("✅ Spec locked: %s\n\n", specLock.Hash[:12])

    // Step 3: Generate execution plan
    fmt.Println("📋 Generating execution plan...")
    execPlan, err := plan.Generate(ctx, productSpec, specLock)
    if err != nil {
        return nil, fmt.Errorf("generate plan: %w", err)
    }
    result.Plan = execPlan
    fmt.Printf("✅ Plan created: %d tasks\n\n", len(execPlan.Tasks))

    // Step 4: Approval gate (if enabled)
    if o.config.RequireApproval && !o.config.DryRun {
        approved, err := ShowApprovalGate(execPlan)
        if err != nil {
            return nil, fmt.Errorf("approval gate: %w", err)
        }
        if !approved {
            return result, fmt.Errorf("plan not approved by user")
        }
        fmt.Println("✅ Plan approved\n")
    }

    if o.config.DryRun {
        fmt.Println("🏁 Dry run complete (no execution)")
        result.Success = true
        result.Duration = time.Since(start)
        return result, nil
    }

    // Step 5: Execute plan
    fmt.Println("🚀 Executing plan...")
    // ... implementation continues in Phase 2

    result.Success = true
    result.Duration = time.Since(start)
    return result, nil
}
```

### 1.7 Testing Strategy

```go
// internal/auto/auto_test.go
package auto_test

import (
    "context"
    "testing"

    "github.com/felixgeelhaar/specular/internal/auto"
    "github.com/felixgeelhaar/specular/internal/router"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestGoalParser_SimpleGoal(t *testing.T) {
    // Test parsing a simple goal
    parser := auto.NewGoalParser(mockRouter())

    spec, err := parser.ParseGoal(context.Background(), "Build a TODO list API")

    require.NoError(t, err)
    assert.NotNil(t, spec)
    assert.Contains(t, spec.Name, "todo")
    assert.Greater(t, len(spec.Features), 0)
}

func TestOrchestrator_DryRun(t *testing.T) {
    // Test dry run mode
    config := auto.Config{
        Goal:            "Build a CLI tool",
        RequireApproval: false,
        DryRun:          true,
    }

    orchestrator := auto.NewOrchestrator(mockRouter(), config)

    result, err := orchestrator.Execute(context.Background())

    require.NoError(t, err)
    assert.True(t, result.Success)
    assert.NotNil(t, result.Spec)
    assert.NotNil(t, result.Plan)
}

func TestOrchestrator_BudgetEnforcement(t *testing.T) {
    // Test budget limits are respected
    config := auto.Config{
        Goal:       "Build a complex system",
        MaxCostUSD: 0.01, // Very low budget
        DryRun:     false,
    }

    orchestrator := auto.NewOrchestrator(mockRouter(), config)

    _, err := orchestrator.Execute(context.Background())

    // Should fail due to budget constraints
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "budget")
}
```

### 1.8 Deliverables

- [ ] `internal/auto` package with orchestrator
- [ ] `specular auto` command in `internal/cmd`
- [ ] Goal parser with AI integration
- [ ] Approval gate TUI
- [ ] Budget tracking and enforcement
- [ ] Unit tests (>80% coverage)
- [ ] Integration tests (end-to-end workflow)
- [ ] Documentation in `docs/commands/auto.md`

### 1.9 Success Criteria

✅ User can run `specular auto --goal "Build X"` and get a spec
✅ Approval gate shows plan details and accepts y/n input
✅ Budget limits prevent over-spending
✅ All tests pass
✅ Dry-run mode works correctly

---

## Phase 2: Error Recovery & Retry

**Timeline:** 1-2 weeks
**Effort:** ~40-60 hours
**Goal:** Automatic task retry with AI-powered regeneration

### 2.1 Failure Analysis

```go
// internal/auto/executor.go
package auto

import (
    "context"
    "fmt"
    "strings"
    "github.com/felixgeelhaar/specular/internal/plan"
    "github.com/felixgeelhaar/specular/internal/router"
)

type FailureAnalysis struct {
    FailedTask   *plan.Task
    Error        error
    Recoverable  bool
    ErrorType    string // compilation, runtime, timeout, etc.
    Suggestions  []string
    Context      map[string]string
}

func analyzeFailure(task *plan.Task, err error) *FailureAnalysis {
    analysis := &FailureAnalysis{
        FailedTask:  task,
        Error:       err,
        Recoverable: true,
        Context:     make(map[string]string),
    }

    errMsg := err.Error()

    // Classify error type
    switch {
    case strings.Contains(errMsg, "compile"):
        analysis.ErrorType = "compilation"
        analysis.Suggestions = []string{
            "Check syntax errors",
            "Verify imports",
            "Review type definitions",
        }

    case strings.Contains(errMsg, "timeout"):
        analysis.ErrorType = "timeout"
        analysis.Recoverable = false // Timeouts are usually environmental

    case strings.Contains(errMsg, "network"):
        analysis.ErrorType = "network"
        analysis.Recoverable = false // Network errors need manual intervention

    case strings.Contains(errMsg, "test failed"):
        analysis.ErrorType = "test_failure"
        analysis.Suggestions = []string{
            "Review test expectations",
            "Check test data",
            "Verify implementation logic",
        }

    default:
        analysis.ErrorType = "unknown"
        analysis.Suggestions = []string{
            "Review error message",
            "Check logs",
        }
    }

    return analysis
}

// RegenerateTask uses AI to create an improved version of a failed task
func (o *Orchestrator) regenerateTask(ctx context.Context, analysis *FailureAnalysis) (*plan.Task, error) {
    systemPrompt := fmt.Sprintf(`You are a software engineer fixing a failed task.

Original Task:
Title: %s
Description: %s
Skill: %s

Error Encountered:
%s

Error Type: %s
Suggestions: %s

Generate an improved implementation that addresses the error. Focus on:
1. Fixing the specific error mentioned
2. Adding error handling
3. Improving robustness
4. Following best practices

Return ONLY the corrected code or configuration, no explanations.`,
        analysis.FailedTask.Title,
        analysis.FailedTask.Description,
        analysis.FailedTask.Skill,
        analysis.Error.Error(),
        analysis.ErrorType,
        strings.Join(analysis.Suggestions, ", "),
    )

    req := router.GenerateRequest{
        Prompt:       "Fix the implementation based on the error analysis above",
        SystemPrompt: systemPrompt,
        ModelHint:    "codegen",
        Complexity:   8,
        Priority:     string(analysis.FailedTask.Priority),
        Temperature:  0.5,
        TaskID:       analysis.FailedTask.ID,
    }

    resp, err := o.router.Generate(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("regenerate task: %w", err)
    }

    // Create new task with regenerated content
    newTask := *analysis.FailedTask
    newTask.Output = resp.Content
    newTask.Attempt++

    return &newTask, nil
}
```

### 2.2 Retry Logic with Exponential Backoff

```go
// internal/auto/executor.go (continued)

import (
    "github.com/avast/retry-go/v4"
    "time"
)

func (o *Orchestrator) executeTaskWithRetry(ctx context.Context, task *plan.Task) error {
    attempt := 0

    return retry.Do(
        func() error {
            attempt++
            fmt.Printf("  🔄 Attempt %d/%d for task: %s\n", attempt, o.config.MaxRetries+1, task.Title)

            err := o.executeTask(ctx, task)
            if err == nil {
                fmt.Printf("  ✅ Task completed: %s\n", task.Title)
                return nil
            }

            // Analyze failure
            analysis := analyzeFailure(task, err)

            if !analysis.Recoverable {
                return retry.Unrecoverable(fmt.Errorf("unrecoverable error: %w", err))
            }

            fmt.Printf("  ❌ Task failed: %s\n", err.Error())
            fmt.Printf("  🤖 Analyzing failure (%s)...\n", analysis.ErrorType)

            // Regenerate task using AI
            newTask, regenerateErr := o.regenerateTask(ctx, analysis)
            if regenerateErr != nil {
                return fmt.Errorf("regeneration failed: %w", regenerateErr)
            }

            // Replace task for next retry
            *task = *newTask

            return err // Trigger retry
        },
        retry.Attempts(uint(o.config.MaxRetries+1)),
        retry.Delay(time.Second*2),
        retry.DelayType(retry.BackOffDelay),
        retry.MaxDelay(time.Second*30),
        retry.Context(ctx),
        retry.OnRetry(func(n uint, err error) {
            fmt.Printf("  ⏳ Waiting before retry %d...\n", n+1)
        }),
    )
}
```

### 2.3 Plan Execution with Progress Tracking

```go
// internal/auto/executor.go (continued)

func (o *Orchestrator) executePlanWithRetry(ctx context.Context, execPlan *plan.Plan) (*ExecutionResult, error) {
    result := &ExecutionResult{
        TasksExecuted: 0,
        TasksFailed:   0,
        TotalCost:     0.0,
    }

    fmt.Printf("🚀 Executing %d tasks...\n\n", len(execPlan.Tasks))

    for i, task := range execPlan.Tasks {
        fmt.Printf("[%d/%d] Starting: %s\n", i+1, len(execPlan.Tasks), task.Title)

        // Check budget before executing
        if result.TotalCost+0.10 > o.config.MaxCostUSD {
            return nil, fmt.Errorf("budget limit reached: $%.2f/$%.2f", result.TotalCost, o.config.MaxCostUSD)
        }

        // Execute with retry
        err := o.executeTaskWithRetry(ctx, &task)

        result.TasksExecuted++

        if err != nil {
            result.TasksFailed++
            fmt.Printf("  ❌ Task permanently failed after %d attempts\n\n", o.config.MaxRetries+1)

            // Decide whether to continue or abort
            if task.Priority == domain.PriorityP0 {
                return result, fmt.Errorf("critical task failed: %s", task.Title)
            }

            fmt.Printf("  ⚠️  Continuing despite failure (non-critical task)\n\n")
            continue
        }

        // Update cost tracking
        result.TotalCost += 0.05 // Estimate per task

        fmt.Println()
    }

    fmt.Printf("✅ Execution complete: %d/%d tasks successful\n",
        result.TasksExecuted-result.TasksFailed, result.TasksExecuted)

    return result, nil
}

type ExecutionResult struct {
    TasksExecuted int
    TasksFailed   int
    TotalCost     float64
}
```

### 2.4 Testing Error Recovery

```go
// internal/auto/executor_test.go
package auto_test

func TestExecutor_RetryOnFailure(t *testing.T) {
    // Mock a task that fails twice then succeeds
    failCount := 0
    mockExecutor := func(task *plan.Task) error {
        failCount++
        if failCount <= 2 {
            return fmt.Errorf("simulated failure %d", failCount)
        }
        return nil // Success on third try
    }

    config := auto.Config{MaxRetries: 3}
    orchestrator := auto.NewOrchestrator(mockRouter(), config)

    task := &plan.Task{
        ID:          domain.TaskID("test-task"),
        Title:       "Test Task",
        Priority:    domain.PriorityP1,
    }

    err := orchestrator.executeTaskWithRetry(context.Background(), task)

    assert.NoError(t, err)
    assert.Equal(t, 3, failCount) // Should have tried 3 times
}

func TestExecutor_UnrecoverableError(t *testing.T) {
    // Test that unrecoverable errors stop immediately
    config := auto.Config{MaxRetries: 3}
    orchestrator := auto.NewOrchestrator(mockRouter(), config)

    task := &plan.Task{
        ID:          domain.TaskID("test-task"),
        Title:       "Test Task",
        Priority:    domain.PriorityP0,
    }

    // Simulate unrecoverable error
    mockFailure := func() error {
        return fmt.Errorf("network timeout")
    }

    err := mockFailure()
    analysis := analyzeFailure(task, err)

    assert.False(t, analysis.Recoverable)
}
```

### 2.5 Deliverables

- [ ] Failure analysis logic
- [ ] Task regeneration with AI
- [ ] Retry with exponential backoff
- [ ] Budget tracking per task
- [ ] Error recovery tests
- [ ] Documentation updates

### 2.6 Success Criteria

✅ Failed tasks automatically retry with regenerated code
✅ Recovery success rate >70% for common errors
✅ Unrecoverable errors stop immediately
✅ Budget limits prevent infinite retries
✅ All tests pass

---

## Phase 3: Watch Mode

**Timeline:** 1 week
**Effort:** ~30-40 hours
**Goal:** Continuous monitoring and auto-rebuild on file changes

### 3.1 Watch Mode Implementation

```go
// internal/cmd/watch.go
package cmd

import (
    "context"
    "fmt"
    "time"

    "github.com/fsnotify/fsnotify"
    "github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
    Use:   "watch",
    Short: "Watch for file changes and auto-rebuild",
    Long: `Monitor the project directory for file changes and automatically:
  - Regenerate plan when spec changes
  - Rebuild when code changes
  - Run eval gate continuously
  - Display results in real-time

Example:
  specular watch --spec spec.yaml --auto-rebuild --auto-eval`,
    RunE: runWatchMode,
}

func runWatchMode(cmd *cobra.Command, args []string) error {
    specPath, _ := cmd.Flags().GetString("spec")
    autoRebuild, _ := cmd.Flags().GetBool("auto-rebuild")
    autoEval, _ := cmd.Flags().GetBool("auto-eval")
    debounceMs, _ := cmd.Flags().GetInt("debounce")
    ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")

    config := WatchConfig{
        SpecPath:       specPath,
        AutoRebuild:    autoRebuild,
        AutoEval:       autoEval,
        DebounceMs:     debounceMs,
        IgnorePatterns: ignorePatterns,
    }

    return runWatch(context.Background(), config)
}

type WatchConfig struct {
    SpecPath       string
    AutoRebuild    bool
    AutoEval       bool
    DebounceMs     int
    IgnorePatterns []string
}

func runWatch(ctx context.Context, config WatchConfig) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return fmt.Errorf("create watcher: %w", err)
    }
    defer watcher.Close()

    // Watch current directory
    err = watcher.Add(".")
    if err != nil {
        return fmt.Errorf("watch directory: %w", err)
    }

    fmt.Println("👀 Watching for changes...")
    fmt.Printf("Spec: %s\n", config.SpecPath)
    fmt.Printf("Auto-rebuild: %v\n", config.AutoRebuild)
    fmt.Printf("Auto-eval: %v\n\n", config.AutoEval)

    debounceTimer := time.NewTimer(0)
    <-debounceTimer.C // Drain initial timer

    for {
        select {
        case event := <-watcher.Events:
            if shouldIgnore(event.Name, config.IgnorePatterns) {
                continue
            }

            fmt.Printf("📝 File changed: %s\n", event.Name)

            // Reset debounce timer
            debounceTimer.Reset(time.Duration(config.DebounceMs) * time.Millisecond)

        case <-debounceTimer.C:
            fmt.Println("\n🔄 Rebuilding...")

            if config.AutoRebuild {
                err := rebuild(ctx, config.SpecPath)
                if err != nil {
                    fmt.Printf("❌ Rebuild failed: %v\n\n", err)
                    continue
                }
                fmt.Println("✅ Rebuild successful")
            }

            if config.AutoEval {
                err := runEvalGate(ctx, config.SpecPath)
                if err != nil {
                    fmt.Printf("❌ Eval failed: %v\n\n", err)
                    continue
                }
                fmt.Println("✅ Eval passed")
            }

            fmt.Println("\n👀 Watching for changes...")

        case err := <-watcher.Errors:
            fmt.Printf("⚠️  Watcher error: %v\n", err)

        case <-ctx.Done():
            return nil
        }
    }
}

func shouldIgnore(path string, patterns []string) bool {
    for _, pattern := range patterns {
        if strings.Contains(path, pattern) {
            return true
        }
    }
    return false
}

func init() {
    rootCmd.AddCommand(watchCmd)

    watchCmd.Flags().String("spec", ".specular/spec.yaml", "Spec file to watch")
    watchCmd.Flags().Bool("auto-rebuild", true, "Auto-rebuild on changes")
    watchCmd.Flags().Bool("auto-eval", true, "Auto-run eval gate")
    watchCmd.Flags().Int("debounce", 1000, "Debounce delay in ms")
    watchCmd.Flags().StringSlice("ignore", []string{"*.test.go", "vendor/**", ".git/**"}, "Patterns to ignore")
}
```

### 3.2 Deliverables

- [ ] `specular watch` command
- [ ] File system watcher implementation
- [ ] Debouncing logic
- [ ] Ignore patterns support
- [ ] Real-time progress display
- [ ] Tests for watch mode
- [ ] Documentation

### 3.3 Success Criteria

✅ File changes trigger rebuilds within debounce time
✅ Ignored patterns are respected
✅ Watch mode integrates with TDD workflows
✅ Ctrl+C cleanly stops watcher

---

## Phase 4: Full Autonomy

**Timeline:** 2-3 weeks
**Effort:** ~60-80 hours
**Goal:** Remove training wheels, add drift-based regeneration

### 4.1 Features

- [ ] Remove approval requirement (opt-in via flag)
- [ ] Drift-triggered plan regeneration
- [ ] Multi-session checkpoint/resume
- [ ] Cost optimization (model selection based on task success rates)
- [ ] Comprehensive audit logging
- [ ] Production readiness review

### 4.2 Deliverables

- [ ] `--no-approval` flag for auto mode
- [ ] Drift-based plan updates
- [ ] Checkpoint persistence
- [ ] Cost optimization algorithms
- [ ] Audit log system
- [ ] Production deployment guide

---

## Testing Strategy

### Unit Tests (Target: >80% coverage)
- ✅ Goal parser with various inputs
- ✅ Failure analysis for different error types
- ✅ Task regeneration logic
- ✅ Retry mechanisms
- ✅ Budget tracking
- ✅ File watcher patterns

### Integration Tests
- ✅ End-to-end auto mode workflow
- ✅ Error recovery scenarios
- ✅ Watch mode with real file changes
- ✅ Multi-phase workflows

### E2E Tests
- ✅ Real Docker execution
- ✅ Real AI provider calls (with mock fallback)
- ✅ Full project generation from goal

---

## Security Considerations

### Approval Gates
- Default to `require_approval: true`
- Show full plan before execution
- Clear cost estimates

### Budget Limits
- Hard caps on cost per run
- Per-task cost tracking
- Alert when approaching limits

### Policy Enforcement
- All executions pass policy gates
- Docker-only execution (no local commands)
- Image allowlist enforcement

### Audit Logging
- Log all autonomous decisions
- Track cost and token usage
- Record approval decisions

---

## Success Metrics

### Phase 1
- Users complete workflows 3x faster
- 90%+ user approval rate on generated plans
- Zero budget overruns

### Phase 2
- 70%+ recovery rate on first retry
- 50% reduction in manual intervention

### Phase 3
- 90%+ of developers use watch mode for TDD
- Sub-second rebuild triggering

### Phase 4
- Multi-hour autonomous workflows successful
- 30% cost reduction via optimization
- Production deployments without manual intervention

---

## Risk Mitigation

### Technical Risks
| Risk | Mitigation |
|------|------------|
| AI hallucinations | Approval gates, validation, policy enforcement |
| Cost overruns | Hard budget limits, cost tracking, alerts |
| Infinite retries | Max retry limits, unrecoverable error detection |
| Security issues | Docker sandbox, policy engine, audit logs |

### User Experience Risks
| Risk | Mitigation |
|------|------------|
| Mode confusion | Clear documentation, examples, help text |
| Over-automation | Approval gates by default, manual override |
| Trust issues | Transparent decisions, audit logs, dry-run mode |

---

## Documentation Deliverables

- [ ] `docs/commands/auto.md` - Auto mode guide
- [ ] `docs/commands/watch.md` - Watch mode guide
- [ ] `docs/guides/autonomous-workflows.md` - Complete guide
- [ ] `docs/guides/error-recovery.md` - Recovery strategies
- [ ] `docs/guides/cost-management.md` - Budget optimization

---

## Conclusion

This roadmap provides a clear, phased approach to implementing autonomous agent capabilities in Specular while preserving the manual control that power users need. Each phase builds on the previous one, with clear success criteria and testing requirements.

**Total Estimated Timeline:** 7-9 weeks
**Total Estimated Effort:** 210-280 hours

**Next Steps:**
1. Review and approve this roadmap
2. Create GitHub issues for Phase 1 tasks
3. Begin implementation of `internal/auto` package
4. Set up testing infrastructure

---

**Maintained by:** Specular Core Team
**Last Updated:** 2025-01-10
**Next Review:** After Phase 1 completion
