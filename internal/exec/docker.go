package exec

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RunDocker executes a step in a Docker container with security constraints
func RunDocker(step Step) (*Result, error) {
	startTime := time.Now()

	// Build Docker command with security constraints
	args := buildDockerArgs(step)

	// Execute command
	cmd := exec.Command("docker", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start
			return nil, fmt.Errorf("failed to execute docker command: %w", err)
		}
	}

	return &Result{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(startTime),
		Error:    err,
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
		"--read-only",        // Read-only root filesystem
		"--pids-limit", "256", // Limit number of processes
		"--cap-drop", "ALL",   // Drop all capabilities
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

// ValidateDockerAvailable checks if Docker is available on the system
func ValidateDockerAvailable() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is not available: %w", err)
	}
	return nil
}

// PullImage pulls a Docker image if not already present
func PullImage(image string) error {
	cmd := exec.Command("docker", "pull", image)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image %s: %s", image, stderr.String())
	}
	return nil
}

// ImageExists checks if a Docker image exists locally
func ImageExists(image string) (bool, error) {
	cmd := exec.Command("docker", "image", "inspect", image)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		// Check if it's a "not found" error
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "No such") || strings.Contains(err.Error(), "exit status 1") {
			return false, nil
		}
		return false, fmt.Errorf("docker image inspect failed: %w: %s", err, stderrStr)
	}
	return true, nil
}
