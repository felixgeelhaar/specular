package auto

import (
	"testing"

	"github.com/felixgeelhaar/specular/internal/router"
)

func TestNewOrchestrator(t *testing.T) {
	// Create a nil router for testing (we're just testing constructor)
	var r *router.Router = nil
	config := DefaultConfig()
	config.Goal = "Test goal"

	orchestrator := NewOrchestrator(r, config)

	if orchestrator == nil {
		t.Fatal("NewOrchestrator returned nil")
	}
	if orchestrator.router != r {
		t.Error("Orchestrator router was not set correctly")
	}
	if orchestrator.config.Goal != "Test goal" {
		t.Errorf("Orchestrator config.Goal = %s, want %s", orchestrator.config.Goal, "Test goal")
	}
	if orchestrator.parser == nil {
		t.Error("Orchestrator parser was not initialized")
	}
}

func TestNewOrchestrator_ParserInitialization(t *testing.T) {
	var r *router.Router = nil
	config := DefaultConfig()

	orchestrator := NewOrchestrator(r, config)

	// Verify parser is created with the same router
	if orchestrator.parser == nil {
		t.Fatal("Parser was not initialized")
	}
	if orchestrator.parser.router != r {
		t.Error("Parser was not initialized with correct router")
	}
}

func TestNewOrchestrator_ConfigPreservation(t *testing.T) {
	var r *router.Router = nil
	config := Config{
		Goal:            "Build a REST API",
		RequireApproval: false,
		MaxCostUSD:      10.0,
		MaxCostPerTask:  2.0,
		MaxRetries:      5,
		TimeoutMinutes:  60,
		Verbose:         true,
		DryRun:          true,
	}

	orchestrator := NewOrchestrator(r, config)

	// Verify all config fields are preserved
	if orchestrator.config.Goal != config.Goal {
		t.Errorf("Goal = %s, want %s", orchestrator.config.Goal, config.Goal)
	}
	if orchestrator.config.RequireApproval != config.RequireApproval {
		t.Errorf("RequireApproval = %v, want %v", orchestrator.config.RequireApproval, config.RequireApproval)
	}
	if orchestrator.config.MaxCostUSD != config.MaxCostUSD {
		t.Errorf("MaxCostUSD = %f, want %f", orchestrator.config.MaxCostUSD, config.MaxCostUSD)
	}
	if orchestrator.config.MaxCostPerTask != config.MaxCostPerTask {
		t.Errorf("MaxCostPerTask = %f, want %f", orchestrator.config.MaxCostPerTask, config.MaxCostPerTask)
	}
	if orchestrator.config.MaxRetries != config.MaxRetries {
		t.Errorf("MaxRetries = %d, want %d", orchestrator.config.MaxRetries, config.MaxRetries)
	}
	if orchestrator.config.TimeoutMinutes != config.TimeoutMinutes {
		t.Errorf("TimeoutMinutes = %d, want %d", orchestrator.config.TimeoutMinutes, config.TimeoutMinutes)
	}
	if orchestrator.config.Verbose != config.Verbose {
		t.Errorf("Verbose = %v, want %v", orchestrator.config.Verbose, config.Verbose)
	}
	if orchestrator.config.DryRun != config.DryRun {
		t.Errorf("DryRun = %v, want %v", orchestrator.config.DryRun, config.DryRun)
	}
}
