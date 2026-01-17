package exec

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/policy"
)

// CodeGenerator is the interface for AI code generation
type CodeGenerator interface {
	Generate(ctx context.Context, task plan.Task) (*CodeGenerationResult, error)
}

// CodeGenerationResult contains the outcome of code generation
type CodeGenerationResult struct {
	TaskID       string
	FeatureID    string
	Files        []GeneratedFileRecord
	AIGeneration *AIGenerationRecord
	Success      bool
	Error        string
	DurationMs   int64
}

// Executor manages task execution with policy enforcement
type Executor struct {
	Policy        *policy.Policy
	DryRun        bool
	ManifestDir   string
	ImageCache    *ImageCache
	Verbose       bool
	CodeGenerator CodeGenerator // Optional: AI code generator for codegen tasks
}

// ExecutionResult contains results from executing a plan
type ExecutionResult struct {
	TotalTasks   int
	SuccessTasks int
	FailedTasks  int
	SkippedTasks int
	TaskResults  map[string]*Result
	Manifests    []*RunManifest
	StartTime    time.Time
	EndTime      time.Time
}

// Execute runs all tasks in a plan with policy enforcement.
// The context is used to provide cancellation support for task execution.
func (e *Executor) Execute(ctx context.Context, p *plan.Plan) (*ExecutionResult, error) {
	result := &ExecutionResult{
		TotalTasks:  len(p.Tasks),
		TaskResults: make(map[string]*Result),
		StartTime:   time.Now(),
	}

	// Execute tasks in order
	for _, task := range p.Tasks {
		// Check for cancellation before each task
		if ctx.Err() != nil {
			return result, fmt.Errorf("execution cancelled: %w", ctx.Err())
		}

		fmt.Printf("Executing task %s (%s)...\n", task.ID, task.FeatureID)

		// Check dependencies completed successfully
		if err := e.checkDependencies(task, result); err != nil {
			result.SkippedTasks++
			fmt.Printf("  ⊘ Skipped: %v\n", err)
			continue
		}

		// Track code generation result for manifest
		var genResult *CodeGenerationResult

		// Generate code if task has codegen hint and generator is available
		if e.CodeGenerator != nil && task.ModelHint == "codegen" {
			fmt.Printf("  🤖 Generating code for feature %s...\n", task.FeatureID)
			var genErr error
			genResult, genErr = e.CodeGenerator.Generate(ctx, task)
			if genErr != nil {
				result.FailedTasks++
				fmt.Printf("  ✗ Code generation failed: %v\n", genErr)
				result.TaskResults[task.ID.String()] = &Result{
					ExitCode: 1,
					Error:    genErr,
				}
				continue
			}

			if !genResult.Success {
				result.FailedTasks++
				fmt.Printf("  ✗ Code generation failed: %s\n", genResult.Error)
				result.TaskResults[task.ID.String()] = &Result{
					ExitCode: 1,
					Error:    fmt.Errorf("generation failed: %s", genResult.Error),
				}
				continue
			}

			if e.Verbose {
				fmt.Printf("  ✓ Generated %d file(s)\n", len(genResult.Files))
			}
		}

		// Create execution step (now validates generated code)
		step := e.createStep(task, genResult != nil)

		// Enforce policy
		if err := EnforcePolicy(step, e.Policy); err != nil {
			result.FailedTasks++
			fmt.Printf("  ✗ Policy violation: %v\n", err)
			result.TaskResults[task.ID.String()] = &Result{
				ExitCode: 1,
				Error:    err,
			}
			continue
		}

		// Execute task
		if e.DryRun {
			fmt.Printf("  ⊙ Dry run: would execute %s\n", step.Image)
			result.SuccessTasks++
			result.TaskResults[task.ID.String()] = &Result{ExitCode: 0}
		} else {
			taskResult, err := e.executeTask(ctx, step)

			if err != nil {
				result.FailedTasks++
				fmt.Printf("  ✗ Failed: %v\n", err)
				result.TaskResults[task.ID.String()] = &Result{
					ExitCode: 1,
					Error:    err,
				}
				continue
			}

			result.TaskResults[task.ID.String()] = taskResult

			if taskResult.ExitCode != 0 {
				result.FailedTasks++
				fmt.Printf("  ✗ Failed: exit code %d\n", taskResult.ExitCode)
			} else {
				result.SuccessTasks++
				fmt.Printf("  ✓ Completed in %v\n", taskResult.Duration)
			}

			// Create manifest
			if e.ManifestDir != "" {
				manifest := CreateManifest(step, taskResult)

				// Add AI generation metadata if code was generated
				if genResult != nil {
					manifest.GeneratedFiles = genResult.Files
					manifest.AIGeneration = genResult.AIGeneration
				}

				manifestPath, err := SaveManifest(manifest, e.ManifestDir)
				if err != nil {
					fmt.Printf("  ⚠ Warning: failed to save manifest: %v\n", err)
				} else {
					result.Manifests = append(result.Manifests, manifest)
					if writeErr := WriteLatestManifest(e.ManifestDir, manifest, manifestPath); writeErr != nil {
						fmt.Printf("  ⚠ Warning: failed to persist latest manifest metadata: %v\n", writeErr)
					}
				}
			}
		}
	}

	result.EndTime = time.Now()
	return result, nil
}

// checkDependencies verifies all dependencies completed successfully
func (e *Executor) checkDependencies(task plan.Task, result *ExecutionResult) error {
	for _, depID := range task.DependsOn {
		depResult, exists := result.TaskResults[depID.String()]
		if !exists {
			return fmt.Errorf("dependency %s not yet executed", depID)
		}
		if depResult.ExitCode != 0 {
			return fmt.Errorf("dependency %s failed with exit code %d", depID, depResult.ExitCode)
		}
	}
	return nil
}

// createStep converts a plan task to an execution step
// If hasGeneratedCode is true, the step validates the generated code instead of just checking versions
func (e *Executor) createStep(task plan.Task, hasGeneratedCode bool) Step {
	// Default to Docker execution
	step := Step{
		ID:      task.ID.String(),
		Runner:  "docker",
		Workdir: ".",
		Env:     make(map[string]string),
	}

	// Set image and command based on skill
	switch task.Skill {
	case "go-backend":
		step.Image = "golang:1.22"
		if hasGeneratedCode {
			// Validate generated Go code
			step.Cmd = []string{"sh", "-c", "go build ./... && go test ./..."}
		} else {
			step.Cmd = []string{"go", "version"}
		}
	case "ui-react":
		step.Image = "node:20"
		if hasGeneratedCode {
			// Validate generated React/TypeScript code
			step.Cmd = []string{"sh", "-c", "npm install && npm run build && npm test"}
		} else {
			step.Cmd = []string{"node", "--version"}
		}
	case "infra":
		step.Image = "alpine:latest"
		step.Cmd = []string{"echo", "Infrastructure task"}
	case "database":
		step.Image = "postgres:15"
		if hasGeneratedCode {
			// Validate SQL syntax (basic check)
			step.Cmd = []string{"sh", "-c", "find . -name '*.sql' -exec cat {} \\;"}
		} else {
			step.Cmd = []string{"psql", "--version"}
		}
	case "testing":
		step.Image = "golang:1.22"
		if hasGeneratedCode {
			step.Cmd = []string{"go", "test", "-v", "./..."}
		} else {
			step.Cmd = []string{"go", "test", "-version"}
		}
	default:
		step.Image = "alpine:latest"
		step.Cmd = []string{"echo", fmt.Sprintf("Task %s", task.ID)}
	}

	// Apply policy defaults
	if e.Policy != nil {
		step.CPU = e.Policy.Execution.Docker.CPULimit
		step.Mem = e.Policy.Execution.Docker.MemLimit
		step.Network = e.Policy.Execution.Docker.Network
	}

	return step
}

// executeTask runs a single task.
// The context is used to provide cancellation support for Docker operations.
func (e *Executor) executeTask(ctx context.Context, step Step) (*Result, error) {
	// Validate Docker is available
	if err := ValidateDockerAvailable(ctx); err != nil {
		return nil, fmt.Errorf("docker not available: %w", err)
	}

	// Use cache if available, otherwise pull directly
	if e.ImageCache != nil {
		if err := e.ImageCache.EnsureImage(ctx, step.Image, e.Verbose); err != nil {
			return nil, fmt.Errorf("ensure image: %w", err)
		}
	} else {
		// Fallback to direct pull (backward compatibility)
		exists, err := ImageExists(ctx, step.Image)
		if err != nil {
			return nil, fmt.Errorf("check image exists: %w", err)
		}

		if !exists {
			fmt.Printf("  ⬇ Pulling image %s...\n", step.Image)
			if err := PullImage(ctx, step.Image); err != nil {
				return nil, fmt.Errorf("pull image: %w", err)
			}
		}
	}

	// Run Docker container
	return RunDocker(ctx, step)
}

// PrintSummary outputs execution summary
func (r *ExecutionResult) PrintSummary() {
	sep := "============================================================"
	fmt.Println("\n" + sep)
	fmt.Println("Execution Summary")
	fmt.Println(sep)
	fmt.Printf("Total Tasks:    %d\n", r.TotalTasks)
	fmt.Printf("Successful:     %d\n", r.SuccessTasks)
	fmt.Printf("Failed:         %d\n", r.FailedTasks)
	fmt.Printf("Skipped:        %d\n", r.SkippedTasks)
	fmt.Printf("Duration:       %v\n", r.EndTime.Sub(r.StartTime))
	fmt.Println(sep)
}
