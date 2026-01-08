package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

var _ = cobra.ShellCompDirectiveError // Ensure cobra import is used

func TestCompletionCommand(t *testing.T) {
	// Test that completion command is registered
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "completion [bash|zsh|fish|powershell]" {
			found = true
			break
		}
	}
	if !found {
		t.Error("completion command not registered under root")
	}
}

func TestCompletionValidArgs(t *testing.T) {
	if len(completionCmd.ValidArgs) != 4 {
		t.Errorf("Expected 4 valid args, got %d", len(completionCmd.ValidArgs))
	}

	expected := map[string]bool{
		"bash":       false,
		"zsh":        false,
		"fish":       false,
		"powershell": false,
	}

	for _, arg := range completionCmd.ValidArgs {
		if _, ok := expected[arg]; ok {
			expected[arg] = true
		}
	}

	for shell, found := range expected {
		if !found {
			t.Errorf("Missing valid arg: %s", shell)
		}
	}
}

func TestCompletionBash(t *testing.T) {
	var buf bytes.Buffer

	// Use rootCmd.GenBashCompletion directly
	err := rootCmd.GenBashCompletion(&buf)
	if err != nil {
		t.Fatalf("completion bash error = %v", err)
	}

	output := buf.String()
	if len(output) < 100 {
		t.Error("Bash completion output seems too short")
	}
	if !strings.Contains(output, "specular") {
		t.Error("Bash completion should contain specular")
	}
}

func TestCompletionZsh(t *testing.T) {
	var buf bytes.Buffer

	err := rootCmd.GenZshCompletion(&buf)
	if err != nil {
		t.Fatalf("completion zsh error = %v", err)
	}

	output := buf.String()
	if len(output) < 100 {
		t.Error("Zsh completion output seems too short")
	}
}

func TestCompletionFish(t *testing.T) {
	var buf bytes.Buffer

	err := rootCmd.GenFishCompletion(&buf, true)
	if err != nil {
		t.Fatalf("completion fish error = %v", err)
	}

	output := buf.String()
	if len(output) < 100 {
		t.Error("Fish completion output seems too short")
	}
}

func TestCompletionPowershell(t *testing.T) {
	var buf bytes.Buffer

	err := rootCmd.GenPowerShellCompletionWithDesc(&buf)
	if err != nil {
		t.Fatalf("completion powershell error = %v", err)
	}

	output := buf.String()
	if len(output) < 100 {
		t.Error("PowerShell completion output seems too short")
	}
}

func TestWorkflowTemplateCompletion(t *testing.T) {
	completions, directive := workflowTemplateCompletion(nil, nil, "")

	if directive == cobra.ShellCompDirectiveError {
		t.Error("workflowTemplateCompletion returned error directive")
	}

	if len(completions) == 0 {
		t.Error("workflowTemplateCompletion should return completions")
	}

	// Check that ci-pipeline is in completions
	hasCIPipeline := false
	for _, c := range completions {
		if strings.HasPrefix(c, "ci-pipeline") {
			hasCIPipeline = true
			break
		}
	}
	if !hasCIPipeline {
		t.Error("Expected ci-pipeline in workflow completions")
	}
}

func TestWorkflowTemplateCompletionFiltered(t *testing.T) {
	completions, _ := workflowTemplateCompletion(nil, nil, "ci")

	// Should filter to only ci-prefixed templates
	for _, c := range completions {
		value := strings.Split(c, "\t")[0]
		if !strings.HasPrefix(value, "ci") {
			t.Errorf("Completion %q should start with 'ci'", value)
		}
	}
}

func TestFilterCompletions(t *testing.T) {
	options := []string{
		"apple\tFruit",
		"apricot\tFruit",
		"banana\tFruit",
	}

	tests := []struct {
		prefix   string
		expected int
	}{
		{"", 3},   // No filter, all options
		{"a", 2},  // apple, apricot
		{"ap", 2}, // apple, apricot
		{"b", 1},  // banana
		{"c", 0},  // nothing
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			result := filterCompletions(options, tt.prefix)
			if len(result) != tt.expected {
				t.Errorf("filterCompletions(%q) = %d items, want %d", tt.prefix, len(result), tt.expected)
			}
		})
	}
}

func TestCompletionShells(t *testing.T) {
	shells := CompletionShells()

	if len(shells) != 4 {
		t.Errorf("Expected 4 shells, got %d", len(shells))
	}

	expected := map[string]bool{
		"bash":       false,
		"zsh":        false,
		"fish":       false,
		"powershell": false,
	}

	for _, shell := range shells {
		expected[shell] = true
	}

	for shell, found := range expected {
		if !found {
			t.Errorf("Missing shell: %s", shell)
		}
	}
}

func TestRegisterDynamicCompletions(t *testing.T) {
	// This test just ensures registerDynamicCompletions doesn't panic
	// The actual registration depends on other commands existing
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("registerDynamicCompletions panicked: %v", r)
		}
	}()

	registerDynamicCompletions()
}
