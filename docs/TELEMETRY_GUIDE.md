# Telemetry Configuration & Integration Guide

## Overview

This guide covers the configuration and integration of the production-grade OpenTelemetry tracing system in Specular. The telemetry package provides distributed tracing with OTLP HTTP export, circuit breaker pattern, and exponential backoff retry logic.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Configuration](#configuration)
3. [Circuit Breaker](#circuit-breaker)
4. [Retry Logic](#retry-logic)
5. [Integration Examples](#integration-examples)
6. [Performance Characteristics](#performance-characteristics)
7. [Troubleshooting](#troubleshooting)

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
