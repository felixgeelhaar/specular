# ADR 0002: Checkpoint and Resume Mechanism

**Status:** Accepted

**Date:** 2025-01-07

**Decision Makers:** Specular Core Team

## Context

Code generation and build execution can be time-consuming operations, often taking 5-30 minutes for complex projects. If a process fails or is interrupted partway through (network issues, Docker errors, resource limits), users must restart from the beginning, wasting time and resources.

### Problems to Solve
1. **Interrupted Operations**: Network failures, Docker crashes, killed processes
2. **Resource Efficiency**: Avoid re-doing completed work
3. **Debugging**: Allow inspection of progress at failure point
4. **Long-Running Tasks**: 20+ minute builds need resilience
5. **CI/CD Timeouts**: GitHub Actions has 6-hour job timeout

### Requirements
- Save progress periodically (every 30 seconds)
- Resume from last successful task
- Preserve task artifacts and outputs
- Minimal performance overhead (<5%)
- Support concurrent operations (different checkpoint IDs)
- Auto-cleanup on success (optional keep for debugging)

## Decision

**We will implement an auto-saving JSON-based checkpoint system with configurable resume support.**

### Architecture

#### Checkpoint Storage Format
```go
type State struct {
    ID        string                 `json:"id"`
    CreatedAt time.Time              `json:"created_at"`
    UpdatedAt time.Time              `json:"updated_at"`
    Status    string                 `json:"status"` // pending, running, completed, failed
    Tasks     map[string]*TaskState  `json:"tasks"`
    Metadata  map[string]string      `json:"metadata"`
}

type TaskState struct {
    ID          string    `json:"id"`
    Status      string    `json:"status"` // pending, in_progress, completed, failed
    StartedAt   time.Time `json:"started_at,omitempty"`
    CompletedAt time.Time `json:"completed_at,omitempty"`
    Error       string    `json:"error,omitempty"`
}
```

#### File Organization
```
.specular/checkpoints/
├── build-plan.json-1704628800/
│   └── state.json
├── eval-plan.json-1704628900/
│   └── state.json
└── manifest.json  # Index of all checkpoints
```

#### Auto-Save Strategy
- Save checkpoint after every task state change
- Save checkpoint every 30 seconds during long-running tasks
- Save final state on completion or failure
- Use atomic writes (write to temp, then rename)

#### Resume Logic
```go
if resume && CheckpointExists(checkpointID) {
    state := LoadCheckpoint(checkpointID)

    // Skip completed tasks
    for task in plan.Tasks {
        if state.Tasks[task.ID].Status == "completed" {
            continue
        }
        ExecuteTask(task)
    }
} else {
    state := NewState(checkpointID)
    ExecuteAllTasks(plan.Tasks)
}
```

## Alternatives Considered

### Option 1: No Checkpointing (Restart on Failure)
**Pros:**
- Simpler implementation
- No disk I/O overhead
- Guaranteed clean state

**Cons:**
- Waste time re-doing work
- Frustrating user experience
- Inefficient in CI/CD (quota waste)

### Option 2: Database-Based Checkpoints (SQLite)
**Pros:**
- ACID transactions
- Query capabilities
- Structured data

**Cons:**
- Additional dependency
- Complexity for simple use case
- File locking issues
- Overhead for small data

### Option 3: Binary Serialization (Protocol Buffers)
**Pros:**
- Compact storage
- Fast serialization

**Cons:**
- Not human-readable (debugging)
- Schema evolution complexity
- Additional tooling needed

### Option 4: Task-Level Artifacts (One File Per Task)
**Pros:**
- Granular recovery
- Parallel writes possible

**Cons:**
- Many small files (filesystem overhead)
- Complex cleanup
- Harder to reason about state

## Implementation Details

### Checkpoint Manager
```go
type Manager struct {
    checkpointDir string
    autoSave      bool
    saveInterval  time.Duration
}

func (m *Manager) Save(state *State) error
func (m *Manager) Load(id string) (*State, error)
func (m *Manager) Exists(id string) bool
func (m *Manager) Delete(id string) error
func (m *Manager) List() ([]Checkpoint, error)
```

### Atomic Writes
```go
// Write to temporary file
tmpFile := filepath.Join(checkpointDir, id, "state.json.tmp")
WriteJSON(tmpFile, state)

// Atomic rename
os.Rename(tmpFile, filepath.Join(checkpointDir, id, "state.json"))
```

### CLI Integration
```bash
# Auto-checkpoint with default ID
specular build --plan plan.json

# Resume from last checkpoint
specular build --plan plan.json --resume

# Explicit checkpoint ID
specular build --plan plan.json --checkpoint-id my-build-001

# Keep checkpoint after success
specular build --plan plan.json --keep-checkpoint

# List checkpoints
specular checkpoint list

# Delete checkpoint
specular checkpoint delete <id>
```

## Consequences

### Positive
- ✅ Resilient to interruptions (network, Docker, process kills)
- ✅ Save time on retries (skip completed tasks)
- ✅ Better debugging (inspect state at failure)
- ✅ CI/CD friendly (handle timeouts gracefully)
- ✅ Low overhead (~50KB per checkpoint, <5% perf impact)

### Negative
- ❌ Disk space usage (~1MB per checkpoint with artifacts)
- ❌ Additional code complexity
- ❌ Potential for stale checkpoints if not cleaned up
- ❌ Resume assumes idempotent tasks

### Mitigations
- Auto-cleanup on successful completion (opt-out with `--keep-checkpoint`)
- Maximum age for checkpoints (default: 7 days)
- Checkpoint pruning command
- Clear error messages if task is not idempotent

### Performance Impact
Measured overhead:
- Save operation: ~5ms (JSON encoding + disk write)
- Load operation: ~3ms (disk read + JSON decoding)
- Total impact: <5% on 20-minute builds

## Usage Patterns

### Pattern 1: Local Development (Interrupted Build)
```bash
# Start build
specular build --plan plan.json

# ... process interrupted at task 15/50 ...

# Resume from task 16
specular build --plan plan.json --resume
```

### Pattern 2: CI/CD (Timeout Recovery)
```yaml
# GitHub Actions workflow
- name: Build (with checkpoint)
  id: build
  run: specular build --plan plan.json --checkpoint-id ci-${{ github.run_id }}
  continue-on-error: true

- name: Resume if timed out
  if: steps.build.outcome == 'failure'
  run: specular build --plan plan.json --resume --checkpoint-id ci-${{ github.run_id }}
```

### Pattern 3: Debugging Failed Execution
```bash
# Build failed at task 8
specular build --plan plan.json --checkpoint-id debug-001

# Inspect checkpoint
cat .specular/checkpoints/debug-001/state.json

# Resume with verbose logging
specular build --plan plan.json --resume --checkpoint-id debug-001 --verbose
```

## Future Enhancements

### Possible Improvements
1. **Artifact Caching**: Store Docker images, build artifacts in checkpoint
2. **Distributed Checkpoints**: Support S3/GCS for CI/CD sharing
3. **Checkpoint Comparison**: Diff tool to compare checkpoint states
4. **Progress Streaming**: Real-time checkpoint updates via WebSocket

### Non-Goals (Explicitly Out of Scope)
- Distributed coordination (use locking primitives if needed)
- Multi-process checkpointing (one checkpoint per operation)
- Checkpoint migration across Specular versions (regenerate on upgrade)

## Related Decisions
- ADR 0003: Docker-Only Execution (affects checkpoint artifact storage)
- Future ADR: Distributed execution and coordination

## References
- [Kubernetes Checkpointing](https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/)
- [Terraform State Management](https://www.terraform.io/language/state)
- [GitHub Actions Job Summaries](https://github.blog/2022-05-09-supercharging-github-actions-with-job-summaries/)
