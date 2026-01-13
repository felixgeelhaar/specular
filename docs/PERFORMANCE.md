# Performance Guide

This document describes performance characteristics and optimization strategies for Specular CLI.

## Current Metrics

| Metric | Current | Target | Status |
|--------|---------|--------|--------|
| Startup time (version) | ~10ms | <100ms | ✅ Achieved |
| Startup time (doctor) | ~700ms | <500ms | ⚠️ Acceptable |
| Binary size (stripped) | 25MB | <10MB | ⚠️ Limited by deps |
| Memory usage | TBD | <50MB | - |

## Startup Time Optimization

### Fast Command Detection

Commands like `version`, `--help`, and `completion` skip observability initialization:

```go
// internal/cmd/root.go
func isFastCommand() bool {
    // Skips: version, completion, help, --help, -h
}
```

This reduces startup time from ~700ms to ~10ms for these commands.

### Commands by Category

**Fast Commands** (no observability, <50ms):
- `specular version`
- `specular --help`
- `specular <command> --help`
- `specular completion <shell>`

**Standard Commands** (with observability, ~500-1000ms):
- `specular doctor`
- `specular auto`
- `specular generate`
- All other commands

## Binary Size

### Current Distribution

| Platform | Size (stripped) |
|----------|-----------------|
| darwin/amd64 | ~25MB |
| darwin/arm64 | ~25MB |
| linux/amd64 | ~25MB |
| linux/arm64 | ~25MB |
| windows/amd64 | ~25MB |

### Size Contributors

Major dependencies contributing to binary size:
- OpenTelemetry packages (~5MB)
- Charmbracelet TUI libraries (~3MB)
- Google container registry (~2MB)
- Sigstore signing (~2MB)
- Prometheus metrics (~1MB)
- OIDC/SAML authentication (~1MB)

### Optimization Flags

Production builds use these flags:
```makefile
go build -ldflags="-s -w" -trimpath
```

- `-s`: Omit symbol table
- `-w`: Omit DWARF debugging info
- `-trimpath`: Remove file paths from binary

### Future Optimizations

1. **UPX Compression** (Linux only)
   - Can reduce binary size by 50-70%
   - Not supported on macOS
   - Adds startup decompression time (~50ms)

2. **Feature-based Build Tags**
   - Conditional compilation for optional features
   - Example: `go build -tags=minimal` for core-only

3. **Lazy Loading**
   - Load heavy dependencies on-demand
   - Trade startup time for feature access time

## Benchmarking

### Running Benchmarks

```bash
# Binary size comparison
make bench-binary

# Startup time measurement
make bench-startup

# Full performance report
make perf-report

# Go benchmarks
make bench
```

### Benchmark Framework

The `internal/benchmark` package provides:
- `MeasureStartup()` - Measure command startup time
- `GetBinaryInfo()` - Get binary metadata
- `RunBenchmarkSuite()` - Run full benchmark suite

### Example Output

```
Performance Report
==================

Binary Info:
  Size: 25M
  Platform: darwin/arm64
  Go Version: go1.24

Dependencies: 269 modules
Transitive:   690 packages

Startup Times (5 runs):
  specular version: 0.010s avg
  specular --help: 0.012s avg
  specular doctor --help: 0.015s avg
```

## Profiling

### CPU Profiling

```bash
# Generate CPU profile
go test -cpuprofile=cpu.prof -bench=. ./internal/benchmark/

# Analyze profile
go tool pprof cpu.prof
```

### Memory Profiling

```bash
# Generate memory profile
go test -memprofile=mem.prof -bench=. ./internal/benchmark/

# Analyze profile
go tool pprof mem.prof
```

### Execution Tracing

```bash
# Generate execution trace
go test -trace=trace.out -bench=. ./internal/benchmark/

# View trace
go tool trace trace.out
```

## Best Practices

### For Plugin Developers

1. **Minimize startup work** - Defer initialization until needed
2. **Use lazy loading** - Don't load resources until required
3. **Avoid global state** - Reduces initialization overhead
4. **Profile regularly** - Catch regressions early

### For Core Development

1. **Add new dependencies carefully** - Each dep increases binary size
2. **Consider build tags** - For optional features
3. **Test startup impact** - Run `make bench-startup` after changes
4. **Document performance** - Update this guide for significant changes

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SPECULAR_LOG_LEVEL` | Logging verbosity | `info` |
| `SPECULAR_LOG_STDOUT` | Log to stdout | `false` |
| `SPECULAR_TELEMETRY` | Enable telemetry | `false` |

### Disabling Observability

For maximum performance in scripts:

```bash
# Disable all observability
export SPECULAR_LOG_LEVEL=error
export SPECULAR_TELEMETRY=false
```

## Monitoring

### Metrics Endpoints

When running with telemetry enabled:
- Prometheus metrics: Available via OTLP export
- Distributed tracing: Jaeger/Zipkin compatible

### Key Metrics

- `specular_command_duration_seconds` - Command execution time
- `specular_startup_duration_seconds` - Startup time
- `specular_memory_bytes` - Memory usage
