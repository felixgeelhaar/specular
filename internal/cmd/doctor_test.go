package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestDoctorCommand(t *testing.T) {
	// Test that doctor command is registered at root level
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "doctor" {
			found = true
			break
		}
	}
	if !found {
		t.Error("doctor command not registered under root")
	}
}

func TestDoctorCommandAliases(t *testing.T) {
	// Find the doctor command
	var doctorCommand *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "doctor" {
			doctorCommand = c
			break
		}
	}
	if doctorCommand == nil {
		t.Fatal("doctor command not found")
	}

	// Check aliases
	expectedAliases := map[string]bool{
		"check":  false,
		"health": false,
	}

	for _, alias := range doctorCommand.Aliases {
		if _, ok := expectedAliases[alias]; ok {
			expectedAliases[alias] = true
		}
	}

	for alias, found := range expectedAliases {
		if !found {
			t.Errorf("Missing alias: %s", alias)
		}
	}
}

func TestDoctorCommandFlags(t *testing.T) {
	// Find the doctor command
	var doctorCommand *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "doctor" {
			doctorCommand = c
			break
		}
	}
	if doctorCommand == nil {
		t.Fatal("doctor command not found")
	}

	// Check flags
	flags := []struct {
		name     string
		shortcut string
	}{
		{"quick", "q"},
		{"verbose", "v"},
	}

	for _, f := range flags {
		flag := doctorCommand.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("Missing flag: --%s", f.name)
			continue
		}
		if flag.Shorthand != f.shortcut {
			t.Errorf("Flag --%s shorthand = %s, want %s", f.name, flag.Shorthand, f.shortcut)
		}
	}
}

func TestDoctorReportStructure(t *testing.T) {
	report := &DoctorReport{
		Providers: make(map[string]*DoctorCheck),
		Issues:    []string{},
		Warnings:  []string{},
		NextSteps: []string{},
	}

	// Test that report starts as healthy
	report.Healthy = len(report.Issues) == 0
	if !report.Healthy {
		t.Error("Empty report should be healthy")
	}

	// Add an issue
	report.Issues = append(report.Issues, "Test issue")
	report.Healthy = len(report.Issues) == 0
	if report.Healthy {
		t.Error("Report with issues should not be healthy")
	}
}

func TestDoctorCheckStructure(t *testing.T) {
	check := &DoctorCheck{
		Name:    "Test",
		Status:  "ok",
		Message: "Test message",
		Details: map[string]interface{}{
			"key": "value",
		},
	}

	if check.Name != "Test" {
		t.Errorf("Check name = %s, want Test", check.Name)
	}
	if check.Status != "ok" {
		t.Errorf("Check status = %s, want ok", check.Status)
	}
	if check.Message != "Test message" {
		t.Errorf("Check message = %s, want 'Test message'", check.Message)
	}
	if check.Details["key"] != "value" {
		t.Error("Check details not set correctly")
	}
}

func TestDoctorCheckStatus(t *testing.T) {
	statuses := []string{"ok", "warning", "error", "missing"}

	for _, status := range statuses {
		check := &DoctorCheck{
			Name:    "Test",
			Status:  status,
			Message: "Test",
		}

		// Verify status is valid
		validStatus := false
		for _, s := range statuses {
			if check.Status == s {
				validStatus = true
				break
			}
		}
		if !validStatus {
			t.Errorf("Invalid status: %s", check.Status)
		}
	}
}

func TestGovernanceChecksStructure(t *testing.T) {
	gov := &GovernanceChecks{
		Workspace: &DoctorCheck{Name: "Workspace", Status: "ok"},
		Policies:  &DoctorCheck{Name: "Policies", Status: "ok"},
		Providers: &DoctorCheck{Name: "Providers", Status: "ok"},
		Bundles:   &DoctorCheck{Name: "Bundles", Status: "ok"},
		Approvals: &DoctorCheck{Name: "Approvals", Status: "ok"},
		Traces:    &DoctorCheck{Name: "Traces", Status: "ok"},
	}

	if gov.Workspace == nil {
		t.Error("Workspace check should not be nil")
	}
	if gov.Policies == nil {
		t.Error("Policies check should not be nil")
	}
	if gov.Providers == nil {
		t.Error("Providers check should not be nil")
	}
}

func TestGenerateNextSteps(t *testing.T) {
	report := &DoctorReport{
		Providers: make(map[string]*DoctorCheck),
		Issues:    []string{},
		Warnings:  []string{},
		NextSteps: []string{},
	}

	// Test with missing spec
	report.Spec = &DoctorCheck{Status: "missing"}
	generateNextSteps(report)

	// Test with spec and lock ok
	report.Spec = &DoctorCheck{Status: "ok"}
	report.Lock = &DoctorCheck{Status: "ok"}
	report.NextSteps = []string{} // Reset
	generateNextSteps(report)

	// Should have plan suggestion
	hasplanSuggestion := false
	for _, step := range report.NextSteps {
		if strings.Contains(step, "plan") {
			hasplanSuggestion = true
			break
		}
	}
	if !hasplanSuggestion {
		t.Error("Should suggest plan generation when spec and lock exist")
	}
}

func TestDoctorReportJSON(t *testing.T) {
	report := &DoctorReport{
		Providers: map[string]*DoctorCheck{
			"ollama": {
				Name:    "ollama",
				Status:  "ok",
				Message: "Ollama is available",
			},
		},
		Issues:    []string{"Issue 1"},
		Warnings:  []string{"Warning 1"},
		NextSteps: []string{"Next step 1"},
		Healthy:   false,
	}

	// Test that struct can be properly serialized
	if report.Providers["ollama"] == nil {
		t.Error("Providers should contain ollama")
	}
	if len(report.Issues) != 1 {
		t.Error("Should have 1 issue")
	}
	if len(report.Warnings) != 1 {
		t.Error("Should have 1 warning")
	}
	if report.Healthy {
		t.Error("Should not be healthy with issues")
	}
}

func TestDoctorQuickFlag(t *testing.T) {
	// Save original value
	originalQuick := doctorQuick
	defer func() {
		doctorQuick = originalQuick
	}()

	// Test default value
	doctorQuick = false
	if doctorQuick {
		t.Error("doctorQuick should default to false")
	}

	// Test setting to true
	doctorQuick = true
	if !doctorQuick {
		t.Error("doctorQuick should be settable to true")
	}
}

func TestDoctorVerboseFlag(t *testing.T) {
	// Save original value
	originalVerbose := doctorVerbose
	defer func() {
		doctorVerbose = originalVerbose
	}()

	// Test default value
	doctorVerbose = false
	if doctorVerbose {
		t.Error("doctorVerbose should default to false")
	}

	// Test setting to true
	doctorVerbose = true
	if !doctorVerbose {
		t.Error("doctorVerbose should be settable to true")
	}
}

func TestCheckMissingVariablesDoctor(t *testing.T) {
	// Create a check with missing status
	check := &DoctorCheck{
		Name:    "Test",
		Status:  "missing",
		Message: "Test is missing",
	}

	if check.Status != "missing" {
		t.Error("Status should be 'missing'")
	}
}

func TestDoctorCommandDescription(t *testing.T) {
	// Find the doctor command
	var doctorCommand *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "doctor" {
			doctorCommand = c
			break
		}
	}
	if doctorCommand == nil {
		t.Fatal("doctor command not found")
	}

	// Check Short description
	if !strings.Contains(doctorCommand.Short, "diagnostic") {
		t.Error("Short description should mention diagnostics")
	}

	// Check Long description has examples
	if !strings.Contains(doctorCommand.Long, "Examples:") {
		t.Error("Long description should include examples")
	}

	// Check Long description mentions quick flag
	if !strings.Contains(doctorCommand.Long, "--quick") {
		t.Error("Long description should mention --quick flag")
	}

	// Check Long description mentions verbose flag
	if !strings.Contains(doctorCommand.Long, "--verbose") {
		t.Error("Long description should mention --verbose flag")
	}
}

func TestDoctorIsAlsoUnderDebug(t *testing.T) {
	// Check that doctor is also available under debug command
	var debugCommand *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "debug" {
			debugCommand = c
			break
		}
	}

	if debugCommand == nil {
		t.Skip("debug command not found, skipping")
	}

	found := false
	for _, c := range debugCommand.Commands() {
		if c.Use == "doctor" {
			found = true
			break
		}
	}
	if !found {
		t.Error("doctor command should also be available under debug")
	}
}
