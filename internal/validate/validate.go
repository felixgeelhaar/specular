// Package validate provides input validation utilities for CLI commands.
// It ensures user inputs meet security and format requirements before processing.
package validate

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// Limits for input validation
const (
	// MaxGoalLength is the maximum allowed length for a goal string
	MaxGoalLength = 2000

	// MinGoalLength is the minimum required length for a goal string
	MinGoalLength = 3

	// MaxProjectNameLength is the maximum allowed length for project names
	MaxProjectNameLength = 100

	// MaxProfileNameLength is the maximum allowed length for profile names
	MaxProfileNameLength = 50

	// MaxModelNameLength is the maximum allowed length for model names
	MaxModelNameLength = 100

	// MaxURLLength is the maximum allowed length for URLs
	MaxURLLength = 2048

	// MaxPathLength is the maximum allowed length for file paths
	MaxPathLength = 4096
)

// ValidationError represents a validation failure with context
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Value != "" {
		// Truncate value for display if too long
		displayValue := e.Value
		if len(displayValue) > 50 {
			displayValue = displayValue[:47] + "..."
		}
		return fmt.Sprintf("invalid %s %q: %s", e.Field, displayValue, e.Message)
	}
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Message)
}

// Goal validates a user-provided goal string
func Goal(goal string) error {
	goal = strings.TrimSpace(goal)

	if goal == "" {
		return &ValidationError{
			Field:   "goal",
			Message: "goal cannot be empty",
		}
	}

	if len(goal) < MinGoalLength {
		return &ValidationError{
			Field:   "goal",
			Value:   goal,
			Message: fmt.Sprintf("goal must be at least %d characters", MinGoalLength),
		}
	}

	if len(goal) > MaxGoalLength {
		return &ValidationError{
			Field:   "goal",
			Value:   goal[:50],
			Message: fmt.Sprintf("goal exceeds maximum length of %d characters", MaxGoalLength),
		}
	}

	// Check for control characters (except common whitespace)
	for _, r := range goal {
		if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
			return &ValidationError{
				Field:   "goal",
				Message: "goal contains invalid control characters",
			}
		}
	}

	return nil
}

// ProfileName validates a profile name
func ProfileName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return &ValidationError{
			Field:   "profile name",
			Message: "profile name cannot be empty",
		}
	}

	if len(name) > MaxProfileNameLength {
		return &ValidationError{
			Field:   "profile name",
			Value:   name,
			Message: fmt.Sprintf("profile name exceeds maximum length of %d characters", MaxProfileNameLength),
		}
	}

	// Profile names should be alphanumeric with hyphens/underscores
	validName := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	if !validName.MatchString(name) {
		return &ValidationError{
			Field:   "profile name",
			Value:   name,
			Message: "profile name must start with a letter and contain only letters, numbers, hyphens, and underscores",
		}
	}

	return nil
}

// ProjectName validates a project name
func ProjectName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return &ValidationError{
			Field:   "project name",
			Message: "project name cannot be empty",
		}
	}

	if len(name) > MaxProjectNameLength {
		return &ValidationError{
			Field:   "project name",
			Value:   name,
			Message: fmt.Sprintf("project name exceeds maximum length of %d characters", MaxProjectNameLength),
		}
	}

	// Check for path traversal attempts first (security priority)
	if strings.Contains(name, "..") {
		return &ValidationError{
			Field:   "project name",
			Value:   name,
			Message: "project name cannot contain path traversal sequences",
		}
	}

	// Project names should be safe for filesystem usage
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	if invalidChars.MatchString(name) {
		return &ValidationError{
			Field:   "project name",
			Value:   name,
			Message: "project name contains invalid characters",
		}
	}

	return nil
}

// ModelName validates an AI model name
func ModelName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return &ValidationError{
			Field:   "model name",
			Message: "model name cannot be empty",
		}
	}

	if len(name) > MaxModelNameLength {
		return &ValidationError{
			Field:   "model name",
			Value:   name,
			Message: fmt.Sprintf("model name exceeds maximum length of %d characters", MaxModelNameLength),
		}
	}

	// Model names should be alphanumeric with common separators
	validName := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:/-]*$`)
	if !validName.MatchString(name) {
		return &ValidationError{
			Field:   "model name",
			Value:   name,
			Message: "model name contains invalid characters",
		}
	}

	return nil
}

// URL validates a URL string
func URL(urlStr string) error {
	urlStr = strings.TrimSpace(urlStr)

	if urlStr == "" {
		return &ValidationError{
			Field:   "URL",
			Message: "URL cannot be empty",
		}
	}

	if len(urlStr) > MaxURLLength {
		return &ValidationError{
			Field:   "URL",
			Value:   urlStr[:50],
			Message: fmt.Sprintf("URL exceeds maximum length of %d characters", MaxURLLength),
		}
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return &ValidationError{
			Field:   "URL",
			Value:   urlStr,
			Message: fmt.Sprintf("invalid URL format: %v", err),
		}
	}

	// Require scheme
	if parsed.Scheme == "" {
		return &ValidationError{
			Field:   "URL",
			Value:   urlStr,
			Message: "URL must include a scheme (e.g., https://)",
		}
	}

	// Only allow http and https schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return &ValidationError{
			Field:   "URL",
			Value:   urlStr,
			Message: "URL scheme must be http or https",
		}
	}

	// Require host
	if parsed.Host == "" {
		return &ValidationError{
			Field:   "URL",
			Value:   urlStr,
			Message: "URL must include a host",
		}
	}

	return nil
}

// FilePath validates a file path for safety
func FilePath(path string) error {
	path = strings.TrimSpace(path)

	if path == "" {
		return &ValidationError{
			Field:   "file path",
			Message: "file path cannot be empty",
		}
	}

	if len(path) > MaxPathLength {
		return &ValidationError{
			Field:   "file path",
			Value:   path[:50],
			Message: fmt.Sprintf("file path exceeds maximum length of %d characters", MaxPathLength),
		}
	}

	// Check for null bytes (security issue)
	if strings.ContainsRune(path, '\x00') {
		return &ValidationError{
			Field:   "file path",
			Message: "file path contains null bytes",
		}
	}

	return nil
}

// CheckpointID validates a checkpoint ID
func CheckpointID(id string) error {
	id = strings.TrimSpace(id)

	if id == "" {
		return &ValidationError{
			Field:   "checkpoint ID",
			Message: "checkpoint ID cannot be empty",
		}
	}

	// Checkpoint IDs should follow format: auto-{timestamp} or similar
	validID := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	if !validID.MatchString(id) {
		return &ValidationError{
			Field:   "checkpoint ID",
			Value:   id,
			Message: "checkpoint ID contains invalid characters",
		}
	}

	if len(id) > 100 {
		return &ValidationError{
			Field:   "checkpoint ID",
			Value:   id,
			Message: "checkpoint ID exceeds maximum length",
		}
	}

	return nil
}

// PositiveFloat validates that a float is positive (or zero if allowZero)
func PositiveFloat(name string, value float64, allowZero bool) error {
	if value < 0 {
		return &ValidationError{
			Field:   name,
			Value:   fmt.Sprintf("%f", value),
			Message: fmt.Sprintf("%s cannot be negative", name),
		}
	}

	if !allowZero && value == 0 {
		return &ValidationError{
			Field:   name,
			Value:   "0",
			Message: fmt.Sprintf("%s must be greater than zero", name),
		}
	}

	return nil
}

// PositiveInt validates that an integer is positive (or zero if allowZero)
func PositiveInt(name string, value int, allowZero bool) error {
	if value < 0 {
		return &ValidationError{
			Field:   name,
			Value:   fmt.Sprintf("%d", value),
			Message: fmt.Sprintf("%s cannot be negative", name),
		}
	}

	if !allowZero && value == 0 {
		return &ValidationError{
			Field:   name,
			Value:   "0",
			Message: fmt.Sprintf("%s must be greater than zero", name),
		}
	}

	return nil
}

// InRange validates that an integer is within a specified range
func InRange(name string, value, min, max int) error {
	if value < min || value > max {
		return &ValidationError{
			Field:   name,
			Value:   fmt.Sprintf("%d", value),
			Message: fmt.Sprintf("%s must be between %d and %d", name, min, max),
		}
	}
	return nil
}

// OneOf validates that a string value is one of the allowed values
func OneOf(name string, value string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}

	return &ValidationError{
		Field:   name,
		Value:   value,
		Message: fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")),
	}
}
