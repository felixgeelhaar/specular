package client

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

// Client represents a client for the Specular Platform API.
// This allows the free CLI to optionally call the enterprise platform for advanced features.
//
// Usage:
//   client := client.New("https://platform.specular.io", apiKey)
//   spec, err := client.GenerateSpec(ctx, request)
//
// Note: This client is currently a stub. Full implementation will be added in v2.0
// when the enterprise platform (specular-platform/) is created.
type Client struct {
	baseURL string
	apiKey  string
	// TODO v2.0: Add HTTP client, retry logic, etc.
}

// New creates a new Specular Platform API client
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// GenerateSpecRequest represents a request to generate a product specification
type GenerateSpecRequest struct {
	Prompt      string            `json:"prompt"`
	Context     string            `json:"context,omitempty"`
	Constraints map[string]string `json:"constraints,omitempty"`
}

// GenerateSpec generates a product specification using the platform API
//
// Note: This is a stub. Implementation will be added in v2.0.
func (c *Client) GenerateSpec(ctx context.Context, req *GenerateSpecRequest) (*types.ProductSpec, error) {
	// TODO v2.0: Implement platform API call
	return nil, fmt.Errorf("platform API not yet implemented (coming in v2.0)")
}

// GeneratePlanRequest represents a request to generate an execution plan
type GeneratePlanRequest struct {
	Spec *types.ProductSpec `json:"spec"`
}

// GeneratePlan generates an execution plan using the platform API
//
// Note: This is a stub. Implementation will be added in v2.0.
func (c *Client) GeneratePlan(ctx context.Context, req *GeneratePlanRequest) (*types.Plan, error) {
	// TODO v2.0: Implement platform API call
	return nil, fmt.Errorf("platform API not yet implemented (coming in v2.0)")
}

// Health checks if the platform API is accessible
//
// Note: This is a stub. Implementation will be added in v2.0.
func (c *Client) Health(ctx context.Context) error {
	// TODO v2.0: Implement health check endpoint call
	return fmt.Errorf("platform API not yet implemented (coming in v2.0)")
}
