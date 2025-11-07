# Progress Indicators

This document describes Specular's progress indication system for real-time feedback during long-running operations.

## Overview

Specular provides comprehensive progress tracking through:
- ✅ Real-time task completion updates
- ✅ Time estimates and ETAs
- ✅ Visual progress bars
- ✅ CI/CD-friendly output formats
- ✅ Detailed logging with context

## Progress Display Modes

### Interactive Mode (TTY)

When running in an interactive terminal, Specular shows rich visual progress:

```
Building Specular Project...

[████████████████████░░░░] 80% (12/15 tasks)

Current: task-13 - Generating unit tests
         Started: 2s ago
         Estimated: 5s remaining

Completed Tasks:
✓ task-1  Setup environment (2s)
✓ task-2  Install dependencies (15s)
✓ task-3  Run linters (8s)
✓ task-4  Type checking (3s)
✓ task-5  Unit tests (12s)
✓ task-6  Integration tests (25s)
✓ task-7  Build artifacts (18s)
✓ task-8  Generate docs (6s)
✓ task-9  Package application (10s)
✓ task-10 Security scan (20s)
✓ task-11 Code coverage (8s)
✓ task-12 Performance benchmarks (30s)

Total Time: 2m 37s
Estimated Completion: 15s
```

### Non-Interactive Mode (CI/CD)

When running in CI/CD or piped output, Specular uses line-based progress:

```
[2024-01-15T10:30:45Z] INFO Starting build (15 tasks)
[2024-01-15T10:30:47Z] PROGRESS [1/15] task-1: Setup environment
[2024-01-15T10:30:49Z] SUCCESS task-1 completed in 2s
[2024-01-15T10:30:49Z] PROGRESS [2/15] task-2: Install dependencies
[2024-01-15T10:31:04Z] SUCCESS task-2 completed in 15s
[2024-01-15T10:31:04Z] PROGRESS [3/15] task-3: Run linters
[2024-01-15T10:31:12Z] SUCCESS task-3 completed in 8s
...
[2024-01-15T10:33:22Z] INFO Build completed successfully
[2024-01-15T10:33:22Z] SUMMARY 15/15 tasks completed in 2m 37s
```

### JSON Mode

For programmatic consumption:

```bash
specular build --plan plan.json --format json
```

Output:
```json
{
  "type": "progress",
  "timestamp": "2024-01-15T10:30:45Z",
  "run_id": "build-2024-01-15T10-30-45",
  "total_tasks": 15,
  "completed_tasks": 12,
  "current_task": {
    "id": "task-13",
    "title": "Generating unit tests",
    "status": "in_progress",
    "started_at": "2024-01-15T10:33:20Z",
    "elapsed_ms": 2000,
    "estimated_ms": 5000
  },
  "summary": {
    "total_duration_ms": 157000,
    "estimated_remaining_ms": 15000,
    "average_task_duration_ms": 13083
  }
}
```

## Progress Components

### Progress Bar

Visual representation of completion:

```
[████████████████████░░░░] 80%
```

- ▓ Completed tasks (dark blocks)
- ░ Remaining tasks (light blocks)
- % Percentage complete

### Task Status

Individual task status indicators:

```
✓ task-1  Setup environment (2s)      # Completed
⧗ task-2  Install dependencies        # In progress
○ task-3  Run linters                 # Pending
✗ task-4  Type checking               # Failed
⊘ task-5  Security scan               # Skipped
```

### Time Estimates

Real-time duration and ETA:

```
Current Task:
  Elapsed: 2s
  Estimated: 5s remaining
  Progress: ████░░░ 40%

Overall:
  Total Time: 2m 37s
  Est. Completion: 15s
  Target: 3m 00s
```

### Throughput Metrics

Tasks per minute and velocity:

```
Performance:
  Tasks/min: 5.8
  Avg Task Duration: 10.3s
  Fastest: task-1 (2s)
  Slowest: task-12 (30s)
```

## Configuration

### Progress Settings

Configure in `~/.specular/config.yaml`:

```yaml
progress:
  # Display mode (auto, interactive, plain, json)
  mode: auto

  # Update frequency
  refresh_rate_ms: 100

  # Progress bar width
  bar_width: 40

  # Show detailed task info
  show_task_details: true

  # Show time estimates
  show_estimates: true

  # Show throughput metrics
  show_metrics: true

  # Color output
  use_colors: true

  # Compact mode (fewer lines)
  compact: false
```

### Environment Variables

Override settings via environment:

```bash
# Force specific mode
export SPECULAR_PROGRESS_MODE=json

# Disable progress output
export SPECULAR_NO_PROGRESS=true

# Set refresh rate (ms)
export SPECULAR_PROGRESS_REFRESH=200

# Disable colors
export NO_COLOR=1
```

### Command-Line Flags

Override per command:

```bash
# JSON output
specular build --format json

# Plain text (no colors/bars)
specular build --format plain

# Quiet mode (minimal output)
specular build --quiet

# Verbose mode (detailed logs)
specular build --verbose
```

## CI/CD Integration

### GitHub Actions

GitHub Actions automatically detects CI environment and uses line-based output:

```yaml
- name: Build with progress
  run: |
    specular build --plan plan.json | tee build.log
```

**Automatic Features:**
- Line-based output (no progress bars)
- Timestamps on each line
- Group folding for task details
- Annotations for errors/warnings

**Example Output in GitHub Actions:**
```
::group::Build Progress
[2024-01-15T10:30:45Z] INFO Starting build (15 tasks)
[2024-01-15T10:30:47Z] PROGRESS [1/15] task-1: Setup environment
::endgroup::

::group::task-1: Setup environment
[task output...]
::endgroup::

::notice::task-1 completed in 2s
```

### GitLab CI

GitLab CI supports ANSI colors and can show progress bars:

```yaml
build:
  script:
    - specular build --plan plan.json
  artifacts:
    reports:
      junit: .specular/test-results.xml
```

**Features:**
- Color-coded output
- Collapsible sections
- Progress percentage in job output

### CircleCI

CircleCI uses plain text mode:

```yaml
- run:
    name: Build
    command: |
      specular build --plan plan.json --format plain
```

### Jenkins

Jenkins supports ANSI colors with plugin:

```groovy
stage('Build') {
  steps {
    ansiColor('xterm') {
      sh 'specular build --plan plan.json'
    }
  }
}
```

## Advanced Features

### Custom Progress Callbacks

Configure custom progress hooks:

```yaml
# .specular/config.yaml
progress:
  hooks:
    on_task_start:
      command: "notify-send 'Task Started' '{{.TaskTitle}}'"

    on_task_complete:
      command: "notify-send 'Task Complete' '{{.TaskTitle}} ({{.Duration}})'"

    on_milestone:
      thresholds: [25, 50, 75, 100]
      command: "echo 'Milestone: {{.Percentage}}% complete'"
```

### Slack Integration

Send progress updates to Slack:

```yaml
progress:
  webhooks:
    - url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
      events: [start, milestone, complete, error]
      milestones: [50, 100]
      format: |
        {
          "text": "Build {{.Status}}: {{.CompletedTasks}}/{{.TotalTasks}} ({{.Percentage}}%)"
        }
```

### Email Notifications

Send completion emails:

```yaml
progress:
  email:
    enabled: true
    smtp_host: smtp.gmail.com
    smtp_port: 587
    from: builds@example.com
    to: team@example.com
    events: [complete, error]
    template: |
      Build {{.RunID}} {{.Status}}

      Completed: {{.CompletedTasks}}/{{.TotalTasks}}
      Duration: {{.Duration}}
```

## Progress Tracking API

### Query Progress

Get current progress programmatically:

```bash
# Get progress JSON
curl http://localhost:9090/api/v1/progress/latest

# Response
{
  "run_id": "build-2024-01-15T10-30-45",
  "status": "in_progress",
  "progress": {
    "total": 15,
    "completed": 12,
    "failed": 0,
    "percentage": 80
  },
  "current_task": {
    "id": "task-13",
    "title": "Generating unit tests",
    "elapsed_ms": 2000
  }
}
```

### Stream Progress

Subscribe to progress events:

```bash
# Server-sent events
curl -N http://localhost:9090/api/v1/progress/stream

# Output (streaming)
event: task_start
data: {"task_id":"task-13","timestamp":"2024-01-15T10:33:20Z"}

event: task_progress
data: {"task_id":"task-13","percentage":40}

event: task_complete
data: {"task_id":"task-13","duration_ms":5000}
```

## Progress Visualization

### Terminal Dashboard

Launch interactive dashboard:

```bash
specular dashboard
```

Features:
- Live progress updates
- Task timeline
- Resource usage graphs
- Recent logs
- Keyboard controls

**Controls:**
- `↑/↓` - Scroll tasks
- `Tab` - Switch panels
- `r` - Refresh
- `q` - Quit

### Web Dashboard

Start web server:

```bash
specular serve --port 8080
```

Navigate to `http://localhost:8080` for:
- Real-time progress visualization
- Task dependency graph
- Historical build analytics
- Cost tracking
- Performance metrics

## Performance Metrics

### Tracking Overhead

Progress tracking overhead is minimal:

| Update Frequency | CPU Overhead | Memory |
|------------------|--------------|--------|
| 10ms | ~2-3% | < 1 MB |
| 100ms (default) | ~0.5-1% | < 500 KB |
| 1000ms | ~0.1% | < 100 KB |

### Optimization Tips

**1. Reduce Update Frequency for CI/CD:**
```yaml
progress:
  refresh_rate_ms: 1000  # Update every second
```

**2. Disable Unnecessary Features:**
```yaml
progress:
  show_estimates: false
  show_metrics: false
  compact: true
```

**3. Use JSON Mode for Parsing:**
```bash
specular build --format json | jq '.progress'
```

## Troubleshooting

### Progress Not Updating

**Problem:** Progress appears frozen

**Solution:**
```bash
# Check if running in CI (uses different mode)
echo $CI

# Force interactive mode
specular build --format interactive

# Check refresh rate
specular config get progress.refresh_rate_ms
```

### Garbled Output

**Problem:** Progress bars show weird characters

**Solution:**
```bash
# Use plain mode
specular build --format plain

# Or disable progress
export SPECULAR_NO_PROGRESS=true
```

### Excessive Output in Logs

**Problem:** CI logs are too verbose

**Solution:**
```bash
# Use quiet mode
specular build --quiet

# Or increase update interval
export SPECULAR_PROGRESS_REFRESH=5000  # 5 seconds
```

### Missing Estimates

**Problem:** No time estimates shown

**Solution:**
```yaml
# Enable estimates in config
progress:
  show_estimates: true

# Requires at least 2 completed tasks for accurate estimates
```

## Examples

### Example 1: Monitoring Long Build

```bash
#!/bin/bash

# Start build in background
specular build --plan plan.json > build.log 2>&1 &
BUILD_PID=$!

# Monitor progress
while kill -0 $BUILD_PID 2>/dev/null; do
  # Extract latest progress
  PROGRESS=$(tail -1 build.log | grep -oP '\[\K[0-9]+/[0-9]+')
  echo "Progress: $PROGRESS"
  sleep 5
done

echo "Build complete!"
```

### Example 2: Progress in tmux Status Line

```bash
# .tmux.conf
set -g status-right '#(specular progress --short)'

# Updates every 5 seconds
set -g status-interval 5
```

### Example 3: Desktop Notifications

```bash
#!/bin/bash

specular build --plan plan.json | while read line; do
  if echo "$line" | grep -q "SUCCESS.*task-"; then
    TASK=$(echo "$line" | grep -oP 'task-\d+')
    notify-send "Task Complete" "$TASK finished"
  fi
done
```

### Example 4: Progress Dashboard Script

```bash
#!/bin/bash

# progress-dashboard.sh
while true; do
  clear
  echo "=== Specular Build Progress ==="
  echo
  specular progress --format table
  echo
  echo "Last Update: $(date)"
  sleep 1
done
```

## Best Practices

### 1. Choose Appropriate Mode

- **Development:** Use interactive mode for visual feedback
- **CI/CD:** Use plain or JSON mode for parseable output
- **Automation:** Use JSON mode for scripting

### 2. Set Reasonable Update Rates

```yaml
# For interactive terminals
progress:
  refresh_rate_ms: 100

# For CI/CD
progress:
  refresh_rate_ms: 1000
```

### 3. Enable Checkpoints for Long Builds

```bash
# Checkpoint + progress tracking
specular build --plan plan.json --checkpoint-interval 5m
```

### 4. Monitor Resource Usage

```bash
# Show resource metrics
specular build --show-resources
```

### 5. Archive Progress Logs

```bash
# Save progress log with timestamp
specular build --plan plan.json 2>&1 | tee "build-$(date +%Y%m%d-%H%M%S).log"
```

## API Reference

### Progress Commands

```bash
# Show current progress
specular progress [--format json|table|plain]

# Show progress for specific run
specular progress --run-id <run-id>

# Stream progress events
specular progress --stream

# Dashboard mode
specular dashboard [--port 8080]

# Web server
specular serve [--port 8080] [--host 0.0.0.0]
```

### Progress Formats

```bash
# Table format (default)
specular progress

# JSON format
specular progress --format json

# Short format (for status lines)
specular progress --short

# Watch mode (updates every second)
watch -n 1 specular progress --short
```

## See Also

- [Checkpoint/Resume](checkpoint-resume.md) - For long-running builds
- [CI/CD Integration](../examples/ci-cd/) - Platform-specific examples
- [Best Practices](best-practices.md) - General usage patterns

---

**Last Updated:** 2024-01-15
**Version:** 1.2.0
