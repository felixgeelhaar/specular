package auto

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"RequireApproval", config.RequireApproval, true},
		{"MaxCostUSD", config.MaxCostUSD, 5.0},
		{"MaxCostPerTask", config.MaxCostPerTask, 1.0},
		{"MaxRetries", config.MaxRetries, 3},
		{"RetryDelay", config.RetryDelay, time.Second * 2},
		{"TimeoutMinutes", config.TimeoutMinutes, 30},
		{"TaskTimeout", config.TaskTimeout, time.Minute * 5},
		{"PolicyPath", config.PolicyPath, ".specular/policy.yaml"},
		{"FallbackToManual", config.FallbackToManual, true},
		{"Verbose", config.Verbose, false},
		{"DryRun", config.DryRun, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("DefaultConfig().%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestConfig_CustomValues(t *testing.T) {
	config := Config{
		Goal:             "Test goal",
		RequireApproval:  false,
		MaxCostUSD:       10.0,
		MaxCostPerTask:   2.0,
		MaxRetries:       5,
		RetryDelay:       time.Second * 5,
		TimeoutMinutes:   60,
		TaskTimeout:      time.Minute * 10,
		PolicyPath:       "custom/policy.yaml",
		FallbackToManual: false,
		Verbose:          true,
		DryRun:           true,
	}

	if config.Goal != "Test goal" {
		t.Errorf("Goal = %s, want %s", config.Goal, "Test goal")
	}
	if config.RequireApproval != false {
		t.Error("RequireApproval should be false")
	}
	if config.MaxCostUSD != 10.0 {
		t.Errorf("MaxCostUSD = %f, want %f", config.MaxCostUSD, 10.0)
	}
	if config.DryRun != true {
		t.Error("DryRun should be true")
	}
}

func TestResult_InitialState(t *testing.T) {
	result := &Result{
		Success: false,
		Errors:  []error{},
	}

	if result.Success {
		t.Error("Result.Success should be false initially")
	}
	if len(result.Errors) != 0 {
		t.Errorf("Result.Errors length = %d, want 0", len(result.Errors))
	}
	if result.TasksExecuted != 0 {
		t.Errorf("Result.TasksExecuted = %d, want 0", result.TasksExecuted)
	}
	if result.TasksFailed != 0 {
		t.Errorf("Result.TasksFailed = %d, want 0", result.TasksFailed)
	}
	if result.TotalCost != 0.0 {
		t.Errorf("Result.TotalCost = %f, want 0.0", result.TotalCost)
	}
}
