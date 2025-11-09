package domain

import "fmt"

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
