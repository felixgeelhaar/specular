# Checkpoint and Resume Mechanism

This document describes Specular's checkpoint/resume system for long-running build operations, enabling graceful interruption and continuation of work.

## Overview

The checkpoint/resume mechanism allows you to:
- ✅ Safely interrupt long-running builds (Ctrl+C)
- ✅ Resume from the last completed task
- ✅ Recover from failures without losing progress
- ✅ Review partial results before continuing
- ✅ Manage multiple concurrent builds

## How It Works

### Checkpoint Structure

Checkpoints are stored in `.specular/checkpoints/` with the following structure:

```plaintext
.specular/
└── checkpoints/
    └── <run-id>/
        ├── metadata.json       # Run information
        ├── state.json          # Current execution state
        ├── progress.json       # Task completion status
        └── outputs/            # Task outputs
            ├── task-1.json
            ├── task-2.json
            └── ...
```

### Metadata Structure

```json
{
  "run_id": "build-2024-01-15T10-30-45",
  "created_at": "2024-01-15T10:30:45Z",
  "updated_at": "2024-01-15T10:45:23Z",
  "plan_file": "plan.json",
  "plan_hash": "blake3:abc123...",
  "policy_file": ".specular/policy.yaml",
  "status": "in_progress",
  "total_tasks": 15,
  "completed_tasks": 8,
  "failed_tasks": 0
}
```

### State Structure

```json
{
  "current_task": "task-9",
  "completed": [
    "task-1",
    "task-2",
    "task-3",
    "task-4",
    "task-5",
    "task-6",
    "task-7",
    "task-8"
  ],
  "failed": [],
  "skipped": [],
  "pending": [
    "task-9",
    "task-10",
    "task-11",
    "task-12",
    "task-13",
    "task-14",
    "task-15"
  ],
  "context": {
    "docker_images_pulled": ["golang:1.22", "node:22"],
    "env_setup": true,
    "dependencies_installed": true
  }
}
```

### Progress Structure

```json
{
  "tasks": {
    "task-1": {
      "status": "completed",
      "started_at": "2024-01-15T10:30:50Z",
      "completed_at": "2024-01-15T10:32:15Z",
      "duration_ms": 85000,
      "exit_code": 0,
      "output_hash": "blake3:def456..."
    },
    "task-2": {
      "status": "completed",
      "started_at": "2024-01-15T10:32:20Z",
      "completed_at": "2024-01-15T10:35:45Z",
      "duration_ms": 205000,
      "exit_code": 0,
      "output_hash": "blake3:ghi789..."
    },
    "task-9": {
      "status": "in_progress",
      "started_at": "2024-01-15T10:45:00Z",
      "completed_at": null,
      "duration_ms": null,
      "exit_code": null
    }
  }
}
```

## Usage

### Automatic Checkpointing

Checkpoints are created automatically during `specular build`:

```bash
# Start a build (creates checkpoint)
specular build --plan plan.json --policy .specular/policy.yaml

# Interrupt with Ctrl+C
# Checkpoint is saved automatically
```

### Resuming from Checkpoint

```bash
# Resume the most recent build
specular build --resume

# Resume a specific checkpoint by run ID
specular build --resume build-2024-01-15T10-30-45

# List available checkpoints
specular checkpoint list

# Show checkpoint details
specular checkpoint show build-2024-01-15T10-30-45
```

### Managing Checkpoints

```bash
# List all checkpoints
specular checkpoint list

# Clean up old checkpoints (>7 days)
specular checkpoint clean --older-than 7d

# Clean up failed checkpoints
specular checkpoint clean --failed

# Remove specific checkpoint
specular checkpoint delete build-2024-01-15T10-30-45

# Prune all checkpoints (WARNING: removes all)
specular checkpoint prune
```

## Advanced Features

### Manual Checkpointing

Force a checkpoint at any time during execution:

```bash
# Enable manual checkpointing
specular build --plan plan.json --checkpoint-interval 5m

# Checkpoint every 5 tasks
specular build --plan plan.json --checkpoint-every 5
```

### Checkpoint Validation

Validate checkpoint integrity before resuming:

```bash
# Validate checkpoint
specular checkpoint validate build-2024-01-15T10-30-45

# Resume with validation
specular build --resume --validate
```

Output:
```
✓ Metadata valid
✓ State file valid
✓ Progress tracking valid
✓ All output files present
✓ Hashes match
```

### Partial Resume

Resume from a specific task:

```bash
# Resume from task-10 onwards
specular build --resume --from-task task-10

# Skip failed tasks and continue
specular build --resume --skip-failed

# Retry only failed tasks
specular build --resume --retry-failed
```

## Checkpoint Configuration

### Policy Settings

Configure checkpoint behavior in `policy.yaml`:

```yaml
checkpoints:
  enabled: true
  directory: ".specular/checkpoints"

  # Auto-checkpoint frequency
  interval_minutes: 5

  # Checkpoint after every N tasks
  task_interval: 10

  # Retention policy
  retention:
    max_age_days: 30
    max_count: 50
    keep_failed: true
    keep_completed: false  # Clean completed runs

  # Validation
  validate_on_resume: true
  strict_hash_check: true

  # Storage
  compress: true  # gzip checkpoint files
  max_size_mb: 500
```

### Environment Variables

Override checkpoint settings via environment:

```bash
# Disable checkpoints
export SPECULAR_CHECKPOINT_ENABLED=false

# Custom checkpoint directory
export SPECULAR_CHECKPOINT_DIR=/tmp/checkpoints

# Checkpoint interval (minutes)
export SPECULAR_CHECKPOINT_INTERVAL=10
```

## Recovery Scenarios

### Scenario 1: Network Interruption

```bash
# Build fails due to network issue
specular build --plan plan.json
# Error: network timeout while pulling Docker image

# Resume (will retry from failed task)
specular build --resume --retry-failed
```

### Scenario 2: Resource Exhaustion

```bash
# Build fails due to out of memory
specular build --plan plan.json
# Error: container killed (OOMKilled)

# Investigate
specular checkpoint show --last

# Adjust policy
vim .specular/policy.yaml  # Increase mem_limit

# Resume with new policy
specular build --resume --policy .specular/policy.yaml
```

### Scenario 3: Policy Changes

```bash
# Build partially complete
specular build --plan plan.json

# Update policy (e.g., add new linter)
vim .specular/policy.yaml

# Resume with updated policy
specular build --resume --policy .specular/policy.yaml --revalidate
```

### Scenario 4: Intentional Pause

```bash
# Start long build
specular build --plan plan.json

# Pause for lunch (Ctrl+C after task-5 completes)

# Review progress
specular checkpoint show --last

# Resume after lunch
specular build --resume
```

## Checkpoint States

### State Transitions

```
[Created] → [In Progress] → [Completed]
              ↓
          [Failed]
              ↓
          [Retrying] → [Completed]
              ↓
          [Abandoned]
```

### State Descriptions

- **Created**: Checkpoint initialized, no tasks started
- **In Progress**: Currently executing tasks
- **Paused**: Interrupted but resumable
- **Completed**: All tasks finished successfully
- **Failed**: Unrecoverable failure
- **Retrying**: Resuming after failure
- **Abandoned**: Manually abandoned or expired

## Integration with CI/CD

### GitHub Actions

```yaml
- name: Build with checkpoint
  uses: ./.github/actions/specular
  with:
    command: build
    checkpoint-enabled: true
  continue-on-error: true  # Save checkpoint on failure

- name: Upload checkpoint
  if: always()
  uses: actions/upload-artifact@v4
  with:
    name: checkpoint
    path: .specular/checkpoints/
    retention-days: 7

# Later run or retry
- name: Download checkpoint
  uses: actions/download-artifact@v4
  with:
    name: checkpoint
    path: .specular/checkpoints/

- name: Resume build
  uses: ./.github/actions/specular
  with:
    command: build
    resume: true
```

### GitLab CI

```yaml
build:
  script:
    - specular build --plan plan.json
  artifacts:
    when: always
    paths:
      - .specular/checkpoints/
    expire_in: 7 days

retry_build:
  dependencies:
    - build
  script:
    - specular build --resume
  when: on_failure
```

## Troubleshooting

### Checkpoint Not Found

**Problem:**
```
Error: checkpoint not found: build-2024-01-15T10-30-45
```

**Solution:**
```bash
# List available checkpoints
specular checkpoint list

# Use correct run ID
specular build --resume <correct-run-id>
```

### Checkpoint Corrupted

**Problem:**
```
Error: checkpoint validation failed: hash mismatch
```

**Solution:**
```bash
# Delete corrupted checkpoint
specular checkpoint delete build-2024-01-15T10-30-45

# Start fresh build
specular build --plan plan.json
```

### Disk Space Issues

**Problem:**
```
Error: failed to write checkpoint: no space left on device
```

**Solution:**
```bash
# Clean old checkpoints
specular checkpoint clean --older-than 7d

# Check disk usage
du -sh .specular/checkpoints/

# Compress large checkpoints
specular checkpoint compress --all
```

### Resume After Plan Change

**Problem:**
```
Warning: plan.json has changed since checkpoint
```

**Solution:**
```bash
# View checkpoint plan hash
specular checkpoint show --last | jq '.plan_hash'

# Compare with current plan
blake3sum plan.json

# If incompatible, start fresh
specular build --plan plan.json --force
```

## Best Practices

### 1. Regular Checkpoint Intervals

Set appropriate intervals based on task duration:

```yaml
# For quick tasks (< 1 min each)
checkpoints:
  task_interval: 10  # Every 10 tasks

# For long tasks (> 5 min each)
checkpoints:
  interval_minutes: 5  # Every 5 minutes
```

### 2. Cleanup Strategy

```bash
# Cron job to clean old checkpoints
0 2 * * * cd /path/to/project && specular checkpoint clean --older-than 7d
```

### 3. Checkpoint Validation

Always validate before resuming in CI/CD:

```bash
specular build --resume --validate || specular build --plan plan.json
```

### 4. Monitor Checkpoint Size

```bash
# Alert if checkpoint directory too large
SIZE=$(du -sm .specular/checkpoints/ | cut -f1)
if [ $SIZE -gt 1000 ]; then
  echo "Warning: Checkpoint directory exceeds 1GB"
  specular checkpoint clean --older-than 3d
fi
```

## Performance Considerations

### Checkpoint Overhead

| Interval | Overhead | Use Case |
|----------|----------|----------|
| Every task | ~5-10% | Critical builds |
| Every 5 tasks | ~2-5% | Standard builds |
| Every 10 tasks | ~1-2% | Fast builds |
| Time-based (5 min) | ~1% | Long-running tasks |

### Storage Requirements

| Task Count | Avg Checkpoint Size | 30-Day Retention |
|------------|---------------------|------------------|
| 10 tasks | ~10 MB | ~300 MB |
| 50 tasks | ~50 MB | ~1.5 GB |
| 100 tasks | ~100 MB | ~3 GB |

**Recommendation:** Enable compression for large checkpoints:

```yaml
checkpoints:
  compress: true  # Reduces size by ~70%
```

## API Reference

### Checkpoint Commands

```bash
# List checkpoints
specular checkpoint list [--status <status>] [--format json|table]

# Show checkpoint details
specular checkpoint show <run-id> [--format json]

# Validate checkpoint
specular checkpoint validate <run-id>

# Clean checkpoints
specular checkpoint clean [--older-than <duration>] [--failed]

# Delete checkpoint
specular checkpoint delete <run-id>

# Prune all checkpoints
specular checkpoint prune [--force]

# Compress checkpoint
specular checkpoint compress <run-id>

# Export checkpoint
specular checkpoint export <run-id> --out checkpoint.tar.gz

# Import checkpoint
specular checkpoint import checkpoint.tar.gz
```

### Build Resume Options

```bash
# Resume options
--resume [run-id]           # Resume from checkpoint
--resume-last               # Resume most recent
--from-task <task-id>       # Start from specific task
--skip-failed               # Skip failed tasks
--retry-failed              # Retry failed tasks only
--validate                  # Validate before resuming
--revalidate                # Re-run validations
--force                     # Ignore checkpoint, start fresh
```

## Examples

### Example 1: Long Build with Automatic Resume

```bash
#!/bin/bash
set -e

# Function to build with auto-resume
build_with_resume() {
  local max_attempts=3
  local attempt=1

  while [ $attempt -le $max_attempts ]; do
    echo "Build attempt $attempt of $max_attempts..."

    if specular build --resume 2>&1 | tee build.log; then
      echo "Build succeeded"
      return 0
    else
      echo "Build failed, will retry..."
      attempt=$((attempt + 1))
      sleep 5
    fi
  done

  echo "Build failed after $max_attempts attempts"
  return 1
}

build_with_resume
```

### Example 2: Checkpoint Monitoring

```bash
#!/bin/bash

# Monitor checkpoint progress
watch -n 5 'specular checkpoint show --last --format json | jq "{
  status,
  completed: .completed_tasks,
  total: .total_tasks,
  progress: (.completed_tasks / .total_tasks * 100 | round)
}"'
```

### Example 3: Incremental CI/CD

```yaml
# GitHub Actions
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      # Download previous checkpoint if exists
      - name: Download checkpoint
        continue-on-error: true
        uses: actions/download-artifact@v4
        with:
          name: checkpoint-${{ github.run_id }}

      # Build or resume
      - name: Build
        run: |
          if [ -d ".specular/checkpoints" ]; then
            specular build --resume
          else
            specular build --plan plan.json
          fi

      # Save checkpoint
      - name: Save checkpoint
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: checkpoint-${{ github.run_id }}
          path: .specular/checkpoints/
```

---

## See Also

- [Progress Indicators](progress-indicators.md) - Real-time progress display
- [Best Practices](best-practices.md) - General usage patterns
- [CI/CD Integration](../examples/ci-cd/) - Platform-specific examples

---

**Last Updated:** 2024-01-15
**Version:** 1.2.0
