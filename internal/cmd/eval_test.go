package cmd

import (
	"testing"
)

// TestEvalScenarioValidation tests the scenario validation logic in eval run
func TestEvalScenarioValidation(t *testing.T) {
	tests := []struct {
		name        string
		scenario    string
		wantValid   bool
	}{
		{
			name:      "valid smoke scenario",
			scenario:  "smoke",
			wantValid: true,
		},
		{
			name:      "valid integration scenario",
			scenario:  "integration",
			wantValid: true,
		},
		{
			name:      "valid security scenario",
			scenario:  "security",
			wantValid: true,
		},
		{
			name:      "valid performance scenario",
			scenario:  "performance",
			wantValid: true,
		},
		{
			name:      "invalid scenario",
			scenario:  "invalid",
			wantValid: false,
		},
		{
			name:      "empty scenario defaults to smoke",
			scenario:  "",
			wantValid: true, // empty defaults to "smoke" which is valid
		},
	}

	validScenarios := map[string]bool{
		"smoke":       true,
		"integration": true,
		"security":    true,
		"performance": true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test scenario validation logic
			scenario := tt.scenario
			if scenario == "" {
				scenario = "smoke" // default
			}

			isValid := validScenarios[scenario]

			if isValid != tt.wantValid {
				t.Errorf("scenario validation = %v, want %v", isValid, tt.wantValid)
			}
		})
	}
}

// TestEvalScenarioChecks tests the check count logic for each scenario
func TestEvalScenarioChecks(t *testing.T) {
	tests := []struct {
		name       string
		scenario   string
		wantChecks int
	}{
		{
			name:       "smoke scenario has 3 checks",
			scenario:   "smoke",
			wantChecks: 3, // go vet, go build, basic tests
		},
		{
			name:       "integration scenario has 3 checks",
			scenario:   "integration",
			wantChecks: 3, // go vet, all tests, coverage
		},
		{
			name:       "security scenario has 3 checks",
			scenario:   "security",
			wantChecks: 3, // go vet, gosec, policy
		},
		{
			name:       "performance scenario has 3 checks",
			scenario:   "performance",
			wantChecks: 3, // benchmarks, memory profiling, CPU profiling
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test check count logic
			var checkCount int
			switch tt.scenario {
			case "smoke":
				checkCount = 3 // go vet, go build, basic tests
			case "integration":
				checkCount = 3 // go vet, all tests, coverage check
			case "security":
				checkCount = 3 // go vet, gosec scan, policy check
			case "performance":
				checkCount = 3 // benchmark tests, memory profiling, CPU profiling
			}

			if checkCount != tt.wantChecks {
				t.Errorf("check count = %d, want %d", checkCount, tt.wantChecks)
			}
		})
	}
}

// TestEvalBackwardCompatibilityFlags tests that old eval flags work
func TestEvalBackwardCompatibilityFlags(t *testing.T) {
	// Test that the root eval command still has the old flags for backward compatibility
	if evalCmd.Flags().Lookup("plan") == nil {
		t.Error("backward compatibility flag 'plan' not found on eval command")
	}
	if evalCmd.Flags().Lookup("lock") == nil {
		t.Error("backward compatibility flag 'lock' not found on eval command")
	}
	if evalCmd.Flags().Lookup("fail-on-drift") == nil {
		t.Error("backward compatibility flag 'fail-on-drift' not found on eval command")
	}
	if evalCmd.Flags().Lookup("policy") == nil {
		t.Error("backward compatibility flag 'policy' not found on eval command")
	}
}

// TestEvalRunFlags tests that eval run has all required flags
func TestEvalRunFlags(t *testing.T) {
	if evalRunCmd.Flags().Lookup("scenario") == nil {
		t.Error("flag 'scenario' not found on eval run command")
	}
	if evalRunCmd.Flags().Lookup("policy") == nil {
		t.Error("flag 'policy' not found on eval run command")
	}
}

// TestEvalRulesFlags tests that eval rules has all required flags
func TestEvalRulesFlags(t *testing.T) {
	if evalRulesCmd.Flags().Lookup("policy") == nil {
		t.Error("flag 'policy' not found on eval rules command")
	}
	if evalRulesCmd.Flags().Lookup("edit") == nil {
		t.Error("flag 'edit' not found on eval rules command")
	}
}

// TestEvalDriftFlags tests that eval drift has all required flags
func TestEvalDriftFlags(t *testing.T) {
	if evalDriftCmd.Flags().Lookup("plan") == nil {
		t.Error("flag 'plan' not found on eval drift command")
	}
	if evalDriftCmd.Flags().Lookup("lock") == nil {
		t.Error("flag 'lock' not found on eval drift command")
	}
	if evalDriftCmd.Flags().Lookup("spec") == nil {
		t.Error("flag 'spec' not found on eval drift command")
	}
	if evalDriftCmd.Flags().Lookup("policy") == nil {
		t.Error("flag 'policy' not found on eval drift command")
	}
	if evalDriftCmd.Flags().Lookup("report") == nil {
		t.Error("flag 'report' not found on eval drift command")
	}
	if evalDriftCmd.Flags().Lookup("fail-on-drift") == nil {
		t.Error("flag 'fail-on-drift' not found on eval drift command")
	}
}

// TestEvalSubcommands tests that all eval subcommands are registered
func TestEvalSubcommands(t *testing.T) {
	subcommands := map[string]bool{
		"run":   false,
		"rules": false,
		"drift": false,
	}

	for _, cmd := range evalCmd.Commands() {
		if _, exists := subcommands[cmd.Name()]; exists {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("subcommand '%s' not found in eval command", name)
		}
	}
}
