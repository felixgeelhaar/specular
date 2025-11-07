# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take the security of Specular seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### Please DO NOT:

- Open a public GitHub issue for security vulnerabilities
- Discuss the vulnerability in public forums or social media
- Exploit the vulnerability beyond what is necessary to demonstrate it

### Please DO:

1. **Email** your findings to [INSERT SECURITY EMAIL]
2. **Provide** detailed information including:
   - Description of the vulnerability
   - Steps to reproduce the issue
   - Potential impact
   - Suggested fix (if any)
3. **Allow** us reasonable time to address the issue before any public disclosure

### What to Expect:

- **Acknowledgment**: Within 48 hours of your report
- **Initial Assessment**: Within 7 days
- **Status Updates**: Regular updates on our progress
- **Resolution Timeline**: We aim to resolve critical issues within 30 days
- **Credit**: You will be credited in the security advisory (if desired)

## Security Best Practices

When using Specular:

### Docker Security

- **Use official images** from verified registries
- **Implement image allowlisting** in policy.yaml
- **Set resource limits** to prevent DoS
- **Use read-only filesystems** where possible
- **Run containers as non-root** users

Example secure policy:

```yaml
execution:
  allow_local: false
  docker:
    required: true
    image_allowlist:
      - golang:1.22
      - alpine:3.19
    resource_limits:
      cpu: "2"
      memory: "2g"
    network: "none"
    read_only: true
    user: "nobody"
```

### API Key Management

- **Never commit** API keys to version control
- **Use environment variables** for sensitive data
- **Rotate keys regularly**
- **Use least-privilege** API keys
- **Monitor key usage** for anomalies

### Input Validation

- **Validate all inputs** from PRDs and specifications
- **Sanitize file paths** to prevent directory traversal
- **Limit input sizes** to prevent resource exhaustion
- **Use schema validation** for YAML/JSON inputs

### Code Generation Security

- **Review generated code** before deployment
- **Run static analysis** on generated code
- **Test in isolated environments** first
- **Use policy enforcement** to prevent unsafe operations
- **Implement code signing** for releases

## Known Security Considerations

### Docker Execution

Specular executes code in Docker containers. Ensure:

- Docker daemon is properly secured
- Container escape vulnerabilities are patched
- Network isolation is configured
- Resource limits are enforced

### LLM Provider Security

When using LLM providers:

- API keys are transmitted over HTTPS
- Responses are not cached with sensitive data
- Rate limiting is implemented
- Provider SLAs include security guarantees

### Dependency Security

We actively monitor dependencies for vulnerabilities:

- **Automated scanning** with Dependabot
- **Regular updates** of dependencies
- **Security advisories** published for known issues
- **Pinned versions** in go.mod for reproducibility

## Security Advisories

Security advisories will be published:

- On the [GitHub Security tab](https://github.com/felixgeelhaar/specular/security)
- In release notes for patched versions
- Via GitHub Security Advisories

## Disclosure Policy

- **Coordinated disclosure**: We prefer coordinated disclosure with security researchers
- **Public disclosure timeline**: 90 days after initial report (flexible based on severity)
- **CVE assignment**: We will request CVEs for confirmed vulnerabilities
- **Credit**: Security researchers will be credited unless they prefer anonymity

## Security Updates

To stay informed about security updates:

- Watch the repository for security advisories
- Subscribe to release notifications
- Check the [Releases page](https://github.com/felixgeelhaar/specular/releases)
- Follow security best practices in documentation

## Security Checklist for Contributors

Before submitting code:

- [ ] Input validation is implemented
- [ ] No hardcoded secrets or credentials
- [ ] Dependencies are up to date
- [ ] Security implications are considered
- [ ] Error messages don't leak sensitive information
- [ ] File operations are safe from path traversal
- [ ] Resource limits prevent DoS
- [ ] Tests include security scenarios

## Additional Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Best Practices](https://go.dev/security/best-practices)
- [Docker Security](https://docs.docker.com/engine/security/)
- [CWE Top 25](https://cwe.mitre.org/top25/)

Thank you for helping keep Specular and its users safe!
