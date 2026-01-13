# Getting Started with Plugin Development

This tutorial walks you through creating your first Specular plugin.

## Prerequisites

- Specular CLI installed (v1.6.0+)
- One of the supported languages: Go, Python, Node.js, or Shell
- Basic understanding of JSON

## Step 1: Choose Your Plugin Type

Before creating a plugin, decide what type best fits your use case:

| If you want to... | Use type |
|-------------------|----------|
| Add a new AI provider | `provider` |
| Enforce policies/rules | `validator` |
| Create custom output formats | `formatter` |
| React to Specular events | `hook` |
| Send notifications | `notifier` |

For this tutorial, we'll create a **hook** plugin that logs events.

## Step 2: Create the Plugin Scaffold

Use the built-in scaffolding command:

```bash
specular plugin create event-logger \
  --type hook \
  --lang go \
  --author "Your Name"
```

This creates:

```
event-logger/
├── plugin.yaml     # Plugin manifest
├── main.go         # Plugin entry point
└── go.mod          # Go module file
```

## Step 3: Understand the Manifest

Open `plugin.yaml`:

```yaml
name: event-logger
version: "1.0.0"
type: hook
description: "Event logging plugin"
author: "Your Name"
license: "MIT"
entrypoint: "./event-logger"

capabilities:
  - hook

config: []
```

### Manifest Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique plugin identifier |
| `version` | Yes | Semantic version (e.g., "1.0.0") |
| `type` | Yes | Plugin type |
| `description` | Yes | Short description |
| `author` | Yes | Author name |
| `license` | No | SPDX license ID |
| `entrypoint` | Yes | Executable path |
| `capabilities` | No | Plugin capabilities |
| `config` | No | Configuration schema |

## Step 4: Implement the Plugin

The generated `main.go` looks like this:

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
)

const Version = "1.0.0"

type PluginRequest struct {
    Action string                 `json:"action"`
    Data   map[string]interface{} `json:"data,omitempty"`
    Config map[string]interface{} `json:"config,omitempty"`
}

type PluginResponse struct {
    Success bool        `json:"success"`
    Result  interface{} `json:"result,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func main() {
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        var req PluginRequest
        if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
            respond(PluginResponse{
                Success: false,
                Error:   fmt.Sprintf("invalid JSON: %v", err),
            })
            continue
        }

        switch req.Action {
        case "health":
            respond(PluginResponse{
                Success: true,
                Result: map[string]string{
                    "status":  "healthy",
                    "version": Version,
                },
            })
        case "pre", "post", "execute":
            result, err := executeHook(req)
            if err != nil {
                respond(PluginResponse{Success: false, Error: err.Error()})
            } else {
                respond(PluginResponse{Success: true, Result: result})
            }
        default:
            respond(PluginResponse{
                Success: false,
                Error:   fmt.Sprintf("unknown action: %s", req.Action),
            })
        }
    }
}

func executeHook(req PluginRequest) (interface{}, error) {
    event := req.Data["event"].(string)
    // Your logic here
    return map[string]interface{}{
        "executed": true,
        "event":    event,
    }, nil
}

func respond(resp PluginResponse) {
    json.NewEncoder(os.Stdout).Encode(resp)
}
```

### Customize the Hook Logic

Let's add event logging:

```go
import (
    "log"
    "os"
    "time"
)

func executeHook(req PluginRequest) (interface{}, error) {
    event, _ := req.Data["event"].(string)

    // Get log file from config or use default
    logFile := "/tmp/specular-events.log"
    if cf, ok := req.Config["log_file"].(string); ok {
        logFile = cf
    }

    // Open log file
    f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    // Write event
    logger := log.New(f, "", 0)
    logger.Printf("[%s] Event: %s - Data: %v\n",
        time.Now().Format(time.RFC3339),
        event,
        req.Data,
    )

    return map[string]interface{}{
        "executed": true,
        "logged":   logFile,
    }, nil
}
```

### Add Configuration Schema

Update `plugin.yaml` to declare the configuration:

```yaml
name: event-logger
version: "1.0.0"
type: hook
description: "Logs Specular events to a file"
author: "Your Name"
license: "MIT"
entrypoint: "./event-logger"

capabilities:
  - hook

config:
  - name: log_file
    type: string
    description: "Path to the log file"
    required: false
    default: "/tmp/specular-events.log"
```

## Step 5: Build the Plugin

For Go plugins:

```bash
cd event-logger
go build -o event-logger
```

For Python/Node.js/Shell, no build step is needed.

## Step 6: Test Locally

Test the health check:

```bash
echo '{"action":"health"}' | ./event-logger
# Output: {"success":true,"result":{"status":"healthy","version":"1.0.0"}}
```

Test the hook:

```bash
echo '{"action":"execute","data":{"event":"test"}}' | ./event-logger
# Output: {"success":true,"result":{"executed":true,"logged":"/tmp/specular-events.log"}}
```

## Step 7: Install the Plugin

```bash
specular plugin install ./event-logger
```

Verify installation:

```bash
specular plugin list
specular plugin info event-logger
specular plugin health event-logger
```

## Step 8: Configure and Use

Add configuration to your `.specular.yaml`:

```yaml
plugins:
  event-logger:
    log_file: "/var/log/specular/events.log"
```

The plugin will now be invoked for relevant events.

## Next Steps

- Read the [Protocol Reference](protocol-reference.md) for complete API details
- Follow [Best Practices](best-practices.md) for production-ready plugins
- Check [Language Guides](language-guides/) for language-specific tips
- Learn how to [Publish](publishing.md) your plugin to the registry

## Troubleshooting

### Plugin not found after installation

Check the plugin directory:

```bash
specular plugin info event-logger
```

Ensure the entrypoint is executable:

```bash
ls -la ~/.specular/plugins/event-logger/
chmod +x ~/.specular/plugins/event-logger/event-logger
```

### Health check fails

Test directly:

```bash
echo '{"action":"health"}' | ~/.specular/plugins/event-logger/event-logger
```

Check for:
- Valid JSON output
- `success: true` in response
- Correct version in response

### Plugin crashes

Enable debug logging:

```bash
SPECULAR_LOG_LEVEL=debug specular plugin health event-logger
```

Check plugin logs and stderr output.
