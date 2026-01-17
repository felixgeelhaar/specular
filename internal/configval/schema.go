// Package configval provides configuration file validation and fix suggestions.
// It supports validating YAML and JSON configuration files against schemas
// and provides intelligent suggestions for fixing common issues.
package configval

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/specular/internal/validate"
)

// ConfigType represents the type of configuration file
type ConfigType string

const (
	// ConfigTypeRouter represents routing.yaml configuration
	ConfigTypeRouter ConfigType = "router"
	// ConfigTypeProviders represents providers.yaml configuration
	ConfigTypeProviders ConfigType = "providers"
	// ConfigTypeSpec represents spec.yaml configuration
	ConfigTypeSpec ConfigType = "spec"
	// ConfigTypePolicy represents policy.yaml configuration
	ConfigTypePolicy ConfigType = "policy"
	// ConfigTypeSLO represents slo.yaml configuration
	ConfigTypeSLO ConfigType = "slo"
	// ConfigTypeUnknown represents an unknown configuration type
	ConfigTypeUnknown ConfigType = "unknown"
)

// FieldRule defines a validation rule for a configuration field
type FieldRule struct {
	Path        string   // JSON path like "providers[*].name"
	Required    bool     // Whether the field is required
	Type        string   // Expected type: string, number, boolean, array, object
	Enum        []string // Allowed values for enum types
	Min         *float64 // Minimum value for numbers
	Max         *float64 // Maximum value for numbers
	Pattern     string   // Regex pattern for strings
	Description string   // Human-readable description
}

// ConfigSchema defines the schema for a configuration type
type ConfigSchema struct {
	Type        ConfigType
	Description string
	Rules       []FieldRule
}

// ValidationContext holds context for validation
type ValidationContext struct {
	FilePath   string
	ConfigType ConfigType
	Content    []byte
	Parsed     map[string]interface{}
}

// NewValidationContext creates a new validation context from a file path
func NewValidationContext(filePath string) (*ValidationContext, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var parsed map[string]interface{}
	if parseErr := yaml.Unmarshal(content, &parsed); parseErr != nil {
		return nil, fmt.Errorf("parse YAML: %w", parseErr)
	}

	configType := DetectConfigType(filePath, parsed)

	return &ValidationContext{
		FilePath:   filePath,
		ConfigType: configType,
		Content:    content,
		Parsed:     parsed,
	}, nil
}

// DetectConfigType attempts to detect the configuration type from file path and content
func DetectConfigType(filePath string, content map[string]interface{}) ConfigType {
	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	baseName = strings.ToLower(baseName)

	// Try to detect from filename
	switch {
	case strings.Contains(baseName, "routing") || strings.Contains(baseName, "router"):
		return ConfigTypeRouter
	case strings.Contains(baseName, "provider"):
		return ConfigTypeProviders
	case strings.Contains(baseName, "spec"):
		return ConfigTypeSpec
	case strings.Contains(baseName, "policy"):
		return ConfigTypePolicy
	case strings.Contains(baseName, "slo"):
		return ConfigTypeSLO
	}

	// Try to detect from content structure
	if _, hasProviders := content["providers"]; hasProviders {
		if _, hasStrategy := content["strategy"]; hasStrategy {
			return ConfigTypeProviders
		}
		// Could be router config if it has budget settings
		if _, hasBudget := content["budget_usd"]; hasBudget {
			return ConfigTypeRouter
		}
		return ConfigTypeProviders
	}

	if _, hasTasks := content["tasks"]; hasTasks {
		return ConfigTypeSpec
	}

	if _, hasObjectives := content["objectives"]; hasObjectives {
		return ConfigTypeSLO
	}

	if _, hasRules := content["rules"]; hasRules {
		return ConfigTypePolicy
	}

	return ConfigTypeUnknown
}

// GetSchema returns the schema for a configuration type
func GetSchema(configType ConfigType) *ConfigSchema {
	switch configType {
	case ConfigTypeProviders:
		return getProvidersSchema()
	case ConfigTypeRouter:
		return getRouterSchema()
	case ConfigTypeSpec:
		return getSpecSchema()
	case ConfigTypePolicy:
		return getPolicySchema()
	case ConfigTypeSLO:
		return getSLOSchema()
	default:
		return nil
	}
}

func getProvidersSchema() *ConfigSchema {
	zero := 0.0
	return &ConfigSchema{
		Type:        ConfigTypeProviders,
		Description: "Provider configuration for AI model access",
		Rules: []FieldRule{
			{Path: "providers", Required: true, Type: "array", Description: "List of provider configurations"},
			{Path: "providers[*].name", Required: true, Type: "string", Description: "Provider name"},
			{Path: "providers[*].type", Required: true, Type: "string", Enum: []string{"cli", "api", "grpc", "native"}, Description: "Provider type"},
			{Path: "providers[*].enabled", Required: false, Type: "boolean", Description: "Whether provider is enabled"},
			{Path: "providers[*].models", Required: false, Type: "object", Description: "Model mappings"},
			{Path: "strategy.budget.max_cost_per_day", Required: false, Type: "number", Min: &zero, Description: "Maximum daily cost"},
			{Path: "strategy.budget.max_cost_per_request", Required: false, Type: "number", Min: &zero, Description: "Maximum cost per request"},
			{Path: "strategy.performance.max_latency_ms", Required: false, Type: "number", Min: &zero, Description: "Maximum latency in milliseconds"},
			{Path: "strategy.fallback.enabled", Required: false, Type: "boolean", Description: "Enable fallback behavior"},
			{Path: "strategy.fallback.max_retries", Required: false, Type: "number", Min: &zero, Description: "Maximum retry attempts"},
		},
	}
}

func getRouterSchema() *ConfigSchema {
	zero := 0.0
	return &ConfigSchema{
		Type:        ConfigTypeRouter,
		Description: "Router configuration for model selection",
		Rules: []FieldRule{
			{Path: "providers", Required: true, Type: "array", Description: "List of provider configurations"},
			{Path: "budget_usd", Required: false, Type: "number", Min: &zero, Description: "Budget in USD"},
			{Path: "max_latency_ms", Required: false, Type: "number", Min: &zero, Description: "Maximum latency"},
			{Path: "prefer_cheap", Required: false, Type: "boolean", Description: "Prefer cheaper models"},
			{Path: "fallback_model", Required: false, Type: "string", Description: "Fallback model name"},
			{Path: "enable_fallback", Required: false, Type: "boolean", Description: "Enable fallback"},
			{Path: "max_retries", Required: false, Type: "number", Min: &zero, Description: "Maximum retries"},
		},
	}
}

func getSpecSchema() *ConfigSchema {
	return &ConfigSchema{
		Type:        ConfigTypeSpec,
		Description: "Specification configuration for task definitions",
		Rules: []FieldRule{
			{Path: "name", Required: false, Type: "string", Description: "Spec name"},
			{Path: "description", Required: false, Type: "string", Description: "Spec description"},
			{Path: "tasks", Required: true, Type: "array", Description: "List of tasks"},
			{Path: "tasks[*].id", Required: true, Type: "string", Description: "Task identifier"},
			{Path: "tasks[*].description", Required: false, Type: "string", Description: "Task description"},
		},
	}
}

func getPolicySchema() *ConfigSchema {
	return &ConfigSchema{
		Type:        ConfigTypePolicy,
		Description: "Policy configuration for governance rules",
		Rules: []FieldRule{
			{Path: "version", Required: false, Type: "string", Description: "Policy version"},
			{Path: "rules", Required: true, Type: "array", Description: "List of policy rules"},
			{Path: "rules[*].id", Required: true, Type: "string", Description: "Rule identifier"},
			{Path: "rules[*].condition", Required: true, Type: "string", Description: "Rule condition"},
		},
	}
}

func getSLOSchema() *ConfigSchema {
	zero := 0.0
	hundred := 100.0
	return &ConfigSchema{
		Type:        ConfigTypeSLO,
		Description: "SLO configuration for service level objectives",
		Rules: []FieldRule{
			{Path: "objectives", Required: true, Type: "array", Description: "List of objectives"},
			{Path: "objectives[*].name", Required: true, Type: "string", Description: "Objective name"},
			{Path: "objectives[*].target", Required: true, Type: "number", Min: &zero, Max: &hundred, Description: "Target percentage (0-100)"},
		},
	}
}

// ValidateConfig validates a configuration file
func ValidateConfig(ctx *ValidationContext) *validate.ValidationResult {
	result := validate.NewValidationResult()

	// Check if we could detect the config type
	if ctx.ConfigType == ConfigTypeUnknown {
		result.AddWarning("", "", "Could not detect configuration type, skipping schema validation")
		return result
	}

	// Get schema for this config type
	schema := GetSchema(ctx.ConfigType)
	if schema == nil {
		result.AddWarning("", "", fmt.Sprintf("No schema defined for config type: %s", ctx.ConfigType))
		return result
	}

	// Validate against schema rules
	for _, rule := range schema.Rules {
		validateField(ctx.Parsed, rule, result)
	}

	return result
}

// validateField validates a single field against a rule
func validateField(data map[string]interface{}, rule FieldRule, result *validate.ValidationResult) {
	// Parse the path to get nested values
	values, exists := getFieldValues(data, rule.Path)

	if rule.Required && !exists {
		result.AddErrorWithCode(rule.Path, "", fmt.Sprintf("required field is missing: %s", rule.Description), "E001")
		return
	}

	if !exists {
		return
	}

	// Validate each value found at the path
	for _, value := range values {
		validateValue(rule, value, result)
	}
}

// getFieldValues extracts values from a nested data structure based on a path
func getFieldValues(data map[string]interface{}, path string) ([]interface{}, bool) {
	parts := strings.Split(path, ".")
	return getFieldValuesRecursive(data, parts)
}

func getFieldValuesRecursive(data interface{}, parts []string) ([]interface{}, bool) {
	if len(parts) == 0 {
		return []interface{}{data}, true
	}

	part := parts[0]
	remaining := parts[1:]

	// Handle array notation like "providers[*]"
	if strings.HasSuffix(part, "[*]") {
		fieldName := strings.TrimSuffix(part, "[*]")
		mapData, isMap := data.(map[string]interface{})
		if !isMap {
			return nil, false
		}

		arr, isArr := mapData[fieldName].([]interface{})
		if !isArr {
			return nil, false
		}

		var results []interface{}
		for _, item := range arr {
			if len(remaining) == 0 {
				results = append(results, item)
			} else {
				if itemMap, isItemMap := item.(map[string]interface{}); isItemMap {
					vals, found := getFieldValuesRecursive(itemMap, remaining)
					if found {
						results = append(results, vals...)
					}
				}
			}
		}
		return results, len(results) > 0
	}

	// Handle regular field access
	mapData, ok := data.(map[string]interface{})
	if !ok {
		return nil, false
	}

	value, exists := mapData[part]
	if !exists {
		return nil, false
	}

	return getFieldValuesRecursive(value, remaining)
}

// validateValue validates a single value against a rule
func validateValue(rule FieldRule, value interface{}, result *validate.ValidationResult) {
	// Type validation
	if !validateType(value, rule.Type) {
		result.AddError(rule.Path, fmt.Sprintf("%v", value),
			fmt.Sprintf("expected type %s, got %T", rule.Type, value))
		return
	}

	// Enum validation
	if len(rule.Enum) > 0 {
		validateEnumValue(rule, value, result)
	}

	// Number range validation
	if rule.Type == "number" {
		validateNumberRange(rule, value, result)
	}

	// Pattern validation
	if rule.Pattern != "" {
		validatePattern(rule, value, result)
	}
}

// validateEnumValue validates that a value is one of the allowed enum values
func validateEnumValue(rule FieldRule, value interface{}, result *validate.ValidationResult) {
	strVal, ok := value.(string)
	if !ok {
		result.AddError(rule.Path, fmt.Sprintf("%v", value), "enum value must be a string")
		return
	}

	for _, allowed := range rule.Enum {
		if strVal == allowed {
			return
		}
	}
	result.AddError(rule.Path, strVal,
		fmt.Sprintf("value must be one of: %s", strings.Join(rule.Enum, ", ")))
}

// validateNumberRange validates that a number is within the allowed range
func validateNumberRange(rule FieldRule, value interface{}, result *validate.ValidationResult) {
	numVal := toFloat64(value)
	if rule.Min != nil && numVal < *rule.Min {
		result.AddError(rule.Path, fmt.Sprintf("%v", value),
			fmt.Sprintf("value must be at least %v", *rule.Min))
	}
	if rule.Max != nil && numVal > *rule.Max {
		result.AddError(rule.Path, fmt.Sprintf("%v", value),
			fmt.Sprintf("value must be at most %v", *rule.Max))
	}
}

// validatePattern validates that a string matches the required pattern
func validatePattern(rule FieldRule, value interface{}, result *validate.ValidationResult) {
	strVal, ok := value.(string)
	if !ok {
		return
	}
	matched, err := regexp.MatchString(rule.Pattern, strVal)
	if err != nil {
		result.AddWarning(rule.Path, strVal, fmt.Sprintf("pattern validation error: %v", err))
	} else if !matched {
		result.AddError(rule.Path, strVal,
			fmt.Sprintf("value does not match pattern: %s", rule.Pattern))
	}
}

// validateType checks if a value matches the expected type
func validateType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		return isNumber(value)
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return true // Unknown type, accept anything
	}
}

// isNumber checks if a value is a numeric type
func isNumber(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

// toFloat64 converts a numeric value to float64
func toFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case float32:
		return float64(v)
	default:
		return 0
	}
}

// ValidateFile validates a configuration file and prints results
func ValidateFile(filePath string, w io.Writer) error {
	ctx, err := NewValidationContext(filePath)
	if err != nil {
		return err
	}

	result := ValidateConfig(ctx)

	// Print results
	errors, warnings, infos := result.Count()

	if !result.HasIssues() {
		_, _ = fmt.Fprintf(w, "✓ %s: Valid %s configuration\n", filePath, ctx.ConfigType)
		return nil
	}

	_, _ = fmt.Fprintf(w, "Validating %s (%s):\n", filePath, ctx.ConfigType)

	for _, issue := range result.Issues {
		_, _ = fmt.Fprintf(w, "  %s %s\n", issue.Severity.Symbol(), issue.Error())
	}

	_, _ = fmt.Fprintf(w, "\nSummary: %d errors, %d warnings, %d infos\n", errors, warnings, infos)

	if result.HasErrors() {
		return result.Error()
	}

	return nil
}
