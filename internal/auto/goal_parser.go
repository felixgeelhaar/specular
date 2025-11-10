package auto

import (
	"context"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/specular/internal/domain"
	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/spec"
	"gopkg.in/yaml.v3"
)

// GoalParser converts natural language goals into structured specifications
type GoalParser struct {
	router *router.Router
}

// NewGoalParser creates a new goal parser with the given router
func NewGoalParser(r *router.Router) *GoalParser {
	return &GoalParser{router: r}
}

// ParseGoal converts a natural language goal into a ProductSpec
func (p *GoalParser) ParseGoal(ctx context.Context, goal string) (*spec.ProductSpec, error) {
	systemPrompt := `You are a software specification expert. Convert the user's goal into a structured YAML specification following this exact format:

name: <project-name>
description: <brief-description>
version: 1.0.0
metadata:
  author: AI Generated
  created: <timestamp-YYYY-MM-DD>

features:
  - id: <feature-id>  # lowercase-with-hyphens
    title: <feature-title>
    description: <detailed-description>
    priority: P0  # or P1, P2
    category: api  # or ui, data, infra, testing
    acceptance_criteria:
      - <testable-criterion-1>
      - <testable-criterion-2>
      - <testable-criterion-3>

IMPORTANT RULES:
1. Feature IDs must be lowercase letters, numbers, and hyphens only (e.g., "user-auth", "payment-api")
2. Priority: P0 = critical/required, P1 = important, P2 = nice-to-have
3. Categories: api, ui, data, infra, testing
4. Each feature must have 2-5 specific, testable acceptance criteria
5. Keep descriptions concise but clear
6. Break down the goal into 2-5 logical features
7. Order features by priority (P0 first, then P1, then P2)

Return ONLY the YAML, no explanations or markdown code blocks.`

	req := router.GenerateRequest{
		Prompt:       goal,
		SystemPrompt: systemPrompt,
		ModelHint:    "agentic",
		Complexity:   7,
		Priority:     "P0",
		Temperature:  0.3, // Lower temperature for structured output
		MaxTokens:    2000,
		TaskID:       domain.TaskID("goal-parse"),
	}

	resp, err := p.router.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generate spec: %w", err)
	}

	// Clean the response (remove markdown code blocks if present)
	yamlContent := cleanYAMLResponse(resp.Content)

	// Parse YAML into ProductSpec
	var productSpec spec.ProductSpec
	if err := yaml.Unmarshal([]byte(yamlContent), &productSpec); err != nil {
		return nil, fmt.Errorf("parse generated spec: %w\nRaw content:\n%s", err, yamlContent)
	}

	// Validate feature IDs
	for i, feature := range productSpec.Features {
		if _, err := domain.NewFeatureID(feature.ID.String()); err != nil {
			return nil, fmt.Errorf("invalid feature ID at index %d: %w", i, err)
		}
	}

	return &productSpec, nil
}

// cleanYAMLResponse removes markdown code blocks and extra whitespace
func cleanYAMLResponse(content string) string {
	// Remove markdown code blocks
	content = strings.TrimPrefix(content, "```yaml")
	content = strings.TrimPrefix(content, "```yml")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")

	// Trim whitespace
	content = strings.TrimSpace(content)

	return content
}
