package plan

import "github.com/felixgeelhaar/specular/internal/domain"

// Plan represents the execution plan as a DAG of tasks
type Plan struct {
	Tasks []Task `json:"tasks"`
}

// Task represents a single unit of work in the plan
type Task struct {
	ID           domain.TaskID     `json:"id"`
	FeatureID    domain.FeatureID  `json:"feature_id"`
	ExpectedHash string            `json:"expected_hash"` // Links to SpecLock feature hash
	DependsOn    []domain.TaskID   `json:"depends_on"`
	Skill        string            `json:"skill"`      // go-backend, ui-react, infra, etc.
	Priority     domain.Priority   `json:"priority"`   // P0, P1, P2
	ModelHint    string            `json:"model_hint"` // long-context, agentic, codegen, etc.
	Estimate     int               `json:"estimate"`   // Estimated complexity/time
}
