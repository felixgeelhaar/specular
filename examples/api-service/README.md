# WeatherAPI - Go API Service Example

High-performance REST API service built with Go demonstrating:
- 10,000+ RPS throughput
- <50ms latency at p95
- Redis caching layer
- PostgreSQL for historical data
- Prometheus metrics
- Complete CI/CD integration

## Quick Start

```bash
cd examples/api-service

# Generate implementation plan
specular plan --spec .specular/spec.yaml --output plan.json

# Execute build with policy enforcement
specular build --plan plan.json --policy .specular/policy.yaml

# Detect drift
specular eval \
  --spec .specular/spec.yaml \
  --plan plan.json \
  --lock .specular/spec.lock.json \
  --api-spec openapi.yaml
```

## Features

### Current Weather (feat-001)
- **Priority:** P0
- **Endpoint:** `GET /api/v1/weather/current?lat={lat}&lon={lon}`
- **Performance:** <50ms response time
- **Load:** 1000+ concurrent requests

### Weather Forecast (feat-002)
- **Priority:** P0
- **Endpoint:** `GET /api/v1/weather/forecast?lat={lat}&lon={lon}`
- **Data:** 5-day forecast with 3-hour intervals
- **Caching:** Redis-backed with smart TTL

### Historical Data (feat-003)
- **Priority:** P1
- **Endpoint:** `GET /api/v1/weather/historical?lat={lat}&lon={lon}&start_date=...&end_date=...`
- **Storage:** PostgreSQL with optimized indexes
- **Export:** JSON/CSV formats

### Rate Limiting (feat-004)
- **Priority:** P0
- **Endpoints:**
  - `POST /api/v1/keys` - Create API key
  - `GET /api/v1/keys/{id}` - Get key details
  - `DELETE /api/v1/keys/{id}` - Revoke key
- **Implementation:** Redis-based rate limiting
- **Tiers:** free, basic, pro, enterprise

### Monitoring (feat-005)
- **Priority:** P0
- **Endpoints:**
  - `GET /health` - Health check (<10ms)
  - `GET /metrics` - Prometheus metrics
- **Metrics:** Request count, latency, errors, cache hit rate

## Policy Highlights

**Performance:**
- Load testing: 10,000 RPS minimum
- Latency: <100ms p95, <200ms p99
- CPU/Memory profiling enabled

**Security:**
- Docker-only execution
- Dependency scanning (gosec)
- SAST enabled
- Secrets scanning

**Quality:**
- Test coverage >85%
- Race detection enabled
- golangci-lint + staticcheck
- Integration tests required

**SLA:**
- 99.9% uptime
- <100ms latency p95
- <0.1% error rate

## Development Workflow

### Local Development

```bash
# Pre-warm Docker images
specular prewarm golang:1.22-alpine postgres:15-alpine redis:7-alpine

# Run tests with race detection
go test -race -coverprofile=coverage.txt ./...

# Run linter
golangci-lint run

# Build binary
go build -o bin/api ./cmd/api
```

### Load Testing

```bash
# Using k6 (defined in policy)
k6 run loadtest.js

# Expected results:
# - RPS: 10,000+
# - Latency p95: <100ms
# - Latency p99: <200ms
```

### CI/CD Integration

`.github/workflows/api.yml`:
```yaml
name: Weather API CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: felixgeelhaar/specular-action@v1
        with:
          command: build
          spec-file: .specular/spec.yaml
          policy-file: .specular/policy.yaml
          enable-cache: true

      - name: Load Test
        run: k6 run loadtest.js
```

## Performance Optimization

### Caching Strategy
- **Current Weather:** 5-minute TTL
- **Forecast:** 1-hour TTL
- **Historical:** No expiry (static data)
- **Target:** >80% cache hit rate

### Database Optimization
- Indexes on: `(latitude, longitude, timestamp)`
- Partitioning by date range
- Connection pooling: 5-50 connections
- Query timeout: 3 seconds

### Concurrency
- Worker pool pattern for API requests
- Go routines for concurrent processing
- Context-based cancellation
- Graceful shutdown with connection draining

## Monitoring & Alerts

### Prometheus Metrics
- `http_requests_total` - Total request count
- `http_request_duration_seconds` - Request latency histogram
- `cache_hits_total` / `cache_misses_total` - Cache performance
- `db_queries_total` / `db_query_duration_seconds` - Database metrics

### Grafana Dashboards
- Request rate and latency trends
- Cache hit rate over time
- Database query performance
- Error rate by endpoint

### Alerting Rules
- Latency p95 > 150ms (warning)
- Latency p95 > 200ms (critical)
- Error rate > 1% (warning)
- Error rate > 2% (critical)
- Cache hit rate < 70% (warning)

## Learn More

- [Specular Documentation](../../docs/getting-started.md)
- [Go Performance Best Practices](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html)
- [API Design Guide](../../docs/getting-started.md#api-validation)
