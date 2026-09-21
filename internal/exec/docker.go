package exec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/safeutil"
)

var (
	dockerImagePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/:@-]*$`)
	envKeyPattern      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	cpuLimitPattern    = regexp.MustCompile(`^\d+(\.\d+)?$`)
	memoryLimitPattern = regexp.MustCompile(`^\d+[bkmgBKMG]?$`)
	networkModePattern = regexp.MustCompile(`^[a-zA-Z0-9_.:-]+$`)
	// userPattern allows names (nobody), uid, or uid:gid / name:gid forms.
	userPattern = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_-]*(:[a-zA-Z0-9_][a-zA-Z0-9_-]*)?$`)
)

// RunDocker executes a step in a Docker container with security constraints.
// The context is used to provide timeout and cancellation support.
func RunDocker(ctx context.Context, step Step) (*Result, error) {
	startTime := time.Now()

	if err := validateDockerStep(step); err != nil {
		return nil, fmt.Errorf("invalid docker step: %w", err)
	}

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
		if exitErr, ok := runErr.(*exec.ExitError); ok {
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

func validateDockerStep(step Step) error {
	if strings.TrimSpace(step.Image) == "" {
		return fmt.Errorf("image is required")
	}
	if !dockerImagePattern.MatchString(step.Image) {
		return fmt.Errorf("image contains invalid characters")
	}

	if len(step.Cmd) == 0 {
		return fmt.Errorf("command is required")
	}
	for i, arg := range step.Cmd {
		if strings.ContainsRune(arg, '\x00') {
			return fmt.Errorf("command argument %d contains null byte", i)
		}
	}

	if step.Workdir != "" {
		if strings.ContainsRune(step.Workdir, '\x00') {
			return fmt.Errorf("workdir contains null byte")
		}
		if !filepath.IsAbs(step.Workdir) {
			return fmt.Errorf("workdir must be an absolute path")
		}
	}

	if step.Network != "" && !networkModePattern.MatchString(step.Network) {
		return fmt.Errorf("network contains invalid characters")
	}

	if step.User != "" && !userPattern.MatchString(step.User) {
		return fmt.Errorf("user contains invalid characters")
	}

	if step.CPU != "" && !cpuLimitPattern.MatchString(step.CPU) {
		return fmt.Errorf("cpu limit has invalid format")
	}

	if step.Mem != "" && !memoryLimitPattern.MatchString(step.Mem) {
		return fmt.Errorf("memory limit has invalid format")
	}

	for k, v := range step.Env {
		if !envKeyPattern.MatchString(k) {
			return fmt.Errorf("invalid environment variable name: %s", k)
		}
		if strings.ContainsRune(v, '\x00') {
			return fmt.Errorf("environment variable %s contains null byte", k)
		}
	}

	return nil
}

// DefaultNetworkMode is the fail-closed Docker network when a step omits one.
const DefaultNetworkMode = "none"

// DefaultContainerUser is the fail-closed non-root user when a step omits one.
const DefaultContainerUser = "nobody"

// effectiveNetwork returns the network mode for a step. Empty is fail-closed to none.
func effectiveNetwork(step Step) string {
	if step.Network == "" {
		return DefaultNetworkMode
	}
	return step.Network
}

// effectiveUser returns the container user for a step. Empty is fail-closed to nobody.
func effectiveUser(step Step) string {
	if step.User == "" {
		return DefaultContainerUser
	}
	return step.User
}

// buildDockerArgs constructs the Docker command arguments with security constraints
func buildDockerArgs(step Step) []string {
	args := []string{
		"run",
		"--rm", // Remove container after exit
	}

	// Network and user are always set; empty values fail closed to none / nobody.
	args = append(args,
		"--network", effectiveNetwork(step),
		"--user", effectiveUser(step),
	)

	// Resource limits
	if step.CPU != "" {
		args = append(args, "--cpus", step.CPU)
	}
	if step.Mem != "" {
		args = append(args, "--memory", step.Mem)
	}

	// Security constraints. Docker's default seccomp profile remains in effect
	// because we never set seccomp=unconfined (seccomp=default is not a valid
	// Docker CLI profile name and fails container create with exit 125).
	args = append(args,
		"--read-only",         // Read-only root filesystem
		"--pids-limit", "256", // Limit number of processes
		"--cap-drop", "ALL", // Drop all capabilities
		"--security-opt", "no-new-privileges", // Prevent privilege escalation
		"--tmpfs", "/tmp:rw,noexec,nosuid,size=64m", // Writable scratch under read-only root
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
