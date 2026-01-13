# Go Plugin Development

This guide covers Go-specific best practices for Specular plugin development.

## Why Go?

- **Performance:** Compiled binaries with fast startup
- **Portability:** Single binary, no runtime dependencies
- **Type Safety:** Compile-time error checking
- **Concurrency:** Built-in goroutines for async operations

## Quick Start

```bash
specular plugin create my-plugin --type hook --lang go
cd my-plugin
go build
echo '{"action":"health"}' | ./my-plugin
```

## Project Structure

```
my-plugin/
├── plugin.yaml      # Plugin manifest
├── go.mod           # Go module file
├── go.sum           # Dependency checksums
├── main.go          # Entry point
├── handler.go       # Action handlers
├── types.go         # Type definitions
├── config.go        # Configuration handling
└── internal/        # Private packages
    └── util/
        └── util.go
```

## Code Templates

### Basic Plugin Structure

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
)

const Version = "1.0.0"

// Request represents incoming plugin requests
type Request struct {
    Action string                 `json:"action"`
    Params map[string]interface{} `json:"params,omitempty"`
    Data   map[string]interface{} `json:"data,omitempty"`
    Config map[string]interface{} `json:"config,omitempty"`
}

// Response represents plugin responses
type Response struct {
    Success bool        `json:"success"`
    Result  interface{} `json:"result,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func main() {
    scanner := bufio.NewScanner(os.Stdin)
    // Increase buffer for large inputs
    scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

    for scanner.Scan() {
        var req Request
        if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
            respond(Response{
                Success: false,
                Error:   fmt.Sprintf("json: %v", err),
            })
            continue
        }

        response := handleRequest(req)
        respond(response)
    }

    if err := scanner.Err(); err != nil {
        fmt.Fprintf(os.Stderr, "stdin error: %v\n", err)
        os.Exit(1)
    }
}

func handleRequest(req Request) Response {
    switch req.Action {
    case "health":
        return handleHealth()
    case "execute":
        return handleExecute(req)
    default:
        return Response{
            Success: false,
            Error:   fmt.Sprintf("unknown action: %s", req.Action),
        }
    }
}

func handleHealth() Response {
    return Response{
        Success: true,
        Result: map[string]string{
            "status":  "healthy",
            "version": Version,
        },
    }
}

func handleExecute(req Request) Response {
    // Your implementation here
    return Response{
        Success: true,
        Result:  map[string]interface{}{"executed": true},
    }
}

func respond(resp Response) {
    if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
        fmt.Fprintf(os.Stderr, "response error: %v\n", err)
    }
}
```

### Type-Safe Configuration

```go
// Config holds plugin configuration
type Config struct {
    APIKey   string        `json:"api_key"`
    Endpoint string        `json:"endpoint"`
    Timeout  time.Duration `json:"timeout"`
    Retries  int           `json:"retries"`
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
    return Config{
        Endpoint: "https://api.example.com",
        Timeout:  30 * time.Second,
        Retries:  3,
    }
}

// ParseConfig parses configuration from request
func ParseConfig(raw map[string]interface{}) (*Config, error) {
    cfg := DefaultConfig()

    if apiKey, ok := raw["api_key"].(string); ok {
        cfg.APIKey = apiKey
    } else {
        return nil, fmt.Errorf("config: api_key is required")
    }

    if endpoint, ok := raw["endpoint"].(string); ok {
        cfg.Endpoint = endpoint
    }

    if timeout, ok := raw["timeout"].(float64); ok {
        cfg.Timeout = time.Duration(timeout) * time.Second
    }

    if retries, ok := raw["retries"].(float64); ok {
        cfg.Retries = int(retries)
    }

    return &cfg, nil
}
```

### Error Handling

```go
// PluginError represents a categorized error
type PluginError struct {
    Category string
    Message  string
    Cause    error
}

func (e *PluginError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s: %v", e.Category, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Category, e.Message)
}

// Error categories
const (
    ErrConfig     = "config"
    ErrValidation = "validation"
    ErrNetwork    = "network"
    ErrInternal   = "internal"
)

func newError(category, message string, cause error) *PluginError {
    return &PluginError{
        Category: category,
        Message:  message,
        Cause:    cause,
    }
}

// Usage
func validateInput(data map[string]interface{}) error {
    if _, ok := data["required_field"]; !ok {
        return newError(ErrValidation, "missing required_field", nil)
    }
    return nil
}
```

### HTTP Client

```go
import (
    "context"
    "net/http"
    "time"
)

// Client is a configured HTTP client
type Client struct {
    http    *http.Client
    baseURL string
    apiKey  string
}

// NewClient creates a new HTTP client
func NewClient(config *Config) *Client {
    return &Client{
        http: &http.Client{
            Timeout: config.Timeout,
            Transport: &http.Transport{
                MaxIdleConns:        10,
                MaxIdleConnsPerHost: 5,
                IdleConnTimeout:     30 * time.Second,
            },
        },
        baseURL: config.Endpoint,
        apiKey:  config.APIKey,
    }
}

// Do executes an HTTP request with context
func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
    req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")

    return c.http.Do(req)
}
```

### Retry Logic

```go
import (
    "math/rand"
    "time"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
    MaxRetries  int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    Multiplier  float64
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() RetryConfig {
    return RetryConfig{
        MaxRetries: 3,
        BaseDelay:  100 * time.Millisecond,
        MaxDelay:   10 * time.Second,
        Multiplier: 2.0,
    }
}

// Retry executes fn with exponential backoff
func Retry(cfg RetryConfig, fn func() error) error {
    var lastErr error
    delay := cfg.BaseDelay

    for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
        }

        if attempt < cfg.MaxRetries {
            // Add jitter
            jitter := time.Duration(rand.Int63n(int64(delay) / 2))
            time.Sleep(delay + jitter)

            // Exponential backoff
            delay = time.Duration(float64(delay) * cfg.Multiplier)
            if delay > cfg.MaxDelay {
                delay = cfg.MaxDelay
            }
        }
    }

    return lastErr
}
```

### Logging

```go
import (
    "fmt"
    "os"
)

// Logger writes to stderr (never stdout)
type Logger struct {
    debug bool
}

func NewLogger() *Logger {
    return &Logger{
        debug: os.Getenv("DEBUG") == "true",
    }
}

func (l *Logger) Debug(format string, args ...interface{}) {
    if l.debug {
        fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
    }
}

func (l *Logger) Info(format string, args ...interface{}) {
    fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
    fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}
```

## Testing

### Unit Tests

```go
package main

import (
    "testing"
)

func TestHandleHealth(t *testing.T) {
    resp := handleHealth()

    if !resp.Success {
        t.Error("expected success")
    }

    result, ok := resp.Result.(map[string]string)
    if !ok {
        t.Error("expected map result")
    }

    if result["status"] != "healthy" {
        t.Errorf("expected healthy, got %s", result["status"])
    }
}

func TestParseConfig(t *testing.T) {
    tests := []struct {
        name    string
        input   map[string]interface{}
        want    *Config
        wantErr bool
    }{
        {
            name: "valid config",
            input: map[string]interface{}{
                "api_key":  "test-key",
                "endpoint": "https://custom.api.com",
            },
            want: &Config{
                APIKey:   "test-key",
                Endpoint: "https://custom.api.com",
                Timeout:  30 * time.Second,
                Retries:  3,
            },
        },
        {
            name:    "missing api_key",
            input:   map[string]interface{}{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseConfig(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseConfig() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Tests

```go
// +build integration

package main

import (
    "bytes"
    "encoding/json"
    "os/exec"
    "testing"
)

func TestPluginIntegration(t *testing.T) {
    // Build plugin
    cmd := exec.Command("go", "build", "-o", "test-plugin", ".")
    if err := cmd.Run(); err != nil {
        t.Fatalf("build failed: %v", err)
    }

    // Test health
    input := `{"action":"health"}`
    cmd = exec.Command("./test-plugin")
    cmd.Stdin = bytes.NewBufferString(input)
    output, err := cmd.Output()
    if err != nil {
        t.Fatalf("health check failed: %v", err)
    }

    var resp Response
    if err := json.Unmarshal(output, &resp); err != nil {
        t.Fatalf("invalid response: %v", err)
    }

    if !resp.Success {
        t.Errorf("health check returned error: %s", resp.Error)
    }
}
```

## Build Configuration

### Cross-Compilation

```bash
# Build for all platforms
GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/plugin
GOOS=linux GOARCH=arm64 go build -o dist/linux-arm64/plugin
GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/plugin
GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/plugin
GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/plugin.exe
```

### Build Tags

```go
// +build !windows

package main

// Unix-specific implementation
func getPlatformConfig() string {
    return "/etc/specular/plugin.conf"
}
```

```go
// +build windows

package main

// Windows-specific implementation
func getPlatformConfig() string {
    return `C:\ProgramData\Specular\plugin.conf`
}
```

### Version Injection

```go
// Set via build flags
var Version = "dev"

// Build with:
// go build -ldflags "-X main.Version=1.0.0"
```

## Performance Tips

1. **Reuse connections** - Use `http.Client` with connection pooling
2. **Buffer I/O** - Use `bufio.Scanner` with appropriate buffer sizes
3. **Minimize allocations** - Reuse buffers and slices where possible
4. **Profile hotspots** - Use `pprof` to identify bottlenecks

```go
// Reuse buffer for encoding
var encodeBuf bytes.Buffer

func respond(resp Response) {
    encodeBuf.Reset()
    json.NewEncoder(&encodeBuf).Encode(resp)
    os.Stdout.Write(encodeBuf.Bytes())
}
```

## Common Pitfalls

1. **Writing to stdout incorrectly** - Only JSON responses go to stdout
2. **Forgetting to flush** - Responses must be written immediately
3. **Blocking forever** - Always use timeouts for external calls
4. **Ignoring context** - Pass context through for cancellation
5. **Large JSON parsing** - Set appropriate scanner buffer sizes
