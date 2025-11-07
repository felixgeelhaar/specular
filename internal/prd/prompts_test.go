package prd

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt(t *testing.T) {
	prompt := buildSystemPrompt()

	// Check that prompt contains key instructions
	requiredPhrases := []string{
		"expert technical product manager",
		"structured product specifications",
		"JSON format",
		"Extract the product name",
		"Create unique feature IDs",
		"Assign priorities",
		"P0",
		"P1",
		"P2",
		"non-functional requirements",
		"valid JSON",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("System prompt missing required phrase: %s", phrase)
		}
	}

	// Check that prompt is not empty
	if len(prompt) < 100 {
		t.Error("System prompt is too short")
	}
}

func TestBuildUserPrompt(t *testing.T) {
	prdContent := "# My Product\n\n## Goals\n\n- Goal 1\n- Goal 2"
	prompt := buildUserPrompt(prdContent)

	// Check that prompt contains the PRD content
	if !strings.Contains(prompt, prdContent) {
		t.Error("User prompt should contain PRD content")
	}

	// Check that prompt includes JSON structure example
	requiredPhrases := []string{
		"Convert the following PRD",
		"PRD Content:",
		"Output the specification as JSON",
		"\"product\":",
		"\"goals\":",
		"\"features\":",
		"\"id\":",
		"\"priority\":",
		"\"non_functional\":",
		"Return ONLY the JSON",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("User prompt missing required phrase: %s", phrase)
		}
	}

	// Check that prompt is not empty
	if len(prompt) < len(prdContent)+100 {
		t.Error("User prompt is too short")
	}
}

func TestExtractJSONFromMarkdown_MultipleCodeBlocks(t *testing.T) {
	// The function returns the first code block it finds
	markdown := `
First code block with JSON:
` + "```json" + `
{"key": "value"}
` + "```" + `

Second code block:
` + "```" + `
Some code
` + "```" + `
`
	result := extractJSONFromMarkdown(markdown)
	expected := `{"key": "value"}`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestExtractJSONFromMarkdown_NestedBraces(t *testing.T) {
	markdown := `Some text {"outer": {"inner": {"deep": "value"}}} more text`
	result := extractJSONFromMarkdown(markdown)
	expected := `{"outer": {"inner": {"deep": "value"}}}`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestExtractJSONFromMarkdown_UnmatchedBraces(t *testing.T) {
	markdown := `{"unmatched": "brace"`
	result := extractJSONFromMarkdown(markdown)
	// Should return empty string for unmatched braces
	if result != "" {
		t.Errorf("Expected empty string for unmatched braces, got '%s'", result)
	}
}
