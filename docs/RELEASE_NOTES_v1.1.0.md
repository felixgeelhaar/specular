# Specular v1.1.0 Release Notes

**Release Date**: November 7, 2025

We're excited to announce Specular v1.1.0, a feature-packed release that significantly enhances the developer experience with an interactive TUI, improved error handling, and expanded AI provider support.

## 🎯 Highlights

### ✨ Interactive TUI Mode

The interview mode now features a beautiful, interactive terminal user interface powered by bubbletea:

- **Visual Progress Tracking**: See your progress through the interview process with intuitive progress bars
- **Keyboard Navigation**: Navigate questions with arrow keys and shortcuts
- **Real-time Validation**: Get immediate feedback on your answers with visual indicators
- **Enhanced UX**: Enjoy a polished interface with consistent theming and smooth transitions

**Try it now:**
```bash
specular interview --tui
```

### 🔍 Enhanced Error System

Debugging issues is now easier than ever with our structured error system:

- **Error Codes**: Every error has a unique code (e.g., `SPEC-001`, `POLICY-003`) for quick reference
- **8 Error Categories**: Organized into SPEC, POLICY, PLAN, INTERVIEW, PROVIDER, EXEC, DRIFT, and IO
- **Actionable Suggestions**: Each error includes specific suggestions for resolution
- **Documentation Links**: Direct links to relevant documentation for troubleshooting

**Example Error:**
```
Error [SPEC-001]: Specification file not found
  → File: .specular/spec.yaml

Suggestions:
  • Run 'specular interview' to create a new specification
  • Check if the file exists at the expected location

Documentation: https://docs.specular.dev/errors/SPEC-001
```

### 🤖 CLI Provider Protocol

Extend Specular with custom AI providers using our new language-agnostic protocol:

- **Simple JSON Protocol**: Stdin/stdout communication with structured JSON
- **Three Commands**: Implement `generate`, `stream`, and `health` for full integration
- **Comprehensive Documentation**: Complete spec in `docs/CLI_PROVIDERS.md`
- **Example Implementations**: Reference implementations for Claude, Codex, and Gemini

**New CLI Providers:**
- **Claude Code**: Anthropic's official CLI wrapper (`providers/claude/`)
- **Codex**: OpenAI's Codex via openai CLI (`providers/codex/`)
- **Gemini CLI**: Google's Gemini/gcloud CLI wrapper (`providers/gemini/`)

**Configuration Example:**
```yaml
# .specular/router.yaml
providers:
  - name: claude
    type: cli
    enabled: true
    priority: 85
    config:
      path: ./providers/claude/claude-provider
      model: claude-sonnet-4-20250514
    models:
      fast: claude-3-5-haiku-20241022
      capable: claude-sonnet-4-20250514
      codegen: claude-sonnet-4-20250514
```

## 🚀 Getting Started

### Installation

**macOS/Linux (Homebrew):**
```bash
brew install felixgeelhaar/tap/specular
```

**Direct Download:**
Download the latest release for your platform from the [releases page](https://github.com/felixgeelhaar/specular/releases/tag/v1.1.0).

### Upgrade from v1.0.x

If you're upgrading from v1.0.x:

1. **Update your installation:**
   ```bash
   brew upgrade specular
   # or download the latest binary
   ```

2. **Try the new TUI mode:**
   ```bash
   specular interview --tui
   ```

3. **Configure CLI providers** (optional):
   ```bash
   cp .specular/router.example.yaml .specular/router.yaml
   # Edit router.yaml to enable desired providers
   ```

## 📚 Documentation

- **CLI Provider Protocol**: [docs/CLI_PROVIDERS.md](../CLI_PROVIDERS.md)
- **Router Configuration**: [.specular/router.example.yaml](../../.specular/router.example.yaml)
- **Error Reference**: See error codes in your terminal output for direct documentation links

## 🔧 What's Changed

### Added
- Interactive TUI mode for interview command with progress tracking and validation
- Structured error system with hierarchical error codes (CATEGORY-NNN format)
- CLI Provider Protocol for language-agnostic provider integration
- Claude Code CLI provider wrapper
- Codex CLI provider wrapper via OpenAI CLI
- Gemini CLI provider wrapper with gcloud fallback
- Comprehensive documentation for CLI providers
- Router configuration example with multiple provider scenarios

### Changed
- Interview mode now defaults to interactive TUI (use `--no-tui` for plain text)
- Error messages include structured codes, suggestions, and documentation links
- Provider selection enhanced with CLI provider support

### Documentation
- Added CLI_PROVIDERS.md with complete protocol specification
- Added router.example.yaml with configuration examples
- Updated README.md with v1.1.0 features
- Updated CLAUDE.md with TUI workflow guidance

## 🐛 Bug Fixes

- Improved error handling across all commands
- Better validation for specification files
- Enhanced provider selection logic

## 🔮 What's Next

Looking ahead to v1.2.0:
- Advanced plan optimization
- Enhanced drift detection
- Additional provider integrations
- Performance improvements

## 📊 Stats

- **6 new features** added
- **3 new CLI providers** implemented
- **All tests passing** across the test suite
- **Comprehensive documentation** updates

## 🙏 Acknowledgments

Thank you to everyone who contributed feedback and suggestions for this release. Your input helps make Specular better for everyone.

## 📝 Full Changelog

See [CHANGELOG.md](../../CHANGELOG.md) for the complete list of changes.

---

**Questions or Issues?**
- 📖 Documentation: https://docs.specular.dev
- 🐛 Report Issues: https://github.com/felixgeelhaar/specular/issues
- 💬 Discussions: https://github.com/felixgeelhaar/specular/discussions

Happy building with Specular v1.1.0! 🚀
