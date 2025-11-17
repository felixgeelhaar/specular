package router

import (
	"time"

	"github.com/felixgeelhaar/specular/internal/provider"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

// Provider represents an AI provider (Anthropic, OpenAI, etc.)
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
	ProviderLocal     Provider = "local"
)

// ModelType categorizes models by their capabilities
type ModelType string

const (
	ModelTypeCodegen     ModelType = "codegen"      // Code generation specialists
	ModelTypeLongContext ModelType = "long-context" // Models with large context windows
	ModelTypeAgentic     ModelType = "agentic"      // Multi-step reasoning models
	ModelTypeFast        ModelType = "fast"         // Low-latency models
	ModelTypeCheap       ModelType = "cheap"        // Budget-friendly models
)

// Model represents an AI model configuration
type Model struct {
	ID              string    `json:"id"`
	Provider        Provider  `json:"provider"`
	Name            string    `json:"name"`
	Type            ModelType `json:"type"`
	ContextWindow   int       `json:"context_window"`   // Tokens
	CostPerMToken   float64   `json:"cost_per_mtoken"`  // USD per million tokens
	MaxLatencyMs    int       `json:"max_latency_ms"`   // Expected max latency
	CapabilityScore float64   `json:"capability_score"` // 0-100 capability rating
	Available       bool      `json:"available"`        // Whether model is accessible
}

// ProviderConfig represents provider-specific configuration
type ProviderConfig struct {
	Name    Provider          `json:"name" yaml:"name"`
	APIKey  string            `json:"api_key" yaml:"api_key"`
	BaseURL string            `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Models  map[string]string `json:"models" yaml:"models"` // Map of capability to model name
	Enabled bool              `json:"enabled" yaml:"enabled"`
}

// RouterConfig represents the router configuration
type RouterConfig struct {
	Providers               []ProviderConfig `json:"providers" yaml:"providers"`
	BudgetUSD               float64          `json:"budget_usd" yaml:"budget_usd"`
	MaxLatencyMs            int              `json:"max_latency_ms" yaml:"max_latency_ms"`
	PreferCheap             bool             `json:"prefer_cheap" yaml:"prefer_cheap"`                           // Prefer cheaper models when possible
	FallbackModel           string           `json:"fallback_model" yaml:"fallback_model"`                       // Model to use if preferred unavailable
	EnableFallback          bool             `json:"enable_fallback" yaml:"enable_fallback"`                     // Enable fallback to alternative providers
	MaxRetries              int              `json:"max_retries" yaml:"max_retries"`                             // Maximum retry attempts (0 = no retries)
	RetryBackoffMs          int              `json:"retry_backoff_ms" yaml:"retry_backoff_ms"`                   // Initial backoff delay in milliseconds
	RetryMaxBackoffMs       int              `json:"retry_max_backoff_ms" yaml:"retry_max_backoff_ms"`           // Maximum backoff delay
	EnableContextValidation bool             `json:"enable_context_validation" yaml:"enable_context_validation"` // Validate context fits in model window
	AutoTruncate            bool             `json:"auto_truncate" yaml:"auto_truncate"`                         // Automatically truncate oversized contexts
	TruncationStrategy      string           `json:"truncation_strategy" yaml:"truncation_strategy"`             // Strategy: oldest, prompt, context, proportional
}

// RoutingRequest represents a request for model selection
type RoutingRequest struct {
	ModelHint   string // Hint from plan generator (codegen, long-context, agentic)
	Complexity  int    // Task complexity (1-10)
	Priority    string // Task priority (P0, P1, P2)
	ContextSize int    // Estimated context size in tokens
}

// RoutingResult represents the router's model selection
type RoutingResult struct {
	Model           *Model
	Reason          string  // Explanation for selection
	EstimatedCost   float64 // Estimated cost in USD
	EstimatedTokens int     // Estimated token usage
}

// Usage represents AI model usage tracking
type Usage struct {
	Model     string       `json:"model"`
	Provider  Provider     `json:"provider"`
	Tokens    int          `json:"tokens"`
	CostUSD   float64      `json:"cost_usd"`
	LatencyMs int          `json:"latency_ms"`
	Timestamp time.Time    `json:"timestamp"`
	TaskID    types.TaskID `json:"task_id,omitempty"`
	Success   bool         `json:"success"`
}

// Budget tracks spending against limits
type Budget struct {
	LimitUSD     float64 `json:"limit_usd"`
	SpentUSD     float64 `json:"spent_usd"`
	RemainingUSD float64 `json:"remaining_usd"`
	UsageCount   int     `json:"usage_count"`
}

// GenerateRequest represents a request to generate AI content
type GenerateRequest struct {
	// Prompt is the main input
	Prompt string `json:"prompt"`

	// SystemPrompt sets system-level instructions
	SystemPrompt string `json:"system_prompt,omitempty"`

	// Model selection hints
	ModelHint  string `json:"model_hint,omitempty"` // codegen, agentic, fast, cheap, long-context
	Complexity int    `json:"complexity,omitempty"` // 1-10 scale
	Priority   string `json:"priority,omitempty"`   // P0, P1, P2

	// Generation parameters
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
	Tools       []provider.Tool    `json:"tools,omitempty"`
	Context     []provider.Message `json:"context,omitempty"`
	ContextSize int                `json:"context_size,omitempty"` // Estimated context in tokens

	// Metadata
	TaskID types.TaskID `json:"task_id,omitempty"`
}

// GenerateResponse represents the response from AI generation
type GenerateResponse struct {
	// Generated content
	Content string `json:"content"`

	// Model information
	Model    string   `json:"model"`    // Model ID that was used
	Provider Provider `json:"provider"` // Provider that handled the request

	// Token usage
	TokensUsed   int `json:"tokens_used"`
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`

	// Cost and performance
	CostUSD float64       `json:"cost_usd"`
	Latency time.Duration `json:"latency"`

	// Completion information
	FinishReason    string              `json:"finish_reason"`
	SelectionReason string              `json:"selection_reason"` // Why this model was selected
	ToolCalls       []provider.ToolCall `json:"tool_calls,omitempty"`

	// Error information
	Error string `json:"error,omitempty"`
}

// StreamChunk represents a chunk of streaming content
type StreamChunk struct {
	Content string `json:"content"` // Full content so far
	Delta   string `json:"delta"`   // Incremental text added
	Done    bool   `json:"done"`    // Whether stream is complete
	Error   error  `json:"error,omitempty"`
}
