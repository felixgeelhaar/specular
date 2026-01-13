# Specular Plugin Examples

This directory contains example plugins demonstrating different plugin types and languages.

## Available Examples

| Plugin | Type | Language | Description |
|--------|------|----------|-------------|
| [slack-notifier](./slack-notifier/) | notifier | Go | Send notifications to Slack channels |
| [content-validator](./content-validator/) | validator | Python | Validate content against configurable rules |
| [markdown-formatter](./markdown-formatter/) | formatter | Node.js | Format content as Markdown |
| [git-hook](./git-hook/) | hook | Shell | Git integration hooks |

## Plugin Types

### Notifier (slack-notifier)

Notifier plugins send notifications to external services when events occur.

```yaml
type: notifier
capabilities:
  - notifications
```

### Validator (content-validator)

Validator plugins enforce policies and rules on content.

```yaml
type: validator
capabilities:
  - validation
```

### Formatter (markdown-formatter)

Formatter plugins transform output into different formats.

```yaml
type: formatter
capabilities:
  - formatting
```

### Hook (git-hook)

Hook plugins respond to events in the Specular lifecycle.

```yaml
type: hook
capabilities:
  - hook
```

## Getting Started

### Install an Example Plugin

```bash
# Navigate to the example
cd examples/plugins/slack-notifier

# Build if needed (Go plugins)
go build

# Install
specular plugin install .

# Verify
specular plugin list
specular plugin health slack-notifier
```

### Create Your Own Plugin

Use the scaffolding command:

```bash
# Create a new plugin
specular plugin create my-plugin --type hook --lang go

# Available types: provider, validator, formatter, hook, notifier
# Available languages: go, python, node, shell
```

## Documentation

For comprehensive documentation, see:

- [Plugin Development Guide](../../docs/plugin-development/README.md)
- [Getting Started Tutorial](../../docs/plugin-development/getting-started.md)
- [Manifest Reference](../../docs/plugin-development/manifest-reference.md)
- [Protocol Reference](../../docs/plugin-development/protocol-reference.md)

## Testing

Each example includes test instructions in its README. Generally:

```bash
# Test health check
echo '{"action":"health"}' | ./plugin-entrypoint

# Test main action
echo '{"action":"<action>","data":{...}}' | ./plugin-entrypoint
```

## Contributing

To contribute example plugins:

1. Create a directory with your plugin name
2. Include:
   - `plugin.yaml` - Plugin manifest
   - Source files - Implementation
   - `README.md` - Documentation
3. Follow the [Best Practices](../../docs/plugin-development/best-practices.md) guide
4. Submit a pull request
