package codegen

import (
	"fmt"
	"strings"
)

// PromptBuilder constructs prompts for code generation based on skill type
type PromptBuilder struct {
	systemPrompts map[string]string
}

// NewPromptBuilder creates a new prompt builder with skill-specific system prompts
func NewPromptBuilder() *PromptBuilder {
	pb := &PromptBuilder{
		systemPrompts: make(map[string]string),
	}
	pb.initSystemPrompts()
	return pb
}

// initSystemPrompts initializes skill-specific system prompts
func (pb *PromptBuilder) initSystemPrompts() {
	// Go backend development
	pb.systemPrompts["go-backend"] = `You are an expert Go developer specializing in backend services.

Follow these principles:
- Use Clean Architecture: separate domain, application, and infrastructure layers
- Write idiomatic Go code following effective Go guidelines
- Include comprehensive error handling with wrapped errors
- Create unit tests alongside implementation code
- Use interfaces for dependency injection and testability
- Follow the standard project layout (cmd/, internal/, pkg/)
- Use context.Context for cancellation and timeouts
- Implement proper logging with structured loggers

Output format:
- Return all code as markdown code blocks with file path annotations
- Use the format: ` + "```go:path/to/file.go" + `
- Include all necessary imports
- Add documentation comments for exported types and functions
- Create corresponding _test.go files for testable code`

	// React/TypeScript frontend
	pb.systemPrompts["ui-react"] = `You are an expert React/TypeScript developer.

Follow these principles:
- Use functional components with TypeScript
- Implement proper type safety with interfaces and type guards
- Follow component composition patterns
- Use custom hooks for reusable logic
- Implement proper error boundaries
- Follow accessibility (a11y) best practices
- Use CSS modules or styled-components for styling
- Create unit tests with React Testing Library

Output format:
- Return all code as markdown code blocks with file path annotations
- Use the format: ` + "```tsx:path/to/Component.tsx" + `
- Include all necessary imports
- Create corresponding .test.tsx files
- Use proper TypeScript types (avoid any)`

	// Database/SQL
	pb.systemPrompts["database"] = `You are an expert database engineer.

Follow these principles:
- Write efficient, indexed SQL migrations
- Use proper normalization (3NF where appropriate)
- Include rollback migrations
- Add appropriate indexes for query patterns
- Use transactions for data integrity
- Follow naming conventions (snake_case for tables/columns)
- Add comments explaining complex queries
- Consider query performance and execution plans

Output format:
- Return all code as markdown code blocks with file path annotations
- Use the format: ` + "```sql:migrations/YYYYMMDD_description.sql" + `
- Include both up and down migrations
- Add comments for complex logic`

	// Testing
	pb.systemPrompts["testing"] = `You are an expert test engineer.

Follow these principles:
- Write comprehensive unit and integration tests
- Use table-driven tests in Go
- Follow Arrange-Act-Assert pattern
- Mock external dependencies appropriately
- Test edge cases and error paths
- Aim for meaningful coverage, not 100%
- Use descriptive test names
- Keep tests independent and isolated

Output format:
- Return all code as markdown code blocks with file path annotations
- Use the format: ` + "```go:path/to/file_test.go" + `
- Group related tests with subtests
- Include helper functions for common setup`

	// Infrastructure
	pb.systemPrompts["infra"] = `You are an expert DevOps/infrastructure engineer.

Follow these principles:
- Use Infrastructure as Code (Terraform, CloudFormation)
- Follow 12-factor app principles
- Implement proper secrets management
- Use containerization with minimal base images
- Include health checks and readiness probes
- Configure proper resource limits
- Implement logging and monitoring hooks
- Follow security best practices

Output format:
- Return all code as markdown code blocks with file path annotations
- Use formats like: ` + "```yaml:k8s/deployment.yaml" + ` or ` + "```hcl:terraform/main.tf" + `
- Include comments explaining configuration choices`

	// Default/generic
	pb.systemPrompts["default"] = `You are an expert software developer.

Follow these principles:
- Write clean, maintainable code
- Include appropriate documentation
- Handle errors gracefully
- Write testable code
- Follow language-specific best practices

Output format:
- Return all code as markdown code blocks with file path annotations
- Use the format: ` + "```language:path/to/file.ext" + `
- Include all necessary imports`
}

// BuildPrompt constructs the complete prompt for a code generation request
func (pb *PromptBuilder) BuildPrompt(req CodeGenerationRequest) (systemPrompt string, userPrompt string) {
	// Get skill-specific system prompt
	systemPrompt = pb.getSystemPrompt(req.Skill)

	// Build user prompt
	var sb strings.Builder

	// Feature context
	sb.WriteString("## Feature to Implement\n\n")
	sb.WriteString(fmt.Sprintf("**ID:** %s\n", req.FeatureID))
	sb.WriteString(fmt.Sprintf("**Title:** %s\n", req.FeatureTitle))
	sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", req.FeatureDesc))

	// Success criteria
	if len(req.SuccessCriteria) > 0 {
		sb.WriteString("## Success Criteria\n\n")
		for i, criterion := range req.SuccessCriteria {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, criterion))
		}
		sb.WriteString("\n")
	}

	// API endpoints
	if len(req.APIs) > 0 {
		sb.WriteString("## API Endpoints to Implement\n\n")
		for _, api := range req.APIs {
			sb.WriteString(fmt.Sprintf("- **%s** `%s`\n", api.Method, api.Path))
			if api.Request != "" {
				sb.WriteString(fmt.Sprintf("  - Request: `%s`\n", api.Request))
			}
			if api.Response != "" {
				sb.WriteString(fmt.Sprintf("  - Response: `%s`\n", api.Response))
			}
		}
		sb.WriteString("\n")
	}

	// Product context
	if req.ProductContext != "" {
		sb.WriteString("## Product Context\n\n")
		sb.WriteString(req.ProductContext)
		sb.WriteString("\n\n")
	}

	// Existing files context
	if len(req.ExistingFiles) > 0 {
		sb.WriteString("## Existing Code Context\n\n")
		for path, content := range req.ExistingFiles {
			ext := getFileExtension(path)
			sb.WriteString(fmt.Sprintf("### %s\n\n", path))
			sb.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", ext, content))
		}
	}

	// Generation instructions
	sb.WriteString("## Instructions\n\n")
	sb.WriteString("Generate production-ready code that implements the feature above.\n")
	sb.WriteString("Include all necessary files with clear path annotations.\n")
	sb.WriteString("Follow the output format specified in the system prompt.\n")

	userPrompt = sb.String()
	return
}

// getSystemPrompt returns the system prompt for a skill
func (pb *PromptBuilder) getSystemPrompt(skill string) string {
	if prompt, ok := pb.systemPrompts[skill]; ok {
		return prompt
	}
	return pb.systemPrompts["default"]
}

// AddSystemPrompt adds or updates a skill-specific system prompt
func (pb *PromptBuilder) AddSystemPrompt(skill, prompt string) {
	pb.systemPrompts[skill] = prompt
}

// getFileExtension extracts the file extension for syntax highlighting
func getFileExtension(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return ""
	}
	ext := parts[len(parts)-1]

	// Map extensions to language names for syntax highlighting
	mapping := map[string]string{
		"go":    "go",
		"ts":    "typescript",
		"tsx":   "tsx",
		"js":    "javascript",
		"jsx":   "jsx",
		"py":    "python",
		"sql":   "sql",
		"yaml":  "yaml",
		"yml":   "yaml",
		"json":  "json",
		"tf":    "hcl",
		"md":    "markdown",
		"sh":    "bash",
		"bash":  "bash",
		"rs":    "rust",
		"java":  "java",
		"kt":    "kotlin",
		"rb":    "ruby",
		"php":   "php",
		"cs":    "csharp",
		"cpp":   "cpp",
		"c":     "c",
		"h":     "c",
		"hpp":   "cpp",
		"swift": "swift",
	}

	if lang, ok := mapping[ext]; ok {
		return lang
	}
	return ext
}

// SkillToLanguage maps skill types to primary languages
func SkillToLanguage(skill string) string {
	mapping := map[string]string{
		"go-backend": "go",
		"ui-react":   "typescript",
		"database":   "sql",
		"testing":    "go",
		"infra":      "yaml",
	}

	if lang, ok := mapping[skill]; ok {
		return lang
	}
	return "go"
}
