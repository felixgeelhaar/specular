package provider

import (
	"context"
	"time"
)

// ProviderClient is the universal interface that all AI providers must implement.
// This interface supports multiple provider types: API clients, CLI tools, and local models.
type ProviderClient interface {
	// Generate sends a prompt and returns a complete response.
	// This is the primary method for non-streaming use cases.
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// Stream sends a prompt and returns a channel of response chunks.
	// This allows for real-time streaming of long responses.
	// Returns a channel that will be closed when streaming completes.
	Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error)

	// GetCapabilities returns what this provider supports (streaming, tools, etc.)
	GetCapabilities() *ProviderCapabilities

	// GetInfo returns metadata about the provider (name, version, author)
	GetInfo() *ProviderInfo

	// IsAvailable checks if the provider is accessible and ready to use.
	// Returns true if the provider can handle requests.
	IsAvailable() bool

	// Health performs a health check on the provider.
	// Returns nil if healthy, error describing the problem otherwise.
	Health(ctx context.Context) error

	// Close cleans up any resources used by the provider.
	// Should be called when the provider is no longer needed.
	Close() error
}

// ProviderCapabilities describes what features a provider supports
type ProviderCapabilities struct {
	// SupportsStreaming indicates if the provider can stream responses
	SupportsStreaming bool

	// SupportsTools indicates if the provider supports tool/function calling
	SupportsTools bool

	// SupportsMultiTurn indicates if the provider maintains conversation context
	SupportsMultiTurn bool

	// SupportsVision indicates if the provider can process images
	SupportsVision bool

	// MaxContextTokens is the maximum context window size
	MaxContextTokens int

	// CostPer1KTokens is the cost in USD per 1000 tokens (input + output combined)
	// Set to 0 for local/free providers
	CostPer1KTokens float64
}

// ProviderInfo contains metadata about a provider
type ProviderInfo struct {
	// Name is the provider identifier (e.g., "openai", "claude-cli", "ollama")
	Name string

	// Version is the provider version (e.g., "1.0.0")
	Version string

	// Author is the provider author (e.g., "ai-dev-plugins")
	Author string

	// Type is the provider implementation type (api, cli, grpc, native)
	Type ProviderType

	// TrustLevel indicates the security trust level (builtin, verified, community)
	TrustLevel TrustLevel

	// Description is a human-readable description of the provider
	Description string
}

// ProviderType represents the implementation type of a provider
type ProviderType string

const (
	// ProviderTypeAPI is an HTTP API client
	ProviderTypeAPI ProviderType = "api"

	// ProviderTypeCLI is a command-line executable
	ProviderTypeCLI ProviderType = "cli"

	// ProviderTypeGRPC is a gRPC service
	ProviderTypeGRPC ProviderType = "grpc"

	// ProviderTypeNative is a Go native plugin (.so file)
	ProviderTypeNative ProviderType = "native"
)

// TrustLevel represents the security trust level of a provider
type TrustLevel string

const (
	// TrustLevelBuiltin providers ship with ai-dev and have full trust
	TrustLevelBuiltin TrustLevel = "builtin"

	// TrustLevelVerified providers are from trusted sources with signed releases
	TrustLevelVerified TrustLevel = "verified"

	// TrustLevelCommunity providers are from unknown sources and run sandboxed
	TrustLevelCommunity TrustLevel = "community"
)

// StreamChunk represents a single chunk in a streaming response
type StreamChunk struct {
	// Content is the text content of this chunk
	Content string

	// Delta is the incremental text added (for efficient updates)
	Delta string

	// Done indicates if this is the final chunk
	Done bool

	// TokensUsed is updated in the final chunk
	TokensUsed int

	// Error contains any error that occurred (in the final chunk)
	Error error

	// Timestamp is when this chunk was generated
	Timestamp time.Time
}
