package cmd

import (
	"testing"

	"github.com/felixgeelhaar/specular/internal/templates"
)

// TestQuickstartCommand tests the quickstart command configuration
func TestQuickstartCommand(t *testing.T) {
	if quickstartCmd == nil {
		t.Fatal("quickstartCmd is nil")
	}

	if quickstartCmd.Use != "quickstart" {
		t.Errorf("quickstartCmd.Use = %q, want %q", quickstartCmd.Use, "quickstart")
	}

	if quickstartCmd.Short == "" {
		t.Error("quickstartCmd.Short is empty")
	}

	if quickstartCmd.Long == "" {
		t.Error("quickstartCmd.Long is empty")
	}
}

// TestQuickstartFlags tests that all quickstart flags exist
func TestQuickstartFlags(t *testing.T) {
	tests := []struct {
		flag     string
		defValue string
	}{
		{"verify", "false"},
		{"demo", "false"},
		{"provider", ""},
		{"force", "false"},
		{"template", ""},
	}

	for _, tc := range tests {
		t.Run(tc.flag, func(t *testing.T) {
			flag := quickstartCmd.Flags().Lookup(tc.flag)
			if flag == nil {
				t.Fatalf("flag %q not found", tc.flag)
			}

			if flag.DefValue != tc.defValue {
				t.Errorf("flag %q default = %q, want %q", tc.flag, flag.DefValue, tc.defValue)
			}
		})
	}
}

// TestQuickstartFlagShortcuts tests that flag shortcuts work
func TestQuickstartFlagShortcuts(t *testing.T) {
	// Force flag should have -f shortcut
	forceFlag := quickstartCmd.Flags().ShorthandLookup("f")
	if forceFlag == nil {
		t.Error("shorthand 'f' not found for force flag")
	}

	// Template flag should have -t shortcut
	templateFlag := quickstartCmd.Flags().ShorthandLookup("t")
	if templateFlag == nil {
		t.Error("shorthand 't' not found for template flag")
	}
}

// TestQuickstartLongDescription tests that Long description contains template info
func TestQuickstartLongDescription(t *testing.T) {
	long := quickstartCmd.Long

	// Should mention templates
	expectedPhrases := []string{
		"rest-api",
		"cli-tool",
		"web-app",
		"library",
		"template",
	}

	for _, phrase := range expectedPhrases {
		found := false
		for i := 0; i <= len(long)-len(phrase); i++ {
			if long[i:i+len(phrase)] == phrase {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected phrase %q not found in Long description", phrase)
		}
	}
}

// TestQuickstartTemplateRegistry tests template integration
func TestQuickstartTemplateRegistry(t *testing.T) {
	registry, err := templates.NewRegistry()
	if err != nil {
		t.Fatalf("failed to create template registry: %v", err)
	}

	// Verify all templates mentioned in quickstart are available
	templateIDs := []string{"rest-api", "cli-tool", "web-app", "library"}
	for _, id := range templateIDs {
		tmpl, err := registry.Get(id)
		if err != nil {
			t.Errorf("template %q not found in registry: %v", id, err)
			continue
		}

		if tmpl.Name == "" {
			t.Errorf("template %q has empty name", id)
		}

		if tmpl.Description == "" {
			t.Errorf("template %q has empty description", id)
		}
	}
}

// TestQuickstartTemplateToSpec tests that templates convert to valid specs
func TestQuickstartTemplateToSpec(t *testing.T) {
	registry, err := templates.NewRegistry()
	if err != nil {
		t.Fatalf("failed to create template registry: %v", err)
	}

	for _, tmpl := range registry.List() {
		t.Run(tmpl.ID, func(t *testing.T) {
			spec := tmpl.ToProductSpec("")

			if spec.Product == "" {
				t.Error("converted spec has empty Product")
			}

			if len(spec.Goals) == 0 {
				t.Error("converted spec has no Goals")
			}

			if len(spec.Features) == 0 {
				t.Error("converted spec has no Features")
			}
		})
	}
}

// TestTruncateString tests the truncateString helper function
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"hello", 5, "hello"},
		{"hi", 10, "hi"},
		{"", 5, ""},
		{"abcdefghij", 7, "abcd..."},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := truncateString(tc.input, tc.maxLen)
			if got != tc.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.want)
			}
		})
	}
}
