package plan

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/specular/internal/spec"
)

// GenerateOptions contains options for plan generation
type GenerateOptions struct {
	// SpecLock provides hash values for tasks
	SpecLock *spec.SpecLock
	// EstimateComplexity enables automatic complexity estimation
	EstimateComplexity bool
}

// Generate creates a Plan from a ProductSpec
func Generate(s *spec.ProductSpec, opts GenerateOptions) (*Plan, error) {
	if opts.SpecLock == nil {
		return nil, fmt.Errorf("SpecLock is required for plan generation")
	}

	var tasks []Task

	// Create tasks for each feature
	for i, feature := range s.Features {
		// Get locked feature for hash
		lockedFeature, exists := opts.SpecLock.Features[feature.ID]
		if !exists {
			return nil, fmt.Errorf("feature %s not found in SpecLock", feature.ID)
		}

		task := Task{
			ID:           fmt.Sprintf("task-%03d", i+1),
			FeatureID:    feature.ID,
			ExpectedHash: lockedFeature.Hash,
			DependsOn:    determineDependencies(feature, s.Features, i),
			Skill:        determineSkill(feature),
			Priority:     feature.Priority,
			ModelHint:    determineModelHint(feature),
		}

		// Estimate complexity if enabled
		if opts.EstimateComplexity {
			task.Estimate = estimateComplexity(feature)
		}

		tasks = append(tasks, task)
	}

	// Validate topological ordering
	if err := validateDependencies(tasks); err != nil {
		return nil, fmt.Errorf("invalid task dependencies: %w", err)
	}

	return &Plan{Tasks: tasks}, nil
}

// determineDependencies identifies task dependencies based on priority and trace
func determineDependencies(feature spec.Feature, allFeatures []spec.Feature, currentIndex int) []string {
	var deps []string

	// P0 features have no dependencies
	if feature.Priority == "P0" {
		return deps
	}

	// P1 and P2 depend on all P0 features that came before
	for i := 0; i < currentIndex; i++ {
		if allFeatures[i].Priority == "P0" {
			taskID := fmt.Sprintf("task-%03d", i+1)
			deps = append(deps, taskID)
		}
	}

	// If this is P2, also depend on P1 features
	if feature.Priority == "P2" {
		for i := 0; i < currentIndex; i++ {
			if allFeatures[i].Priority == "P1" {
				taskID := fmt.Sprintf("task-%03d", i+1)
				deps = append(deps, taskID)
			}
		}
	}

	return deps
}

// determineSkill assigns a skill tag based on feature characteristics
func determineSkill(feature spec.Feature) string {
	// Check for API endpoints
	if len(feature.API) > 0 {
		for _, api := range feature.API {
			if strings.Contains(strings.ToLower(api.Path), "/api") {
				return "go-backend"
			}
		}
	}

	// Check title and description for keywords
	text := strings.ToLower(feature.Title + " " + feature.Desc)

	if strings.Contains(text, "ui") || strings.Contains(text, "interface") || strings.Contains(text, "component") {
		return "ui-react"
	}

	if strings.Contains(text, "docker") || strings.Contains(text, "deploy") || strings.Contains(text, "infrastructure") {
		return "infra"
	}

	if strings.Contains(text, "database") || strings.Contains(text, "schema") || strings.Contains(text, "migration") {
		return "database"
	}

	if strings.Contains(text, "test") || strings.Contains(text, "validation") {
		return "testing"
	}

	// Default to backend
	return "go-backend"
}

// determineModelHint suggests which type of model should handle this task
func determineModelHint(feature spec.Feature) string {
	// Complex features with many API endpoints need long-context models
	if len(feature.API) > 5 {
		return "long-context"
	}

	// Features with many success criteria need careful planning
	if len(feature.Success) > 5 {
		return "agentic"
	}

	// Standard code generation
	return "codegen"
}

// estimateComplexity provides a rough complexity estimate (1-10)
func estimateComplexity(feature spec.Feature) int {
	complexity := 1

	// API endpoints add complexity
	complexity += len(feature.API)

	// Success criteria add complexity
	complexity += len(feature.Success) / 2

	// Trace references add complexity (indicates cross-cutting concerns)
	complexity += len(feature.Trace)

	// Cap at 10
	if complexity > 10 {
		complexity = 10
	}

	return complexity
}

// validateDependencies ensures the task graph is acyclic
func validateDependencies(tasks []Task) error {
	// Build task ID set for validation
	taskIDs := make(map[string]bool)
	for _, task := range tasks {
		taskIDs[task.ID] = true
	}

	// Check all dependencies exist
	for _, task := range tasks {
		for _, dep := range task.DependsOn {
			if !taskIDs[dep] {
				return fmt.Errorf("task %s depends on non-existent task %s", task.ID, dep)
			}
		}
	}

	// Simple cycle detection: tasks can only depend on earlier tasks
	taskIndices := make(map[string]int)
	for i, task := range tasks {
		taskIndices[task.ID] = i
	}

	for _, task := range tasks {
		currentIndex := taskIndices[task.ID]
		for _, dep := range task.DependsOn {
			depIndex := taskIndices[dep]
			if depIndex >= currentIndex {
				return fmt.Errorf("task %s has forward or circular dependency on %s", task.ID, dep)
			}
		}
	}

	return nil
}
