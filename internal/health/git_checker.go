package health

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
)

// GitChecker checks if Git is installed and accessible.
type GitChecker struct{}

// NewGitChecker creates a new Git health checker.
func NewGitChecker() *GitChecker {
	return &GitChecker{}
}

// Name returns the name of this health check.
func (c *GitChecker) Name() string {
	return "git-binary"
}

// Check verifies Git is installed and accessible.
// Returns:
//   - Healthy if Git is installed with version >= 2.0
//   - Degraded if Git is installed but version < 2.0
//   - Unhealthy if Git is not installed or not accessible
//
// The check runs `git --version` to verify installation.
func (c *GitChecker) Check(ctx context.Context) *Result {
	// Check if git command exists
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return Unhealthy("git command not found in PATH").
			WithDetail("error", err.Error()).
			WithDetail("suggestion", "Install Git from https://git-scm.com/downloads")
	}

	// Run git --version to check installation and version
	cmd := exec.CommandContext(ctx, gitPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return Unhealthy("failed to execute git command").
			WithDetail("error", err.Error()).
			WithDetail("output", strings.TrimSpace(string(output)))
	}

	versionStr := strings.TrimSpace(string(output))
	if versionStr == "" {
		return Degraded("git executing but version unknown").
			WithDetail("git_path", gitPath)
	}

	// Parse version (format: "git version 2.42.0")
	version := parseGitVersion(versionStr)
	if version == "" {
		return Degraded("git installed but version cannot be parsed").
			WithDetail("git_path", gitPath).
			WithDetail("version_output", versionStr)
	}

	// Check if version is >= 2.0 (minimum recommended)
	majorVersion := getMajorVersion(version)
	if majorVersion != "" && majorVersion < "2" {
		return Degraded("git version is older than 2.0").
			WithDetail("git_path", gitPath).
			WithDetail("version", version).
			WithDetail("suggestion", "Upgrade Git to version 2.0 or later")
	}

	return Healthy("git is installed and accessible").
		WithDetail("git_path", gitPath).
		WithDetail("version", version)
}

// parseGitVersion extracts version number from "git version X.Y.Z" format.
func parseGitVersion(versionOutput string) string {
	// Example: "git version 2.42.0" or "git version 2.42.0.windows.1"
	parts := strings.Fields(versionOutput)
	if len(parts) < 3 {
		return ""
	}

	// The version is usually the 3rd field
	version := parts[2]

	// Validate it looks like a version (starts with a digit)
	if len(version) == 0 || version[0] < '0' || version[0] > '9' {
		return ""
	}

	// Strip platform suffixes like ".windows.1", ".darwin", etc.
	if idx := strings.Index(version, ".windows"); idx > 0 {
		version = version[:idx]
	}
	if idx := strings.Index(version, ".darwin"); idx > 0 {
		version = version[:idx]
	}
	if idx := strings.Index(version, ".linux"); idx > 0 {
		version = version[:idx]
	}

	return version
}

// getMajorVersion extracts the major version number (e.g., "2" from "2.42.0").
func getMajorVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return ""
	}

	// Validate it's a number
	if _, err := strconv.Atoi(parts[0]); err != nil {
		return ""
	}

	return parts[0]
}
