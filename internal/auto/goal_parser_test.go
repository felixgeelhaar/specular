package auto

import (
	"testing"

	"github.com/felixgeelhaar/specular/internal/router"
)

func TestNewGoalParser(t *testing.T) {
	// Create a nil router for testing (we're just testing constructor)
	var r *router.Router = nil
	parser := NewGoalParser(r)

	if parser == nil {
		t.Fatal("NewGoalParser returned nil")
	}
	if parser.router != r {
		t.Error("GoalParser router was not set correctly")
	}
}

func TestCleanYAMLResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean YAML without markers",
			input:    "name: test\ndescription: A test",
			expected: "name: test\ndescription: A test",
		},
		{
			name:     "YAML with ```yaml markers",
			input:    "```yaml\nname: test\ndescription: A test\n```",
			expected: "name: test\ndescription: A test",
		},
		{
			name:     "YAML with ```yml markers",
			input:    "```yml\nname: test\ndescription: A test\n```",
			expected: "name: test\ndescription: A test",
		},
		{
			name:     "YAML with generic ``` markers",
			input:    "```\nname: test\ndescription: A test\n```",
			expected: "name: test\ndescription: A test",
		},
		{
			name:     "YAML with extra whitespace",
			input:    "\n\n  name: test\n  description: A test  \n\n",
			expected: "name: test\n  description: A test",
		},
		{
			name:     "YAML with mixed markers and whitespace",
			input:    "```yaml\n  name: test\n  description: A test\n```",
			expected: "name: test\n  description: A test",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only markers",
			input:    "```yaml\n```",
			expected: "",
		},
		{
			name:     "YAML with code block prefix only",
			input:    "```yaml\nname: test\ndescription: A test",
			expected: "name: test\ndescription: A test",
		},
		{
			name:     "YAML with code block suffix only",
			input:    "name: test\ndescription: A test\n```",
			expected: "name: test\ndescription: A test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanYAMLResponse(tt.input)
			if result != tt.expected {
				t.Errorf("cleanYAMLResponse() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanYAMLResponse_PreservesContent(t *testing.T) {
	// Test that actual YAML content is preserved
	input := "```yaml\n" +
		"name: my-project\n" +
		"description: A test project\n" +
		"version: 1.0.0\n" +
		"features:\n" +
		"  - id: feature-1\n" +
		"    title: Feature One\n" +
		"    priority: P0\n" +
		"```"
	expected := `name: my-project
description: A test project
version: 1.0.0
features:
  - id: feature-1
    title: Feature One
    priority: P0`

	result := cleanYAMLResponse(input)
	if result != expected {
		t.Errorf("cleanYAMLResponse() did not preserve YAML structure.\nGot:\n%s\nWant:\n%s", result, expected)
	}
}

func TestCleanYAMLResponse_MultipleCodeBlocks(t *testing.T) {
	// Only removes the outermost markers
	input := "```yaml\ncode: `value`\nmore: ```inline```\n```"
	result := cleanYAMLResponse(input)

	// Should remove outer markers but preserve inner backticks
	if !contains(result, "`value`") {
		t.Error("cleanYAMLResponse removed inner backticks")
	}
	if !contains(result, "```inline```") {
		t.Error("cleanYAMLResponse removed inline triple backticks")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
