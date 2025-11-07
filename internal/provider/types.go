package provider

import "time"

// GenerateRequest contains all parameters for generating a response
type GenerateRequest struct {
	// Prompt is the main input text for the model
	Prompt string `json:"prompt"`

	// SystemPrompt sets the system-level instructions (e.g., "You are a helpful assistant")
	SystemPrompt string `json:"system_prompt,omitempty"`

	// MaxTokens limits the maximum response length
	// Set to 0 to use provider default
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0 = deterministic, 1.0+ = creative)
	// Typical range: 0.0 to 2.0
	Temperature float64 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling (alternative to temperature)
	// Range: 0.0 to 1.0
	TopP float64 `json:"top_p,omitempty"`

	// Tools available for the model to call (if provider supports tool use)
	Tools []Tool `json:"tools,omitempty"`

	// Context provides previous messages for multi-turn conversations
	Context []Message `json:"context,omitempty"`

	// Config contains provider-specific configuration options
	// Examples: {"model": "gpt-4", "stream": true, "stop": ["\n"]}
	Config map[string]interface{} `json:"config,omitempty"`

	// Metadata for tracking and debugging
	Metadata map[string]string `json:"metadata,omitempty"`
}

// GenerateResponse contains the model's response
type GenerateResponse struct {
	// Content is the generated text
	Content string `json:"content"`

	// TokensUsed is the total tokens consumed (input + output)
	TokensUsed int `json:"tokens_used"`

	// InputTokens is tokens in the prompt
	InputTokens int `json:"input_tokens,omitempty"`

	// OutputTokens is tokens in the response
	OutputTokens int `json:"output_tokens,omitempty"`

	// Model is the actual model that generated the response
	// May differ from requested model (e.g., if fallback occurred)
	Model string `json:"model"`

	// Latency is how long the generation took
	Latency time.Duration `json:"latency"`

	// FinishReason explains why generation stopped
	// Common values: "stop" (natural end), "length" (max tokens), "error"
	FinishReason string `json:"finish_reason"`

	// ToolCalls contains any tool calls the model made
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// Error contains any error message
	Error string `json:"error,omitempty"`

	// Provider is the name of the provider that handled this request
	Provider string `json:"provider"`

	// Metadata returned from the provider
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Message represents a single message in a conversation
type Message struct {
	// Role is who sent the message: "user", "assistant", or "system"
	Role string `json:"role"`

	// Content is the message text
	Content string `json:"content"`

	// ToolCalls are function calls made by the assistant
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// ToolCallID links a tool response to the original call
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// Tool describes a function the model can call
type Tool struct {
	// Type is typically "function"
	Type string `json:"type"`

	// Function contains the function definition
	Function ToolFunction `json:"function"`
}

// ToolFunction defines a callable function
type ToolFunction struct {
	// Name is the function identifier
	Name string `json:"name"`

	// Description explains what the function does
	Description string `json:"description"`

	// Parameters is a JSON Schema describing the function's parameters
	Parameters map[string]interface{} `json:"parameters"`
}

// ToolCall represents a function call made by the model
type ToolCall struct {
	// ID uniquely identifies this tool call
	ID string `json:"id"`

	// Type is typically "function"
	Type string `json:"type"`

	// Function contains the call details
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction contains function call details
type ToolCallFunction struct {
	// Name is the function being called
	Name string `json:"name"`

	// Arguments is a JSON string with the function arguments
	Arguments string `json:"arguments"`
}

// ProviderConfig represents a provider configuration from router.yaml
type ProviderConfig struct {
	// Name is the provider identifier
	Name string `yaml:"name" json:"name"`

	// Source is where to get the provider (github URL, "builtin", etc.)
	Source string `yaml:"source" json:"source"`

	// Type is the provider implementation type
	Type ProviderType `yaml:"type" json:"type"`

	// Enabled controls if this provider is active
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Config contains provider-specific configuration
	Config map[string]interface{} `yaml:"config" json:"config"`

	// Models maps model hints to actual model names
	// Example: {"codegen": "gpt-4", "fast": "gpt-3.5-turbo"}
	Models map[string]string `yaml:"models" json:"models"`

	// Version specifies the provider version (for source resolution)
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}
