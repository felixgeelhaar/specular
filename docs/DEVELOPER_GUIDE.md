# Developer's Guide to Specular

**Welcome!** This guide provides an overview of the Specular codebase, recent improvements, and where to find specific documentation.

**Last Updated:** 2025-01-10
**Project:** Specular - AI-Native Development Framework
**Version:** v1.3.0+

---

## 🚀 Quick Start

### For New Contributors
1. Read [getting-started.md](getting-started.md) for project overview
2. Review [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines
3. Check [installation.md](installation.md) for development setup
4. Browse [best-practices.md](best-practices.md) for coding standards

### For Existing Contributors
- Recent changes: [Code Quality Metrics](CODE_QUALITY_METRICS.md)
- Future plans: [Improvement Roadmap](IMPROVEMENT_ROADMAP.md)
- Architecture decisions: [adr/](adr/) directory

---

## 📚 Documentation Map

### Core Documentation

#### **Getting Started**
- [installation.md](installation.md) - Installation and setup (991 lines)
- [getting-started.md](getting-started.md) - Quick start guide
- [CLI_PROVIDERS.md](CLI_PROVIDERS.md) - CLI command reference
- [provider-guide.md](provider-guide.md) - AI provider configuration

#### **Architecture & Design**
- [tech_design.md](tech_design.md) - Technical design overview
- [ARCHITECTURE_REVIEW.md](ARCHITECTURE_REVIEW.md) - Architecture review
- [adr/](adr/) - Architecture Decision Records (6 ADRs)
  - [0001-spec-lock-format.md](adr/0001-spec-lock-format.md) - Lock file design
  - [0002-checkpoint-mechanism.md](adr/0002-checkpoint-mechanism.md) - State persistence
  - [0003-docker-only-execution.md](adr/0003-docker-only-execution.md) - Container strategy
  - [0004-provider-abstraction.md](adr/0004-provider-abstraction.md) - AI provider abstraction
  - [0005-drift-detection-approach.md](adr/0005-drift-detection-approach.md) - Drift detection
  - **[0006-domain-value-objects.md](adr/0006-domain-value-objects.md)** - ⭐ NEW: Domain model refactoring

#### **Quality & Best Practices**
- **[CODE_QUALITY_METRICS.md](CODE_QUALITY_METRICS.md)** - ⭐ NEW: Test coverage and quality analysis
- **[IMPROVEMENT_ROADMAP.md](IMPROVEMENT_ROADMAP.md)** - ⭐ NEW: Future development roadmap
- [best-practices.md](best-practices.md) - Coding standards
- [SECURITY_AUDIT.md](SECURITY_AUDIT.md) - Security guidelines
- [E2E_TEST_PLAN.md](E2E_TEST_PLAN.md) - End-to-end testing strategy

### Feature-Specific Documentation

#### **Bundle System**
- [BUNDLE_USER_GUIDE.md](BUNDLE_USER_GUIDE.md) - Bundle management guide
- [governance-bundle-roadmap.md](governance-bundle-roadmap.md) - Bundle governance
- [v1.3.0-governance-bundle-plan.md](v1.3.0-governance-bundle-plan.md) - v1.3.0 bundle features

#### **Checkpoint & Resume**
- [checkpoint-resume.md](checkpoint-resume.md) - State persistence guide
- [progress-indicators.md](progress-indicators.md) - Progress tracking

#### **Product Requirements**
- [prd.md](prd.md) - Product requirements document
- [mvp-action-plan.md](mvp-action-plan.md) - MVP implementation plan

### Release & Deployment

#### **Release Management**
- [RELEASE_CHECKLIST.md](RELEASE_CHECKLIST.md) - Pre-release verification
- [RELEASE_NOTES_v1.3.0.md](RELEASE_NOTES_v1.3.0.md) - v1.3.0 release notes
- [RELEASE_NOTES_v1.2.0.md](release-notes-v1.2.0.md) - v1.2.0 release notes
- [RELEASE_NOTES_v1.1.0.md](RELEASE_NOTES_v1.1.0.md) - v1.1.0 release notes

#### **Deployment**
- [homebrew-tap-setup.md](homebrew-tap-setup.md) - Homebrew tap configuration
- [pro-tier-licensing-plan.md](pro-tier-licensing-plan.md) - Licensing strategy

### Sprint Planning

#### **Completed Sprints**
- [sprint1-summary.md](sprint1-summary.md) - Sprint 1 retrospective
- [sprint2-summary.md](sprint2-summary.md) - Sprint 2 retrospective
- [sprint3-summary.md](sprint3-summary.md) - Sprint 3 retrospective

#### **Version Planning**
- [v1.2.0-plan.md](v1.2.0-plan.md) - v1.2.0 planning
- [v1.2.0-implementation-status.md](v1.2.0-implementation-status.md) - Implementation tracking
- [v1.2.0-release-preparation.md](v1.2.0-release-preparation.md) - Release prep
- [v1.2.0-release-summary.md](v1.2.0-release-summary.md) - Release summary
- [release-strategy-v1.2.0.md](release-strategy-v1.2.0.md) - Release strategy

---

## 🏗️ Architecture Overview

### Project Structure

```
specular/
├── cmd/                    # CLI entry points
│   └── specular/          # Main CLI application
├── internal/               # Internal packages
│   ├── bundle/            # Bundle management (36% coverage)
│   ├── checkpoint/        # State persistence (89% coverage)
│   ├── cmd/               # Cobra command definitions
│   ├── detect/            # Project detection (39% coverage)
│   ├── domain/            # ⭐ Domain value objects (98% coverage)
│   ├── drift/             # Drift detection (94% coverage)
│   ├── errors/            # Error types (100% coverage)
│   ├── eval/              # Evaluation engine (82% coverage)
│   ├── exec/              # Docker execution (55% coverage)
│   ├── exitcode/          # Exit code handling (89% coverage)
│   ├── interview/         # User interview (98% coverage)
│   ├── plan/              # Plan generation (94% coverage)
│   ├── policy/            # Policy validation (100% coverage)
│   ├── prd/               # PRD parsing (92% coverage)
│   ├── progress/          # Progress tracking (75% coverage)
│   ├── provider/          # AI provider abstraction (81% coverage)
│   ├── router/            # AI model routing (74% coverage)
│   ├── spec/              # Spec parsing (96% coverage)
│   ├── tui/               # Terminal UI (41% coverage)
│   ├── ux/                # User experience (79% coverage)
│   ├── version/           # Version info (100% coverage)
│   └── workflow/          # Workflow orchestration (61% coverage)
├── providers/              # Provider implementations
│   ├── claude/            # Anthropic Claude
│   ├── codex/             # OpenAI Codex
│   ├── gemini/            # Google Gemini
│   └── ollama/            # Ollama (local)
├── docs/                   # Documentation (YOU ARE HERE)
└── examples/               # Example projects
```

### Key Components

#### **Domain Layer** ⭐ NEW (v1.3.0+)
**Location:** `internal/domain/`
**Coverage:** 98%
**Purpose:** Type-safe value objects for domain entities

**Value Objects:**
- `TaskID` - Unique task identifier with validation
- `FeatureID` - Unique feature identifier with validation
- `Priority` - Task/feature priority (P0, P1, P2)

**Key Benefits:**
- Compile-time type safety
- Runtime validation
- Self-documenting types
- Zero runtime overhead

**Documentation:** [ADR 0006](adr/0006-domain-value-objects.md)

#### **Specification System**
**Location:** `internal/spec/`
**Coverage:** 96%
**Purpose:** Parse and validate product specifications

**Key Files:**
- `loader.go` - Load specs from YAML
- `lock.go` - Generate deterministic lock files
- `validation.go` - Spec validation logic

**Documentation:** [ADR 0001](adr/0001-spec-lock-format.md)

#### **Plan Generation**
**Location:** `internal/plan/`
**Coverage:** 94%
**Purpose:** Generate execution plans from specifications

**Key Files:**
- `generator.go` - Plan generation logic
- `task.go` - Task definitions
- `dependencies.go` - Dependency resolution

#### **Drift Detection**
**Location:** `internal/drift/`
**Coverage:** 94%
**Purpose:** Detect divergence between spec and implementation

**Key Files:**
- `detector.go` - Plan drift detection
- `code.go` - Code drift detection
- `openapi.go` - API drift detection
- `infra.go` - Infrastructure drift detection

**Documentation:** [ADR 0005](adr/0005-drift-detection-approach.md)

#### **Provider System**
**Location:** `internal/provider/`, `providers/`
**Coverage:** 81%
**Purpose:** Abstract AI provider interactions

**Supported Providers:**
- Anthropic Claude (Claude 3.5 Sonnet)
- OpenAI (GPT-4, GPT-3.5)
- Google Gemini
- Ollama (local models)

**Documentation:** [ADR 0004](adr/0004-provider-abstraction.md), [provider-guide.md](provider-guide.md)

#### **Router System**
**Location:** `internal/router/`
**Coverage:** 74%
**Purpose:** Intelligent model selection and request routing

**Key Features:**
- Cost-aware routing
- Latency-based selection
- Fallback strategies
- Context validation

#### **Execution Engine**
**Location:** `internal/exec/`
**Coverage:** 55%
**Purpose:** Docker-based sandboxed execution

**Documentation:** [ADR 0003](adr/0003-docker-only-execution.md)

---

## 🎯 Recent Improvements (2025-01-10)

### Domain-Driven Design Refactoring

**Summary:** Complete refactoring introducing strongly-typed value objects for domain identifiers.

**Commits:** 13 total
**Files Changed:** 50+
**Documentation:** 1,116 new lines across 3 documents

#### What Changed

**Created `internal/domain` package** with 3 value objects:
```go
type TaskID string      // Task identifiers
type FeatureID string   // Feature identifiers
type Priority string    // Priority levels (P0, P1, P2)
```

**Integrated across 6 packages:**
- `internal/spec` - Feature definitions
- `internal/plan` - Task definitions
- `internal/drift` - Drift findings
- `internal/router` - Usage tracking
- `internal/cmd` - CLI commands
- Test files (50+ files updated)

#### Before & After

**Before:**
```go
// Type confusion possible
func ProcessTask(taskID string, featureID string) {
    DoSomething(featureID, taskID) // Bug! IDs swapped
    // Compiler doesn't catch this ❌
}
```

**After:**
```go
// Compile-time type safety
func ProcessTask(taskID domain.TaskID, featureID domain.FeatureID) {
    DoSomething(featureID, taskID) // Compilation error! ✅
}
```

#### Impact

- **Type Safety:** Prevents ID mixing at compile time
- **Validation:** All IDs validated at construction
- **Test Coverage:** 98% on domain package
- **Zero Regressions:** All tests passing
- **Documentation:** ADR 0006, quality metrics, roadmap

**See:** [ADR 0006](adr/0006-domain-value-objects.md) for complete details

---

## 📊 Code Quality Status

**Overall Coverage:** 45.9%
**Core Business Logic:** 96.7% average
**Domain Package:** 98%

### Coverage by Category

#### Excellent (>90%)
9 packages averaging 96.7%:
- errors, policy, version (100%)
- domain (98%)
- interview (97.7%)
- spec (95.8%)
- drift, plan, prd (92-94%)

#### Good (70-90%)
7 packages averaging 81.4%:
- checkpoint, exitcode, eval, provider, ux, progress, router

#### Needs Improvement (<70%)
5 packages averaging 46.2%:
- workflow, exec, tui, detect, bundle

**See:** [CODE_QUALITY_METRICS.md](CODE_QUALITY_METRICS.md) for detailed analysis

---

## 🗺️ Development Roadmap

### High Priority (Next 2 Weeks)
1. **Improve infrastructure testing** (36% → 60%)
2. **Add integration test suite**
3. **Performance benchmarking**

### Medium Priority (1-2 Months)
1. **Property-based testing** for value objects
2. **Mutation testing** (target: >80%)
3. **Enhanced error handling**

### Long-Term (3-6 Months)
1. **Observability & monitoring**
2. **Advanced domain patterns**
3. **Plugin architecture**

**See:** [IMPROVEMENT_ROADMAP.md](IMPROVEMENT_ROADMAP.md) for complete roadmap

---

## 🔧 Development Workflows

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/domain/...

# With coverage
go test -cover ./...

# With race detector
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Building

```bash
# Build binary
make build

# Or directly
go build -o specular ./cmd/specular
```

### Linting

```bash
# Run linter
golangci-lint run

# Specific package
golangci-lint run ./internal/domain/...

# Fix auto-fixable issues
golangci-lint run --fix
```

### Code Quality

```bash
# Check cyclomatic complexity
gocyclo -over 15 internal/...

# Format code
go fmt ./...
goimports -w .

# Vet code
go vet ./...
```

---

## 🧪 Testing Strategy

### Test Pyramid

- **Unit Tests (70%):** Test individual functions and methods
- **Integration Tests (20%):** Test component interactions
- **E2E Tests (10%):** Test complete workflows

### Testing Guidelines

1. **Test file naming:** `*_test.go`
2. **Table-driven tests:** Use subtests for variations
3. **Test coverage:** Aim for >90% on critical paths
4. **Mocking:** Use interfaces for external dependencies
5. **Fixtures:** Store test data in `testdata/` directories

### Example Test Structure

```go
func TestTaskID_Validate(t *testing.T) {
    tests := []struct {
        name    string
        id      domain.TaskID
        wantErr bool
    }{
        {
            name:    "valid ID",
            id:      domain.TaskID("task-001"),
            wantErr: false,
        },
        {
            name:    "empty ID",
            id:      domain.TaskID(""),
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.id.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

---

## 📖 ADR Reading Guide

### What are ADRs?

Architecture Decision Records (ADRs) document significant architectural decisions, their context, and consequences.

### Reading Order for New Contributors

1. **Start here:** [0001-spec-lock-format.md](adr/0001-spec-lock-format.md) - Foundation
2. **Next:** [0004-provider-abstraction.md](adr/0004-provider-abstraction.md) - Core abstraction
3. **Then:** [0005-drift-detection-approach.md](adr/0005-drift-detection-approach.md) - Key feature
4. **Recent:** [0006-domain-value-objects.md](adr/0006-domain-value-objects.md) - Type safety

### ADR Template

When creating new ADRs, follow this structure:
- **Status:** Proposed/Accepted/Deprecated
- **Date:** YYYY-MM-DD
- **Context:** What led to this decision?
- **Decision:** What did we decide?
- **Implementation:** How was it implemented?
- **Consequences:** What are the trade-offs?
- **Related Decisions:** Links to other ADRs

---

## 🤝 Contributing

### Code Review Checklist

- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] ADR created (if significant architectural change)
- [ ] Coverage maintained/improved
- [ ] No hardcoded secrets
- [ ] Error handling comprehensive
- [ ] Code formatted (`go fmt`, `goimports`)
- [ ] Linter passing (`golangci-lint`)
- [ ] Commits follow conventional format

### Commit Message Format

```
type(scope): subject

body

footer
```

**Types:** feat, fix, docs, refactor, test, chore
**Scope:** package name (domain, spec, plan, etc.)

**Examples:**
```
feat(domain): add TaskID value object
fix(router): correct model selection logic
docs(adr): add ADR 0006 for domain value objects
refactor(plan): simplify dependency resolution
test(spec): add tests for validation edge cases
```

---

## 🆘 Getting Help

### Documentation
- Check this guide first
- Browse [docs/](.) directory
- Read relevant ADRs

### Code
- Check [best-practices.md](best-practices.md)
- Look for similar code in codebase
- Review tests for usage examples

### Issues
- Search existing GitHub issues
- Check [IMPROVEMENT_ROADMAP.md](IMPROVEMENT_ROADMAP.md)
- Create new issue with template

---

## 📚 Further Reading

### Books
- *Domain-Driven Design* - Eric Evans
- *Clean Architecture* - Robert C. Martin
- *The Go Programming Language* - Donovan & Kernighan

### Go Resources
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Proverbs](https://go-proverbs.github.io/)

### DDD Resources
- [Domain-Driven Design Community](https://domainlanguage.com/)
- [Value Objects - Martin Fowler](https://martinfowler.com/bliki/ValueObject.html)

---

## 🎯 Quick Reference

### Key Commands
```bash
# Build
make build

# Test
go test ./...

# Coverage
go test -cover ./...

# Lint
golangci-lint run

# Format
go fmt ./... && goimports -w .
```

### Key Directories
- `internal/domain/` - Domain value objects ⭐
- `internal/spec/` - Specification parsing
- `internal/plan/` - Plan generation
- `internal/drift/` - Drift detection
- `docs/` - All documentation
- `docs/adr/` - Architecture decisions

### Key Documents
- [ADR 0006](adr/0006-domain-value-objects.md) - Recent refactoring
- [CODE_QUALITY_METRICS.md](CODE_QUALITY_METRICS.md) - Quality status
- [IMPROVEMENT_ROADMAP.md](IMPROVEMENT_ROADMAP.md) - Future plans

---

**Last Updated:** 2025-01-10
**Maintained By:** Specular Core Team
**Questions?** Check [CONTRIBUTING.md](../CONTRIBUTING.md) or create an issue
