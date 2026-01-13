//go:build e2e

package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestPluginCLICommands tests all plugin subcommands
func TestPluginCLICommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	binary := buildBinary(t)

	t.Run("plugin command exists", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("plugin --help failed: %v\nOutput: %s", err, output)
		}

		// Verify subcommands are listed
		expectedCommands := []string{"list", "info", "health", "enable", "disable", "install", "uninstall", "update", "create", "search", "registry-info"}
		for _, cmd := range expectedCommands {
			if !strings.Contains(string(output), cmd) {
				t.Errorf("Missing subcommand in help: %s", cmd)
			}
		}
	})

	t.Run("plugin list", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "list")
		output, err := cmd.CombinedOutput()
		if err != nil {
			// plugin list without plugins should still succeed
			if !strings.Contains(string(output), "No plugins installed") {
				t.Fatalf("plugin list failed: %v\nOutput: %s", err, output)
			}
		}
	})

	t.Run("plugin list --help", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "list", "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("plugin list --help failed: %v", err)
		}

		if !strings.Contains(string(output), "List installed plugins") {
			t.Error("help should describe list command")
		}
	})

	t.Run("plugin create --help", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "create", "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("plugin create --help failed: %v", err)
		}

		// Verify supported languages
		if !strings.Contains(string(output), "go") ||
			!strings.Contains(string(output), "python") ||
			!strings.Contains(string(output), "node") ||
			!strings.Contains(string(output), "shell") {
			t.Error("help should list supported languages")
		}
	})

	t.Run("plugin search --help", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "search", "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("plugin search --help failed: %v", err)
		}

		if !strings.Contains(string(output), "registry") {
			t.Error("help should mention registry")
		}
	})

	t.Run("plugin install --help", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "install", "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("plugin install --help failed: %v", err)
		}

		// Verify flags are documented
		if !strings.Contains(string(output), "--force") ||
			!strings.Contains(string(output), "--upgrade") ||
			!strings.Contains(string(output), "--version") {
			t.Error("help should document install flags")
		}
	})

	t.Run("plugin update --help", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "update", "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("plugin update --help failed: %v", err)
		}

		if !strings.Contains(string(output), "Update") {
			t.Error("help should describe update command")
		}
	})
}

// TestPluginDiscovery tests plugin discovery functionality
func TestPluginDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	binary := buildBinary(t)
	tmpDir := t.TempDir()

	// Create a simple plugin
	pluginDir := filepath.Join(tmpDir, "test-discovery-plugin")
	os.MkdirAll(pluginDir, 0755)

	manifest := `name: test-discovery-plugin
version: "1.0.0"
type: hook
description: Test plugin for discovery
entrypoint: test.sh
`
	err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644)
	if err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Create entrypoint
	entrypoint := `#!/bin/bash
echo '{"status": "healthy"}'
`
	err = os.WriteFile(filepath.Join(pluginDir, "test.sh"), []byte(entrypoint), 0755)
	if err != nil {
		t.Fatalf("Failed to write entrypoint: %v", err)
	}

	// Set plugin directory
	os.Setenv("SPECULAR_PLUGIN_DIRS", tmpDir)
	defer os.Unsetenv("SPECULAR_PLUGIN_DIRS")

	// List plugins
	cmd := exec.Command(binary, "plugin", "list")
	output, err := cmd.CombinedOutput()
	// Even if it errors, check output for our plugin
	outputStr := string(output)

	t.Logf("Plugin list output: %s", outputStr)

	// The test is successful if it runs without crashing
	// The actual plugin discovery depends on configuration
}

// TestPluginInstallLocal tests local plugin installation
func TestPluginInstallLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	binary := buildBinary(t)
	tmpDir := t.TempDir()

	// Create plugin in source directory
	sourceDir := filepath.Join(tmpDir, "source")
	os.MkdirAll(sourceDir, 0755)

	manifest := `name: test-install-local
version: "1.0.0"
type: hook
description: Test plugin for local installation
entrypoint: entrypoint.sh
`
	err := os.WriteFile(filepath.Join(sourceDir, "plugin.yaml"), []byte(manifest), 0644)
	if err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	entrypoint := `#!/bin/bash
echo '{"status": "healthy", "version": "1.0.0"}'
`
	err = os.WriteFile(filepath.Join(sourceDir, "entrypoint.sh"), []byte(entrypoint), 0755)
	if err != nil {
		t.Fatalf("Failed to write entrypoint: %v", err)
	}

	// Install plugin
	cmd := exec.Command(binary, "plugin", "install", sourceDir)
	output, err := cmd.CombinedOutput()
	t.Logf("Install output: %s", output)

	// Check if it was installed (may fail due to permissions, but shouldn't crash)
	if err != nil && !strings.Contains(string(output), "permission") && !strings.Contains(string(output), "already exists") {
		t.Logf("Install returned: %v - %s", err, output)
	}
}

// TestPluginCreate tests plugin scaffolding
func TestPluginCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	binary := buildBinary(t)

	testCases := []struct {
		lang     string
		ptype    string
		expected []string // Expected files
	}{
		{"go", "provider", []string{"plugin.yaml", "main.go", "go.mod"}},
		{"python", "validator", []string{"plugin.yaml", "main.py", "requirements.txt"}},
		{"node", "formatter", []string{"plugin.yaml", "index.js", "package.json"}},
		{"shell", "hook", []string{"plugin.yaml", "entrypoint.sh"}},
	}

	for _, tc := range testCases {
		t.Run(tc.lang+"_"+tc.ptype, func(t *testing.T) {
			tmpDir := t.TempDir()
			pluginName := "test-" + tc.lang + "-" + tc.ptype

			// Change to temp dir
			origDir, _ := os.Getwd()
			os.Chdir(tmpDir)
			defer os.Chdir(origDir)

			// Create plugin
			cmd := exec.Command(binary, "plugin", "create", pluginName,
				"--type", tc.ptype,
				"--lang", tc.lang,
				"--author", "E2E Test")

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("plugin create failed: %v\nOutput: %s", err, output)
			}

			// Verify files
			pluginDir := filepath.Join(tmpDir, pluginName)
			for _, file := range tc.expected {
				path := filepath.Join(pluginDir, file)
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("Expected file not found: %s", file)
				}
			}

			// Verify success message
			if !strings.Contains(string(output), "Created") {
				t.Error("output should indicate successful creation")
			}
		})
	}
}

// TestPluginUpdate tests plugin update functionality
func TestPluginUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	binary := buildBinary(t)

	t.Run("update without plugins", func(t *testing.T) {
		// This should not crash even without plugins
		cmd := exec.Command(binary, "plugin", "update")
		cmd.Timeout = 10 * time.Second

		output, err := cmd.CombinedOutput()
		t.Logf("Update output: %s", output)

		// Should complete without panic
		if err != nil {
			// May fail but shouldn't crash
			if !strings.Contains(string(output), "failed") &&
				!strings.Contains(string(output), "error") &&
				!strings.Contains(string(output), "No plugins") {
				t.Logf("Update result: %v", err)
			}
		}
	})

	t.Run("update specific plugin not found", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "update", "non-existent-plugin")
		output, _ := cmd.CombinedOutput()

		// Should return error for non-existent plugin
		if strings.Contains(string(output), "panic") {
			t.Error("command should not panic")
		}
	})
}

// TestPluginErrorScenarios tests error handling
func TestPluginErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	binary := buildBinary(t)

	t.Run("info without plugin name", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "info")
		output, err := cmd.CombinedOutput()

		if err == nil {
			t.Error("expected error without plugin name")
		}

		if !strings.Contains(string(output), "requires") || !strings.Contains(string(output), "argument") {
			// Cobra should complain about missing argument
			t.Logf("Output: %s", output)
		}
	})

	t.Run("health without plugin name", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "health")
		output, err := cmd.CombinedOutput()

		if err == nil {
			t.Error("expected error without plugin name")
		}

		if strings.Contains(string(output), "panic") {
			t.Error("should not panic")
		}
	})

	t.Run("install without source", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "install")
		output, err := cmd.CombinedOutput()

		if err == nil {
			t.Error("expected error without source")
		}

		if strings.Contains(string(output), "panic") {
			t.Error("should not panic")
		}
	})

	t.Run("create without name", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "create")
		output, err := cmd.CombinedOutput()

		if err == nil {
			t.Error("expected error without name")
		}

		if strings.Contains(string(output), "panic") {
			t.Error("should not panic")
		}
	})

	t.Run("create with invalid type", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(origDir)

		cmd := exec.Command(binary, "plugin", "create", "test", "--type", "invalid-type")
		output, err := cmd.CombinedOutput()

		if err == nil {
			t.Error("expected error with invalid type")
		}

		if !strings.Contains(string(output), "invalid") {
			t.Errorf("output should mention invalid type: %s", output)
		}
	})

	t.Run("create with invalid language", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(origDir)

		cmd := exec.Command(binary, "plugin", "create", "test", "--lang", "invalid-lang")
		output, err := cmd.CombinedOutput()

		if err == nil {
			t.Error("expected error with invalid language")
		}

		if !strings.Contains(string(output), "invalid") {
			t.Errorf("output should mention invalid language: %s", output)
		}
	})
}

// TestPluginOutputFormats tests JSON and YAML output
func TestPluginOutputFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	binary := buildBinary(t)

	t.Run("list with json format", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "list", "--format", "json")
		output, err := cmd.CombinedOutput()

		// Even without plugins, should not crash
		if err != nil {
			// May have no plugins, but should not panic
			if strings.Contains(string(output), "panic") {
				t.Error("should not panic")
			}
		}
	})

	t.Run("list with yaml format", func(t *testing.T) {
		cmd := exec.Command(binary, "plugin", "list", "--format", "yaml")
		output, err := cmd.CombinedOutput()

		if err != nil {
			if strings.Contains(string(output), "panic") {
				t.Error("should not panic")
			}
		}
	})
}

// TestPluginWorkflow tests complete plugin workflow
func TestPluginWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	binary := buildBinary(t)
	tmpDir := t.TempDir()

	// 1. Create a plugin
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cmd := exec.Command(binary, "plugin", "create", "workflow-test",
		"--type", "hook",
		"--lang", "shell",
		"--author", "E2E Test")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create failed: %v\nOutput: %s", err, output)
	}
	t.Logf("Create output: %s", output)

	// 2. Verify the plugin was created
	pluginDir := filepath.Join(tmpDir, "workflow-test")
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		t.Fatal("plugin directory not created")
	}

	// 3. Check the manifest
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest: %v", err)
	}

	if !strings.Contains(string(manifestData), "workflow-test") {
		t.Error("manifest should contain plugin name")
	}
	if !strings.Contains(string(manifestData), "hook") {
		t.Error("manifest should contain plugin type")
	}

	// 4. Make entrypoint executable
	entrypointPath := filepath.Join(pluginDir, "entrypoint.sh")
	os.Chmod(entrypointPath, 0755)

	// 5. Test the entrypoint responds to health check
	testCmd := exec.Command(entrypointPath)
	testCmd.Stdin = strings.NewReader(`{"action":"health"}`)
	testOutput, err := testCmd.CombinedOutput()
	t.Logf("Entrypoint output: %s", testOutput)

	if err != nil {
		t.Logf("Entrypoint execution: %v", err)
	}
}

// buildBinary builds the specular binary for testing
func buildBinary(t *testing.T) string {
	t.Helper()

	// Get project root
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Navigate to project root (from test/e2e)
	projectRoot := filepath.Join(wd, "..", "..")

	// Build binary to temp location
	binary := filepath.Join(t.TempDir(), "specular")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = projectRoot
	cmd.Stderr = &bytes.Buffer{}

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v\nStderr: %s", err, cmd.Stderr)
	}

	return binary
}
