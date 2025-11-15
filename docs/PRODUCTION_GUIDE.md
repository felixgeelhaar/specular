# Specular Production Deployment Guide

This guide covers production deployment, operation, and maintenance of Specular in enterprise environments.

## Table of Contents

- [Deployment Patterns](#deployment-patterns)
- [Configuration Management](#configuration-management)
- [Security Hardening](#security-hardening)
- [Performance Tuning](#performance-tuning)
- [Monitoring & Observability](#monitoring--observability)
- [Disaster Recovery](#disaster-recovery)
- [Troubleshooting](#troubleshooting)
- [CI/CD Integration](#cicd-integration)
- [Scaling Considerations](#scaling-considerations)

---

## Deployment Patterns

### Single Binary Deployment

**Best for:** Small teams, simple workflows, local development

```bash
# Install via package manager (recommended)
brew install felixgeelhaar/tap/specular

# Or download binary directly
wget https://github.com/felixgeelhaar/specular/releases/latest/download/specular_linux_amd64.tar.gz
tar -xzf specular_linux_amd64.tar.gz
sudo mv specular /usr/local/bin/

# Verify installation
specular version
specular doctor --format json
```

**Configuration:**
```bash
# System-wide configuration
sudo mkdir -p /etc/specular
sudo cp .specular/policy.yaml /etc/specular/policy.yaml

# Per-user configuration
mkdir -p ~/.specular
cp .specular/providers.yaml ~/.specular/
```

### Containerized Deployment

**Best for:** CI/CD pipelines, isolated environments, reproducible builds

**Dockerfile:**
```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY . .
RUN go build -o specular -ldflags="-s -w" ./cmd/specular

FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    docker-cli \
    && addgroup -g 1000 specular \
    && adduser -D -u 1000 -G specular specular

# Copy binary
COPY --from=builder /build/specular /usr/local/bin/specular

# Create workspace
RUN mkdir -p /workspace/.specular && chown -R specular:specular /workspace
WORKDIR /workspace

# Run as non-root
USER specular

ENTRYPOINT ["specular"]
CMD ["--help"]
```

**Docker Compose for Development:**
```yaml
version: '3.8'

services:
  specular:
    build: .
    volumes:
      - ./workspace:/workspace
      - ./config:/home/specular/.specular:ro
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - SPECULAR_NO_TELEMETRY=false
    networks:
      - specular-net

networks:
  specular-net:
    driver: bridge
```

### Kubernetes Deployment

**Best for:** Large-scale deployments, high availability, enterprise environments

**Deployment manifest:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: specular
  namespace: specular-system
  labels:
    app: specular
spec:
  replicas: 3
  selector:
    matchLabels:
      app: specular
  template:
    metadata:
      labels:
        app: specular
    spec:
      serviceAccountName: specular
      securityContext:
        fsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      containers:
      - name: specular
        image: ghcr.io/felixgeelhaar/specular:v1.4.0
        imagePullPolicy: IfNotPresent
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        env:
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: specular-secrets
              key: anthropic-api-key
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: specular-secrets
              key: openai-api-key
        volumeMounts:
        - name: config
          mountPath: /home/specular/.specular
          readOnly: true
        - name: workspace
          mountPath: /workspace
        - name: cache
          mountPath: /workspace/.specular/cache
      volumes:
      - name: config
        configMap:
          name: specular-config
      - name: workspace
        emptyDir: {}
      - name: cache
        persistentVolumeClaim:
          claimName: specular-cache
```

**ConfigMap:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: specular-config
  namespace: specular-system
data:
  policy.yaml: |
    execution:
      allow_local: false
      docker:
        required: true
        image_allowlist:
          - "docker.io/library/golang:1.22"
          - "docker.io/library/node:22"
          - "docker.io/library/python:3.12"
        cpu_limit: "2"
        mem_limit: "2g"
        network: "none"
    security:
      secrets_scan: true
      dep_scan: true
    tests:
      require_pass: true
      min_coverage: 0.70
```

**Secret management:**
```bash
# Create secrets
kubectl create secret generic specular-secrets \
  --from-literal=anthropic-api-key="${ANTHROPIC_API_KEY}" \
  --from-literal=openai-api-key="${OPENAI_API_KEY}" \
  --namespace=specular-system

# Or use external secrets operator
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: specular-secrets
  namespace: specular-system
spec:
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: specular-secrets
  data:
  - secretKey: anthropic-api-key
    remoteRef:
      key: /specular/production/anthropic-api-key
  - secretKey: openai-api-key
    remoteRef:
      key: /specular/production/openai-api-key
```

---

## Configuration Management

### Configuration Hierarchy

Specular uses a three-tier configuration system:

1. **Built-in defaults** - Compiled into binary
2. **System-wide config** - `/etc/specular/` (Linux) or `/usr/local/etc/specular/` (macOS)
3. **User config** - `~/.specular/`
4. **Project config** - `.specular/` in project directory

**Load order:** Project → User → System → Built-in (first found wins)

### Production Configuration Structure

```
/etc/specular/                    # System-wide (Linux)
├── policy.yaml                   # Global policy enforcement
├── providers.yaml                # Provider defaults
└── router.yaml                   # Model routing rules

~/.specular/                      # Per-user overrides
├── providers.yaml                # User-specific API keys
├── auto.profiles.yaml            # Custom profiles
└── logs/                         # Trace logs

.specular/                        # Per-project (checked into git)
├── policy.yaml                   # Project-specific policies
├── spec.yaml                     # Product specification
├── spec.lock.json                # Locked specification
├── router.yaml                   # Project routing rules
├── runs/                         # Execution manifests
├── checkpoints/                  # Auto mode checkpoints
└── cache/                        # Docker image cache
```

### Environment Variables

**Required for production:**
```bash
# AI Provider API Keys
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."
export GEMINI_API_KEY="..."

# Telemetry (set to "true" to disable)
export SPECULAR_NO_TELEMETRY="false"

# Logging
export SPECULAR_LOG_LEVEL="info"
export SPECULAR_LOG_FORMAT="json"

# Performance
export SPECULAR_CACHE_DIR="/var/cache/specular"
export SPECULAR_CACHE_MAX_AGE="168h"
```

### Configuration Validation

```bash
# Validate configuration before deployment
specular doctor --format json | jq .

# Test provider connectivity
specular provider health

# Verify policy syntax
specular policy validate --policy .specular/policy.yaml

# Check router configuration
specular router validate --router .specular/router.yaml
```

---

## Security Hardening

### Secret Management

**DO NOT:**
- ❌ Commit API keys to version control
- ❌ Store secrets in environment files checked into git
- ❌ Use the same API keys across all environments

**DO:**
- ✅ Use environment-specific secrets
- ✅ Rotate API keys regularly (90 days)
- ✅ Use secret management tools (Vault, AWS Secrets Manager, Azure Key Vault)
- ✅ Enable automatic secret scanning

**Implementation:**
```bash
# Enable automatic secret scanning (already built-in)
specular auto "Build REST API" \
  --security-scan \
  --fail-on-secrets

# Secrets are automatically redacted from logs
# Example log output:
# API Key: ***REDACTED*** (pattern: AWS_ACCESS_KEY_ID)
```

### Docker Security

**Restrict allowed Docker images:**
```yaml
# .specular/policy.yaml
execution:
  docker:
    required: true
    image_allowlist:
      - "docker.io/library/golang:1.22*"
      - "docker.io/library/node:22*"
      - "docker.io/library/python:3.12*"
      # Block: latest, alpine-edge, untrusted registries
    network: "none"
    cap_drop_all: true
    read_only: true
    cpu_limit: "2"
    mem_limit: "2g"
    pids_limit: 256
```

### Network Isolation

**Disable network access in Docker containers:**
```yaml
execution:
  docker:
    network: "none"  # No network access
    # Or use bridge with specific DNS
    network: "bridge"
    dns_servers:
      - "8.8.8.8"
      - "8.8.4.4"
```

### Audit Logging

**Enable comprehensive audit trails:**
```bash
# Enable trace logging for all workflows
specular auto "Build feature" \
  --trace \
  --attest \
  --save-patches

# Audit logs stored at:
# ~/.specular/logs/trace_auto-1234567890.json
# ~/.specular/attestations/auto-1234567890.json
```

**Audit log retention:**
```bash
# Rotate logs older than 90 days
find ~/.specular/logs -name "trace_*.json" -mtime +90 -delete

# Archive attestations before deletion
find ~/.specular/attestations -name "*.json" -mtime +365 \
  -exec tar -czf attestations-archive-$(date +%Y%m%d).tar.gz {} + \
  -delete
```

### File System Permissions

```bash
# Restrict config directory permissions
chmod 700 ~/.specular
chmod 600 ~/.specular/providers.yaml  # Contains API keys

# System-wide config (read-only for users)
sudo chown root:root /etc/specular/policy.yaml
sudo chmod 644 /etc/specular/policy.yaml
```

---

## Performance Tuning

### Docker Image Caching

**Enable persistent caching:**
```bash
# Local development
specular build run \
  --plan plan.json \
  --enable-cache \
  --cache-dir .specular/cache \
  --cache-max-age 168h  # 7 days

# CI/CD with cache export/import
# In CI pipeline:
specular build run \
  --enable-cache \
  --cache-dir /tmp/specular-cache

# Export cache for next run
tar -czf docker-cache.tar.gz .specular/cache

# Import cache in next run
tar -xzf docker-cache.tar.gz
```

**Cache statistics:**
```bash
# View cache performance
specular cache stats

# Output:
# Docker Image Cache Statistics
# =============================
# Cache Directory: .specular/cache
# Total Images: 15
# Total Size: 2.3 GB
# Cache Hit Rate: 87.5% (14/16 pulls)
# Average Pull Time (miss): 45.2s
# Average Restore Time (hit): 2.1s
```

### Profile Optimization

**Production profile:**
```yaml
# ~/.specular/auto.profiles.yaml
production:
  safety:
    max_steps: 8
    timeout: "15m"
    max_cost_usd: 2.0
    max_cost_per_task: 0.25
    max_retries: 2
  execution:
    save_patches: true
    enable_cache: true
    cache_max_age: "336h"  # 14 days
  output:
    json: true
    tui: false
    trace: true
  hooks:
    enabled: true
    slack_webhook: "${SLACK_WEBHOOK_URL}"
```

**Usage:**
```bash
specular auto "Build feature" --profile production
```

### Cost Optimization

**Set budget constraints:**
```bash
# Limit total workflow cost
specular auto "Build feature" \
  --max-cost 1.00 \
  --max-cost-per-task 0.10

# Use cheaper models for simple tasks
specular auto "Build feature" \
  --router .specular/router-cost-optimized.yaml
```

**Cost-optimized router:**
```yaml
# .specular/router-cost-optimized.yaml
task_type_routing:
  simple:
    - provider: anthropic
      model: claude-3-haiku-20240307
      priority: 1
  complex:
    - provider: anthropic
      model: claude-3-5-sonnet-20241022
      priority: 1

budget:
  max_total_cost: 5.0
  max_per_task_cost: 0.50
```

---

## Monitoring & Observability

### Metrics Collection

**Prometheus metrics endpoint:**
```go
// Enable metrics collection (already built-in)
// Metrics are automatically collected when using observability framework
```

**Key metrics to monitor:**
```promql
# Command execution rate
rate(specular_command_executions_total[5m])

# Provider API latency (p95)
histogram_quantile(0.95, rate(specular_provider_latency_seconds_bucket[5m]))

# Error rate by provider
rate(specular_provider_errors_total[5m])

# Cost accumulation
rate(specular_provider_cost_total[1h])

# Docker cache hit rate
rate(specular_docker_cache_hits_total[5m]) /
  (rate(specular_docker_cache_hits_total[5m]) + rate(specular_docker_cache_misses_total[5m]))
```

### Distributed Tracing

**OpenTelemetry configuration:**
```bash
# Enable tracing export to Jaeger
export OTEL_EXPORTER_OTLP_ENDPOINT="http://jaeger:4318"
export OTEL_SERVICE_NAME="specular"
export OTEL_SERVICE_VERSION="1.4.0"

# Run with tracing enabled
specular auto "Build feature" --trace
```

**Trace spans to monitor:**
- `auto.workflow` - Full workflow execution
- `provider.generate` - AI provider calls
- `docker.pull_image` - Image pull operations
- `policy.check` - Policy enforcement
- `drift.detect` - Drift detection

### Structured Logging

**Log aggregation setup:**
```bash
# Configure JSON logging for machine parsing
export SPECULAR_LOG_FORMAT="json"
export SPECULAR_LOG_LEVEL="info"

# Example structured log output:
{
  "level": "info",
  "time": "2025-01-15T10:30:45Z",
  "msg": "workflow started",
  "workflow_id": "auto-1234567890",
  "goal": "Build REST API",
  "profile": "production",
  "max_cost": 2.0,
  "max_steps": 8
}
```

**Log aggregation with ELK Stack:**
```yaml
# filebeat.yml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /home/*/.specular/logs/trace_*.json
  json.keys_under_root: true
  json.add_error_key: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "specular-logs-%{+yyyy.MM.dd}"
```

### Alerting Rules

**Prometheus alerting:**
```yaml
groups:
- name: specular_alerts
  rules:
  # High error rate
  - alert: SpecularHighErrorRate
    expr: rate(specular_command_errors_total[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate detected"
      description: "Error rate is {{ $value }} errors/sec"

  # High provider latency
  - alert: SpecularSlowProvider
    expr: histogram_quantile(0.95, rate(specular_provider_latency_seconds_bucket[5m])) > 30
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "AI provider is slow"
      description: "P95 latency is {{ $value }} seconds"

  # Cost budget exceeded
  - alert: SpecularCostExceeded
    expr: increase(specular_provider_cost_total[1h]) > 10
    labels:
      severity: critical
    annotations:
      summary: "Cost budget exceeded"
      description: "Hourly cost is ${{ $value }}"

  # Low cache hit rate
  - alert: SpecularLowCacheHitRate
    expr: |
      rate(specular_docker_cache_hits_total[5m]) /
      (rate(specular_docker_cache_hits_total[5m]) + rate(specular_docker_cache_misses_total[5m])) < 0.5
    for: 30m
    labels:
      severity: info
    annotations:
      summary: "Low Docker cache hit rate"
      description: "Cache hit rate is {{ $value | humanizePercentage }}"
```

### Health Checks

**Kubernetes liveness/readiness probes:**
```yaml
livenessProbe:
  exec:
    command:
    - specular
    - doctor
    - --format
    - json
  initialDelaySeconds: 30
  periodSeconds: 60
  timeoutSeconds: 10
  failureThreshold: 3

readinessProbe:
  exec:
    command:
    - specular
    - provider
    - health
  initialDelaySeconds: 10
  periodSeconds: 30
  timeoutSeconds: 5
  failureThreshold: 2
```

---

## Disaster Recovery

### Backup Strategy

**What to backup:**
1. **Configuration files:**
   - `.specular/policy.yaml`
   - `.specular/router.yaml`
   - `.specular/spec.lock.json`
   - `~/.specular/providers.yaml` (encrypted)

2. **Execution history:**
   - `~/.specular/logs/` (trace logs)
   - `~/.specular/attestations/` (cryptographic attestations)
   - `.specular/runs/` (execution manifests)

3. **Checkpoints:**
   - `.specular/checkpoints/` (auto mode state)

**Backup script:**
```bash
#!/bin/bash
# backup-specular.sh

BACKUP_DIR="/backup/specular/$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"

# Backup system config
sudo tar -czf "$BACKUP_DIR/system-config.tar.gz" /etc/specular/

# Backup user config (encrypt API keys)
tar -czf - ~/.specular/ | \
  gpg --encrypt --recipient ops@company.com > \
  "$BACKUP_DIR/user-config.tar.gz.gpg"

# Backup project config (version controlled, no backup needed)
# .specular/ should already be in git

# Backup execution history (last 90 days)
find ~/.specular/logs -name "*.json" -mtime -90 | \
  tar -czf "$BACKUP_DIR/logs.tar.gz" -T -

find ~/.specular/attestations -name "*.json" -mtime -90 | \
  tar -czf "$BACKUP_DIR/attestations.tar.gz" -T -

# Verify backups
tar -tzf "$BACKUP_DIR/logs.tar.gz" > /dev/null
echo "Backup completed: $BACKUP_DIR"
```

### Recovery Procedures

**Restore from backup:**
```bash
#!/bin/bash
# restore-specular.sh

BACKUP_DIR="/backup/specular/20250115"

# Restore system config
sudo tar -xzf "$BACKUP_DIR/system-config.tar.gz" -C /

# Restore user config
gpg --decrypt "$BACKUP_DIR/user-config.tar.gz.gpg" | \
  tar -xzf - -C ~/

# Restore logs and attestations
tar -xzf "$BACKUP_DIR/logs.tar.gz" -C ~/.specular/logs/
tar -xzf "$BACKUP_DIR/attestations.tar.gz" -C ~/.specular/attestations/

# Verify restoration
specular doctor --format json
specular provider health
```

**Checkpoint recovery:**
```bash
# List available checkpoints
specular checkpoint list

# Resume from checkpoint
specular auto --resume auto-1234567890

# If checkpoint is corrupted, restore from backup
tar -xzf "$BACKUP_DIR/checkpoints.tar.gz" -C .specular/
```

---

## Troubleshooting

### Common Issues and Solutions

#### Issue: API Key Authentication Failures

**Symptoms:**
```
Error: failed to generate spec
Code: PROVIDER_ERROR
Message: authentication failed (401 Unauthorized)
```

**Diagnosis:**
```bash
# Check API key is set
env | grep -E "ANTHROPIC_API_KEY|OPENAI_API_KEY"

# Test provider connectivity
specular provider health

# Verify API key format
echo $ANTHROPIC_API_KEY | cut -c1-10  # Should start with "sk-ant-"
```

**Solutions:**
```bash
# Re-export API keys
export ANTHROPIC_API_KEY="sk-ant-..."

# Or update providers.yaml
specular init

# Test again
specular generate "Hello world" --provider anthropic
```

#### Issue: Docker Image Pull Failures

**Symptoms:**
```
Error: failed to pull Docker image
Code: DOCKER_ERROR
Message: image not found or pull timeout
```

**Diagnosis:**
```bash
# Check Docker daemon
docker info

# Test image pull manually
docker pull golang:1.22

# Check policy allowlist
cat .specular/policy.yaml | grep -A 10 image_allowlist
```

**Solutions:**
```bash
# Add image to allowlist
# Edit .specular/policy.yaml:
execution:
  docker:
    image_allowlist:
      - "docker.io/library/golang:1.22"

# Prewarm cache
specular cache prewarm --images golang:1.22,node:22

# Increase pull timeout
specular build run --timeout 30m
```

#### Issue: High Provider Costs

**Symptoms:**
```
Warning: workflow cost $15.50 exceeds budget $5.00
Code: COST_EXCEEDED
```

**Diagnosis:**
```bash
# Check cost breakdown
specular explain --workflow auto-1234567890

# Review routing decisions
cat ~/.specular/logs/trace_auto-1234567890.json | \
  jq '.events[] | select(.type=="provider_call") | {model, cost, tokens}'
```

**Solutions:**
```bash
# Use cost-optimized profile
specular auto "Build feature" \
  --profile ci \  # Lower cost limits
  --max-cost 2.0

# Use cheaper models
specular auto "Build feature" \
  --router .specular/router-cost-optimized.yaml

# Limit scope
specular auto "Build feature" \
  --scope feature-123 \
  --no-dependencies
```

#### Issue: Slow Performance

**Symptoms:**
- Workflow execution takes > 10 minutes
- High Docker image pull times
- Slow AI provider responses

**Diagnosis:**
```bash
# Check cache hit rate
specular cache stats

# Profile workflow
specular auto "Build feature" --trace
cat ~/.specular/logs/trace_*.json | \
  jq '.events[] | select(.type=="step_completed") | {step, duration_ms}'

# Check network latency to providers
time curl -I https://api.anthropic.com
```

**Solutions:**
```bash
# Enable aggressive caching
specular build run \
  --enable-cache \
  --cache-max-age 336h  # 14 days

# Prewarm Docker images
specular cache prewarm \
  --images golang:1.22,node:22,python:3.12 \
  --concurrency 4

# Use faster profile
specular auto "Build feature" --profile ci
```

### Debug Mode

**Enable verbose logging:**
```bash
# Full debug output
specular auto "Build feature" \
  --verbose \
  --trace \
  --json > debug.json 2> debug.log

# Analyze execution
cat debug.json | jq .
```

**Collect diagnostic bundle:**
```bash
# Create support bundle
tar -czf specular-debug-$(date +%Y%m%d).tar.gz \
  ~/.specular/logs/ \
  .specular/policy.yaml \
  .specular/router.yaml \
  debug.json \
  debug.log

# Share with support (redact API keys first!)
```

---

## CI/CD Integration

See [GitHub Actions Integration](../.github/actions/specular/README.md) for comprehensive CI/CD examples.

**Quick start:**
```yaml
# .github/workflows/specular.yml
name: Specular CI

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run Specular
        uses: ./.github/actions/specular
        with:
          command: eval
          enable-cache: 'true'
          anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

---

## Scaling Considerations

### Horizontal Scaling

**GitHub Actions matrix builds:**
```yaml
jobs:
  validate:
    strategy:
      matrix:
        feature: [user-auth, payments, notifications]
    steps:
      - uses: ./.github/actions/specular
        with:
          command: build
          additional-args: '--scope ${{ matrix.feature }}'
```

### Resource Limits

**Per-workflow limits:**
```yaml
# auto.profiles.yaml
high-concurrency:
  safety:
    max_steps: 20
    timeout: "45m"
    max_cost_usd: 10.0
  execution:
    concurrent_tasks: 4  # Run 4 tasks in parallel
```

---

## Production Checklist

Before deploying to production:

- [ ] API keys stored in secret manager (not environment files)
- [ ] Policy file configured with strict enforcement
- [ ] Docker image allowlist restricted to approved images
- [ ] Network isolation enabled (`network: "none"`)
- [ ] Audit logging enabled (`--trace --attest`)
- [ ] Monitoring and alerting configured (Prometheus + Grafana)
- [ ] Backup strategy implemented and tested
- [ ] Disaster recovery runbook documented
- [ ] Cost budgets configured (`max_cost`, `max_cost_per_task`)
- [ ] Docker cache enabled for performance
- [ ] Health checks configured (Kubernetes probes)
- [ ] Secrets scanning enabled (`--security-scan`)
- [ ] Log rotation configured (90-day retention)
- [ ] Team training completed on operations procedures

---

## Support and Resources

- **Documentation**: https://github.com/felixgeelhaar/specular/docs
- **Issues**: https://github.com/felixgeelhaar/specular/issues
- **Observability Guide**: [ADR 0009](./adr/0009-observability-monitoring-strategy.md)
- **Security Guide**: [Advanced Security](../README.md#advanced-security)
- **GitHub Action**: [CI/CD Integration](../.github/actions/specular/README.md)

---

**Last Updated:** 2025-01-15
**Version:** 1.0
**Owner:** Specular Core Team
