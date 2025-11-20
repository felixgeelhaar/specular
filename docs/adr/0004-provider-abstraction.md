# ADR 0004: Provider Abstraction for Multi-LLM Support

**Status:** Accepted

**Date:** 2025-01-07

**Decision Makers:** Specular Core Team

## Context

Specular relies on Large Language Models (LLMs) for code generation, planning, and analysis. The LLM landscape is rapidly evolving with multiple providers (Anthropic, OpenAI, Google, Meta) offering different capabilities, pricing, and specializations.

### Requirements
1. **Provider Flexibility**: Support multiple LLM providers
2. **Model Selection**: Allow users to choose specific models (e.g., GPT-4, Claude Sonnet)
3. **Fallback Strategy**: Gracefully handle provider failures
4. **Cost Optimization**: Route tasks to cost-effective models
5. **Model Hints**: Optimize for speed, quality, or code generation
6. **Future-Proof**: Easy to add new providers

### Challenges
- Different API interfaces (OpenAI REST, Anthropic Messages, Google Vertex AI)
- Varying capabilities (context length, streaming, function calling)
- Authentication mechanisms (API keys, OAuth, service accounts)
- Rate limits and quotas
- Cost models (per-token vs. per-request)
- Model naming conventions

## Decision

**We will implement a provider abstraction layer with a routing system that intelligently selects models based on task requirements and model hints.**

### Architecture

```
┌─────────────────────────────────────────┐
│          Specular Core                  │
│  (Plan, Build, Eval, Interview)         │
└────────────┬────────────────────────────┘
             │
             │ GenerateOptions (prompt, model_hint, max_tokens)
             v
┌─────────────────────────────────────────┐
│          Router (Model Selection)       │
│  - Resolve model_hint to specific model │
│  - Apply fallback strategy              │
│  - Handle provider health checks        │
└────────────┬────────────────────────────┘
             │
             │ Selected provider + model
             v
┌─────────────────────────────────────────┐
│       Provider Interface                │
│  type Provider interface {              │
│    Generate(opts GenerateOptions)       │
│    StreamGenerate(opts StreamOptions)   │
│    Health() error                       │
│  }                                      │
└─────┬─────────┬─────────┬───────────────┘
      │         │         │
      v         v         v
 ┌────────┐ ┌────────┐ ┌────────┐
 │Anthropic│ │ OpenAI │ │ Google │
 │Provider │ │Provider│ │Provider│
 └────────┘ └────────┘ └────────┘
```

## Implementation Details

### Provider Interface
```go
type Provider interface {
    // Generate produces text from a prompt
    Generate(ctx context.Context, opts GenerateOptions) (*Response, error)

    // StreamGenerate streams responses for long outputs
    StreamGenerate(ctx context.Context, opts StreamOptions) (<-chan Chunk, error)

    // Health checks if provider is available
    Health(ctx context.Context) error

    // Name returns the provider name
    Name() string

    // SupportedModels returns available models
    SupportedModels() []ModelInfo
}

type GenerateOptions struct {
    Prompt      string
    ModelHint   string  // "fast", "balanced", "quality", "codegen"
    MaxTokens   int
    Temperature float64
    SystemPrompt string
}

type Response struct {
    Text         string
    Model        string
    TokensUsed   int
    FinishReason string
}
```

### Router Implementation
```go
type Router struct {
    providers map[string]Provider
    config    *Config
}

func (r *Router) SelectProvider(hint string) (Provider, string, error) {
    // Model hint resolution
    switch hint {
    case "fast":
        return r.providers["anthropic"], "claude-3-haiku-20240307", nil
    case "balanced":
        return r.providers["openai"], "gpt-4-turbo-preview", nil
    case "quality":
        return r.providers["anthropic"], "claude-3-opus-20240229", nil
    case "codegen":
        return r.providers["openai"], "gpt-4", nil
    default:
        // Parse explicit model (e.g., "anthropic:claude-3-sonnet")
        parts := strings.Split(hint, ":")
        if len(parts) == 2 {
            return r.providers[parts[0]], parts[1], nil
        }
        return nil, "", fmt.Errorf("unknown model hint: %s", hint)
    }
}

func (r *Router) Generate(ctx context.Context, opts GenerateOptions) (*Response, error) {
    provider, model, err := r.SelectProvider(opts.ModelHint)
    if err != nil {
        return nil, err
    }

    // Try primary provider
    opts.Model = model
    resp, err := provider.Generate(ctx, opts)
    if err == nil {
        return resp, nil
    }

    // Fallback to alternative providers
    for name, fallbackProvider := range r.providers {
        if name == provider.Name() {
            continue // Skip failed provider
        }

        // Attempt fallback
        fallbackModel := r.getFallbackModel(name, opts.ModelHint)
        opts.Model = fallbackModel
        resp, err = fallbackProvider.Generate(ctx, opts)
        if err == nil {
            log.Printf("Fallback to %s:%s succeeded", name, fallbackModel)
            return resp, nil
        }
    }

    return nil, fmt.Errorf("all providers failed")
}
```

### Configuration Format
```yaml
# .specular/providers.yaml
version: 1.0

providers:
  anthropic:
    enabled: true
    api_key_env: "ANTHROPIC_API_KEY"
    models:
      - claude-3-opus-20240229
      - claude-3-sonnet-20240229
      - claude-3-haiku-20240307
    timeout: 60s
    max_retries: 3

  openai:
    enabled: true
    api_key_env: "OPENAI_API_KEY"
    organization_env: "OPENAI_ORG_ID"
    models:
      - gpt-4-turbo-preview
      - gpt-4
      - gpt-3.5-turbo
    timeout: 60s
    max_retries: 3

  google:
    enabled: false  # Not configured
    api_key_env: "GEMINI_API_KEY"
    models:
      - gemini-pro
      - gemini-pro-vision
    timeout: 60s

model_hints:
  fast:
    - anthropic:claude-3-haiku-20240307
    - openai:gpt-3.5-turbo
  balanced:
    - openai:gpt-4-turbo-preview
    - anthropic:claude-3-sonnet-20240229
  quality:
    - anthropic:claude-3-opus-20240229
    - openai:gpt-4
  codegen:
    - openai:gpt-4
    - anthropic:claude-3-sonnet-20240229

fallback:
  enabled: true
  strategy: "cascade"  # Try all providers in order
  max_attempts: 3
```

## Alternatives Considered

### Option 1: Single Provider (Anthropic Only)
**Pros:**
- Simpler implementation
- No provider abstraction needed
- Lower testing burden

**Cons:**
- ❌ Vendor lock-in
- ❌ No fallback if provider is down
- ❌ Can't optimize for cost or speed
- ❌ Limited by single provider's capabilities

**Verdict:** REJECTED (too limiting)

### Option 2: Hardcoded Provider Support
**Pros:**
- No configuration needed
- Simpler error handling

**Cons:**
- ❌ Difficult to add new providers
- ❌ No user control over model selection
- ❌ Can't adapt to new models

**Verdict:** REJECTED (not future-proof)

### Option 3: Plugin System (Dynamic Loading)
**Pros:**
- Maximum flexibility
- Community can add providers
- No code changes for new providers

**Cons:**
- ❌ Security risks (loading untrusted code)
- ❌ Complex plugin API
- ❌ Versioning and compatibility issues
- ❌ Harder to distribute

**Verdict:** Future consideration for v2.0

### Option 4: LangChain Integration
**Pros:**
- Mature ecosystem
- Many providers supported
- Active community

**Cons:**
- ❌ Python dependency (Specular is Go)
- ❌ Large dependency footprint
- ❌ Opinionated architecture
- ❌ Overhead for our use case

**Verdict:** REJECTED (wrong language, too heavy)

## Model Hint System

### Hint Categories

#### Fast (Low Latency, Lower Cost)
**Use Case:** Interactive workflows, quick feedback
**Models:**
- Claude 3 Haiku (Anthropic)
- GPT-3.5 Turbo (OpenAI)
- Gemini Pro (Google)

**Characteristics:**
- Response time: <2 seconds
- Cost: $0.50-$1 per 1M tokens
- Context length: 200K tokens

#### Balanced (Best Value)
**Use Case:** General planning, moderate complexity
**Models:**
- GPT-4 Turbo (OpenAI)
- Claude 3 Sonnet (Anthropic)

**Characteristics:**
- Response time: 3-5 seconds
- Cost: $3-$15 per 1M tokens
- Context length: 200K tokens
- Best quality/price ratio

#### Quality (Maximum Capability)
**Use Case:** Complex architecture, critical decisions
**Models:**
- Claude 3 Opus (Anthropic)
- GPT-4 (OpenAI)

**Characteristics:**
- Response time: 10-30 seconds
- Cost: $15-$75 per 1M tokens
- Highest reasoning capability

#### Codegen (Optimized for Code)
**Use Case:** Code generation, refactoring
**Models:**
- GPT-4 (OpenAI)
- Claude 3 Sonnet (Anthropic)

**Characteristics:**
- Best at understanding code patterns
- Good at multi-file changes
- Strong at following conventions

### Usage Examples
```bash
# Fast iteration
specular plan --model-hint fast

# Best value (default)
specular plan --model-hint balanced

# Maximum quality
specular plan --model-hint quality

# Optimized for code
specular generate "Implement auth" --model-hint codegen

# Explicit model selection
specular plan --model anthropic:claude-3-opus-20240229
```

## Consequences

### Positive
- ✅ **Flexibility**: Users choose provider/model
- ✅ **Resilience**: Automatic fallback on failure
- ✅ **Cost Optimization**: Route to appropriate model
- ✅ **Future-Proof**: Easy to add new providers
- ✅ **Performance**: Optimize for latency when needed

### Negative
- ❌ **Complexity**: More code to maintain
- ❌ **Testing**: Must test all provider combinations
- ❌ **Configuration**: Users must configure multiple API keys
- ❌ **Cost Tracking**: Harder to predict total costs

### Mitigations
- Provide sensible defaults (works with single provider)
- Comprehensive provider health checks
- Clear error messages for configuration issues
- Cost estimation in plan output

## Security Considerations

### API Key Management
- Never log API keys
- Use environment variables
- Support vault integration (Hashicorp Vault, AWS Secrets Manager)
- Validate keys before use

### Prompt Injection Protection
- Sanitize user inputs
- Separate system prompts from user content
- Use provider-specific safety features
- Log suspicious prompts

### Rate Limiting
- Respect provider rate limits
- Implement exponential backoff
- Queue requests if necessary

## Performance Characteristics

### Measured Latencies (p95)
| Provider  | Model           | Latency | Tokens/sec |
|-----------|-----------------|---------|------------|
| Anthropic | Claude 3 Haiku  | 1.2s    | 200        |
| Anthropic | Claude 3 Sonnet | 3.5s    | 150        |
| Anthropic | Claude 3 Opus   | 12s     | 80         |
| OpenAI    | GPT-3.5 Turbo   | 1.5s    | 180        |
| OpenAI    | GPT-4 Turbo     | 4.2s    | 120        |
| OpenAI    | GPT-4           | 15s     | 60         |

### Cost Comparison (per 1M tokens)
| Provider  | Model           | Input   | Output  |
|-----------|-----------------|---------|---------|
| Anthropic | Claude 3 Haiku  | $0.25   | $1.25   |
| Anthropic | Claude 3 Sonnet | $3      | $15     |
| Anthropic | Claude 3 Opus   | $15     | $75     |
| OpenAI    | GPT-3.5 Turbo   | $0.50   | $1.50   |
| OpenAI    | GPT-4 Turbo     | $10     | $30     |
| OpenAI    | GPT-4           | $30     | $60     |

## Future Enhancements

### v1.1-v1.2
- [ ] Local model support (Ollama, LM Studio)
- [ ] Streaming responses for better UX
- [ ] Cost tracking and budgets
- [ ] Model performance analytics

### v2.0+
- [ ] Plugin system for community providers
- [ ] Multi-model ensembles
- [ ] Fine-tuned models for Specular
- [ ] On-premises enterprise models

## Related Decisions
- ADR 0003: Docker-Only Execution (affects prompt safety)
- Future ADR: Prompt engineering and safety
- Future ADR: Cost optimization strategies

## References
- [Anthropic API Documentation](https://docs.anthropic.com/claude/reference/getting-started-with-the-api)
- [OpenAI API Documentation](https://platform.openai.com/docs/api-reference)
- [Google Gemini API](https://ai.google.dev/docs)
- [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)
