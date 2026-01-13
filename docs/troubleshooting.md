# Specular CLI Troubleshooting Guide

This guide helps you diagnose and resolve common issues with the Specular CLI.

## Quick Diagnostics

Before diving into specific issues, run these diagnostic commands:

```bash
# Check Specular version
specular version

# Check environment and providers
specular debug context

# Verify provider health
specular provider doctor

# List available providers
specular provider list
```

## Common Issues

### Issue: "No providers available"

**Symptoms:**
- Error: `no available providers for model type "fast"`
- Error: `failed to create router: no providers configured`

**Causes:**
- No AI providers are installed or configured
- API keys are not set
- Provider services are not running

**Solutions:**

1. **Check available providers:**
   ```bash
   specular provider list
   ```

2. **Set API keys** (choose based on your provider):
   ```bash
   export OPENAI_API_KEY="sk-..."
   export ANTHROPIC_API_KEY="sk-ant-..."
   export GEMINI_API_KEY="..."
   ```

3. **For local providers (Ollama):**
   ```bash
   # Ensure Ollama is installed and running
   ollama serve

   # Pull a model
   ollama pull llama3.2
   ```

4. **Initialize provider configuration:**
   ```bash
   specular init
   ```

---

### Issue: "Authentication failed"

**Symptoms:**
- Error: `authentication failed: invalid api key`
- Error: `401 Unauthorized`
- Error: `403 Forbidden`

**Causes:**
- Invalid or expired API key
- API key not set in environment
- Rate limiting or quota exceeded

**Solutions:**

1. **Verify API key is set:**
   ```bash
   echo $OPENAI_API_KEY
   echo $ANTHROPIC_API_KEY
   ```

2. **Test the API key directly:**
   ```bash
   specular provider doctor
   ```

3. **Check for typos in the key:**
   - OpenAI keys start with `sk-`
   - Anthropic keys start with `sk-ant-`

4. **For SSO/SAML authentication:**
   ```bash
   # Re-authenticate
   specular auth login

   # Check session status
   specular auth status
   ```

---

### Issue: "Budget exceeded"

**Symptoms:**
- Error: `policy violation: budget exceeded`
- Execution stops mid-workflow

**Causes:**
- Default budget is too low for the task
- Previous runs consumed the budget

**Solutions:**

1. **Increase budget:**
   ```bash
   specular auto "goal" --max-cost 5.0
   ```

2. **Check current spending:**
   ```bash
   specular monitor --json | jq '.budget'
   ```

3. **Use a cheaper model:**
   ```bash
   specular auto "goal" --profile ci  # Uses cheaper models
   ```

4. **Use local models (free):**
   ```bash
   # Configure Ollama for local inference
   specular provider add ollama --local
   ```

---

### Issue: "Session not found" when resuming

**Symptoms:**
- Error: `checkpoint "auto-xxx" not found`
- Cannot resume previous session

**Causes:**
- Session was cleaned up
- Wrong session ID
- Checkpoint directory changed

**Solutions:**

1. **List available sessions:**
   ```bash
   specular monitor list
   ```

2. **Resume most recent session:**
   ```bash
   specular auto resume
   ```

3. **Resume specific session:**
   ```bash
   specular auto resume session-id
   ```

4. **Check checkpoint directory:**
   ```bash
   ls -la .specular/checkpoints/
   ```

---

### Issue: "Timeout" during execution

**Symptoms:**
- Error: `context deadline exceeded`
- Error: `task timed out`
- Long-running operations fail

**Causes:**
- Default timeout too short for complex tasks
- Network latency
- Provider rate limiting

**Solutions:**

1. **Increase timeout:**
   ```bash
   specular auto "goal" --timeout 10m
   ```

2. **Break into smaller tasks:**
   ```bash
   # Instead of one large goal
   specular auto "Build complete system"

   # Use smaller, focused goals
   specular auto "Create database schema"
   specular auto "Implement REST endpoints"
   ```

3. **Use local providers for faster response:**
   ```bash
   specular auto "goal" --prefer-local
   ```

---

### Issue: "Docker not available"

**Symptoms:**
- Error: `docker: command not found`
- Error: `Cannot connect to the Docker daemon`

**Causes:**
- Docker not installed
- Docker daemon not running
- Permission issues

**Solutions:**

1. **Install Docker:**
   - [Docker Desktop](https://www.docker.com/products/docker-desktop/)
   - [Podman](https://podman.io/) as alternative

2. **Start Docker daemon:**
   ```bash
   # macOS/Windows: Start Docker Desktop

   # Linux
   sudo systemctl start docker
   ```

3. **Check Docker access:**
   ```bash
   docker info
   ```

4. **Fix permissions (Linux):**
   ```bash
   sudo usermod -aG docker $USER
   # Log out and back in
   ```

---

### Issue: "Profile not found"

**Symptoms:**
- Error: `failed to load profile "custom": profile not found`

**Causes:**
- Profile name is incorrect
- Custom profile file doesn't exist
- Wrong file location

**Solutions:**

1. **List available profiles:**
   ```bash
   specular auto --list-profiles
   ```

2. **Use built-in profile:**
   ```bash
   specular auto "goal" --profile default
   specular auto "goal" --profile ci
   specular auto "goal" --profile strict
   ```

3. **Create custom profile:**
   ```bash
   # Copy built-in profile as starting point
   specular profile show default > ~/.specular/profiles/custom.yaml

   # Edit the profile
   vim ~/.specular/profiles/custom.yaml
   ```

---

### Issue: "Drift detected"

**Symptoms:**
- Error: `drift detected in specification`
- Verification fails after changes

**Causes:**
- Files were modified after planning
- External changes to tracked files
- Specification hash mismatch

**Solutions:**

1. **Check what changed:**
   ```bash
   specular eval drift
   ```

2. **Accept the changes:**
   ```bash
   specular auto "goal" --force
   ```

3. **Re-plan with current state:**
   ```bash
   specular plan create
   ```

---

### Issue: "Network error" / "Connection refused"

**Symptoms:**
- Error: `connection refused`
- Error: `no route to host`
- Error: `DNS lookup failed`

**Causes:**
- No internet connection
- Firewall blocking requests
- Proxy misconfiguration
- Provider API outage

**Solutions:**

1. **Check internet connectivity:**
   ```bash
   curl -I https://api.openai.com
   ```

2. **Configure proxy if needed:**
   ```bash
   export HTTP_PROXY=http://proxy.example.com:8080
   export HTTPS_PROXY=http://proxy.example.com:8080
   ```

3. **Check provider status:**
   - [OpenAI Status](https://status.openai.com/)
   - [Anthropic Status](https://status.anthropic.com/)

4. **Use local fallback:**
   ```bash
   specular auto "goal" --prefer-local
   ```

---

## Log Files

### Log Locations

```
~/.specular/logs/
├── latest.log          # Most recent execution
├── specular-YYYY-MM-DD.log  # Daily logs
└── sessions/
    └── session-id.log  # Per-session logs
```

### Enable Debug Logging

```bash
export SPECULAR_LOG_LEVEL=debug
specular auto "goal" --verbose
```

### View Logs

```bash
# View latest log
cat ~/.specular/logs/latest.log

# Follow logs in real-time
tail -f ~/.specular/logs/latest.log

# Search for errors
grep -i error ~/.specular/logs/latest.log
```

---

## Getting Help

### Built-in Help

```bash
# General help
specular --help

# Command-specific help
specular auto --help
specular provider --help
```

### Community Resources

- **Documentation:** [docs.specular.dev](https://docs.specular.dev)
- **GitHub Issues:** Report bugs and feature requests
- **Discussions:** Ask questions and share ideas

### Reporting Bugs

When reporting issues, include:

1. Specular version: `specular version`
2. Operating system and version
3. The exact command that failed
4. Complete error message
5. Relevant log output

```bash
# Generate diagnostic bundle
specular debug bundle > diagnostic.txt
```
