package configval

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/felixgeelhaar/specular/internal/validate"
	"gopkg.in/yaml.v3"
)

// FixSuggestion represents a suggested fix for a configuration issue
type FixSuggestion struct {
	Field       string
	Description string
	OldValue    interface{}
	NewValue    interface{}
	IsAutoFix   bool // Whether this can be automatically applied
}

// FixResult contains the results of attempting to fix configuration issues
type FixResult struct {
	FilePath    string
	ConfigType  ConfigType
	Suggestions []FixSuggestion
	Applied     []FixSuggestion
	Skipped     []FixSuggestion
	Modified    bool
}

// SuggestFixes analyzes a configuration and suggests fixes for issues
func SuggestFixes(ctx *ValidationContext) *FixResult {
	result := &FixResult{
		FilePath:    ctx.FilePath,
		ConfigType:  ctx.ConfigType,
		Suggestions: make([]FixSuggestion, 0),
	}

	// Run validation to find issues
	validationResult := ValidateConfig(ctx)

	// Generate fix suggestions for each issue
	for _, issue := range validationResult.Issues {
		suggestions := generateFixSuggestions(ctx, issue)
		result.Suggestions = append(result.Suggestions, suggestions...)
	}

	// Add additional improvement suggestions
	improvements := suggestImprovements(ctx)
	result.Suggestions = append(result.Suggestions, improvements...)

	return result
}

// generateFixSuggestions generates fix suggestions for a validation issue
func generateFixSuggestions(ctx *ValidationContext, issue validate.ValidationIssue) []FixSuggestion {
	var suggestions []FixSuggestion

	switch issue.Code {
	case "E001": // Missing required field
		suggestion := suggestMissingFieldFix(ctx, issue.Field)
		if suggestion != nil {
			suggestions = append(suggestions, *suggestion)
		}
	default:
		// Generate generic suggestions based on the issue
		suggestions = append(suggestions, generateGenericSuggestions(ctx, issue)...)
	}

	return suggestions
}

// suggestMissingFieldFix suggests a fix for a missing required field
func suggestMissingFieldFix(ctx *ValidationContext, fieldPath string) *FixSuggestion {
	// Provide default values based on field type
	schema := GetSchema(ctx.ConfigType)
	if schema == nil {
		return nil
	}

	for _, rule := range schema.Rules {
		if rule.Path == fieldPath {
			defaultValue := getDefaultValue(rule)
			return &FixSuggestion{
				Field:       fieldPath,
				Description: fmt.Sprintf("Add missing field: %s (%s)", fieldPath, rule.Description),
				OldValue:    nil,
				NewValue:    defaultValue,
				IsAutoFix:   true,
			}
		}
	}

	return nil
}

// getDefaultValue returns a sensible default value for a field rule
func getDefaultValue(rule FieldRule) interface{} {
	switch rule.Type {
	case "string":
		if len(rule.Enum) > 0 {
			return rule.Enum[0]
		}
		return ""
	case "number":
		if rule.Min != nil {
			return *rule.Min
		}
		return 0
	case "boolean":
		return false
	case "array":
		return []interface{}{}
	case "object":
		return map[string]interface{}{}
	default:
		return nil
	}
}

// generateGenericSuggestions generates suggestions for issues without specific codes
func generateGenericSuggestions(ctx *ValidationContext, issue validate.ValidationIssue) []FixSuggestion {
	var suggestions []FixSuggestion

	// Check for common patterns in error messages
	if strings.Contains(issue.Message, "must be one of:") {
		// Enum value suggestion
		validValues := extractEnumValues(issue.Message)
		if len(validValues) > 0 {
			suggestions = append(suggestions, FixSuggestion{
				Field:       issue.Field,
				Description: fmt.Sprintf("Change value to one of: %s", strings.Join(validValues, ", ")),
				OldValue:    issue.Value,
				NewValue:    validValues[0],
				IsAutoFix:   true,
			})
		}
	}

	if strings.Contains(issue.Message, "must be at least") {
		// Minimum value suggestion
		suggestions = append(suggestions, FixSuggestion{
			Field:       issue.Field,
			Description: "Increase value to meet minimum requirement",
			OldValue:    issue.Value,
			NewValue:    0, // Will need to parse the actual minimum
			IsAutoFix:   false,
		})
	}

	return suggestions
}

// extractEnumValues extracts valid values from an error message
func extractEnumValues(message string) []string {
	// Look for pattern "must be one of: a, b, c"
	prefix := "must be one of: "
	idx := strings.Index(message, prefix)
	if idx == -1 {
		return nil
	}

	valueStr := message[idx+len(prefix):]
	values := strings.Split(valueStr, ", ")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	return values
}

// suggestImprovements suggests improvements beyond fixing errors
func suggestImprovements(ctx *ValidationContext) []FixSuggestion {
	var suggestions []FixSuggestion

	switch ctx.ConfigType {
	case ConfigTypeProviders:
		suggestions = append(suggestions, suggestProviderImprovements(ctx)...)
	case ConfigTypeRouter:
		suggestions = append(suggestions, suggestRouterImprovements(ctx)...)
	}

	return suggestions
}

// suggestProviderImprovements suggests improvements for provider configuration
func suggestProviderImprovements(ctx *ValidationContext) []FixSuggestion {
	var suggestions []FixSuggestion

	// Check if fallback is configured
	strategy, hasStrategy := ctx.Parsed["strategy"].(map[string]interface{})
	if hasStrategy {
		fallback, hasFallback := strategy["fallback"].(map[string]interface{})
		if !hasFallback || fallback == nil {
			suggestions = append(suggestions, FixSuggestion{
				Field:       "strategy.fallback",
				Description: "Consider enabling fallback for resilience",
				OldValue:    nil,
				NewValue: map[string]interface{}{
					"enabled":        true,
					"max_retries":    3,
					"retry_delay_ms": 1000,
				},
				IsAutoFix: false,
			})
		}
	}

	// Check providers for missing model mappings
	providers, hasProviders := ctx.Parsed["providers"].([]interface{})
	if hasProviders {
		for i, p := range providers {
			provider, ok := p.(map[string]interface{})
			if !ok {
				continue
			}

			_, hasModels := provider["models"]
			if !hasModels {
				name := provider["name"]
				suggestions = append(suggestions, FixSuggestion{
					Field:       fmt.Sprintf("providers[%d].models", i),
					Description: fmt.Sprintf("Add model mappings for provider '%v' to enable model hints", name),
					OldValue:    nil,
					NewValue: map[string]string{
						"fast":    "default-fast-model",
						"codegen": "default-codegen-model",
					},
					IsAutoFix: false,
				})
			}
		}
	}

	return suggestions
}

// suggestRouterImprovements suggests improvements for router configuration
func suggestRouterImprovements(ctx *ValidationContext) []FixSuggestion {
	var suggestions []FixSuggestion

	// Check if context validation is enabled
	_, hasContextValidation := ctx.Parsed["enable_context_validation"]
	if !hasContextValidation {
		suggestions = append(suggestions, FixSuggestion{
			Field:       "enable_context_validation",
			Description: "Consider enabling context validation to prevent token limit issues",
			OldValue:    nil,
			NewValue:    true,
			IsAutoFix:   false,
		})
	}

	return suggestions
}

// ApplyFixes applies suggested fixes to a configuration file
func ApplyFixes(ctx *ValidationContext, suggestions []FixSuggestion, autoOnly bool) (*FixResult, error) {
	result := &FixResult{
		FilePath:   ctx.FilePath,
		ConfigType: ctx.ConfigType,
		Applied:    make([]FixSuggestion, 0),
		Skipped:    make([]FixSuggestion, 0),
	}

	modified := ctx.Parsed

	for _, suggestion := range suggestions {
		if autoOnly && !suggestion.IsAutoFix {
			result.Skipped = append(result.Skipped, suggestion)
			continue
		}

		// Apply the fix
		if err := applyFix(modified, suggestion); err != nil {
			result.Skipped = append(result.Skipped, suggestion)
			continue
		}

		result.Applied = append(result.Applied, suggestion)
		result.Modified = true
	}

	// Save the modified configuration if changes were made
	if result.Modified {
		if err := saveConfig(ctx.FilePath, modified); err != nil {
			return result, fmt.Errorf("save config: %w", err)
		}
	}

	return result, nil
}

// applyFix applies a single fix to the configuration
func applyFix(data map[string]interface{}, suggestion FixSuggestion) error {
	parts := strings.Split(suggestion.Field, ".")
	return setNestedValue(data, parts, suggestion.NewValue)
}

// setNestedValue sets a value in a nested map structure
func setNestedValue(data map[string]interface{}, parts []string, value interface{}) error {
	if len(parts) == 0 {
		return fmt.Errorf("empty path")
	}

	if len(parts) == 1 {
		// Handle array index notation
		part := parts[0]
		if idx := strings.Index(part, "["); idx != -1 {
			// Array notation like "providers[0]"
			fieldName := part[:idx]
			// For now, skip array modifications
			_ = fieldName
			return fmt.Errorf("array modification not supported")
		}
		data[part] = value
		return nil
	}

	// Navigate to the parent
	part := parts[0]
	remaining := parts[1:]

	// Create intermediate maps if needed
	if _, exists := data[part]; !exists {
		data[part] = make(map[string]interface{})
	}

	child, ok := data[part].(map[string]interface{})
	if !ok {
		return fmt.Errorf("cannot navigate through non-map at %s", part)
	}

	return setNestedValue(child, remaining, value)
}

// saveConfig saves the modified configuration to a file
func saveConfig(filePath string, data map[string]interface{}) error {
	content, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal YAML: %w", err)
	}

	if err := os.WriteFile(filePath, content, 0600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// PrintFixSuggestions prints fix suggestions to a writer
func PrintFixSuggestions(result *FixResult, w io.Writer) {
	if len(result.Suggestions) == 0 {
		fmt.Fprintf(w, "✓ No fixes needed for %s\n", result.FilePath)
		return
	}

	fmt.Fprintf(w, "Fix suggestions for %s (%s):\n\n", result.FilePath, result.ConfigType)

	autoFixCount := 0
	manualCount := 0

	for i, suggestion := range result.Suggestions {
		prefix := "○"
		if suggestion.IsAutoFix {
			prefix = "●"
			autoFixCount++
		} else {
			manualCount++
		}

		fmt.Fprintf(w, "%d. %s %s\n", i+1, prefix, suggestion.Description)
		fmt.Fprintf(w, "   Field: %s\n", suggestion.Field)
		if suggestion.OldValue != nil {
			fmt.Fprintf(w, "   Current: %v\n", suggestion.OldValue)
		}
		if suggestion.NewValue != nil {
			fmt.Fprintf(w, "   Suggested: %v\n", suggestion.NewValue)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "Summary: %d auto-fixable (●), %d manual (○)\n", autoFixCount, manualCount)
	if autoFixCount > 0 {
		fmt.Fprintf(w, "\nRun 'specular config fix %s' to apply auto-fixable suggestions\n", result.FilePath)
	}
}

// PrintFixResult prints the results of applying fixes
func PrintFixResult(result *FixResult, w io.Writer) {
	if len(result.Applied) == 0 && len(result.Skipped) == 0 {
		fmt.Fprintf(w, "✓ No fixes to apply for %s\n", result.FilePath)
		return
	}

	fmt.Fprintf(w, "Fix results for %s:\n\n", result.FilePath)

	if len(result.Applied) > 0 {
		fmt.Fprintln(w, "Applied fixes:")
		for _, fix := range result.Applied {
			fmt.Fprintf(w, "  ✓ %s\n", fix.Description)
		}
		fmt.Fprintln(w)
	}

	if len(result.Skipped) > 0 {
		fmt.Fprintln(w, "Skipped (manual review required):")
		for _, fix := range result.Skipped {
			fmt.Fprintf(w, "  ○ %s\n", fix.Description)
		}
		fmt.Fprintln(w)
	}

	if result.Modified {
		fmt.Fprintf(w, "✓ Configuration saved to %s\n", result.FilePath)
	}
}
