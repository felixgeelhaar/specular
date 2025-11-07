# CLI Provider Protocol

This document describes the CLI provider protocol that allows external executables to be used as AI providers in the system.

## Overview

CLI providers are standalone executables that implement a simple JSON-based protocol over stdin/stdout. The `ExecutableProvider` adapter wraps these executables and integrates them with the internal provider system.

### Benefits

- **Language Agnostic**: Write providers in any language
- **Process Isolation**: Each provider runs in its own process
- **Simple Protocol**: JSON-based request/response over stdin/stdout
- **Hot Reload**: Update providers without rebuilding the main application
- **Easy Testing**: Test providers independently with JSON files

## Protocol Specification

### Commands

All CLI providers must support these three subcommands:

1. **`generate`** - Synchronous text generation
2. **`stream`** - Streaming text generation
3. **`health`** - Health check

### Command Format

```bash
<provider-executable> <command>
```

Examples:
```bash
./ollama-provider generate
./ollama-provider stream
./ollama-provider health
```

## Command Details

### 1. Generate Command

**Purpose**: Synchronous text generation with full response

**Input**: JSON request via stdin
```json
{
  "prompt": "What is 2+2?",
  "system_prompt": "You are a helpful assistant.",
  "max_tokens": 1000,
  "temperature": 0.7,
  "top_p": 0.9,
  "tools": [],
  "context": [
    {"role": "user", "content": "Previous message"},
    {"role": "assistant", "content": "Previous response"}
  ],
  "config": {
    "model": "llama3.2:latest",
    "custom_option": "value"
  },
  "metadata": {
    "task_id": "abc123",
    "user_id": "user456"
  }
}
```

**Output**: JSON response via stdout
```json
{
  "content": "2 + 2 = 4.",
  "tokens_used": 41,
  "input_tokens": 32,
  "output_tokens": 9,
  "model": "llama3.2:latest",
  "latency": 4364650667,
  "finish_reason": "stop",
  "error": "",
  "provider": "ollama"
}
```

**Fields**:

**Request**:
- `prompt` (string, required): The user's prompt
- `system_prompt` (string, optional): System instructions
- `max_tokens` (int, optional): Maximum tokens to generate
- `temperature` (float, optional): Sampling temperature (0-1)
- `top_p` (float, optional): Nucleus sampling parameter
- `tools` (array, optional): Available tools/functions
- `context` (array, optional): Conversation history
- `config` (object, optional): Provider-specific configuration
- `metadata` (object, optional): Request metadata

**Response**:
- `content` (string, required): Generated text
- `tokens_used` (int, required): Total tokens consumed
- `input_tokens` (int, optional): Prompt tokens
- `output_tokens` (int, optional): Generated tokens
- `model` (string, required): Model used
- `latency` (duration, required): Generation time in nanoseconds
- `finish_reason` (string, required): Completion reason ("stop", "length", "tool_use")
- `error` (string, optional): Error message if failed
- `provider` (string, required): Provider name

### 2. Stream Command

**Purpose**: Streaming text generation with incremental updates

**Input**: Same as generate command

**Output**: Newline-delimited JSON (NDJSON) chunks via stdout
```json
{"content":"2","delta":"2","done":false,"tokens_used":0,"error":"","timestamp":"2025-01-07T10:30:00Z"}
{"content":"2 +","delta":" +","done":false,"tokens_used":0,"error":"","timestamp":"2025-01-07T10:30:00Z"}
{"content":"2 + 2","delta":" 2","done":false,"tokens_used":0,"error":"","timestamp":"2025-01-07T10:30:00Z"}
{"content":"2 + 2 = 4.","delta":" = 4.","done":true,"tokens_used":41,"error":"","timestamp":"2025-01-07T10:30:00Z"}
```

**Chunk Fields**:
- `content` (string, required): Accumulated content so far
- `delta` (string, required): New content in this chunk
- `done` (bool, required): Whether generation is complete
- `tokens_used` (int, optional): Total tokens (only in final chunk)
- `error` (string, optional): Error message if failed
- `timestamp` (timestamp, required): Chunk timestamp

### 3. Health Command

**Purpose**: Check if provider is available and working

**Input**: None (no stdin)

**Output**: Exit code and optional message
- Exit code 0: Provider healthy
- Exit code non-zero: Provider unhealthy

**Success Example**:
```bash
$ ./ollama-provider health
OK
$ echo $?
0
```

**Failure Example**:
```bash
$ ./ollama-provider health
Error: ollama not available: exec: "ollama": executable file not found
$ echo $?
1
```

## Reference Implementation

See `providers/ollama/main.go` for a complete reference implementation that:

- Uses Ollama HTTP API (`http://localhost:11434/api/generate`)
- Implements all three commands (generate, stream, health)
- Handles conversation context
- Properly tracks token usage
- Implements streaming with NDJSON
- Includes error handling and validation

## Configuration

### Router Configuration

Add your CLI provider to `.specular/router.yaml`:

```yaml
providers:
  # CLI provider example
  - name: ollama
    type: cli
    enabled: true
    priority: 50
    config:
      path: /path/to/ollama-provider  # Absolute or relative path
      args: []                        # Optional additional arguments
      timeout: 120s                   # Request timeout
      model: llama3.2:latest         # Default model

    # Model hints routing
    models:
      fast: llama3.2:latest          # Fast, simple tasks
      capable: qwen3:8b              # Complex reasoning
      codegen: deepseek-coder:6.7b   # Code generation
```

For complete router configuration examples, see `.specular/router.example.yaml`.

## Testing Your Provider

### Generate Test
```bash
echo '{"prompt":"Hello, world!"}' | ./my-provider generate
```

### Health Test
```bash
./my-provider health
echo $?  # Should be 0
```

### Stream Test
```bash
echo '{"prompt":"Count to 5"}' | ./my-provider stream
```

## Next Steps

1. **Read the Ollama Provider**: Study `providers/ollama/main.go` as reference
2. **Check Router Config**: See `.specular/router.example.yaml` for configuration examples
3. **Build Your Provider**: Follow the protocol specification
4. **Test Thoroughly**: Use the testing checklist above
5. **Configure Router**: Add to `.specular/router.yaml`

## Additional Resources

- **ExecutableProvider Code**: `internal/provider/executable.go`
- **Provider Interface**: `internal/provider/interface.go`
- **Type Definitions**: `internal/provider/types.go`
- **Router Configuration**: `.specular/router.example.yaml`
- **Example Ollama Provider**: `providers/ollama/main.go`
