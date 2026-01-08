# Specular CLI Exit Codes and Error Reference

This document describes the exit codes used by the Specular CLI and provides guidance on interpreting and resolving common errors.

## Exit Codes

| Code | Name | Description |
|------|------|-------------|
| 0 | Success | Command completed successfully |
| 1 | GeneralError | An unspecified error occurred |
| 2 | UsageError | Invalid command usage (bad flags, missing arguments) |
| 3 | PolicyViolation | A policy enforcement check failed |
| 4 | DriftDetected | Configuration or state drift was detected |
| 5 | AuthError | Authentication or authorization failure |
| 6 | NetworkError | Network connectivity issue |
| 130 | Interrupted | Operation was cancelled by user (Ctrl+C) |

## Exit Code Details

### 0 - Success

The command completed without errors. Any output files were created successfully, and all operations passed.

### 1 - General Error

A catch-all error code for unexpected failures. Check the error message for specific details.

**Common causes:**
- Internal errors
- Unexpected runtime exceptions
- Configuration parsing failures
- Unhandled edge cases

**Resolution:**
1. Check the error message for details
2. Review the log file at `~/.specular/logs/latest.log`
3. If the issue persists, report it with the log output

### 2 - Usage Error

The command was invoked incorrectly.

**Common causes:**
- Missing required arguments
- Unknown flags or commands
- Invalid argument values
- Conflicting flags

**Resolution:**
1. Run `specular <command> --help` for usage information
2. Check that all required arguments are provided
3. Verify flag names and values are correct

**Examples:**
```bash
# Missing required goal
specular auto
# Error: requires at least 1 arg(s), only received 0

# Unknown flag
specular auto --invalid-flag "goal"
# Error: unknown flag: --invalid-flag
```

### 3 - Policy Violation

A policy check failed during execution.

**Common causes:**
- Budget exceeded
- Maximum steps limit reached
- Blocked operation type
- Governance rule violation

**Resolution:**
1. Review the policy configuration in your profile
2. Check the `--max-cost` or `--max-steps` flags
3. Review the policy event in the output for specific details
4. Adjust your policy or request approval if needed

**Examples:**
```bash
# Budget exceeded
specular auto "Build a complex system" --max-cost 0.01
# Error: policy violation: budget exceeded: $0.01 spent > $0.01 limit

# Step limit reached
specular auto "Goal" --max-steps 2
# Error: policy violation: maximum step count exceeded: 3 > 2 limit
```

### 4 - Drift Detected

The current state differs from the expected state.

**Common causes:**
- Specification was modified after planning
- External changes to tracked files
- Checkpoint state mismatch
- Hash verification failure

**Resolution:**
1. Run `specular drift check` to see detailed differences
2. If changes are intentional, run with `--force` to acknowledge
3. If changes are unexpected, investigate the source
4. Re-run the planning phase if needed

### 5 - Auth Error

Authentication or authorization failed.

**Common causes:**
- Invalid or expired API key
- Missing API key environment variable
- Token expiration
- Insufficient permissions
- Invalid SSO/SAML configuration

**Resolution:**
1. Verify your API keys are set correctly:
   - `OPENAI_API_KEY` for OpenAI
   - `ANTHROPIC_API_KEY` for Anthropic
   - `GEMINI_API_KEY` for Google Gemini
2. Check token expiration and refresh if needed
3. Verify SSO/SAML configuration if using enterprise auth
4. Run `specular provider doctor` to diagnose provider issues

### 6 - Network Error

A network operation failed.

**Common causes:**
- No internet connection
- DNS resolution failure
- Provider API unreachable
- Timeout exceeded
- Firewall blocking requests

**Resolution:**
1. Check your internet connection
2. Verify the provider API is accessible
3. Check for proxy or firewall configuration
4. Increase timeout with `--timeout` if needed
5. Try a different provider or model

### 130 - Interrupted

The user cancelled the operation with Ctrl+C.

**Resolution:**
- This is not an error; the operation was intentionally cancelled
- Use `specular monitor` to see the session state
- Use `specular auto resume` to continue from the last checkpoint

## Error Categories

### Provider Errors

Errors related to AI provider connectivity and responses.

```bash
# Provider not available
Error: no available providers for model type "fast"
# Resolution: Check provider configuration with `specular provider list`

# API key issues
Error: authentication failed: invalid api key
# Resolution: Verify API key environment variable is set correctly

# Rate limiting
Error: rate limit exceeded, retry after 60s
# Resolution: Wait and retry, or use a different provider
```

### Configuration Errors

Errors related to configuration files and settings.

```bash
# Invalid profile
Error: failed to load profile "custom": profile not found
# Resolution: Run `specular auto --list-profiles` to see available profiles

# Invalid YAML
Error: yaml: line 5: mapping values are not allowed in this context
# Resolution: Check YAML syntax in configuration files
```

### Execution Errors

Errors during task execution.

```bash
# Task failure
Error: task failed: go build returned exit code 1
# Resolution: Review the task output and fix the underlying issue

# Timeout
Error: task timed out after 5m0s
# Resolution: Increase timeout or break task into smaller steps
```

## Debugging

### Enable Verbose Output

Add `-v` or `--verbose` for detailed logging:

```bash
specular auto "Build API" --verbose
```

### Enable Trace Mode

For detailed execution tracing:

```bash
specular auto "Build API" --trace session-123
```

### View Logs

Check the log files for detailed information:

```bash
cat ~/.specular/logs/latest.log
```

### Run Provider Diagnostics

Verify provider health:

```bash
specular provider doctor
```

## Reporting Issues

When reporting issues, include:

1. The exact command that failed
2. The complete error message
3. The exit code
4. Relevant log output (`~/.specular/logs/latest.log`)
5. Your environment (OS, Specular version)

Get the Specular version:

```bash
specular version
```
