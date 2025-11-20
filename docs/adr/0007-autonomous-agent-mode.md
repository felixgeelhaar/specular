# ADR 0007: Autonomous Agent Mode

**Status:** Accepted

**Date:** 2025-01-10

**Decision Makers:** Specular Core Team

## Context

Specular currently operates as a **manual CLI tool** where users explicitly run each command in the development workflow:

```bash
# Current workflow requires 5+ manual steps
specular interview --tui               # 1. Generate spec
specular spec lock --in spec.yaml      # 2. Lock spec
specular plan --in spec.yaml           # 3. Generate plan
specular build --plan plan.json        # 4. Execute
specular eval --plan plan.json         # 5. Quality gate
```

This approach provides **explicit control** but requires:
- Users to understand the correct command sequence
- Manual intervention at each step
- No automatic error recovery
- Workflow knowledge for new users

### Comparison: Tool vs Agent Philosophies

Modern development tools follow two distinct philosophies:

#### Philosophy A: Tool-Oriented (Current Specular)
**Examples:** `make`, `docker`, `terraform`, `kubectl`

**Characteristics:**
- User controls workflow execution
- Each command is explicit and composable
- Minimal automation, maximum control
- Easy to debug and understand
- CI/CD friendly (scriptable)
- Low risk of unintended actions

**User Experience:**
```bash
$ specular plan --in spec.yaml
Plan generated: 12 tasks

$ specular build --plan plan.json
Building...

$ specular eval --plan plan.json
Tests passed ✓
```

#### Philosophy B: Agent-Oriented (Claude Code, Devin, Cursor)
**Examples:** Claude Code, GitHub Copilot Workspace, Devin

**Characteristics:**
- Agent controls workflow execution
- Single command triggers full workflow
- Automatic error recovery
- Minimal user intervention
- Learns from failures
- Higher productivity for complex tasks

**User Experience:**
```bash
$ specular auto --goal "Build a REST API"
🤖 Interviewing... ✓
📝 Generating spec... ✓
🔒 Locking spec... ✓
📋 Creating plan... ✓
🚀 Executing (12 tasks)...
  [██████████████] Task 8/12 failed
  🔄 Analyzing error...
  🔄 Regenerating task...
  ✓ Retry successful
✅ All tasks completed
```

### Current Architecture Capabilities

Specular **already has the building blocks** for autonomous operation:

1. ✅ **Workflow Orchestration** - `internal/workflow` orchestrates multi-step processes
2. ✅ **AI Provider System** - Can communicate with multiple LLMs
3. ✅ **Task DAG Execution** - Dependency-aware plan execution with `internal/exec`
4. ✅ **Docker Sandbox** - Secure isolated execution environment
5. ✅ **Policy Enforcement** - Safety guardrails via `internal/policy`
6. ✅ **Interactive TUI** - Beautiful terminal UI with progress tracking
7. ✅ **Checkpoint/Resume** - Can pause and resume long operations
8. ✅ **Error System** - Structured errors with suggestions and codes
9. ✅ **Model Router** - Intelligent model selection and retry/fallback

### User Feedback

During development, patterns emerged showing users want:
- **Quick iterations** without manual command chains
- **Error recovery** when tasks fail
- **Continuous monitoring** for file changes
- **Approval gates** for safety-critical operations
- **Manual override** when agent gets stuck

## Decision

**We will implement a hybrid approach supporting three operational modes:**

### 1. Manual Mode (Current - Keep)
```bash
# Explicit step-by-step control
specular plan --in spec.yaml
specular build --plan plan.json
specular eval --plan plan.json
```

**Use cases:**
- CI/CD pipelines
- Debugging specific steps
- Learning the tool
- Maximum control scenarios

### 2. Auto Mode (New - Autonomous Agent)
```bash
# Full workflow with approval gates
specular auto --goal "Build a CLI tool" --approve-each-step

# Or fully autonomous
specular auto --goal "..." --no-approval --max-cost $5.00
```

**Use cases:**
- Rapid prototyping
- Complex multi-step workflows
- Error recovery needed
- Interactive development

### 3. Watch Mode (New - Continuous Monitoring)
```bash
# Continuous rebuild on file changes
specular watch --spec spec.yaml --auto-rebuild --auto-eval
```

**Use cases:**
- TDD workflows
- Iterative development
- Continuous validation
- Live documentation

## Implementation Design

### Auto Mode Architecture

```go
// internal/cmd/auto.go
type AutoConfig struct {
    Goal            string
    RequireApproval bool
    MaxCost         float64
    MaxRetries      int
    TimeoutMinutes  int
    PolicyPath      string
}

func runAuto(ctx context.Context, config AutoConfig) error {
    // 1. Generate spec from natural language goal
    spec, err := generateSpecFromGoal(ctx, config.Goal)
    if err != nil {
        return fmt.Errorf("generate spec: %w", err)
    }

    // 2. Lock specification
    specLock, err := generateSpecLock(spec)
    if err != nil {
        return fmt.Errorf("lock spec: %w", err)
    }

    // 3. Generate execution plan
    plan, err := generatePlan(ctx, spec, specLock)
    if err != nil {
        return fmt.Errorf("generate plan: %w", err)
    }

    // 4. APPROVAL GATE (if enabled)
    if config.RequireApproval {
        approved, err := showPlanAndRequestApproval(plan)
        if err != nil || !approved {
            return fmt.Errorf("plan not approved")
        }
    }

    // 5. Execute plan with progress tracking
    result, err := executeWithRecovery(ctx, plan, config)
    if err != nil {
        return fmt.Errorf("execution failed: %w", err)
    }

    // 6. Run evaluation gate
    evalResult, err := runEvalGate(ctx, plan, config.PolicyPath)
    if err != nil {
        return fmt.Errorf("eval gate: %w", err)
    }

    // 7. Detect drift
    findings, err := detectDrift(ctx, spec, specLock, plan)
    if err != nil {
        return fmt.Errorf("drift detection: %w", err)
    }

    // 8. Handle drift (if any)
    if len(findings) > 0 {
        if config.RequireApproval {
            action := askUserAboutDrift(findings)
            if action == "retry" {
                return runAuto(ctx, config) // Recursive retry
            }
        }
    }

    return displayResults(result, evalResult, findings)
}
```

### Error Recovery Strategy

```go
func executeWithRecovery(ctx context.Context, plan *plan.Plan, config AutoConfig) (*ExecutionResult, error) {
    attempts := 0
    maxAttempts := config.MaxRetries

    for attempts < maxAttempts {
        result, err := executePlan(ctx, plan)

        if err == nil {
            return result, nil // Success
        }

        // Analyze failure
        analysis := analyzeFailure(err, result)

        if !analysis.Recoverable {
            return nil, fmt.Errorf("unrecoverable error: %w", err)
        }

        // Attempt recovery
        fmt.Printf("🔄 Task failed: %s\n", analysis.FailedTask)
        fmt.Printf("🤖 Analyzing error...\n")

        // Use AI to regenerate failing task
        newTask, err := regenerateTask(ctx, analysis.FailedTask, analysis.Error)
        if err != nil {
            return nil, fmt.Errorf("task regeneration failed: %w", err)
        }

        // Update plan with new task
        plan = updatePlanWithTask(plan, newTask)

        attempts++
        fmt.Printf("🔄 Retry %d/%d...\n", attempts, maxAttempts)
    }

    return nil, fmt.Errorf("max retries exceeded")
}
```

### Watch Mode Architecture

```go
// internal/cmd/watch.go
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

    // Watch project directory
    err = watcher.Add(".")
    if err != nil {
        return fmt.Errorf("watch directory: %w", err)
    }

    debounce := time.NewTimer(time.Duration(config.DebounceMs) * time.Millisecond)
    defer debounce.Stop()

    for {
        select {
        case event := <-watcher.Events:
            if shouldIgnore(event.Name, config.IgnorePatterns) {
                continue
            }

            // Debounce rapid changes
            debounce.Reset(time.Duration(config.DebounceMs) * time.Millisecond)

        case <-debounce.C:
            fmt.Println("🔄 File changes detected, rebuilding...")

            if config.AutoRebuild {
                err := rebuild(ctx, config.SpecPath)
                if err != nil {
                    fmt.Printf("❌ Rebuild failed: %v\n", err)
                    continue
                }
            }

            if config.AutoEval {
                err := runEval(ctx, config.SpecPath)
                if err != nil {
                    fmt.Printf("❌ Eval failed: %v\n", err)
                    continue
                }
            }

            fmt.Println("✅ Rebuild complete")

        case err := <-watcher.Errors:
            fmt.Printf("⚠️  Watcher error: %v\n", err)

        case <-ctx.Done():
            return nil
        }
    }
}
```

### Configuration

New configuration file `.specular/config.yaml`:

```yaml
# Operational mode
mode: manual  # Options: manual, auto, watch

# Auto mode settings
auto_mode:
  require_approval: true       # Ask before executing plans
  auto_retry: true             # Retry failed tasks automatically
  max_retries: 3               # Maximum retry attempts
  fallback_to_manual: true     # Drop to manual mode if agent stuck
  max_cost_per_run: 5.0        # USD budget limit
  timeout_minutes: 30          # Maximum workflow duration

# Watch mode settings
watch_mode:
  auto_rebuild: true           # Auto-rebuild on file changes
  auto_eval: true              # Auto-run eval gate
  debounce_ms: 1000            # Delay before triggering rebuild
  ignore_patterns:
    - "*.test.go"
    - "vendor/**"
    - ".git/**"
    - "node_modules/**"
```

## Consequences

### Positive

#### Manual Mode (Preserved)
- ✅ **Backward Compatible** - Existing users/scripts unaffected
- ✅ **CI/CD Friendly** - Still composable and scriptable
- ✅ **Debuggable** - Step-by-step inspection remains possible
- ✅ **Low Risk** - No autonomous operations by default

#### Auto Mode (New Capability)
- ✅ **Higher Productivity** - Single command for full workflow
- ✅ **Error Recovery** - Automatic retry and task regeneration
- ✅ **Better UX** - Interactive progress tracking
- ✅ **Approval Gates** - Safety mechanisms for critical operations
- ✅ **Budget Control** - Cost limits prevent runaway spending
- ✅ **Learning Aid** - Users see complete workflow execution

#### Watch Mode (New Capability)
- ✅ **TDD Support** - Continuous testing during development
- ✅ **Fast Feedback** - Immediate validation on changes
- ✅ **Documentation** - Live spec validation
- ✅ **CI/CD Integration** - Local dev matches CI behavior

### Negative

#### Implementation Complexity
- ❌ **More Code** - Significant new functionality (~3-5K LOC)
- ❌ **Testing Burden** - Complex workflows require extensive testing
- ❌ **Maintenance** - Two execution paths to maintain

#### User Experience Risks
- ❌ **Mode Confusion** - Users may not understand when to use each mode
- ❌ **Over-Automation** - Agent may take unexpected actions
- ❌ **Cost Surprises** - Autonomous retries could exceed budgets
- ❌ **Debugging Harder** - Autonomous workflows harder to trace

### Mitigations

#### Safety Mechanisms
1. **Approval Gates** - Default to `require_approval: true`
2. **Budget Limits** - Hard caps on cost per run
3. **Timeout Limits** - Prevent infinite loops
4. **Policy Enforcement** - All executions pass policy gates
5. **Audit Logging** - Complete logs of all autonomous actions
6. **Manual Override** - `Ctrl+C` always stops execution

#### Documentation
1. **Mode Selection Guide** - When to use each mode
2. **Auto Mode Tutorial** - Step-by-step examples
3. **Watch Mode Best Practices** - TDD workflows
4. **Troubleshooting** - Common issues and solutions
5. **Cost Management** - Budget and spending tracking

#### Testing Strategy
1. **Unit Tests** - Each component thoroughly tested
2. **Integration Tests** - Full workflow end-to-end
3. **E2E Tests** - Real Docker execution scenarios
4. **Failure Tests** - Error recovery paths validated
5. **Cost Tests** - Budget enforcement verified

## Alternatives Considered

### Alternative 1: Stay Manual Only
**Rejected Reason:** Doesn't address user friction with multi-step workflows

### Alternative 2: Full Agent Only
**Rejected Reason:** Loses composability, CI/CD integration, and debugging capability

### Alternative 3: Plugin-Based Modes
**Rejected Reason:** Over-engineers the solution, users just want modes not plugins

### Alternative 4: External Orchestrator
**Rejected Reason:** Adds deployment complexity, users want single binary

## Implementation Phases

### Phase 1: Auto Mode Foundation (2-3 weeks)
- [ ] Create `internal/auto` package
- [ ] Implement `auto` command with approval gates
- [ ] Add goal-to-spec generation using AI
- [ ] Implement basic progress tracking
- [ ] Add budget tracking
- [ ] Write unit and integration tests
- [ ] Document auto mode usage

**Deliverable:** `specular auto --goal "..." --approve-each-step`

### Phase 2: Error Recovery (1-2 weeks)
- [ ] Implement failure analysis
- [ ] Add task regeneration using AI
- [ ] Implement retry with exponential backoff
- [ ] Add max retries enforcement
- [ ] Test recovery paths extensively
- [ ] Document error recovery behavior

**Deliverable:** Automatic task retry on failure

### Phase 3: Watch Mode (1 week)
- [ ] Create `internal/watch` package
- [ ] Implement file system watcher
- [ ] Add debouncing logic
- [ ] Implement ignore patterns
- [ ] Add TUI for watch status
- [ ] Test watch mode workflows
- [ ] Document watch mode usage

**Deliverable:** `specular watch --auto-rebuild --auto-eval`

### Phase 4: Full Autonomy (2-3 weeks)
- [ ] Remove approval requirement (optional flag)
- [ ] Implement drift-based plan regeneration
- [ ] Add multi-session checkpoint/resume
- [ ] Implement cost optimization strategies
- [ ] Add comprehensive audit logging
- [ ] Extensive E2E testing
- [ ] Production readiness review

**Deliverable:** `specular auto --no-approval --max-cost $10`

## Success Metrics

### Phase 1 Success Criteria
- Users can run `specular auto --goal "..."` and get working code
- Approval gates prevent unintended actions
- Budget limits are enforced
- All tests pass with >80% coverage

### Phase 2 Success Criteria
- Failed tasks automatically retry with AI-regenerated code
- Recovery success rate >70% for common errors
- No infinite retry loops

### Phase 3 Success Criteria
- File changes trigger rebuilds within configured debounce time
- Ignored patterns are respected
- Watch mode integrates with TDD workflows

### Phase 4 Success Criteria
- Users can complete multi-hour workflows with checkpoints
- Drift detection triggers plan regeneration
- Cost optimization reduces spending by 30%
- Production deployments successful

## Related Decisions

- **ADR 0001:** Spec Lock File Format - Provides stable hashes for drift detection
- **ADR 0005:** Drift Detection Approach - Enables auto-regeneration triggers
- **ADR 0006:** Domain Value Objects - Type-safe identifiers for tasks and features

## References

- [Claude Code](https://claude.ai/code) - Interactive AI development assistant
- [GitHub Copilot Workspace](https://github.com/features/copilot) - Agent-based development
- [Devin](https://www.cognition-labs.com/devin) - Autonomous AI software engineer
- [Cursor](https://www.cursor.com/) - AI-native code editor
- [Make Manual](https://www.gnu.org/software/make/manual/) - Classic tool-oriented approach

## Future Enhancements

### Multi-Agent Collaboration
- Separate agents for spec generation, coding, testing, review
- Parallel task execution with agent coordination
- Agent specialization (frontend, backend, infra)

### Learning from Feedback
- Track which tasks frequently fail
- Learn user preferences for approval decisions
- Optimize model selection based on success rates

### Integration with External Tools
- GitHub integration for PR creation
- Slack notifications for workflow status
- VS Code extension for inline approval

### Advanced Error Recovery
- Semantic error analysis
- Context-aware task regeneration
- Learning from previous failures

---

**Last Updated:** 2025-01-10
**Next Review:** After Phase 1 implementation
**Owner:** Specular Core Team
