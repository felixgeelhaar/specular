package exec

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/ai-dev/internal/policy"
)

// EnforcePolicy validates a step against policy constraints
func EnforcePolicy(step Step, pol *policy.Policy) error {
	// Check local execution policy
	if !pol.Execution.AllowLocal && step.Runner != "docker" {
		return fmt.Errorf("policy violation: local execution not allowed (Docker-only enforced)")
	}

	// Check Docker-specific policies
	if step.Runner == "docker" {
		if err := enforceDockerPolicy(step, pol); err != nil {
			return err
		}
	}

	return nil
}

// enforceDockerPolicy validates Docker-specific constraints
func enforceDockerPolicy(step Step, pol *policy.Policy) error {
	dockerPolicy := pol.Execution.Docker

	// Check if Docker is required
	if dockerPolicy.Required && step.Runner != "docker" {
		return fmt.Errorf("policy violation: Docker execution required")
	}

	// Check image allowlist
	if len(dockerPolicy.ImageAllowlist) > 0 {
		allowed := false
		for _, allowedImage := range dockerPolicy.ImageAllowlist {
			if matchesImagePattern(step.Image, allowedImage) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("policy violation: image not in allowlist: %s", step.Image)
		}
	}

	// Validate network mode
	if step.Network != "" && dockerPolicy.Network != "" {
		if step.Network != dockerPolicy.Network {
			return fmt.Errorf("policy violation: network mode '%s' not allowed (required: '%s')",
				step.Network, dockerPolicy.Network)
		}
	}

	return nil
}

// matchesImagePattern checks if an image matches a pattern
// Supports exact match and wildcard patterns
func matchesImagePattern(image, pattern string) bool {
	// Exact match
	if image == pattern {
		return true
	}

	// Wildcard pattern (simple implementation)
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(image, prefix)
	}

	return false
}
