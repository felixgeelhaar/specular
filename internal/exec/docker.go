package exec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/safeutil"
)

// RunDocker executes a step in a Docker container with security constraints.
// The context is used to provide timeout and cancellation support.
func RunDocker(ctx context.Context, step Step) (*Result, error) {
	startTime := time.Now()

	// Build Docker command with security constraints
	args := buildDockerArgs(step)

	// Execute command with context for cancellation support
	cmd, err := safeutil.SafeCommand(ctx, "docker", args...)
	if err != nil {
		return nil, fmt.Errorf("failed docker command: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	runErr := cmd.Run()

	// Get exit code
	exitCode := 0
	if runErr != nil {
		// Check for context cancellation/timeout
		if ctx.Err() != nil {
			return nil, fmt.Errorf("docker command cancelled or timed out: %w", ctx.Err())
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start
			return nil, fmt.Errorf("failed to execute docker command: %w", runErr)
		}
	}

	return &Result{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(startTime),
		Error:    runErr,
	}, nil
}

// buildDockerArgs constructs the Docker command arguments with security constraints
func buildDockerArgs(step Step) []string {
	args := []string{
		"run",
		"--rm", // Remove container after exit
	}

	// Network configuration
	if step.Network != "" {
		args = append(args, "--network", step.Network)
	}

	// Resource limits
	if step.CPU != "" {
		args = append(args, "--cpus", step.CPU)
	}
	if step.Mem != "" {
		args = append(args, "--memory", step.Mem)
	}

	// Security constraints
	args = append(args,
		"--read-only",         // Read-only root filesystem
		"--pids-limit", "256", // Limit number of processes
		"--cap-drop", "ALL", // Drop all capabilities
	)

	// Working directory mount
	if step.Workdir != "" {
		args = append(args,
			"-v", fmt.Sprintf("%s:/workspace", step.Workdir),
			"-w", "/workspace",
		)
	}

	// Environment variables
	for key, value := range step.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Image
	args = append(args, step.Image)

	// Command and arguments
	args = append(args, step.Cmd...)

	return args
}

// ValidateDockerAvailable checks if Docker is available on the system.
// The context is used to provide timeout and cancellation support.
func ValidateDockerAvailable(ctx context.Context) error {
	cmd, err := safeutil.SafeCommand(ctx, "docker", "version")
	if err != nil {
		return fmt.Errorf("validate docker: %w", err)
	}
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("docker availability check cancelled or timed out: %w", ctx.Err())
		}
		return fmt.Errorf("docker is not available: %w", err)
	}
	return nil
}

// PullImage pulls a Docker image if not already present.
// The context is used to provide timeout and cancellation support.
func PullImage(ctx context.Context, image string) error {
	cmd, err := safeutil.SafeCommand(ctx, "docker", "pull", image)
	if err != nil {
		return fmt.Errorf("pull image: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("docker pull cancelled or timed out: %w", ctx.Err())
		}
		return fmt.Errorf("failed to pull image %s: %s", image, stderr.String())
	}
	return nil
}

// ImageExists checks if a Docker image exists locally.
// The context is used to provide timeout and cancellation support.
func ImageExists(ctx context.Context, image string) (bool, error) {
	cmd, err := safeutil.SafeCommand(ctx, "docker", "image", "inspect", image)
	if err != nil {
		return false, fmt.Errorf("image inspect: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if runErr != nil {
		// Check for context cancellation
		if ctx.Err() != nil {
			return false, fmt.Errorf("docker image inspect cancelled or timed out: %w", ctx.Err())
		}
		// Check if it's a "not found" error
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "No such") || strings.Contains(runErr.Error(), "exit status 1") {
			return false, nil
		}
		return false, fmt.Errorf("docker image inspect failed: %w: %s", runErr, stderrStr)
	}
	return true, nil
}
