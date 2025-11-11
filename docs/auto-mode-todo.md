# Autonomous Mode Implementation TODO

This document provides a comprehensive, actionable TODO list for implementing the missing features identified in the autonomous mode gap analysis.

## Overview

- **Total Features**: 14 features ✅ **ALL COMPLETE**
- **Total Estimated Effort**: 16 weeks
- **Target Releases**: v1.4.0 (6 weeks), v1.5.0 (4 weeks), v1.6.0 (6 weeks)
- **Status**: ✅ **ALL PHASES COMPLETE** - Phase 1 (v1.3.0), Phase 2 (v1.4.0), Phase 3 (v1.5.0), Phase 4 (v1.6.0)

## Completion Summary

All 14 autonomous mode features have been successfully implemented, tested, and documented:

**Phase 2 (v1.4.0) - Production-Ready Features**: ✅ 5/5 Complete
- Feature #1: Profile System ✅
- Feature #2: Structured Action Plan Format ✅
- Feature #3: Exit Codes ✅
- Feature #4: Per-Step Policy Checks ✅
- Feature #5: JSON Output Format ✅

**Phase 3 (v1.5.0) - Enhanced UX**: ✅ 5/5 Complete
- Feature #6: Scope Filtering ✅
- Feature #7: Max Steps Limit ✅
- Feature #8: Interactive TUI ✅
- Feature #9: Trace Logging ✅
- Feature #10: Patch Generation ✅

**Phase 4 (v1.6.0) - Advanced Features**: ✅ 4/4 Complete
- Feature #11: Cryptographic Attestations ✅
- Feature #12: Explain Routing ✅ (minor: checkpoint loading placeholder)
- Feature #13: Hooks System ✅
- Feature #14: Advanced Security ✅

**Documentation**: 4363 lines in README.md covering all features with examples, use cases, and best practices

**Test Coverage**:
- All packages tested with comprehensive test suites
- Hooks: 32 tests passing
- Security: 47 tests passing
- Patch: 25 tests passing
- Attestation: 21 tests passing
- Explain: 13 tests passing

---

## Phase 2: v1.4.0 - Production-Ready Features (6 weeks)

### Priority 1: Must Have (Blocks Production)

---

### 1. Profile System (`auto.profiles.yaml`)

**Effort**: 2 weeks | **Priority**: CRITICAL | **Dependencies**: None

#### Implementation Tasks

- [ ] **1.1 Design Profile Schema**
  - [ ] Define YAML structure for profiles (default, ci, custom)
  - [ ] Specify approval rules per profile
  - [ ] Define safety limits per profile (max_steps, timeout)
  - [ ] Add routing preferences per profile
  - **Files**: Design document in `docs/profiles-schema.md`
  - **Acceptance**: Schema approved by team, covers all use cases

- [ ] **1.2 Profile Data Structures**
  - [ ] Create `internal/profiles/profile.go` with structs
    ```go
    type Profile struct {
        Name      string
        Approvals ApprovalConfig
        Safety    SafetyConfig
        Routing   RoutingConfig
    }

    type ApprovalConfig struct {
        CriticalOnly    bool
        Noninteractive  bool
        AutoApprove     []string
    }

    type SafetyConfig struct {
        MaxSteps       int
        Timeout        time.Duration
        MaxCostUSD     float64
        RequirePolicy  bool
    }

    type RoutingConfig struct {
        PreferredAgent string
        FallbackAgent  string
        Temperature    float64
    }
    ```
  - **Files**: `internal/profiles/profile.go`
  - **Tests**: `internal/profiles/profile_test.go`
  - **Acceptance**: All structs defined with proper validation

- [ ] **1.3 Profile Loader**
  - [ ] Implement `LoadProfile(name string) (*Profile, error)`
  - [ ] Search locations: `./auto.profiles.yaml`, `~/.specular/auto.profiles.yaml`
  - [ ] Handle default, ci, and custom profiles
  - [ ] Validate profile configuration
  - **Files**: `internal/profiles/loader.go`
  - **Tests**: `internal/profiles/loader_test.go`
  - **Acceptance**: Can load all profile types, proper error handling

- [ ] **1.4 CLI Integration**
  - [ ] Add `--profile <name>` flag to `specular auto`
  - [ ] Add `--list-profiles` flag to show available profiles
  - [ ] Merge profile config with CLI flags (CLI flags take precedence)
  - **Files**: `internal/cmd/auto.go`
  - **Tests**: `internal/cmd/auto_test.go`
  - **Acceptance**: Profiles work in interactive and CI modes

- [ ] **1.5 Default Profiles**
  - [ ] Create example `auto.profiles.yaml` with default, ci, and custom profiles
  - [ ] Add profile documentation
  - **Files**: `examples/auto.profiles.yaml`, `docs/profiles.md`
  - **Acceptance**: Users can copy and customize example profiles

- [ ] **1.6 Profile Tests**
  - [ ] Unit tests for profile loading and validation
  - [ ] Integration tests for profile-based execution
  - [ ] Test profile override behavior
  - **Files**: `internal/profiles/*_test.go`
  - **Acceptance**: 90%+ test coverage, all edge cases covered

---

### 2. Structured Action Plan Format

**Effort**: 1 week | **Priority**: CRITICAL | **Dependencies**: None

#### Implementation Tasks

- [ ] **2.1 Action Plan Schema**
  - [ ] Define ActionPlan and ActionStep structures
    ```go
    type ActionPlan struct {
        Schema   string       `json:"schema"`  // "specular.auto.plan/v1"
        Goal     string       `json:"goal"`
        Steps    []ActionStep `json:"steps"`
        Metadata PlanMetadata `json:"metadata"`
    }

    type ActionStep struct {
        ID               string            `json:"id"`
        Type             string            `json:"type"` // spec:update, spec:lock, plan:gen, build:run
        Description      string            `json:"description"`
        RequiresApproval bool              `json:"requiresApproval"`
        Reason           string            `json:"reason,omitempty"`
        Signals          map[string]string `json:"signals,omitempty"`
        Dependencies     []string          `json:"dependencies,omitempty"`
    }

    type PlanMetadata struct {
        CreatedAt time.Time `json:"createdAt"`
        Version   string    `json:"version"`
        Profile   string    `json:"profile,omitempty"`
    }
    ```
  - **Files**: `internal/auto/action_plan.go`
  - **Acceptance**: Schema matches spec requirements

- [ ] **2.2 Plan Generation Updates**
  - [ ] Update `generatePlan()` to produce ActionPlan format
  - [ ] Assign step types: spec:update, spec:lock, plan:gen, build:run
  - [ ] Mark critical steps for approval (spec:lock, build:run)
  - [ ] Add signals for routing hints
  - **Files**: `internal/auto/orchestrator.go`
  - **Tests**: `internal/auto/orchestrator_test.go`
  - **Acceptance**: Generated plans match ActionPlan schema

- [ ] **2.3 Plan Execution Updates**
  - [ ] Update executor to handle ActionPlan format
  - [ ] Implement per-step approval checks
  - [ ] Track step execution status
  - **Files**: `internal/auto/executor.go`
  - **Tests**: `internal/auto/executor_test.go`
  - **Acceptance**: Executor properly handles new plan format

- [ ] **2.4 Plan Serialization**
  - [ ] Save ActionPlan to `plan.json` with schema version
  - [ ] Support loading ActionPlan from checkpoint
  - **Files**: `internal/auto/orchestrator.go`
  - **Acceptance**: Plans are properly serialized and can be resumed

- [ ] **2.5 Plan Validation**
  - [ ] Validate plan structure and dependencies
  - [ ] Detect circular dependencies
  - [ ] Validate step types
  - **Files**: `internal/auto/plan_validator.go`
  - **Tests**: `internal/auto/plan_validator_test.go`
  - **Acceptance**: Invalid plans are rejected with clear errors

---

### 3. Exit Codes (0-6)

**Effort**: 1 day | **Priority**: HIGH | **Dependencies**: None

#### Implementation Tasks

- [ ] **3.1 Define Exit Codes**
  - [ ] Create exit code constants
    ```go
    const (
        ExitSuccess         = 0 // Success
        ExitGenericError    = 1 // Generic error
        ExitUsageError      = 2 // Invalid CLI usage
        ExitPolicyViolation = 3 // Policy check failed
        ExitDriftDetected   = 4 // Spec drift detected
        ExitAuthError       = 5 // Authentication/permission failure
        ExitNetworkError    = 6 // Network or service unavailable
    )
    ```
  - **Files**: `internal/cmd/exit_codes.go`
  - **Acceptance**: All exit codes defined and documented

- [ ] **3.2 Error Classification**
  - [ ] Update error handling to classify errors by exit code
  - [ ] Map internal errors to appropriate exit codes
  - **Files**: `internal/cmd/auto.go`
  - **Acceptance**: All error types properly mapped

- [ ] **3.3 Exit Code Usage**
  - [ ] Return appropriate exit codes from `runAuto()`
  - [ ] Add exit code to error messages
  - **Files**: `internal/cmd/auto.go`
  - **Acceptance**: CLI returns correct exit codes for all scenarios

- [ ] **3.4 Documentation**
  - [ ] Document exit codes in CLI help
  - [ ] Add exit codes to error messages
  - **Files**: `internal/cmd/auto.go`, `README.md`
  - **Acceptance**: Users can understand exit codes from documentation

- [ ] **3.5 Tests**
  - [ ] Test each exit code scenario
  - [ ] Verify exit codes in integration tests
  - **Files**: `internal/cmd/auto_test.go`, `test/e2e/auto_test.go`
  - **Acceptance**: All exit codes tested and working correctly

---

### 4. Per-Step Policy Checks

**Effort**: 1 week | **Priority**: HIGH | **Dependencies**: Action Plan Format

#### Implementation Tasks

- [ ] **4.1 Policy Check Interface**
  - [ ] Define policy check interface
    ```go
    type PolicyChecker interface {
        CheckStep(ctx context.Context, step ActionStep) (*PolicyResult, error)
    }

    type PolicyResult struct {
        Allowed  bool
        Reason   string
        Warnings []string
    }
    ```
  - **Files**: `internal/policy/checker.go`
  - **Acceptance**: Interface supports extensible policy checks

- [ ] **4.2 Built-in Policy Checks**
  - [ ] Implement cost limit check
  - [ ] Implement timeout check
  - [ ] Implement step type whitelist/blacklist
  - [ ] Implement agent restriction check
  - **Files**: `internal/policy/builtin.go`
  - **Tests**: `internal/policy/builtin_test.go`
  - **Acceptance**: All built-in checks working correctly

- [ ] **4.3 Profile-Based Policies**
  - [ ] Integrate policy checks with profile system
  - [ ] Support profile-specific policy rules
  - **Files**: `internal/policy/profile_policy.go`
  - **Acceptance**: Policies can be configured per profile

- [ ] **4.4 Executor Integration**
  - [ ] Add policy check before each step execution
  - [ ] Handle policy violations (abort or skip step)
  - [ ] Log policy check results
  - **Files**: `internal/auto/executor.go`
  - **Acceptance**: Policy checks run before each step

- [ ] **4.5 Policy Violation Handling**
  - [ ] Return ExitPolicyViolation (3) on policy failure
  - [ ] Provide clear error messages for violations
  - [ ] Support override flag for non-critical policies
  - **Files**: `internal/cmd/auto.go`
  - **Acceptance**: Policy violations properly handled and reported

- [ ] **4.6 Tests**
  - [ ] Unit tests for policy checks
  - [ ] Integration tests for policy enforcement
  - [ ] Test policy override behavior
  - **Files**: `internal/policy/*_test.go`, `test/e2e/policy_test.go`
  - **Acceptance**: 90%+ test coverage, all scenarios covered

---

### 5. JSON Output Format

**Effort**: 3 days | **Priority**: HIGH | **Dependencies**: Action Plan Format

#### Implementation Tasks

- [ ] **5.1 Output Schema**
  - [ ] Define JSON output structure
    ```go
    type AutoOutput struct {
        Schema    string            `json:"schema"` // "specular.auto.output/v1"
        Goal      string            `json:"goal"`
        Status    string            `json:"status"` // completed, failed, partial
        Steps     []StepResult      `json:"steps"`
        Artifacts []ArtifactInfo    `json:"artifacts"`
        Metrics   ExecutionMetrics  `json:"metrics"`
        Audit     AuditTrail        `json:"audit"`
    }

    type StepResult struct {
        ID          string    `json:"id"`
        Type        string    `json:"type"`
        Status      string    `json:"status"`
        StartedAt   time.Time `json:"startedAt"`
        CompletedAt time.Time `json:"completedAt"`
        Error       string    `json:"error,omitempty"`
    }

    type ExecutionMetrics struct {
        TotalDuration time.Duration `json:"totalDuration"`
        TotalCost     float64       `json:"totalCost"`
        StepsExecuted int           `json:"stepsExecuted"`
        StepsFailed   int           `json:"stepsFailed"`
    }

    type AuditTrail struct {
        CheckpointID string          `json:"checkpointId"`
        Profile      string          `json:"profile"`
        Approvals    []ApprovalEvent `json:"approvals"`
        Policies     []PolicyEvent   `json:"policies"`
    }
    ```
  - **Files**: `internal/auto/output.go`
  - **Acceptance**: Schema covers all required information

- [ ] **5.2 Output Generation**
  - [ ] Collect step results during execution
  - [ ] Collect metrics and audit events
  - [ ] Generate AutoOutput structure
  - **Files**: `internal/auto/orchestrator.go`
  - **Acceptance**: Output properly populated from execution

- [ ] **5.3 CLI Integration**
  - [ ] Add `--output json` flag
  - [ ] Write JSON output to stdout or file
  - [ ] Ensure JSON is valid and parseable
  - **Files**: `internal/cmd/auto.go`
  - **Acceptance**: JSON output flag working correctly

- [ ] **5.4 Documentation**
  - [ ] Document JSON schema
  - [ ] Provide example JSON output
  - **Files**: `docs/auto-json-output.md`, `examples/auto-output.json`
  - **Acceptance**: Schema documented with examples

- [ ] **5.5 Tests**
  - [ ] Test JSON serialization
  - [ ] Test output with different execution scenarios
  - [ ] Validate JSON schema compliance
  - **Files**: `internal/auto/output_test.go`
  - **Acceptance**: JSON output tested for all scenarios

---

## Phase 3: v1.5.0 - Enhanced UX Features (4 weeks)

### Priority 2: Should Have (Enhances UX)

---

### 6. Scope Filtering (`--scope`)

**Effort**: 3 days | **Priority**: MEDIUM | **Dependencies**: Action Plan Format

#### Implementation Tasks

- [ ] **6.1 Scope Parser**
  - [ ] Parse scope patterns (paths, features)
  - [ ] Support glob patterns (e.g., `src/components/**`)
  - [ ] Support feature tags
  - **Files**: `internal/auto/scope.go`
  - **Tests**: `internal/auto/scope_test.go`
  - **Acceptance**: Scope patterns parsed correctly

- [ ] **6.2 Plan Filtering**
  - [ ] Filter action plan steps based on scope
  - [ ] Preserve step dependencies
  - [ ] Update step count and estimates
  - **Files**: `internal/auto/plan_filter.go`
  - **Acceptance**: Plans correctly filtered by scope

- [ ] **6.3 CLI Integration**
  - [ ] Add `--scope <pattern>` flag
  - [ ] Support multiple scope patterns
  - **Files**: `internal/cmd/auto.go`
  - **Acceptance**: Scope filtering works from CLI

- [ ] **6.4 Tests**
  - [ ] Test scope parsing
  - [ ] Test plan filtering
  - [ ] Test scope with dependencies
  - **Files**: `internal/auto/*_test.go`
  - **Acceptance**: Scope filtering fully tested

---

### 7. Max Steps Limit (`--max-steps`)

**Effort**: 1 day | **Priority**: MEDIUM | **Dependencies**: Action Plan Format

#### Implementation Tasks

- [ ] **7.1 Step Counter**
  - [ ] Track executed steps
  - [ ] Check against max_steps limit
  - **Files**: `internal/auto/executor.go`
  - **Acceptance**: Steps properly counted

- [ ] **7.2 Limit Enforcement**
  - [ ] Abort execution when limit reached
  - [ ] Save partial results
  - [ ] Return appropriate exit code
  - **Files**: `internal/auto/executor.go`
  - **Acceptance**: Limit enforced correctly

- [ ] **7.3 CLI Integration**
  - [ ] Add `--max-steps <n>` flag
  - [ ] Integrate with profile system
  - **Files**: `internal/cmd/auto.go`
  - **Acceptance**: Max steps flag working

- [ ] **7.4 Tests**
  - [ ] Test limit enforcement
  - [ ] Test partial completion
  - **Files**: `internal/auto/executor_test.go`
  - **Acceptance**: Max steps fully tested

---

### 8. Interactive TUI

**Effort**: 2 weeks | **Priority**: MEDIUM | **Dependencies**: Action Plan Format

#### Implementation Tasks

- [ ] **8.1 TUI Framework Setup**
  - [ ] Add Bubble Tea dependency
  - [ ] Create TUI model structure
  - **Files**: `internal/tui/model.go`
  - **Acceptance**: TUI framework initialized

- [ ] **8.2 Main View**
  - [ ] Display goal and current step
  - [ ] Show progress (X/Y steps)
  - [ ] Display current cost
  - **Files**: `internal/tui/views/main.go`
  - **Acceptance**: Main view displays correctly

- [ ] **8.3 Step List View**
  - [ ] Show all steps with status
  - [ ] Highlight current step
  - [ ] Show step types and approval status
  - **Files**: `internal/tui/views/steps.go`
  - **Acceptance**: Step list working

- [ ] **8.4 Hotkeys**
  - [ ] Implement `?` for help
  - [ ] Implement `v` for verbose mode
  - [ ] Implement `s` for step list
  - [ ] Implement `a` for approval
  - [ ] Implement `q` for quit
  - **Files**: `internal/tui/keys.go`
  - **Acceptance**: All hotkeys working

- [ ] **8.5 Approval Flow**
  - [ ] Show approval prompt
  - [ ] Display step details
  - [ ] Handle approve/reject
  - **Files**: `internal/tui/views/approval.go`
  - **Acceptance**: Approval flow working

- [ ] **8.6 CLI Integration**
  - [ ] Add `--tui` flag
  - [ ] Fall back to text mode if TUI fails
  - **Files**: `internal/cmd/auto.go`
  - **Acceptance**: TUI mode toggleable

- [ ] **8.7 Tests**
  - [ ] Test TUI model state transitions
  - [ ] Test hotkey handling
  - **Files**: `internal/tui/*_test.go`
  - **Acceptance**: TUI components tested

---

### 9. Trace Logging

**Effort**: 3 days | **Priority**: MEDIUM | **Dependencies**: None

#### Implementation Tasks

- [ ] **9.1 Trace Schema**
  - [ ] Define trace event structure
  - [ ] Support different event types
  - **Files**: `internal/trace/event.go`
  - **Acceptance**: Trace schema defined

- [ ] **9.2 Trace Logger**
  - [ ] Implement trace event logger
  - [ ] Write to `~/.specular/logs/trace_<id>.json`
  - [ ] Support log rotation
  - **Files**: `internal/trace/logger.go`
  - **Tests**: `internal/trace/logger_test.go`
  - **Acceptance**: Trace logging working

- [ ] **9.3 Event Collection**
  - [ ] Log all critical events
  - [ ] Include timestamps and context
  - [ ] Log policy checks and approvals
  - **Files**: `internal/auto/orchestrator.go`
  - **Acceptance**: All events logged

- [ ] **9.4 CLI Integration**
  - [ ] Add `--trace` flag
  - [ ] Show trace file location
  - **Files**: `internal/cmd/auto.go`
  - **Acceptance**: Trace flag working

- [ ] **9.5 Tests**
  - [ ] Test trace logging
  - [ ] Test event structure
  - **Files**: `internal/trace/*_test.go`
  - **Acceptance**: Trace logging tested

---

### 10. Patch Generation

**Effort**: 1 week | **Priority**: MEDIUM | **Dependencies**: Action Plan Format

#### Implementation Tasks

- [ ] **10.1 Diff Generation**
  - [ ] Capture file changes per step
  - [ ] Generate unified diff format
  - **Files**: `internal/patch/diff.go`
  - **Acceptance**: Diffs generated correctly

- [ ] **10.2 Patch Files**
  - [ ] Write `.patch` files per step
  - [ ] Include metadata (step ID, timestamp)
  - **Files**: `internal/patch/writer.go`
  - **Acceptance**: Patch files created

- [ ] **10.3 Rollback Support**
  - [ ] Apply patches in reverse
  - [ ] Handle conflicts
  - **Files**: `internal/patch/rollback.go`
  - **Acceptance**: Rollback working

- [ ] **10.4 CLI Integration**
  - [ ] Add `--save-patches` flag
  - [ ] Add `specular auto rollback` command
  - **Files**: `internal/cmd/auto.go`, `internal/cmd/auto_rollback.go`
  - **Acceptance**: Patch generation toggleable

- [ ] **10.5 Tests**
  - [ ] Test diff generation
  - [ ] Test patch application
  - [ ] Test rollback
  - **Files**: `internal/patch/*_test.go`
  - **Acceptance**: Patch system tested

---

## Phase 4: v1.6.0 - Enterprise Features (6 weeks)

### Priority 3: Nice to Have (Enterprise/Advanced)

---

### 11. Attestation (Sigstore)

**Effort**: 2 weeks | **Priority**: LOW | **Dependencies**: JSON Output

#### Implementation Tasks

- [ ] **11.1 Sigstore Integration**
  - [ ] Add sigstore-go dependency
  - [ ] Implement signing interface
  - **Files**: `internal/attestation/signer.go`
  - **Acceptance**: Sigstore integration working

- [ ] **11.2 Attestation Generation**
  - [ ] Sign plan and output
  - [ ] Include provenance data
  - **Files**: `internal/attestation/generate.go`
  - **Acceptance**: Attestations generated

- [ ] **11.3 Verification**
  - [ ] Verify attestation signatures
  - [ ] Validate provenance
  - **Files**: `internal/attestation/verify.go`
  - **Acceptance**: Verification working

- [ ] **11.4 CLI Integration**
  - [ ] Add `--attest` flag
  - [ ] Add verification command
  - **Files**: `internal/cmd/auto.go`
  - **Acceptance**: Attestation toggleable

- [ ] **11.5 Tests**
  - [ ] Test signing
  - [ ] Test verification
  - **Files**: `internal/attestation/*_test.go`
  - **Acceptance**: Attestation system tested

---

### 12. Explain Routing (`specular explain`)

**Effort**: 1 week | **Priority**: LOW | **Dependencies**: Profile System

#### Implementation Tasks

- [ ] **12.1 Routing Explainer**
  - [ ] Analyze plan routing decisions
  - [ ] Show agent selection rationale
  - **Files**: `internal/explain/routing.go`
  - **Acceptance**: Routing explanation working

- [ ] **12.2 CLI Command**
  - [ ] Add `specular explain <checkpoint-id>` command
  - [ ] Format explanation output
  - **Files**: `internal/cmd/explain.go`
  - **Acceptance**: Explain command working

- [ ] **12.3 Tests**
  - [ ] Test routing analysis
  - [ ] Test output formatting
  - **Files**: `internal/explain/*_test.go`
  - **Acceptance**: Explain command tested

---

### 13. Hooks System ✅

**Effort**: 2 weeks | **Priority**: LOW | **Dependencies**: Action Plan Format

#### Implementation Tasks

- [x] **13.1 Hook Interface** ✅
  - [x] Define hook types (on_plan_created, on_step_before, etc.)
  - [x] Create hook executor interface
  - **Files**: `internal/hooks/hooks.go` (183 lines)
  - **Acceptance**: Hook interface defined with 11 event types

- [x] **13.2 Hook Registry** ✅
  - [x] Register and manage hooks
  - [x] Support multiple hooks per event
  - **Files**: `internal/hooks/registry.go` (181 lines)
  - **Acceptance**: Hook registry working with thread-safe operations

- [x] **13.3 Hook Execution** ✅
  - [x] Execute hooks at appropriate points
  - [x] Handle hook failures
  - **Files**: `internal/hooks/executor.go` (169 lines)
  - **Acceptance**: Hooks execute correctly with concurrency control (max 10 concurrent)

- [x] **13.4 Built-in Hooks** ✅
  - [x] Slack notification hook
  - [x] Webhook hook
  - [x] Script hook (bonus)
  - **Files**: `internal/hooks/builtin.go` (318 lines)
  - **Acceptance**: Built-in hooks working (script, webhook, slack)

- [x] **13.5 Configuration** ✅
  - [x] Add hooks to profile configuration
  - [x] Document hook configuration
  - **Files**: `internal/profiles/profile.go` (HooksConfig), `README.md` (lines 2515-3057, 544 lines)
  - **Acceptance**: Hooks configurable via profiles, comprehensive documentation added

- [x] **13.6 Tests** ✅
  - [x] Test hook execution
  - [x] Test hook failures
  - **Files**: `internal/hooks/*_test.go` (32 tests)
  - **Acceptance**: Hook system fully tested and passing

---

### 14. Advanced Security ✅

**Effort**: 1 week | **Priority**: LOW | **Dependencies**: Profile System, Policy Checks

#### Implementation Tasks

- [x] **14.1 Credential Management** ✅
  - [x] Secure credential storage
  - [x] Credential rotation support
  - **Files**: `internal/security/credentials.go` (344 lines)
  - **Acceptance**: Credentials managed securely with AES-GCM encryption, PBKDF2, rotation policies

- [x] **14.2 Audit Logging** ✅
  - [x] Enhanced audit logging
  - [x] Compliance reporting
  - **Files**: `internal/security/audit.go` (300+ lines)
  - **Acceptance**: Comprehensive audit logging with daily rotation, querying, JSON format

- [x] **14.3 Secret Scanning** ✅
  - [x] Scan for secrets in code
  - [x] Block commits with secrets
  - **Files**: `internal/security/secrets.go` (382 lines)
  - **Acceptance**: Secret scanning working with 10 secret types, git diff scanning, pre-commit integration

- [x] **14.4 Tests** ✅
  - [x] Test credential management
  - [x] Test audit logging
  - [x] Test secret scanning
  - **Files**: `internal/security/*_test.go` (47 tests)
  - **Acceptance**: All security features fully tested and passing

- [x] **14.5 Documentation** ✅
  - [x] Comprehensive documentation added to README.md
  - **Files**: `README.md` (lines 3059-3813, 758 lines)
  - **Acceptance**: Complete documentation with examples, use cases, best practices, troubleshooting

---

## Testing Strategy

### Unit Tests
- **Coverage Target**: 90%+ for all new packages
- **Focus**: Individual component behavior, edge cases, error handling
- **Tools**: Go testing package, testify for assertions

### Integration Tests
- **Scope**: Multi-component interactions, profile system, policy checks
- **Focus**: Component integration, data flow, configuration
- **Location**: `test/integration/`

### End-to-End Tests
- **Scope**: Full autonomous mode workflows, CLI interactions
- **Focus**: User scenarios, complete workflows, error recovery
- **Location**: `test/e2e/`

### Performance Tests
- **Metrics**: Execution time, memory usage, cost tracking
- **Benchmarks**: Plan generation, step execution, checkpoint operations
- **Location**: `test/performance/`

---

## Dependencies Between Features

```
Profile System (1)
├── Per-Step Policy Checks (4)
├── Explain Routing (12)
└── Hooks System (13)

Action Plan Format (2)
├── Per-Step Policy Checks (4)
├── JSON Output Format (5)
├── Scope Filtering (6)
├── Max Steps Limit (7)
├── Interactive TUI (8)
├── Patch Generation (10)
└── Hooks System (13)

Exit Codes (3)
└── (No dependencies)

JSON Output Format (5)
└── Attestation (11)

Trace Logging (9)
└── (No dependencies)

Advanced Security (14)
├── Profile System (1)
└── Per-Step Policy Checks (4)
```

---

## Timeline Summary

| Phase | Version | Duration | Features | Priority |
|-------|---------|----------|----------|----------|
| Phase 2 | v1.4.0 | 6 weeks | 1-5 | Must Have |
| Phase 3 | v1.5.0 | 4 weeks | 6-10 | Should Have |
| Phase 4 | v1.6.0 | 6 weeks | 11-14 | Nice to Have |
| **Total** | - | **16 weeks** | **14 features** | - |

---

## Success Criteria

### v1.4.0 (Production-Ready)
- ✅ All Priority 1 features implemented
- ✅ 90%+ test coverage
- ✅ Profile system working in CI and interactive modes
- ✅ Policy checks prevent unsafe operations
- ✅ JSON output enables automation
- ✅ Exit codes support CI/CD integration

### v1.5.0 (Enhanced UX)
- ✅ All Priority 2 features implemented
- ✅ Interactive TUI improves developer experience
- ✅ Trace logging enables debugging
- ✅ Scope filtering enables targeted execution
- ✅ Patch generation enables rollback

### v1.6.0 (Enterprise-Ready)
- ✅ All Priority 3 features implemented
- ✅ Attestation provides cryptographic proof
- ✅ Hooks enable workflow customization
- ✅ Advanced security meets compliance requirements
- ✅ Explain command improves transparency

---

## Implementation Notes

### Code Quality
- Follow existing code patterns in `internal/auto/`
- Use structured logging with context
- Implement proper error handling with error wrapping
- Add comprehensive unit tests for all new code
- Document all public APIs with godoc comments

### Configuration Management
- Extend profile system for new features
- Support both CLI flags and config file settings
- CLI flags should override config file settings
- Provide sensible defaults for all settings

### Backward Compatibility
- Maintain compatibility with v1.3.0 checkpoints
- Support old plan format during transition
- Deprecate old features gracefully
- Document breaking changes in CHANGELOG

### Performance
- Monitor checkpoint save/load performance
- Optimize plan generation for large projects
- Cache expensive operations where possible
- Profile memory usage for long-running workflows

### Security
- Validate all user inputs
- Sanitize file paths and command arguments
- Implement proper access controls
- Log security-relevant events

---

## Next Steps

1. **Review and Prioritize**: Review this TODO list with the team and adjust priorities
2. **Assign Owners**: Assign feature owners for each major feature
3. **Create Issues**: Create GitHub issues for each feature with this TODO as a template
4. **Start Implementation**: Begin with Profile System (Feature #1) as it unlocks other features
5. **Iterate**: Review progress weekly and adjust timeline as needed

---

**Last Updated**: 2025-11-11
**Status**: Phase 2 (v1.4.0) Planning Complete
**Next Milestone**: Feature #1 (Profile System) - 2 weeks
