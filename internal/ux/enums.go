package ux

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// EnumValue represents a single value in an enum definition
type EnumValue struct {
	Value       string
	Label       string
	Description string
}

// EnumDef defines an enum with its possible values
type EnumDef struct {
	Name        string
	Description string
	Values      []EnumValue
}

// String returns a formatted string representation of the enum value
func (v EnumValue) String() string {
	if v.Label != "" {
		return fmt.Sprintf("%s (%s)", v.Value, v.Label)
	}
	return v.Value
}

// GetValue returns the underlying value
func (v EnumValue) GetValue() string {
	return v.Value
}

// PriorityLevels defines the priority levels for tasks
var PriorityLevels = EnumDef{
	Name:        "Priority",
	Description: "Task priority level",
	Values: []EnumValue{
		{Value: "P0", Label: "Critical", Description: "Must be addressed immediately, blocks other work"},
		{Value: "P1", Label: "High", Description: "Important task, should be completed soon"},
		{Value: "P2", Label: "Medium", Description: "Standard priority, complete in normal course"},
		{Value: "P3", Label: "Low", Description: "Nice to have, complete when time permits"},
	},
}

// ModelHints defines the model hint types for routing
var ModelHints = EnumDef{
	Name:        "Model Hint",
	Description: "Hint for model selection during routing",
	Values: []EnumValue{
		{Value: "fast", Label: "Fast", Description: "Optimize for speed over quality"},
		{Value: "codegen", Label: "Code Generation", Description: "Optimize for code generation tasks"},
		{Value: "agentic", Label: "Agentic", Description: "Optimize for multi-step agentic tasks"},
		{Value: "long-context", Label: "Long Context", Description: "Optimize for long context windows"},
		{Value: "reasoning", Label: "Reasoning", Description: "Optimize for complex reasoning tasks"},
	},
}

// GovernanceLevels defines the governance levels for compliance
var GovernanceLevels = EnumDef{
	Name:        "Governance Level",
	Description: "Governance and compliance level",
	Values: []EnumValue{
		{Value: "L1", Label: "Basic", Description: "Basic governance, minimal oversight"},
		{Value: "L2", Label: "Standard", Description: "Standard governance with audit trails"},
		{Value: "L3", Label: "Enhanced", Description: "Enhanced governance with approval workflows"},
		{Value: "L4", Label: "Strict", Description: "Strict governance with mandatory reviews"},
	},
}

// SkillTypes defines the available skill types
var SkillTypes = EnumDef{
	Name:        "Skill",
	Description: "Developer skill category",
	Values: []EnumValue{
		{Value: "backend", Label: "Backend", Description: "Server-side development"},
		{Value: "frontend", Label: "Frontend", Description: "Client-side development"},
		{Value: "fullstack", Label: "Full Stack", Description: "End-to-end development"},
		{Value: "devops", Label: "DevOps", Description: "Infrastructure and deployment"},
		{Value: "data", Label: "Data", Description: "Data engineering and analytics"},
		{Value: "ml", Label: "Machine Learning", Description: "ML model development"},
		{Value: "security", Label: "Security", Description: "Security engineering"},
		{Value: "mobile", Label: "Mobile", Description: "Mobile app development"},
	},
}

// OutputFormats defines the available output formats
var OutputFormats = EnumDef{
	Name:        "Format",
	Description: "Output format",
	Values: []EnumValue{
		{Value: "json", Label: "JSON", Description: "JSON format (machine-readable)"},
		{Value: "yaml", Label: "YAML", Description: "YAML format (human-friendly)"},
		{Value: "table", Label: "Table", Description: "Tabular format for terminal"},
		{Value: "markdown", Label: "Markdown", Description: "Markdown format for documentation"},
	},
}

// EvalScenarios defines the available evaluation scenarios
var EvalScenarios = EnumDef{
	Name:        "Scenario",
	Description: "Evaluation scenario type",
	Values: []EnumValue{
		{Value: "smoke", Label: "Smoke Test", Description: "Basic health checks"},
		{Value: "integration", Label: "Integration", Description: "Full integration tests"},
		{Value: "security", Label: "Security", Description: "Security scan and policy check"},
		{Value: "performance", Label: "Performance", Description: "Performance benchmarks"},
	},
}

// GetValues returns all value strings from the enum
func (e EnumDef) GetValues() []string {
	values := make([]string, len(e.Values))
	for i, v := range e.Values {
		values[i] = v.Value
	}
	return values
}

// GetLabels returns all label strings from the enum
func (e EnumDef) GetLabels() []string {
	labels := make([]string, len(e.Values))
	for i, v := range e.Values {
		if v.Label != "" {
			labels[i] = v.Label
		} else {
			labels[i] = v.Value
		}
	}
	return labels
}

// FindByValue finds an enum value by its value string
func (e EnumDef) FindByValue(value string) (EnumValue, bool) {
	for _, v := range e.Values {
		if v.Value == value {
			return v, true
		}
	}
	return EnumValue{}, false
}

// FindByLabel finds an enum value by its label string
func (e EnumDef) FindByLabel(label string) (EnumValue, bool) {
	for _, v := range e.Values {
		if v.Label == label {
			return v, true
		}
	}
	return EnumValue{}, false
}

// IsValid checks if a value is valid for this enum
func (e EnumDef) IsValid(value string) bool {
	_, found := e.FindByValue(value)
	return found
}

// Validate validates a value against the enum and returns an error if invalid
func (e EnumDef) Validate(value string) error {
	if !e.IsValid(value) {
		return fmt.Errorf("invalid %s: %q (valid options: %s)",
			strings.ToLower(e.Name), value, strings.Join(e.GetValues(), ", "))
	}
	return nil
}

// PromptForEnum prompts the user to select a value from the enum
func PromptForEnum(def EnumDef) (string, error) {
	// Build options for the select
	options := make([]huh.Option[string], len(def.Values))
	for i, v := range def.Values {
		label := v.Value
		if v.Label != "" {
			label = fmt.Sprintf("%s - %s", v.Value, v.Label)
		}
		if v.Description != "" {
			label = fmt.Sprintf("%s (%s)", label, v.Description)
		}
		options[i] = huh.NewOption(label, v.Value)
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(def.Name).
				Description(def.Description).
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("failed to prompt for %s: %w", def.Name, err)
	}

	return selected, nil
}

// PromptForEnumWithDefault prompts the user with a default value pre-selected
func PromptForEnumWithDefault(def EnumDef, defaultValue string) (string, error) {
	// Validate default value
	if defaultValue != "" && !def.IsValid(defaultValue) {
		return "", fmt.Errorf("invalid default value: %q", defaultValue)
	}

	// Build options for the select
	options := make([]huh.Option[string], len(def.Values))
	for i, v := range def.Values {
		label := v.Value
		if v.Label != "" {
			label = fmt.Sprintf("%s - %s", v.Value, v.Label)
		}
		if v.Description != "" {
			label = fmt.Sprintf("%s (%s)", label, v.Description)
		}
		options[i] = huh.NewOption(label, v.Value)
	}

	selected := defaultValue
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(def.Name).
				Description(def.Description).
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("failed to prompt for %s: %w", def.Name, err)
	}

	return selected, nil
}

// PromptForMultiEnum prompts the user to select multiple values from the enum
func PromptForMultiEnum(def EnumDef) ([]string, error) {
	// Build options for the multi-select
	options := make([]huh.Option[string], len(def.Values))
	for i, v := range def.Values {
		label := v.Value
		if v.Label != "" {
			label = fmt.Sprintf("%s - %s", v.Value, v.Label)
		}
		if v.Description != "" {
			label = fmt.Sprintf("%s (%s)", label, v.Description)
		}
		options[i] = huh.NewOption(label, v.Value)
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(def.Name).
				Description(def.Description).
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("failed to prompt for %s: %w", def.Name, err)
	}

	return selected, nil
}

// FormatEnumHelp returns a formatted help string for the enum
func FormatEnumHelp(def EnumDef) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s options:\n", def.Name))
	for _, v := range def.Values {
		sb.WriteString(fmt.Sprintf("  %s", v.Value))
		if v.Label != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", v.Label))
		}
		if v.Description != "" {
			sb.WriteString(fmt.Sprintf(": %s", v.Description))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
