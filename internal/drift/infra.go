package drift

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/specular/internal/exec"
	"github.com/felixgeelhaar/specular/internal/policy"
)

// InfraDriftOptions configures infrastructure drift detection
type InfraDriftOptions struct {
	Policy       *policy.Policy     // Policy to validate against
	RunManifests []exec.RunManifest // Actual execution manifests (if available)
	TaskImages   map[string]string  // Task ID to Docker image mapping (for validation)
}

// DetectInfraDrift checks for infrastructure configuration drift
func DetectInfraDrift(opts InfraDriftOptions) []Finding {
	var findings []Finding

	if opts.Policy == nil {
		return findings
	}

	// Validate Docker image policy compliance
	if len(opts.TaskImages) > 0 {
		imageFindings := checkDockerImagePolicy(opts.TaskImages, opts.Policy)
		findings = append(findings, imageFindings...)
	}

	// Validate execution policy compliance
	execFindings := checkExecutionPolicy(opts.Policy)
	findings = append(findings, execFindings...)

	// If run manifests available, validate executions
	if len(opts.RunManifests) > 0 {
		manifestFindings := checkRunManifests(opts.RunManifests, opts.Policy)
		findings = append(findings, manifestFindings...)
	}

	return findings
}

// checkDockerImagePolicy validates that task images are allowed by policy
func checkDockerImagePolicy(taskImages map[string]string, pol *policy.Policy) []Finding {
	var findings []Finding

	if !pol.Execution.Docker.Required {
		return findings // Docker not required, skip validation
	}

	allowlist := pol.Execution.Docker.ImageAllowlist
	if len(allowlist) == 0 {
		return findings // No allowlist defined
	}

	for taskID, image := range taskImages {
		// Check if task specifies a Docker image
		if image == "" {
			findings = append(findings, Finding{
				Code:     "MISSING_DOCKER_IMAGE",
				Message:  fmt.Sprintf("Task %s missing Docker image (Docker required by policy)", taskID),
				Severity: "error",
				Location: taskID,
			})
			continue
		}

		// Check if image is in allowlist
		if !isImageAllowed(image, allowlist) {
			findings = append(findings, Finding{
				Code:     "DISALLOWED_DOCKER_IMAGE",
				Message:  fmt.Sprintf("Task %s uses disallowed Docker image: %s", taskID, image),
				Severity: "error",
				Location: taskID,
			})
		}
	}

	return findings
}

// checkExecutionPolicy validates execution configuration against policy
func checkExecutionPolicy(pol *policy.Policy) []Finding {
	var findings []Finding

	// Check if local execution is allowed
	if pol.Execution.AllowLocal {
		findings = append(findings, Finding{
			Code:     "ALLOW_LOCAL_EXECUTION",
			Message:  "Policy allows local execution (security risk)",
			Severity: "warning",
			Location: "policy.execution.allow_local",
		})
	}

	// Validate Docker configuration
	if pol.Execution.Docker.Required {
		// Check network mode
		network := pol.Execution.Docker.Network
		if network != "none" && network != "" {
			findings = append(findings, Finding{
				Code:     "NETWORK_ACCESS_ENABLED",
				Message:  fmt.Sprintf("Docker network mode '%s' allows network access", network),
				Severity: "info",
				Location: "policy.execution.docker.network",
			})
		}

		// Validate resource limits are set
		if pol.Execution.Docker.CPULimit == "" {
			findings = append(findings, Finding{
				Code:     "MISSING_CPU_LIMIT",
				Message:  "No CPU limit configured (resource exhaustion risk)",
				Severity: "warning",
				Location: "policy.execution.docker.cpu_limit",
			})
		}

		if pol.Execution.Docker.MemLimit == "" {
			findings = append(findings, Finding{
				Code:     "MISSING_MEMORY_LIMIT",
				Message:  "No memory limit configured (resource exhaustion risk)",
				Severity: "warning",
				Location: "policy.execution.docker.mem_limit",
			})
		}
	}

	// Check test policy
	if !pol.Tests.RequirePass {
		findings = append(findings, Finding{
			Code:     "TESTS_NOT_REQUIRED",
			Message:  "Tests not required to pass (quality risk)",
			Severity: "warning",
			Location: "policy.tests.require_pass",
		})
	}

	// Check security policies
	if !pol.Security.SecretsScan {
		findings = append(findings, Finding{
			Code:     "SECRETS_SCAN_DISABLED",
			Message:  "Secrets scanning disabled (security risk)",
			Severity: "warning",
			Location: "policy.security.secrets_scan",
		})
	}

	if !pol.Security.DepScan {
		findings = append(findings, Finding{
			Code:     "DEPENDENCY_SCAN_DISABLED",
			Message:  "Dependency scanning disabled (security risk)",
			Severity: "warning",
			Location: "policy.security.dep_scan",
		})
	}

	return findings
}

// checkRunManifests validates actual executions against policy
func checkRunManifests(manifests []exec.RunManifest, pol *policy.Policy) []Finding {
	var findings []Finding

	for _, manifest := range manifests {
		// Check if execution failed
		if manifest.ExitCode != 0 {
			findings = append(findings, Finding{
				Code:     "EXECUTION_FAILED",
				Message:  fmt.Sprintf("Step %s execution failed with exit code %d", manifest.StepID, manifest.ExitCode),
				Severity: "error",
				Location: manifest.StepID,
			})
		}

		// Validate Docker image if policy requires it
		if pol.Execution.Docker.Required && manifest.Image != "" {
			allowlist := pol.Execution.Docker.ImageAllowlist
			if len(allowlist) > 0 && !isImageAllowed(manifest.Image, allowlist) {
				findings = append(findings, Finding{
					Code:     "DISALLOWED_EXECUTION_IMAGE",
					Message:  fmt.Sprintf("Step %s used disallowed Docker image: %s", manifest.StepID, manifest.Image),
					Severity: "error",
					Location: manifest.StepID,
				})
			}
		}

		// Check for output hash changes that might indicate drift
		// This is informational for now - could be enhanced to compare against baselines
		//nolint:staticcheck // Empty branch reserved for future output hash validation
		if len(manifest.OutputHashes) > 0 {
			// Future: Compare against baseline hashes to detect unexpected changes
		}
	}

	return findings
}

// isImageAllowed checks if a Docker image is in the allowlist
func isImageAllowed(image string, allowlist []string) bool {
	for _, allowed := range allowlist {
		// Exact match
		if image == allowed {
			return true
		}

		// Wildcard match (e.g., "golang:*" matches "golang:1.22")
		if strings.HasSuffix(allowed, ":*") {
			prefix := strings.TrimSuffix(allowed, ":*")
			if strings.HasPrefix(image, prefix+":") {
				return true
			}
		}

		// Prefix match (e.g., "ghcr.io/acme/*" matches "ghcr.io/acme/builder:latest")
		if strings.HasSuffix(allowed, "/*") {
			prefix := strings.TrimSuffix(allowed, "/*")
			if strings.HasPrefix(image, prefix+"/") {
				return true
			}
		}
	}

	return false
}
