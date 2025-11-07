# AI Provider Selection Guide

This guide helps you choose the right AI provider and understand how the intelligent router selects models for your tasks.

## Quick Reference

### Provider Comparison

| Provider | Context Window | Streaming | Vision | Cost | Best For |
|----------|---------------|-----------|--------|------|----------|
| **Ollama (Local)** | 8K-32K | ✅ | ❌ | Free | Development, experimentation, cost-sensitive workloads |
| **OpenAI** | 128K | ✅ | Partial | $$ | Production APIs, general-purpose tasks |
| **Anthropic** | 200K | ✅ | ✅ | $$$ | Long documents, complex reasoning, vision tasks |
| **Gemini** | 1M | ✅ | ✅ | $ | Massive context, multi-modal, cost-effective cloud option |

### Model Selection by Hint

The router automatically selects the best model based on your task hint:

| Hint | Ollama | OpenAI | Anthropic | Gemini | Use When |
|------|--------|--------|-----------|--------|----------|
| `fast` | llama3.2 | gpt-4o-mini | claude-haiku-3.5 | gemini-2.0-flash-exp | Quick responses, simple queries |
| `codegen` | codellama | gpt-4o | claude-sonnet-3.5 | gemini-2.0-flash-exp | Code generation, refactoring |
| `agentic` | llama3 | gpt-4o | claude-sonnet-4 | gemini-2.5-pro-exp-03 | Complex reasoning, planning |
| `cheap` | llama3.2 | gpt-4o-mini | claude-haiku-3.5 | gemini-2.0-flash-exp | Cost optimization |
| `long-context` | llama3 | gpt-4-turbo | claude-sonnet-3.5 | gemini-2.5-pro-exp-03 | Large documents, extensive context |

## Provider Selection Logic

The router uses a multi-factor decision process:

### 1. Provider Availability
- Checks which providers are enabled and healthy
- Filters based on `strategy.preference` order in `providers.yaml`

### 2. Model Capability Matching
- Matches task requirements to model capabilities
- Considers: streaming support, context window, vision, tools

### 3. Cost Optimization
- Prioritizes free (local) models when `prefer_cheap: true`
- Checks budget constraints (`max_cost_per_day`, `max_cost_per_request`)
- Tracks cumulative spending

### 4. Task Complexity Analysis
- **Complexity 1-3**: Fast, lightweight models
- **Complexity 4-6**: Mid-tier models
- **Complexity 7-10**: Most capable models

### 5. Priority Weighting
- **P0** (Critical): Best available model
- **P1** (High): Mid to high-tier model
- **P2** (Normal): Cost-optimized selection

## Retry and Fallback

The router includes production-grade error handling with automatic retry and fallback capabilities to ensure reliable AI interactions even when providers experience issues.

### Retry with Exponential Backoff

When a provider request fails with a **retryable error**, the router automatically retries with increasing delays:

**Retryable Errors:**
- Network timeouts and connection failures
- Rate limits (HTTP 429, 503)
- Temporary service unavailability
- Connection refused errors

**Non-Retryable Errors:**
- Authentication failures (HTTP 401, 403)
- Invalid API keys
- Context deadline exceeded
- Forbidden resources

**Retry Configuration:**
```yaml
# .specular/providers.yaml
strategy:
  fallback:
    enabled: true
    max_retries: 3              # Retry up to 3 times (default: 3)
    retry_delay_ms: 1000        # Initial backoff: 1 second (default: 1000)
    fallback_model: ollama/llama3.2
```

**Backoff Strategy:**
- **Attempt 1**: Initial delay (1s)
- **Attempt 2**: 2s (2x)
- **Attempt 3**: 4s (2x)
- **Attempt 4**: 8s (2x)
- **Max**: Capped at 30s

**Example:**
```bash
# Request with automatic retry
./specular generate "What is AI?" --model-hint fast

# If provider times out:
# - Retry 1: Wait 1s, try again
# - Retry 2: Wait 2s, try again
# - Retry 3: Wait 4s, try again
# - If still failing: Try fallback provider
```

### Provider Fallback

When the primary provider fails after all retries, the router automatically tries alternative providers in preference order.

**Fallback Flow:**
1. **Primary Provider Fails**: All retries exhausted
2. **Select Next Provider**: Use next available provider from preference list
3. **Retry with Fallback**: Apply same retry logic to fallback provider
4. **Cascade**: Continue through all available providers
5. **Final Failure**: Return error only if all providers fail

**Configuration:**
```yaml
strategy:
  preference:
    - ollama      # Primary: Try first
    - openai      # Fallback 1: If ollama fails
    - anthropic   # Fallback 2: If openai fails
    - gemini      # Fallback 3: Final fallback

  fallback:
    enabled: true                    # Enable automatic fallback
    max_retries: 3                   # Retries per provider
    fallback_model: ollama/llama3.2  # Emergency fallback model
```

**Example Scenarios:**

**Scenario 1: Ollama Down**
```bash
./specular generate "Explain databases" --verbose

# Router behavior:
# 1. Try ollama (primary) → Connection refused
# 2. Retry ollama: 1s, 2s, 4s → Still failing
# 3. Fallback to openai → Success!
# Output: Uses openai, selection reason includes "Fallback: ollama failed"
```

**Scenario 2: Rate Limited**
```bash
./specular generate "Complex task" --model-hint agentic

# Router behavior:
# 1. Try anthropic → HTTP 429 Rate Limit
# 2. Retry with backoff: 1s, 2s, 4s → Still rate limited
# 3. Fallback to gemini → Success!
# Output: Uses gemini, notes rate limit fallback
```

**Scenario 3: All Providers Fail**
```bash
./specular generate "Query" --verbose

# Router behavior:
# 1. Try ollama → Network error (retries: fail)
# 2. Try openai → Auth error (no retry: non-retryable)
# 3. Try anthropic → Timeout (retries: fail)
# 4. Try gemini → Rate limit (retries: fail)
# Error: "all fallback providers failed"
```

### Disabling Retry/Fallback

For specific use cases, you can disable retry and fallback:

```yaml
# Disable all retry and fallback
strategy:
  fallback:
    enabled: false    # No fallback to alternative providers
    max_retries: 0    # No retries (fail immediately)
```

**When to Disable:**
- Testing error handling in your application
- Debugging provider-specific issues
- Strict provider requirements (e.g., only use ollama)
- Cost control (avoid unexpected fallback costs)

### Monitoring Retry/Fallback

Use `--verbose` to see retry and fallback activity:

```bash
./specular generate "test" --model-hint fast --verbose

# Output includes:
# → Selected: llama3.2 (local)
# → Error: connection timeout
# → Retry 1/3: waiting 1s...
# → Retry 2/3: waiting 2s...
# → Retry 3/3: waiting 4s...
# → Fallback to: gpt-4o-mini (openai)
# → Success! (cost: $0.0001, tokens: 50)
# → Selection reason: Fallback after ollama failure
```

### Best Practices

1. **Enable Fallback in Production**: Always have backup providers configured
2. **Set Reasonable Retries**: 3 retries balances reliability and latency
3. **Monitor Costs**: Fallback to cloud providers can increase costs
4. **Use Verbose Mode**: Understand retry/fallback behavior during development
5. **Test Failure Scenarios**: Validate behavior when providers are unavailable
6. **Configure Preference Order**: Put most reliable/preferred providers first
7. **Set Budget Limits**: Prevent unexpected costs from extensive fallback usage

## Streaming Support

The router provides comprehensive streaming support with the same retry and fallback capabilities as standard generation.

### Streaming vs Standard Generation

**Standard Generation (`generate`):**
- Returns complete response at once
- Better for short responses
- Lower perceived latency for small outputs
- Easier to work with programmatically

**Streaming (`stream`):**
- Returns response incrementally as tokens are generated
- Better user experience for long responses
- Lower time-to-first-token
- Allows progressive display of content
- Real-time cost tracking

### Using Streaming

The router's `Stream()` method provides the same intelligent model selection, retry, and fallback as `Generate()`:

```go
// Router streaming API
stream, err := router.Stream(ctx, router.GenerateRequest{
    Prompt:     "Write a long essay about AI",
    ModelHint:  "agentic",
    Complexity: 7,
})

for chunk := range stream {
    if chunk.Error != nil {
        // Handle error
        log.Printf("Stream error: %v", chunk.Error)
        break
    }

    // Display incremental content
    fmt.Print(chunk.Delta)

    if chunk.Done {
        // Stream complete
        fmt.Printf("\n\nTotal: %s\n", chunk.Content)
    }
}
```

### Streaming Features

**Automatic Retry:**
- Same exponential backoff as Generate()
- Retries stream connection failures
- Classifies errors (retryable vs non-retryable)
- Respects context cancellation

**Provider Fallback:**
- Falls back to alternative providers on failure
- Each fallback provider gets full retry treatment
- Maintains provider preference order
- Tracks costs across fallback attempts

**Real-Time Cost Tracking:**
- Tokens counted as they arrive
- Cost calculated on stream completion
- Usage recorded automatically
- Budget limits enforced

**Channel-Based Architecture:**
- Non-blocking streaming with Go channels
- Buffered channels prevent blocking (10-chunk buffer)
- Automatic cleanup on completion
- Context-aware cancellation

### Streaming Configuration

Streaming uses the same retry and fallback configuration as standard generation:

```yaml
# .specular/providers.yaml
strategy:
  fallback:
    enabled: true          # Enable fallback for streaming
    max_retries: 3         # Retry stream connection failures
    retry_delay_ms: 1000   # Initial backoff
```

### Streaming Best Practices

1. **Use for Long Responses**: Streaming is ideal for essays, code generation, long explanations
2. **Handle Partial Content**: Be prepared to handle incomplete streams on error
3. **Display Progressively**: Show content as it arrives for better UX
4. **Set Timeouts**: Use context with timeout for stream operations
5. **Monitor Costs**: Track token usage in real-time during streaming
6. **Test Cancellation**: Verify context cancellation works correctly
7. **Buffer Appropriately**: Default 10-chunk buffer works for most cases

### Streaming Examples

**CLI Streaming:**
```bash
# Streaming is automatic for `generate` command
./specular generate "Write a detailed guide" --model-hint agentic --verbose

# Output shows tokens as they arrive in real-time
```

**Streaming with Fallback:**
```bash
# If ollama fails, automatically falls back to openai
./specular generate "Long response" --model-hint fast

# Logs show:
# → Selected: llama3.2 (ollama)
# → Stream connection failed: connection refused
# → Retry 1/3: waiting 1s...
# → All retries failed
# → Fallback to: gpt-4o-mini (openai)
# → Streaming... (successful with fallback)
```

### When to Use Streaming

**Use Streaming When:**
- Generating long-form content (>500 tokens)
- Building interactive chat interfaces
- Providing real-time feedback to users
- Processing large documents with summaries
- Time-to-first-token matters

**Use Standard Generation When:**
- Responses are short (<100 tokens)
- You need complete response for processing
- Working with structured output (JSON)
- Batch processing multiple requests
- Simplicity preferred over streaming complexity

## Context Window Management

The router includes intelligent context window management to prevent errors from oversized prompts exceeding model token limits. This feature automatically validates that your requests fit within the selected model's context window and can optionally truncate requests that are too large.

### Why Context Window Management Matters

**Problem:** Different AI models have different context window sizes (e.g., llama3.2: 8K, gpt-4o: 128K, claude-sonnet-3.5: 200K). If your prompt + context + expected output exceeds the model's window, the request will fail with a cryptic error.

**Solution:** The router validates context size before sending requests and can automatically truncate oversized contexts using intelligent strategies.

**Benefits:**
- **Prevent costly errors**: Avoid wasting API calls on requests that will fail
- **Automatic recovery**: Auto-truncation keeps your application running
- **Smart truncation**: Choose which parts of context to preserve
- **Cost optimization**: Smaller contexts = lower token costs

### Context Validation

Context validation checks that your request fits within the model's context window before sending it to the provider.

**How it Works:**
1. **Estimate tokens**: Uses character-based estimation (4 chars/token conservative estimate)
2. **Calculate total**: Counts prompt + system prompt + context messages + expected output
3. **Validate fit**: Checks if total tokens ≤ model's context window
4. **Error or truncate**: Either returns error or attempts auto-truncation

**Token Estimation:**
```go
// Estimation formula
inputTokens = promptTokens + systemPromptTokens + contextTokens + overhead
outputTokens = req.MaxTokens (or 2048 default)
totalTokens = inputTokens + outputTokens

// Must fit: totalTokens ≤ model.ContextWindow
```

**Overhead Calculation:**
- ~5 tokens per context message (role + formatting)
- ~20 tokens for request structure
- Conservative rounding to prevent edge cases

### Configuration

Configure context window management in your router configuration:

```yaml
# .specular/providers.yaml or router config
strategy:
  context:
    enable_context_validation: true   # Validate context fits in model window (default: true)
    auto_truncate: false              # Automatically truncate oversized contexts (default: false)
    truncation_strategy: "oldest"     # Strategy: oldest, prompt, context, proportional (default: oldest)
```

**Configuration Options:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enable_context_validation` | bool | `true` | Enable context window validation |
| `auto_truncate` | bool | `false` | Automatically truncate oversized contexts |
| `truncation_strategy` | string | `"oldest"` | Which truncation strategy to use |

**Default Behavior:**
- ✅ **Validation enabled**: Prevents context window errors
- ❌ **Auto-truncate disabled**: Safer - requires explicit opt-in to modify requests
- 🔄 **Strategy: oldest**: Removes oldest context messages first when truncating

### Truncation Strategies

When `auto_truncate: true`, the router uses one of four intelligent strategies to reduce context size:

#### 1. TruncateOldest (Recommended)

**Strategy:** Removes oldest context messages first, preserving recent conversation.

**Best for:**
- Chat applications with conversation history
- Long-running sessions where recent context is most important
- Maintaining conversation continuity

**Example:**
```go
// Before truncation (exceeds 8K context window)
req := GenerateRequest{
    Prompt: "What did we discuss about databases?",
    Context: []Message{
        {Role: "user", Content: "Tell me about Go"},           // OLD - will be removed first
        {Role: "assistant", Content: "Go is..."},              // OLD - will be removed first
        {Role: "user", Content: "What about databases?"},      // RECENT - preserved
        {Role: "assistant", Content: "PostgreSQL is..."},      // RECENT - preserved
        {Role: "user", Content: "Indexes?"},                   // RECENT - preserved
    },
}

// After truncation: Oldest 2 messages removed, recent 3 preserved
```

**Configuration:**
```yaml
strategy:
  context:
    truncation_strategy: "oldest"
```

#### 2. TruncatePrompt

**Strategy:** Truncates the main prompt while preserving all context messages.

**Best for:**
- Very long prompts (e.g., pasting large documents)
- When context history is more important than full prompt
- Document summarization with discussion history

**Example:**
```go
// Before truncation
req := GenerateRequest{
    Prompt: strings.Repeat("Long document content... ", 10000),  // TRUNCATED
    Context: []Message{
        {Role: "user", Content: "Previous question"},          // PRESERVED
        {Role: "assistant", Content: "Previous answer"},       // PRESERVED
    },
}

// After truncation: Prompt shortened to fit, all context preserved
// Prompt becomes: "Long document content... Long do...[truncated]"
```

**Configuration:**
```yaml
strategy:
  context:
    truncation_strategy: "prompt"
```

#### 3. TruncateContext

**Strategy:** Removes all context messages, preserving only the prompt.

**Best for:**
- Single-shot queries without conversation history
- When the current prompt is most important
- Stateless API calls

**Example:**
```go
// Before truncation
req := GenerateRequest{
    Prompt: "Explain databases",                    // PRESERVED
    Context: []Message{
        {Role: "user", Content: "..."},             // REMOVED
        {Role: "assistant", Content: "..."},        // REMOVED
        {Role: "user", Content: "..."},             // REMOVED
    },
}

// After truncation: All context removed, only prompt remains
```

**Configuration:**
```yaml
strategy:
  context:
    truncation_strategy: "context"
```

#### 4. TruncateProportional

**Strategy:** Reduces both prompt and context proportionally based on their sizes.

**Best for:**
- Balanced reduction when both prompt and context matter
- Long prompts with important context
- Maintaining relative proportions

**Example:**
```go
// Before truncation (need to remove 1000 tokens)
promptTokens := 3000    // 60% of total
contextTokens := 2000   // 40% of total

// Proportional reduction:
// Prompt: Remove 600 tokens (60% of 1000)
// Context: Remove 400 tokens (40% of 1000)

// After truncation: Both reduced proportionally
```

**Configuration:**
```yaml
strategy:
  context:
    truncation_strategy: "proportional"
```

### Strategy Comparison

| Strategy | Preserves Prompt | Preserves Context | Best Use Case |
|----------|------------------|-------------------|---------------|
| **oldest** | ✅ Fully | 🟡 Recent only | Chat/conversation apps |
| **prompt** | 🟡 Partial | ✅ Fully | Document Q&A with history |
| **context** | ✅ Fully | ❌ None | Single-shot queries |
| **proportional** | 🟡 Partial | 🟡 Partial | Balanced reduction |

### Using Context Validation

#### Example 1: Validation Only (Fail on Oversized)

```go
// Configuration
config := &RouterConfig{
    EnableContextValidation: true,   // Validate context
    AutoTruncate:           false,   // Don't auto-truncate
}

router, _ := NewRouter(config)

// Request with large context
req := GenerateRequest{
    Prompt: strings.Repeat("word ", 10000),  // Very long prompt
    MaxTokens: 2048,
}

// Result: Error returned if exceeds context window
resp, err := router.Generate(ctx, req)
if err != nil {
    // Error: "context validation failed: request exceeds model context window:
    //         need 12000 tokens (input: 10000 + output: 2048), model supports 8000 tokens"
}
```

#### Example 2: Auto-Truncation with Oldest Strategy

```go
// Configuration
config := &RouterConfig{
    EnableContextValidation: true,
    AutoTruncate:           true,             // Enable auto-truncate
    TruncationStrategy:     "oldest",         // Remove oldest messages
}

router, _ := NewRouter(config)

// Request with conversation history
req := GenerateRequest{
    Prompt: "Summarize our discussion",
    Context: []Message{
        {Role: "user", Content: "Old message 1..."},
        {Role: "assistant", Content: "Old response 1..."},
        // ... 50 more messages ...
        {Role: "user", Content: "Recent message"},
    },
    MaxTokens: 1024,
}

// Result: Automatically truncates oldest messages to fit
resp, err := router.Generate(ctx, req)
// Success! Oldest messages removed, recent conversation preserved
```

#### Example 3: Prompt Truncation for Long Documents

```go
// Configuration
config := &RouterConfig{
    EnableContextValidation: true,
    AutoTruncate:           true,
    TruncationStrategy:     "prompt",  // Truncate prompt, keep context
}

router, _ := NewRouter(config)

// Request with very long document
req := GenerateRequest{
    Prompt: veryLongDocument,  // 20K tokens
    Context: []Message{
        {Role: "user", Content: "What's the main theme?"},
        {Role: "assistant", Content: "The document discusses..."},
    },
    MaxTokens: 1024,
}

// Result: Prompt truncated to fit, conversation history preserved
resp, err := router.Generate(ctx, req)
```

### Context Validation with Streaming

Context validation works identically with streaming:

```go
// Streaming with auto-truncation
stream, err := router.Stream(ctx, GenerateRequest{
    Prompt: largePrompt,
    Context: longConversationHistory,
    ModelHint: "fast",
})

// If context exceeds window:
// - Validates before streaming starts
// - Auto-truncates if enabled
// - Returns error if validation fails and auto-truncate disabled
// - Streams normally after successful validation/truncation

for chunk := range stream {
    if chunk.Error != nil {
        // Context validation errors appear here
        log.Printf("Error: %v", chunk.Error)
        break
    }
    fmt.Print(chunk.Delta)
}
```

### Error Handling

#### Validation Failure (auto_truncate: false)

```go
resp, err := router.Generate(ctx, req)
if err != nil {
    // Error format:
    // "context validation failed: request exceeds model context window:
    //  need 12000 tokens (input: 10000 + output: 2048), model supports 8000 tokens"

    // Handle by:
    // 1. Reducing prompt size
    // 2. Removing context messages
    // 3. Lowering MaxTokens
    // 4. Enabling auto-truncate
    // 5. Selecting model with larger context window
}
```

#### Truncation Failure

```go
// If even after truncation, request still doesn't fit:
resp, err := router.Generate(ctx, req)
if err != nil {
    // Error: "context validation failed and truncation failed:
    //         insufficient context window: model supports 8000 tokens,
    //         but output needs 10000 tokens"

    // This happens when MaxTokens is too large for model
    // Solution: Lower MaxTokens or select larger model
}
```

### Best Practices

1. **Enable Validation in Production**: Always validate context to prevent runtime errors
   ```yaml
   enable_context_validation: true  # Always enable
   ```

2. **Auto-Truncate Based on Use Case**:
   - **Disable** for critical applications (safer, explicit control)
   - **Enable** for user-facing apps (better UX, automatic recovery)
   ```yaml
   auto_truncate: false  # Production: explicit control
   auto_truncate: true   # User apps: automatic recovery
   ```

3. **Choose Strategy by Application Type**:
   ```yaml
   # Chat/conversation apps
   truncation_strategy: "oldest"

   # Document Q&A
   truncation_strategy: "prompt"

   # Single queries
   truncation_strategy: "context"

   # Balanced needs
   truncation_strategy: "proportional"
   ```

4. **Set Reasonable MaxTokens**: Leave room for input context
   ```go
   // Bad: MaxTokens uses entire context window
   req.MaxTokens = 8000  // Model has 8K window

   // Good: Leave room for input
   req.MaxTokens = 2048  // Leaves 6K for input
   ```

5. **Monitor Token Usage**: Use verbose mode to see truncation activity
   ```bash
   ./specular generate "query" --verbose
   # Shows: Context validation passed, tokens used, whether truncated
   ```

6. **Test with Large Contexts**: Validate behavior with edge cases
   ```go
   // Test with oversized context
   testReq := GenerateRequest{
       Prompt: strings.Repeat("word ", 10000),
       MaxTokens: 1024,
   }
   ```

7. **Consider Model Selection**: Choose models with appropriate context windows
   ```go
   // Small context: Use any model
   req.ModelHint = "fast"

   // Large context: Use long-context models
   req.ModelHint = "long-context"  // Selects 128K-200K+ models
   ```

8. **Handle Errors Gracefully**: Provide user feedback on truncation
   ```go
   resp, err := router.Generate(ctx, req)
   if err != nil {
       if strings.Contains(err.Error(), "context validation failed") {
           return "Your request is too large. Please reduce context or prompt size."
       }
   }
   ```

### Performance Considerations

**Token Estimation Performance:**
- Character-based estimation is fast (~microseconds)
- No external API calls for token counting
- Conservative estimates prevent edge cases

**Truncation Performance:**
- Truncation creates a copy of the request (doesn't modify original)
- Oldest/Context strategies: O(n) where n = number of messages
- Prompt strategy: O(1) string truncation
- Proportional: O(n) message removal + O(1) prompt truncation

**Memory Usage:**
- Original request preserved (not modified in-place)
- Truncated request is new allocation
- Both coexist briefly during truncation

### Monitoring Context Window Usage

Use verbose mode to see context validation activity:

```bash
./specular generate "query with large context" --verbose

# Output shows:
# → Selected: llama3.2 (8K context window)
# → Context validation: 5234 tokens (input: 4210 + output: 1024)
# → Validation: PASSED (65% of context window used)
# → Success! (cost: $0.00, tokens: 5234)
```

With auto-truncation enabled:
```bash
./specular generate "oversized query" --verbose

# Output shows:
# → Selected: llama3.2 (8K context window)
# → Context validation: 9500 tokens (input: 8476 + output: 1024)
# → Validation: FAILED (exceeds window by 1500 tokens)
# → Auto-truncate: Applying 'oldest' strategy
# → Truncated: Removed 3 oldest messages (-1600 tokens)
# → Context validation: 7900 tokens (input: 6876 + output: 1024)
# → Validation: PASSED (98% of context window used)
# → Success! (cost: $0.00, tokens: 7900)
```

### Context Window by Provider

Reference table for planning context usage:

| Provider | Model | Context Window | With 2K Output | Available for Input |
|----------|-------|----------------|----------------|---------------------|
| Ollama | llama3.2 | 8K | 6K | Small contexts |
| Ollama | llama3 | 8K | 6K | Small contexts |
| Ollama | codellama | 16K | 14K | Medium contexts |
| OpenAI | gpt-4o-mini | 128K | 126K | Large contexts |
| OpenAI | gpt-4o | 128K | 126K | Large contexts |
| OpenAI | gpt-4-turbo | 128K | 126K | Large contexts |
| Anthropic | claude-haiku-3.5 | 200K | 198K | Very large contexts |
| Anthropic | claude-sonnet-3.5 | 200K | 198K | Very large contexts |
| Anthropic | claude-sonnet-4 | 200K | 198K | Very large contexts |
| Gemini | gemini-2.0-flash-exp | 1M | 998K | Massive contexts |
| Gemini | gemini-2.5-pro-exp-03 | 1M | 998K | Massive contexts |

**Planning Guidelines:**
- **< 6K input**: Any model works
- **6K - 14K input**: Use codellama or cloud models
- **14K - 126K input**: Use OpenAI or Anthropic
- **126K - 198K input**: Use Anthropic
- **> 198K input**: Use Gemini (1M window)

## Real-World Examples

### Example 1: PRD → Spec Generation
```bash
./specular spec generate --in PRD.md --out spec.yaml
```

**Router Decision:**
- Task: PRD parsing (complex reasoning)
- Hint: `agentic`
- Complexity: 8 (high)
- Priority: P0 (critical)
- **Selected**: `llama3` (most capable local model)
- **Reasoning**: Budget-optimized (prefer local), high complexity requires capable model

### Example 2: Simple Code Generation
```bash
./specular generate "Write a function to reverse a string" --model-hint codegen
```

**Router Decision:**
- Task: Code generation
- Hint: `codegen`
- Complexity: 4 (medium)
- **Selected**: `codellama` (specialized for code)
- **Reasoning**: Matched hint, budget-optimized selection

### Example 3: Fast Query
```bash
./specular generate "What is 2+2?" --model-hint fast
```

**Router Decision:**
- Task: Simple query
- Hint: `fast`
- Complexity: 1 (low)
- **Selected**: `llama3.2` (fastest local model)
- **Reasoning**: Matched hint, budget-optimized, low latency

## Provider Selection Strategy

### Development Workflow
```yaml
strategy:
  preference:
    - ollama      # Free, local, fast iteration
    - openai      # API fallback
```

**Best for:**
- Local development
- Experimentation
- Cost-sensitive projects
- No network dependency

### Production Workflow
```yaml
strategy:
  preference:
    - anthropic   # High quality, long context
    - gemini      # Massive context, cost-effective
    - openai      # Reliable fallback
```

**Best for:**
- Production applications
- Customer-facing features
- Complex reasoning tasks
- Long document processing

### Hybrid Workflow
```yaml
strategy:
  preference:
    - ollama      # Local first
    - gemini      # Cost-effective cloud
    - anthropic   # High-quality fallback
```

**Best for:**
- Mixed workloads
- Cost optimization with cloud backup
- Development with production testing

## Context Window Recommendations

### Small Context (< 8K tokens)
**Any provider works well**
- Ollama: llama3.2 (8K)
- OpenAI: gpt-4o-mini (128K)
- Anthropic: claude-haiku-3.5 (200K)
- Gemini: gemini-2.0-flash-exp (1M)

### Medium Context (8K - 32K tokens)
**Good options:**
- OpenAI: gpt-4o (128K)
- Anthropic: claude-sonnet-3.5 (200K)
- Gemini: gemini-2.0-flash-exp (1M)

### Large Context (32K - 128K tokens)
**Recommended:**
- OpenAI: gpt-4-turbo (128K)
- Anthropic: claude-sonnet-3.5 (200K)
- Gemini: gemini-2.5-pro-exp-03 (1M)

### Massive Context (> 128K tokens)
**Best options:**
- Anthropic: claude-sonnet-3.5 (200K) ⭐
- **Gemini: gemini-2.5-pro-exp-03 (1M)** ⭐⭐⭐ (Largest!)

## Cost Optimization Tips

### 1. Use Local Models When Possible
```yaml
strategy:
  preference:
    - ollama  # Always try free local first
```

### 2. Set Budget Constraints
```yaml
strategy:
  budget:
    max_cost_per_day: 10.0   # Limit daily spending
    max_cost_per_request: 0.50  # Cap per-request cost
```

### 3. Enable Cost Optimization
```yaml
strategy:
  performance:
    prefer_cheap: true  # Choose cheaper models when quality is similar
```

### 4. Use Appropriate Model Hints
```bash
# Simple tasks - use 'fast' or 'cheap'
./specular generate "Simple query" --model-hint fast

# Complex tasks - use 'agentic' only when needed
./specular generate "Complex reasoning" --model-hint agentic --complexity 8
```

## Streaming Support

All API providers support streaming for real-time responses:

### OpenAI Streaming
- Format: Server-Sent Events (SSE)
- Simple data chunks
- End marker: `data: [DONE]`

### Anthropic Streaming
- Format: Event-based SSE
- Events: `content_block_delta`, `message_stop`
- Partial content assembly

### Gemini Streaming
- Format: SSE with `alt=sse` parameter
- Contents/Parts structure
- Finish reason in chunks

### Ollama Streaming
- Format: Newline-delimited JSON
- Real-time delta updates
- Final chunk with total tokens

## Vision Capabilities

### Anthropic Claude (Full Support)
- High-quality image analysis
- Document understanding
- Visual reasoning
- Multiple images per request

### Gemini (Full Support)
- Multi-modal understanding
- Image and text combination
- Visual question answering
- Video frame analysis

### OpenAI (Partial Support)
- Image understanding
- Basic visual analysis
- Limited to certain models

### Ollama (No Support)
- Text-only models currently
- Future: llava variants possible

## Provider Configuration Examples

### Development Setup (Local First)
```yaml
providers:
  - name: ollama
    type: cli
    enabled: true
    source: local

strategy:
  preference: [ollama]
  budget:
    max_cost_per_day: 0.0  # Free only
  performance:
    prefer_cheap: true
```

### Production Setup (Cloud)
```yaml
providers:
  - name: anthropic
    type: api
    enabled: true
    config:
      api_key: ${ANTHROPIC_API_KEY}

  - name: gemini
    type: api
    enabled: true
    config:
      api_key: ${GEMINI_API_KEY}

strategy:
  preference: [anthropic, gemini]
  budget:
    max_cost_per_day: 50.0
    max_cost_per_request: 5.0
  performance:
    prefer_cheap: false  # Quality over cost
```

### Hybrid Setup (Best of Both)
```yaml
providers:
  - name: ollama
    type: cli
    enabled: true

  - name: gemini
    type: api
    enabled: true
    config:
      api_key: ${GEMINI_API_KEY}

  - name: anthropic
    type: api
    enabled: true
    config:
      api_key: ${ANTHROPIC_API_KEY}

strategy:
  preference: [ollama, gemini, anthropic]
  budget:
    max_cost_per_day: 20.0
  performance:
    prefer_cheap: true  # Try free first
  fallback:
    enabled: true
    max_retries: 3
```

## Testing Provider Selection

### Check Current Configuration
```bash
# List all providers and their status
./specular provider list

# Check provider health
./specular provider health

# Test specific provider
./specular provider health ollama
```

### Test Model Selection
```bash
# Test with different hints (verbose shows selection reasoning)
./specular generate "Test query" --model-hint fast --verbose
./specular generate "Code task" --model-hint codegen --verbose
./specular generate "Complex task" --model-hint agentic --complexity 8 --verbose
```

### Validate End-to-End Workflow
```bash
# Create test PRD
cat > test.md <<EOF
# My Product
## Goals
- Build amazing software
## Features
### Feature 1
Description of feature
EOF

# Generate spec (shows provider selection)
./specular spec generate --in test.md --out spec.yaml

# Validate workflow
./specular spec validate --in spec.yaml
./specular spec lock --in spec.yaml --out spec.lock.json
./specular plan --in spec.yaml --lock spec.lock.json --out plan.json
```

## Troubleshooting

### Provider Not Available
```bash
# Check provider status
./specular provider health

# Common issues:
# - Ollama: Is ollama running? (ollama serve)
# - API providers: Is API key set? (echo $OPENAI_API_KEY)
# - Enabled: Check providers.yaml enabled: true
```

### High Costs
```bash
# Check budget usage
./specular generate "query" --verbose  # See budget at end

# Lower budget limits
# Edit .specular/providers.yaml:
strategy:
  budget:
    max_cost_per_day: 5.0  # Lower limit
  performance:
    prefer_cheap: true  # Favor cheaper models
```

### Slow Responses
```bash
# Use faster models
./specular generate "query" --model-hint fast

# Check latency limits
# Edit .specular/providers.yaml:
strategy:
  performance:
    max_latency_ms: 30000  # 30 seconds max
```

## Best Practices

1. **Start Local**: Use Ollama for development and iteration
2. **Enable Fallbacks**: Configure cloud providers as backup (retry + fallback enabled by default)
3. **Set Retry Limits**: Use 3 retries for production (balance reliability and latency)
4. **Configure Preference Order**: Put most reliable providers first in preference list
5. **Set Budgets**: Always set reasonable cost limits (prevent unexpected fallback costs)
6. **Use Hints**: Provide model hints for better selection
7. **Monitor Costs**: Use `--verbose` to track spending and fallback usage
8. **Test Failure Scenarios**: Validate retry/fallback behavior when providers are unavailable
9. **Test Early**: Validate provider setup before production
10. **Document Selection**: Use verbose mode to understand routing and fallback decisions

## Further Reading

- [Provider System README](../internal/provider/README.md)
- [Router Documentation](../internal/router/README.md)
- [PRD Parser Guide](../internal/prd/README.md)
- [Main README](../README.md)
