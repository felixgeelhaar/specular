# Kubernetes Deployment

This directory contains Kubernetes manifests for deploying Specular with zero-downtime capabilities.

## Overview

The deployment configuration implements:

- **Zero-Downtime Deployments** - Rolling updates with maxUnavailable=0
- **Kubernetes Health Probes** - Startup, liveness, and readiness probes
- **Graceful Shutdown** - Proper connection draining before termination
- **Production-Ready Settings** - Resource limits, security contexts, and best practices

## Files

- `deployment.yaml` - Main Deployment with health probe configuration
- `service.yaml` - ClusterIP and Headless services
- `README.md` - This file

## Quick Start

### Deploy to Kubernetes

```bash
# Apply all manifests
kubectl apply -f deployments/kubernetes/

# Check deployment status
kubectl rollout status deployment/specular

# View pods
kubectl get pods -l app=specular

# Check pod health
kubectl describe pod -l app=specular
```

### Access the Service

```bash
# Port-forward to test locally
kubectl port-forward svc/specular 8080:80

# Test health endpoints
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
curl http://localhost:8080/health/startup
```

## Health Probes

The deployment configures three types of Kubernetes health probes:

### Startup Probe (`/health/startup`)

**Purpose**: Determines if the application has finished initialization.

**Configuration**:
- Initial delay: 0 seconds (checks immediately)
- Period: 5 seconds
- Timeout: 3 seconds
- Failure threshold: 12 (60 seconds max startup time)

**Behavior**:
- Kubernetes won't check liveness/readiness until startup succeeds
- Allows slow-starting applications more time to initialize
- Returns 200 OK when application is initialized
- Returns 503 Service Unavailable while starting

**Example Response**:
```json
{
  "status": "healthy",
  "version": "2.0.0",
  "uptime": "5s",
  "checks": {},
  "timestamp": "2025-01-20T10:30:00Z"
}
```

### Liveness Probe (`/health/live`)

**Purpose**: Determines if the application is alive and responsive.

**Configuration**:
- Initial delay: 10 seconds
- Period: 10 seconds
- Timeout: 3 seconds
- Failure threshold: 3 (restart after 30 seconds of failures)

**Behavior**:
- If this check fails, Kubernetes will restart the container
- Always returns 200 OK (even during shutdown)
- Status is "healthy" during normal operation
- Status is "degraded" during shutdown (but still alive)

**Example Response (Normal)**:
```json
{
  "status": "healthy",
  "version": "2.0.0",
  "uptime": "5m30s",
  "checks": {},
  "timestamp": "2025-01-20T10:30:00Z"
}
```

**Example Response (Shutting Down)**:
```json
{
  "status": "degraded",
  "version": "2.0.0",
  "uptime": "5m30s",
  "checks": {},
  "timestamp": "2025-01-20T10:30:00Z"
}
```

### Readiness Probe (`/health/ready`)

**Purpose**: Determines if the application is ready to accept traffic.

**Configuration**:
- Initial delay: 5 seconds
- Period: 5 seconds
- Timeout: 3 seconds
- Failure threshold: 2 (remove from service after 10 seconds of failures)

**Behavior**:
- If this check fails, Kubernetes removes the pod from service endpoints
- Returns 200 OK when ready to serve requests
- Returns 503 Service Unavailable when not ready (shutting down or dependencies unhealthy)
- Checks all registered health checkers (Docker, Git, etc.)

**Example Response (Ready)**:
```json
{
  "status": "healthy",
  "version": "2.0.0",
  "uptime": "5m30s",
  "checks": {
    "docker": {
      "status": "healthy",
      "message": "Docker is running",
      "timestamp": "2025-01-20T10:30:00Z"
    },
    "git": {
      "status": "healthy",
      "message": "Git is available",
      "timestamp": "2025-01-20T10:30:00Z"
    }
  },
  "timestamp": "2025-01-20T10:30:00Z"
}
```

**Example Response (Not Ready)**:
```json
{
  "status": "unhealthy",
  "version": "2.0.0",
  "uptime": "5m30s",
  "checks": {},
  "timestamp": "2025-01-20T10:30:00Z"
}
```

## Zero-Downtime Deployments

The deployment is configured for zero-downtime rolling updates:

### Rolling Update Strategy

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1          # Create 1 extra pod during update
    maxUnavailable: 0    # Never reduce capacity below desired replicas
```

**How it Works**:
1. Kubernetes creates 1 new pod with the updated version
2. Waits for startup probe to succeed
3. Waits for readiness probe to succeed
4. Adds new pod to service endpoints
5. Marks old pod for termination (sends SIGTERM)
6. Old pod's readiness probe fails (removed from service)
7. Old pod drains connections for 15 seconds (preStop hook)
8. Old pod shuts down gracefully (up to 60s total)
9. Repeats for remaining pods

### Graceful Shutdown Flow

When a pod receives SIGTERM (during update or scale-down):

1. **Readiness Probe Fails** - Pod is immediately removed from service endpoints
2. **PreStop Hook Executes** - Sleeps for 15 seconds to allow in-flight requests to complete
3. **Application Shutdown** - Specular receives SIGTERM and initiates graceful shutdown:
   - Marks server as shutting down
   - Disables HTTP keep-alives (stops accepting new requests)
   - Waits for existing connections to drain (up to 30s)
   - Forces closure of remaining connections
4. **Termination** - If not stopped within 60s, Kubernetes forcefully kills the pod

### Configuration Tuning

**Shutdown Timeout** (default: 30s):
```bash
specular serve --shutdown-timeout=30s
```

**Termination Grace Period** (default: 60s):
```yaml
terminationGracePeriodSeconds: 60  # Should be > shutdown-timeout + preStop sleep
```

**PreStop Sleep** (default: 15s):
```yaml
lifecycle:
  preStop:
    exec:
      command: ["sh", "-c", "sleep 15"]
```

**Formula**: `terminationGracePeriodSeconds > shutdown-timeout + preStop sleep + buffer`

Example: `60s > 30s + 15s + 15s` ✅

## Resource Management

The deployment includes resource requests and limits:

```yaml
resources:
  requests:
    cpu: 100m      # Guaranteed minimum
    memory: 128Mi
  limits:
    cpu: 500m      # Maximum allowed
    memory: 512Mi
```

**Requests**: Guaranteed resources for scheduling decisions

**Limits**: Maximum resources the container can use

Adjust based on your workload requirements.

## Security

The deployment includes security best practices:

### Pod Security Context

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
    - ALL
  seccompProfile:
    type: RuntimeDefault
```

**Security Features**:
- Non-root user execution
- Read-only root filesystem
- No privilege escalation
- Minimal capabilities
- Seccomp profile for syscall filtering

### Network Policies (Optional)

Consider adding NetworkPolicy resources to restrict traffic:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: specular-network-policy
spec:
  podSelector:
    matchLabels:
      app: specular
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector: {}  # Allow from same namespace
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector: {}
```

## Scaling

### Manual Scaling

```bash
# Scale to 5 replicas
kubectl scale deployment specular --replicas=5

# Check scaling progress
kubectl rollout status deployment/specular
```

### Horizontal Pod Autoscaling (HPA)

Create an HPA resource for automatic scaling:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: specular-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: specular
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

```bash
# Apply HPA
kubectl apply -f hpa.yaml

# Check HPA status
kubectl get hpa specular-hpa
```

## Monitoring

### Check Pod Status

```bash
# View pod events
kubectl describe pod -l app=specular

# Check probe status
kubectl get events --field-selector involvedObject.name=<pod-name>

# View logs
kubectl logs -l app=specular --tail=100 -f
```

### Health Check Status

```bash
# Port-forward to a pod
kubectl port-forward <pod-name> 8080:8080

# Check all health endpoints
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
curl http://localhost:8080/health/startup
```

### Readiness Status

```bash
# Check which pods are ready
kubectl get pods -l app=specular -o wide

# Watch readiness changes
kubectl get pods -l app=specular -w
```

## Troubleshooting

### Pod Stuck in "Not Ready" State

**Check readiness probe**:
```bash
kubectl describe pod <pod-name> | grep -A 10 "Readiness:"
```

**Common causes**:
- Application not initialized (check startup probe)
- Dependencies unhealthy (check health checker logs)
- Application shutting down (check for SIGTERM)

**Solution**:
```bash
# Check application logs
kubectl logs <pod-name>

# Check health endpoint directly
kubectl exec <pod-name> -- wget -qO- http://localhost:8080/health/ready
```

### Pod Restarting Frequently

**Check liveness probe**:
```bash
kubectl describe pod <pod-name> | grep -A 10 "Liveness:"
```

**Common causes**:
- Application deadlock or hang
- Resource limits too low (OOMKilled)
- Probe timeout too aggressive

**Solution**:
```bash
# Check restart count and reason
kubectl get pods -l app=specular

# View previous container logs
kubectl logs <pod-name> --previous

# Check resource usage
kubectl top pod <pod-name>
```

### Slow Rolling Updates

**Check probe timing**:
```bash
kubectl rollout status deployment/specular
```

**Common causes**:
- Startup probe taking too long
- Old pods not terminating quickly
- Insufficient resources for new pods

**Solution**:
```bash
# Check pod events during rollout
kubectl get events --sort-by='.lastTimestamp'

# Increase progress deadline
kubectl patch deployment specular -p '{"spec":{"progressDeadlineSeconds":600}}'
```

## Advanced Configuration

### Custom Health Checkers

Add custom health checkers in your application code:

```go
// In cmd/serve.go
pm.AddChecker(health.NewDockerChecker())
pm.AddChecker(health.NewGitChecker())
// Add your custom checkers:
pm.AddChecker(NewDatabaseChecker())
pm.AddChecker(NewCacheChecker())
```

### Environment-Specific Configuration

Use ConfigMaps and Secrets for environment-specific settings:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: specular-config
data:
  SHUTDOWN_TIMEOUT: "30s"
  READ_TIMEOUT: "10s"
---
apiVersion: v1
kind: Secret
metadata:
  name: specular-secrets
type: Opaque
stringData:
  API_KEY: "your-secret-key"
```

Reference in deployment:

```yaml
envFrom:
- configMapRef:
    name: specular-config
- secretRef:
    name: specular-secrets
```

### Multi-Environment Deployments

Use Kustomize for environment-specific overlays:

```
deployments/kubernetes/
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    │   └── kustomization.yaml
    ├── staging/
    │   └── kustomization.yaml
    └── production/
        └── kustomization.yaml
```

Deploy to specific environment:

```bash
kubectl apply -k deployments/kubernetes/overlays/production
```

## Best Practices

1. **Always test updates in staging first** - Validate rolling updates work correctly
2. **Monitor probe metrics** - Track probe success/failure rates
3. **Set appropriate timeouts** - Balance responsiveness vs. false positives
4. **Use resource limits** - Prevent resource exhaustion
5. **Implement custom health checkers** - Verify critical dependencies
6. **Test graceful shutdown** - Ensure no request failures during updates
7. **Review security contexts** - Follow principle of least privilege
8. **Document configuration** - Keep README updated with changes

## References

- [Kubernetes Health Checks](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Rolling Updates](https://kubernetes.io/docs/tutorials/kubernetes-basics/update/update-intro/)
- [Pod Lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)
- [Termination Lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-termination)
