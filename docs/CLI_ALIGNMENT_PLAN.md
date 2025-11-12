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

### Phase 1: Core Commands (Priority: HIGH)

**Goal:** Add essential missing commands for basic workflow

1. **`context` command** - Environment detection
   - Detect installed models (Ollama)
   - Check API keys (OpenAI, Anthropic, Gemini)
   - Verify Docker installation
   - Output: JSON/YAML summary

2. **`config` command** - Configuration management
   - `config view` - Show current config
   - `config edit` - Open in $EDITOR
   - `config get <key>` - Get specific value
   - `config set <key> <value>` - Set value

3. **`status` command** - Overall status
   - Current spec version and lock status
   - Active plan status
   - Last build/eval results
   - Environment health check

4. **`logs` command** - Log management
   - `logs` - Show recent logs
   - `logs --tail` - Tail logs
   - `logs --trace <id>` - Show specific trace
   - Logs stored in `~/.specular/logs/trace_<id>.json`

### Phase 2: Spec Management Refactor (Priority: HIGH)

**Goal:** Align spec commands with v1.0 specification

1. **Refactor `spec` subcommands:**
   - `spec new [--from <file>]` - Merge `interview` and `spec generate`
   - `spec edit` - Open current spec in editor
   - `spec validate` - ✅ Already exists
   - `spec lock [--note "<msg>"]` - ✅ Already exists, add note flag
   - `spec diff <versionA> <versionB>` - NEW: Compare spec versions
   - `spec approve` - NEW: Approve spec for use

2. **Deprecation path for `interview`:**
   - Add deprecation notice to `interview`
   - Point users to `spec new`
   - Keep `interview` as alias for 1-2 releases

### Phase 3: Plan Management Refactor (Priority: HIGH)

**Goal:** Add plan review and drift detection

1. **Refactor `plan` to `plan gen`:**
   - Current `plan` becomes `plan gen`
   - Add `--feature <id>` flag for feature-specific plans
   - Keep backward compatibility

2. **Add new plan subcommands:**
   - `plan review` - Interactive plan review (TUI)
   - `plan drift` - Detect drift between plan and repo
   - `plan explain <step>` - Explain routing for specific step

### Phase 4: Build & Execution Enhancement (Priority: MEDIUM)

**Goal:** Add verification and approval steps

1. **Refactor `build` to `build run`:**
   - Current `build` becomes `build run`
   - Add `--feature <id>` flag
   - Keep backward compatibility

2. **Add build subcommands:**
   - `build verify` - Run lint, tests, policy checks
   - `build approve` - Approve build results
   - `build explain` - Show logs and routing decisions

### Phase 5: Evaluation Framework (Priority: MEDIUM)

**Goal:** Structured evaluation and guardrails

1. **Add `eval` subcommands:**
   - `eval run [--scenario <name>]` - Run evaluation scenarios
   - `eval rules` - Edit or list guardrail rules
   - `eval drift` - Compare eval metrics across runs

2. **Define eval scenarios:**
   - smoke - Basic health checks
   - integration - Full integration tests
   - security - Security scan + policy check
   - performance - Performance benchmarks

### Phase 6: Auto Mode Enhancement (Priority: MEDIUM)

**Goal:** Add Auto history and explain

1. **Add auto subcommands:**
   - `auto resume` - Resume paused session
   - `auto history` - View logs and history
   - `auto explain` - Explain reasoning per step

2. **Session persistence:**
   - Store sessions in `~/.specular/auto/sessions/`
   - Allow resume after interruption

### Phase 7: Routing Intelligence (Priority: LOW)

**Goal:** Expose routing decisions

1. **Add routing commands:**
   - `route list` - List providers and costs
   - `route override <provider>` - Override for session
   - `route explain <task>` - Explain routing logic

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
