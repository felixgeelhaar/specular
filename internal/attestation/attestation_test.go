package attestation

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAttestationSerialization(t *testing.T) {
	att := &Attestation{
		Version:    "1.0",
		WorkflowID: "test-workflow",
		Goal:       "test goal",
		StartTime:  time.Now().Add(-1 * time.Hour),
		EndTime:    time.Now(),
		Duration:   "1h",
		Status:     "success",
		Provenance: Provenance{
			Hostname:        "test-host",
			Platform:        "linux",
			Arch:            "amd64",
			SpecularVersion: "1.0.0",
			Profile:         "default",
			TotalCost:       1.23,
			TasksExecuted:   5,
			TasksFailed:     0,
		},
		PlanHash:   "abc123",
		OutputHash: "def456",
		SignedAt:   time.Now(),
		SignedBy:   "test@example.com",
		Signature:  "signature",
		PublicKey:  "publickey",
	}

	// Test ToJSON
	data, err := att.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("ToJSON produced invalid JSON: %v", err)
	}

	// Test FromJSON
	att2, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	// Verify fields
	if att2.Version != att.Version {
		t.Errorf("Version mismatch: %s != %s", att2.Version, att.Version)
	}
	if att2.WorkflowID != att.WorkflowID {
		t.Errorf("WorkflowID mismatch: %s != %s", att2.WorkflowID, att.WorkflowID)
	}
	if att2.Goal != att.Goal {
		t.Errorf("Goal mismatch: %s != %s", att2.Goal, att.Goal)
	}
	if att2.Status != att.Status {
		t.Errorf("Status mismatch: %s != %s", att2.Status, att.Status)
	}
	if att2.PlanHash != att.PlanHash {
		t.Errorf("PlanHash mismatch: %s != %s", att2.PlanHash, att.PlanHash)
	}
	if att2.OutputHash != att.OutputHash {
		t.Errorf("OutputHash mismatch: %s != %s", att2.OutputHash, att.OutputHash)
	}
	if att2.SignedBy != att.SignedBy {
		t.Errorf("SignedBy mismatch: %s != %s", att2.SignedBy, att.SignedBy)
	}
}

func TestProvenanceFields(t *testing.T) {
	provenance := Provenance{
		Hostname:        "test-host",
		Platform:        "darwin",
		Arch:            "arm64",
		GitRepo:         "https://github.com/user/repo",
		GitCommit:       "abc123",
		GitBranch:       "main",
		GitDirty:        false,
		SpecularVersion: "1.0.0",
		Profile:         "ci",
		Models: []ModelUsage{
			{
				Provider: "openai",
				Model:    "gpt-4",
				Requests: 10,
				Cost:     0.50,
			},
		},
		TotalCost:     1.50,
		TasksExecuted: 15,
		TasksFailed:   2,
	}

	// Serialize
	data, err := json.Marshal(provenance)
	if err != nil {
		t.Fatalf("Failed to marshal provenance: %v", err)
	}

	// Deserialize
	var parsed Provenance
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal provenance: %v", err)
	}

	// Verify
	if parsed.Hostname != provenance.Hostname {
		t.Errorf("Hostname mismatch")
	}
	if parsed.Platform != provenance.Platform {
		t.Errorf("Platform mismatch")
	}
	if parsed.GitRepo != provenance.GitRepo {
		t.Errorf("GitRepo mismatch")
	}
	if len(parsed.Models) != len(provenance.Models) {
		t.Errorf("Models length mismatch")
	}
}

func TestModelUsage(t *testing.T) {
	usage := ModelUsage{
		Provider: "anthropic",
		Model:    "claude-3",
		Requests: 5,
		Cost:     2.50,
	}

	data, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("Failed to marshal model usage: %v", err)
	}

	var parsed ModelUsage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal model usage: %v", err)
	}

	if parsed.Provider != usage.Provider {
		t.Errorf("Provider mismatch")
	}
	if parsed.Model != usage.Model {
		t.Errorf("Model mismatch")
	}
	if parsed.Requests != usage.Requests {
		t.Errorf("Requests mismatch")
	}
	if parsed.Cost != usage.Cost {
		t.Errorf("Cost mismatch")
	}
}

func TestInvalidJSON(t *testing.T) {
	_, err := FromJSON([]byte("invalid json"))
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestEmptyAttestation(t *testing.T) {
	att := &Attestation{}
	data, err := att.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	att2, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if att2 == nil {
		t.Error("FromJSON returned nil attestation")
	}
}
