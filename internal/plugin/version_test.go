package plugin

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input      string
		wantMajor  int
		wantMinor  int
		wantPatch  int
		wantPre    string
		wantMeta   string
		wantErr    bool
	}{
		// Basic versions
		{"1.0.0", 1, 0, 0, "", "", false},
		{"0.1.0", 0, 1, 0, "", "", false},
		{"0.0.1", 0, 0, 1, "", "", false},
		{"1.2.3", 1, 2, 3, "", "", false},
		{"10.20.30", 10, 20, 30, "", "", false},

		// With v prefix
		{"v1.0.0", 1, 0, 0, "", "", false},
		{"v2.3.4", 2, 3, 4, "", "", false},

		// Partial versions
		{"1", 1, 0, 0, "", "", false},
		{"1.2", 1, 2, 0, "", "", false},

		// With prerelease
		{"1.0.0-alpha", 1, 0, 0, "alpha", "", false},
		{"1.0.0-alpha.1", 1, 0, 0, "alpha.1", "", false},
		{"1.0.0-beta.2", 1, 0, 0, "beta.2", "", false},
		{"1.0.0-rc.1", 1, 0, 0, "rc.1", "", false},
		{"2.0.0-alpha.1.beta.2", 2, 0, 0, "alpha.1.beta.2", "", false},

		// With metadata
		{"1.0.0+build", 1, 0, 0, "", "build", false},
		{"1.0.0+build.123", 1, 0, 0, "", "build.123", false},
		{"1.0.0+20230101", 1, 0, 0, "", "20230101", false},

		// With both prerelease and metadata
		{"1.0.0-alpha+build", 1, 0, 0, "alpha", "build", false},
		{"1.0.0-beta.1+build.456", 1, 0, 0, "beta.1", "build.456", false},

		// Invalid versions
		{"", 0, 0, 0, "", "", true},
		{"invalid", 0, 0, 0, "", "", true},
		{"1.2.3.4", 0, 0, 0, "", "", true},
		{"a.b.c", 0, 0, 0, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := ParseVersion(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseVersion(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseVersion(%q) unexpected error: %v", tt.input, err)
				return
			}
			if v.Major != tt.wantMajor {
				t.Errorf("Major = %d, want %d", v.Major, tt.wantMajor)
			}
			if v.Minor != tt.wantMinor {
				t.Errorf("Minor = %d, want %d", v.Minor, tt.wantMinor)
			}
			if v.Patch != tt.wantPatch {
				t.Errorf("Patch = %d, want %d", v.Patch, tt.wantPatch)
			}
			if v.Prerelease != tt.wantPre {
				t.Errorf("Prerelease = %q, want %q", v.Prerelease, tt.wantPre)
			}
			if v.Metadata != tt.wantMeta {
				t.Errorf("Metadata = %q, want %q", v.Metadata, tt.wantMeta)
			}
		})
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		version *PluginVersion
		want    string
	}{
		{&PluginVersion{Major: 1, Minor: 0, Patch: 0}, "1.0.0"},
		{&PluginVersion{Major: 1, Minor: 2, Patch: 3}, "1.2.3"},
		{&PluginVersion{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha"}, "1.0.0-alpha"},
		{&PluginVersion{Major: 1, Minor: 0, Patch: 0, Metadata: "build"}, "1.0.0+build"},
		{&PluginVersion{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha", Metadata: "build"}, "1.0.0-alpha+build"},
		{nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.version.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		v1   string
		v2   string
		want int
	}{
		// Equal versions
		{"1.0.0", "1.0.0", 0},
		{"0.1.0", "0.1.0", 0},
		{"1.2.3", "1.2.3", 0},

		// Major comparison
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},

		// Minor comparison
		{"1.2.0", "1.1.0", 1},
		{"1.1.0", "1.2.0", -1},

		// Patch comparison
		{"1.0.2", "1.0.1", 1},
		{"1.0.1", "1.0.2", -1},

		// Prerelease comparison
		{"1.0.0", "1.0.0-alpha", 1},        // Release > prerelease
		{"1.0.0-alpha", "1.0.0", -1},       // Prerelease < release
		{"1.0.0-beta", "1.0.0-alpha", 1},   // beta > alpha (lexical)
		{"1.0.0-alpha", "1.0.0-beta", -1},  // alpha < beta
		{"1.0.0-alpha.2", "1.0.0-alpha.1", 1},
		{"1.0.0-alpha.1", "1.0.0-alpha.2", -1},

		// Numeric vs alphanumeric in prerelease
		{"1.0.0-1", "1.0.0-alpha", -1}, // Numeric < alphanumeric
		{"1.0.0-alpha", "1.0.0-1", 1},  // Alphanumeric > numeric

		// Metadata is ignored
		{"1.0.0+build1", "1.0.0+build2", 0},
		{"1.0.0", "1.0.0+build", 0},
	}

	for _, tt := range tests {
		t.Run(tt.v1+"_vs_"+tt.v2, func(t *testing.T) {
			v1, err := ParseVersion(tt.v1)
			if err != nil {
				t.Fatalf("ParseVersion(%q) error: %v", tt.v1, err)
			}
			v2, err := ParseVersion(tt.v2)
			if err != nil {
				t.Fatalf("ParseVersion(%q) error: %v", tt.v2, err)
			}

			got := v1.Compare(v2)
			if got != tt.want {
				t.Errorf("Compare(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestVersionCompareMethods(t *testing.T) {
	v1, _ := ParseVersion("1.0.0")
	v2, _ := ParseVersion("2.0.0")

	if !v1.LessThan(v2) {
		t.Error("1.0.0 should be less than 2.0.0")
	}
	if v2.LessThan(v1) {
		t.Error("2.0.0 should not be less than 1.0.0")
	}

	if !v1.LessThanOrEqual(v2) {
		t.Error("1.0.0 should be <= 2.0.0")
	}
	if !v1.LessThanOrEqual(v1) {
		t.Error("1.0.0 should be <= 1.0.0")
	}

	if !v2.GreaterThan(v1) {
		t.Error("2.0.0 should be greater than 1.0.0")
	}
	if v1.GreaterThan(v2) {
		t.Error("1.0.0 should not be greater than 2.0.0")
	}

	if !v2.GreaterThanOrEqual(v1) {
		t.Error("2.0.0 should be >= 1.0.0")
	}
	if !v1.GreaterThanOrEqual(v1) {
		t.Error("1.0.0 should be >= 1.0.0")
	}

	if !v1.Equal(v1) {
		t.Error("1.0.0 should equal 1.0.0")
	}
	if v1.Equal(v2) {
		t.Error("1.0.0 should not equal 2.0.0")
	}
}

func TestParseConstraint(t *testing.T) {
	tests := []struct {
		input    string
		operator string
		version  string
		wantErr  bool
	}{
		{">=1.0.0", ">=", "1.0.0", false},
		{"<=1.0.0", "<=", "1.0.0", false},
		{">1.0.0", ">", "1.0.0", false},
		{"<1.0.0", "<", "1.0.0", false},
		{"=1.0.0", "=", "1.0.0", false},
		{"~1.0.0", "~", "1.0.0", false},
		{"^1.0.0", "^", "1.0.0", false},
		{"1.0.0", "=", "1.0.0", false}, // Default to exact match
		{"*", "*", "", false},
		{"", "*", "", false},
		{">=invalid", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			c, err := ParseConstraint(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseConstraint(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseConstraint(%q) error: %v", tt.input, err)
				return
			}
			if c.Operator != tt.operator {
				t.Errorf("Operator = %q, want %q", c.Operator, tt.operator)
			}
			if tt.version != "" && c.Version.String() != tt.version {
				t.Errorf("Version = %q, want %q", c.Version.String(), tt.version)
			}
		})
	}
}

func TestVersionSatisfies(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		want       bool
	}{
		// Exact match
		{"1.0.0", "=1.0.0", true},
		{"1.0.0", "1.0.0", true},
		{"1.0.1", "=1.0.0", false},

		// Greater than
		{"2.0.0", ">1.0.0", true},
		{"1.0.0", ">1.0.0", false},
		{"0.9.0", ">1.0.0", false},

		// Greater than or equal
		{"2.0.0", ">=1.0.0", true},
		{"1.0.0", ">=1.0.0", true},
		{"0.9.0", ">=1.0.0", false},

		// Less than
		{"0.9.0", "<1.0.0", true},
		{"1.0.0", "<1.0.0", false},
		{"2.0.0", "<1.0.0", false},

		// Less than or equal
		{"0.9.0", "<=1.0.0", true},
		{"1.0.0", "<=1.0.0", true},
		{"2.0.0", "<=1.0.0", false},

		// Tilde (~) - patch-level changes
		{"1.2.3", "~1.2.3", true},
		{"1.2.4", "~1.2.3", true},
		{"1.2.9", "~1.2.3", true},
		{"1.2.2", "~1.2.3", false}, // Lower patch
		{"1.3.0", "~1.2.3", false}, // Different minor
		{"2.2.3", "~1.2.3", false}, // Different major

		// Caret (^) - compatible changes
		{"1.2.3", "^1.2.3", true},
		{"1.2.4", "^1.2.3", true},
		{"1.9.9", "^1.2.3", true},
		{"1.2.2", "^1.2.3", false}, // Lower version
		{"2.0.0", "^1.2.3", false}, // Different major

		// Caret with 0.x.x
		{"0.2.3", "^0.2.3", true},
		{"0.2.9", "^0.2.3", true},
		{"0.3.0", "^0.2.3", false}, // Different minor for 0.x
		{"0.2.2", "^0.2.3", false}, // Lower version

		// Caret with 0.0.x
		{"0.0.3", "^0.0.3", true},
		{"0.0.4", "^0.0.3", false}, // Different patch for 0.0.x
		{"0.0.2", "^0.0.3", false},

		// Wildcard
		{"1.0.0", "*", true},
		{"0.0.1", "*", true},
		{"99.99.99", "*", true},
	}

	for _, tt := range tests {
		t.Run(tt.version+"_satisfies_"+tt.constraint, func(t *testing.T) {
			v, err := ParseVersion(tt.version)
			if err != nil {
				t.Fatalf("ParseVersion(%q) error: %v", tt.version, err)
			}
			c, err := ParseConstraint(tt.constraint)
			if err != nil {
				t.Fatalf("ParseConstraint(%q) error: %v", tt.constraint, err)
			}

			got := v.Satisfies(c)
			if got != tt.want {
				t.Errorf("Satisfies(%q, %q) = %v, want %v", tt.version, tt.constraint, got, tt.want)
			}
		})
	}
}

func TestVersionSatisfiesAll(t *testing.T) {
	tests := []struct {
		version     string
		constraints string
		want        bool
	}{
		{"1.5.0", ">=1.0.0 <2.0.0", true},
		{"1.0.0", ">=1.0.0 <2.0.0", true},
		{"2.0.0", ">=1.0.0 <2.0.0", false},
		{"0.9.0", ">=1.0.0 <2.0.0", false},
		{"1.5.0", ">=1.0.0, <2.0.0", true}, // Comma-separated
		{"1.2.3", "^1.2.0 >=1.2.3", true},
		{"1.2.2", "^1.2.0 >=1.2.3", false},
	}

	for _, tt := range tests {
		t.Run(tt.version+"_satisfies_all_"+tt.constraints, func(t *testing.T) {
			v, err := ParseVersion(tt.version)
			if err != nil {
				t.Fatalf("ParseVersion(%q) error: %v", tt.version, err)
			}
			constraints, err := ParseConstraints(tt.constraints)
			if err != nil {
				t.Fatalf("ParseConstraints(%q) error: %v", tt.constraints, err)
			}

			got := v.SatisfiesAll(constraints)
			if got != tt.want {
				t.Errorf("SatisfiesAll(%q, %q) = %v, want %v", tt.version, tt.constraints, got, tt.want)
			}
		})
	}
}

func TestVersionIsPrerelease(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"1.0.0", false},
		{"1.0.0-alpha", true},
		{"1.0.0-beta.1", true},
		{"1.0.0+build", false}, // Metadata is not prerelease
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			v, err := ParseVersion(tt.version)
			if err != nil {
				t.Fatalf("ParseVersion(%q) error: %v", tt.version, err)
			}
			if got := v.IsPrerelease(); got != tt.want {
				t.Errorf("IsPrerelease() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersionIncrement(t *testing.T) {
	v, _ := ParseVersion("1.2.3")

	major := v.IncrementMajor()
	if major.String() != "2.0.0" {
		t.Errorf("IncrementMajor() = %s, want 2.0.0", major.String())
	}

	minor := v.IncrementMinor()
	if minor.String() != "1.3.0" {
		t.Errorf("IncrementMinor() = %s, want 1.3.0", minor.String())
	}

	patch := v.IncrementPatch()
	if patch.String() != "1.2.4" {
		t.Errorf("IncrementPatch() = %s, want 1.2.4", patch.String())
	}
}

func TestConstraintString(t *testing.T) {
	tests := []struct {
		constraint string
		want       string
	}{
		{">=1.0.0", ">=1.0.0"},
		{"^1.2.3", "^1.2.3"},
		{"*", "*"},
	}

	for _, tt := range tests {
		t.Run(tt.constraint, func(t *testing.T) {
			c, err := ParseConstraint(tt.constraint)
			if err != nil {
				t.Fatalf("ParseConstraint(%q) error: %v", tt.constraint, err)
			}
			if got := c.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNilVersionHandling(t *testing.T) {
	var v *PluginVersion

	// String should return empty
	if v.String() != "" {
		t.Error("nil version String() should return empty string")
	}

	// Compare with nil
	v2, _ := ParseVersion("1.0.0")
	if v.Compare(v2) != -1 {
		t.Error("nil.Compare(v) should return -1")
	}
	if v2.Compare(v) != 1 {
		t.Error("v.Compare(nil) should return 1")
	}
	if v.Compare(nil) != 0 {
		t.Error("nil.Compare(nil) should return 0")
	}
}
