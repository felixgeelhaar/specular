package plan

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadPlan reads a Plan from a JSON file
func LoadPlan(path string) (*Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plan file: %w", err)
	}

	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("unmarshal plan: %w", err)
	}

	// Validate the loaded plan using domain validation
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("validate plan: %w", err)
	}

	return &p, nil
}

// SavePlan writes a Plan to a JSON file
func SavePlan(p *Plan, path string) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}

	return nil
}
