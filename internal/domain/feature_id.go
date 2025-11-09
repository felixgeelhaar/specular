package domain

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
