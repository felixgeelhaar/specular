package types

import (
	"fmt"
	"regexp"
	"strings"
)

// FeatureID represents a unique identifier for a feature.
// This is a value object that enforces valid ID formats.
type FeatureID string

var (
	// featureIDPattern validates that the ID contains only alphanumeric characters and hyphens
	// Must start with a letter, and can contain lowercase letters, numbers, and hyphens
	featureIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

	// maxFeatureIDLength is the maximum allowed length for a feature ID
	maxFeatureIDLength = 100
)

// NewFeatureID creates a new FeatureID value object with validation
func NewFeatureID(value string) (FeatureID, error) {
	id := FeatureID(value)
	if err := id.Validate(); err != nil {
		return "", err
	}
	return id, nil
}

// Validate checks if the feature ID is valid
func (f FeatureID) Validate() error {
	s := string(f)

	if s == "" {
		return fmt.Errorf("feature ID cannot be empty")
	}

	if len(s) > maxFeatureIDLength {
		return fmt.Errorf("feature ID %q exceeds maximum length of %d characters", s, maxFeatureIDLength)
	}

	if !featureIDPattern.MatchString(s) {
		return fmt.Errorf("feature ID %q must start with a letter and contain only lowercase letters, numbers, and hyphens", s)
	}

	// Check for consecutive hyphens
	if strings.Contains(s, "--") {
		return fmt.Errorf("feature ID %q cannot contain consecutive hyphens", s)
	}

	// Check for trailing hyphen
	if strings.HasSuffix(s, "-") {
		return fmt.Errorf("feature ID %q cannot end with a hyphen", s)
	}

	return nil
}

// String returns the string representation
func (f FeatureID) String() string {
	return string(f)
}

// Equals checks if this feature ID equals another
func (f FeatureID) Equals(other FeatureID) bool {
	return f == other
}

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

// Priority represents a feature or task priority level.
// This is a value object that enforces valid priority values.
type Priority string

// Valid priority levels
const (
	PriorityP0 Priority = "P0" // Critical - must have
	PriorityP1 Priority = "P1" // Important - should have
	PriorityP2 Priority = "P2" // Nice to have - could have
)

// NewPriority creates a new Priority value object with validation
func NewPriority(value string) (Priority, error) {
	p := Priority(value)
	if err := p.Validate(); err != nil {
		return "", err
	}
	return p, nil
}

// Validate checks if the priority is valid
func (p Priority) Validate() error {
	switch p {
	case PriorityP0, PriorityP1, PriorityP2:
		return nil
	default:
		return fmt.Errorf("invalid priority %q: must be P0, P1, or P2", string(p))
	}
}

// String returns the string representation
func (p Priority) String() string {
	return string(p)
}

// IsHigherThan checks if this priority is higher than another
func (p Priority) IsHigherThan(other Priority) bool {
	return priorityRank(p) > priorityRank(other)
}

// IsLowerThan checks if this priority is lower than another
func (p Priority) IsLowerThan(other Priority) bool {
	return priorityRank(p) < priorityRank(other)
}

// priorityRank returns the numeric rank of a priority (higher = more important)
func priorityRank(p Priority) int {
	switch p {
	case PriorityP0:
		return 3
	case PriorityP1:
		return 2
	case PriorityP2:
		return 1
	default:
		return 0
	}
}
