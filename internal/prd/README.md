# PRD Parser

The PRD (Product Requirements Document) Parser converts unstructured markdown PRD documents into structured YAML product specifications using AI-powered parsing.

## Overview

The PRD parser integrates with the AI provider system and router to intelligently select the best AI model for parsing PRDs. It uses carefully crafted prompts to extract product information, features, goals, non-functional requirements, and other specification elements from natural language PRDs.

## Architecture

### Components

1. **Parser** (`parser.go`): Core PRD parsing logic
   - Integrates with the router for intelligent model selection
   - Builds prompts for AI-powered conversion
   - Parses and validates JSON responses
   - Handles markdown code block extraction

2. **Prompts** (`prompts.go`): Prompt engineering for PRD-to-spec conversion
   - System prompt: Defines the AI's role and guidelines
   - User prompt: Provides PRD content and output structure
   - JSON extraction: Handles responses in various formats

3. **Tests** (`parser_test.go`, `prompts_test.go`): Comprehensive test coverage
   - Unit tests with mock providers
   - Edge case handling
   - Prompt validation
   - JSON extraction scenarios

## Usage

### Basic Usage

```go
import (
	"context"
	"github.com/felixgeelhaar/specular/internal/prd"
	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/internal/router"
)

// Load providers and create router
registry, _ := provider.LoadRegistryFromConfig(".specular/providers.yaml")
providerConfig, _ := provider.LoadProvidersConfig(".specular/providers.yaml")

routerConfig := &router.RouterConfig{
	BudgetUSD:    providerConfig.Strategy.Budget.MaxCostPerDay,
	MaxLatencyMs: providerConfig.Strategy.Performance.MaxLatencyMs,
	PreferCheap:  providerConfig.Strategy.Performance.PreferCheap,
}

r, _ := router.NewRouterWithProviders(routerConfig, registry)

// Create parser and parse PRD
parser := prd.NewParser(r)
ctx := context.Background()

prdContent := `
# My Product

## Goals
- Simplify user onboarding
- Increase engagement

## Features
...
`

spec, err := parser.ParsePRD(ctx, prdContent)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Generated spec with %d features\n", len(spec.Features))
```

### CLI Usage

The PRD parser is integrated into the `specular spec generate` command:

```bash
# Generate spec from PRD
specular spec generate --in PRD.md --out .specular/spec.yaml

# Use custom provider configuration
specular spec generate --in PRD.md --out spec.yaml --config custom-providers.yaml
```

## Prompt Engineering

### System Prompt

The system prompt establishes the AI's role as an "expert technical product manager and software architect" and provides clear guidelines for:

- Extracting product names, goals, and features
- Creating unique feature IDs (feat-001, feat-002, etc.)
- Assigning priorities (P0, P1, P2)
- Identifying non-functional requirements
- Structuring output as valid JSON

### User Prompt

The user prompt includes:

1. The full PRD content
2. A complete JSON structure example
3. Specific instructions for output format
4. Guidelines for handling missing information

### Model Selection

The parser uses the router with these parameters:

- **ModelHint**: `agentic` - Complex reasoning required for PRD analysis
- **Complexity**: `8` - High complexity task (1-10 scale)
- **Priority**: `P0` - Critical task for spec generation
- **ContextSize**: Estimated from PRD content length

The router selects the best available model based on:
- Task requirements (agentic capabilities)
- Budget constraints
- Provider availability
- Performance characteristics

## Output Format

The parser generates a `ProductSpec` structure:

```go
type ProductSpec struct {
	Product       string        // Product name
	Goals         []string      // Product goals
	Features      []Feature     // Feature list
	NonFunctional NonFunctional // Performance, security, etc.
	Acceptance    []string      // Acceptance criteria
	Milestones    []Milestone   // Project milestones
}

type Feature struct {
	ID       string   // Unique ID (feat-001, etc.)
	Title    string   // Feature title
	Desc     string   // Detailed description
	Priority string   // P0, P1, or P2
	API      []API    // API/CLI specifications
	Success  []string // Success criteria
	Trace    []string // PRD section references
}
```

## JSON Extraction

The parser handles AI responses in multiple formats:

1. **Plain JSON**: Direct JSON object
2. **Markdown code blocks**: JSON wrapped in \`\`\`json ... \`\`\`
3. **Embedded JSON**: JSON object within other text

Extraction Strategy:
1. Try direct JSON parsing
2. Extract from markdown code blocks (with regex)
3. Find JSON by brace matching
4. Return error if no valid JSON found

## Error Handling

The parser validates the generated specification:

- **Product name required**: Spec must have a non-empty product name
- **Features required**: At least one feature must be defined
- **Priority validation**: Features must have valid priorities (P0, P1, P2)
- **JSON parsing errors**: Clear error messages with response content

## Testing

### Test Coverage

- ✅ Successful PRD parsing
- ✅ Markdown code block extraction
- ✅ Missing product name validation
- ✅ Missing features validation
- ✅ JSON extraction edge cases
- ✅ System prompt validation
- ✅ User prompt validation
- ✅ Nested JSON handling

### Running Tests

```bash
# Run all PRD tests
go test -v ./internal/prd/...

# Run with coverage
go test -cover ./internal/prd/...

# Run specific test
go test -v -run TestParsePRD_Success ./internal/prd/...
```

### Mock Provider

Tests use a mock provider that:
- Returns configurable JSON responses
- Simulates AI generation without actual API calls
- Supports error injection for testing error paths

## Integration

### Integration with Router

The PRD parser leverages the router system for:

- Intelligent model selection based on task requirements
- Budget tracking and cost optimization
- Provider failover and retry logic
- Performance monitoring

### Integration with Spec System

Generated specs are compatible with:

- `spec.SaveSpec()` - Save to YAML file
- `spec.LoadSpec()` - Load from YAML file
- `spec validate` - Validation command
- `spec lock` - SpecLock generation

## Performance

Typical parsing times:

- **Small PRD** (< 1000 words): 10-20 seconds
- **Medium PRD** (1000-3000 words): 30-45 seconds
- **Large PRD** (> 3000 words): 45-60 seconds

Performance depends on:
- Selected AI provider (local vs API)
- Model size and capabilities
- Network latency (for API providers)
- PRD complexity and length

## Future Enhancements

Potential improvements:

1. **Incremental parsing**: Parse sections incrementally for large PRDs
2. **Interactive refinement**: Ask clarifying questions for ambiguous sections
3. **Multi-pass parsing**: Use different models for different sections
4. **Confidence scoring**: Provide confidence scores for extracted features
5. **Diff generation**: Compare PRD versions and update specs incrementally
6. **Template support**: PRD templates for common product types
7. **Validation rules**: Custom validation rules for domain-specific requirements

## References

- [Product Specification Format](../spec/README.md)
- [Provider System](../provider/README.md)
- [Router System](../router/README.md)
- [PRD Example](../../docs/prd.md)
