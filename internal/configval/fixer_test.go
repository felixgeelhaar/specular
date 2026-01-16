package configval

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSuggestFixes_MissingField(t *testing.T) {
	content := map[string]interface{}{
		// Missing "providers" field
	}

	ctx := &ValidationContext{
		FilePath:   "providers.yaml",
		ConfigType: ConfigTypeProviders,
		Parsed:     content,
	}

	result := SuggestFixes(ctx)

	if len(result.Suggestions) == 0 {
		t.Error("SuggestFixes() expected at least one suggestion")
	}

	// Should suggest adding the missing providers field
	found := false
	for _, suggestion := range result.Suggestions {
		if strings.Contains(suggestion.Field, "providers") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should suggest fix for missing providers field")
	}
}

func TestSuggestFixes_InvalidEnumValue(t *testing.T) {
	content := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name":    "test",
				"type":    "invalid-type",
				"enabled": true,
			},
		},
	}

	ctx := &ValidationContext{
		FilePath:   "providers.yaml",
		ConfigType: ConfigTypeProviders,
		Parsed:     content,
	}

	result := SuggestFixes(ctx)

	// Should suggest valid enum values
	found := false
	for _, suggestion := range result.Suggestions {
		if strings.Contains(suggestion.Description, "one of") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should suggest valid enum values")
	}
}

func TestSuggestFixes_Improvements(t *testing.T) {
	content := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name":    "test",
				"type":    "api",
				"enabled": true,
				// Missing "models" mapping
			},
		},
		// Missing fallback configuration
	}

	ctx := &ValidationContext{
		FilePath:   "providers.yaml",
		ConfigType: ConfigTypeProviders,
		Parsed:     content,
	}

	result := SuggestFixes(ctx)

	// Should suggest adding model mappings
	foundModels := false
	for _, suggestion := range result.Suggestions {
		if strings.Contains(suggestion.Description, "model") {
			foundModels = true
			break
		}
	}
	if !foundModels {
		t.Error("Should suggest adding model mappings")
	}
}

func TestSuggestFixes_RouterImprovements(t *testing.T) {
	content := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name":    "test",
				"type":    "api",
				"enabled": true,
			},
		},
		// Missing context validation
	}

	ctx := &ValidationContext{
		FilePath:   "routing.yaml",
		ConfigType: ConfigTypeRouter,
		Parsed:     content,
	}

	result := SuggestFixes(ctx)

	// Should suggest enabling context validation
	found := false
	for _, suggestion := range result.Suggestions {
		if strings.Contains(suggestion.Description, "context validation") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should suggest enabling context validation")
	}
}

func TestGetDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		rule     FieldRule
		expected interface{}
	}{
		{
			name:     "string with enum",
			rule:     FieldRule{Type: "string", Enum: []string{"a", "b", "c"}},
			expected: "a",
		},
		{
			name:     "string without enum",
			rule:     FieldRule{Type: "string"},
			expected: "",
		},
		{
			name:     "number with min",
			rule:     FieldRule{Type: "number", Min: ptrFloat64(10)},
			expected: 10.0,
		},
		{
			name:     "number without min",
			rule:     FieldRule{Type: "number"},
			expected: 0,
		},
		{
			name:     "boolean",
			rule:     FieldRule{Type: "boolean"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDefaultValue(tt.rule)
			if got != tt.expected {
				t.Errorf("getDefaultValue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func ptrFloat64(f float64) *float64 {
	return &f
}

// toFloat64ForTest converts various numeric types to float64 for testing
func toFloat64ForTest(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case float32:
		return float64(val)
	default:
		return 0
	}
}

func TestExtractEnumValues(t *testing.T) {
	tests := []struct {
		message string
		want    []string
	}{
		{
			message: "value must be one of: cli, api, grpc, native",
			want:    []string{"cli", "api", "grpc", "native"},
		},
		{
			message: "some other error message",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			got := extractEnumValues(tt.message)
			if len(got) != len(tt.want) {
				t.Errorf("extractEnumValues() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractEnumValues()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSetNestedValue(t *testing.T) {
	data := make(map[string]interface{})

	// Test simple set
	err := setNestedValue(data, []string{"key"}, "value")
	if err != nil {
		t.Errorf("setNestedValue() error = %v", err)
	}
	if data["key"] != "value" {
		t.Errorf("data[key] = %v, want 'value'", data["key"])
	}

	// Test nested set
	data = make(map[string]interface{})
	err = setNestedValue(data, []string{"level1", "level2"}, "nested")
	if err != nil {
		t.Errorf("setNestedValue() error = %v", err)
	}
	level1, ok := data["level1"].(map[string]interface{})
	if !ok {
		t.Fatal("level1 should be a map")
	}
	if level1["level2"] != "nested" {
		t.Errorf("level1[level2] = %v, want 'nested'", level1["level2"])
	}

	// Test empty path
	err = setNestedValue(data, []string{}, "value")
	if err == nil {
		t.Error("setNestedValue() expected error for empty path")
	}
}

func TestApplyFixes(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "providers.yaml")

	// Create initial config
	initialContent := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name":    "test",
				"type":    "api",
				"enabled": true,
			},
		},
	}
	content, _ := yaml.Marshal(initialContent)
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := &ValidationContext{
		FilePath:   testFile,
		ConfigType: ConfigTypeProviders,
		Parsed:     initialContent,
	}

	suggestions := []FixSuggestion{
		{
			Field:       "strategy.budget.max_cost_per_day",
			Description: "Add budget limit",
			NewValue:    100.0,
			IsAutoFix:   true,
		},
	}

	result, err := ApplyFixes(ctx, suggestions, true)
	if err != nil {
		t.Fatalf("ApplyFixes() error = %v", err)
	}

	if len(result.Applied) != 1 {
		t.Errorf("Applied count = %d, want 1", len(result.Applied))
	}

	if !result.Modified {
		t.Error("Modified should be true")
	}

	// Verify file was updated
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	var updated map[string]interface{}
	if err := yaml.Unmarshal(updatedContent, &updated); err != nil {
		t.Fatalf("Failed to parse updated file: %v", err)
	}

	// Check that the value was added
	strategy, ok := updated["strategy"].(map[string]interface{})
	if !ok {
		t.Fatal("strategy should exist")
	}
	budget, ok := strategy["budget"].(map[string]interface{})
	if !ok {
		t.Fatal("budget should exist")
	}
	// YAML may unmarshal as int or float depending on value
	maxCost := toFloat64ForTest(budget["max_cost_per_day"])
	if maxCost != 100.0 {
		t.Errorf("max_cost_per_day = %v, want 100.0", budget["max_cost_per_day"])
	}
}

func TestApplyFixes_AutoOnly(t *testing.T) {
	ctx := &ValidationContext{
		FilePath:   "test.yaml",
		ConfigType: ConfigTypeProviders,
		Parsed:     map[string]interface{}{},
	}

	suggestions := []FixSuggestion{
		{
			Field:       "auto",
			Description: "Auto-fixable",
			IsAutoFix:   true,
		},
		{
			Field:       "manual",
			Description: "Manual only",
			IsAutoFix:   false,
		},
	}

	result, _ := ApplyFixes(ctx, suggestions, true)

	if len(result.Skipped) != 1 {
		t.Errorf("Skipped count = %d, want 1", len(result.Skipped))
	}
	if result.Skipped[0].Field != "manual" {
		t.Error("Manual fix should be skipped")
	}
}

func TestPrintFixSuggestions(t *testing.T) {
	result := &FixResult{
		FilePath:   "test.yaml",
		ConfigType: ConfigTypeProviders,
		Suggestions: []FixSuggestion{
			{
				Field:       "field1",
				Description: "Auto-fixable issue",
				IsAutoFix:   true,
			},
			{
				Field:       "field2",
				Description: "Manual issue",
				IsAutoFix:   false,
			},
		},
	}

	var buf bytes.Buffer
	PrintFixSuggestions(result, &buf)

	output := buf.String()

	if !strings.Contains(output, "Auto-fixable issue") {
		t.Error("Output should contain auto-fixable suggestion")
	}
	if !strings.Contains(output, "Manual issue") {
		t.Error("Output should contain manual suggestion")
	}
	if !strings.Contains(output, "1 auto-fixable") {
		t.Error("Output should show count of auto-fixable suggestions")
	}
	if !strings.Contains(output, "1 manual") {
		t.Error("Output should show count of manual suggestions")
	}
}

func TestPrintFixSuggestions_NoSuggestions(t *testing.T) {
	result := &FixResult{
		FilePath:   "test.yaml",
		ConfigType: ConfigTypeProviders,
	}

	var buf bytes.Buffer
	PrintFixSuggestions(result, &buf)

	if !strings.Contains(buf.String(), "No fixes needed") {
		t.Error("Output should indicate no fixes needed")
	}
}

func TestPrintFixResult(t *testing.T) {
	result := &FixResult{
		FilePath: "test.yaml",
		Applied: []FixSuggestion{
			{Description: "Applied fix"},
		},
		Skipped: []FixSuggestion{
			{Description: "Skipped fix"},
		},
		Modified: true,
	}

	var buf bytes.Buffer
	PrintFixResult(result, &buf)

	output := buf.String()

	if !strings.Contains(output, "Applied fix") {
		t.Error("Output should show applied fix")
	}
	if !strings.Contains(output, "Skipped fix") {
		t.Error("Output should show skipped fix")
	}
	if !strings.Contains(output, "saved") {
		t.Error("Output should indicate file was saved")
	}
}

func TestPrintFixResult_NoFixes(t *testing.T) {
	result := &FixResult{
		FilePath: "test.yaml",
	}

	var buf bytes.Buffer
	PrintFixResult(result, &buf)

	if !strings.Contains(buf.String(), "No fixes to apply") {
		t.Error("Output should indicate no fixes to apply")
	}
}

func TestSuggestProviderImprovements_WithFallback(t *testing.T) {
	// Config already has fallback configured
	content := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name":    "test",
				"type":    "api",
				"enabled": true,
				"models": map[string]string{
					"fast": "model-1",
				},
			},
		},
		"strategy": map[string]interface{}{
			"fallback": map[string]interface{}{
				"enabled": true,
			},
		},
	}

	ctx := &ValidationContext{
		FilePath:   "providers.yaml",
		ConfigType: ConfigTypeProviders,
		Parsed:     content,
	}

	result := SuggestFixes(ctx)

	// Should not suggest fallback since it's already configured
	for _, suggestion := range result.Suggestions {
		if strings.Contains(suggestion.Field, "fallback") {
			t.Error("Should not suggest fallback when already configured")
		}
	}
}
