# ADR 0017: M9.3 Observability & Monitoring Implementation

## Status

Accepted

## Context

Building on the observability foundation established in ADR-0009 (Observability & Monitoring Strategy), M9.3 implements the enterprise-grade features required for production operations:

1. **Existing Foundation (ADR-0009)**:
   - Structured logging with `log/slog` (~3,300 lines, 95%+ coverage)
   - Prometheus metrics via OpenTelemetry (~1,500 lines, 90%+ coverage)
   - Distributed tracing (~1,800 lines, 89% coverage)
   - Health checks with K8s probes (~1,700 lines, 78-100% coverage)
   - 40+ metrics already defined (command, provider, spec, plan, task, docker, policy, drift, auto, interview)

2. **Enterprise Requirements**:
   - **Correlation IDs**: Track requests across components and services
   - **SLO/SLI Framework**: Define and monitor Service Level Objectives
   - **Alert Routing**: Integrate with enterprise incident management
   - **Log Aggregation**: Export logs to centralized systems (ELK, Splunk, Loki)
   - **Operational Tooling**: CLI commands for observability management
   - **Dashboard Templates**: Ready-to-use Grafana dashboards

3. **Compliance Requirements**:
   - **SOC2**: Incident management and monitoring capabilities
   - **ISO 27001**: Audit logging and alerting mechanisms
   - **HIPAA**: Breach notification support (alerting integration)
   - **PCI DSS**: Monitoring and incident response

## Decision

We will implement a comprehensive observability extension with six components:

### 1. Correlation ID Support (`internal/log/correlation.go`)

Request correlation across components using context propagation:

```go
// Context-based correlation ID propagation
func WithCorrelationID(ctx context.Context, id string) context.Context
func CorrelationID(ctx context.Context) string
func NewCorrelationID() string
func MustNewCorrelationID() string

// HTTP middleware for automatic correlation
func CorrelationMiddleware(next http.Handler) http.Handler
```

**Features**:
- UUID v4 generation for unique correlation IDs
- HTTP middleware for automatic extraction/injection
- Integration with slog Logger for automatic correlation in log entries
- Works with existing distributed tracing (TraceID/SpanID)

### 2. SLO/SLI Framework (`internal/slo/`)

Service Level Objective definition and tracking:

```go
// SLO definition
type SLO struct {
    Name        string        `yaml:"name"`
    Description string        `yaml:"description"`
    Target      float64       `yaml:"target"`      // e.g., 0.995 for 99.5%
    Window      time.Duration `yaml:"window"`      // e.g., 30d
    SLI         SLISpec       `yaml:"sli"`
}

type SLISpec struct {
    Type      SLIType `yaml:"type"`      // availability, latency, throughput
    Metric    string  `yaml:"metric"`    // Prometheus metric name
    Threshold float64 `yaml:"threshold"` // For latency SLIs
}

// SLI calculation
type SLIResult struct {
    SLOName          string
    Value            float64
    Target           float64
    ErrorBudget      float64    // Remaining budget (0-1)
    BurnRate         float64    // Budget consumption rate
    IsHealthy        bool
    ProjectedBudget  float64    // Budget at window end
}
```

**Default SLOs**:
| SLO | Target | Metric | Window |
|-----|--------|--------|--------|
| Command Success Rate | 99.5% | `specular_command_executions_total` | 30 days |
| Provider Latency p95 | 95% < 30s | `specular_provider_latency_seconds` | 30 days |
| Auto Mode Success | 99% | `specular_auto_workflows_total` | 30 days |
| Spec Validation | 99% | `specular_spec_validations_total` | 30 days |

### 3. Alert Routing (`internal/alerting/`)

Enterprise incident management integration:

```go
type Alert struct {
    ID          string
    Severity    Severity   // Critical, High, Warning, Info
    Title       string
    Description string
    Labels      map[string]string
    DedupeKey   string
    Timestamp   time.Time
}

type AlertManager interface {
    Send(ctx context.Context, alert *Alert) error
    Resolve(ctx context.Context, dedupeKey string) error
    Test(ctx context.Context) error
    Name() string
}
```

**Supported Destinations**:
- **PagerDuty**: Events API v2 with routing keys
- **Opsgenie**: US/EU regions, team routing
- **Slack**: Webhook integration with rich formatting
- **Webhook**: Generic HTTP with HMAC signature verification

**Router Pattern**:
```go
// Route alerts to multiple destinations by severity
router := alerting.NewRouter().
    Route(alerting.SeverityCritical, pagerduty).
    Route(alerting.SeverityHigh, opsgenie).
    Route(alerting.SeverityWarning, slack).
    RouteDefault(webhook)
```

### 4. Log Export (`internal/logexport/`)

Enterprise log aggregation support:

```go
type Exporter interface {
    Export(ctx context.Context, entries []LogEntry) error
    Close() error
    Name() string
    Healthy(ctx context.Context) bool
}

type LogEntry struct {
    Timestamp     time.Time
    Level         LogLevel
    Message       string
    CorrelationID string
    TraceID       string
    SpanID        string
    Attributes    map[string]any
}
```

**Supported Systems**:
- **Elasticsearch/ELK**: Bulk API with date-pattern indexes
- **Splunk**: HTTP Event Collector (HEC)
- **Grafana Loki**: Push API with label grouping

**Buffered Export**:
```go
// Batch logs for efficient export
buffered := logexport.NewBufferedExporter(exporter, logexport.Config{
    BatchSize:     100,
    FlushInterval: 5 * time.Second,
    MaxRetries:    3,
    RetryDelay:    time.Second,
})
```

### 5. Configuration (`internal/cmd/config.go`)

```go
type ObservabilityConfig struct {
    SLO       SLOConfig       `yaml:"slo,omitempty"`
    Alerting  AlertingConfig  `yaml:"alerting,omitempty"`
    LogExport LogExportConfig `yaml:"log_export,omitempty"`
}

type SLOConfig struct {
    Enabled    bool     `yaml:"enabled,omitempty"`
    ConfigPath string   `yaml:"config_path,omitempty"`
    SLOs       []*SLO   `yaml:"slos,omitempty"`
}

type AlertingConfig struct {
    Enabled   bool               `yaml:"enabled,omitempty"`
    PagerDuty PagerDutyConfig    `yaml:"pagerduty,omitempty"`
    Opsgenie  OpsgenieConfig     `yaml:"opsgenie,omitempty"`
    Slack     SlackConfig        `yaml:"slack,omitempty"`
    Webhook   WebhookAlertConfig `yaml:"webhook,omitempty"`
}

type LogExportConfig struct {
    Type     string `yaml:"type,omitempty"`     // elk, splunk, loki
    Endpoint string `yaml:"endpoint,omitempty"`
    APIKey   string `yaml:"api_key,omitempty"`
}
```

### 6. CLI Commands (`internal/cmd/observability_cmd.go`)

Operational tooling for observability management:

```bash
# Overall status
specular observability status
specular obs status  # alias

# SLO management
specular observability slo status
specular observability slo burn-rate
specular observability slo config

# Alert testing
specular observability alerts test
specular observability alerts status

# Metrics overview
specular observability metrics
```

### 7. Grafana Dashboards (`deployments/grafana/`)

Ready-to-use dashboard templates:

**specular-overview.json**:
- Command Success Rate (stat)
- Provider Latency p95 (stat)
- Error Rate (stat)
- Total Tokens Used (stat)
- Command Execution Rate (timeseries)
- Command Duration p95 (timeseries)
- Provider Latency by Percentile (timeseries)
- Token Usage by Provider (timeseries)

**specular-slos.json**:
- SLO Status gauges (Command Success, Provider Latency, Auto Mode, Spec Validation)
- Error Budget gauges (remaining budget for each SLO)
- Burn Rate charts (1h and 6h burn rates with alerting thresholds)
- SLI Trends over time

## Architecture

### Request Flow with Correlation

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Request Lifecycle                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────┐    ┌────────────┐    ┌──────────────┐    ┌───────────┐   │
│  │  CLI     │───>│ Correlation│───>│   Business   │───>│  Provider │   │
│  │ Command  │    │ Middleware │    │    Logic     │    │   API     │   │
│  └──────────┘    └────────────┘    └──────────────┘    └───────────┘   │
│       │               │                   │                  │          │
│       │               │                   │                  │          │
│       ▼               ▼                   ▼                  ▼          │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                     Context (correlation_id)                    │    │
│  └────────────────────────────────────────────────────────────────┘    │
│       │               │                   │                  │          │
│       ▼               ▼                   ▼                  ▼          │
│  ┌─────────┐    ┌──────────┐    ┌──────────────┐    ┌───────────┐      │
│  │  Logs   │    │  Metrics │    │   Traces     │    │   Alerts  │      │
│  │ (slog)  │    │(OTel/Prom)│   │ (OTel/Jaeger)│    │(PD/Slack) │      │
│  └─────────┘    └──────────┘    └──────────────┘    └───────────┘      │
│       │               │                   │                  │          │
│       ▼               ▼                   ▼                  ▼          │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │              Log Export (ELK/Splunk/Loki)                       │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### SLO/Error Budget Calculation

```
Error Budget = 1 - SLO Target
Consumed Budget = 1 - (Current SLI / Target)
Burn Rate = Consumed Budget / Time Elapsed

Example (99.5% availability SLO over 30 days):
- Error Budget: 0.5% (≈ 21.6 minutes of downtime)
- If 10 minutes consumed in first 7 days: Burn Rate = 1.85x
- Projected budget at window end: ~55% remaining
```

### Alert Routing Pattern

```
┌─────────────────┐
│  Alert Event    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│     Router      │
├─────────────────┤
│ severity rules  │
└────────┬────────┘
         │
    ┌────┴────┬──────────┬──────────┐
    │         │          │          │
    ▼         ▼          ▼          ▼
┌───────┐ ┌───────┐ ┌────────┐ ┌────────┐
│PageDuty│ │Opsgenie│ │ Slack  │ │Webhook │
│Critical│ │ High   │ │Warning │ │Default │
└───────┘ └───────┘ └────────┘ └────────┘
```

## File Structure

```
internal/
├── log/
│   ├── correlation.go           # Correlation ID support
│   └── correlation_test.go      # Unit tests
├── slo/
│   ├── slo.go                   # SLO definitions
│   ├── sli.go                   # SLI calculations
│   ├── tracker.go               # SLO tracking
│   ├── config.go                # Configuration
│   └── *_test.go                # Tests
├── alerting/
│   ├── alerting.go              # Interface & Router
│   ├── pagerduty.go             # PagerDuty integration
│   ├── opsgenie.go              # Opsgenie integration
│   ├── slack.go                 # Slack integration
│   ├── webhook.go               # Generic webhook
│   └── alerting_test.go         # Tests
├── logexport/
│   ├── exporter.go              # Interface & BufferedExporter
│   ├── elk.go                   # Elasticsearch/ELK
│   ├── splunk.go                # Splunk HEC
│   ├── loki.go                  # Grafana Loki
│   └── exporter_test.go         # Tests
└── cmd/
    ├── config.go                # ObservabilityConfig
    └── observability_cmd.go     # CLI commands

deployments/
└── grafana/
    ├── dashboards/
    │   ├── specular-overview.json
    │   └── specular-slos.json
    └── README.md
```

## Security Considerations

### Alert Credentials

- API keys and tokens stored in configuration (not hardcoded)
- Webhook signatures using HMAC-SHA256 for verification
- TLS/HTTPS enforced for all external communications
- Support for environment variable substitution

### Log Data Protection

- No PII in default log fields
- Correlation IDs are UUIDs (not user-identifiable)
- Log export supports authentication (API keys, basic auth)
- Sensitive fields can be redacted via configuration

### Webhook Security

```go
// HMAC signature verification for incoming webhooks
func VerifySignature(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

## Performance Characteristics

| Operation | Latency (p50) | Latency (p99) | Notes |
|-----------|---------------|---------------|-------|
| **Correlation ID generation** | ~100ns | ~200ns | UUID v4 |
| **SLI calculation** | ~1ms | ~5ms | With Prometheus query |
| **Alert send (PagerDuty)** | ~50ms | ~200ms | HTTP API |
| **Alert send (Slack)** | ~100ms | ~500ms | Webhook |
| **Log export (batch 100)** | ~30ms | ~100ms | ELK bulk API |
| **Buffered export (async)** | <1ms | <5ms | Returns immediately |

## Usage Examples

### Configuration

```yaml
# ~/.specular/config.yaml
observability:
  slo:
    enabled: true
    config_path: ~/.specular/slos.yaml  # optional external file
  alerting:
    enabled: true
    pagerduty:
      routing_key: "${PAGERDUTY_ROUTING_KEY}"
    slack:
      webhook_url: "${SLACK_WEBHOOK_URL}"
      channel: "#alerts"
  log_export:
    type: elk
    endpoint: https://elasticsearch.example.com:9200
    api_key: "${ELASTICSEARCH_API_KEY}"
```

### SLO Definition

```yaml
# ~/.specular/slos.yaml
slos:
  - name: command-success-rate
    description: "Percentage of commands completing successfully"
    target: 0.995  # 99.5%
    window: 720h   # 30 days
    sli:
      type: availability
      metric: specular_command_executions_total

  - name: provider-latency-p95
    description: "95th percentile provider response time"
    target: 0.95   # 95% of requests under threshold
    window: 720h
    sli:
      type: latency
      metric: specular_provider_latency_seconds
      threshold: 30  # 30 seconds
```

### Programmatic Usage

```go
// Create alert with correlation
ctx := log.WithCorrelationID(context.Background(), log.NewCorrelationID())

alert := alerting.NewAlert("High Error Rate", "Command failures above threshold").
    WithSeverity(alerting.SeverityHigh).
    WithLabel("command", "apply").
    WithLabel("correlation_id", log.CorrelationID(ctx)).
    Build()

router.Send(ctx, alert)
```

## Testing

### Test Coverage

| Package | Tests | Coverage |
|---------|-------|----------|
| `internal/log/correlation` | 8 | 95%+ |
| `internal/slo` | 15 | 90%+ |
| `internal/alerting` | 25 | 92%+ |
| `internal/logexport` | 18 | 88%+ |

### Test Categories

1. **Unit Tests**: Individual function/method behavior
2. **Integration Tests**: Component interaction (e.g., SLO with metrics provider)
3. **Mock Server Tests**: HTTP-level API simulation
4. **Failure Scenarios**: Timeout, retry, fallback behavior

## Rollout Plan

### Phase 1: Foundation (COMPLETE)
- Correlation ID support with context propagation
- SLO/SLI framework with default definitions
- Configuration schema updates

### Phase 2: Alerting Integration (COMPLETE)
- PagerDuty Events API v2 integration
- Opsgenie integration (US/EU regions)
- Slack webhook integration
- Generic webhook with HMAC signatures
- Router pattern for multi-destination routing

### Phase 3: Log Export (COMPLETE)
- Elasticsearch/ELK bulk exporter
- Splunk HEC exporter
- Grafana Loki push exporter
- Buffered export with retry logic

### Phase 4: Operational Tooling (COMPLETE)
- CLI commands for observability management
- Grafana dashboard templates
- Documentation and examples

## Alternatives Considered

### SLO Framework

**Option 1: Google SRE Libraries**
- Pros: Industry standard, well-documented
- Cons: Heavy dependencies, Java-centric

**Option 2: OpenSLO Standard**
- Pros: Vendor-neutral specification
- Cons: Limited Go ecosystem support

**Decision**: Custom implementation following OpenSLO principles for flexibility.

### Alerting

**Option 1: Alertmanager Integration**
- Pros: Prometheus ecosystem, deduplication
- Cons: Requires running Alertmanager instance

**Option 2: Direct API Integration**
- Pros: Simpler, no additional infrastructure
- Cons: Must implement deduplication

**Decision**: Direct API integration with built-in deduplication via dedupe keys.

### Log Export

**Option 1: Fluentd/Fluent Bit Sidecar**
- Pros: Standard log shipping, flexible routing
- Cons: Additional infrastructure component

**Option 2: Direct API Export**
- Pros: No additional components, configurable
- Cons: Application-level buffering needed

**Decision**: Direct API export with buffered async delivery.

## Compliance Mapping

| Requirement | Feature | Implementation |
|------------|---------|----------------|
| **SOC2**: Incident response | Alert routing | PagerDuty/Opsgenie integration |
| **SOC2**: Monitoring | SLO tracking | SLI calculations, dashboards |
| **ISO 27001**: Audit trail | Correlation IDs | Request tracing across logs |
| **ISO 27001**: Alerting | Alert managers | Multi-destination routing |
| **HIPAA**: Breach notification | Alerting | PagerDuty escalation policies |
| **PCI DSS**: Monitoring | Metrics/SLOs | Prometheus metrics, Grafana |

## References

- [ADR-0009: Observability & Monitoring Strategy](0009-observability-monitoring-strategy.md)
- [OpenSLO Specification](https://openslo.com/)
- [Google SRE: Service Level Objectives](https://sre.google/sre-book/service-level-objectives/)
- [PagerDuty Events API v2](https://developer.pagerduty.com/docs/events-api-v2/overview/)
- [Slack Incoming Webhooks](https://api.slack.com/messaging/webhooks)
- [Elasticsearch Bulk API](https://www.elastic.co/guide/en/elasticsearch/reference/current/docs-bulk.html)
- [Grafana Loki API](https://grafana.com/docs/loki/latest/api/)

## Decision Makers

- @felixgeelhaar (Architecture, Implementation)

## Date

2026-01-02
