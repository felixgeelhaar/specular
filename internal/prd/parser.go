package prd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/internal/spec"
)

// Parser converts PRD markdown to structured specifications using AI
type Parser struct {
	router *router.Router
}

// NewParser creates a new PRD parser with AI integration
func NewParser(r *router.Router) *Parser {
	return &Parser{
		router: r,
	}
}

// ParsePRD converts a PRD markdown document into a structured ProductSpec
// using AI to extract and structure the information
func (p *Parser) ParsePRD(ctx context.Context, prdContent string) (*spec.ProductSpec, error) {
	// Build the conversion prompt
	systemPrompt := buildSystemPrompt()
	userPrompt := buildUserPrompt(prdContent)

	// Build the generation request
	req := router.GenerateRequest{
		Prompt:       userPrompt,
		SystemPrompt: systemPrompt,
		ModelHint:    "agentic",              // Use agentic model for complex reasoning
		Complexity:   8,                      // High complexity task
		Priority:     "P0",                   // Critical task
		Temperature:  0.1,                    // Low temperature for consistent structured output
		MaxTokens:    4000,                   // Allow space for full spec
		ContextSize:  len(prdContent) / 4,    // Rough estimate of context tokens
	}

	// Generate the spec using AI (router handles model selection and provider lookup)
	resp, err := p.router.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate spec from PRD: %w", err)
	}

	fmt.Printf("Using %s/%s for PRD parsing (%s)\n", resp.Provider, resp.Model, resp.SelectionReason)

	// Parse the JSON response
	var productSpec spec.ProductSpec
	if err := json.Unmarshal([]byte(resp.Content), &productSpec); err != nil {
		// Try to extract JSON from markdown code blocks
		jsonContent := extractJSONFromMarkdown(resp.Content)
		if jsonContent == "" {
			return nil, fmt.Errorf("failed to parse AI response as JSON: %w\nResponse:\n%s", err, resp.Content)
		}
		if err := json.Unmarshal([]byte(jsonContent), &productSpec); err != nil {
			return nil, fmt.Errorf("failed to parse extracted JSON: %w", err)
		}
	}

	// Basic validation
	if productSpec.Product == "" {
		return nil, fmt.Errorf("invalid spec: product name is required")
	}
	if len(productSpec.Features) == 0 {
		return nil, fmt.Errorf("invalid spec: at least one feature is required")
	}

	return &productSpec, nil
}
