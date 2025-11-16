# Specular Architecture

This document describes the repository structure, architecture patterns, and design decisions for the Specular CLI.

## Repository Structure

```
specular/
├── cmd/                    # Application entrypoints
│   └── specular/          # Main CLI application
├── internal/              # Private application code
│   ├── auto/             # Autonomous mode implementation
│   ├── autopolicy/       # Auto-mode policy enforcement
│   ├── bundle/           # Bundle management (spec + plan packaging)
│   ├── cmd/              # Command implementations
│   ├── detect/           # Environment detection
│   ├── domain/           # Domain models and value objects
│   ├── drift/            # Drift detection
│   ├── exec/             # Docker execution engine
│   ├── genplan/          # Plan generation
│   ├── genspec/          # Specification generation
│   ├── hooks/            # Webhook and notification hooks
│   ├── patch/            # Patch generation and management
│   ├── plan/             # Plan models
│   ├── policy/           # Policy evaluation
│   ├── provider/         # AI provider management
│   ├── security/         # Security utilities
│   ├── session/          # Session management
│   ├── spec/             # Specification models
│   ├── trace/            # Trace logging
│   └── ux/               # User experience (formatting, output)
├── pkg/                   # Public SDK for external integrations
│   └── specular/         # Public API
│       ├── types/        # Domain value objects (FeatureID, TaskID, etc.)
│       ├── provider/     # Provider plugin interface
│       ├── platform/     # Platform API client (stub for v2.0)
│       └── features/     # Feature flags (free vs enterprise)
├── providers/             # AI provider implementations
│   ├── anthropic/        # Anthropic Claude provider
│   ├── codex-cli/        # Codex CLI provider
│   ├── gemini-cli/       # Google Gemini provider
│   ├── ollama/           # Ollama local models
│   └── openai/           # OpenAI GPT provider
├── docs/                  # Documentation
│   ├── adr/              # Architecture Decision Records
│   ├── assets/           # Documentation assets (images, diagrams)
│   ├── getting-started.md
│   ├── installation.md
│   ├── CLI_REFERENCE.md
│   ├── BUNDLE_USER_GUIDE.md
│   ├── PRODUCTION_GUIDE.md
│   ├── provider-guide.md
│   └── IP_AUDIT_PLAN.md
├── test/                  # End-to-end tests
│   └── e2e/              # E2E test suites
├── tests/                 # Integration tests
│   ├── fixtures/         # Test fixtures and data
│   └── integration/      # Integration test suites
├── examples/              # Example configurations and use cases
├── scripts/               # Build, release, and utility scripts
├── packaging/             # Distribution packaging
├── completions/           # Shell completions (bash, zsh, fish)
├── .github/               # GitHub-specific files
│   ├── workflows/        # CI/CD workflows
│   ├── actions/          # Custom GitHub Actions
│   ├── ISSUE_TEMPLATE/   # Issue templates
│   └── PULL_REQUEST_TEMPLATE.md
├── LICENSE                # Business Source License 1.1
├── README.md              # Project overview and quick start
├── CONTRIBUTING.md        # Contribution guidelines
├── CODE_OF_CONDUCT.md     # Community code of conduct
├── SECURITY.md            # Security policy and reporting
├── CHANGELOG.md           # Version history
├── Makefile               # Build automation
├── go.mod                 # Go module definition
├── go.sum                 # Go dependency checksums
├── .gitignore             # Git ignore patterns
├── .editorconfig          # Editor configuration
├── .golangci.yml          # Go linter configuration
└── .goreleaser.yml        # Release configuration
```

## Architecture Principles

### 1. Open-Core Model

Specular follows an **open-core architecture** with clear separation between public and proprietary code:

- **Public SDK** (`pkg/specular/`): Type-safe interfaces for external integrations
- **Internal Implementation** (`internal/`): Private application logic protected by Go visibility
- **Provider Plugins**: Extensible AI provider system via `pkg/specular/provider`
- **Feature Flags**: Edition differentiation (free CLI vs enterprise platform)

### 2. Domain-Driven Design

The codebase follows DDD principles:

- **Value Objects** (`pkg/specular/types/`): `FeatureID`, `TaskID`, `Priority` with validation
- **Domain Models** (`internal/domain/`): Core business entities
- **Bounded Contexts**: Clear separation between spec, plan, build, and policy domains
- **Ubiquitous Language**: Consistent terminology across code and documentation

### 3. Hexagonal Architecture (Ports & Adapters)

- **Domain Core** (`internal/domain/`): Business logic independent of external concerns
- **Ports** (`pkg/specular/provider/`): Interfaces for external systems
- **Adapters** (`providers/`): Concrete implementations for AI providers
- **Infrastructure** (`internal/exec/`, `internal/hooks/`): External system integrations

### 4. SOLID Principles

- **Single Responsibility**: Each package has a focused purpose
- **Open/Closed**: Provider plugin system allows extension without modification
- **Liskov Substitution**: All providers implement `ProviderClient` interface
- **Interface Segregation**: Small, focused interfaces (e.g., `ProviderClient`, `PolicyChecker`)
- **Dependency Inversion**: Domain depends on abstractions, not concrete implementations

## Package Organization

### `cmd/` - Application Entrypoints

Contains main packages that produce executables. Only initialization and wiring code.

### `internal/` - Private Application Code

Go's internal package visibility ensures this code cannot be imported by external projects.

#### Key Internal Packages

- **`cmd/`**: Cobra command implementations (not to be confused with `cmd/specular/`)
- **`domain/`**: Core business logic and value objects
- **`genspec/`** & **`genplan/`**: AI-powered generation engines
- **`exec/`**: Docker-based execution with image caching and policy enforcement
- **`policy/`**: OPA-based policy evaluation
- **`provider/`**: AI provider management and routing

### `pkg/` - Public SDK

Public API that external projects can import. Follows semantic versioning.

#### SDK Components

1. **`types/`**: Value objects with validation
   ```go
   featureID, err := types.NewFeatureID("user-authentication")
   ```

2. **`provider/`**: Universal AI provider interface
   ```go
   type ProviderClient interface {
       Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
       Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error)
   }
   ```

3. **`features/`**: Feature flag system
   ```go
   if features.IsEnabled(features.FlagAdvancedPolicy) {
       // Enterprise feature
   }
   ```

4. **`platform/`**: Future enterprise platform integration (v2.0)

### `providers/` - AI Provider Implementations

Each provider implements the `ProviderClient` interface:

- **anthropic**: Claude models (Sonnet, Opus, Haiku)
- **openai**: GPT models
- **gemini-cli**: Google Gemini
- **ollama**: Local models
- **codex-cli**: Codex CLI integration

## Design Patterns

### 1. Provider Plugin System

**Pattern**: Strategy + Factory

```go
// Routing configuration selects provider
router := provider.NewRouter(config)
client, err := router.Route(ctx, task)

// All providers implement same interface
response, err := client.Generate(ctx, request)
```

### 2. Feature Flags

**Pattern**: Toggle Pattern

```go
// Free tier features (always enabled)
const FlagBasicSpec Flag = "basic_spec"

// Enterprise features (disabled in public builds)
const FlagAdvancedPolicy Flag = "advanced_policy"

if features.IsEnabled(FlagAdvancedPolicy) {
    // Use enterprise policy engine
}
```

### 3. Policy Enforcement

**Pattern**: Chain of Responsibility + OPA Integration

```go
// Policies evaluated at multiple checkpoints
- Before spec generation (validate requirements)
- Before plan execution (validate tasks)
- Before build execution (validate changes)
- After execution (validate outcomes)
```

### 4. Docker Execution

**Pattern**: Template Method + Command

```go
// Standardized execution workflow
1. Ensure image exists (pull if needed)
2. Create container with policy-defined constraints
3. Execute command
4. Stream output
5. Collect results
6. Cleanup
```

## Data Flow

### Specification Generation

```
User Requirements
    ↓
genspec (AI-powered)
    ↓
Specification (YAML)
    ↓
spec.lock.json (frozen snapshot)
```

### Plan Generation

```
Specification + Lock File
    ↓
genplan (AI-powered routing)
    ↓
Execution Plan (YAML)
    ├─ Features
    ├─ Tasks (with provider routing)
    └─ Dependencies
```

### Build Execution

```
Plan + Policy
    ↓
Task Routing (by skill/language)
    ↓
Provider Selection (per task)
    ↓
Docker Execution (isolated)
    ↓
Results Collection
    ↓
Trace + Attestation
```

## Testing Strategy

### Unit Tests

- Co-located with source code (`*_test.go`)
- Test individual packages in isolation
- Use table-driven tests
- Mock external dependencies

### Integration Tests

- `tests/integration/`: Cross-package integration
- `tests/fixtures/`: Shared test data
- Test provider integrations with real APIs (when configured)

### End-to-End Tests

- `test/e2e/`: Full workflow tests
- Test complete user scenarios
- Validate CLI commands end-to-end

## Dependency Management

- **Go Modules**: `go.mod` defines dependencies
- **Vendor-Free**: No vendoring; rely on Go module cache
- **Minimal Dependencies**: Prefer standard library where possible
- **Security Scanning**: GitHub Dependabot + `go mod verify`

## Build & Release

### Build Process

```bash
# Local development build
make build

# Run tests
make test

# Lint code
make lint

# Full CI/CD pipeline
make ci
```

### Release Process

- **GoReleaser**: Automated multi-platform builds
- **Semantic Versioning**: `vX.Y.Z` tags
- **Changelog**: Auto-generated from conventional commits
- **Distributions**: Binaries, packages (deb, rpm, apk), Docker images

## Open-Source Best Practices

### Licensing

- **Business Source License 1.1** (BSL 1.1)
- **Change License**: Apache 2.0 (after 2 years)
- **Usage Grant**: All uses except competing SaaS

### Community Health

- ✅ **CODE_OF_CONDUCT.md**: Contributor Covenant
- ✅ **CONTRIBUTING.md**: Contribution guidelines
- ✅ **SECURITY.md**: Security policy and reporting
- ✅ **LICENSE**: BSL 1.1 with clear terms
- ✅ **CHANGELOG.md**: Version history
- ✅ **README.md**: Comprehensive documentation
- ✅ **Issue Templates**: Bug reports, feature requests
- ✅ **PR Template**: Structured pull request descriptions

### Code Quality

- ✅ **`.editorconfig`**: Consistent editor settings
- ✅ **`.golangci.yml`**: Comprehensive linting rules
- ✅ **`.github/workflows/`**: Automated CI/CD
- ✅ **Test Coverage**: 90%+ on critical paths
- ✅ **Static Analysis**: golangci-lint, gosec

## Future Architecture (v2.0)

### Dual-Repository Strategy

```
specular/                  # Public repository (BSL 1.1)
  cli/                    # Current repository
    pkg/specular/        # Public SDK

specular-platform/        # Private repository (proprietary)
  internal/
    intelligence/        # Advanced AI features
    enterprise/          # Enterprise-only features
    web/                # Web dashboard
```

### Platform Integration

- **Public SDK**: `pkg/specular/platform/` provides client stub
- **One-Way Dependency**: Platform imports public SDK, never reverse
- **Feature Flags**: Graceful degradation for missing platform features
- **API Authentication**: OAuth 2.0 / API keys

## References

- [Business Source License 1.1](LICENSE)
- [ADR-0001: IP Protection & Open-Core Strategy](docs/adr/0001-ip-protection-open-core-strategy.md)
- [IP Audit Plan](docs/IP_AUDIT_PLAN.md)
- [Contributing Guidelines](CONTRIBUTING.md)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [SOLID Principles](https://en.wikipedia.org/wiki/SOLID)
- [Domain-Driven Design](https://martinfowler.com/bliki/DomainDrivenDesign.html)
