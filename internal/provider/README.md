# AI Provider System

The AI Provider System is a pluggable architecture for integrating multiple AI providers (local models, API services, CLI tools) into specular. This system enables intelligent model selection, budget tracking, and provider management through a simple configuration file.

## Quick Start

```bash
# 1. Initialize provider configuration
specular provider init

# 2. Edit .specular/providers.yaml to enable desired providers
# 3. List configured providers
specular provider list

# 4. Check provider health
specular provider health
```

## Architecture

### Core Components

1. **ProviderClient Interface** (`interface.go`)
   - Universal interface that all providers must implement
   - Methods: Generate, Stream, GetCapabilities, GetInfo, IsAvailable, Health, Close
   - Supports streaming, tool calling, multi-turn conversations, and vision

2. **Provider Registry** (`registry.go`)
   - Thread-safe provider management
   - Auto-loading from YAML configuration
   - Lifecycle management (register, get, remove, close)

3. **Executable Provider** (`executable.go`)
   - Wraps any executable that speaks JSON over stdin/stdout
   - Supports both standard and streaming responses
   - Streaming mode reads newline-delimited JSON chunks
   - Process management with context cancellation

4. **Type Definitions** (`types.go`)
   - Request/response structures
   - Provider configuration format
   - Tool calling support

### Provider Types

The system supports four provider types:

- **CLI** (`ProviderTypeCLI`): Command-line executables (ollama, claude-cli)
- **API** (`ProviderTypeAPI`): Native Go HTTP API clients (OpenAI, Anthropic, Gemini) - ✅ **Implemented**
- **gRPC** (`ProviderTypeGRPC`): gRPC services - *Coming soon*
- **Native** (`ProviderTypeNative`): Go native plugins (.so files) - *Coming soon*

### Trust Levels

Providers have three trust levels for security:

- **Builtin** (`TrustLevelBuiltin`): Ships with specular, full trust
- **Verified** (`TrustLevelVerified`): From trusted sources with signed releases
- **Community** (`TrustLevelCommunity`): Unknown sources, run sandboxed

## Usage

## CLI Commands

The provider system comes with built-in CLI commands for management:

### Initialize Configuration

```bash
# Create providers.yaml from example
specular provider init

# Overwrite existing configuration
specular provider init --force
```

### List Providers

```bash
# List all configured providers
specular provider list

# Use custom config file
specular provider list --config path/to/providers.yaml
```

Example output:
```
NAME         TYPE   ENABLED   SOURCE    VERSION
----         ----   -------   ------    -------
ollama       cli    yes       local     1.0.0
openai       api    no        builtin   1.0.0
anthropic    api    no        builtin   1.0.0
gemini       api    no        builtin   1.0.0

Budget Constraints:
  Max cost per day: $20.00
  Max cost per request: $1.00

Provider Preference Order:
  1. ollama
  2. claude-cli
  3. anthropic
  4. openai
```

### Check Provider Health

Health checks verify that providers are accessible and properly configured. Each provider performs a lightweight test to ensure it can handle requests:

- **CLI Providers**: Executes health command to verify executable is working
- **API Providers**: Makes a simple API call to verify authentication and connectivity
  - OpenAI: Lists available models via `/v1/models` endpoint
  - Anthropic: Sends test generation request
  - Gemini: Sends minimal generation request with query parameter auth
- **gRPC Providers**: Performs health check RPC call (when implemented)

```bash
# Check all enabled providers
specular provider health

# Check specific provider(s)
specular provider health ollama
specular provider health openai anthropic

# Use custom config file
specular provider health --config path/to/providers.yaml
```

Example output:
```
PROVIDER    STATUS        MESSAGE
--------    ------        -------
ollama      ✅ HEALTHY     Executable provider: ./providers/ollama/ollama-provider
openai      ❌ UNHEALTHY   401 Unauthorized: Invalid API key
anthropic   ✅ HEALTHY     Anthropic API provider (claude-sonnet-4)
```

**Health check features:**
- 30-second timeout per provider
- Concurrent health checks for multiple providers
- Detailed error messages for troubleshooting
- Validates API keys, network connectivity, and service availability

## Configuration

### 1. Configure Providers

Create `.specular/providers.yaml` using the init command or by manually copying from the example:

```yaml
providers:
  - name: ollama
    type: cli
    enabled: true
    source: local
    config:
      path: ./providers/ollama/ollama-provider
      trust_level: community
      capabilities:
        streaming: false
        tools: false
        multi_turn: true
        max_context_tokens: 8192
    models:
      fast: llama3.2
      codegen: codellama
```

### 2. Load Providers

```go
import "github.com/felixgeelhaar/specular/internal/provider"

// Create registry
registry := provider.NewRegistry()

// Load from config
config := &provider.ProviderConfig{
    Name:    "ollama",
    Type:    provider.ProviderTypeCLI,
    Enabled: true,
    Config: map[string]interface{}{
        "path": "./providers/ollama/ollama-provider",
    },
}

err := registry.LoadFromConfig(config)
if err != nil {
    log.Fatal(err)
}
```

### 3. Use Providers

```go
// Get provider from registry
provider, err := registry.Get("ollama")
if err != nil {
    log.Fatal(err)
}

// Check health
ctx := context.Background()
if err := provider.Health(ctx); err != nil {
    log.Fatal("Provider unhealthy:", err)
}

// Generate text
req := &provider.GenerateRequest{
    Prompt:      "What is 2 + 2?",
    Temperature: 0.1,
    Config: map[string]interface{}{
        "model": "llama3.2",
    },
}

resp, err := provider.Generate(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Response: %s\n", resp.Content)
fmt.Printf("Tokens: %d (input: %d, output: %d)\n",
    resp.TokensUsed, resp.InputTokens, resp.OutputTokens)
fmt.Printf("Latency: %v\n", resp.Latency)
```

### 4. Multi-Turn Conversations

```go
// First message
resp1, _ := provider.Generate(ctx, &provider.GenerateRequest{
    Prompt: "What is 2 + 2?",
    Config: map[string]interface{}{"model": "llama3.2"},
})

// Follow-up with context
resp2, _ := provider.Generate(ctx, &provider.GenerateRequest{
    Prompt: "What about 3 + 3?",
    Context: []provider.Message{
        {Role: "user", Content: "What is 2 + 2?"},
        {Role: "assistant", Content: resp1.Content},
    },
    Config: map[string]interface{}{"model": "llama3.2"},
})
```

## Ollama Provider

The ollama provider (`providers/ollama/`) is a reference implementation that demonstrates how to wrap existing CLI tools with both standard and streaming support.

### How It Works

**Generate Mode** (non-streaming):
1. Reads `GenerateRequest` JSON from stdin
2. Converts to ollama API format with `Stream: false`
3. Calls ollama HTTP API at `http://localhost:11434/api/generate`
4. Parses complete response and converts to `GenerateResponse`
5. Writes JSON to stdout

**Stream Mode** (streaming):
1. Reads `GenerateRequest` JSON from stdin
2. Converts to ollama API format with `Stream: true`
3. Calls ollama HTTP API with streaming enabled
4. Reads newline-delimited JSON responses from API
5. Outputs `StreamChunk` JSON for each delta (newline-delimited)
6. Final chunk includes `done: true` and token count

### Building

```bash
go build -o providers/ollama/ollama-provider ./providers/ollama/
```

### Manual Testing

```bash
# Health check
./providers/ollama/ollama-provider health

# Generate (non-streaming)
echo '{"prompt": "What is 2+2?", "config": {"model": "llama3.2"}}' | \
  ./providers/ollama/ollama-provider generate

# Stream (streaming with real-time output)
echo '{"prompt": "Count from 1 to 5", "config": {"model": "llama3.2"}}' | \
  ./providers/ollama/ollama-provider stream
```

**Example streaming output**:
```json
{"content":"Here","delta":"Here","done":false,"timestamp":"2025-11-05T13:50:21.4Z"}
{"content":"Here we go","delta":" we go","done":false,"timestamp":"2025-11-05T13:50:21.5Z"}
{"content":"Here we go:\n\n1, 2, 3, 4, 5.","delta":"","done":true,"tokens_used":51,"timestamp":"2025-11-05T13:50:22.5Z"}
```

## OpenAI and Anthropic API Providers

The OpenAI and Anthropic providers are native Go HTTP clients that implement the ProviderClient interface directly. Unlike CLI providers, they don't require external executables.

### OpenAI Provider (`openai.go`)

**Features:**
- Native HTTP client using `net/http` (no external dependencies)
- Chat Completions API (`/chat/completions`)
- Streaming support with Server-Sent Events (SSE)
- Bearer token authentication
- System prompts as first message in messages array
- Full error handling with OpenAI error messages
- Supports temperature, max_tokens, top_p

**Configuration:**
```yaml
providers:
  - name: openai
    type: api
    enabled: true
    config:
      api_key: ${OPENAI_API_KEY}
      base_url: https://api.openai.com/v1
      capabilities:
        streaming: true
        tools: true
        multi_turn: true
        max_context_tokens: 128000
    models:
      fast: gpt-4o-mini
      codegen: gpt-4o
      long-context: gpt-4-turbo
```

**Usage:**
```go
// Registry automatically creates OpenAI provider when name="openai"
provider, _ := registry.Get("openai")

resp, err := provider.Generate(ctx, &GenerateRequest{
    Prompt:      "What is Go?",
    Temperature: 0.7,
    MaxTokens:   500,
})
```

### Anthropic Provider (`anthropic.go`)

**Features:**
- Native HTTP client using `net/http` (no external dependencies)
- Messages API (`/messages`)
- Streaming support with event-based SSE
- x-api-key authentication with anthropic-version header
- System prompt as separate field (Claude-specific)
- Vision capability support
- 200K context window
- Event-based streaming (content_block_delta, message_stop)

**Configuration:**
```yaml
providers:
  - name: anthropic
    type: api
    enabled: true
    config:
      api_key: ${ANTHROPIC_API_KEY}
      base_url: https://api.anthropic.com/v1
      capabilities:
        streaming: true
        tools: true
        multi_turn: true
        vision: true
        max_context_tokens: 200000
    models:
      fast: claude-haiku-3.5
      codegen: claude-sonnet-3.5
      agentic: claude-sonnet-4
      long-context: claude-sonnet-3.5
```

**Usage:**
```go
// Registry automatically creates Anthropic provider when name="anthropic"
provider, _ := registry.Get("anthropic")

resp, err := provider.Generate(ctx, &GenerateRequest{
    Prompt:       "Explain microservices",
    SystemPrompt: "You are a software architect",
    Temperature:  0.7,
    MaxTokens:    1000,
})
```

### Gemini Provider (`gemini.go`)

**Features:**
- Native HTTP client using `net/http` (no external dependencies)
- Generate Content API (`/generateContent` and `/streamGenerateContent`)
- Streaming support with Server-Sent Events (SSE)
- Query parameter authentication (`?key=API_KEY`)
- System instruction as separate field
- Vision capability support
- 1M context window (largest among supported providers)
- Contents/Parts structure for messages
- Role mapping: "assistant" → "model"

**Configuration:**
```yaml
providers:
  - name: gemini
    type: api
    enabled: true
    config:
      api_key: ${GEMINI_API_KEY}
      base_url: https://generativelanguage.googleapis.com/v1beta
      capabilities:
        streaming: true
        tools: true
        multi_turn: true
        vision: true
        max_context_tokens: 1000000
    models:
      fast: gemini-2.0-flash-exp
      codegen: gemini-2.0-flash-exp
      agentic: gemini-2.5-pro-exp-03
      long-context: gemini-2.5-pro-exp-03
```

**Usage:**
```go
// Registry automatically creates Gemini provider when name="gemini"
provider, _ := registry.Get("gemini")

resp, err := provider.Generate(ctx, &GenerateRequest{
    Prompt:       "Explain distributed systems",
    SystemPrompt: "You are an expert systems architect",
    Temperature:  0.7,
    MaxTokens:    2000,
})
```

### Key Differences Between OpenAI, Anthropic, and Gemini

| Feature | OpenAI | Anthropic | Gemini |
|---------|--------|-----------|--------|
| Authentication | `Authorization: Bearer {key}` | `x-api-key: {key}` + `anthropic-version` | Query param `?key={key}` |
| System Prompt | First message in array | Separate `system` field | Separate `systemInstruction` field |
| Role Name | "assistant" | "assistant" | "model" (mapped from "assistant") |
| Message Structure | Messages array | Messages array | Contents array with parts |
| Streaming Format | Simple data chunks | Event-based (content_block_delta) | SSE with `alt=sse` parameter |
| Stream End Marker | `data: [DONE]` | `event: message_stop` | `finishReason` in chunk |
| Context Window | 128K tokens | 200K tokens | 1M tokens |
| Vision Support | Partial | Full support | Full support |

### Testing API Providers

All three API providers have comprehensive test suites using `httptest` mock servers:

```bash
# Run OpenAI provider tests
go test -v -run TestOpenAI ./internal/provider

# Run Anthropic provider tests
go test -v -run TestAnthropic ./internal/provider

# Run Gemini provider tests
go test -v -run TestGemini ./internal/provider

# Run all API provider integration tests
go test -v -run "TestIntegration_(OpenAI|Anthropic|Gemini)" ./internal/provider
```

## Creating Custom Providers

### Executable Provider (Easiest)

Create a program that:

1. Accepts commands: `generate`, `stream`, `health`
2. For `generate`:
   - Reads JSON from stdin (GenerateRequest format)
   - Calls your AI service
   - Writes JSON to stdout (GenerateResponse format)
3. For `stream` (optional):
   - Reads JSON from stdin (GenerateRequest format)
   - Calls your AI service with streaming enabled
   - Writes newline-delimited JSON to stdout (StreamChunk format)
   - Each line contains: content, delta, done, tokens_used, timestamp
4. For `health`:
   - Returns exit code 0 if healthy
   - Prints error message to stderr if unhealthy

Example in any language:

```python
#!/usr/bin/env python3
import json
import sys
from datetime import datetime

def generate():
    # Read request from stdin
    req = json.load(sys.stdin)

    # Call your AI service
    response = your_ai_service.generate(req['prompt'])

    # Write response to stdout
    result = {
        'content': response.text,
        'tokens_used': response.tokens,
        'model': 'your-model',
        'latency': response.latency_ms * 1000000,  # nanoseconds
        'finish_reason': 'stop',
        'provider': 'your-provider'
    }
    json.dump(result, sys.stdout)

def stream():
    # Read request from stdin
    req = json.load(sys.stdin)

    # Stream from your AI service
    full_content = ""
    for chunk in your_ai_service.stream(req['prompt']):
        full_content += chunk.text

        # Output StreamChunk as newline-delimited JSON
        result = {
            'content': full_content,
            'delta': chunk.text,
            'done': chunk.is_final,
            'tokens_used': chunk.tokens if chunk.is_final else 0,
            'timestamp': datetime.utcnow().isoformat() + 'Z'
        }
        print(json.dumps(result))

        if chunk.is_final:
            break

if __name__ == '__main__':
    if sys.argv[1] == 'generate':
        generate()
    elif sys.argv[1] == 'stream':
        stream()
    elif sys.argv[1] == 'health':
        # Check if service is available
        sys.exit(0 if your_ai_service.is_available() else 1)
```

### Go Native Provider (Advanced)

Implement the `ProviderClient` interface directly in Go:

```go
type MyProvider struct {
    client *YourSDK.Client
    config *ProviderConfig
}

func (p *MyProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
    // Implement using your SDK
    return &GenerateResponse{...}, nil
}

// Implement other ProviderClient methods...
```

## Testing

```bash
# Run all provider tests
go test -v ./internal/provider/...

# Run only integration tests (requires ollama)
go test -v ./internal/provider/... -run TestIntegration

# Run with race detection
go test -race ./internal/provider/...
```

## Future Enhancements

### Phase 1: API Providers ✅ **COMPLETED**
- [x] Implement OpenAI API client (native Go HTTP client)
- [x] Implement Anthropic Claude API client (native Go HTTP client)
- [x] Implement Google Gemini API client (native Go HTTP client)
- [x] Streaming support with Server-Sent Events for all API providers
- [x] Comprehensive test suites with mock HTTP servers
- [x] Integration tests for all three provider types

### Phase 2: CLI Providers (Week 2-3)
- [ ] Claude CLI wrapper
- [ ] Gemini CLI wrapper
- [ ] Auto-detect installed CLI tools

### Phase 3: Security & Sandboxing (Week 3-4)
- [ ] Docker isolation for community providers
- [ ] Permission system (filesystem, network access)
- [ ] Code signing verification for verified providers
- [ ] Sandboxed execution environment

### Phase 4: Marketplace (Week 4-5)
- [ ] Provider discovery and installation
- [ ] Version management
- [ ] Dependency resolution
- [ ] Update notifications

### Phase 5: Integration (Week 5-6)
- [ ] Integrate with router package
- [ ] Use providers in interview mode
- [ ] Use providers for PRD parsing
- [ ] Use providers for code generation

## Files

```
internal/provider/
├── README.md                  # This file
├── interface.go               # ProviderClient interface and core types
├── types.go                   # Request/response structures
├── registry.go                # Provider registry and management
├── executable.go              # Executable (CLI) provider implementation
├── openai.go                  # OpenAI API provider (native Go HTTP client)
├── openai_test.go             # OpenAI provider tests with mock server
├── anthropic.go               # Anthropic API provider (native Go HTTP client)
├── anthropic_test.go          # Anthropic provider tests with mock server
├── gemini.go                  # Google Gemini API provider (native Go HTTP client)
├── gemini_test.go             # Gemini provider tests with mock server
└── integration_test.go        # Integration tests for all provider types

providers/
└── ollama/
    ├── main.go                # Ollama wrapper implementation
    └── ollama-provider        # Built executable

.specular/
└── providers.yaml             # Provider configuration (includes Gemini)
```

## Architecture Decisions

### Why Executable Providers?

1. **Language Agnostic**: Write providers in any language
2. **Sandboxing**: Easy to isolate with Docker/containers
3. **Zero Dependencies**: No SDK version conflicts
4. **Simple Protocol**: JSON over stdin/stdout is universal
5. **Debugging**: Easy to test manually with echo/curl

### Why JSON Protocol?

1. **Universal**: Supported by every language
2. **Self-Documenting**: Easy to understand request/response
3. **Extensible**: Add fields without breaking compatibility
4. **Debuggable**: Human-readable for development

### Why Registry Pattern?

1. **Centralized Management**: Single source of truth
2. **Lifecycle Control**: Proper initialization and cleanup
3. **Thread Safety**: Concurrent access with mutex
4. **Lazy Loading**: Load providers on demand
5. **Hot Reload**: Replace providers without restart (future)

## Contributing

When adding new provider types:

1. Implement the `ProviderClient` interface
2. Add factory method to `registry.go`
3. Add configuration fields to `ProviderConfig`
4. Write integration tests
5. Update this README with usage examples
6. Add to `.specular/providers.yaml.example`
