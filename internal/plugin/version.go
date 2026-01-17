package plugin

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// PluginVersion represents a parsed semantic version
type PluginVersion struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Metadata   string
	Raw        string
}

// VersionConstraint represents a version requirement
type VersionConstraint struct {
	Operator string         // >=, <=, >, <, =, ~, ^
	Version  *PluginVersion // nil for "any" constraint
}

// semver regex pattern (simplified but functional)
var semverRegex = regexp.MustCompile(`^v?(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)

// constraintRegex parses version constraints
var constraintRegex = regexp.MustCompile(`^(>=|<=|>|<|=|~|\^)?(.+)$`)

// ParseVersion parses a semantic version string
func ParseVersion(v string) (*PluginVersion, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, fmt.Errorf("empty version string")
	}

	matches := semverRegex.FindStringSubmatch(v)
	if matches == nil {
		return nil, fmt.Errorf("invalid semver format: %s", v)
	}

	pv := &PluginVersion{Raw: v}

	// Major version (required)
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %w", err)
	}
	pv.Major = major

	// Minor version (optional, defaults to 0)
	if matches[2] != "" {
		minor, err := strconv.Atoi(matches[2])
		if err != nil {
			return nil, fmt.Errorf("invalid minor version: %w", err)
		}
		pv.Minor = minor
	}

	// Patch version (optional, defaults to 0)
	if matches[3] != "" {
		patch, err := strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("invalid patch version: %w", err)
		}
		pv.Patch = patch
	}

	// Prerelease (optional)
	if len(matches) > 4 {
		pv.Prerelease = matches[4]
	}

	// Metadata (optional)
	if len(matches) > 5 {
		pv.Metadata = matches[5]
	}

	return pv, nil
}

// String returns the canonical string representation
func (v *PluginVersion) String() string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	if v.Metadata != "" {
		s += "+" + v.Metadata
	}
	return s
}

// compareInts compares two integers and returns -1, 0, or 1
func compareInts(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// Compare compares two versions
// Returns -1 if v < other, 0 if v == other, 1 if v > other
func (v *PluginVersion) Compare(other *PluginVersion) int {
	if v == nil && other == nil {
		return 0
	}
	if v == nil {
		return -1
	}
	if other == nil {
		return 1
	}

	// Compare major, minor, patch
	if cmp := compareInts(v.Major, other.Major); cmp != 0 {
		return cmp
	}
	if cmp := compareInts(v.Minor, other.Minor); cmp != 0 {
		return cmp
	}
	if cmp := compareInts(v.Patch, other.Patch); cmp != 0 {
		return cmp
	}

	// Compare prerelease
	// A version with prerelease has lower precedence than one without
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1
	}
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	if v.Prerelease != other.Prerelease {
		return comparePrerelease(v.Prerelease, other.Prerelease)
	}

	// Metadata is ignored in comparisons per semver spec
	return 0
}

// comparePrerelease compares prerelease identifiers
func comparePrerelease(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	minLen := len(partsA)
	if len(partsB) < minLen {
		minLen = len(partsB)
	}

	for i := 0; i < minLen; i++ {
		cmp := comparePrereleaseIdentifier(partsA[i], partsB[i])
		if cmp != 0 {
			return cmp
		}
	}

	// More identifiers means higher precedence
	if len(partsA) < len(partsB) {
		return -1
	}
	if len(partsA) > len(partsB) {
		return 1
	}
	return 0
}

// comparePrereleaseIdentifier compares individual prerelease identifiers
func comparePrereleaseIdentifier(a, b string) int {
	// Numeric identifiers have lower precedence than alphanumeric
	aNum, aErr := strconv.Atoi(a)
	bNum, bErr := strconv.Atoi(b)

	if aErr == nil && bErr == nil {
		// Both numeric
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
		return 0
	}

	if aErr == nil {
		// a is numeric, b is alphanumeric - numeric has lower precedence
		return -1
	}
	if bErr == nil {
		// a is alphanumeric, b is numeric - numeric has lower precedence
		return 1
	}

	// Both alphanumeric - compare lexically
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// LessThan returns true if v < other
func (v *PluginVersion) LessThan(other *PluginVersion) bool {
	return v.Compare(other) < 0
}

// LessThanOrEqual returns true if v <= other
func (v *PluginVersion) LessThanOrEqual(other *PluginVersion) bool {
	return v.Compare(other) <= 0
}

// GreaterThan returns true if v > other
func (v *PluginVersion) GreaterThan(other *PluginVersion) bool {
	return v.Compare(other) > 0
}

// GreaterThanOrEqual returns true if v >= other
func (v *PluginVersion) GreaterThanOrEqual(other *PluginVersion) bool {
	return v.Compare(other) >= 0
}

// Equal returns true if v == other (ignoring metadata)
func (v *PluginVersion) Equal(other *PluginVersion) bool {
	return v.Compare(other) == 0
}

// ParseConstraint parses a version constraint string
// Examples: ">=1.0.0", "^1.2.3", "~1.2.0", "=1.0.0", ">1.0.0", "<2.0.0"
func ParseConstraint(s string) (*VersionConstraint, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		// Any version
		return &VersionConstraint{Operator: "*", Version: nil}, nil
	}

	matches := constraintRegex.FindStringSubmatch(s)
	if matches == nil {
		return nil, fmt.Errorf("invalid constraint format: %s", s)
	}

	operator := matches[1]
	if operator == "" {
		operator = "=" // Default to exact match
	}

	version, err := ParseVersion(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid version in constraint: %w", err)
	}

	return &VersionConstraint{
		Operator: operator,
		Version:  version,
	}, nil
}

// String returns the string representation of the constraint
func (c *VersionConstraint) String() string {
	if c == nil || c.Version == nil {
		return "*"
	}
	return c.Operator + c.Version.String()
}

// Satisfies checks if version v satisfies the constraint
func (v *PluginVersion) Satisfies(c *VersionConstraint) bool {
	if c == nil || c.Version == nil || c.Operator == "*" {
		return true // Any version satisfies an empty constraint
	}

	switch c.Operator {
	case "=", "":
		return v.Equal(c.Version)
	case ">":
		return v.GreaterThan(c.Version)
	case ">=":
		return v.GreaterThanOrEqual(c.Version)
	case "<":
		return v.LessThan(c.Version)
	case "<=":
		return v.LessThanOrEqual(c.Version)
	case "~":
		// Tilde: allows patch-level changes
		// ~1.2.3 means >=1.2.3 and <1.3.0
		return v.satisfiesTilde(c.Version)
	case "^":
		// Caret: allows changes that do not modify the left-most non-zero digit
		// ^1.2.3 means >=1.2.3 and <2.0.0
		// ^0.2.3 means >=0.2.3 and <0.3.0
		// ^0.0.3 means >=0.0.3 and <0.0.4
		return v.satisfiesCaret(c.Version)
	default:
		return false
	}
}

// satisfiesTilde checks if v satisfies tilde constraint (~)
// ~1.2.3 means >=1.2.3 and <1.3.0
func (v *PluginVersion) satisfiesTilde(target *PluginVersion) bool {
	if v.Major != target.Major {
		return false
	}
	if v.Minor != target.Minor {
		return false
	}
	if v.Patch < target.Patch {
		return false
	}
	// Allow any patch >= target.Patch within same major.minor
	return true
}

// satisfiesCaret checks if v satisfies caret constraint (^)
// ^1.2.3 means >=1.2.3 and <2.0.0
// ^0.2.3 means >=0.2.3 and <0.3.0
// ^0.0.3 means >=0.0.3 and <0.0.4
func (v *PluginVersion) satisfiesCaret(target *PluginVersion) bool {
	// Must be >= target
	if v.Compare(target) < 0 {
		return false
	}

	// Find the left-most non-zero digit
	if target.Major != 0 {
		// ^1.2.3 -> allow <2.0.0
		return v.Major == target.Major
	}

	if target.Minor != 0 {
		// ^0.2.3 -> allow <0.3.0
		return v.Major == 0 && v.Minor == target.Minor
	}

	// ^0.0.3 -> allow <0.0.4
	return v.Major == 0 && v.Minor == 0 && v.Patch == target.Patch
}

// SatisfiesAll checks if version v satisfies all constraints
func (v *PluginVersion) SatisfiesAll(constraints []*VersionConstraint) bool {
	for _, c := range constraints {
		if !v.Satisfies(c) {
			return false
		}
	}
	return true
}

// ParseConstraints parses multiple constraints separated by spaces or commas
func ParseConstraints(s string) ([]*VersionConstraint, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		return []*VersionConstraint{{Operator: "*", Version: nil}}, nil
	}

	// Split by comma or space
	parts := regexp.MustCompile(`[,\s]+`).Split(s, -1)
	constraints := make([]*VersionConstraint, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		c, err := ParseConstraint(part)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, c)
	}

	return constraints, nil
}

// IsPrerelease returns true if the version has a prerelease identifier
func (v *PluginVersion) IsPrerelease() bool {
	return v != nil && v.Prerelease != ""
}

// IncrementMajor returns a new version with major incremented
func (v *PluginVersion) IncrementMajor() *PluginVersion {
	return &PluginVersion{
		Major: v.Major + 1,
		Minor: 0,
		Patch: 0,
	}
}

// IncrementMinor returns a new version with minor incremented
func (v *PluginVersion) IncrementMinor() *PluginVersion {
	return &PluginVersion{
		Major: v.Major,
		Minor: v.Minor + 1,
		Patch: 0,
	}
}

// IncrementPatch returns a new version with patch incremented
func (v *PluginVersion) IncrementPatch() *PluginVersion {
	return &PluginVersion{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch + 1,
	}
}
