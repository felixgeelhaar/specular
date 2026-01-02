# Specular Grafana Dashboards

This directory contains Grafana dashboard templates for monitoring Specular CLI operations.

## Dashboards

### specular-overview.json

Overview dashboard providing a high-level view of Specular operations:

**Panels:**
- Command Success Rate (stat)
- Provider Latency p95 (stat)
- Error Rate (stat)
- Total Tokens Used (stat)
- Command Execution Rate (timeseries)
- Command Duration p95 (timeseries)
- Provider Latency by Percentile (timeseries)
- Token Usage by Provider (timeseries)

### specular-slos.json

SLO/SLI tracking dashboard for service level monitoring:

**Panels:**
- SLO Status gauges (Command Success, Provider Latency, Auto Mode, Spec Validation)
- Error Budget gauges (remaining budget for each SLO)
- Burn Rate charts (1h and 6h burn rates with alerting thresholds)
- SLI Trends over time

**Default SLOs:**
| SLO | Target | Window |
|-----|--------|--------|
| Command Success Rate | 99.5% | 30 days |
| Provider Latency p95 | 95% < 30s | 30 days |
| Auto Mode Success | 99% | 30 days |
| Spec Validation | 99% | 30 days |

## Installation

### Method 1: Import via Grafana UI

1. Open Grafana and navigate to **Dashboards > Import**
2. Upload the JSON file or paste its contents
3. Select your Prometheus datasource
4. Click **Import**

### Method 2: Provisioning

Add to your Grafana provisioning configuration:

```yaml
# /etc/grafana/provisioning/dashboards/specular.yaml
apiVersion: 1

providers:
  - name: 'Specular'
    orgId: 1
    folder: 'Specular'
    folderUid: 'specular'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 30
    options:
      path: /var/lib/grafana/dashboards/specular
```

Copy dashboard files:

```bash
cp deployments/grafana/dashboards/*.json /var/lib/grafana/dashboards/specular/
```

### Method 3: Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: specular-dashboards
  labels:
    grafana_dashboard: "1"
data:
  specular-overview.json: |
    <contents of specular-overview.json>
  specular-slos.json: |
    <contents of specular-slos.json>
```

## Required Metrics

These dashboards expect the following Prometheus metrics from Specular:

### Command Metrics
- `specular_command_executions_total{command, status}` - Counter
- `specular_command_duration_seconds{command}` - Histogram

### Provider Metrics
- `specular_provider_latency_seconds{provider}` - Histogram
- `specular_provider_tokens_total{provider, token_type}` - Counter

### Workflow Metrics
- `specular_auto_workflows_total{status}` - Counter
- `specular_spec_validations_total{status}` - Counter

## Variables

All dashboards include:

| Variable | Type | Description |
|----------|------|-------------|
| `datasource` | datasource | Prometheus datasource selector |

## Customization

### Adjusting SLO Targets

Edit the dashboard JSON to modify SLO targets. Key expressions to update:

```promql
# Command Success SLO (currently 99.5%)
# Change 0.005 (error budget) and 0.995 (threshold) as needed
(1 - sum(rate(specular_command_executions_total{status="success"}[30d])) / sum(rate(specular_command_executions_total[30d]))) / 0.005

# Provider Latency SLO (currently 95% < 30s)
# Change le="30" for different latency threshold
sum(rate(specular_provider_latency_seconds_bucket{le="30"}[30d])) / sum(rate(specular_provider_latency_seconds_count[30d]))
```

### Burn Rate Alerting

The burn rate charts include thresholds:
- **1x**: Normal consumption rate
- **14.4x**: Critical burn rate (consumes 100% budget in 2 hours)

Configure alerts in Grafana when burn rate exceeds these thresholds.

## Troubleshooting

### No Data

1. Verify Prometheus is scraping Specular metrics:
   ```bash
   curl http://localhost:9090/api/v1/query?query=specular_command_executions_total
   ```

2. Check the datasource variable is correctly configured

3. Ensure the time range includes data (try "Last 1 hour")

### Missing Panels

Some panels require specific metrics that may not be present if certain Specular features aren't used:
- Auto mode panels require `specular_auto_workflows_total`
- Spec validation panels require `specular_spec_validations_total`

## Integration with Alerting

### Grafana Alerting

Create alert rules based on dashboard queries:

```yaml
# Example alert rule for high burn rate
groups:
  - name: specular-slos
    rules:
      - alert: HighCommandBurnRate
        expr: |
          (1 - (sum(rate(specular_command_executions_total{status="success"}[1h]))
          / sum(rate(specular_command_executions_total[1h])))) / 0.005 > 14.4
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Command success SLO burn rate is critical"
          description: "Burn rate {{ $value | humanize }} exceeds 14.4x threshold"
```

### PagerDuty/Opsgenie Integration

Configure Specular's alerting integration to receive SLO alerts:

```yaml
# specular.yaml
observability:
  alerting:
    enabled: true
    pagerduty:
      routing_key: "your-routing-key"
    slack:
      webhook_url: "https://hooks.slack.com/services/..."
```
