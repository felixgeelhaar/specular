package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

// Client represents a client for the Specular Platform API.
// This allows the free CLI to optionally call the enterprise platform for advanced features.
//
// Usage:
//
//	client := client.New("https://platform.specular.io", apiKey)
//	spec, err := client.GenerateSpec(ctx, request)
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
}

// Config holds client configuration options
type Config struct {
	// MaxRetries is the maximum number of retry attempts (default: 3)
	MaxRetries int
	// RetryDelay is the initial delay between retries (default: 1s)
	RetryDelay time.Duration
	// Timeout is the HTTP client timeout (default: 30s)
	Timeout time.Duration
}

// DefaultConfig returns the default client configuration
func DefaultConfig() *Config {
	return &Config{
		MaxRetries: 3,
		RetryDelay: time.Second,
		Timeout:    30 * time.Second,
	}
}

// New creates a new Specular Platform API client with default configuration
func New(baseURL, apiKey string) *Client {
	return NewWithConfig(baseURL, apiKey, DefaultConfig())
}

// NewWithConfig creates a new client with custom configuration
func NewWithConfig(baseURL, apiKey string, cfg *Config) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		maxRetries: cfg.MaxRetries,
		retryDelay: cfg.RetryDelay,
	}
}

// GenerateSpecRequest represents a request to generate a product specification
type GenerateSpecRequest struct {
	Prompt      string            `json:"prompt"`
	Context     string            `json:"context,omitempty"`
	Constraints map[string]string `json:"constraints,omitempty"`
}

// APIError represents an error response from the platform API
type APIError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id,omitempty"`
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("platform API error (status %d, request_id %s): %s",
			e.StatusCode, e.RequestID, e.Message)
	}
	return fmt.Sprintf("platform API error (status %d): %s", e.StatusCode, e.Message)
}

// doRequest performs an HTTP request with retry logic and exponential backoff
func (c *Client) doRequest(ctx context.Context, method, path string, reqBody, respBody interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: delay * 2^(attempt-1)
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * c.retryDelay
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := c.doSingleRequest(ctx, method, path, reqBody, respBody)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on client errors (4xx) or context cancellation
		if apiErr, ok := err.(*APIError); ok {
			if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
				return err
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doSingleRequest performs a single HTTP request without retries
func (c *Client) doSingleRequest(ctx context.Context, method, path string, reqBody, respBody interface{}) error {
	url := c.baseURL + path

	var bodyReader io.Reader
	if reqBody != nil {
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("User-Agent", "specular-cli/1.6.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // deferred close, error already handled via response reading

	// Read response body
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	// Handle non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			RequestID:  resp.Header.Get("X-Request-ID"),
		}

		// Try to parse error message from response
		var errResp struct {
			Error string `json:"error"`
		}
		if unmarshalErr := json.Unmarshal(respBytes, &errResp); unmarshalErr == nil && errResp.Error != "" {
			apiErr.Message = errResp.Error
		} else {
			apiErr.Message = string(respBytes)
		}

		return apiErr
	}

	// Parse successful response
	if respBody != nil {
		err = json.Unmarshal(respBytes, respBody)
		if err != nil {
			return fmt.Errorf("unmarshaling response body: %w", err)
		}
	}

	return nil
}

// GenerateSpec generates a product specification using the platform API
func (c *Client) GenerateSpec(ctx context.Context, req *GenerateSpecRequest) (*types.ProductSpec, error) {
	var spec types.ProductSpec
	err := c.doRequest(ctx, http.MethodPost, "/v1/spec/generate", req, &spec)
	if err != nil {
		return nil, fmt.Errorf("generating spec: %w", err)
	}
	return &spec, nil
}

// GeneratePlanRequest represents a request to generate an execution plan
type GeneratePlanRequest struct {
	Spec *types.ProductSpec `json:"spec"`
}

// GeneratePlan generates an execution plan using the platform API
func (c *Client) GeneratePlan(ctx context.Context, req *GeneratePlanRequest) (*types.Plan, error) {
	var plan types.Plan
	err := c.doRequest(ctx, http.MethodPost, "/v1/plan/generate", req, &plan)
	if err != nil {
		return nil, fmt.Errorf("generating plan: %w", err)
	}
	return &plan, nil
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

// Health checks if the platform API is accessible
func (c *Client) Health(ctx context.Context) error {
	var health HealthResponse
	err := c.doRequest(ctx, http.MethodGet, "/health", nil, &health)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if health.Status != "ok" && health.Status != "healthy" {
		return fmt.Errorf("unhealthy status: %s", health.Status)
	}

	return nil
}
