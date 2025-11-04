package router

import "time"

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
	ID              string        `json:"id"`
	Provider        Provider      `json:"provider"`
	Name            string        `json:"name"`
	Type            ModelType     `json:"type"`
	ContextWindow   int           `json:"context_window"`    // Tokens
	CostPerMToken   float64       `json:"cost_per_mtoken"`   // USD per million tokens
	MaxLatencyMs    int           `json:"max_latency_ms"`    // Expected max latency
	CapabilityScore float64       `json:"capability_score"`  // 0-100 capability rating
	Available       bool          `json:"available"`         // Whether model is accessible
}

// ProviderConfig represents provider-specific configuration
type ProviderConfig struct {
	Name    Provider          `json:"name"`
	APIKey  string            `json:"api_key"`
	BaseURL string            `json:"base_url,omitempty"`
	Models  map[string]string `json:"models"` // Map of capability to model name
	Enabled bool              `json:"enabled"`
}

// RouterConfig represents the router configuration
type RouterConfig struct {
	Providers     []ProviderConfig `json:"providers"`
	BudgetUSD     float64          `json:"budget_usd"`
	MaxLatencyMs  int              `json:"max_latency_ms"`
	PreferCheap   bool             `json:"prefer_cheap"`    // Prefer cheaper models when possible
	FallbackModel string           `json:"fallback_model"`  // Model to use if preferred unavailable
}

// RoutingRequest represents a request for model selection
type RoutingRequest struct {
	ModelHint  string  // Hint from plan generator (codegen, long-context, agentic)
	Complexity int     // Task complexity (1-10)
	Priority   string  // Task priority (P0, P1, P2)
	ContextSize int    // Estimated context size in tokens
}

// RoutingResult represents the router's model selection
type RoutingResult struct {
	Model          *Model
	Reason         string  // Explanation for selection
	EstimatedCost  float64 // Estimated cost in USD
	EstimatedTokens int    // Estimated token usage
}

// Usage represents AI model usage tracking
type Usage struct {
	Model        string    `json:"model"`
	Provider     Provider  `json:"provider"`
	Tokens       int       `json:"tokens"`
	CostUSD      float64   `json:"cost_usd"`
	LatencyMs    int       `json:"latency_ms"`
	Timestamp    time.Time `json:"timestamp"`
	TaskID       string    `json:"task_id,omitempty"`
	Success      bool      `json:"success"`
}

// Budget tracks spending against limits
type Budget struct {
	LimitUSD     float64 `json:"limit_usd"`
	SpentUSD     float64 `json:"spent_usd"`
	RemainingUSD float64 `json:"remaining_usd"`
	UsageCount   int     `json:"usage_count"`
}
