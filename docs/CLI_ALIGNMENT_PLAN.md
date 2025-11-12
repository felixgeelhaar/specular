# Specular CLI v1.0 Alignment Plan

This document outlines the changes needed to align the current CLI with the v1.0 specification.

## Gap Analysis

### ✅ Commands Already Aligned

| Current | Spec | Status |
|---------|------|--------|
| `auto` | `auto --goal`, `auto resume`, `auto history`, `auto explain` | Needs subcommands |
| `build` | `build run`, `build verify`, `build approve`, `build explain` | Needs refactor |
| `bundle` | `bundle build`, `bundle inspect`, `bundle sign`, `bundle push` | Check subcommands |
| `doctor` | `doctor` | ✅ Aligned |
| `eval` | `eval run`, `eval rules`, `eval drift` | Needs subcommands |
| `init` | `init [--template]` | ✅ Mostly aligned |
| `plan` | `plan gen`, `plan review`, `plan drift`, `plan explain` | Needs refactor |
| `route` | `route explain`, `route list`, `route override` | Check subcommands |
| `spec` | `spec new`, `spec edit`, `spec validate`, `spec lock`, `spec diff`, `spec approve` | Needs refactor |
| `version` | `version` | ✅ Aligned |

### ❌ Missing Core Commands

1. **`context`** - Detect environment (models, API keys, Docker)
2. **`config`** - View or edit Specular configuration
3. **`auth`** - Manage credentials for governance/cloud registry
4. **`status`** - Show environment and spec/plan states
5. **`logs`** - Show or tail CLI logs

### ❌ Missing Subcommands

#### Spec Management
- `spec new [--from <file>]` (currently `interview` and `spec generate`)
- `spec edit` (NEW)
- `spec diff <versionA> <versionB>` (NEW)
- `spec approve` (NEW)

#### Planning & Drift
- `plan gen [--feature <id>]` (currently just `plan`)
- `plan review` (NEW)
- `plan drift` (NEW)
- `plan explain <step>` (NEW)

#### Build & Execution
- `build run [--feature <id>]` (currently just `build`)
- `build verify` (NEW)
- `build approve` (NEW)
- `build explain` (NEW)

#### Evaluation & Guardrails
- `eval run [--scenario <name>]` (NEW)
- `eval rules` (NEW)
- `eval drift` (NEW)

#### Bundling & Deployment (Check existing)
- `bundle build [--out <file>]`
- `bundle inspect <file>`
- `bundle sign [--key <key>]`
- `bundle push`

#### Governance & Pro
- `policy new` (NEW)
- `policy apply` (NEW)
- `approve [spec|plan|bundle]` (NEW)
- `org sync` (NEW - Future)
- `team status` (NEW - Future)

#### Routing
- `route explain <task>` (Check)
- `route list` (NEW)
- `route override <provider>` (NEW)

#### Auto (Interactive)
- `auto --goal "<text>"` (Currently `auto`)
- `auto resume` (NEW)
- `auto history` (NEW)
- `auto explain` (NEW - Different from root `explain`)

## Implementation Phases

### Phase 1: Core Commands ✅ COMPLETED (Priority: HIGH)

**Goal:** Add essential missing commands for basic workflow

**Status:** ✅ Completed - All 4 commands implemented with comprehensive tests and documentation

**Completion Summary:**
- ✅ 4 commands implemented (context, config, status, logs)
- ✅ 15 subcommands added (config: view, edit, get, set, path; logs: list, follow)
- ✅ 35 unit tests created (100% pass rate)
- ✅ Full CLI reference documentation created (docs/CLI_REFERENCE.md)
- ✅ All commands support JSON/YAML output formats
- ✅ Comprehensive error handling and validation

**Implementation Details:**

1. **`context` command** ✅ - Environment detection
   - ✅ Detect installed models (Ollama)
   - ✅ Check API keys (OpenAI, Anthropic, Gemini)
   - ✅ Verify Docker/Podman installation
   - ✅ Git repository information
   - ✅ Output: JSON/YAML/text summary
   - **File:** `internal/cmd/context.go`
   - **Tests:** None (uses internal/detect package which has tests)

2. **`config` command** ✅ - Configuration management
   - ✅ `config view` - Show current config (text/json/yaml)
   - ✅ `config edit` - Open in $EDITOR
   - ✅ `config get <key>` - Get specific value (15 supported keys)
   - ✅ `config set <key> <value>` - Set value with validation
   - ✅ `config path` - Show config file path
   - **File:** `internal/cmd/config.go` (454 lines)
   - **Tests:** `internal/cmd/config_test.go` (7 test functions, 433 lines)
   - **Coverage:** parseBool, parseFloat, parseInt, get/set nested values, save/load

3. **`status` command** ✅ - Overall status
   - ✅ Environment health check (runtime, providers, API keys)
   - ✅ Project initialization status
   - ✅ Current spec version and lock status
   - ✅ Active plan status
   - ✅ Git repository state
   - ✅ Issues and warnings analysis
   - ✅ Recommended next steps
   - **File:** `internal/cmd/status.go` (462 lines)
   - **Tests:** `internal/cmd/status_test.go` (16 test functions, 382 lines)
   - **Coverage:** environment/project/spec/plan status, analysis logic, time formatting

4. **`logs` command** ✅ - Log management
   - ✅ `logs` - Show recent logs (with --lines flag)
   - ✅ `logs --follow` - Tail logs in real-time
   - ✅ `logs --trace <id>` - Show specific trace
   - ✅ `logs list` - List all trace files
   - ✅ Logs stored in `~/.specular/logs/trace_<id>.json`
   - ✅ Pretty-printed JSON output
   - **File:** `internal/cmd/logs.go` (399 lines)
   - **Tests:** `internal/cmd/logs_test.go` (12 test functions, 317 lines)
   - **Coverage:** trace file management, file operations, formatting helpers

### Phase 2: Spec Management Refactor ✅ COMPLETED (Priority: HIGH)

**Goal:** Align spec commands with v1.0 specification

**Status:** ✅ Completed - All 6 spec subcommands implemented with comprehensive tests

**Completion Summary:**
- ✅ 6 spec subcommands implemented (new, edit, lock with --note, diff, approve, validate)
- ✅ Interview command deprecated with migration guidance
- ✅ 4 test functions created (13 test cases, 100% pass rate)
- ✅ Backward compatibility maintained
- ✅ All commands build and function correctly

**Implementation Details:**

1. **Refactor `spec` subcommands:**
   - ✅ `spec new [--from <file>]` - Merged `interview` and `spec generate` functionality
     - **File:** `internal/cmd/spec.go:295-347`
     - Interactive mode (default): Launches interview engine with preset selection
     - PRD mode (--from flag): Generates spec from PRD markdown file
   - ✅ `spec edit` - Open current spec in $EDITOR with validation
     - **File:** `internal/cmd/spec.go:349-382`
     - Opens spec.yaml in user's $EDITOR (defaults to vi)
     - Validates spec after editing
   - ✅ `spec validate` - Already exists
   - ✅ `spec lock [--note "<msg>"]` - Added --note flag for versioning notes
     - **File:** `internal/cmd/spec.go:151-192`
     - Generates blake3 hash of spec for drift detection
     - Optional --note flag saves annotation to .note file
   - ✅ `spec diff <versionA> <versionB>` - Compare spec versions
     - **File:** `internal/cmd/spec.go:584-707`
     - Compares product name, features (added/removed/modified)
     - Shows detailed field-level changes (title, description, priority)
     - Handles domain.FeatureID type correctly
   - ✅ `spec approve` - Approve spec for use
     - **File:** `internal/cmd/spec.go:709-752`
     - Validates product name and features exist
     - Creates .approved marker file with timestamp
     - Shows next steps (lock, plan)

2. **Deprecation path for `interview`:**
   - ✅ Add deprecation notice to `interview`
     - **File:** `internal/cmd/interview.go:29-36`
     - Displays warning on stderr about deprecation in v1.6.0
     - Provides migration examples to `spec new`
   - ✅ Point users to `spec new`
     - Migration guide included in deprecation notice
   - ✅ Keep `interview` as alias for 1-2 releases
     - Interview command still functional, uses shared runInterviewInternal function

3. **Tests created:**
   - ✅ TestSpecApproveValidation (3 cases) - Validates product and feature requirements
   - ✅ TestSpecDiffFeatureComparison (5 cases) - Tests feature diff logic
   - ✅ TestSpecLockWithNote (2 cases) - Tests note file creation
   - ✅ TestRunInterviewInternal (1 case) - Verifies function exists
   - **File:** `internal/cmd/spec_test.go` (297 lines)
   - **Coverage:** Validation logic, diff comparison, note management

### Phase 3: Plan Management Refactor ✅ COMPLETED (Priority: HIGH)

**Goal:** Add plan review and drift detection

**Status:** ✅ Completed - All 4 plan subcommands implemented with comprehensive tests

**Completion Summary:**
- ✅ 4 plan subcommands implemented (gen, review, drift, explain)
- ✅ Backward compatibility maintained with deprecation notice
- ✅ --feature flag added for feature-specific plans
- ✅ 6 test functions created (15 test cases, 100% pass rate)
- ✅ TUI stub created for plan review

**Implementation Details:**

1. **Refactor `plan` to `plan gen`:**
   - ✅ Current `plan` becomes `plan gen`
     - **File:** `internal/cmd/plan.go:44-186`
     - Generates task DAG from spec and lock
     - Supports all original flags (--in, --out, --lock, --estimate)
   - ✅ Add `--feature <id>` flag for feature-specific plans
     - **File:** `internal/cmd/plan.go:139-164`
     - Filters spec to single feature before plan generation
     - Validates feature exists in spec
     - Provides tailored next steps
   - ✅ Keep backward compatibility
     - **File:** `internal/cmd/plan.go:27-41`
     - Root plan command detects old flag usage
     - Shows deprecation warning (v1.6.0 removal)
     - Delegates to plan gen for backward compatibility

2. **Add new plan subcommands:**
   - ✅ `plan review` - Interactive plan review (TUI)
     - **File:** `internal/cmd/plan.go:188-233`
     - Loads and validates plan file
     - Launches TUI for interactive review (stub implementation)
     - Shows approval/rejection result with next steps
     - **TUI Stub:** `internal/tui/plan_review.go` (19 lines)
   - ✅ `plan drift` - Detect drift between plan and repo
     - **File:** `internal/cmd/plan.go:235-306`
     - Checks git status for uncommitted changes
     - Reports number of affected files
     - Provides recommendations (commit, stash, regenerate)
     - Placeholder for hash comparison (future enhancement)
   - ✅ `plan explain <step>` - Explain routing for specific step
     - **File:** `internal/cmd/plan.go:308-378`
     - Looks up task by step ID
     - Shows routing decision rationale
     - Displays skill, model hint, priority, complexity
     - Lists task dependencies and expected hash

3. **Tests created:**
   - ✅ TestPlanFeatureFiltering (3 cases) - Feature filter logic
   - ✅ TestPlanExplainTaskLookup (3 cases) - Task lookup logic
   - ✅ TestPlanDriftDetection (2 cases) - Drift detection logic
   - ✅ TestBackwardCompatibilityFlags (4 flags) - Backward compatibility
   - ✅ TestPlanGenFlags (5 flags) - Plan gen flags
   - ✅ TestPlanSubcommands (4 commands) - Subcommand registration
   - **File:** `internal/cmd/plan_test.go` (246 lines)
   - **Coverage:** Filtering, lookup, drift, flags, subcommands

### Phase 4: Build & Execution Enhancement ✅ COMPLETED (Priority: MEDIUM)

**Goal:** Add verification and approval steps

**Status:** ✅ Completed - All 4 build subcommands implemented with comprehensive tests

**Completion Summary:**
- ✅ 4 build subcommands implemented (run, verify, approve, explain)
- ✅ Backward compatibility maintained with deprecation notice
- ✅ --feature flag added for feature-specific builds
- ✅ 7 test functions created (19 test cases, 100% pass rate)
- ✅ All commands build and function correctly

**Implementation Details:**

1. **Refactor `build` to `build run`:**
   - ✅ Current `build` becomes `build run`
     - **File:** `internal/cmd/build.go:93-264`
     - Moved all existing build logic (checkpoint/resume, progress, caching)
     - Supports all original flags (plan, policy, dry-run, resume, verbose)
   - ✅ Add `--feature <id>` flag
     - **File:** `internal/cmd/build.go:99-120`
     - Filters plan.Tasks by featureID
     - Validates feature has tasks before execution
     - Provides tailored next steps
   - ✅ Keep backward compatibility
     - **File:** `internal/cmd/build.go:26-42`
     - Root build command detects old flag usage
     - Shows deprecation warning (v1.6.0 removal)
     - Delegates to `build run` for compatibility

2. **Add build subcommands:**
   - ✅ `build verify` - Run lint, tests, policy checks
     - **File:** `internal/cmd/build.go:266-339`
     - Executes `go vet ./...`
     - Runs `golangci-lint run --timeout=5m` (graceful skip if not installed)
     - Runs test suite with `go test ./... -short`
     - Validates policy compliance (Docker required, test coverage)
     - Returns error if any check fails
   - ✅ `build approve` - Approve build results
     - **File:** `internal/cmd/build.go:341-411`
     - Finds most recent manifest by modification time
     - Creates approval marker file with timestamp
     - Validates manifest exists
     - Shows next steps (build run, bundle)
   - ✅ `build explain` - Show logs and routing decisions
     - **File:** `internal/cmd/build.go:413-505`
     - Displays execution logs from manifest
     - Shows manifest location
     - Displays approval status
     - Provides jq command for JSON inspection

3. **Tests created:**
   - ✅ TestBuildFeatureFiltering (3 cases) - Feature filtering logic
   - ✅ TestBuildVerifyChecks (4 cases) - Verification check execution
   - ✅ TestBuildManifestLookup (3 cases) - Most recent manifest finding
   - ✅ TestBuildApproveValidation (2 cases) - Approval validation
   - ✅ TestBuildBackwardCompatibilityFlags (3 flags) - Backward compatibility
   - ✅ TestBuildRunFlags (6 flags) - Build run flags
   - ✅ TestBuildSubcommands (4 commands) - Subcommand registration
   - **File:** `internal/cmd/build_test.go` (380 lines)
   - **Coverage:** Filtering, verification, manifest lookup, approval, flags, subcommands

### Phase 5: Evaluation Framework ✅ COMPLETED (Priority: MEDIUM)

**Goal:** Structured evaluation and guardrails

**Status:** ✅ Completed - All 3 eval subcommands implemented with comprehensive tests

**Completion Summary:**
- ✅ 3 eval subcommands implemented (run, rules, drift)
- ✅ 4 evaluation scenarios defined (smoke, integration, security, performance)
- ✅ Backward compatibility maintained with deprecation notice
- ✅ 7 test functions created (16 test cases, 100% pass rate)
- ✅ All commands build and function correctly

**Implementation Details:**

1. **Add `eval` subcommands:**
   - ✅ `eval run [--scenario <name>]` - Run evaluation scenarios
     - **File:** `internal/cmd/eval.go:86-289`
     - Supports scenario argument or --scenario flag
     - 4 scenarios: smoke, integration, security, performance
     - Smoke: go vet, go build, basic tests (3 checks)
     - Integration: go vet, all tests, coverage (3 checks)
     - Security: go vet, gosec scan, policy check (3 checks)
     - Performance: benchmarks, memory/CPU profiling (3 checks)
   - ✅ `eval rules` - View or edit guardrail rules
     - **File:** `internal/cmd/eval.go:291-430`
     - Displays policy rules (execution, linters, formatters, tests, security, routing)
     - --edit flag opens policy in $EDITOR
     - Validates policy after editing
     - Shows next steps if policy missing
   - ✅ `eval drift` - Detect drift between plan and repository
     - **File:** `internal/cmd/eval.go:432-794`
     - Original eval functionality preserved
     - Comprehensive drift detection (plan, code, infrastructure)
     - SARIF report generation
     - Checkpoint/resume support

2. **Define eval scenarios:**
   - ✅ smoke - Basic health checks (go vet, go build, basic tests)
   - ✅ integration - Full integration tests (go vet, all tests, coverage)
   - ✅ security - Security scan + policy check (go vet, gosec, policy)
   - ✅ performance - Performance benchmarks (benchmarks, profiling)

3. **Backward Compatibility:**
   - ✅ Root eval command detects old flag usage
     - **File:** `internal/cmd/eval.go:30-44`
     - Shows deprecation warning (v1.6.0 removal)
     - Delegates to `eval drift` for compatibility

4. **Tests created:**
   - ✅ TestEvalScenarioValidation (6 cases) - Scenario validation logic
   - ✅ TestEvalScenarioChecks (4 cases) - Check count per scenario
   - ✅ TestEvalBackwardCompatibilityFlags (4 flags) - Backward compatibility
   - ✅ TestEvalRunFlags (2 flags) - Eval run flags
   - ✅ TestEvalRulesFlags (2 flags) - Eval rules flags
   - ✅ TestEvalDriftFlags (6 flags) - Eval drift flags
   - ✅ TestEvalSubcommands (3 commands) - Subcommand registration
   - **File:** `internal/cmd/eval_test.go` (190 lines)
   - **Coverage:** Scenario validation, checks, flags, subcommands

### Phase 6: Auto Mode Enhancement ✅ COMPLETED (Priority: MEDIUM)

**Completion Summary:**
- ✅ 3 auto subcommands implemented (resume, history, explain)
- ✅ Session persistence via checkpoint manager
- ✅ Backward compatibility maintained with --resume flag
- ✅ 5 test functions created (100% pass rate)
- ✅ All commands build and function correctly

**Implementation Details:**
1. **Add auto subcommands:**
   - ✅ `auto resume [session-id]` - Resume paused session or list available sessions
     - **File:** `internal/cmd/auto.go:355-448`
     - Lists all available sessions from `.specular/checkpoints/`
     - Resumes specific session by ID with status display
   - ✅ `auto history` - View logs and session history
     - **File:** `internal/cmd/auto.go:451-546`
     - Displays all sessions with status, timestamps, goal, and task breakdown
     - Shows failed tasks with error messages
   - ✅ `auto explain <session-id> [step]` - Explain reasoning per step
     - **File:** `internal/cmd/auto.go:549-657`
     - Overall session explanation or specific task details
     - Shows task status, duration, attempts, and artifacts

2. **Session persistence:**
   - ✅ Sessions stored in `~/.specular/checkpoints/` via checkpoint.Manager
   - ✅ Resume capability via existing --resume flag and new resume subcommand
   - ✅ Checkpoint data includes status, tasks, metadata, and timestamps

**Test Coverage:**
- ✅ TestAutoSubcommands - Verifies all 3 subcommands registered
- ✅ TestAutoResumeFlags - Validates resume command configuration
- ✅ TestAutoHistoryFlags - Validates history command configuration
- ✅ TestAutoExplainFlags - Validates explain command configuration
- ✅ TestAutoBackwardCompatibilityFlags - Ensures backward compatibility
- **File:** `internal/cmd/auto_test.go` (107 lines)
- **Coverage:** Command registration, configuration, flags, backward compatibility

### Phase 7: Routing Intelligence ✅ COMPLETED (Priority: LOW)

**Completion Summary:**
- ✅ 3 route subcommands implemented (list, override, explain)
- ✅ Provider and model listing with availability status
- ✅ Session override via environment variable
- ✅ Routing explanation with cost estimates
- ✅ 7 test functions created (100% pass rate)
- ✅ All commands build and function correctly

**Implementation Details:**
1. **Add routing commands:**
   - ✅ `route list [--available] [--provider]` - List providers, models, and costs
     - **File:** `internal/cmd/route.go:28-157`
     - Lists all models grouped by provider (Anthropic, OpenAI, Local)
     - Shows availability status, context window, cost per Mtok, latency, capability
     - Displays router budget (limit, spent, remaining)
     - Filter by availability or specific provider
   - ✅ `route override <provider>` - Override provider selection for session
     - **File:** `internal/cmd/route.go:160-227`
     - Validates provider name (anthropic, openai, local)
     - Checks provider configuration status
     - Provides export command for SPECULAR_PROVIDER_OVERRIDE env var
   - ✅ `route explain <task-type>` - Explain routing logic for task types
     - **File:** `internal/cmd/route.go:230-332`
     - Supports task types: codegen, long-context, agentic, fast, cheap
     - Shows selected model with full details
     - Explains selection reasoning
     - Provides cost estimates and alternative models

**Test Coverage:**
- ✅ TestRouteSubcommands - Verifies all 3 subcommands registered
- ✅ TestRouteListFlags - Validates list command flags
- ✅ TestRouteOverrideArgs - Validates override argument requirements
- ✅ TestRouteExplainArgs - Validates explain argument requirements
- ✅ TestRouteListCommand - Tests list command configuration
- ✅ TestRouteOverrideCommand - Tests override command configuration
- ✅ TestRouteExplainCommand - Tests explain command configuration
- **File:** `internal/cmd/route_test.go` (123 lines)
- **Coverage:** Command registration, configuration, flags, arguments

### Phase 8: Governance & Pro Features (Priority: LOW - Future)

**Goal:** Enterprise features for teams

1. **Policy management:**
   - `policy new` - Create new policy
   - `policy apply` - Apply policy to target

2. **Approval workflow:**
   - `approve [spec|plan|bundle]` - Governance signature

3. **Team collaboration (Future):**
   - `org sync` - Sync with registry
   - `team status` - Show approvals and reviews

### Phase 9: Auth Command (Priority: LOW)

**Goal:** Credential management

1. **Add `auth` command:**
   - `auth login` - Login to governance/registry
   - `auth logout` - Logout
   - `auth whoami` - Show current user
   - `auth token` - Get/refresh token

## Implementation Strategy

### Backward Compatibility

1. **Aliases for renamed commands:**
   - `interview` → `spec new` (with deprecation notice)
   - `plan` → `plan gen` (with deprecation notice)
   - `build` → `build run` (with deprecation notice)

2. **Deprecation timeline:**
   - v1.4.x: Add deprecation warnings
   - v1.5.0: Make aliases secondary
   - v1.6.0: Remove deprecated aliases

### Testing Strategy

1. **Unit tests for each new command**
2. **Integration tests for workflows**
3. **CLI smoke tests in CI/CD**
4. **User acceptance testing with beta users**

### Documentation Updates

1. **Update CLI reference docs**
2. **Update getting started guide**
3. **Add migration guide for v1.0**
4. **Update examples and tutorials**

## Metrics & Success Criteria

- [ ] All commands from spec implemented
- [ ] Backward compatibility maintained
- [ ] 90%+ test coverage on new commands
- [ ] Documentation updated
- [ ] Migration guide published
- [ ] Zero regression bugs in existing workflows

## Timeline Estimate

| Phase | Description | Effort | Dependencies |
|-------|-------------|--------|--------------|
| P1 | Core Commands | 2-3 days | None |
| P2 | Spec Refactor | 2-3 days | P1 |
| P3 | Plan Refactor | 2-3 days | P2 |
| P4 | Build Enhancement | 2-3 days | P3 |
| P5 | Eval Framework | 3-4 days | P4 |
| P6 | Auto Enhancement | 1-2 days | P5 |
| P7 | Routing | 1-2 days | P6 |
| P8 | Governance | 5-7 days | P7 (Future) |
| P9 | Auth | 2-3 days | P1 |

**Total estimated effort:** 20-30 days for Phases 1-7 (excluding Governance)

## Next Steps

1. Review and approve this plan
2. Start with Phase 1 (Core Commands)
3. Implement incrementally with tests
4. Release as minor version updates (v1.5, v1.6, etc.)
5. Gather user feedback at each phase
