package plan

import (
	"fmt"
	"strings"
)

// Validate checks if the Task is valid according to domain rules
func (t *Task) Validate() error {
	// Validate TaskID using domain validation
	if err := t.ID.Validate(); err != nil {
		return fmt.Errorf("invalid task ID: %w", err)
	}

	// Validate FeatureID using domain validation
	if err := t.FeatureID.Validate(); err != nil {
		return fmt.Errorf("invalid feature ID: %w", err)
	}

	// Validate ExpectedHash is non-empty
	if strings.TrimSpace(t.ExpectedHash) == "" {
		return fmt.Errorf("expected hash cannot be empty")
	}

	// Validate DependsOn contains valid TaskID values
	for i, depID := range t.DependsOn {
		if err := depID.Validate(); err != nil {
			return fmt.Errorf("dependency at index %d has invalid task ID: %w", i, err)
		}
	}

	// Validate Skill is non-empty
	if strings.TrimSpace(t.Skill) == "" {
		return fmt.Errorf("skill cannot be empty")
	}

	// Validate Priority using domain validation
	if err := t.Priority.Validate(); err != nil {
		return fmt.Errorf("invalid priority: %w", err)
	}

	// Validate ModelHint is non-empty
	if strings.TrimSpace(t.ModelHint) == "" {
		return fmt.Errorf("model hint cannot be empty")
	}

	// Validate Estimate is positive
	if t.Estimate <= 0 {
		return fmt.Errorf("estimate must be positive, got %d", t.Estimate)
	}

	return nil
}

// Validate checks if the Plan is valid
func (p *Plan) Validate() error {
	// Validate Tasks - must have at least one
	if len(p.Tasks) == 0 {
		return fmt.Errorf("plan must have at least one task")
	}

	// Track task IDs to check for duplicates and validate dependencies
	taskIDs := make(map[string]bool)
	for i, task := range p.Tasks {
		// Validate each task
		if err := task.Validate(); err != nil {
			return fmt.Errorf("task at index %d (%s) is invalid: %w", i, task.ID, err)
		}

		// Check for duplicate task IDs
		if taskIDs[task.ID.String()] {
			return fmt.Errorf("duplicate task ID %q at index %d", task.ID, i)
		}
		taskIDs[task.ID.String()] = true
	}

	// Validate that all dependencies reference existing tasks
	for i, task := range p.Tasks {
		for _, depID := range task.DependsOn {
			if !taskIDs[depID.String()] {
				return fmt.Errorf("task at index %d (%s) has dependency %q that does not exist in plan", i, task.ID, depID)
			}
		}
	}

	// Check for circular dependencies
	if err := p.checkCircularDependencies(); err != nil {
		return err
	}

	return nil
}

// checkCircularDependencies detects cycles in the task dependency graph
func (p *Plan) checkCircularDependencies() error {
	// Build adjacency list (convert domain types to strings for graph)
	graph := make(map[string][]string)
	for _, task := range p.Tasks {
		deps := make([]string, len(task.DependsOn))
		for i, dep := range task.DependsOn {
			deps[i] = dep.String()
		}
		graph[task.ID.String()] = deps
	}

	// Track visited and recursion stack
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	// DFS helper function
	var hasCycle func(taskID string, path []string) error
	hasCycle = func(taskID string, path []string) error {
		visited[taskID] = true
		recStack[taskID] = true
		path = append(path, taskID)

		// Check all dependencies
		for _, dep := range graph[taskID] {
			if !visited[dep] {
				if err := hasCycle(dep, path); err != nil {
					return err
				}
			} else if recStack[dep] {
				// Found a cycle
				cyclePath := append(path, dep)
				return fmt.Errorf("circular dependency detected: %s", strings.Join(cyclePath, " -> "))
			}
		}

		recStack[taskID] = false
		return nil
	}

	// Check each task
	for _, task := range p.Tasks {
		if !visited[task.ID.String()] {
			if err := hasCycle(task.ID.String(), []string{}); err != nil {
				return err
			}
		}
	}

	return nil
}
