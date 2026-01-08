package ux

import (
	"strings"
	"testing"
)

func TestEnumValue_String(t *testing.T) {
	tests := []struct {
		name  string
		value EnumValue
		want  string
	}{
		{
			name:  "with label",
			value: EnumValue{Value: "P0", Label: "Critical"},
			want:  "P0 (Critical)",
		},
		{
			name:  "without label",
			value: EnumValue{Value: "fast"},
			want:  "fast",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.value.String()
			if got != tt.want {
				t.Errorf("EnumValue.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnumValue_GetValue(t *testing.T) {
	value := EnumValue{Value: "P1", Label: "High"}
	if got := value.GetValue(); got != "P1" {
		t.Errorf("EnumValue.GetValue() = %q, want %q", got, "P1")
	}
}

func TestEnumDef_GetValues(t *testing.T) {
	def := EnumDef{
		Name: "Test",
		Values: []EnumValue{
			{Value: "a"},
			{Value: "b"},
			{Value: "c"},
		},
	}

	got := def.GetValues()
	want := []string{"a", "b", "c"}

	if len(got) != len(want) {
		t.Errorf("GetValues() length = %d, want %d", len(got), len(want))
	}

	for i, v := range got {
		if v != want[i] {
			t.Errorf("GetValues()[%d] = %q, want %q", i, v, want[i])
		}
	}
}

func TestEnumDef_GetLabels(t *testing.T) {
	def := EnumDef{
		Name: "Test",
		Values: []EnumValue{
			{Value: "a", Label: "Alpha"},
			{Value: "b"}, // No label
			{Value: "c", Label: "Charlie"},
		},
	}

	got := def.GetLabels()
	want := []string{"Alpha", "b", "Charlie"}

	if len(got) != len(want) {
		t.Errorf("GetLabels() length = %d, want %d", len(got), len(want))
	}

	for i, v := range got {
		if v != want[i] {
			t.Errorf("GetLabels()[%d] = %q, want %q", i, v, want[i])
		}
	}
}

func TestEnumDef_FindByValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantOK  bool
		wantVal string
	}{
		{"found", "P0", true, "P0"},
		{"not found", "P9", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := PriorityLevels.FindByValue(tt.value)
			if ok != tt.wantOK {
				t.Errorf("FindByValue() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got.Value != tt.wantVal {
				t.Errorf("FindByValue() value = %q, want %q", got.Value, tt.wantVal)
			}
		})
	}
}

func TestEnumDef_FindByLabel(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		wantOK  bool
		wantVal string
	}{
		{"found", "Critical", true, "P0"},
		{"not found", "Unknown", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := PriorityLevels.FindByLabel(tt.label)
			if ok != tt.wantOK {
				t.Errorf("FindByLabel() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got.Value != tt.wantVal {
				t.Errorf("FindByLabel() value = %q, want %q", got.Value, tt.wantVal)
			}
		})
	}
}

func TestEnumDef_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid P0", "P0", true},
		{"valid P1", "P1", true},
		{"valid P2", "P2", true},
		{"valid P3", "P3", true},
		{"invalid", "P9", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PriorityLevels.IsValid(tt.value)
			if got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestEnumDef_Validate(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "P0", false},
		{"invalid", "P9", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PriorityLevels.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestPriorityLevels(t *testing.T) {
	if PriorityLevels.Name != "Priority" {
		t.Errorf("PriorityLevels.Name = %q, want %q", PriorityLevels.Name, "Priority")
	}

	values := PriorityLevels.GetValues()
	expected := []string{"P0", "P1", "P2", "P3"}
	if len(values) != len(expected) {
		t.Errorf("PriorityLevels has %d values, want %d", len(values), len(expected))
	}

	for i, v := range expected {
		if values[i] != v {
			t.Errorf("PriorityLevels[%d] = %q, want %q", i, values[i], v)
		}
	}
}

func TestModelHints(t *testing.T) {
	if ModelHints.Name != "Model Hint" {
		t.Errorf("ModelHints.Name = %q, want %q", ModelHints.Name, "Model Hint")
	}

	expected := []string{"fast", "codegen", "agentic", "long-context", "reasoning"}
	values := ModelHints.GetValues()
	if len(values) != len(expected) {
		t.Errorf("ModelHints has %d values, want %d", len(values), len(expected))
	}
}

func TestGovernanceLevels(t *testing.T) {
	if GovernanceLevels.Name != "Governance Level" {
		t.Errorf("GovernanceLevels.Name = %q, want %q", GovernanceLevels.Name, "Governance Level")
	}

	expected := []string{"L1", "L2", "L3", "L4"}
	values := GovernanceLevels.GetValues()
	if len(values) != len(expected) {
		t.Errorf("GovernanceLevels has %d values, want %d", len(values), len(expected))
	}
}

func TestSkillTypes(t *testing.T) {
	if SkillTypes.Name != "Skill" {
		t.Errorf("SkillTypes.Name = %q, want %q", SkillTypes.Name, "Skill")
	}

	// Check that backend is valid
	if !SkillTypes.IsValid("backend") {
		t.Error("SkillTypes should have 'backend' as valid value")
	}

	// Check expected count
	if len(SkillTypes.Values) != 8 {
		t.Errorf("SkillTypes has %d values, want %d", len(SkillTypes.Values), 8)
	}
}

func TestOutputFormats(t *testing.T) {
	if OutputFormats.Name != "Format" {
		t.Errorf("OutputFormats.Name = %q, want %q", OutputFormats.Name, "Format")
	}

	expected := []string{"json", "yaml", "table", "markdown"}
	values := OutputFormats.GetValues()
	if len(values) != len(expected) {
		t.Errorf("OutputFormats has %d values, want %d", len(values), len(expected))
	}
}

func TestEvalScenarios(t *testing.T) {
	if EvalScenarios.Name != "Scenario" {
		t.Errorf("EvalScenarios.Name = %q, want %q", EvalScenarios.Name, "Scenario")
	}

	expected := []string{"smoke", "integration", "security", "performance"}
	values := EvalScenarios.GetValues()
	if len(values) != len(expected) {
		t.Errorf("EvalScenarios has %d values, want %d", len(values), len(expected))
	}
}

func TestFormatEnumHelp(t *testing.T) {
	def := EnumDef{
		Name: "Test",
		Values: []EnumValue{
			{Value: "a", Label: "Alpha", Description: "First option"},
			{Value: "b", Description: "Second option"},
			{Value: "c"},
		},
	}

	help := FormatEnumHelp(def)

	// Check that help contains expected content
	if !strings.Contains(help, "Test options:") {
		t.Error("Help should contain enum name")
	}
	if !strings.Contains(help, "a (Alpha): First option") {
		t.Error("Help should contain first option with label and description")
	}
	if !strings.Contains(help, "b: Second option") {
		t.Error("Help should contain second option with description")
	}
	if !strings.Contains(help, "  c\n") {
		t.Error("Help should contain third option without description")
	}
}

func TestEnumDef_EmptyValues(t *testing.T) {
	def := EnumDef{
		Name:   "Empty",
		Values: []EnumValue{},
	}

	values := def.GetValues()
	if len(values) != 0 {
		t.Errorf("GetValues() should return empty slice, got %v", values)
	}

	labels := def.GetLabels()
	if len(labels) != 0 {
		t.Errorf("GetLabels() should return empty slice, got %v", labels)
	}

	if def.IsValid("anything") {
		t.Error("IsValid() should return false for empty enum")
	}
}
