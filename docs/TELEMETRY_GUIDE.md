# Telemetry Configuration & Integration Guide

## Overview

This guide covers the configuration and integration of the production-grade OpenTelemetry tracing system in Specular. The telemetry package provides distributed tracing with OTLP HTTP export, circuit breaker pattern, and exponential backoff retry logic.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Configuration](#configuration)
3. [Distributed Tracing](#distributed-tracing)
4. [Metrics Collection](#metrics-collection)
5. [Provider Health Checks](#provider-health-checks)
6. [Circuit Breaker](#circuit-breaker)
7. [Retry Logic](#retry-logic)
8. [Integration Examples](#integration-examples)
9. [Performance Characteristics](#performance-characteristics)
10. [Troubleshooting](#troubleshooting)

## Quick Start

### Basic Setup

```go
import (
    "context"
    "github.com/felixgeelhaar/specular/internal/telemetry"
)

func main() {
    ctx := context.Background()

    // Configure telemetry
    cfg := telemetry.Config{
        Enabled:        true,
        ServiceName:    "specular",
        ServiceVersion: "1.4.0",
        Endpoint:       "localhost:4318",  // OTLP HTTP endpoint
        SampleRate:     1.0,                // 100% sampling
    }

    // Initialize provider
    shutdown, err := telemetry.InitProvider(ctx, cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer shutdown(ctx)

    // Start using traces
    tracer := telemetry.GetTracerProvider().Tracer("my-component")
    ctx, span := tracer.Start(ctx, "operation-name")
    defer span.End()

    // Your code here
}
```

### Disabled Mode (Zero Overhead)

```go
cfg := telemetry.Config{
    Enabled: false,  // Uses noop provider (~35 ns/op overhead)
}
shutdown, _ := telemetry.InitProvider(ctx, cfg)
defer shutdown(ctx)
```

## Configuration

### Config Structure

```go
type Config struct {
    Enabled        bool    // Enable/disable telemetry
    ServiceName    string  // Service name for resource detection
    ServiceVersion string  // Service version (optional)
    Endpoint       string  // OTLP HTTP endpoint (e.g., "localhost:4318")
    SampleRate     float64 // Trace sampling rate (0.0 - 1.0)
}
```

### Configuration Options

#### 1. Enabling/Disabling Telemetry

```go
// Production: disabled by default for minimal overhead
cfg := telemetry.DefaultConfig()
cfg.Enabled = false

// Development: enabled for debugging
cfg.Enabled = true
```

**Performance Impact:**
- Disabled: 35.24 ns/op, 48 B/op, 1 alloc/op
- Enabled: 454.0 ns/op, 2623 B/op, 3 allocs/op (with batching)

#### 2. Service Identification

```go
cfg.ServiceName = "specular"
cfg.ServiceVersion = "1.4.0"
```

These values are automatically added to all spans as resource attributes:
- `service.name`: "specular"
- `service.version`: "1.4.0"
- `host.name`: auto-detected
- `os.type`: auto-detected
- `process.runtime.name`: "go"
- `process.runtime.version`: Go version

#### 3. OTLP Endpoint

```go
// Local development
cfg.Endpoint = "localhost:4318"

// Production collector
cfg.Endpoint = "otlp-collector.prod.example.com:4318"

// No endpoint (stdout exporter)
cfg.Endpoint = ""
```

**Note:** The endpoint uses OTLP HTTP protocol with gzip compression.

#### 4. Sampling Rate

```go
// 100% sampling (development/debugging)
cfg.SampleRate = 1.0

// 10% sampling (high-traffic production)
cfg.SampleRate = 0.1

// 1% sampling (very high traffic)
cfg.SampleRate = 0.01
```

**Sampling Strategy:**
- Uses trace ID ratio-based sampling (deterministic)
- Same trace ID always makes same sampling decision
- Maintains trace completeness across services

## Distributed Tracing

Distributed tracing allows you to track requests as they flow through your application. The telemetry package provides convenient helper functions for common tracing patterns.

### Helper Functions

#### Command Tracing

```go
import "github.com/felixgeelhaar/specular/internal/telemetry"

func executeAutoCommand(ctx context.Context, goal string, profile string) error {
    // Start command span - automatically records command name and component
    ctx, span := telemetry.StartCommandSpan(ctx, "auto")
    defer span.End()

    // Add custom attributes
    span.SetAttributes(
        attribute.String("goal", goal),
        attribute.String("profile", profile),
    )

    start := time.Now()

    // Your command logic here
    err := runAutoMode(ctx, goal, profile)

    duration := time.Since(start)

    if err != nil {
        // Record failure in both traces and metrics
        telemetry.RecordCommandFailure(ctx, span, "auto", err)
        return err
    }

    // Record success in both traces and metrics
    telemetry.RecordCommandSuccess(ctx, span, "auto", duration,
        attribute.Int("tasks_completed", 5),
    )

    return nil
}
```

**StartCommandSpan** automatically adds:
- `command`: Command name
- `component`: "cli"
- Records command invocation metric with status "started"

#### Provider API Tracing

```go
func callProviderAPI(ctx context.Context, provider string, prompt string) (*Response, error) {
    // Start provider span - automatically records provider and operation
    ctx, span := telemetry.StartProviderSpan(ctx, provider, "generate")
    defer span.End()

    // Add request metadata
    span.SetAttributes(
        attribute.String("model", "claude-3-sonnet"),
        attribute.Int("prompt_length", len(prompt)),
    )

    start := time.Now()

    // Make API call
    resp, err := apiClient.Generate(ctx, prompt)

    duration := time.Since(start)

    if err != nil {
        // Record failure in both traces and metrics
        telemetry.RecordProviderFailure(ctx, span, provider, "generate", err)
        return nil, err
    }

    // Record success in both traces and metrics
    telemetry.RecordProviderSuccess(ctx, span, provider, "generate", duration,
        attribute.String("model", resp.Model),
        attribute.Int("prompt_tokens", resp.PromptTokens),
        attribute.Int("completion_tokens", resp.CompletionTokens),
    )

    // Also record token usage metrics
    telemetry.RecordProviderTokens(ctx, provider, resp.Model, "input", resp.PromptTokens)
    telemetry.RecordProviderTokens(ctx, provider, resp.Model, "output", resp.CompletionTokens)

    return resp, nil
}
```

**StartProviderSpan** automatically adds:
- `provider`: Provider name (e.g., "anthropic", "openai")
- `operation`: Operation name (e.g., "generate", "stream")
- `component`: "provider"
- Records provider call metric with status "started"

#### Subprocess/Step Tracing (Auto Mode)

```go
func executeAutoWorkflow(ctx context.Context, goal string) error {
    ctx, span := telemetry.StartCommandSpan(ctx, "auto")
    defer span.End()

    // Subprocess 1: Spec generation
    ctx, specSpan := telemetry.StartSubprocessSpan(ctx, "spec_generation")
    spec, err := generateSpec(ctx, goal)
    if err != nil {
        telemetry.RecordError(specSpan, err)
        specSpan.End()
        return err
    }
    telemetry.RecordSuccess(specSpan, attribute.Int("requirements_count", len(spec.Requirements)))
    specSpan.End()

    // Subprocess 2: Plan generation
    ctx, planSpan := telemetry.StartSubprocessSpan(ctx, "plan_generation")
    plan, err := generatePlan(ctx, spec)
    if err != nil {
        telemetry.RecordError(planSpan, err)
        planSpan.End()
        return err
    }
    telemetry.RecordSuccess(planSpan, attribute.Int("steps_count", len(plan.Steps)))
    planSpan.End()

    // Subprocess 3-N: Execute each step
    for i, step := range plan.Steps {
        _, stepSpan := telemetry.StartSubprocessSpan(ctx, "step_execution")
        stepSpan.SetAttributes(
            attribute.Int("step_number", i+1),
            attribute.Int("total_steps", len(plan.Steps)),
            attribute.String("step_description", step.Description),
        )

        err := executeStep(ctx, step)
        if err != nil {
            telemetry.RecordError(stepSpan, err)
            stepSpan.End()
            return err
        }

        telemetry.RecordSuccess(stepSpan)
        stepSpan.End()
    }

    telemetry.RecordSuccess(span)
    return nil
}
```

**StartSubprocessSpan** automatically adds:
- `step`: Step name (e.g., "spec_generation", "plan_generation")
- `component`: "auto"

### Helper Functions Reference

#### RecordSuccess
Marks a span as successful with optional result attributes.

```go
telemetry.RecordSuccess(span,
    attribute.Int("tokens_used", 1234),
    attribute.String("model", "claude-3-sonnet"),
)
```

#### RecordError
Records an error in a span and sets error status.

```go
if err != nil {
    telemetry.RecordError(span, err)
    return err
}
```

#### RecordDuration
Records the duration of an operation as a span attribute.

```go
start := time.Now()
// ... operation ...
telemetry.RecordDuration(span, "api_call_duration", time.Since(start))
```

#### RecordMetrics
Records multiple metrics as span attributes.

```go
telemetry.RecordMetrics(span, map[string]int64{
    "lines_of_code": 1234,
    "files_modified": 5,
    "tests_added": 12,
})
```

#### TraceFunction
Wraps a function call with automatic span creation and error handling.

```go
result, err := telemetry.TraceFunction(ctx, "process_spec", func(ctx context.Context) (interface{}, error) {
    return processSpec(ctx, spec)
})
```

## Metrics Collection

The telemetry package provides OpenTelemetry metrics that are compatible with Prometheus and other monitoring systems. Metrics complement distributed tracing by providing aggregated statistical data.

### Metrics Architecture

**Export Configuration:**
- Protocol: OTLP HTTP
- Endpoint: Same as traces (defaults to `localhost:4318`)
- Export interval: 10 seconds (periodic batching)
- Compression: gzip

**Metric Types:**
1. **Counter** (Int64Counter): Monotonically increasing values
   - Command invocations, error counts, API calls, token usage
2. **Histogram** (Float64Histogram): Distribution of values
   - Command duration, API latency

### Available Metrics

#### Command Metrics

**1. Command Invocations** (Counter)
```
Metric: specular.command.invocations
Type: Counter
Description: Total number of command invocations
Attributes:
  - command: Command name (e.g., "auto", "spec", "plan")
  - status: Invocation status ("started", "success", "failed")
  - [custom attributes]
```

**Example:**
```go
telemetry.RecordCommandInvocation(ctx, "auto", "started",
    attribute.String("profile", "dev"),
    attribute.String("scope", "feature/*"),
)
```

**2. Command Duration** (Histogram)
```
Metric: specular.command.duration
Type: Histogram
Description: Command execution duration in seconds
Attributes:
  - command: Command name
  - [custom attributes]
```

**Example:**
```go
duration := time.Since(start)
telemetry.RecordCommandDuration(ctx, "auto", duration,
    attribute.Int("steps_executed", 5),
)
```

**3. Command Errors** (Counter)
```
Metric: specular.command.errors
Type: Counter
Description: Total command errors
Attributes:
  - command: Command name
  - error_type: Error classification
```

**Example:**
```go
telemetry.RecordCommandError(ctx, "auto", "validation_error")
```

#### Provider Metrics

**1. Provider Calls** (Counter)
```
Metric: specular.provider.calls
Type: Counter
Description: Total provider API calls
Attributes:
  - provider: Provider name (e.g., "anthropic", "openai")
  - operation: Operation type ("generate", "stream", "health")
  - status: Call status ("started", "success", "failed")
  - [custom attributes like model]
```

**Example:**
```go
telemetry.RecordProviderCall(ctx, "anthropic", "generate", "success",
    attribute.String("model", "claude-3-sonnet"),
)
```

**2. Provider Latency** (Histogram)
```
Metric: specular.provider.latency
Type: Histogram
Description: Provider API call latency in seconds
Attributes:
  - provider: Provider name
  - operation: Operation type
  - [custom attributes]
```

**Example:**
```go
duration := time.Since(start)
telemetry.RecordProviderLatency(ctx, "anthropic", "generate", duration,
    attribute.String("model", "claude-3-sonnet"),
)
```

**3. Provider Errors** (Counter)
```
Metric: specular.provider.errors
Type: Counter
Description: Total provider API errors
Attributes:
  - provider: Provider name
  - operation: Operation type
  - error_type: Error classification
```

**Example:**
```go
telemetry.RecordProviderError(ctx, "anthropic", "generate", "api_error")
```

**4. Provider Tokens** (Counter)
```
Metric: specular.provider.tokens
Type: Counter
Description: Token usage by provider and model
Attributes:
  - provider: Provider name
  - model: Model name
  - token_type: Token type ("input", "output")
```

**Example:**
```go
telemetry.RecordProviderTokens(ctx, "anthropic", "claude-3-sonnet", "input", 1234)
telemetry.RecordProviderTokens(ctx, "anthropic", "claude-3-sonnet", "output", 567)
```

### Complete Integration Example

```go
func executeCommand(ctx context.Context, cmdName string, args []string) error {
    // Start tracing
    ctx, span := telemetry.StartCommandSpan(ctx, cmdName)
    defer span.End()

    start := time.Now()

    // Execute command logic
    result, err := performWork(ctx, args)

    duration := time.Since(start)

    if err != nil {
        // Record failure in both traces and metrics
        telemetry.RecordCommandFailure(ctx, span, cmdName, err)
        return err
    }

    // Record success in both traces and metrics
    telemetry.RecordCommandSuccess(ctx, span, cmdName, duration,
        attribute.Int("items_processed", result.Count),
    )

    return nil
}

func callProvider(ctx context.Context, provider string, prompt string) (*Response, error) {
    // Start tracing
    ctx, span := telemetry.StartProviderSpan(ctx, provider, "generate")
    defer span.End()

    span.SetAttributes(
        attribute.String("model", "claude-3-sonnet"),
        attribute.Int("prompt_length", len(prompt)),
    )

    start := time.Now()

    // Make API call
    resp, err := api.Generate(ctx, prompt)

    duration := time.Since(start)

    if err != nil {
        // Record failure in both traces and metrics
        telemetry.RecordProviderFailure(ctx, span, provider, "generate", err)
        return nil, err
    }

    // Record success in both traces and metrics
    telemetry.RecordProviderSuccess(ctx, span, provider, "generate", duration,
        attribute.String("model", resp.Model),
        attribute.Int("prompt_tokens", resp.PromptTokens),
        attribute.Int("completion_tokens", resp.CompletionTokens),
    )

    // Record token usage
    telemetry.RecordProviderTokens(ctx, provider, resp.Model, "input", resp.PromptTokens)
    telemetry.RecordProviderTokens(ctx, provider, resp.Model, "output", resp.CompletionTokens)

    return resp, nil
}
```

### Querying Metrics

**Prometheus Queries:**

```promql
# Command invocation rate
rate(specular_command_invocations_total[5m])

# Command success rate
rate(specular_command_invocations_total{status="success"}[5m])
/ rate(specular_command_invocations_total[5m])

# Average command duration
rate(specular_command_duration_sum[5m])
/ rate(specular_command_duration_count[5m])

# 95th percentile command duration
histogram_quantile(0.95, rate(specular_command_duration_bucket[5m]))

# Provider API error rate
rate(specular_provider_errors_total[5m])

# Total tokens used by provider
sum by (provider, model) (specular_provider_tokens_total)

# Average provider latency by operation
rate(specular_provider_latency_sum[5m])
/ rate(specular_provider_latency_count[5m])
```

### Metrics Best Practices

1. **Use Unified Success/Failure Recording**
   ```go
   // GOOD: Records both traces and metrics
   telemetry.RecordCommandSuccess(ctx, span, "auto", duration)

   // BAD: Only records trace, misses metrics
   telemetry.RecordSuccess(span)
   ```

2. **Always Record Token Usage**
   ```go
   // After successful provider call
   telemetry.RecordProviderTokens(ctx, provider, model, "input", promptTokens)
   telemetry.RecordProviderTokens(ctx, provider, model, "output", completionTokens)
   ```

3. **Add Meaningful Attributes**
   ```go
   // Attributes help with debugging and analysis
   telemetry.RecordCommandInvocation(ctx, "auto", "started",
       attribute.String("profile", profile),
       attribute.String("scope", scope),
       attribute.Int("max_steps", maxSteps),
   )
   ```

4. **Classify Errors Properly**
   ```go
   // Use specific error types for better alerting
   if errors.Is(err, ErrValidation) {
       telemetry.RecordCommandError(ctx, "auto", "validation_error")
   } else if errors.Is(err, ErrTimeout) {
       telemetry.RecordCommandError(ctx, "auto", "timeout_error")
   } else {
       telemetry.RecordCommandError(ctx, "auto", "execution_error")
   }
   ```

## Provider Health Checks

The doctor command includes comprehensive provider health checks that test actual API connectivity, not just environment variable presence.

### How It Works

Health checks:
1. Load provider registry using `LoadRegistryWithAutoDiscovery()`
2. Test each configured provider concurrently
3. Call `provider.Health(ctx)` with 10-second timeout
4. Measure and report latency
5. Warn if latency > 5 seconds

### Usage

```bash
# Run doctor command to check all providers
specular doctor

# Example output:
✓ Provider: anthropic
  Status: ok
  API connectivity verified (latency: 234ms)

⚠ Provider: openai
  Status: warning
  API connectivity verified (latency: 6543ms) - High latency detected

✗ Provider: cohere (API)
  Status: error
  Health check failed: authentication failed
  Error: invalid API key
```

### Implementation Details

The health check implementation in `internal/cmd/doctor.go`:

```go
func checkProviderHealth(report *DoctorReport) {
    // Load provider registry
    registry, err := provider.LoadRegistryWithAutoDiscovery(providerConfigPath)
    if err != nil {
        return  // Skip if registry can't be loaded
    }

    providerNames := registry.List()
    var wg sync.WaitGroup
    var mu sync.Mutex

    for _, name := range providerNames {
        wg.Add(1)

        go func(providerName string) {
            defer wg.Done()

            prov, err := registry.Get(providerName)
            if err != nil {
                return
            }

            check := &DoctorCheck{
                Name: fmt.Sprintf("%s (API)", providerName),
            }

            // Health check with timeout
            healthCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
            defer cancel()

            start := time.Now()
            healthErr := prov.Health(healthCtx)
            latency := time.Since(start)

            details := map[string]interface{}{
                "latency_ms": latency.Milliseconds(),
            }

            if healthErr != nil {
                check.Status = "error"
                check.Message = fmt.Sprintf("Health check failed: %v", healthErr)
                details["error"] = healthErr.Error()
            } else {
                check.Status = "ok"
                check.Message = fmt.Sprintf("API connectivity verified (latency: %dms)", latency.Milliseconds())

                // Warn on high latency
                if latency.Milliseconds() > 5000 {
                    check.Status = "warning"
                    check.Message += " - High latency detected"
                }
            }

            check.Details = details

            mu.Lock()
            report.Providers[providerName+" (API)"] = check
            mu.Unlock()
        }(name)
    }

    wg.Wait()
}
```

### Provider Interface

All providers implement the `ProviderClient` interface defined in `internal/provider/interface.go`:

```go
type ProviderClient interface {
    // Health performs a health check on the provider.
    // Returns nil if healthy, error describing the problem otherwise.
    Health(ctx context.Context) error

    // ... other methods ...
}
```

### Implementing Provider Health Checks

Example implementation for a custom provider:

```go
type MyProvider struct {
    apiKey string
    client *http.Client
}

func (p *MyProvider) Health(ctx context.Context) error {
    // Create minimal API request to test connectivity
    req, err := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/v1/health", nil)
    if err != nil {
        return fmt.Errorf("failed to create health check request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+p.apiKey)

    resp, err := p.client.Do(req)
    if err != nil {
        return fmt.Errorf("health check request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, body)
    }

    return nil
}
```

### Health Check Best Practices

1. **Keep Health Checks Lightweight**
   - Use dedicated health endpoints when available
   - Avoid expensive operations
   - Use minimal request payload

2. **Implement Proper Timeouts**
   ```go
   ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
   defer cancel()
   err := provider.Health(ctx)
   ```

3. **Return Meaningful Errors**
   ```go
   // GOOD: Specific error messages
   return fmt.Errorf("authentication failed: invalid API key")

   // BAD: Generic error
   return fmt.Errorf("health check failed")
   ```

4. **Test Actual API Connectivity**
   ```go
   // GOOD: Tests real API endpoint
   resp, err := client.Get("https://api.example.com/v1/health")

   // BAD: Only checks environment variable
   if apiKey == "" {
       return fmt.Errorf("API key not set")
   }
   ```

## Circuit Breaker

The circuit breaker protects against cascading failures when the OTLP collector is unavailable or experiencing issues.

### How It Works

The circuit breaker has three states:

1. **Closed (Normal Operation)**
   - All export requests are allowed through
   - Failures are counted but exports continue

2. **Open (Failing Fast)**
   - Triggered after 5 consecutive export failures
   - All export requests are rejected immediately
   - Prevents overwhelming a struggling collector
   - Returns error: "circuit breaker open: too many export failures"

3. **Half-Open (Testing Recovery)**
   - Automatically transitions after 30 seconds
   - Allows one request through to test if collector recovered
   - Success → returns to Closed state
   - Failure → returns to Open state

### Configuration

Circuit breaker configuration is built-in and cannot be modified:

```go
type circuitBreaker struct {
    failureThreshold int           // 5 failures
    resetTimeout     time.Duration // 30 seconds
    state            string         // "closed", "open", "half-open"
}
```

### Behavior Examples

**Scenario 1: Temporary Collector Outage**
```
Time 0s:  Collector goes down
Time 0s:  Exports 1-5 fail → Circuit opens
Time 1s:  Export 6 rejected (circuit open)
Time 30s: Circuit transitions to half-open
Time 31s: Export 7 succeeds → Circuit closes
Time 32s: Normal operation resumes
```

**Scenario 2: Persistent Collector Issues**
```
Time 0s:  Collector degraded
Time 0s:  Exports 1-5 fail → Circuit opens
Time 30s: Circuit transitions to half-open
Time 31s: Export 6 fails → Circuit reopens
Time 61s: Circuit transitions to half-open again
... repeats until collector recovers
```

### Monitoring Circuit Breaker

Check circuit breaker state in logs (when verbose logging enabled):
```
Circuit breaker open: too many export failures
```

## Retry Logic

Export failures trigger exponential backoff retry logic to handle transient network issues.

### Retry Configuration

Built-in retry parameters:

```go
const (
    initialInterval = 100 * time.Millisecond  // Start at 100ms
    maxInterval     = 2 * time.Second         // Cap at 2s
    maxElapsedTime  = 10 * time.Second        // Total timeout
    multiplier      = 1.5                     // Backoff factor
    maxRetries      = 5                       // Maximum attempts
)
```

### Retry Behavior

**Retry Schedule:**
1. First retry: 100ms delay
2. Second retry: 150ms delay (100ms × 1.5)
3. Third retry: 225ms delay (150ms × 1.5)
4. Fourth retry: 338ms delay (225ms × 1.5)
5. Fifth retry: 507ms delay (338ms × 1.5)

**Total maximum time:** ~1.3 seconds (retry delays) or 10 seconds (elapsed time), whichever comes first.

### Context Cancellation

Retries respect context cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

shutdown, err := telemetry.InitProvider(ctx, cfg)
// Export attempts will stop if context is cancelled
```

### Retry Examples

**Scenario 1: Transient Network Blip**
```
Attempt 1: Fails (network timeout)
Wait 100ms
Attempt 2: Succeeds
Result: Span exported successfully after 1 retry
```

**Scenario 2: Network Congestion**
```
Attempt 1: Fails (connection refused)
Wait 100ms
Attempt 2: Fails (connection refused)
Wait 150ms
Attempt 3: Succeeds
Result: Span exported successfully after 2 retries
Circuit breaker: 2 failures recorded (threshold: 5)
```

**Scenario 3: Persistent Failure**
```
Attempt 1-5: All fail
Result: Final error returned
Circuit breaker: 5 failures recorded → Opens
Next exports: Rejected immediately until circuit resets
```

## Integration Examples

### Example 1: Command Execution Tracing

```go
func executeCommand(ctx context.Context, cmd string) error {
    tracer := telemetry.GetTracerProvider().Tracer("commands")
    ctx, span := tracer.Start(ctx, "command.execute")
    defer span.End()

    // Add command metadata
    span.SetAttributes(
        attribute.String("command", cmd),
        attribute.String("user", getCurrentUser()),
    )

    // Execute command
    err := runCommand(ctx, cmd)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }

    span.SetStatus(codes.Ok, "")
    return nil
}
```

### Example 2: Provider API Call Tracing

```go
func callProvider(ctx context.Context, prompt string) (*Response, error) {
    tracer := telemetry.GetTracerProvider().Tracer("provider")
    ctx, span := tracer.Start(ctx, "provider.generate")
    defer span.End()

    start := time.Now()

    // Add request metadata
    span.SetAttributes(
        attribute.String("provider", "anthropic"),
        attribute.String("model", "claude-3-sonnet"),
        attribute.Int("prompt_length", len(prompt)),
    )

    // Make API call
    resp, err := apiClient.Generate(ctx, prompt)

    duration := time.Since(start)

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }

    // Add response metadata
    span.SetAttributes(
        attribute.Int("prompt_tokens", resp.PromptTokens),
        attribute.Int("completion_tokens", resp.CompletionTokens),
        attribute.Float64("duration_ms", duration.Seconds()*1000),
        attribute.Float64("cost", resp.Cost),
    )

    span.SetStatus(codes.Ok, "")
    return resp, nil
}
```

### Example 3: Nested Spans (Auto Mode Workflow)

```go
func executeWorkflow(ctx context.Context, goal string) error {
    tracer := telemetry.GetTracerProvider().Tracer("auto")

    // Parent span: entire workflow
    ctx, workflowSpan := tracer.Start(ctx, "auto.workflow")
    defer workflowSpan.End()

    workflowSpan.SetAttributes(
        attribute.String("goal", goal),
    )

    // Child span: spec generation
    ctx, specSpan := tracer.Start(ctx, "auto.spec_generation")
    spec, err := generateSpec(ctx, goal)
    specSpan.End()
    if err != nil {
        workflowSpan.RecordError(err)
        return err
    }

    // Child span: plan generation
    ctx, planSpan := tracer.Start(ctx, "auto.plan_generation")
    plan, err := generatePlan(ctx, spec)
    planSpan.SetAttributes(
        attribute.Int("feature_count", len(plan.Features)),
        attribute.Int("task_count", len(plan.Tasks)),
    )
    planSpan.End()
    if err != nil {
        workflowSpan.RecordError(err)
        return err
    }

    // Child spans: step execution (multiple)
    for i, step := range plan.Steps {
        _, stepSpan := tracer.Start(ctx, "auto.step_execution")
        stepSpan.SetAttributes(
            attribute.Int("step_index", i),
            attribute.String("step_type", step.Type),
        )

        err := executeStep(ctx, step)
        stepSpan.End()
        if err != nil {
            workflowSpan.RecordError(err)
            return err
        }
    }

    workflowSpan.SetStatus(codes.Ok, "")
    return nil
}
```

### Example 4: Concurrent Operations

```go
func processBatch(ctx context.Context, items []Item) error {
    tracer := telemetry.GetTracerProvider().Tracer("batch")
    ctx, batchSpan := tracer.Start(ctx, "batch.process")
    defer batchSpan.End()

    batchSpan.SetAttributes(
        attribute.Int("batch_size", len(items)),
    )

    var wg sync.WaitGroup
    errChan := make(chan error, len(items))

    for i, item := range items {
        wg.Add(1)
        go func(idx int, it Item) {
            defer wg.Done()

            // Each goroutine gets its own span
            _, itemSpan := tracer.Start(ctx, "batch.process_item")
            defer itemSpan.End()

            itemSpan.SetAttributes(
                attribute.Int("item_index", idx),
                attribute.String("item_id", it.ID),
            )

            if err := processItem(ctx, it); err != nil {
                itemSpan.RecordError(err)
                errChan <- err
            }
        }(i, item)
    }

    wg.Wait()
    close(errChan)

    // Check for errors
    var errors []error
    for err := range errChan {
        errors = append(errors, err)
    }

    if len(errors) > 0 {
        batchSpan.SetStatus(codes.Error, fmt.Sprintf("%d items failed", len(errors)))
        return fmt.Errorf("%d items failed", len(errors))
    }

    batchSpan.SetStatus(codes.Ok, "")
    return nil
}
```

## Performance Characteristics

### Benchmark Results (darwin/arm64, Apple M1)

**Telemetry Disabled (Noop Provider):**
```
BenchmarkNoopProvider-8         35.24 ns/op     48 B/op     1 alloc/op
BenchmarkGetTracerProvider-8    14.01 ns/op      0 B/op     0 allocs/op
```

**Telemetry Enabled (OTLP Export):**
```
BenchmarkBatchProcessor-8       454.0 ns/op   2623 B/op     3 allocs/op
BenchmarkSpanWithSampling-8     542.7 ns/op   2176 B/op     3 allocs/op
```

**Development/Debug Scenarios:**
```
BenchmarkSpanCreation-8        1100 ns/op     4063 B/op     5 allocs/op
BenchmarkSpanWithAttributes-8  1496 ns/op     4568 B/op     7 allocs/op
BenchmarkNestedSpans-8         1950 ns/op     7886 B/op    11 allocs/op
```

### Performance Recommendations

1. **Production Deployment:**
   - Disable telemetry by default: `Enabled: false`
   - Enable only for debugging specific issues
   - Overhead when disabled: ~35 ns/op (negligible)

2. **High-Traffic Services:**
   - Use sampling: `SampleRate: 0.1` (10%)
   - Reduces export volume by 90%
   - Maintains statistical visibility

3. **Development/Staging:**
   - Enable full tracing: `SampleRate: 1.0`
   - Endpoint: local OTLP collector
   - Full visibility into system behavior

4. **Batch Size Tuning:**
   - Default: 512 spans per batch, 5s timeout
   - Balances latency vs throughput
   - Automatically configured, no tuning needed

## Troubleshooting

### Problem: Spans Not Appearing in Collector

**Symptoms:**
- Code runs without errors
- No spans visible in tracing UI (Jaeger, Zipkin, etc.)

**Possible Causes & Solutions:**

1. **Telemetry Disabled**
   ```go
   // Check: cfg.Enabled should be true
   cfg := telemetry.Config{
       Enabled: true,  // Must be true
       // ...
   }
   ```

2. **Wrong Endpoint**
   ```go
   // Check: endpoint should point to OTLP HTTP collector
   cfg.Endpoint = "localhost:4318"  // Not gRPC port 4317
   ```

3. **Sampling Excluded Trace**
   ```go
   // Check: with 10% sampling, 90% of traces are dropped
   cfg.SampleRate = 1.0  // Use 100% for testing
   ```

4. **Shutdown Not Called**
   ```go
   shutdown, _ := telemetry.InitProvider(ctx, cfg)
   defer shutdown(ctx)  // Must call to flush pending spans
   ```

5. **Collector Not Running**
   ```bash
   # Test if collector is reachable
   curl http://localhost:4318/v1/traces
   ```

### Problem: Circuit Breaker Opening Frequently

**Symptoms:**
- Error messages: "circuit breaker open: too many export failures"
- Spans being dropped

**Possible Causes & Solutions:**

1. **Collector Unavailable**
   - Check collector health: `curl http://localhost:4318`
   - Verify network connectivity
   - Check collector logs for errors

2. **Network Timeout Issues**
   - Increase collector timeout configuration
   - Check for network congestion
   - Verify firewall rules

3. **Collector Overloaded**
   - Reduce sampling rate: `cfg.SampleRate = 0.1`
   - Scale collector horizontally
   - Increase collector resources

### Problem: High Memory Usage

**Symptoms:**
- Application memory grows over time
- Out of memory errors

**Possible Causes & Solutions:**

1. **Batch Processor Not Flushing**
   ```go
   // Ensure shutdown is called to flush pending spans
   shutdown, _ := telemetry.InitProvider(ctx, cfg)
   defer shutdown(ctx)
   ```

2. **Too Many Attributes**
   ```go
   // Avoid adding large payloads as attributes
   // BAD: span.SetAttributes(attribute.String("response", largeJSON))
   // GOOD: span.SetAttributes(attribute.Int("response_size", len(largeJSON)))
   ```

3. **Span Not Ended**
   ```go
   // Always end spans, preferably with defer
   ctx, span := tracer.Start(ctx, "operation")
   defer span.End()  // Ensures span is ended even if panic occurs
   ```

### Problem: Performance Degradation

**Symptoms:**
- Application slower after enabling telemetry
- High CPU usage

**Possible Causes & Solutions:**

1. **Too Much Instrumentation**
   - Instrument only critical paths
   - Avoid tracing hot loops
   - Use sampling for high-frequency operations

2. **Synchronous Export**
   - Default configuration uses batch processor (asynchronous)
   - Verify not using simple processor in custom setup

3. **Excessive Attributes**
   - Limit number of attributes per span (< 20)
   - Use semantic conventions for standard attributes
   - Aggregate data before adding as attributes

### Debug Mode

Enable verbose logging to debug telemetry issues:

```go
// Set log level to debug
import "github.com/felixgeelhaar/specular/internal/log"

logger := log.New(log.Config{
    Level: log.LevelDebug,
    Format: log.FormatJSON,
})
```

Check for telemetry-related log messages:
- "failed to create OTLP exporter"
- "circuit breaker open"
- "export failed after N attempts"
- "failed to start runtime instrumentation"

## Best Practices

1. **Always Use Context**
   ```go
   // GOOD: Pass context through call chain
   func operation(ctx context.Context) {
       ctx, span := tracer.Start(ctx, "operation")
       defer span.End()
       subOperation(ctx)  // Pass context
   }

   // BAD: Don't break context chain
   func operation(ctx context.Context) {
       ctx, span := tracer.Start(ctx, "operation")
       defer span.End()
       subOperation(context.Background())  // Lost trace context!
   }
   ```

2. **Use Defer for Span.End()**
   ```go
   // GOOD: Span always ended, even on panic
   ctx, span := tracer.Start(ctx, "operation")
   defer span.End()

   // BAD: Span leaked if error occurs
   ctx, span := tracer.Start(ctx, "operation")
   if err := doWork(); err != nil {
       return err  // Span never ended!
   }
   span.End()
   ```

3. **Record Errors**
   ```go
   if err != nil {
       span.RecordError(err)
       span.SetStatus(codes.Error, err.Error())
       return err
   }
   span.SetStatus(codes.Ok, "")
   ```

4. **Use Semantic Attributes**
   ```go
   // Use standard attribute names from semconv
   import "go.opentelemetry.io/otel/semconv/v1.4.0"

   span.SetAttributes(
       semconv.HTTPMethodKey.String("GET"),
       semconv.HTTPStatusCodeKey.Int(200),
   )
   ```

5. **Name Spans Consistently**
   ```go
   // GOOD: Hierarchical naming
   "auto.workflow"
   "auto.spec_generation"
   "auto.plan_generation"
   "auto.step_execution"

   // BAD: Inconsistent naming
   "DoWorkflow"
   "generate_spec"
   "PlanGeneration"
   "execute-step"
   ```

## Further Reading

- [OpenTelemetry Go SDK Documentation](https://opentelemetry.io/docs/instrumentation/go/)
- [OTLP Specification](https://opentelemetry.io/docs/reference/specification/protocol/otlp/)
- [Semantic Conventions](https://opentelemetry.io/docs/reference/specification/trace/semantic_conventions/)
- [ADR 0009: Observability & Monitoring Strategy](./adr/0009-observability-monitoring-strategy.md)
- [Internal Telemetry Package Documentation](https://pkg.go.dev/github.com/felixgeelhaar/specular/internal/telemetry)
