package types

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// genValidTaskID generates valid TaskID strings for property testing
func genValidTaskID() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Start with a lowercase letter
		firstChar := rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz")).Draw(t, "first_char")

		// Generate remaining characters (lowercase letters, numbers, hyphens)
		// Maximum length is 99 more characters (total 100 with first char)
		length := rapid.IntRange(0, 99).Draw(t, "length")

		if length == 0 {
			return string(firstChar)
		}

		var sb strings.Builder
		sb.WriteRune(firstChar)

		lastWasHyphen := false
		for i := 0; i < length; i++ {
			// Choose character type: letter (60%), number (30%), hyphen (10%)
			charType := rapid.IntRange(1, 10).Draw(t, fmt.Sprintf("char_type_%d", i))

			if charType <= 6 {
				// Lowercase letter
				char := rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz")).Draw(t, fmt.Sprintf("letter_%d", i))
				sb.WriteRune(char)
				lastWasHyphen = false
			} else if charType <= 9 {
				// Number
				char := rapid.RuneFrom([]rune("0123456789")).Draw(t, fmt.Sprintf("number_%d", i))
				sb.WriteRune(char)
				lastWasHyphen = false
			} else {
				// Hyphen (but avoid consecutive hyphens and trailing hyphens)
				if !lastWasHyphen && i < length-1 {
					sb.WriteRune('-')
					lastWasHyphen = true
				} else {
					// Use a letter instead
					char := rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz")).Draw(t, fmt.Sprintf("letter_alt_%d", i))
					sb.WriteRune(char)
					lastWasHyphen = false
				}
			}
		}

		return sb.String()
	})
}

// TestTaskID_ValidIDsAlwaysValidate tests that generated valid IDs always pass validation
func TestTaskID_ValidIDsAlwaysValidate(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		validID := genValidTaskID().Draw(t, "valid_id")

		taskID, err := NewTaskID(validID)
		if err != nil {
			t.Fatalf("valid ID %q should not produce error: %v", validID, err)
		}

		if err := taskID.Validate(); err != nil {
			t.Fatalf("valid ID %q should pass validation: %v", validID, err)
		}

		// Verify the ID matches what we put in
		if taskID.String() != validID {
			t.Fatalf("String() should return original value: got %q, want %q", taskID.String(), validID)
		}
	})
}

// TestTaskID_EmptyStringFails tests that empty strings always fail validation
func TestTaskID_EmptyStringFails(t *testing.T) {
	taskID := TaskID("")
	err := taskID.Validate()
	if err == nil {
		t.Error("empty string should fail validation")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("error should mention empty string: %v", err)
	}
}

// TestTaskID_TooLongFails tests that strings exceeding max length fail validation
func TestTaskID_TooLongFails(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate strings that are too long (101-200 characters)
		length := rapid.IntRange(101, 200).Draw(t, "length")

		var sb strings.Builder
		sb.WriteRune('a') // Start with valid character

		for i := 1; i < length; i++ {
			sb.WriteRune('a')
		}

		longID := sb.String()
		taskID := TaskID(longID)

		err := taskID.Validate()
		if err == nil {
			t.Fatalf("string of length %d should fail validation", length)
		}
		if !strings.Contains(err.Error(), "exceeds maximum length") {
			t.Errorf("error should mention max length: %v", err)
		}
	})
}

// TestTaskID_InvalidStartCharacterFails tests that IDs not starting with a letter fail
func TestTaskID_InvalidStartCharacterFails(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate invalid first characters (numbers, uppercase, special chars)
		firstChar := rapid.RuneFrom([]rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ-_@#$%")).Draw(t, "invalid_first")

		// Add some valid continuation
		rest := rapid.StringMatching(`[a-z0-9-]{0,50}`).Draw(t, "rest")

		invalidID := string(firstChar) + rest
		taskID := TaskID(invalidID)

		err := taskID.Validate()
		if err == nil {
			t.Fatalf("ID starting with %q should fail validation", firstChar)
		}
		if !strings.Contains(err.Error(), "must start with a letter") {
			t.Errorf("error should mention starting with a letter: %v", err)
		}
	})
}

// TestTaskID_ConsecutiveHyphensFails tests that IDs with consecutive hyphens fail
func TestTaskID_ConsecutiveHyphensFails(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate ID with consecutive hyphens somewhere in the middle
		prefix := rapid.StringMatching(`[a-z][a-z0-9]{0,20}`).Draw(t, "prefix")
		suffix := rapid.StringMatching(`[a-z0-9]{0,20}`).Draw(t, "suffix")

		invalidID := prefix + "--" + suffix
		taskID := TaskID(invalidID)

		err := taskID.Validate()
		if err == nil {
			t.Fatalf("ID with consecutive hyphens %q should fail validation", invalidID)
		}
		if !strings.Contains(err.Error(), "consecutive hyphens") {
			t.Errorf("error should mention consecutive hyphens: %v", err)
		}
	})
}

// TestTaskID_TrailingHyphenFails tests that IDs ending with a hyphen fail
func TestTaskID_TrailingHyphenFails(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate valid prefix and add trailing hyphen
		prefix := rapid.StringMatching(`[a-z][a-z0-9]{0,50}`).Draw(t, "prefix")

		invalidID := prefix + "-"
		taskID := TaskID(invalidID)

		err := taskID.Validate()
		if err == nil {
			t.Fatalf("ID with trailing hyphen %q should fail validation", invalidID)
		}
		if !strings.Contains(err.Error(), "cannot end with a hyphen") {
			t.Errorf("error should mention trailing hyphen: %v", err)
		}
	})
}

// TestTaskID_UppercaseLettersFails tests that IDs with uppercase letters fail
func TestTaskID_UppercaseLettersFails(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate ID with at least one uppercase letter
		prefix := rapid.StringMatching(`[a-z][a-z0-9]{0,10}`).Draw(t, "prefix")
		uppercase := rapid.RuneFrom([]rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")).Draw(t, "uppercase")
		suffix := rapid.StringMatching(`[a-z0-9]{0,10}`).Draw(t, "suffix")

		invalidID := prefix + string(uppercase) + suffix
		taskID := TaskID(invalidID)

		err := taskID.Validate()
		if err == nil {
			t.Fatalf("ID with uppercase letter %q should fail validation", invalidID)
		}
		if !strings.Contains(err.Error(), "must start with a letter and contain only lowercase") {
			t.Errorf("error should mention lowercase requirement: %v", err)
		}
	})
}

// TestTaskID_EqualsIsReflexive tests that a task ID always equals itself
func TestTaskID_EqualsIsReflexive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		validID := genValidTaskID().Draw(t, "valid_id")
		taskID, err := NewTaskID(validID)
		if err != nil {
			t.Fatalf("valid ID should not produce error: %v", err)
		}

		if !taskID.Equals(taskID) {
			t.Fatal("task ID should equal itself (reflexive property)")
		}
	})
}

// TestTaskID_EqualsIsSymmetric tests that equals is symmetric (if a==b then b==a)
func TestTaskID_EqualsIsSymmetric(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		validID := genValidTaskID().Draw(t, "valid_id")
		taskID1, err := NewTaskID(validID)
		if err != nil {
			t.Fatalf("valid ID should not produce error: %v", err)
		}
		taskID2, err := NewTaskID(validID)
		if err != nil {
			t.Fatalf("valid ID should not produce error: %v", err)
		}

		if taskID1.Equals(taskID2) != taskID2.Equals(taskID1) {
			t.Fatal("equals should be symmetric")
		}
	})
}

// TestTaskID_EqualsIsTransitive tests that equals is transitive (if a==b and b==c then a==c)
func TestTaskID_EqualsIsTransitive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		validID := genValidTaskID().Draw(t, "valid_id")

		taskID1, _ := NewTaskID(validID)
		taskID2, _ := NewTaskID(validID)
		taskID3, _ := NewTaskID(validID)

		if taskID1.Equals(taskID2) && taskID2.Equals(taskID3) {
			if !taskID1.Equals(taskID3) {
				t.Fatal("equals should be transitive")
			}
		}
	})
}

// TestTaskID_RoundTripThroughString tests that valid IDs survive round-trip through String()
func TestTaskID_RoundTripThroughString(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		validID := genValidTaskID().Draw(t, "valid_id")

		taskID1, err := NewTaskID(validID)
		if err != nil {
			t.Fatalf("valid ID should not produce error: %v", err)
		}

		// Convert to string and back
		strValue := taskID1.String()
		taskID2, err := NewTaskID(strValue)
		if err != nil {
			t.Fatalf("round-trip should not produce error: %v", err)
		}

		if !taskID1.Equals(taskID2) {
			t.Fatalf("round-trip should preserve equality: %q != %q", taskID1, taskID2)
		}
	})
}
