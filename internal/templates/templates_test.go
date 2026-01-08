package templates

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	// Verify templates were loaded
	templates := registry.List()
	if len(templates) == 0 {
		t.Error("expected at least one template to be loaded")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tests := []struct {
		name      string
		id        string
		wantErr   bool
		wantName  string
	}{
		{
			name:     "rest-api template",
			id:       "rest-api",
			wantErr:  false,
			wantName: "REST API Service",
		},
		{
			name:     "cli-tool template",
			id:       "cli-tool",
			wantErr:  false,
			wantName: "CLI Tool",
		},
		{
			name:     "web-app template",
			id:       "web-app",
			wantErr:  false,
			wantName: "Web Application",
		},
		{
			name:     "library template",
			id:       "library",
			wantErr:  false,
			wantName: "Go Library",
		},
		{
			name:    "non-existent template",
			id:      "non-existent",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := registry.Get(tc.id)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error for non-existent template")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tmpl.Name != tc.wantName {
				t.Errorf("expected name %q, got %q", tc.wantName, tmpl.Name)
			}
		})
	}
}

func TestRegistry_ListByTag(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	// API tag should return rest-api template
	apiTemplates := registry.ListByTag("api")
	if len(apiTemplates) == 0 {
		t.Error("expected at least one template with 'api' tag")
	}

	// CLI tag should return cli-tool template
	cliTemplates := registry.ListByTag("cli")
	if len(cliTemplates) == 0 {
		t.Error("expected at least one template with 'cli' tag")
	}
}

func TestRegistry_ListByLanguage(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	// Go templates
	goTemplates := registry.ListByLanguage("go")
	if len(goTemplates) == 0 {
		t.Error("expected at least one Go template")
	}
}

func TestTemplate_ToProductSpec(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tmpl, err := registry.Get("rest-api")
	if err != nil {
		t.Fatalf("failed to get rest-api template: %v", err)
	}

	// Convert with custom product name
	spec := tmpl.ToProductSpec("My API")
	if spec.Product != "My API" {
		t.Errorf("expected product 'My API', got %q", spec.Product)
	}

	// Convert with default product name
	spec = tmpl.ToProductSpec("")
	if spec.Product != "REST API Service" {
		t.Errorf("expected product 'REST API Service', got %q", spec.Product)
	}

	// Verify features were converted
	if len(spec.Features) == 0 {
		t.Error("expected features to be converted")
	}

	// Verify goals were converted
	if len(spec.Goals) == 0 {
		t.Error("expected goals to be converted")
	}
}

func TestAvailableTemplates(t *testing.T) {
	output := AvailableTemplates()
	if output == "" {
		t.Error("expected non-empty template list")
	}

	// Should contain our template IDs
	if !containsString(output, "rest-api") {
		t.Error("expected output to contain 'rest-api'")
	}
	if !containsString(output, "cli-tool") {
		t.Error("expected output to contain 'cli-tool'")
	}
}

func TestGetTemplateIDs(t *testing.T) {
	ids := GetTemplateIDs()
	if len(ids) == 0 {
		t.Error("expected at least one template ID")
	}

	// Verify expected IDs
	hasRestAPI := false
	hasCLITool := false
	for _, id := range ids {
		if id == "rest-api" {
			hasRestAPI = true
		}
		if id == "cli-tool" {
			hasCLITool = true
		}
	}

	if !hasRestAPI {
		t.Error("expected 'rest-api' in template IDs")
	}
	if !hasCLITool {
		t.Error("expected 'cli-tool' in template IDs")
	}
}

func TestTemplate_Features(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tests := []struct {
		templateID      string
		minFeatures     int
		hasAPI          bool
		hasNonFunctional bool
	}{
		{"rest-api", 3, true, true},
		{"cli-tool", 4, false, true},
		{"web-app", 4, true, true},
		{"library", 5, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.templateID, func(t *testing.T) {
			tmpl, err := registry.Get(tc.templateID)
			if err != nil {
				t.Fatalf("failed to get template: %v", err)
			}

			if len(tmpl.Spec.Features) < tc.minFeatures {
				t.Errorf("expected at least %d features, got %d", tc.minFeatures, len(tmpl.Spec.Features))
			}

			if tc.hasAPI {
				hasAPI := false
				for _, f := range tmpl.Spec.Features {
					if len(f.API) > 0 {
						hasAPI = true
						break
					}
				}
				if !hasAPI {
					t.Error("expected template to have API endpoints")
				}
			}

			if tc.hasNonFunctional {
				nf := tmpl.Spec.NonFunctional
				if len(nf.Performance) == 0 && len(nf.Security) == 0 {
					t.Error("expected non-functional requirements")
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
