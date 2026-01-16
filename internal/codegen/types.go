// Package codegen provides AI-powered code generation capabilities for Specular.
//
// The codegen package handles:
//   - Building prompts from feature specifications
//   - Calling AI providers via the router
//   - Parsing generated code from AI responses
//   - Writing generated files to disk
//
// Key types:
//   - Generator: Main orchestrator for code generation
//   - Config: Configuration for generation behavior
//   - GenerationResult: Result of a generation operation
//   - GeneratedFile: A single generated file with content and metadata
package codegen

import (
	"context"
	"time"

	"github.com/felixgeelhaar/specular/internal/router"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

// Config configures the code generator behavior
type Config struct {
	// OutputDir is the directory where generated files are written
	// Defaults to "." (project root)
	OutputDir string `json:"output_dir" yaml:"output_dir"`

	// Verbose enables detailed logging during generation
	Verbose bool `json:"verbose" yaml:"verbose"`

	// DryRun prevents writing files to disk (for testing/preview)
	DryRun bool `json:"dry_run" yaml:"dry_run"`

	// MaxTokens limits the AI response size
	MaxTokens int `json:"max_tokens" yaml:"max_tokens"`

	// Temperature controls AI creativity (0.0-1.0)
	Temperature float64 `json:"temperature" yaml:"temperature"`

	// Language override (defaults to skill-based detection)
	Language string `json:"language" yaml:"language"`
}

// DefaultConfig returns sensible defaults for code generation
func DefaultConfig() Config {
	return Config{
		OutputDir:   ".",
		Verbose:     false,
		DryRun:      false,
		MaxTokens:   4096,
		Temperature: 0.7,
	}
}

// GenerationResult contains the outcome of a code generation operation
type GenerationResult struct {
	// TaskID is the ID of the task that triggered generation
	TaskID types.TaskID `json:"task_id"`

	// FeatureID is the feature being implemented
	FeatureID types.FeatureID `json:"feature_id"`

	// Files contains all generated files
	Files []GeneratedFile `json:"files"`

	// AIMetadata contains information about the AI generation
	AIMetadata AIGenerationMetadata `json:"ai_metadata"`

	// Duration is how long generation took
	Duration time.Duration `json:"duration"`

	// Success indicates if generation completed without errors
	Success bool `json:"success"`

	// Error message if generation failed
	Error string `json:"error,omitempty"`
}

// GeneratedFile represents a single generated source file
type GeneratedFile struct {
	// Path is the relative file path (e.g., "cmd/main.go")
	Path string `json:"path"`

	// Content is the file content
	Content string `json:"content"`

	// Language is the programming language (go, typescript, sql, etc.)
	Language string `json:"language"`

	// Hash is the SHA-256 hash of the content
	Hash string `json:"hash"`

	// Size is the file size in bytes
	Size int `json:"size"`

	// Written indicates if the file was written to disk
	Written bool `json:"written"`
}

// AIGenerationMetadata tracks AI usage for audit/billing
type AIGenerationMetadata struct {
	// Model is the AI model used
	Model string `json:"model"`

	// Provider is the AI provider (anthropic, openai, etc.)
	Provider string `json:"provider"`

	// TokensUsed is the total token count
	TokensUsed int `json:"tokens_used"`

	// InputTokens is the prompt token count
	InputTokens int `json:"input_tokens"`

	// OutputTokens is the completion token count
	OutputTokens int `json:"output_tokens"`

	// CostUSD is the estimated cost in USD
	CostUSD float64 `json:"cost_usd"`

	// Latency is the AI response time
	Latency time.Duration `json:"latency"`

	// SelectionReason explains why this model was chosen
	SelectionReason string `json:"selection_reason"`
}

// CodeGenerationRequest is passed to the generator
type CodeGenerationRequest struct {
	// FeatureID to generate code for
	FeatureID types.FeatureID

	// FeatureTitle is the feature name
	FeatureTitle string

	// FeatureDesc is the detailed description
	FeatureDesc string

	// SuccessCriteria are the acceptance criteria
	SuccessCriteria []string

	// Skill determines the type of code to generate
	Skill string

	// ProductContext provides project context
	ProductContext string

	// APIs defines any API endpoints to implement
	APIs []APIEndpoint

	// Priority affects model selection
	Priority types.Priority

	// ExistingFiles provides context about existing code
	ExistingFiles map[string]string
}

// APIEndpoint represents an API endpoint to implement
type APIEndpoint struct {
	Method   string `json:"method"`
	Path     string `json:"path"`
	Request  string `json:"request,omitempty"`
	Response string `json:"response,omitempty"`
}

// RouterAdapter provides the interface needed from router.Router
type RouterAdapter interface {
	Generate(ctx context.Context, req router.GenerateRequest) (*router.GenerateResponse, error)
	GetBudget() *router.Budget
}

// Ensure router.Router implements RouterAdapter
var _ RouterAdapter = (*router.Router)(nil)
