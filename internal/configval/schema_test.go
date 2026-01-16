package configval

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDetectConfigType(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		content  map[string]interface{}
		want     ConfigType
	}{
		{
			name:     "routing from filename",
			filePath: "routing.yaml",
			content:  map[string]interface{}{},
			want:     ConfigTypeRouter,
		},
		{
			name:     "providers from filename",
			filePath: "providers.yaml",
			content:  map[string]interface{}{},
			want:     ConfigTypeProviders,
		},
		{
			name:     "spec from filename",
			filePath: "spec.yaml",
			content:  map[string]interface{}{},
			want:     ConfigTypeSpec,
		},
		{
			name:     "policy from filename",
			filePath: "policy.yaml",
			content:  map[string]interface{}{},
			want:     ConfigTypePolicy,
		},
		{
			name:     "slo from filename",
			filePath: "slo.yaml",
			content:  map[string]interface{}{},
			want:     ConfigTypeSLO,
		},
		{
			name:     "providers from content",
			filePath: "config.yaml",
			content: map[string]interface{}{
				"providers": []interface{}{},
				"strategy":  map[string]interface{}{},
			},
			want: ConfigTypeProviders,
		},
		{
			name:     "spec from content",
			filePath: "config.yaml",
			content: map[string]interface{}{
				"tasks": []interface{}{},
			},
			want: ConfigTypeSpec,
		},
		{
			name:     "slo from content",
			filePath: "config.yaml",
			content: map[string]interface{}{
				"objectives": []interface{}{},
			},
			want: ConfigTypeSLO,
		},
		{
			name:     "policy from content",
			filePath: "config.yaml",
			content: map[string]interface{}{
				"rules": []interface{}{},
			},
			want: ConfigTypePolicy,
		},
		{
			name:     "unknown",
			filePath: "unknown.yaml",
			content:  map[string]interface{}{},
			want:     ConfigTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectConfigType(tt.filePath, tt.content)
			if got != tt.want {
				t.Errorf("DetectConfigType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSchema(t *testing.T) {
	tests := []struct {
		name       string
		configType ConfigType
		wantNil    bool
	}{
		{"providers", ConfigTypeProviders, false},
		{"router", ConfigTypeRouter, false},
		{"spec", ConfigTypeSpec, false},
		{"policy", ConfigTypePolicy, false},
		{"slo", ConfigTypeSLO, false},
		{"unknown", ConfigTypeUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := GetSchema(tt.configType)
			if tt.wantNil && schema != nil {
				t.Errorf("GetSchema() = %v, want nil", schema)
			}
			if !tt.wantNil && schema == nil {
				t.Error("GetSchema() = nil, want non-nil")
			}
			if schema != nil && schema.Type != tt.configType {
				t.Errorf("GetSchema().Type = %v, want %v", schema.Type, tt.configType)
			}
		})
	}
}

func TestValidateConfig_ValidProviders(t *testing.T) {
	content := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name":    "test-provider",
				"type":    "api",
				"enabled": true,
			},
		},
	}

	ctx := &ValidationContext{
		FilePath:   "providers.yaml",
		ConfigType: ConfigTypeProviders,
		Parsed:     content,
	}

	result := ValidateConfig(ctx)

	if result.HasErrors() {
		t.Errorf("ValidateConfig() unexpected errors: %v", result.Errors())
	}
}

func TestValidateConfig_MissingRequiredField(t *testing.T) {
	content := map[string]interface{}{
		// Missing "providers" field
	}

	ctx := &ValidationContext{
		FilePath:   "providers.yaml",
		ConfigType: ConfigTypeProviders,
		Parsed:     content,
	}

	result := ValidateConfig(ctx)

	if !result.HasErrors() {
		t.Error("ValidateConfig() expected errors for missing required field")
	}

	// Check that the error mentions the missing field (can be in Field or Message)
	found := false
	for _, err := range result.Errors() {
		if strings.Contains(err.Field, "providers") || strings.Contains(err.Message, "providers") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Error should mention the missing 'providers' field, got errors: %v", result.Errors())
	}
}

func TestValidateConfig_InvalidEnumValue(t *testing.T) {
	content := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name":    "test-provider",
				"type":    "invalid-type", // Should be cli, api, grpc, or native
				"enabled": true,
			},
		},
	}

	ctx := &ValidationContext{
		FilePath:   "providers.yaml",
		ConfigType: ConfigTypeProviders,
		Parsed:     content,
	}

	result := ValidateConfig(ctx)

	if !result.HasErrors() {
		t.Error("ValidateConfig() expected errors for invalid enum value")
	}
}

func TestValidateConfig_NegativeNumber(t *testing.T) {
	content := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name":    "test",
				"type":    "api",
				"enabled": true,
			},
		},
		"strategy": map[string]interface{}{
			"budget": map[string]interface{}{
				"max_cost_per_day": -10.0, // Should be non-negative
			},
		},
	}

	ctx := &ValidationContext{
		FilePath:   "providers.yaml",
		ConfigType: ConfigTypeProviders,
		Parsed:     content,
	}

	result := ValidateConfig(ctx)

	if !result.HasErrors() {
		t.Error("ValidateConfig() expected errors for negative number")
	}
}

func TestValidateConfig_UnknownType(t *testing.T) {
	content := map[string]interface{}{
		"some_field": "value",
	}

	ctx := &ValidationContext{
		FilePath:   "unknown.yaml",
		ConfigType: ConfigTypeUnknown,
		Parsed:     content,
	}

	result := ValidateConfig(ctx)

	// Should have a warning about unknown type but not error
	if result.HasErrors() {
		t.Error("ValidateConfig() should not error for unknown config type")
	}
	if !result.HasWarnings() {
		t.Error("ValidateConfig() should warn about unknown config type")
	}
}

func TestGetFieldValues(t *testing.T) {
	data := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{"name": "provider1", "type": "api"},
			map[string]interface{}{"name": "provider2", "type": "cli"},
		},
		"strategy": map[string]interface{}{
			"budget": map[string]interface{}{
				"max_cost_per_day": 100.0,
			},
		},
	}

	tests := []struct {
		path      string
		wantCount int
		wantExist bool
	}{
		{"providers", 1, true},
		{"providers[*].name", 2, true},
		{"strategy.budget.max_cost_per_day", 1, true},
		{"nonexistent", 0, false},
		{"providers[*].nonexistent", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			values, exists := getFieldValues(data, tt.path)
			if exists != tt.wantExist {
				t.Errorf("getFieldValues(%s) exists = %v, want %v", tt.path, exists, tt.wantExist)
			}
			if len(values) != tt.wantCount {
				t.Errorf("getFieldValues(%s) count = %d, want %d", tt.path, len(values), tt.wantCount)
			}
		})
	}
}

func TestValidateType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		typeName string
		want     bool
	}{
		{"string valid", "hello", "string", true},
		{"string invalid", 123, "string", false},
		{"number int valid", 123, "number", true},
		{"number float valid", 123.45, "number", true},
		{"number invalid", "123", "number", false},
		{"boolean valid", true, "boolean", true},
		{"boolean invalid", "true", "boolean", false},
		{"array valid", []interface{}{"a", "b"}, "array", true},
		{"array invalid", "a,b", "array", false},
		{"object valid", map[string]interface{}{"key": "value"}, "object", true},
		{"object invalid", "object", "object", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateType(tt.value, tt.typeName)
			if got != tt.want {
				t.Errorf("validateType(%v, %s) = %v, want %v", tt.value, tt.typeName, got, tt.want)
			}
		})
	}
}

func TestNewValidationContext(t *testing.T) {
	// Create a temp file with valid YAML
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "providers.yaml")

	content := `providers:
  - name: test
    type: api
    enabled: true
strategy:
  budget:
    max_cost_per_day: 100
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx, err := NewValidationContext(testFile)
	if err != nil {
		t.Fatalf("NewValidationContext() error = %v", err)
	}

	if ctx.ConfigType != ConfigTypeProviders {
		t.Errorf("ConfigType = %v, want %v", ctx.ConfigType, ConfigTypeProviders)
	}

	if ctx.Parsed == nil {
		t.Error("Parsed should not be nil")
	}
}

func TestNewValidationContext_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.yaml")

	content := `invalid: yaml: content: [}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := NewValidationContext(testFile)
	if err == nil {
		t.Error("NewValidationContext() expected error for invalid YAML")
	}
}

func TestNewValidationContext_MissingFile(t *testing.T) {
	_, err := NewValidationContext("/nonexistent/file.yaml")
	if err == nil {
		t.Error("NewValidationContext() expected error for missing file")
	}
}

func TestValidateFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Valid config
	validFile := filepath.Join(tmpDir, "providers.yaml")
	validContent, _ := yaml.Marshal(map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name":    "test",
				"type":    "api",
				"enabled": true,
			},
		},
	})
	if err := os.WriteFile(validFile, validContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var buf bytes.Buffer
	err := ValidateFile(validFile, &buf)
	if err != nil {
		t.Errorf("ValidateFile() unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "Valid") {
		t.Error("Output should indicate valid configuration")
	}
}

func TestValidateFile_Invalid(t *testing.T) {
	tmpDir := t.TempDir()

	// Invalid config - missing required field
	invalidFile := filepath.Join(tmpDir, "providers.yaml")
	invalidContent, _ := yaml.Marshal(map[string]interface{}{
		// Missing providers
	})
	if err := os.WriteFile(invalidFile, invalidContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var buf bytes.Buffer
	err := ValidateFile(invalidFile, &buf)
	if err == nil {
		t.Error("ValidateFile() expected error for invalid config")
	}

	if !strings.Contains(buf.String(), "error") {
		t.Error("Output should indicate errors")
	}
}

func TestIsNumber(t *testing.T) {
	tests := []struct {
		value interface{}
		want  bool
	}{
		{42, true},
		{int64(42), true},
		{float64(42.5), true},
		{float32(42.5), true},
		{"42", false},
		{true, false},
		{nil, false},
	}

	for _, tt := range tests {
		got := isNumber(tt.value)
		if got != tt.want {
			t.Errorf("isNumber(%v) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		value interface{}
		want  float64
	}{
		{42, 42.0},
		{int64(42), 42.0},
		{float64(42.5), 42.5},
		{float32(42.5), float64(float32(42.5))},
		{"42", 0}, // Non-numeric returns 0
	}

	for _, tt := range tests {
		got := toFloat64(tt.value)
		if got != tt.want {
			t.Errorf("toFloat64(%v) = %v, want %v", tt.value, got, tt.want)
		}
	}
}
