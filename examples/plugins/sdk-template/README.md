# Specular Plugin SDK Template

This is a starting template for creating Specular plugins in Go.

## Quick Start

1. Copy this template to your new plugin directory
2. Update `plugin.yaml` with your plugin information
3. Implement your logic in `main.go`
4. Build and install

```bash
# Build the plugin
go build -o plugin .

# Install to Specular plugins directory
mkdir -p ~/.specular/plugins/my-plugin
cp plugin plugin.yaml ~/.specular/plugins/my-plugin/

# Verify installation
specular plugin list
specular plugin health my-plugin
```

## Plugin Protocol

Specular plugins communicate via stdin/stdout using JSON:

### Request Format (stdin)

```json
{
  "action": "notify",
  "params": {
    "key": "value"
  },
  "config": {
    "api_key": "xxx"
  }
}
```

### Response Format (stdout)

```json
{
  "success": true,
  "result": {
    "message": "Success"
  }
}
```

Or for errors:

```json
{
  "success": false,
  "error": "Error message"
}
```

## Plugin Types

### Notifier
Send notifications to external services (Slack, email, etc.)

Actions:
- `notify`: Send a notification

### Validator
Validate content against rules

Actions:
- `validate`: Check content validity

### Formatter
Format output in custom ways

Actions:
- `format`: Transform data to output

### Provider
Custom AI model provider

Actions:
- `generate`: Generate text
- `list_models`: List available models

### Hook
React to Specular events

Actions:
- `hook`: Handle an event

## Configuration

Plugin configuration is set by users and passed in the `config` field:

```yaml
# In plugin.yaml
config:
  - name: webhook_url
    type: string
    description: Slack webhook URL
    required: true
    secret: true
```

Users configure plugins via:
```bash
specular plugin config my-plugin webhook_url "https://..."
```

## Testing

Test your plugin locally:

```bash
# Build
go build -o plugin .

# Test health check
echo '{"action":"health"}' | ./plugin

# Test custom action
echo '{"action":"notify","params":{"message":"test"},"config":{"api_key":"xxx"}}' | ./plugin
```

## Publishing

1. Create a GitHub repository for your plugin
2. Include `plugin.yaml` and built binaries for each platform
3. Users can install with:
   ```bash
   specular plugin install github.com/you/specular-my-plugin
   ```

## Examples

See the `examples/plugins` directory for complete plugin implementations:
- `slack-notifier`: Send notifications to Slack
- `hello-world`: Simple test plugin

## Support

- [Specular Documentation](https://github.com/felixgeelhaar/specular)
- [Plugin API Reference](https://github.com/felixgeelhaar/specular/docs/plugins.md)
