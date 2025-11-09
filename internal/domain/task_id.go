package domain

import (
	"fmt"
	"regexp"
	"strings"
)

// TaskID represents a unique identifier for a task.
// This is a value object that enforces valid ID formats.
type TaskID string

var (
	// taskIDPattern validates that the ID contains only alphanumeric characters and hyphens
	// Must start with a letter, and can contain lowercase letters, numbers, and hyphens
	taskIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

	// maxTaskIDLength is the maximum allowed length for a task ID
	maxTaskIDLength = 100
)

// NewTaskID creates a new TaskID value object with validation
func NewTaskID(value string) (TaskID, error) {
	id := TaskID(value)
	if err := id.Validate(); err != nil {
		return "", err
	}
	return id, nil
}

// Validate checks if the task ID is valid
func (t TaskID) Validate() error {
	s := string(t)

	if s == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	if len(s) > maxTaskIDLength {
		return fmt.Errorf("task ID %q exceeds maximum length of %d characters", s, maxTaskIDLength)
	}

	if !taskIDPattern.MatchString(s) {
		return fmt.Errorf("task ID %q must start with a letter and contain only lowercase letters, numbers, and hyphens", s)
	}

	// Check for consecutive hyphens
	if strings.Contains(s, "--") {
		return fmt.Errorf("task ID %q cannot contain consecutive hyphens", s)
	}

	// Check for trailing hyphen
	if strings.HasSuffix(s, "-") {
		return fmt.Errorf("task ID %q cannot end with a hyphen", s)
	}

	return nil
}

// String returns the string representation
func (t TaskID) String() string {
	return string(t)
}

// Equals checks if this task ID equals another
func (t TaskID) Equals(other TaskID) bool {
	return t == other
}
