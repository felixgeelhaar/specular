package health

import (
	"context"
	"os/exec"
	"strings"
)

// DockerChecker checks if Docker daemon is running and accessible.
type DockerChecker struct{}

// NewDockerChecker creates a new Docker health checker.
func NewDockerChecker() *DockerChecker {
	return &DockerChecker{}
}

// Name returns the name of this health check.
func (c *DockerChecker) Name() string {
	return "docker-daemon"
}

// Check verifies Docker daemon is running and accessible.
// Returns:
//   - Healthy if Docker is running and responding
//   - Unhealthy if Docker is not running or inaccessible
//
// The check runs `docker info` to verify daemon connectivity.
func (c *DockerChecker) Check(ctx context.Context) *Result {
	// Check if docker command exists
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return Unhealthy("docker command not found in PATH").
			WithDetail("error", err.Error()).
			WithDetail("suggestion", "Install Docker Desktop or Docker Engine")
	}

	// Run docker info to check daemon connectivity
	cmd := exec.CommandContext(ctx, dockerPath, "info", "--format", "{{.ServerVersion}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check for specific error conditions
		errMsg := string(output)
		if strings.Contains(errMsg, "Cannot connect to the Docker daemon") {
			return Unhealthy("Docker daemon is not running").
				WithDetail("error", strings.TrimSpace(errMsg)).
				WithDetail("suggestion", "Start Docker Desktop or Docker daemon")
		}

		return Unhealthy("Failed to connect to Docker daemon").
			WithDetail("error", err.Error()).
			WithDetail("output", strings.TrimSpace(errMsg))
	}

	version := strings.TrimSpace(string(output))
	if version == "" {
		return Degraded("Docker daemon responding but version unknown").
			WithDetail("docker_path", dockerPath)
	}

	return Healthy("Docker daemon is running").
		WithDetail("docker_path", dockerPath).
		WithDetail("server_version", version)
}
