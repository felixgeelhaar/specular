# Plugin Protocol Reference

Specular plugins communicate with the host via JSON messages over stdin/stdout. This document details the complete protocol specification.

## Communication Model

```
┌──────────────┐     stdin (JSON)      ┌──────────────┐
│              │ ──────────────────────▶│              │
│   Specular   │                       │    Plugin    │
│     CLI      │ ◀─────────────────────│              │
└──────────────┘    stdout (JSON)      └──────────────┘
```

- **Input:** Specular sends JSON requests via stdin (one per line)
- **Output:** Plugin responds with JSON via stdout (one per line)
- **Errors:** Plugin errors go to stderr (logged by Specular)

## Request Format

All requests follow this structure:

```json
{
  "action": "string",
  "params": {},
  "data": {},
  "config": {}
}
```

| Field | Type | Description |
|-------|------|-------------|
| `action` | `string` | The operation to perform (required) |
| `params` | `object` | Action-specific parameters |
| `data` | `object` | Input data for the action |
| `config` | `object` | Plugin configuration from manifest |

## Response Format

All responses must follow this structure:

```json
{
  "success": true,
  "result": {},
  "error": ""
}
```

| Field | Type | Description |
|-------|------|-------------|
| `success` | `boolean` | Whether the action succeeded (required) |
| `result` | `any` | Action result (when success is true) |
| `error` | `string` | Error message (when success is false) |

## Common Actions

### Health Check

All plugins must implement the `health` action.

**Request:**
```json
{"action": "health"}
```

**Response:**
```json
{
  "success": true,
  "result": {
    "status": "healthy",
    "version": "1.0.0",
    "name": "my-plugin"
  }
}
```

**Status values:**
- `healthy` - Plugin is functioning normally
- `degraded` - Plugin is functional but has issues
- `unhealthy` - Plugin is not functioning

## Type-Specific Actions

### Provider Plugins

#### Generate

**Request:**
```json
{
  "action": "generate",
  "params": {
    "model": "gpt-4",
    "max_tokens": 1000,
    "temperature": 0.7
  },
  "data": {
    "prompt": "Generate a summary of...",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Hello!"}
    ]
  },
  "config": {
    "api_key": "sk-...",
    "api_endpoint": "https://api.example.com"
  }
}
```

**Response:**
```json
{
  "success": true,
  "result": {
    "content": "Generated text...",
    "model": "gpt-4",
    "usage": {
      "prompt_tokens": 50,
      "completion_tokens": 100,
      "total_tokens": 150
    },
    "finish_reason": "stop"
  }
}
```

### Validator Plugins

#### Validate

**Request:**
```json
{
  "action": "validate",
  "data": {
    "content": "Content to validate...",
    "context": {
      "file": "example.txt",
      "type": "document"
    }
  },
  "params": {
    "rules": ["rule1", "rule2"],
    "severity_threshold": "warning"
  },
  "config": {
    "policy_file": ".policy.yaml"
  }
}
```

**Response:**
```json
{
  "success": true,
  "result": {
    "valid": false,
    "messages": [
      {
        "severity": "error",
        "message": "Content exceeds maximum length",
        "rule": "max_length",
        "line": 42,
        "column": 1
      },
      {
        "severity": "warning",
        "message": "Consider adding more context",
        "rule": "clarity"
      }
    ]
  }
}
```

**Severity levels:**
- `error` - Validation failure, blocks processing
- `warning` - Potential issue, does not block
- `info` - Informational message

### Formatter Plugins

#### Format

**Request:**
```json
{
  "action": "format",
  "data": {
    "content": "Raw content to format...",
    "metadata": {
      "title": "Report Title",
      "date": "2024-01-15"
    }
  },
  "params": {
    "format": "html",
    "options": {
      "theme": "dark",
      "include_toc": true
    }
  },
  "config": {
    "template_dir": "./templates"
  }
}
```

**Response:**
```json
{
  "success": true,
  "result": {
    "output": "<html>...</html>",
    "format": "html",
    "metadata": {
      "size": 1234,
      "generated_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Hook Plugins

#### Pre/Post/Execute

**Request:**
```json
{
  "action": "pre",
  "data": {
    "event": "generate.start",
    "payload": {
      "prompt": "...",
      "model": "gpt-4"
    },
    "context": {
      "session_id": "abc123",
      "user": "john"
    }
  },
  "config": {
    "log_level": "info"
  }
}
```

**Response:**
```json
{
  "success": true,
  "result": {
    "executed": true,
    "modified": false,
    "data": {}
  }
}
```

**Hook actions:**
- `pre` - Before the event occurs (can modify data)
- `post` - After the event occurs
- `execute` - Generic hook execution

**Hook events:**
- `generate.start` / `generate.end`
- `validate.start` / `validate.end`
- `format.start` / `format.end`
- `session.start` / `session.end`

### Notifier Plugins

#### Notify

**Request:**
```json
{
  "action": "notify",
  "data": {
    "event": "generation.complete",
    "title": "Generation Completed",
    "message": "Successfully generated response",
    "level": "info",
    "metadata": {
      "model": "gpt-4",
      "tokens": 150,
      "duration_ms": 1200
    }
  },
  "config": {
    "webhook_url": "https://hooks.slack.com/...",
    "channel": "#alerts"
  }
}
```

**Response:**
```json
{
  "success": true,
  "result": {
    "delivered": true,
    "message_id": "msg_123",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

**Notification levels:**
- `info` - Informational
- `warning` - Warning
- `error` - Error
- `success` - Success

## Error Handling

### Error Response Format

```json
{
  "success": false,
  "error": "Descriptive error message"
}
```

### Common Errors

| Error | Description | Resolution |
|-------|-------------|------------|
| `invalid JSON` | Malformed request | Fix JSON syntax |
| `unknown action` | Unrecognized action | Check action spelling |
| `missing required field` | Required field not provided | Add missing field |
| `configuration error` | Invalid config | Check plugin config |
| `timeout` | Operation took too long | Increase timeout or optimize |

### Error Categories

Prefix error messages with a category for easier debugging:

```json
{"success": false, "error": "config: missing required field 'api_key'"}
{"success": false, "error": "network: connection refused to api.example.com"}
{"success": false, "error": "validation: content exceeds maximum size"}
```

## Streaming Responses

For long-running operations, plugins can stream responses:

```json
{"type": "chunk", "data": "Partial content..."}
{"type": "chunk", "data": "More content..."}
{"type": "done", "result": {"total_chunks": 2}}
```

**Stream message types:**
- `chunk` - Partial result
- `progress` - Progress update
- `done` - Stream complete
- `error` - Stream error

## Timeouts

| Operation | Default Timeout | Configurable |
|-----------|-----------------|--------------|
| Health check | 5s | No |
| Generate | 120s | Yes |
| Validate | 30s | Yes |
| Format | 30s | Yes |
| Hook | 10s | Yes |
| Notify | 15s | Yes |

Set custom timeouts via environment variable:

```bash
SPECULAR_PLUGIN_TIMEOUT=60s specular generate
```

## Implementation Examples

### Go

```go
package main

import (
    "bufio"
    "encoding/json"
    "os"
)

type Request struct {
    Action string                 `json:"action"`
    Params map[string]interface{} `json:"params,omitempty"`
    Data   map[string]interface{} `json:"data,omitempty"`
    Config map[string]interface{} `json:"config,omitempty"`
}

type Response struct {
    Success bool        `json:"success"`
    Result  interface{} `json:"result,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func main() {
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        var req Request
        if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
            respond(Response{Success: false, Error: err.Error()})
            continue
        }

        result, err := handleRequest(req)
        if err != nil {
            respond(Response{Success: false, Error: err.Error()})
        } else {
            respond(Response{Success: true, Result: result})
        }
    }
}

func respond(resp Response) {
    json.NewEncoder(os.Stdout).Encode(resp)
}
```

### Python

```python
import json
import sys

def main():
    for line in sys.stdin:
        try:
            request = json.loads(line.strip())
            result = handle_request(request)
            print(json.dumps({"success": True, "result": result}))
        except Exception as e:
            print(json.dumps({"success": False, "error": str(e)}))
        sys.stdout.flush()

if __name__ == "__main__":
    main()
```

### Node.js

```javascript
const readline = require('readline');

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  terminal: false
});

rl.on('line', async (line) => {
  try {
    const request = JSON.parse(line);
    const result = await handleRequest(request);
    console.log(JSON.stringify({ success: true, result }));
  } catch (error) {
    console.log(JSON.stringify({ success: false, error: error.message }));
  }
});
```

### Shell (Bash)

```bash
#!/bin/bash
INPUT=$(cat)
ACTION=$(echo "$INPUT" | jq -r '.action // "unknown"')

case "$ACTION" in
  "health")
    echo '{"success":true,"result":{"status":"healthy","version":"1.0.0"}}'
    ;;
  *)
    echo "{\"success\":false,\"error\":\"Unknown action: $ACTION\"}"
    ;;
esac
```

## Best Practices

1. **Always respond** - Every request must have a response
2. **Use proper JSON** - Validate JSON before sending
3. **Handle errors gracefully** - Never crash, always return error response
4. **Flush output** - Ensure stdout is flushed after each response
5. **Log to stderr** - Use stderr for debugging, never stdout
6. **Implement health** - Always implement the health check action
7. **Validate input** - Check required fields before processing
8. **Set timeouts** - Don't block indefinitely on external calls
