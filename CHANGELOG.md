# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2025-11-07

### Added

- **Interactive TUI Mode**: Beautiful terminal UI for interview mode powered by bubbletea
  - Real-time progress tracking with progress bars
  - Visual question navigation
  - Answer validation with immediate feedback
  - Enhanced user experience with keyboard shortcuts and visual feedback

- **Enhanced Error System**: Structured errors with hierarchical error codes
  - 8 error categories: SPEC, POLICY, PLAN, INTERVIEW, PROVIDER, EXEC, DRIFT, IO
  - Actionable suggestions for every error
  - Documentation links for troubleshooting
  - Error code format: CATEGORY-NNN (e.g., SPEC-001, POLICY-003)

- **CLI Provider Protocol**: Language-agnostic protocol for custom AI providers
  - JSON-based stdin/stdout communication
  - Three required commands: generate, stream, health
  - Comprehensive documentation in `docs/CLI_PROVIDERS.md`
  - Example router configuration in `.specular/router.example.yaml`

- **New CLI Providers**: Support for additional AI provider CLIs
  - Claude Code provider (`providers/claude/`) - Anthropic's claude CLI wrapper
  - Codex provider (`providers/codex/`) - OpenAI's codex via openai CLI wrapper
  - Gemini CLI provider (`providers/gemini/`) - Google's gemini/gcloud CLI wrapper
  - All providers support model configuration, temperature, and max tokens

### Changed

- Interview mode now defaults to interactive TUI instead of plain text Q&A
- Error messages now include structured codes and suggestions
- Provider selection now supports CLI-based providers via router configuration

### Documentation

- Added `docs/CLI_PROVIDERS.md` - Complete CLI provider protocol specification
- Added `.specular/router.example.yaml` - Router configuration template with examples
- Updated README.md with v1.1.0 features and usage examples
- Updated CLAUDE.md with TUI mode documentation and workflow guidance

## [1.0.1] - 2025-11-06

### Fixed

- Minor bug fixes and improvements
- Documentation updates

## [1.0.0] - 2025-11-06

### Added

- **Specification-driven Development**: YAML-based feature specifications with lock files
- **Docker-based Sandboxed Execution**: Policy enforcement and secure execution environment
- **Multi-layer Drift Detection**: Plan, code, and infrastructure drift detection
- **Multi-LLM Provider Support**: Anthropic, OpenAI, Gemini, Ollama integration
- **Checkpoint/Resume**: Long-running operation support with state management
- **SARIF Reporting**: Standardized drift reporting format
- **Policy Enforcement**: Docker image allowlisting and execution policies
- **Interview Mode**: Guided Q&A for generating best-practice specifications
- **Plan Generation**: AI-powered task decomposition and planning
- **Comprehensive Testing**: E2E test coverage across all components

### Documentation

- Installation guide for all platforms (Linux, macOS, Windows)
- Architecture Decision Records (ADRs)
- API documentation and examples
- Troubleshooting guides

### Deliverables

- Multi-platform binaries (Linux, macOS, Windows × AMD64/ARM64)
- Docker images with multi-architecture support
- Homebrew formula for macOS/Linux
- DEB/RPM packages for Linux distributions

[unreleased]: https://github.com/felixgeelhaar/specular/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/felixgeelhaar/specular/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/felixgeelhaar/specular/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/felixgeelhaar/specular/releases/tag/v1.0.0
