# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
