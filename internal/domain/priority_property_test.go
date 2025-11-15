package domain

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// genValidPriority generates valid Priority values for property testing
func genValidPriority() *rapid.Generator[Priority] {
	return rapid.SampledFrom([]Priority{PriorityP0, PriorityP1, PriorityP2})
}

// genInvalidPriority generates invalid Priority strings
func genInvalidPriority() *rapid.Generator[string] {
	return rapid.OneOf(
		// Empty string
		rapid.Just(""),
		// Wrong case
		rapid.SampledFrom([]string{"p0", "p1", "p2", "P0 ", " P1", "P2 "}),
		// Wrong format
		rapid.SampledFrom([]string{"P3", "P4", "P-1", "Priority0", "HIGH", "LOW"}),
		// Random strings
		rapid.StringMatching(`[A-Za-z]{1,10}`).Filter(func(s string) bool {
			return s != "P0" && s != "P1" && s != "P2"
		}),
	)
}

// TestPriority_ValidPrioritiesAlwaysValidate tests that all valid priorities pass validation
func TestPriority_ValidPrioritiesAlwaysValidate(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		validPriority := genValidPriority().Draw(t, "valid_priority")

		if err := validPriority.Validate(); err != nil {
			t.Fatalf("valid priority %q should pass validation: %v", validPriority, err)
		}

		// Verify String() returns the expected value
		str := validPriority.String()
		if str != "P0" && str != "P1" && str != "P2" {
			t.Fatalf("String() should return P0, P1, or P2, got %q", str)
		}
	})
}

// TestPriority_InvalidPrioritiesFail tests that invalid priorities fail validation
func TestPriority_InvalidPrioritiesFail(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invalidPriorityStr := genInvalidPriority().Draw(t, "invalid_priority")
		invalidPriority := Priority(invalidPriorityStr)

		err := invalidPriority.Validate()
		if err == nil {
			t.Fatalf("invalid priority %q should fail validation", invalidPriorityStr)
		}
		if !strings.Contains(err.Error(), "must be P0, P1, or P2") {
			t.Errorf("error should mention valid values: %v", err)
		}
	})
}

// TestPriority_RoundTripThroughString tests that priorities survive round-trip through String()
func TestPriority_RoundTripThroughString(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		priority1 := genValidPriority().Draw(t, "priority")

		// Convert to string and back
		strValue := priority1.String()
		priority2, err := NewPriority(strValue)
		if err != nil {
			t.Fatalf("round-trip should not produce error: %v", err)
		}

		if priority1 != priority2 {
			t.Fatalf("round-trip should preserve value: %q != %q", priority1, priority2)
		}
	})
}

// TestPriority_ComparisonIsConsistent tests that IsHigherThan and IsLowerThan are consistent
func TestPriority_ComparisonIsConsistent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		priority1 := genValidPriority().Draw(t, "priority1")
		priority2 := genValidPriority().Draw(t, "priority2")

		// If priority1 is higher than priority2, then priority2 must be lower than priority1
		if priority1.IsHigherThan(priority2) {
			if !priority2.IsLowerThan(priority1) {
				t.Fatalf("%s is higher than %s, so %s should be lower than %s", priority1, priority2, priority2, priority1)
			}
		}

		// If priority1 is lower than priority2, then priority2 must be higher than priority1
		if priority1.IsLowerThan(priority2) {
			if !priority2.IsHigherThan(priority1) {
				t.Fatalf("%s is lower than %s, so %s should be higher than %s", priority1, priority2, priority2, priority1)
			}
		}
	})
}

// TestPriority_ComparisonIsAntisymmetric tests antisymmetry: not (a > b and b > a)
func TestPriority_ComparisonIsAntisymmetric(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		priority1 := genValidPriority().Draw(t, "priority1")
		priority2 := genValidPriority().Draw(t, "priority2")

		// Can't both be higher than each other
		if priority1.IsHigherThan(priority2) && priority2.IsHigherThan(priority1) {
			t.Fatalf("antisymmetry violated: %s and %s can't both be higher than each other", priority1, priority2)
		}

		// Can't both be lower than each other
		if priority1.IsLowerThan(priority2) && priority2.IsLowerThan(priority1) {
			t.Fatalf("antisymmetry violated: %s and %s can't both be lower than each other", priority1, priority2)
		}
	})
}

// TestPriority_ComparisonIsTransitive tests transitivity: if a > b and b > c then a > c
func TestPriority_ComparisonIsTransitive(t *testing.T) {
	// Use all three priorities to test transitivity
	priorities := []Priority{PriorityP0, PriorityP1, PriorityP2}

	// P0 > P1, P1 > P2, therefore P0 > P2
	if !priorities[0].IsHigherThan(priorities[1]) {
		t.Fatal("P0 should be higher than P1")
	}
	if !priorities[1].IsHigherThan(priorities[2]) {
		t.Fatal("P1 should be higher than P2")
	}
	if !priorities[0].IsHigherThan(priorities[2]) {
		t.Fatal("transitivity violated: P0 should be higher than P2")
	}

	// P2 < P1, P1 < P0, therefore P2 < P0
	if !priorities[2].IsLowerThan(priorities[1]) {
		t.Fatal("P2 should be lower than P1")
	}
	if !priorities[1].IsLowerThan(priorities[0]) {
		t.Fatal("P1 should be lower than P0")
	}
	if !priorities[2].IsLowerThan(priorities[0]) {
		t.Fatal("transitivity violated: P2 should be lower than P0")
	}
}

// TestPriority_ComparisonIsComplete tests that any two priorities can be compared
func TestPriority_ComparisonIsComplete(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		priority1 := genValidPriority().Draw(t, "priority1")
		priority2 := genValidPriority().Draw(t, "priority2")

		// Either a > b, a < b, or a == b (one must be true)
		isGreater := priority1.IsHigherThan(priority2)
		isLess := priority1.IsLowerThan(priority2)
		isEqual := priority1 == priority2

		// Exactly one should be true
		trueCount := 0
		if isGreater {
			trueCount++
		}
		if isLess {
			trueCount++
		}
		if isEqual {
			trueCount++
		}

		if trueCount != 1 {
			t.Fatalf("comparison completeness violated: %s vs %s has %d true conditions (should be exactly 1)", priority1, priority2, trueCount)
		}
	})
}

// TestPriority_SelfComparisonIsConsistent tests that a priority is neither higher nor lower than itself
func TestPriority_SelfComparisonIsConsistent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		priority := genValidPriority().Draw(t, "priority")

		if priority.IsHigherThan(priority) {
			t.Fatalf("%s should not be higher than itself", priority)
		}

		if priority.IsLowerThan(priority) {
			t.Fatalf("%s should not be lower than itself", priority)
		}
	})
}

// TestPriority_OrderingIsCorrect tests that P0 > P1 > P2
func TestPriority_OrderingIsCorrect(t *testing.T) {
	// P0 (Critical) is highest priority
	if !PriorityP0.IsHigherThan(PriorityP1) {
		t.Fatal("P0 should be higher priority than P1")
	}
	if !PriorityP0.IsHigherThan(PriorityP2) {
		t.Fatal("P0 should be higher priority than P2")
	}

	// P1 (Important) is middle priority
	if !PriorityP1.IsHigherThan(PriorityP2) {
		t.Fatal("P1 should be higher priority than P2")
	}
	if !PriorityP1.IsLowerThan(PriorityP0) {
		t.Fatal("P1 should be lower priority than P0")
	}

	// P2 (Nice to have) is lowest priority
	if !PriorityP2.IsLowerThan(PriorityP1) {
		t.Fatal("P2 should be lower priority than P1")
	}
	if !PriorityP2.IsLowerThan(PriorityP0) {
		t.Fatal("P2 should be lower priority than P0")
	}
}

// TestPriority_ConstructorValidates tests that NewPriority performs validation
func TestPriority_ConstructorValidates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Valid priorities should succeed
		validPriorityStr := genValidPriority().Draw(t, "valid").String()
		_, err := NewPriority(validPriorityStr)
		if err != nil {
			t.Fatalf("NewPriority with valid priority %q should not error: %v", validPriorityStr, err)
		}

		// Invalid priorities should fail
		invalidPriorityStr := genInvalidPriority().Draw(t, "invalid")
		_, err = NewPriority(invalidPriorityStr)
		if err == nil {
			t.Fatalf("NewPriority with invalid priority %q should error", invalidPriorityStr)
		}
	})
}
