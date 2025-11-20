# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Interactive Plan Review TUI**
  - Full BubbleTea-based terminal UI for reviewing execution plans
  - Two-view system: list view for overview, detail view for task inspection
  - Vim-style navigation (j/k, h/l, enter, esc)
  - Approve/reject workflow with rejection reason prompt
  - Auto-approve empty plans for convenience
  - Styled with lipgloss for professional appearance
  - Comprehensive test suite with 11 tests

- **Platform API Client v2.0**
  - Production-grade HTTP client for Specular Platform integration
  - Configurable retry logic with exponential backoff
  - Smart retry strategy: retries 5xx errors, fails fast on 4xx
  - Context propagation for request cancellation
  - Structured APIError type with request ID tracking
  - Three endpoints: Health, GenerateSpec, GeneratePlan
  - Comprehensive test suite with 14 tests including retry scenarios

- **Plugin System Enhancements**
  - Plugin installation from local directories
  - Plugin installation from GitHub repositories
  - Automatic dependency resolution

- **Build System Improvements**
  - ImageCache support in autonomous mode executor
  - Manifest loading and validation for build approval
  - Plan task counting and build manifest loading
  - Actual version tracking from builds instead of hardcoded values

- **License & Documentation**
  - PRO tier gates for bundle commands
  - Step-by-step tutorial guides for PRO features
  - Tutorial documentation for advanced workflows

### Fixed

- **Security**: Updated golang.org/x/crypto from v0.43.0 to v0.45.0 (fixes 2 moderate CVEs)
- **Concurrency**: Resolved race condition in DefaultLogger
- **Code Quality**: Fixed import grouping and type assertions in TUI code
- **Linting**: Resolved golangci-lint errors (errcheck, govet shadow, goimports)
- **Policy**: Implemented policy hash change detection for approval workflow

### Changed

- **Build Artifacts**: Added specular build artifacts to .gitignore

## [1.2.0] - 2025-11-17

### Major Changes

This release implements **ADR-0010: Governance-First CLI Redesign**, restructuring the CLI around governance, policy, and approval workflows while maintaining full backward compatibility.

### Added

- **Governance Commands** (NEW)
  - `governance init` - Initialize .specular workspace with governance structure
  - `governance doctor` - Comprehensive governance health checks
  - `governance status` - Display governance workflow status
  - Workspace structure: approvals/, bundles/, traces/, policies.yaml, providers.yaml

- **Policy Management Commands** (NEW)
  - `policy init` - Initialize policy configuration with templates
  - `policy validate` - Validate policies with strict mode support
  - `policy approve` - Approve policy changes with audit trail
  - `policy list` - List all policies with metadata
  - `policy diff` - Compare policy versions

- **Approval Workflow Commands** (NEW)
  - `approval approve` - Approve plans, builds, or drifts with role verification
  - `approval list` - List all approval records with filtering
  - `approval pending` - Show pending approvals requiring action

- **Bundle Command Enhancements**
  - `bundle create` - Create bundles (replaces `bundle build`)
  - `bundle gate` - Quality gate checks (replaces `bundle verify`)
  - `bundle inspect` - Inspect bundle contents
  - `bundle list` - List all bundles with metadata
  - Backward compatibility: `bundle build` and `bundle verify` still work

- **Plan Command Enhancements**
  - `plan create` - Generate plans (replaces `plan gen`)
  - `plan visualize` - Visualize task dependencies
  - `plan validate` - Validate plan structure
  - Backward compatibility: `plan gen` still works with deprecation warning

- **Drift Commands** (PROMOTED)
  - `drift check` - Run drift detection (promoted from `plan drift`)
  - `drift approve` - Approve detected drift
  - Backward compatibility: `plan drift` still works

- **Provider Enhancements**
  - `provider add` - Add providers dynamically (ollama, anthropic, openai, claude-code, gemini-cli, codex-cli, copilot-cli)
  - `provider remove` - Remove providers from configuration
  - `provider doctor` - Health checks (renamed from `provider health`)
  - Backward compatibility: `provider health` still works as alias

- **Unified Doctor Command**
  - Top-level `doctor` command with comprehensive system health checks
  - Governance health checks: workspace, policies, providers config, bundles, approvals, traces
  - Container runtime, AI providers, git, and project validation
  - JSON/YAML output support for automation

### Changed

- **CLI Structure**: Reorganized commands around governance workflows
  - Build commands: `build run`, `build verify`, `build approve`, `build explain`
  - Plan commands now use `create` instead of `gen` as primary command
  - Bundle commands use idiomatic names (`create`, `gate` instead of `build`, `verify`)
  - Provider commands enhanced with add/remove capabilities

- **Command Naming**: More idiomatic and consistent across the CLI
  - `gen` → `create` (for plan generation)
  - `build` → `create` (for bundle creation)
  - `verify` → `gate` (for quality gates)
  - `health` → `doctor` (for diagnostics)

- **Deprecation Warnings**: Added helpful deprecation messages for renamed commands
  - Commands show migration path to new structure
  - All deprecated commands remain functional with backward compatibility

### Fixed

- **Backward Compatibility**: Fixed nil pointer dereferences when using deprecated command forms
  - Safely handle missing flags on root commands
  - Proper default values for optional flags
  - Tested with comprehensive E2E test suite (9/10 passing)

- **Build Command**: Fixed nil pointer bugs in `runBuildRun` for flags: resume, checkpoint-dir, checkpoint-id, feature, verbose, enable-cache, cache-dir, cache-max-age, keep-checkpoint

### Documentation

- **ADR-0010**: Complete governance-first CLI redesign specification
- **CLI Reference**: Updated for new command structure (pending)
- **Migration Guide**: Backward compatibility and migration paths documented

### Statistics

- 12 new governance/policy/approval commands
- 7 enhanced provider commands
- 5 refactored build commands
- 5 refactored plan commands
- 4 refactored bundle commands
- Full backward compatibility maintained
- 9/10 E2E tests passing
- All unit tests passing

## [1.1.0] - 2025-11-16

### Added

- **Public SDK**: Domain types (FeatureID, TaskID, Priority) now available in `pkg/specular/types/` for external integrations
  - Enables third-party tools to use Specular's type system
  - Comprehensive test coverage with property-based tests
  - Full API documentation and validation rules

### Changed

- **SDK Architecture**: Migrated domain types from `internal/domain/` to public SDK (`pkg/specular/types/`)
  - Established one-way dependency: internal packages now import from SDK
  - Eliminated 201 lines of duplicate code
  - Net reduction of 198 lines across 40 files
  - Single source of truth for core domain types
- **CI/CD**: Updated GitHub Actions and integration tests to use Go 1.22 for compatibility

### Documentation

- **ARCHITECTURE.md**: Expanded with detailed directory structure and component descriptions
- **OPEN_SOURCE_PRACTICES.md**: Added comprehensive best practices documentation for contributing
- **SDK README**: Complete documentation for public SDK usage

### Fixed

- **Build**: Removed example workflow and applied consistent code formatting
- **.gitignore**: Properly respect .gitignore for docs/adr/ directory
- **GoReleaser**: Fixed SBOM configuration to prevent duplicate upload errors

### Statistics

- 13 commits since v1.0.0
- 40 files modified in SDK migration
- 849 lines of production SDK code
- Zero breaking changes - fully backward compatible

## [1.0.1] - 2025-11-06

### Fixed

- Minor bug fixes and improvements
- Documentation updates

## [1.0.0] - 2025-11-06

### Core Features

- **Specification-driven Development**: YAML-based feature specifications with lock files
- **Docker-based Sandboxed Execution**: Policy enforcement and secure execution environment
- **Multi-layer Drift Detection**: Plan, code, and infrastructure drift detection
- **Multi-LLM Provider Support**: Anthropic, OpenAI, Gemini, Ollama integration
- **Checkpoint/Resume**: Long-running operation support with state management
- **SARIF Reporting**: Standardized drift reporting format
- **Policy Enforcement**: Docker image allowlisting and execution policies
- **Interview Mode**: Guided Q&A for generating best-practice specifications with interactive TUI
- **Plan Generation**: AI-powered task decomposition and planning
- **Comprehensive Testing**: E2E test coverage across all components

### Autonomous Mode

- **Complete autonomous agent implementation** with 14 major features:
  - Profile system with environment-specific configurations (default, ci, production, strict)
  - Structured action plan format with JSON/YAML serialization
  - Standardized exit codes (0-6) for CI/CD integration
  - Per-step policy checks with context-aware enforcement
  - JSON output format for machine-readable results
  - Scope filtering for feature and path-based execution
  - Max steps limit with configurable safety guardrails
  - Interactive TUI with real-time progress visualization
  - Trace logging with comprehensive execution tracking
  - Patch generation with rollback support and safety verification
  - Cryptographic attestations with ECDSA P-256 signatures and SLSA compliance
  - Explain routing command for routing strategy analysis
  - Hooks system with built-in Script, Webhook, and Slack hooks
  - Advanced security with credential management, audit logging, and secret scanning

### Smart Diagnostics & UX

- **Doctor Command**: Comprehensive system health checks
  - Container runtime detection (Docker, Podman)
  - AI provider detection (Ollama, Claude, OpenAI, Gemini, Anthropic)
  - Language/framework detection (7 languages, 6 frameworks)
  - Git repository context
  - CI environment detection (6 CI systems)
  - Project structure validation
  - Dual output formats: colored text and JSON for automation
  - Actionable next steps based on system state

- **Route Command**: Intelligent routing with five subcommands
  - `route show` - Display routing configuration and model catalog
  - `route test` - Test model selection without provider calls
  - `route explain` - Detailed selection reasoning and cost estimates
  - `route optimize` - Historical routing analysis and cost optimization recommendations
  - `route bench` - Model performance benchmarking and comparison
  - Model catalog with 10 models across 3 providers
  - Support for routing hints: `codegen`, `agentic`, `fast`, `cheap`, `long-context`

- **Init Command**: Smart project initialization
  - 5 project templates (web-app, api-service, cli-tool, microservice, data-pipeline)
  - Automatic environment detection and configuration
  - Provider strategy selection (local, cloud, hybrid)
  - Governance level support (L2, L3, L4)
  - Interactive and non-interactive modes

- **Enhanced UX**:
  - CI-safe interactive prompts for missing required flags
  - Smart path defaults for all file operations
  - Enhanced error messages with actionable recovery suggestions
  - Structured error handling with hierarchical error codes
  - Standardized exit codes for CI/CD integration
  - Environment detection that automatically disables prompts in CI/CD

### CLI Provider Protocol

- **Language-agnostic protocol** for custom AI providers
  - JSON-based stdin/stdout communication
  - Three required commands: generate, stream, health
  - Support for multiple CLI-based providers (Claude Code, Codex, Gemini)
  - Comprehensive documentation and router configuration examples

### CI/CD Integration

- **GitHub Actions Integration**:
  - Composite GitHub Action for seamless CI/CD integration
  - SARIF drift report upload for GitHub Security tab
  - Failure annotations on pull requests with drift/policy violations
  - Docker image caching with 80%+ performance improvement
  - Comprehensive example workflows

- **Platform Support**:
  - GitHub Actions workflow examples
  - GitLab CI pipeline configuration
  - CircleCI workflow
  - Jenkins pipeline
  - Multi-stage workflows (validate, plan, build, evaluate, report)
  - Artifact management and caching strategies

### Production Readiness

- **Deployment Patterns**: Single binary, containerized, Kubernetes
- **Security Hardening**: Secret management, Docker security, network isolation, audit logging
- **Performance Tuning**: Docker caching, profile optimization, cost optimization
- **Monitoring & Observability**: Prometheus metrics, OpenTelemetry tracing, structured logging, alerting
- **Disaster Recovery**: Backup strategy, recovery procedures, checkpoint recovery
- **Troubleshooting**: Common issues, debug mode, diagnostic bundles
- **Production checklist** for deployment validation

### Documentation

- **PRODUCTION_GUIDE.md** (1,043 lines): Complete production deployment guide
- **RELEASE_PROCESS.md** (636 lines): Comprehensive release management documentation
- **Best Practices Guide** (1,200+ lines): Workflows, policies, optimization, troubleshooting
- **Checkpoint/Resume Guide** (800+ lines): Complete checkpoint system documentation
- **Progress Indicators Guide** (750+ lines): Display modes, tracking, integration patterns
- **CLI Providers Documentation**: Complete CLI provider protocol specification
- Installation guides for all platforms (Linux, macOS, Windows)
- Architecture Decision Records (ADRs)
- API documentation and examples

### Deliverables

- Multi-platform binaries (Linux, macOS, Windows × AMD64/ARM64)
- Docker images with multi-architecture support
- Homebrew formula for macOS/Linux
- DEB/RPM packages for Linux distributions

### Statistics

- 8,100+ lines of production code
- 138+ tests with comprehensive coverage
- 6,500+ lines of documentation
- Zero security issues (gosec compliance)
- Production-ready quality across all components

[unreleased]: https://github.com/felixgeelhaar/specular/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/felixgeelhaar/specular/compare/v1.0.0...v1.1.0
[1.0.1]: https://github.com/felixgeelhaar/specular/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/felixgeelhaar/specular/releases/tag/v1.0.0
