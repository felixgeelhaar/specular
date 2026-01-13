# Specular Plugin Development

This guide covers everything you need to know about developing plugins for the Specular CLI.

## Overview

Specular's plugin system allows you to extend the CLI's functionality through external executables that communicate via JSON over stdin/stdout. Plugins can be written in any language that supports standard input/output operations.

## Plugin Types

| Type | Description | Use Case |
|------|-------------|----------|
| `provider` | AI provider integrations | Custom LLM backends, model proxies |
| `validator` | Policy enforcement | Content validation, compliance checks |
| `formatter` | Output customization | Custom report formats, export options |
| `hook` | Event handlers | Pre/post actions, CI/CD integration |
| `notifier` | Notification delivery | Slack, Discord, email alerts |

## Supported Languages

Specular provides first-class support for:

- **Go** - Compiled binary, best performance
- **Python** - Easy development, rich ecosystem
- **Node.js** - JavaScript/TypeScript support
- **Shell** - Simple scripts, system integration

## Quick Start

```bash
# Create a new plugin
specular plugin create my-plugin --type hook --lang go

# Build (for Go plugins)
cd my-plugin
go build

# Install
specular plugin install ./my-plugin

# Verify
specular plugin health my-plugin
specular plugin list
```

## Documentation

| Document | Description |
|----------|-------------|
| [Getting Started](getting-started.md) | Step-by-step tutorial |
| [Manifest Reference](manifest-reference.md) | Complete plugin.yaml schema |
| [Protocol Reference](protocol-reference.md) | JSON communication protocol |
| [Best Practices](best-practices.md) | Development guidelines |
| [Publishing Guide](publishing.md) | Registry submission process |

### Language Guides

- [Go Plugin Development](language-guides/go.md)
- [Python Plugin Development](language-guides/python.md)
- [Node.js Plugin Development](language-guides/node.md)
- [Shell Plugin Development](language-guides/shell.md)

## CLI Commands

```bash
# Plugin Management
specular plugin list              # List installed plugins
specular plugin info <name>       # Show plugin details
specular plugin health <name>     # Run health check
specular plugin enable <name>     # Enable a plugin
specular plugin disable <name>    # Disable a plugin

# Installation
specular plugin install <source>  # Install from source
specular plugin uninstall <name>  # Remove a plugin
specular plugin update [name]     # Update plugin(s)

# Development
specular plugin create <name>     # Scaffold new plugin

# Registry
specular plugin search <query>    # Search plugin registry
specular plugin registry-info <n> # Show registry entry
```

## Installation Sources

Plugins can be installed from multiple sources:

```bash
# Local directory
specular plugin install ./my-plugin
specular plugin install /absolute/path/to/plugin

# GitHub repository
specular plugin install github.com/user/repo
specular plugin install github.com/user/repo@v1.2.0
specular plugin install github.com/user/repo@main

# Official registry
specular plugin install registry:slack-notifier
specular plugin install registry:slack-notifier@1.0.0
```

## Plugin Lifecycle

```
┌─────────────────────────────────────────────────────────────┐
│                      Plugin Lifecycle                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   discovered → loaded → enabled ←→ disabled                 │
│       │          │         ↓                                │
│       └──────────┴─────→ error                              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

1. **Discovered** - Plugin directory found
2. **Loaded** - Manifest parsed successfully
3. **Enabled** - Plugin active and ready
4. **Disabled** - Plugin installed but not active
5. **Error** - Plugin failed to load or initialize

## Configuration

Plugins are configured through the main Specular configuration:

```yaml
# .specular.yaml
plugins:
  slack-notifier:
    webhook_url: https://hooks.slack.com/...
    channel: "#alerts"

  custom-validator:
    rules:
      max_length: 1000
      required_fields:
        - title
        - description
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `SPECULAR_PLUGIN_DIRS` | Additional plugin search directories |
| `SPECULAR_REGISTRY_URL` | Custom registry URL |
| `SPECULAR_PLUGIN_TIMEOUT` | Plugin execution timeout |

## Security Considerations

- Plugins run as separate processes with the same user permissions
- Validate plugin sources before installation
- Review plugin manifests for required capabilities
- Use `--force` flag carefully when overwriting plugins
- Consider running in a sandbox for untrusted plugins

## Contributing

To contribute a plugin to the official registry:

1. Follow the [Best Practices](best-practices.md) guide
2. Ensure comprehensive test coverage
3. Create clear documentation
4. Submit to the [specular-plugins](https://github.com/specular/specular-plugins) repository

See [Publishing Guide](publishing.md) for detailed instructions.
