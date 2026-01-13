# Plugin Development Best Practices

This guide covers best practices for developing production-quality Specular plugins.

## Code Quality

### Error Handling

**Always handle errors gracefully:**

```go
// Good: Return structured error
func executeHook(req PluginRequest) (*Result, error) {
    if req.Data == nil {
        return nil, fmt.Errorf("validation: data field is required")
    }
    // ...
}

// Bad: Panic on error
func executeHook(req PluginRequest) *Result {
    event := req.Data["event"].(string) // Panic if not string!
    // ...
}
```

**Categorize errors:**

```go
const (
    ErrValidation = "validation"
    ErrNetwork    = "network"
    ErrConfig     = "config"
    ErrInternal   = "internal"
)

func wrapError(category string, err error) error {
    return fmt.Errorf("%s: %w", category, err)
}
```

### Input Validation

**Validate all inputs:**

```python
def validate_request(request: dict) -> None:
    if "action" not in request:
        raise ValueError("validation: missing required field 'action'")

    if request["action"] == "notify":
        if "data" not in request or "message" not in request["data"]:
            raise ValueError("validation: notify action requires 'data.message'")
```

**Type check carefully:**

```go
func getString(data map[string]interface{}, key string) (string, error) {
    val, ok := data[key]
    if !ok {
        return "", fmt.Errorf("missing field: %s", key)
    }
    str, ok := val.(string)
    if !ok {
        return "", fmt.Errorf("field %s must be string, got %T", key, val)
    }
    return str, nil
}
```

### Configuration

**Provide sensible defaults:**

```yaml
config:
  - name: timeout
    type: int
    description: "Request timeout in seconds"
    default: 30

  - name: retry_count
    type: int
    description: "Number of retries on failure"
    default: 3
```

**Validate configuration at startup:**

```go
func validateConfig(config map[string]interface{}) error {
    timeout, ok := config["timeout"].(float64)
    if !ok || timeout <= 0 {
        return fmt.Errorf("config: timeout must be a positive number")
    }
    if timeout > 300 {
        return fmt.Errorf("config: timeout cannot exceed 300 seconds")
    }
    return nil
}
```

## Performance

### Startup Time

**Keep initialization minimal:**

```go
// Good: Lazy initialization
var client *http.Client
var once sync.Once

func getClient() *http.Client {
    once.Do(func() {
        client = &http.Client{Timeout: 30 * time.Second}
    })
    return client
}

// Bad: Heavy initialization at startup
var client = createExpensiveClient() // Delays first response
```

### Resource Management

**Clean up resources:**

```python
def notify(data: dict, config: dict) -> dict:
    session = requests.Session()
    try:
        response = session.post(config["webhook_url"], json=data)
        return {"delivered": response.ok}
    finally:
        session.close()
```

**Use connection pooling:**

```go
var httpClient = &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        10,
        MaxIdleConnsPerHost: 5,
        IdleConnTimeout:     30 * time.Second,
    },
    Timeout: 30 * time.Second,
}
```

### Timeouts

**Always set timeouts for external calls:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

req, _ := http.NewRequestWithContext(ctx, "POST", url, body)
resp, err := client.Do(req)
```

```python
import requests

response = requests.post(
    url,
    json=data,
    timeout=10  # 10 second timeout
)
```

## Security

### Secrets Handling

**Mark secrets in manifest:**

```yaml
config:
  - name: api_key
    type: string
    required: true
    secret: true  # Will be masked in logs
```

**Never log secrets:**

```go
// Good: Mask sensitive data
func logRequest(req PluginRequest) {
    masked := maskSecrets(req.Config)
    log.Printf("Processing request with config: %v", masked)
}

// Bad: Log raw config
func logRequest(req PluginRequest) {
    log.Printf("Config: %v", req.Config) // May contain API keys!
}
```

### Input Sanitization

**Sanitize user input:**

```python
import html

def format_message(data: dict) -> str:
    # Escape HTML to prevent XSS in notifications
    message = html.escape(data.get("message", ""))
    return f"<p>{message}</p>"
```

**Validate URLs:**

```go
func validateURL(rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil {
        return err
    }
    if u.Scheme != "https" {
        return fmt.Errorf("only HTTPS URLs are allowed")
    }
    return nil
}
```

### File System Safety

**Restrict file access:**

```go
func readFile(basePath, filename string) ([]byte, error) {
    // Prevent path traversal
    cleanPath := filepath.Clean(filename)
    if strings.HasPrefix(cleanPath, "..") {
        return nil, fmt.Errorf("path traversal not allowed")
    }
    fullPath := filepath.Join(basePath, cleanPath)
    return os.ReadFile(fullPath)
}
```

## Testing

### Unit Tests

**Test all code paths:**

```go
func TestExecuteHook(t *testing.T) {
    tests := []struct {
        name    string
        request PluginRequest
        want    *Result
        wantErr bool
    }{
        {
            name:    "valid request",
            request: PluginRequest{Action: "execute", Data: map[string]interface{}{"event": "test"}},
            want:    &Result{Executed: true},
            wantErr: false,
        },
        {
            name:    "missing data",
            request: PluginRequest{Action: "execute"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := executeHook(tt.request)
            if (err != nil) != tt.wantErr {
                t.Errorf("executeHook() error = %v, wantErr %v", err, tt.wantErr)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("executeHook() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Tests

**Test the full plugin flow:**

```bash
#!/bin/bash
# test_plugin.sh

# Build
go build -o test-plugin

# Test health
RESULT=$(echo '{"action":"health"}' | ./test-plugin)
if ! echo "$RESULT" | jq -e '.success == true' > /dev/null; then
    echo "Health check failed: $RESULT"
    exit 1
fi

# Test main action
RESULT=$(echo '{"action":"execute","data":{"event":"test"}}' | ./test-plugin)
if ! echo "$RESULT" | jq -e '.success == true' > /dev/null; then
    echo "Execute failed: $RESULT"
    exit 1
fi

echo "All tests passed!"
```

### Mock External Services

```python
import unittest
from unittest.mock import patch, MagicMock

class TestNotifier(unittest.TestCase):
    @patch('requests.post')
    def test_notify_success(self, mock_post):
        mock_response = MagicMock()
        mock_response.ok = True
        mock_post.return_value = mock_response

        result = notify(
            {"message": "Test"},
            {"webhook_url": "https://example.com/hook"}
        )

        self.assertTrue(result["delivered"])
        mock_post.assert_called_once()
```

## Documentation

### Code Comments

**Document public functions:**

```go
// ExecuteHook processes a hook event and returns the result.
// It validates the input, executes the configured hook logic,
// and handles any errors that occur during execution.
//
// Supported events: "pre", "post", "execute"
//
// Example:
//
//	result, err := ExecuteHook(PluginRequest{
//	    Action: "execute",
//	    Data:   map[string]interface{}{"event": "test"},
//	})
func ExecuteHook(req PluginRequest) (*Result, error) {
    // ...
}
```

### README

Include a comprehensive README:

```markdown
# My Plugin

Brief description of what the plugin does.

## Installation

```bash
specular plugin install github.com/user/my-plugin
```

## Configuration

```yaml
plugins:
  my-plugin:
    api_key: "your-api-key"
    timeout: 30
```

## Usage

Describe how the plugin integrates with Specular.

## Development

```bash
# Build
go build

# Test
go test ./...

# Run locally
echo '{"action":"health"}' | ./my-plugin
```
```

### Changelog

Maintain a CHANGELOG.md:

```markdown
# Changelog

## [1.1.0] - 2024-01-15

### Added
- Support for batch notifications
- Configurable retry logic

### Fixed
- Handle empty message field gracefully

## [1.0.0] - 2024-01-01

### Added
- Initial release
- Basic notification support
```

## Versioning

### Semantic Versioning

Follow [semver](https://semver.org/):

- **MAJOR** - Breaking changes (config schema, protocol changes)
- **MINOR** - New features (backward compatible)
- **PATCH** - Bug fixes (backward compatible)

### Breaking Changes

Document breaking changes clearly:

```markdown
## Migration Guide: v1.x to v2.x

### Configuration Changes

The `webhook` field has been renamed to `webhook_url`:

```yaml
# Before (v1.x)
plugins:
  my-plugin:
    webhook: "https://..."

# After (v2.x)
plugins:
  my-plugin:
    webhook_url: "https://..."
```
```

## Compatibility

### Specular Version

**Declare minimum version:**

```yaml
min_specular_version: "1.6.0"
```

**Check features at runtime:**

```go
func init() {
    if os.Getenv("SPECULAR_VERSION") != "" {
        // Can check version compatibility
    }
}
```

### Dependencies

**Pin dependency versions:**

```yaml
dependencies:
  - name: common-utils
    version: "^1.2.0"  # Compatible with 1.2.x
```

**Test with multiple versions:**

```yaml
# .github/workflows/test.yml
strategy:
  matrix:
    specular: ['1.6.0', '1.7.0', 'latest']
```

## Checklist

Before publishing, ensure:

- [ ] All tests pass
- [ ] Health check implemented and working
- [ ] Error handling covers all paths
- [ ] Configuration validated at startup
- [ ] Secrets marked appropriately
- [ ] Documentation complete
- [ ] Version follows semver
- [ ] Changelog updated
- [ ] License file included
- [ ] README includes installation and usage
