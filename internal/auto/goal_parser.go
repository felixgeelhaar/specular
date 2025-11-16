package auto

import (
	"context"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/specular/pkg/specular/types"
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
// It retries up to 3 times if parsing fails
func (p *GoalParser) ParseGoal(ctx context.Context, goal string) (*spec.ProductSpec, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		productSpec, err := p.parseGoalAttempt(ctx, goal)
		if err == nil {
			return productSpec, nil
		}

		lastErr = err
		if attempt < maxRetries {
			// Simple retry - AI models are non-deterministic, different attempt may succeed
			continue
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// parseGoalAttempt performs a single attempt at parsing the goal
func (p *GoalParser) parseGoalAttempt(ctx context.Context, goal string) (*spec.ProductSpec, error) {
	systemPrompt := `You are a software specification expert. Convert the user's goal into a structured YAML specification following this exact format:

product: <project-name>
goals:
  - <high-level-goal-1>
  - <high-level-goal-2>

features:
  - id: <feature-id>  # lowercase-with-hyphens
    title: <feature-title>
    desc: <detailed-description>
    priority: P0  # or P1, P2
    success:
      - <testable-success-criterion-1>
      - <testable-success-criterion-2>
      - <testable-success-criterion-3>
    trace:
      - <implementation-detail-1>
      - <implementation-detail-2>

IMPORTANT RULES:
1. Product name should be short and descriptive (e.g., "Todo API", "Weather Service")
2. Goals should be high-level objectives (1-3 goals)
3. Feature IDs must be lowercase letters, numbers, and hyphens only (e.g., "user-auth", "payment-api")
4. Priority: P0 = critical/required, P1 = important, P2 = nice-to-have
5. Each feature must have 2-5 specific, testable success criteria
6. Trace items describe implementation details or technical requirements
7. Break down the goal into 2-5 logical features
8. Order features by priority (P0 first, then P1, then P2)

Return ONLY the YAML, no explanations or markdown code blocks.`

	req := router.GenerateRequest{
		Prompt:       goal,
		SystemPrompt: systemPrompt,
		ModelHint:    "agentic",
		Complexity:   7,
		Priority:     "P0",
		Temperature:  0.3, // Lower temperature for structured output
		MaxTokens:    2000,
		TaskID:       types.TaskID("goal-parse"),
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
		// Provide helpful error message with context
		return nil, fmt.Errorf("parse generated spec: %w\n\n"+
			"This error usually means the AI generated invalid YAML format.\n"+
			"The spec will be automatically retried with a fresh AI generation.\n\n"+
			"Raw YAML content:\n%s", err, yamlContent)
	}

	// Validate required fields
	if productSpec.Product == "" {
		return nil, fmt.Errorf("generated spec missing required 'product' field\n\nRaw content:\n%s", yamlContent)
	}

	if len(productSpec.Features) == 0 {
		return nil, fmt.Errorf("generated spec has no features\n\nRaw content:\n%s", yamlContent)
	}

	// Validate feature IDs
	for i, feature := range productSpec.Features {
		if _, err := types.NewFeatureID(feature.ID.String()); err != nil {
			return nil, fmt.Errorf("invalid feature ID '%s' at index %d: %w\n"+
				"Feature IDs must be lowercase letters, numbers, and hyphens only",
				feature.ID.String(), i, err)
		}

		// Validate feature has required fields
		if feature.Title == "" {
			return nil, fmt.Errorf("feature %d ('%s') missing required 'title' field", i, feature.ID.String())
		}

		if len(feature.Success) == 0 {
			return nil, fmt.Errorf("feature %d ('%s') has no success criteria", i, feature.ID.String())
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
