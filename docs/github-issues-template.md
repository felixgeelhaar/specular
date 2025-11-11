# GitHub Issues for Autonomous Mode Features

This document contains GitHub issue templates for all 14 autonomous mode features. Copy and paste these into GitHub issues to track implementation progress.

**Repository**: felixgeelhaar/specular
**Labels to create**: `auto-mode`, `v1.4.0`, `v1.5.0`, `v1.6.0`, `priority-critical`, `priority-high`, `priority-medium`, `priority-low`

---

## Phase 2: v1.4.0 - Production-Ready Features

### Issue #1: Profile System

**Title**: [Auto Mode] Feature #1: Profile System

**Labels**: `enhancement`, `auto-mode`, `v1.4.0`, `priority-critical`

**Body**:
```markdown
## Overview

Implement profile system for environment-specific configurations (default, ci, custom) to enable different behavior in CI vs interactive modes.

**Priority**: CRITICAL
**Effort**: 2 weeks
**Phase**: v1.4.0
**Dependencies**: None

## Business Value

- Enable CI/CD integration with non-interactive profiles
- Adapt behavior for different environments
- Configure approval rules per profile
- Set safety limits per profile (max_steps, timeout)
- Control routing preferences per profile

## Implementation Tasks

### 1.1 Design Profile Schema
- [ ] Define YAML structure for profiles (default, ci, custom)
- [ ] Specify approval rules per profile
- [ ] Define safety limits per profile (max_steps, timeout)
- [ ] Add routing preferences per profile
- **Files**: Design document in `docs/profiles-schema.md`

### 1.2 Profile Data Structures
- [ ] Create `internal/profiles/profile.go` with structs
- [ ] Implement Profile, ApprovalConfig, SafetyConfig, RoutingConfig structs
- [ ] Add proper validation
- **Files**: `internal/profiles/profile.go`
- **Tests**: `internal/profiles/profile_test.go`

### 1.3 Profile Loader
- [ ] Implement `LoadProfile(name string) (*Profile, error)`
- [ ] Search locations: `./auto.profiles.yaml`, `~/.specular/auto.profiles.yaml`
- [ ] Handle default, ci, and custom profiles
- [ ] Validate profile configuration
- **Files**: `internal/profiles/loader.go`
- **Tests**: `internal/profiles/loader_test.go`

### 1.4 CLI Integration
- [ ] Add `--profile <name>` flag to `specular auto`
- [ ] Add `--list-profiles` flag to show available profiles
- [ ] Merge profile config with CLI flags (CLI flags take precedence)
- **Files**: `internal/cmd/auto.go`
- **Tests**: `internal/cmd/auto_test.go`

### 1.5 Default Profiles
- [ ] Create example `auto.profiles.yaml` with default, ci, and custom profiles
- [ ] Add profile documentation
- **Files**: `examples/auto.profiles.yaml`, `docs/profiles.md`

### 1.6 Profile Tests
- [ ] Unit tests for profile loading and validation
- [ ] Integration tests for profile-based execution
- [ ] Test profile override behavior
- **Files**: `internal/profiles/*_test.go`

## Acceptance Criteria

- ✅ Profile system can load default, ci, and custom profiles
- ✅ Profiles work in both interactive and CI modes
- ✅ CLI flags override profile settings
- ✅ 90%+ test coverage
- ✅ Documentation complete with examples

## References

- Spec: `specular_auto_spec_v1.md`
- TODO: `docs/auto-mode-todo.md` (Feature #1)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 2)
```

---

### Issue #2: Structured Action Plan Format

**Title**: [Auto Mode] Feature #2: Structured Action Plan Format

**Labels**: `enhancement`, `auto-mode`, `v1.4.0`, `priority-critical`

**Body**:
```markdown
## Overview

Implement structured ActionPlan format with typed steps (spec:update, spec:lock, plan:gen, build:run) and approval requirements.

**Priority**: CRITICAL
**Effort**: 1 week
**Phase**: v1.4.0
**Dependencies**: None

## Business Value

- Standardized plan format for automation
- Clear step types for routing decisions
- Per-step approval control
- Better visibility into workflow stages
- Machine-readable plan structure

## Implementation Tasks

### 2.1 Action Plan Schema
- [ ] Define ActionPlan and ActionStep structures
- [ ] Add schema versioning ("specular.auto.plan/v1")
- [ ] Include metadata (created_at, version, profile)
- **Files**: `internal/auto/action_plan.go`

### 2.2 Plan Generation Updates
- [ ] Update `generatePlan()` to produce ActionPlan format
- [ ] Assign step types: spec:update, spec:lock, plan:gen, build:run
- [ ] Mark critical steps for approval (spec:lock, build:run)
- [ ] Add signals for routing hints
- **Files**: `internal/auto/orchestrator.go`
- **Tests**: `internal/auto/orchestrator_test.go`

### 2.3 Plan Execution Updates
- [ ] Update executor to handle ActionPlan format
- [ ] Implement per-step approval checks
- [ ] Track step execution status
- **Files**: `internal/auto/executor.go`
- **Tests**: `internal/auto/executor_test.go`

### 2.4 Plan Serialization
- [ ] Save ActionPlan to `plan.json` with schema version
- [ ] Support loading ActionPlan from checkpoint
- **Files**: `internal/auto/orchestrator.go`

### 2.5 Plan Validation
- [ ] Validate plan structure and dependencies
- [ ] Detect circular dependencies
- [ ] Validate step types
- **Files**: `internal/auto/plan_validator.go`
- **Tests**: `internal/auto/plan_validator_test.go`

## Acceptance Criteria

- ✅ Generated plans match ActionPlan schema
- ✅ Step types correctly assigned
- ✅ Plans can be serialized and loaded
- ✅ Plan validation catches invalid plans
- ✅ Backward compatible with v1.3.0 checkpoints

## References

- Spec: `specular_auto_spec_v1.md` (Action Plan Schema)
- TODO: `docs/auto-mode-todo.md` (Feature #2)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 2)
```

---

### Issue #3: Exit Codes (0-6)

**Title**: [Auto Mode] Feature #3: Exit Codes (0-6)

**Labels**: `enhancement`, `auto-mode`, `v1.4.0`, `priority-high`

**Body**:
```markdown
## Overview

Implement standardized exit codes (0-6) for different failure scenarios to enable proper CI/CD integration.

**Priority**: HIGH
**Effort**: 1 day
**Phase**: v1.4.0
**Dependencies**: None

## Business Value

- Enable CI/CD pipeline integration with proper error handling
- Distinguish between error types (usage, policy, network, etc.)
- Support automated retry logic based on exit codes
- Improve troubleshooting with clear error classification

## Exit Code Definitions

- `0`: Success
- `1`: Generic error
- `2`: Invalid CLI usage
- `3`: Policy violation
- `4`: Spec drift detected
- `5`: Authentication/permission failure
- `6`: Network or service unavailable

## Implementation Tasks

### 3.1 Define Exit Codes
- [ ] Create exit code constants
- [ ] Document each exit code
- **Files**: `internal/cmd/exit_codes.go`

### 3.2 Error Classification
- [ ] Update error handling to classify errors by exit code
- [ ] Map internal errors to appropriate exit codes
- **Files**: `internal/cmd/auto.go`

### 3.3 Exit Code Usage
- [ ] Return appropriate exit codes from `runAuto()`
- [ ] Add exit code to error messages
- **Files**: `internal/cmd/auto.go`

### 3.4 Documentation
- [ ] Document exit codes in CLI help
- [ ] Add exit codes to error messages
- **Files**: `internal/cmd/auto.go`, `README.md`

### 3.5 Tests
- [ ] Test each exit code scenario
- [ ] Verify exit codes in integration tests
- **Files**: `internal/cmd/auto_test.go`, `test/e2e/auto_test.go`

## Acceptance Criteria

- ✅ All exit codes defined and documented
- ✅ CLI returns correct exit codes for all scenarios
- ✅ Error messages include exit codes
- ✅ CI/CD can distinguish between error types
- ✅ All exit codes tested

## References

- Spec: `specular_auto_spec_v1.md` (Exit Codes)
- TODO: `docs/auto-mode-todo.md` (Feature #3)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 2)
```

---

### Issue #4: Per-Step Policy Checks

**Title**: [Auto Mode] Feature #4: Per-Step Policy Checks

**Labels**: `enhancement`, `auto-mode`, `v1.4.0`, `priority-high`

**Body**:
```markdown
## Overview

Implement per-step policy checks to enforce safety limits and governance rules before executing each workflow step.

**Priority**: HIGH
**Effort**: 1 week
**Phase**: v1.4.0
**Dependencies**: Feature #2 (Action Plan Format)

## Business Value

- Prevent unsafe operations before execution
- Enforce organizational policies
- Support compliance requirements
- Enable profile-specific policy rules
- Provide clear policy violation messages

## Implementation Tasks

### 4.1 Policy Check Interface
- [ ] Define PolicyChecker interface
- [ ] Define PolicyResult structure
- **Files**: `internal/policy/checker.go`

### 4.2 Built-in Policy Checks
- [ ] Implement cost limit check
- [ ] Implement timeout check
- [ ] Implement step type whitelist/blacklist
- [ ] Implement agent restriction check
- **Files**: `internal/policy/builtin.go`
- **Tests**: `internal/policy/builtin_test.go`

### 4.3 Profile-Based Policies
- [ ] Integrate policy checks with profile system
- [ ] Support profile-specific policy rules
- **Files**: `internal/policy/profile_policy.go`

### 4.4 Executor Integration
- [ ] Add policy check before each step execution
- [ ] Handle policy violations (abort or skip step)
- [ ] Log policy check results
- **Files**: `internal/auto/executor.go`

### 4.5 Policy Violation Handling
- [ ] Return ExitPolicyViolation (3) on policy failure
- [ ] Provide clear error messages for violations
- [ ] Support override flag for non-critical policies
- **Files**: `internal/cmd/auto.go`

### 4.6 Tests
- [ ] Unit tests for policy checks
- [ ] Integration tests for policy enforcement
- [ ] Test policy override behavior
- **Files**: `internal/policy/*_test.go`, `test/e2e/policy_test.go`

## Acceptance Criteria

- ✅ Policy checks run before each step
- ✅ Policy violations properly handled and reported
- ✅ Profile-specific policies working
- ✅ Exit code 3 returned on policy violation
- ✅ 90%+ test coverage

## References

- Spec: `specular_auto_spec_v1.md` (Policy Checks)
- TODO: `docs/auto-mode-todo.md` (Feature #4)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 2)
```

---

### Issue #5: JSON Output Format

**Title**: [Auto Mode] Feature #5: JSON Output Format

**Labels**: `enhancement`, `auto-mode`, `v1.4.0`, `priority-high`

**Body**:
```markdown
## Overview

Implement machine-readable JSON output format for automation, dashboards, and CI/CD integration.

**Priority**: HIGH
**Effort**: 3 days
**Phase**: v1.4.0
**Dependencies**: Feature #2 (Action Plan Format)

## Business Value

- Enable integration with external tools and dashboards
- Support automated analysis of workflow results
- Provide comprehensive execution metrics
- Include full audit trail
- Machine-readable artifacts list

## Implementation Tasks

### 5.1 Output Schema
- [ ] Define AutoOutput structure with schema version
- [ ] Define StepResult, ExecutionMetrics, AuditTrail structures
- [ ] Add artifact information
- **Files**: `internal/auto/output.go`

### 5.2 Output Generation
- [ ] Collect step results during execution
- [ ] Collect metrics and audit events
- [ ] Generate AutoOutput structure
- **Files**: `internal/auto/orchestrator.go`

### 5.3 CLI Integration
- [ ] Add `--output json` flag
- [ ] Write JSON output to stdout or file
- [ ] Ensure JSON is valid and parseable
- **Files**: `internal/cmd/auto.go`

### 5.4 Documentation
- [ ] Document JSON schema
- [ ] Provide example JSON output
- **Files**: `docs/auto-json-output.md`, `examples/auto-output.json`

### 5.5 Tests
- [ ] Test JSON serialization
- [ ] Test output with different execution scenarios
- [ ] Validate JSON schema compliance
- **Files**: `internal/auto/output_test.go`

## Acceptance Criteria

- ✅ JSON output properly formatted
- ✅ Schema includes all required information
- ✅ Output parseable by standard JSON tools
- ✅ Documentation complete with examples
- ✅ All scenarios tested

## References

- Spec: `specular_auto_spec_v1.md` (JSON Output)
- TODO: `docs/auto-mode-todo.md` (Feature #5)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 2)
```

---

## Phase 3: v1.5.0 - Enhanced UX Features

### Issue #6: Scope Filtering

**Title**: [Auto Mode] Feature #6: Scope Filtering

**Labels**: `enhancement`, `auto-mode`, `v1.5.0`, `priority-medium`

**Body**:
```markdown
## Overview

Implement scope filtering to enable targeted execution of specific paths, features, or components.

**Priority**: MEDIUM
**Effort**: 3 days
**Phase**: v1.5.0
**Dependencies**: Feature #2 (Action Plan Format)

## Business Value

- Enable working on specific features without full workflow
- Reduce execution time for targeted changes
- Support incremental development
- Filter by path patterns or feature tags

## Implementation Tasks

### 6.1 Scope Parser
- [ ] Parse scope patterns (paths, features)
- [ ] Support glob patterns (e.g., `src/components/**`)
- [ ] Support feature tags
- **Files**: `internal/auto/scope.go`
- **Tests**: `internal/auto/scope_test.go`

### 6.2 Plan Filtering
- [ ] Filter action plan steps based on scope
- [ ] Preserve step dependencies
- [ ] Update step count and estimates
- **Files**: `internal/auto/plan_filter.go`

### 6.3 CLI Integration
- [ ] Add `--scope <pattern>` flag
- [ ] Support multiple scope patterns
- **Files**: `internal/cmd/auto.go`

### 6.4 Tests
- [ ] Test scope parsing
- [ ] Test plan filtering
- [ ] Test scope with dependencies
- **Files**: `internal/auto/*_test.go`

## Acceptance Criteria

- ✅ Scope patterns parsed correctly
- ✅ Plans correctly filtered by scope
- ✅ Dependencies preserved
- ✅ Multiple scopes supported
- ✅ Fully tested

## References

- Spec: `specular_auto_spec_v1.md` (Scope Filtering)
- TODO: `docs/auto-mode-todo.md` (Feature #6)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 3)
```

---

### Issue #7: Max Steps Limit

**Title**: [Auto Mode] Feature #7: Max Steps Limit

**Labels**: `enhancement`, `auto-mode`, `v1.5.0`, `priority-medium`

**Body**:
```markdown
## Overview

Implement max steps limit to prevent runaway workflows and enforce safety constraints.

**Priority**: MEDIUM
**Effort**: 1 day
**Phase**: v1.5.0
**Dependencies**: Feature #2 (Action Plan Format)

## Business Value

- Prevent runaway workflows
- Enforce safety limits in CI environments
- Save costs by limiting execution
- Support profile-specific limits

## Implementation Tasks

### 7.1 Step Counter
- [ ] Track executed steps
- [ ] Check against max_steps limit
- **Files**: `internal/auto/executor.go`

### 7.2 Limit Enforcement
- [ ] Abort execution when limit reached
- [ ] Save partial results
- [ ] Return appropriate exit code
- **Files**: `internal/auto/executor.go`

### 7.3 CLI Integration
- [ ] Add `--max-steps <n>` flag
- [ ] Integrate with profile system
- **Files**: `internal/cmd/auto.go`

### 7.4 Tests
- [ ] Test limit enforcement
- [ ] Test partial completion
- **Files**: `internal/auto/executor_test.go`

## Acceptance Criteria

- ✅ Steps properly counted
- ✅ Limit enforced correctly
- ✅ Partial results saved
- ✅ Profile integration working
- ✅ Fully tested

## References

- Spec: `specular_auto_spec_v1.md` (Max Steps)
- TODO: `docs/auto-mode-todo.md` (Feature #7)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 3)
```

---

### Issue #8: Interactive TUI

**Title**: [Auto Mode] Feature #8: Interactive TUI

**Labels**: `enhancement`, `auto-mode`, `v1.5.0`, `priority-medium`

**Body**:
```markdown
## Overview

Implement interactive Terminal UI using Bubble Tea framework for enhanced developer experience.

**Priority**: MEDIUM
**Effort**: 2 weeks
**Phase**: v1.5.0
**Dependencies**: Feature #2 (Action Plan Format)

## Business Value

- Significantly improve developer experience
- Real-time progress visualization
- Interactive approval flow
- Hotkeys for common actions
- Better workflow visibility

## Features

- Main view with goal, current step, progress, cost
- Step list view with status indicators
- Hotkeys: `?` (help), `v` (verbose), `s` (steps), `a` (approve), `q` (quit)
- Interactive approval flow
- Fallback to text mode if TUI unavailable

## Implementation Tasks

### 8.1 TUI Framework Setup
- [ ] Add Bubble Tea dependency
- [ ] Create TUI model structure
- **Files**: `internal/tui/model.go`

### 8.2 Main View
- [ ] Display goal and current step
- [ ] Show progress (X/Y steps)
- [ ] Display current cost
- **Files**: `internal/tui/views/main.go`

### 8.3 Step List View
- [ ] Show all steps with status
- [ ] Highlight current step
- [ ] Show step types and approval status
- **Files**: `internal/tui/views/steps.go`

### 8.4 Hotkeys
- [ ] Implement `?` for help
- [ ] Implement `v` for verbose mode
- [ ] Implement `s` for step list
- [ ] Implement `a` for approval
- [ ] Implement `q` for quit
- **Files**: `internal/tui/keys.go`

### 8.5 Approval Flow
- [ ] Show approval prompt
- [ ] Display step details
- [ ] Handle approve/reject
- **Files**: `internal/tui/views/approval.go`

### 8.6 CLI Integration
- [ ] Add `--tui` flag
- [ ] Fall back to text mode if TUI fails
- **Files**: `internal/cmd/auto.go`

### 8.7 Tests
- [ ] Test TUI model state transitions
- [ ] Test hotkey handling
- **Files**: `internal/tui/*_test.go`

## Acceptance Criteria

- ✅ TUI renders correctly on major terminals
- ✅ All hotkeys working
- ✅ Approval flow functional
- ✅ Graceful fallback to text mode
- ✅ User satisfaction: 80%+ (NPS)

## References

- Spec: `specular_auto_spec_v1.md` (Interactive TUI)
- TODO: `docs/auto-mode-todo.md` (Feature #8)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 3)
```

---

### Issue #9: Trace Logging

**Title**: [Auto Mode] Feature #9: Trace Logging

**Labels**: `enhancement`, `auto-mode`, `v1.5.0`, `priority-medium`

**Body**:
```markdown
## Overview

Implement comprehensive trace logging to `~/.specular/logs/trace_<id>.json` for debugging and audit trails.

**Priority**: MEDIUM
**Effort**: 3 days
**Phase**: v1.5.0
**Dependencies**: None

## Business Value

- Enable comprehensive debugging
- Provide audit trails for compliance
- Support post-mortem analysis
- Track policy checks and approvals
- Correlate events across workflow

## Implementation Tasks

### 9.1 Trace Schema
- [ ] Define trace event structure
- [ ] Support different event types
- **Files**: `internal/trace/event.go`

### 9.2 Trace Logger
- [ ] Implement trace event logger
- [ ] Write to `~/.specular/logs/trace_<id>.json`
- [ ] Support log rotation
- **Files**: `internal/trace/logger.go`
- **Tests**: `internal/trace/logger_test.go`

### 9.3 Event Collection
- [ ] Log all critical events
- [ ] Include timestamps and context
- [ ] Log policy checks and approvals
- **Files**: `internal/auto/orchestrator.go`

### 9.4 CLI Integration
- [ ] Add `--trace` flag
- [ ] Show trace file location
- **Files**: `internal/cmd/auto.go`

### 9.5 Tests
- [ ] Test trace logging
- [ ] Test event structure
- **Files**: `internal/trace/*_test.go`

## Acceptance Criteria

- ✅ All critical events logged
- ✅ Trace files properly formatted
- ✅ Log rotation working
- ✅ Trace flag functional
- ✅ Fully tested

## References

- Spec: `specular_auto_spec_v1.md` (Trace Logging)
- TODO: `docs/auto-mode-todo.md` (Feature #9)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 3)
```

---

### Issue #10: Patch Generation

**Title**: [Auto Mode] Feature #10: Patch Generation

**Labels**: `enhancement`, `auto-mode`, `v1.5.0`, `priority-medium`

**Body**:
```markdown
## Overview

Implement patch generation for each step to enable rollback of unwanted changes.

**Priority**: MEDIUM
**Effort**: 1 week
**Phase**: v1.5.0
**Dependencies**: Feature #2 (Action Plan Format)

## Business Value

- Enable quick rollback of unwanted changes
- Provide reversible workflow execution
- Support incremental testing
- Generate audit trail of changes
- Handle conflicts gracefully

## Implementation Tasks

### 10.1 Diff Generation
- [ ] Capture file changes per step
- [ ] Generate unified diff format
- **Files**: `internal/patch/diff.go`

### 10.2 Patch Files
- [ ] Write `.patch` files per step
- [ ] Include metadata (step ID, timestamp)
- **Files**: `internal/patch/writer.go`

### 10.3 Rollback Support
- [ ] Apply patches in reverse
- [ ] Handle conflicts
- **Files**: `internal/patch/rollback.go`

### 10.4 CLI Integration
- [ ] Add `--save-patches` flag
- [ ] Add `specular auto rollback` command
- **Files**: `internal/cmd/auto.go`, `internal/cmd/auto_rollback.go`

### 10.5 Tests
- [ ] Test diff generation
- [ ] Test patch application
- [ ] Test rollback
- **Files**: `internal/patch/*_test.go`

## Acceptance Criteria

- ✅ Diffs generated correctly
- ✅ Patches can be applied in reverse
- ✅ Conflicts handled gracefully
- ✅ Rollback command working
- ✅ 95%+ patch application success

## References

- Spec: `specular_auto_spec_v1.md` (Patch Generation)
- TODO: `docs/auto-mode-todo.md` (Feature #10)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 3)
```

---

## Phase 4: v1.6.0 - Enterprise Features

### Issue #11: Attestation (Sigstore)

**Title**: [Auto Mode] Feature #11: Attestation (Sigstore)

**Labels**: `enhancement`, `auto-mode`, `v1.6.0`, `priority-low`

**Body**:
```markdown
## Overview

Implement cryptographic attestation using Sigstore for compliance and provenance tracking.

**Priority**: LOW
**Effort**: 2 weeks
**Phase**: v1.6.0
**Dependencies**: Feature #5 (JSON Output)

## Business Value

- Provide cryptographic proof of execution
- Meet compliance requirements (SOC 2, ISO 27001)
- Enable supply chain security
- Support provenance tracking
- Enable verification of workflow results

## Implementation Tasks

### 11.1 Sigstore Integration
- [ ] Add sigstore-go dependency
- [ ] Implement signing interface
- **Files**: `internal/attestation/signer.go`

### 11.2 Attestation Generation
- [ ] Sign plan and output
- [ ] Include provenance data
- **Files**: `internal/attestation/generate.go`

### 11.3 Verification
- [ ] Verify attestation signatures
- [ ] Validate provenance
- **Files**: `internal/attestation/verify.go`

### 11.4 CLI Integration
- [ ] Add `--attest` flag
- [ ] Add verification command
- **Files**: `internal/cmd/auto.go`

### 11.5 Tests
- [ ] Test signing
- [ ] Test verification
- **Files**: `internal/attestation/*_test.go`

## Acceptance Criteria

- ✅ Attestations generated correctly
- ✅ 100% verification success rate
- ✅ Sigstore integration working
- ✅ Passes compliance audit
- ✅ Fully tested

## References

- Spec: `specular_auto_spec_v1.md` (Attestation)
- TODO: `docs/auto-mode-todo.md` (Feature #11)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 4)
```

---

### Issue #12: Explain Routing

**Title**: [Auto Mode] Feature #12: Explain Routing

**Labels**: `enhancement`, `auto-mode`, `v1.6.0`, `priority-low`

**Body**:
```markdown
## Overview

Implement `specular explain` command to show routing decisions and agent selection rationale.

**Priority**: LOW
**Effort**: 1 week
**Phase**: v1.6.0
**Dependencies**: Feature #1 (Profile System)

## Business Value

- Improve transparency in routing decisions
- Help users understand agent selection
- Support debugging and optimization
- Build trust in autonomous mode
- Enable routing tuning

## Implementation Tasks

### 12.1 Routing Explainer
- [ ] Analyze plan routing decisions
- [ ] Show agent selection rationale
- **Files**: `internal/explain/routing.go`

### 12.2 CLI Command
- [ ] Add `specular explain <checkpoint-id>` command
- [ ] Format explanation output
- **Files**: `internal/cmd/explain.go`

### 12.3 Tests
- [ ] Test routing analysis
- [ ] Test output formatting
- **Files**: `internal/explain/*_test.go`

## Acceptance Criteria

- ✅ Routing decisions explained clearly
- ✅ Agent selection rationale provided
- ✅ Command working correctly
- ✅ Output easy to understand
- ✅ Fully tested

## References

- Spec: `specular_auto_spec_v1.md` (Explain)
- TODO: `docs/auto-mode-todo.md` (Feature #12)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 4)
```

---

### Issue #13: Hooks System

**Title**: [Auto Mode] Feature #13: Hooks System

**Labels**: `enhancement`, `auto-mode`, `v1.6.0`, `priority-low`

**Body**:
```markdown
## Overview

Implement extensible hooks system for workflow customization (Slack notifications, webhooks, etc.).

**Priority**: LOW
**Effort**: 2 weeks
**Phase**: v1.6.0
**Dependencies**: Feature #2 (Action Plan Format)

## Business Value

- Enable workflow customization
- Support integration with existing tools (Slack, webhooks)
- Provide lifecycle hooks (on_plan_created, on_step_before, etc.)
- Enable organizational workflows
- Support custom actions

## Hook Types

- `on_plan_created`: After plan generation
- `on_step_before`: Before step execution
- `on_step_after`: After step completion
- `on_approval_requested`: When approval needed
- `on_complete`: When workflow completes
- `on_error`: When errors occur

## Implementation Tasks

### 13.1 Hook Interface
- [ ] Define hook types
- [ ] Create hook executor interface
- **Files**: `internal/hooks/interface.go`

### 13.2 Hook Registry
- [ ] Register and manage hooks
- [ ] Support multiple hooks per event
- **Files**: `internal/hooks/registry.go`

### 13.3 Hook Execution
- [ ] Execute hooks at appropriate points
- [ ] Handle hook failures
- **Files**: `internal/hooks/executor.go`

### 13.4 Built-in Hooks
- [ ] Slack notification hook
- [ ] Webhook hook
- **Files**: `internal/hooks/builtin.go`

### 13.5 Configuration
- [ ] Add hooks to profile configuration
- [ ] Document hook configuration
- **Files**: `internal/profiles/profile.go`, `docs/hooks.md`

### 13.6 Tests
- [ ] Test hook execution
- [ ] Test hook failures
- **Files**: `internal/hooks/*_test.go`

## Acceptance Criteria

- ✅ Hooks execute at correct lifecycle points
- ✅ Multiple hooks per event supported
- ✅ Built-in hooks working (Slack, webhook)
- ✅ Hook failures handled gracefully
- ✅ Fully tested and documented

## References

- Spec: `specular_auto_spec_v1.md` (Hooks)
- TODO: `docs/auto-mode-todo.md` (Feature #13)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 4)
```

---

### Issue #14: Advanced Security

**Title**: [Auto Mode] Feature #14: Advanced Security

**Labels**: `enhancement`, `auto-mode`, `v1.6.0`, `priority-low`

**Body**:
```markdown
## Overview

Implement advanced security features including credential management, enhanced audit logging, and secret scanning.

**Priority**: LOW
**Effort**: 1 week
**Phase**: v1.6.0
**Dependencies**: Feature #1 (Profile System), Feature #4 (Policy Checks)

## Business Value

- Meet enterprise security requirements
- Support credential rotation
- Prevent secret leakage
- Comprehensive audit logging
- Compliance reporting (SOC 2, ISO 27001)

## Implementation Tasks

### 14.1 Credential Management
- [ ] Secure credential storage
- [ ] Credential rotation support
- **Files**: `internal/security/credentials.go`

### 14.2 Audit Logging
- [ ] Enhanced audit logging
- [ ] Compliance reporting
- **Files**: `internal/security/audit.go`

### 14.3 Secret Scanning
- [ ] Scan for secrets in code
- [ ] Block commits with secrets
- **Files**: `internal/security/secrets.go`

### 14.4 Tests
- [ ] Test credential management
- [ ] Test audit logging
- [ ] Test secret scanning
- **Files**: `internal/security/*_test.go`

## Acceptance Criteria

- ✅ Credentials managed securely
- ✅ Audit logging comprehensive
- ✅ Secret scanning catches common secrets
- ✅ Passes security audit
- ✅ Fully tested

## References

- Spec: `specular_auto_spec_v1.md` (Security)
- TODO: `docs/auto-mode-todo.md` (Feature #14)
- Roadmap: `docs/auto-mode-roadmap.md` (Phase 4)
```

---

## How to Use This Template

1. **Create GitHub Labels** (if not already created):
   ```bash
   # Auto mode label
   gh label create "auto-mode" --color "0E8A16" --description "Autonomous mode features"

   # Version labels
   gh label create "v1.4.0" --color "1D76DB" --description "Phase 2: Production-Ready"
   gh label create "v1.5.0" --color "5319E7" --description "Phase 3: Enhanced UX"
   gh label create "v1.6.0" --color "B60205" --description "Phase 4: Enterprise"

   # Priority labels
   gh label create "priority-critical" --color "D93F0B" --description "Critical priority"
   gh label create "priority-high" --color "FBCA04" --description "High priority"
   gh label create "priority-medium" --color "FEF2C0" --description "Medium priority"
   gh label create "priority-low" --color "C2E0C6" --description "Low priority"
   ```

2. **Create Issues**:
   - Copy each issue template above
   - Go to https://github.com/felixgeelhaar/specular/issues/new
   - Paste the title and body
   - Add the specified labels
   - Click "Submit new issue"

3. **Create Project Board** (optional):
   - Create a new GitHub Project for "Autonomous Mode Roadmap"
   - Add columns: Backlog, v1.4.0, v1.5.0, v1.6.0, In Progress, Review, Done
   - Add all created issues to the board

4. **Milestone Setup** (optional):
   ```bash
   gh milestone create "v1.4.0 Production-Ready" --due-date "2025-12-22"
   gh milestone create "v1.5.0 Enhanced UX" --due-date "2026-01-19"
   gh milestone create "v1.6.0 Enterprise" --due-date "2026-03-01"
   ```

---

**Generated**: 2025-11-11
**Total Issues**: 14
**Total Effort**: 16 weeks
