package auto

import (
	"testing"
	"time"
)

func TestNewActionPlan(t *testing.T) {
	goal := "implement user authentication"
	profile := "default"

	plan := NewActionPlan(goal, profile)

	if plan.Schema != "specular.auto.plan/v1" {
		t.Errorf("expected schema 'specular.auto.plan/v1', got %s", plan.Schema)
	}
	if plan.Goal != goal {
		t.Errorf("expected goal %q, got %q", goal, plan.Goal)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(plan.Steps))
	}
	if plan.Metadata.Profile != profile {
		t.Errorf("expected profile %q, got %q", profile, plan.Metadata.Profile)
	}
	if plan.Metadata.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", plan.Metadata.Version)
	}
}

func TestActionPlan_AddStep(t *testing.T) {
	plan := NewActionPlan("test goal", "default")

	step := ActionStep{
		Type:        StepTypeSpecUpdate,
		Description: "Generate spec",
	}

	plan.AddStep(step)

	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}

	// Check auto-generated ID
	if plan.Steps[0].ID == "" {
		t.Error("expected auto-generated ID")
	}

	// Check auto-initialized status
	if plan.Steps[0].Status != StepStatusPending {
		t.Errorf("expected status pending, got %s", plan.Steps[0].Status)
	}
}

func TestActionPlan_GetStep(t *testing.T) {
	plan := NewActionPlan("test goal", "default")

	step := ActionStep{
		ID:          "test-step",
		Type:        StepTypeSpecUpdate,
		Description: "Test step",
	}
	plan.AddStep(step)

	t.Run("existing step", func(t *testing.T) {
		retrieved, err := plan.GetStep("test-step")
		if err != nil {
			t.Fatalf("GetStep() error = %v", err)
		}
		if retrieved.ID != "test-step" {
			t.Errorf("expected ID 'test-step', got %s", retrieved.ID)
		}
	})

	t.Run("nonexistent step", func(t *testing.T) {
		_, err := plan.GetStep("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent step")
		}
	})
}

func TestActionPlan_UpdateStepStatus(t *testing.T) {
	plan := NewActionPlan("test goal", "default")

	step := ActionStep{
		ID:          "test-step",
		Type:        StepTypeSpecUpdate,
		Description: "Test step",
	}
	plan.AddStep(step)

	t.Run("update to in_progress", func(t *testing.T) {
		err := plan.UpdateStepStatus("test-step", StepStatusInProgress)
		if err != nil {
			t.Fatalf("UpdateStepStatus() error = %v", err)
		}

		retrieved, _ := plan.GetStep("test-step")
		if retrieved.Status != StepStatusInProgress {
			t.Errorf("expected status in_progress, got %s", retrieved.Status)
		}
		if retrieved.StartedAt == nil {
			t.Error("expected StartedAt to be set")
		}
	})

	t.Run("update to completed", func(t *testing.T) {
		err := plan.UpdateStepStatus("test-step", StepStatusCompleted)
		if err != nil {
			t.Fatalf("UpdateStepStatus() error = %v", err)
		}

		retrieved, _ := plan.GetStep("test-step")
		if retrieved.Status != StepStatusCompleted {
			t.Errorf("expected status completed, got %s", retrieved.Status)
		}
		if retrieved.CompletedAt == nil {
			t.Error("expected CompletedAt to be set")
		}
	})

	t.Run("nonexistent step", func(t *testing.T) {
		err := plan.UpdateStepStatus("nonexistent", StepStatusCompleted)
		if err == nil {
			t.Error("expected error for nonexistent step")
		}
	})
}

func TestActionPlan_GetPendingSteps(t *testing.T) {
	plan := NewActionPlan("test goal", "default")

	plan.AddStep(ActionStep{
		ID:          "step-1",
		Type:        StepTypeSpecUpdate,
		Description: "Step 1",
		Status:      StepStatusPending,
	})
	plan.AddStep(ActionStep{
		ID:          "step-2",
		Type:        StepTypeSpecLock,
		Description: "Step 2",
		Status:      StepStatusCompleted,
	})
	plan.AddStep(ActionStep{
		ID:          "step-3",
		Type:        StepTypePlanGen,
		Description: "Step 3",
		Status:      StepStatusPending,
	})

	pending := plan.GetPendingSteps()
	if len(pending) != 2 {
		t.Errorf("expected 2 pending steps, got %d", len(pending))
	}
}

func TestActionPlan_GetCompletedSteps(t *testing.T) {
	plan := NewActionPlan("test goal", "default")

	plan.AddStep(ActionStep{
		ID:          "step-1",
		Type:        StepTypeSpecUpdate,
		Description: "Step 1",
		Status:      StepStatusCompleted,
	})
	plan.AddStep(ActionStep{
		ID:          "step-2",
		Type:        StepTypeSpecLock,
		Description: "Step 2",
		Status:      StepStatusPending,
	})

	completed := plan.GetCompletedSteps()
	if len(completed) != 1 {
		t.Errorf("expected 1 completed step, got %d", len(completed))
	}
}

func TestActionPlan_GetFailedSteps(t *testing.T) {
	plan := NewActionPlan("test goal", "default")

	plan.AddStep(ActionStep{
		ID:          "step-1",
		Type:        StepTypeSpecUpdate,
		Description: "Step 1",
		Status:      StepStatusFailed,
		Error:       "test error",
	})
	plan.AddStep(ActionStep{
		ID:          "step-2",
		Type:        StepTypeSpecLock,
		Description: "Step 2",
		Status:      StepStatusCompleted,
	})

	failed := plan.GetFailedSteps()
	if len(failed) != 1 {
		t.Errorf("expected 1 failed step, got %d", len(failed))
	}
	if failed[0].Error != "test error" {
		t.Errorf("expected error 'test error', got %s", failed[0].Error)
	}
}

func TestActionPlan_IsComplete(t *testing.T) {
	t.Run("all completed", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecUpdate,
			Description: "Step 1",
			Status:      StepStatusCompleted,
		})
		plan.AddStep(ActionStep{
			ID:          "step-2",
			Type:        StepTypeSpecLock,
			Description: "Step 2",
			Status:      StepStatusCompleted,
		})

		if !plan.IsComplete() {
			t.Error("expected plan to be complete")
		}
	})

	t.Run("some pending", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecUpdate,
			Description: "Step 1",
			Status:      StepStatusCompleted,
		})
		plan.AddStep(ActionStep{
			ID:          "step-2",
			Type:        StepTypeSpecLock,
			Description: "Step 2",
			Status:      StepStatusPending,
		})

		if plan.IsComplete() {
			t.Error("expected plan to not be complete")
		}
	})

	t.Run("with skipped steps", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecUpdate,
			Description: "Step 1",
			Status:      StepStatusCompleted,
		})
		plan.AddStep(ActionStep{
			ID:          "step-2",
			Type:        StepTypeSpecLock,
			Description: "Step 2",
			Status:      StepStatusSkipped,
		})

		if !plan.IsComplete() {
			t.Error("expected plan with skipped steps to be complete")
		}
	})
}

func TestActionPlan_HasFailedSteps(t *testing.T) {
	t.Run("with failed steps", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecUpdate,
			Description: "Step 1",
			Status:      StepStatusFailed,
		})

		if !plan.HasFailedSteps() {
			t.Error("expected plan to have failed steps")
		}
	})

	t.Run("without failed steps", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecUpdate,
			Description: "Step 1",
			Status:      StepStatusCompleted,
		})

		if plan.HasFailedSteps() {
			t.Error("expected plan to not have failed steps")
		}
	})
}

func TestActionPlan_GetNextStep(t *testing.T) {
	t.Run("simple sequence", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecUpdate,
			Description: "Step 1",
			Status:      StepStatusCompleted,
		})
		plan.AddStep(ActionStep{
			ID:           "step-2",
			Type:         StepTypeSpecLock,
			Description:  "Step 2",
			Status:       StepStatusPending,
			Dependencies: []string{"step-1"},
		})

		next, err := plan.GetNextStep()
		if err != nil {
			t.Fatalf("GetNextStep() error = %v", err)
		}
		if next.ID != "step-2" {
			t.Errorf("expected next step 'step-2', got %s", next.ID)
		}
	})

	t.Run("with unmet dependencies", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecUpdate,
			Description: "Step 1",
			Status:      StepStatusPending,
		})
		plan.AddStep(ActionStep{
			ID:           "step-2",
			Type:         StepTypeSpecLock,
			Description:  "Step 2",
			Status:       StepStatusPending,
			Dependencies: []string{"step-1"},
		})

		next, err := plan.GetNextStep()
		if err != nil {
			t.Fatalf("GetNextStep() error = %v", err)
		}
		if next.ID != "step-1" {
			t.Errorf("expected next step 'step-1', got %s", next.ID)
		}
	})

	t.Run("all steps completed", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecUpdate,
			Description: "Step 1",
			Status:      StepStatusCompleted,
		})

		_, err := plan.GetNextStep()
		if err == nil {
			t.Error("expected error when all steps completed")
		}
	})
}

func TestActionPlan_Validate(t *testing.T) {
	t.Run("valid plan", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecUpdate,
			Description: "Step 1",
		})

		err := plan.Validate()
		if err != nil {
			t.Errorf("Validate() error = %v", err)
		}
	})

	t.Run("missing schema", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.Schema = ""
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecUpdate,
			Description: "Step 1",
		})

		err := plan.Validate()
		if err == nil {
			t.Error("expected error for missing schema")
		}
	})

	t.Run("missing goal", func(t *testing.T) {
		plan := NewActionPlan("", "default")
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecUpdate,
			Description: "Step 1",
		})

		err := plan.Validate()
		if err == nil {
			t.Error("expected error for missing goal")
		}
	})

	t.Run("no steps", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")

		err := plan.Validate()
		if err == nil {
			t.Error("expected error for no steps")
		}
	})

	t.Run("invalid step type", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        "invalid:type",
			Description: "Step 1",
		})

		err := plan.Validate()
		if err == nil {
			t.Error("expected error for invalid step type")
		}
	})

	t.Run("duplicate step ID", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecUpdate,
			Description: "Step 1",
		})
		plan.AddStep(ActionStep{
			ID:          "step-1",
			Type:        StepTypeSpecLock,
			Description: "Step 2",
		})

		err := plan.Validate()
		if err == nil {
			t.Error("expected error for duplicate step ID")
		}
	})

	t.Run("invalid dependency", func(t *testing.T) {
		plan := NewActionPlan("test goal", "default")
		plan.AddStep(ActionStep{
			ID:           "step-1",
			Type:         StepTypeSpecUpdate,
			Description:  "Step 1",
			Dependencies: []string{"nonexistent"},
		})

		err := plan.Validate()
		if err == nil {
			t.Error("expected error for invalid dependency")
		}
	})

	t.Run("circular dependency", func(t *testing.T) {
		plan := &ActionPlan{
			Schema: "specular.auto.plan/v1",
			Goal:   "test goal",
			Steps: []ActionStep{
				{
					ID:           "step-1",
					Type:         StepTypeSpecUpdate,
					Description:  "Step 1",
					Dependencies: []string{"step-2"},
				},
				{
					ID:           "step-2",
					Type:         StepTypeSpecLock,
					Description:  "Step 2",
					Dependencies: []string{"step-1"},
				},
			},
			Metadata: PlanMetadata{
				CreatedAt: time.Now(),
				Version:   "1.0.0",
			},
		}

		err := plan.Validate()
		if err == nil {
			t.Error("expected error for circular dependency")
		}
	})
}

func TestCreateDefaultActionPlan(t *testing.T) {
	goal := "implement user authentication"
	profile := "default"

	plan := CreateDefaultActionPlan(goal, profile)

	if plan.Goal != goal {
		t.Errorf("expected goal %q, got %q", goal, plan.Goal)
	}

	if len(plan.Steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(plan.Steps))
	}

	// Check step types in order
	expectedTypes := []StepType{
		StepTypeSpecUpdate,
		StepTypeSpecLock,
		StepTypePlanGen,
		StepTypeBuildRun,
	}

	for i, expectedType := range expectedTypes {
		if plan.Steps[i].Type != expectedType {
			t.Errorf("step %d: expected type %s, got %s", i, expectedType, plan.Steps[i].Type)
		}
	}

	// Check that critical steps require approval
	if !plan.Steps[1].RequiresApproval {
		t.Error("expected spec:lock to require approval")
	}
	if !plan.Steps[3].RequiresApproval {
		t.Error("expected build:run to require approval")
	}

	// Validate dependencies
	if len(plan.Steps[0].Dependencies) != 0 {
		t.Error("expected step-1 to have no dependencies")
	}
	if len(plan.Steps[1].Dependencies) != 1 || plan.Steps[1].Dependencies[0] != "step-1" {
		t.Error("expected step-2 to depend on step-1")
	}
	if len(plan.Steps[2].Dependencies) != 1 || plan.Steps[2].Dependencies[0] != "step-2" {
		t.Error("expected step-3 to depend on step-2")
	}
	if len(plan.Steps[3].Dependencies) != 1 || plan.Steps[3].Dependencies[0] != "step-3" {
		t.Error("expected step-4 to depend on step-3")
	}

	// Validate the plan
	if err := plan.Validate(); err != nil {
		t.Errorf("default plan validation failed: %v", err)
	}
}

func TestStepTypeConstants(t *testing.T) {
	// Verify step type constants match expected values
	if StepTypeSpecUpdate != "spec:update" {
		t.Errorf("expected StepTypeSpecUpdate to be 'spec:update', got %s", StepTypeSpecUpdate)
	}
	if StepTypeSpecLock != "spec:lock" {
		t.Errorf("expected StepTypeSpecLock to be 'spec:lock', got %s", StepTypeSpecLock)
	}
	if StepTypePlanGen != "plan:gen" {
		t.Errorf("expected StepTypePlanGen to be 'plan:gen', got %s", StepTypePlanGen)
	}
	if StepTypeBuildRun != "build:run" {
		t.Errorf("expected StepTypeBuildRun to be 'build:run', got %s", StepTypeBuildRun)
	}
}

func TestStepStatusConstants(t *testing.T) {
	// Verify step status constants match expected values
	if StepStatusPending != "pending" {
		t.Errorf("expected StepStatusPending to be 'pending', got %s", StepStatusPending)
	}
	if StepStatusInProgress != "in_progress" {
		t.Errorf("expected StepStatusInProgress to be 'in_progress', got %s", StepStatusInProgress)
	}
	if StepStatusCompleted != "completed" {
		t.Errorf("expected StepStatusCompleted to be 'completed', got %s", StepStatusCompleted)
	}
	if StepStatusFailed != "failed" {
		t.Errorf("expected StepStatusFailed to be 'failed', got %s", StepStatusFailed)
	}
	if StepStatusSkipped != "skipped" {
		t.Errorf("expected StepStatusSkipped to be 'skipped', got %s", StepStatusSkipped)
	}
}
