package auto

import (
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/specular/internal/domain"
	"github.com/felixgeelhaar/specular/internal/plan"
	"github.com/felixgeelhaar/specular/internal/spec"
)

// PatternType indicates what kind of scope pattern this is.
type PatternType string

const (
	// PatternTypeFeatureID matches by exact feature ID
	PatternTypeFeatureID PatternType = "feature_id"

	// PatternTypeFeatureTitle matches feature titles with glob patterns
	PatternTypeFeatureTitle PatternType = "feature_title"

	// PatternTypePath matches file paths with glob patterns
	PatternTypePath PatternType = "path"

	// PatternTypeTag matches feature tags (future use)
	PatternTypeTag PatternType = "tag"
)

// ScopePattern represents a single scope filter pattern.
type ScopePattern struct {
	// Type indicates what this pattern matches against
	Type PatternType

	// Pattern is the matching pattern (glob, ID, or exact match)
	Pattern string
}

// Scope represents a collection of scope patterns for filtering.
type Scope struct {
	// Patterns contains all scope filter patterns
	Patterns []ScopePattern

	// IncludeDependencies if true, includes tasks that matched tasks depend on
	IncludeDependencies bool
}

// NewScope creates a new scope filter from pattern strings.
// Pattern format:
//   - "feature:ID" - Match feature by exact ID
//   - "feature:title pattern" - Match feature titles with glob
//   - "path/pattern/**" - Match paths with glob
//   - "@tag" - Match by tag (future)
func NewScope(patterns []string, includeDeps bool) (*Scope, error) {
	scope := &Scope{
		Patterns:            make([]ScopePattern, 0, len(patterns)),
		IncludeDependencies: includeDeps,
	}

	for _, p := range patterns {
		pattern, err := parsePattern(p)
		if err != nil {
			return nil, err
		}
		scope.Patterns = append(scope.Patterns, pattern)
	}

	return scope, nil
}

// parsePattern parses a pattern string into a ScopePattern.
func parsePattern(pattern string) (ScopePattern, error) {
	// Check for feature: prefix
	if strings.HasPrefix(pattern, "feature:") {
		featurePattern := strings.TrimPrefix(pattern, "feature:")
		// If it looks like an ID (no wildcards), treat as feature ID
		if !strings.ContainsAny(featurePattern, "*?[]") {
			return ScopePattern{
				Type:    PatternTypeFeatureID,
				Pattern: featurePattern,
			}, nil
		}
		// Otherwise treat as feature title pattern
		return ScopePattern{
			Type:    PatternTypeFeatureTitle,
			Pattern: featurePattern,
		}, nil
	}

	// Check for tag prefix
	if strings.HasPrefix(pattern, "@") {
		return ScopePattern{
			Type:    PatternTypeTag,
			Pattern: strings.TrimPrefix(pattern, "@"),
		}, nil
	}

	// Default to path pattern
	return ScopePattern{
		Type:    PatternTypePath,
		Pattern: pattern,
	}, nil
}

// MatchesFeature checks if a feature matches any scope pattern.
func (s *Scope) MatchesFeature(feature spec.Feature) bool {
	if len(s.Patterns) == 0 {
		return true // No scope means match all
	}

	for _, pattern := range s.Patterns {
		if matchesFeaturePattern(feature, pattern) {
			return true
		}
	}
	return false
}

// matchesFeaturePattern checks if a feature matches a single pattern.
func matchesFeaturePattern(feature spec.Feature, pattern ScopePattern) bool {
	switch pattern.Type {
	case PatternTypeFeatureID:
		return string(feature.ID) == pattern.Pattern

	case PatternTypeFeatureTitle:
		matched, _ := filepath.Match(pattern.Pattern, feature.Title)
		return matched

	case PatternTypePath:
		// Check if any API path in the feature matches
		for _, api := range feature.API {
			if matched, _ := filepath.Match(pattern.Pattern, api.Path); matched {
				return true
			}
		}
		return false

	case PatternTypeTag:
		// Tag matching not yet implemented
		return false

	default:
		return false
	}
}

// MatchesTask checks if a task matches the scope based on its feature.
func (s *Scope) MatchesTask(task plan.Task, productSpec *spec.ProductSpec) bool {
	if len(s.Patterns) == 0 {
		return true // No scope means match all
	}

	// Find the feature this task belongs to
	var taskFeature *spec.Feature
	for i := range productSpec.Features {
		if productSpec.Features[i].ID == task.FeatureID {
			taskFeature = &productSpec.Features[i]
			break
		}
	}

	if taskFeature == nil {
		return false // Task's feature not found
	}

	return s.MatchesFeature(*taskFeature)
}

// FilterPlan filters a plan to only include tasks matching the scope.
// It preserves dependencies by including tasks that matched tasks depend on.
func (s *Scope) FilterPlan(execPlan *plan.Plan, productSpec *spec.ProductSpec) *plan.Plan {
	if len(s.Patterns) == 0 {
		return execPlan // No filtering needed
	}

	// Phase 1: Find directly matching tasks
	matchedTasks := make(map[domain.TaskID]bool)
	for _, task := range execPlan.Tasks {
		if s.MatchesTask(task, productSpec) {
			matchedTasks[task.ID] = true
		}
	}

	// Phase 2: Include dependencies if enabled
	if s.IncludeDependencies {
		matchedTasks = s.expandDependencies(execPlan, matchedTasks)
	}

	// Phase 3: Build filtered plan
	filteredTasks := make([]plan.Task, 0)
	for _, task := range execPlan.Tasks {
		if matchedTasks[task.ID] {
			filteredTasks = append(filteredTasks, task)
		}
	}

	return &plan.Plan{
		Tasks: filteredTasks,
	}
}

// expandDependencies recursively includes all dependencies of matched tasks.
func (s *Scope) expandDependencies(execPlan *plan.Plan, matched map[domain.TaskID]bool) map[domain.TaskID]bool {
	expanded := make(map[domain.TaskID]bool)
	for id := range matched {
		expanded[id] = true
	}

	// Keep expanding until no new dependencies are found
	changed := true
	for changed {
		changed = false
		for _, task := range execPlan.Tasks {
			if !expanded[task.ID] {
				// Check if this task is a dependency of any expanded task
				for _, otherTask := range execPlan.Tasks {
					if expanded[otherTask.ID] {
						for _, depID := range otherTask.DependsOn {
							if depID == task.ID {
								expanded[task.ID] = true
								changed = true
								break
							}
						}
						if changed {
							break
						}
					}
				}
			}
		}
	}

	return expanded
}

// Summary returns a human-readable description of the scope.
func (s *Scope) Summary() string {
	if len(s.Patterns) == 0 {
		return "all features"
	}

	parts := make([]string, len(s.Patterns))
	for i, pattern := range s.Patterns {
		switch pattern.Type {
		case PatternTypeFeatureID:
			parts[i] = "feature:" + pattern.Pattern
		case PatternTypeFeatureTitle:
			parts[i] = "title:" + pattern.Pattern
		case PatternTypePath:
			parts[i] = "path:" + pattern.Pattern
		case PatternTypeTag:
			parts[i] = "@" + pattern.Pattern
		default:
			parts[i] = pattern.Pattern
		}
	}

	result := strings.Join(parts, ", ")
	if s.IncludeDependencies {
		result += " (with dependencies)"
	}
	return result
}

// EstimateImpact estimates how many tasks will be affected by this scope.
func (s *Scope) EstimateImpact(execPlan *plan.Plan, productSpec *spec.ProductSpec) (matched int, total int) {
	total = len(execPlan.Tasks)
	if len(s.Patterns) == 0 {
		return total, total
	}

	matchedTasks := make(map[domain.TaskID]bool)
	for _, task := range execPlan.Tasks {
		if s.MatchesTask(task, productSpec) {
			matchedTasks[task.ID] = true
		}
	}

	if s.IncludeDependencies {
		matchedTasks = s.expandDependencies(execPlan, matchedTasks)
	}

	return len(matchedTasks), total
}
