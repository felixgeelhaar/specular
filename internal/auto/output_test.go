package auto

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewAutoOutput(t *testing.T) {
	output := NewAutoOutput("test goal", "test-profile")

	if output.Schema != "specular.auto.output/v1" {
		t.Errorf("expected schema 'specular.auto.output/v1', got %q", output.Schema)
	}

	if output.Goal != "test goal" {
		t.Errorf("expected goal 'test goal', got %q", output.Goal)
	}

	if output.Status != "in_progress" {
		t.Errorf("expected status 'in_progress', got %q", output.Status)
	}

	if output.Audit.Profile != "test-profile" {
		t.Errorf("expected profile 'test-profile', got %q", output.Audit.Profile)
	}

	if len(output.Steps) != 0 {
		t.Errorf("expected empty steps slice, got %d steps", len(output.Steps))
	}

	if len(output.Artifacts) != 0 {
		t.Errorf("expected empty artifacts slice, got %d artifacts", len(output.Artifacts))
	}

	if output.Metrics.StepsExecuted != 0 {
		t.Errorf("expected 0 steps executed, got %d", output.Metrics.StepsExecuted)
	}
}

func TestAddStepResult(t *testing.T) {
	output := NewAutoOutput("test goal", "default")

	step1 := StepResult{
		ID:          "step-1",
		Type:        "spec:update",
		Status:      "completed",
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
		Duration:    5 * time.Second,
		CostUSD:     0.50,
	}

	output.AddStepResult(step1)

	if len(output.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(output.Steps))
	}

	if output.Steps[0].ID != "step-1" {
		t.Errorf("expected step ID 'step-1', got %q", output.Steps[0].ID)
	}

	if output.Metrics.StepsExecuted != 1 {
		t.Errorf("expected 1 step executed, got %d", output.Metrics.StepsExecuted)
	}

	if output.Metrics.TotalCost != 0.50 {
		t.Errorf("expected total cost $0.50, got $%.2f", output.Metrics.TotalCost)
	}
}

func TestAddStepResult_Failed(t *testing.T) {
	output := NewAutoOutput("test goal", "default")

	failedStep := StepResult{
		ID:          "step-1",
		Type:        "build:run",
		Status:      "failed",
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
		Error:       "build failed",
		CostUSD:     1.00,
	}

	output.AddStepResult(failedStep)

	if output.Metrics.StepsFailed != 1 {
		t.Errorf("expected 1 failed step, got %d", output.Metrics.StepsFailed)
	}

	if output.Metrics.StepsExecuted != 1 {
		t.Errorf("expected 1 executed step, got %d", output.Metrics.StepsExecuted)
	}
}

func TestAddStepResult_Skipped(t *testing.T) {
	output := NewAutoOutput("test goal", "default")

	skippedStep := StepResult{
		ID:     "step-1",
		Type:   "spec:lock",
		Status: "skipped",
	}

	output.AddStepResult(skippedStep)

	if output.Metrics.StepsSkipped != 1 {
		t.Errorf("expected 1 skipped step, got %d", output.Metrics.StepsSkipped)
	}
}

func TestAddArtifact(t *testing.T) {
	output := NewAutoOutput("test goal", "default")

	artifact := ArtifactInfo{
		Path:      "spec.yaml",
		Type:      "spec",
		Size:      1024,
		Hash:      "abc123",
		CreatedAt: time.Now(),
	}

	output.AddArtifact(artifact)

	if len(output.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(output.Artifacts))
	}

	if output.Artifacts[0].Path != "spec.yaml" {
		t.Errorf("expected artifact path 'spec.yaml', got %q", output.Artifacts[0].Path)
	}
}

func TestAddApproval(t *testing.T) {
	output := NewAutoOutput("test goal", "default")

	approval := ApprovalEvent{
		StepID:    "step-1",
		Timestamp: time.Now(),
		Approved:  true,
		Reason:    "critical step",
		User:      "john.doe",
	}

	output.AddApproval(approval)

	if len(output.Audit.Approvals) != 1 {
		t.Fatalf("expected 1 approval event, got %d", len(output.Audit.Approvals))
	}

	if output.Audit.Approvals[0].StepID != "step-1" {
		t.Errorf("expected approval for 'step-1', got %q", output.Audit.Approvals[0].StepID)
	}

	if !output.Audit.Approvals[0].Approved {
		t.Error("expected approval to be true")
	}
}

func TestAddPolicy(t *testing.T) {
	output := NewAutoOutput("test goal", "default")

	policy := PolicyEvent{
		StepID:      "step-1",
		Timestamp:   time.Now(),
		CheckerName: "cost_limit",
		Allowed:     true,
		Warnings:    []string{"approaching budget limit"},
	}

	output.AddPolicy(policy)

	if len(output.Audit.Policies) != 1 {
		t.Fatalf("expected 1 policy event, got %d", len(output.Audit.Policies))
	}

	if output.Audit.Policies[0].CheckerName != "cost_limit" {
		t.Errorf("expected checker 'cost_limit', got %q", output.Audit.Policies[0].CheckerName)
	}

	if output.Metrics.PolicyViolations != 0 {
		t.Errorf("expected 0 violations, got %d", output.Metrics.PolicyViolations)
	}
}

func TestAddPolicy_Denied(t *testing.T) {
	output := NewAutoOutput("test goal", "default")

	policy := PolicyEvent{
		StepID:      "step-1",
		Timestamp:   time.Now(),
		CheckerName: "cost_limit",
		Allowed:     false,
		Reason:      "budget exceeded",
	}

	output.AddPolicy(policy)

	if output.Metrics.PolicyViolations != 1 {
		t.Errorf("expected 1 policy violation, got %d", output.Metrics.PolicyViolations)
	}
}

func TestSetCompleted(t *testing.T) {
	output := NewAutoOutput("test goal", "default")
	time.Sleep(10 * time.Millisecond) // Small delay to get measurable duration

	output.SetCompleted()

	if output.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", output.Status)
	}

	if output.Audit.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be set")
	}

	if output.Metrics.TotalDuration == 0 {
		t.Error("expected non-zero TotalDuration")
	}

	if output.Audit.CompletedAt.Before(output.Audit.StartedAt) {
		t.Error("CompletedAt should be after StartedAt")
	}
}

func TestSetFailed(t *testing.T) {
	output := NewAutoOutput("test goal", "default")
	time.Sleep(10 * time.Millisecond)

	output.SetFailed()

	if output.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", output.Status)
	}

	if output.Audit.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be set")
	}
}

func TestSetPartial(t *testing.T) {
	output := NewAutoOutput("test goal", "default")
	time.Sleep(10 * time.Millisecond)

	output.SetPartial()

	if output.Status != "partial" {
		t.Errorf("expected status 'partial', got %q", output.Status)
	}

	if output.Audit.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be set")
	}
}

func TestSetCheckpointID(t *testing.T) {
	output := NewAutoOutput("test goal", "default")
	output.SetCheckpointID("auto-1234567890")

	if output.Audit.CheckpointID != "auto-1234567890" {
		t.Errorf("expected checkpoint ID 'auto-1234567890', got %q", output.Audit.CheckpointID)
	}
}

func TestSetUser(t *testing.T) {
	output := NewAutoOutput("test goal", "default")
	output.SetUser("alice@example.com")

	if output.Audit.User != "alice@example.com" {
		t.Errorf("expected user 'alice@example.com', got %q", output.Audit.User)
	}
}

func TestSetHostname(t *testing.T) {
	output := NewAutoOutput("test goal", "default")
	output.SetHostname("ci-runner-01")

	if output.Audit.Hostname != "ci-runner-01" {
		t.Errorf("expected hostname 'ci-runner-01', got %q", output.Audit.Hostname)
	}
}

func TestSetVersion(t *testing.T) {
	output := NewAutoOutput("test goal", "default")
	output.SetVersion("v1.4.0")

	if output.Audit.Version != "v1.4.0" {
		t.Errorf("expected version 'v1.4.0', got %q", output.Audit.Version)
	}
}

func TestToJSON(t *testing.T) {
	output := NewAutoOutput("test goal", "default")
	output.AddStepResult(StepResult{
		ID:          "step-1",
		Type:        "spec:update",
		Status:      "completed",
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
		CostUSD:     0.50,
	})
	output.SetCompleted()

	jsonData, err := output.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("expected non-empty JSON output")
	}

	// Verify it's valid JSON
	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("generated JSON is invalid: %v", err)
	}

	// Verify schema field
	schema, ok := decoded["schema"].(string)
	if !ok || schema != "specular.auto.output/v1" {
		t.Errorf("expected schema 'specular.auto.output/v1', got %v", decoded["schema"])
	}

	// Verify goal field
	goal, ok := decoded["goal"].(string)
	if !ok || goal != "test goal" {
		t.Errorf("expected goal 'test goal', got %v", decoded["goal"])
	}

	// Verify status field
	status, ok := decoded["status"].(string)
	if !ok || status != "completed" {
		t.Errorf("expected status 'completed', got %v", decoded["status"])
	}
}

func TestFromJSON(t *testing.T) {
	original := NewAutoOutput("test goal", "default")
	original.AddStepResult(StepResult{
		ID:      "step-1",
		Type:    "spec:update",
		Status:  "completed",
		CostUSD: 0.50,
	})
	original.SetCompleted()

	jsonData, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	decoded, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if decoded.Schema != original.Schema {
		t.Errorf("schema mismatch: expected %q, got %q", original.Schema, decoded.Schema)
	}

	if decoded.Goal != original.Goal {
		t.Errorf("goal mismatch: expected %q, got %q", original.Goal, decoded.Goal)
	}

	if decoded.Status != original.Status {
		t.Errorf("status mismatch: expected %q, got %q", original.Status, decoded.Status)
	}

	if len(decoded.Steps) != len(original.Steps) {
		t.Errorf("steps count mismatch: expected %d, got %d", len(original.Steps), len(decoded.Steps))
	}

	if decoded.Metrics.StepsExecuted != original.Metrics.StepsExecuted {
		t.Errorf("metrics mismatch: expected %d steps executed, got %d",
			original.Metrics.StepsExecuted, decoded.Metrics.StepsExecuted)
	}
}

func TestCompleteWorkflow(t *testing.T) {
	// Simulate a complete workflow
	output := NewAutoOutput("Build REST API", "ci")
	output.SetCheckpointID("auto-1234567890")
	output.SetUser("ci-bot")
	output.SetHostname("ci-runner-01")
	output.SetVersion("v1.4.0")

	// Add steps
	steps := []StepResult{
		{
			ID:          "step-1",
			Type:        "spec:update",
			Status:      "completed",
			StartedAt:   time.Now(),
			CompletedAt: time.Now().Add(5 * time.Second),
			Duration:    5 * time.Second,
			CostUSD:     0.50,
		},
		{
			ID:          "step-2",
			Type:        "spec:lock",
			Status:      "completed",
			StartedAt:   time.Now().Add(5 * time.Second),
			CompletedAt: time.Now().Add(6 * time.Second),
			Duration:    1 * time.Second,
			CostUSD:     0.01,
		},
		{
			ID:          "step-3",
			Type:        "plan:gen",
			Status:      "completed",
			StartedAt:   time.Now().Add(6 * time.Second),
			CompletedAt: time.Now().Add(9 * time.Second),
			Duration:    3 * time.Second,
			CostUSD:     0.30,
		},
		{
			ID:          "step-4",
			Type:        "build:run",
			Status:      "completed",
			StartedAt:   time.Now().Add(9 * time.Second),
			CompletedAt: time.Now().Add(19 * time.Second),
			Duration:    10 * time.Second,
			CostUSD:     1.00,
		},
	}

	for _, step := range steps {
		output.AddStepResult(step)

		// Add policy event for each step
		output.AddPolicy(PolicyEvent{
			StepID:      step.ID,
			Timestamp:   step.StartedAt,
			CheckerName: "cost_limit",
			Allowed:     true,
		})
	}

	// Add artifacts
	output.AddArtifact(ArtifactInfo{
		Path:      "spec.yaml",
		Type:      "spec",
		Size:      2048,
		Hash:      "abc123",
		CreatedAt: time.Now(),
	})

	output.AddArtifact(ArtifactInfo{
		Path:      "spec.lock",
		Type:      "lock",
		Size:      512,
		Hash:      "def456",
		CreatedAt: time.Now(),
	})

	// Add approval
	output.AddApproval(ApprovalEvent{
		StepID:    "step-4",
		Timestamp: time.Now(),
		Approved:  true,
		Reason:    "build step requires approval",
		User:      "ci-bot",
	})

	output.SetCompleted()

	// Verify final state
	if output.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", output.Status)
	}

	if output.Metrics.StepsExecuted != 4 {
		t.Errorf("expected 4 steps executed, got %d", output.Metrics.StepsExecuted)
	}

	expectedCost := 0.50 + 0.01 + 0.30 + 1.00
	if output.Metrics.TotalCost != expectedCost {
		t.Errorf("expected total cost $%.2f, got $%.2f", expectedCost, output.Metrics.TotalCost)
	}

	if len(output.Artifacts) != 2 {
		t.Errorf("expected 2 artifacts, got %d", len(output.Artifacts))
	}

	if len(output.Audit.Approvals) != 1 {
		t.Errorf("expected 1 approval, got %d", len(output.Audit.Approvals))
	}

	if len(output.Audit.Policies) != 4 {
		t.Errorf("expected 4 policy events, got %d", len(output.Audit.Policies))
	}

	// Verify JSON serialization
	jsonData, err := output.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify round-trip
	decoded, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if decoded.Metrics.StepsExecuted != 4 {
		t.Errorf("round-trip failed: expected 4 steps executed, got %d", decoded.Metrics.StepsExecuted)
	}
}

func TestJSONSchema_RequiredFields(t *testing.T) {
	output := NewAutoOutput("test goal", "default")
	jsonData, err := output.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Verify required fields are present
	requiredFields := []string{"schema", "goal", "status", "steps", "artifacts", "metrics", "audit"}
	for _, field := range requiredFields {
		if _, exists := decoded[field]; !exists {
			t.Errorf("required field %q missing from JSON output", field)
		}
	}

	// Verify metrics fields
	metrics, ok := decoded["metrics"].(map[string]interface{})
	if !ok {
		t.Fatal("metrics field is not an object")
	}

	metricsFields := []string{"totalDuration", "totalCost", "stepsExecuted", "stepsFailed", "stepsSkipped", "policyViolations"}
	for _, field := range metricsFields {
		if _, exists := metrics[field]; !exists {
			t.Errorf("required metrics field %q missing", field)
		}
	}

	// Verify audit fields
	audit, ok := decoded["audit"].(map[string]interface{})
	if !ok {
		t.Fatal("audit field is not an object")
	}

	auditFields := []string{"checkpointId", "profile", "startedAt", "completedAt", "approvals", "policies"}
	for _, field := range auditFields {
		if _, exists := audit[field]; !exists {
			t.Errorf("required audit field %q missing", field)
		}
	}
}
