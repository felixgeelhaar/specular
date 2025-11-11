package auto

import (
	"testing"

	"github.com/felixgeelhaar/specular/internal/domain"
	"github.com/felixgeelhaar/specular/internal/plan"
)

func TestCountTasksByPriority(t *testing.T) {
	tasks := []plan.Task{
		{Priority: domain.PriorityP0},
		{Priority: domain.PriorityP0},
		{Priority: domain.PriorityP1},
		{Priority: domain.PriorityP1},
		{Priority: domain.PriorityP1},
		{Priority: domain.PriorityP2},
	}

	p0, p1, p2 := countTasksByPriority(tasks)

	if p0 != 2 {
		t.Errorf("P0 count = %d, want 2", p0)
	}
	if p1 != 3 {
		t.Errorf("P1 count = %d, want 3", p1)
	}
	if p2 != 1 {
		t.Errorf("P2 count = %d, want 1", p2)
	}
}

func TestCountTasksByPriority_Empty(t *testing.T) {
	tasks := []plan.Task{}
	p0, p1, p2 := countTasksByPriority(tasks)

	if p0 != 0 || p1 != 0 || p2 != 0 {
		t.Errorf("Empty tasks should return all zeros, got P0=%d, P1=%d, P2=%d", p0, p1, p2)
	}
}

func TestCountTasksByPriority_SinglePriority(t *testing.T) {
	tasks := []plan.Task{
		{Priority: domain.PriorityP0},
		{Priority: domain.PriorityP0},
		{Priority: domain.PriorityP0},
	}

	p0, p1, p2 := countTasksByPriority(tasks)

	if p0 != 3 {
		t.Errorf("P0 count = %d, want 3", p0)
	}
	if p1 != 0 {
		t.Errorf("P1 count = %d, want 0", p1)
	}
	if p2 != 0 {
		t.Errorf("P2 count = %d, want 0", p2)
	}
}

func TestCountTasksBySkill(t *testing.T) {
	tasks := []plan.Task{
		{Skill: "go-backend"},
		{Skill: "go-backend"},
		{Skill: "ui-react"},
		{Skill: "database"},
		{Skill: "testing"},
		{Skill: "testing"},
		{Skill: "testing"},
		{Skill: ""}, // Empty skill should not be counted
	}

	counts := countTasksBySkill(tasks)

	expectedCounts := map[string]int{
		"go-backend": 2,
		"ui-react":   1,
		"database":   1,
		"testing":    3,
	}

	for skill, expected := range expectedCounts {
		if counts[skill] != expected {
			t.Errorf("Skill %s count = %d, want %d", skill, counts[skill], expected)
		}
	}

	// Empty skill should not be in the map
	if _, exists := counts[""]; exists {
		t.Error("Empty skill should not be counted")
	}
}

func TestCountTasksBySkill_Empty(t *testing.T) {
	tasks := []plan.Task{}
	counts := countTasksBySkill(tasks)

	if len(counts) != 0 {
		t.Errorf("Empty tasks should return empty map, got %d entries", len(counts))
	}
}

func TestCountTasksBySkill_AllEmptySkills(t *testing.T) {
	tasks := []plan.Task{
		{Skill: ""},
		{Skill: ""},
		{Skill: ""},
	}

	counts := countTasksBySkill(tasks)

	if len(counts) != 0 {
		t.Errorf("All empty skills should return empty map, got %d entries", len(counts))
	}
}

func TestGetPriorityColor(t *testing.T) {
	tests := []struct {
		priority domain.Priority
		expected string
	}{
		{domain.PriorityP0, "1"},          // Red
		{domain.PriorityP1, "3"},          // Yellow
		{domain.PriorityP2, "2"},          // Green
		{domain.Priority("P3"), "8"},      // Gray (default)
		{domain.Priority("invalid"), "8"}, // Gray (default)
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			result := getPriorityColor(tt.priority)
			if result != tt.expected {
				t.Errorf("getPriorityColor(%s) = %s, want %s", tt.priority, result, tt.expected)
			}
		})
	}
}

func TestRenderCount(t *testing.T) {
	// We can't test the exact lipgloss output, but we can test that it returns a non-empty string
	// and contains the expected count text
	tests := []struct {
		count    int
		expected string
	}{
		{0, "0 tasks"},
		{1, "1 tasks"},
		{5, "5 tasks"},
		{100, "100 tasks"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.count)), func(t *testing.T) {
			result := renderCount(tt.count)
			if result == "" {
				t.Error("renderCount returned empty string")
			}
			// Check if the result contains the expected text (ignoring ANSI codes)
			if !containsText(result, tt.expected) {
				t.Errorf("renderCount(%d) does not contain %q, got %q", tt.count, tt.expected, result)
			}
		})
	}
}

// Helper to check if string contains text (ignoring ANSI escape codes)
func containsText(s, text string) bool {
	// Simple check that just looks for the text substring
	// In a real scenario, we might want to strip ANSI codes first
	for i := 0; i <= len(s)-len(text); i++ {
		if s[i:i+len(text)] == text {
			return true
		}
	}
	return false
}
