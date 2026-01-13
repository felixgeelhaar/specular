package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetSupportedLanguages(t *testing.T) {
	langs := GetSupportedLanguages()

	expected := []string{"go", "python", "node", "shell"}
	if len(langs) != len(expected) {
		t.Errorf("GetSupportedLanguages() returned %d languages, want %d", len(langs), len(expected))
	}

	for _, exp := range expected {
		found := false
		for _, lang := range langs {
			if lang == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetSupportedLanguages() missing %q", exp)
		}
	}
}

func TestScaffold_AllLanguagesAllTypes(t *testing.T) {
	languages := GetSupportedLanguages()
	types := []string{"provider", "validator", "formatter", "hook", "notifier"}

	for _, lang := range languages {
		for _, pluginType := range types {
			t.Run(lang+"_"+pluginType, func(t *testing.T) {
				// Create temp directory
				tmpDir := t.TempDir()
				pluginDir := filepath.Join(tmpDir, "test-plugin")

				cfg := ScaffoldConfig{
					Name:   "test-plugin",
					Type:   pluginType,
					Lang:   lang,
					Author: "Test Author",
				}

				err := Scaffold(pluginDir, cfg)
				if err != nil {
					t.Fatalf("Scaffold() error = %v", err)
				}

				// Verify plugin.yaml exists
				pluginYaml := filepath.Join(pluginDir, "plugin.yaml")
				if _, err := os.Stat(pluginYaml); os.IsNotExist(err) {
					t.Errorf("plugin.yaml not created")
				}

				// Read plugin.yaml and verify content
				content, err := os.ReadFile(pluginYaml)
				if err != nil {
					t.Fatalf("Failed to read plugin.yaml: %v", err)
				}

				// Verify template variables were substituted
				contentStr := string(content)
				if !strings.Contains(contentStr, "name: test-plugin") {
					t.Errorf("plugin.yaml missing name, got:\n%s", contentStr)
				}
				if !strings.Contains(contentStr, "type: "+pluginType) {
					t.Errorf("plugin.yaml missing type, got:\n%s", contentStr)
				}

				// Verify language-specific entrypoint exists
				var entrypoint string
				switch lang {
				case "go":
					entrypoint = "main.go"
				case "python":
					entrypoint = "main.py"
				case "node":
					entrypoint = "index.js"
				case "shell":
					entrypoint = "entrypoint.sh"
				}

				entrypointPath := filepath.Join(pluginDir, entrypoint)
				info, err := os.Stat(entrypointPath)
				if os.IsNotExist(err) {
					t.Errorf("%s not created", entrypoint)
				} else if err != nil {
					t.Errorf("Failed to stat %s: %v", entrypoint, err)
				} else {
					// Verify entrypoints are executable
					if info.Mode().Perm()&0100 == 0 {
						t.Errorf("%s should be executable", entrypoint)
					}
				}
			})
		}
	}
}

func TestScaffold_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "default-plugin")

	cfg := ScaffoldConfig{
		Name: "default-plugin",
		Type: "provider",
		Lang: "shell",
		// Version and Description left empty to test defaults
	}

	err := Scaffold(pluginDir, cfg)
	if err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	// Read plugin.yaml
	content, err := os.ReadFile(filepath.Join(pluginDir, "plugin.yaml"))
	if err != nil {
		t.Fatalf("Failed to read plugin.yaml: %v", err)
	}

	contentStr := string(content)

	// Check default version
	if !strings.Contains(contentStr, `version: "0.1.0"`) {
		t.Errorf("plugin.yaml should have default version 0.1.0, got:\n%s", contentStr)
	}

	// Check default description
	if !strings.Contains(contentStr, "A Specular provider plugin") {
		t.Errorf("plugin.yaml should have default description, got:\n%s", contentStr)
	}
}

func TestScaffold_InvalidLanguage(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ScaffoldConfig{
		Name: "test-plugin",
		Type: "provider",
		Lang: "invalid-lang",
	}

	err := Scaffold(filepath.Join(tmpDir, "test-plugin"), cfg)
	if err == nil {
		t.Error("Scaffold() should error for invalid language")
	}
	if !strings.Contains(err.Error(), "unsupported language") {
		t.Errorf("Error should mention unsupported language, got: %v", err)
	}
}

func TestScaffold_InvalidType(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ScaffoldConfig{
		Name: "test-plugin",
		Type: "invalid-type",
		Lang: "shell",
	}

	err := Scaffold(filepath.Join(tmpDir, "test-plugin"), cfg)
	if err == nil {
		t.Error("Scaffold() should error for invalid type")
	}
	if !strings.Contains(err.Error(), "unsupported plugin type") {
		t.Errorf("Error should mention unsupported plugin type, got: %v", err)
	}
}

func TestScaffold_DirectoryAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "existing-plugin")

	// Create first plugin
	cfg := ScaffoldConfig{
		Name: "existing-plugin",
		Type: "provider",
		Lang: "shell",
	}

	err := Scaffold(pluginDir, cfg)
	if err != nil {
		t.Fatalf("First Scaffold() error = %v", err)
	}

	// MkdirAll doesn't error on existing directory, but we can test
	// that files are overwritten/created in an existing directory
	err = Scaffold(pluginDir, cfg)
	if err != nil {
		t.Fatalf("Second Scaffold() should not error: %v", err)
	}
}

func TestScaffold_GoSpecificFiles(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "go-plugin")

	cfg := ScaffoldConfig{
		Name: "go-plugin",
		Type: "provider",
		Lang: "go",
	}

	err := Scaffold(pluginDir, cfg)
	if err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	// Check go.mod exists
	goMod := filepath.Join(pluginDir, "go.mod")
	if _, err := os.Stat(goMod); os.IsNotExist(err) {
		t.Error("go.mod not created for Go plugin")
	}

	// Verify go.mod content
	content, err := os.ReadFile(goMod)
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}
	if !strings.Contains(string(content), "module go-plugin") {
		t.Errorf("go.mod should have module name, got:\n%s", content)
	}
}

func TestScaffold_PythonSpecificFiles(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "py-plugin")

	cfg := ScaffoldConfig{
		Name: "py-plugin",
		Type: "validator",
		Lang: "python",
	}

	err := Scaffold(pluginDir, cfg)
	if err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	// Check requirements.txt exists
	reqFile := filepath.Join(pluginDir, "requirements.txt")
	if _, err := os.Stat(reqFile); os.IsNotExist(err) {
		t.Error("requirements.txt not created for Python plugin")
	}
}

func TestScaffold_NodeSpecificFiles(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "node-plugin")

	cfg := ScaffoldConfig{
		Name: "node-plugin",
		Type: "notifier",
		Lang: "node",
	}

	err := Scaffold(pluginDir, cfg)
	if err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	// Check package.json exists
	pkgFile := filepath.Join(pluginDir, "package.json")
	if _, err := os.Stat(pkgFile); os.IsNotExist(err) {
		t.Error("package.json not created for Node.js plugin")
	}

	// Verify package.json content
	content, err := os.ReadFile(pkgFile)
	if err != nil {
		t.Fatalf("Failed to read package.json: %v", err)
	}
	if !strings.Contains(string(content), `"name": "node-plugin"`) {
		t.Errorf("package.json should have name, got:\n%s", content)
	}
}

func TestIsEntrypoint(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"main.go", true},
		{"main.py", true},
		{"index.js", true},
		{"entrypoint.sh", true},
		{"plugin.yaml", false},
		{"go.mod", false},
		{"package.json", false},
		{"requirements.txt", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEntrypoint(tt.name); got != tt.expected {
				t.Errorf("isEntrypoint(%q) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"a", "b", "c"}

	tests := []struct {
		item     string
		expected bool
	}{
		{"a", true},
		{"b", true},
		{"c", true},
		{"d", false},
		{"", false},
		{"A", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			if got := containsString(slice, tt.item); got != tt.expected {
				t.Errorf("containsString(%v, %q) = %v, want %v", slice, tt.item, got, tt.expected)
			}
		})
	}

	// Test with empty slice
	if containsString([]string{}, "a") {
		t.Error("containsString([], 'a') should return false")
	}
}

func TestScaffold_TemplateVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "my-custom-plugin")

	cfg := ScaffoldConfig{
		Name:        "my-custom-plugin",
		Type:        "provider",
		Lang:        "go",
		Author:      "John Doe",
		Description: "Custom description here",
		Version:     "1.2.3",
	}

	err := Scaffold(pluginDir, cfg)
	if err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	// Read plugin.yaml and verify all variables
	content, err := os.ReadFile(filepath.Join(pluginDir, "plugin.yaml"))
	if err != nil {
		t.Fatalf("Failed to read plugin.yaml: %v", err)
	}

	contentStr := string(content)

	// Check all substitutions
	checks := []struct {
		desc     string
		contains string
	}{
		{"name", "name: my-custom-plugin"},
		{"version", `version: "1.2.3"`},
		{"author", `author: "John Doe"`},
		{"description", `description: "Custom description here"`},
		{"type", "type: provider"},
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check.contains) {
			t.Errorf("plugin.yaml missing %s: expected %q in:\n%s", check.desc, check.contains, contentStr)
		}
	}

	// Also check main.go has version substituted
	mainGo, err := os.ReadFile(filepath.Join(pluginDir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	if !strings.Contains(string(mainGo), `Version = "1.2.3"`) {
		t.Errorf("main.go should have version substituted, got:\n%s", mainGo)
	}
}
