package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AnthropicProvider implements the ProviderClient interface for Anthropic Claude API
type AnthropicProvider struct {
	apiKey     string
	baseURL    string
	client     *http.Client
	config     *ProviderConfig
	model      string
	maxTokens  int
	trustLevel TrustLevel
}

// Anthropic API request/response structures
type anthropicRequest struct {
	Model      string              `json:"model"`
	Messages   []anthropicMessage  `json:"messages"`
	System     string              `json:"system,omitempty"`
	MaxTokens  int                 `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []anthropicContent `json:"content"`
	Model        string             `json:"model"`
	StopReason   string             `json:"stop_reason,omitempty"`
	StopSequence string             `json:"stop_sequence,omitempty"`
	Usage        anthropicUsage     `json:"usage"`
	Error        *anthropicError    `json:"error,omitempty"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewAnthropicProvider creates a new Anthropic provider instance
func NewAnthropicProvider(config *ProviderConfig) (*AnthropicProvider, error) {
	// Extract API key from config
	apiKey, ok := config.Config["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("api_key not found in provider config")
	}

	// Get base URL (defaults to Anthropic)
	baseURL, ok := config.Config["base_url"].(string)
	if !ok || baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	// Get default model (if specified)
	model := "claude-sonnet-3.5" // default
	if modelVal, ok := config.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	// Get max tokens (if specified)
	maxTokens := 4096 // Anthropic requires max_tokens
	if maxVal, ok := config.Config["max_tokens"].(int); ok {
		maxTokens = maxVal
	}

	// Get trust level from config capabilities
	trustLevel := TrustLevelBuiltin
	if caps, ok := config.Config["capabilities"].(map[string]interface{}); ok {
		if tl, ok := caps["trust_level"].(string); ok {
			trustLevel = TrustLevel(tl)
		}
	}

	return &AnthropicProvider{
		apiKey:     apiKey,
		baseURL:    baseURL,
		client:     &http.Client{Timeout: 120 * time.Second},
		config:     config,
		model:      model,
		maxTokens:  maxTokens,
		trustLevel: trustLevel,
	}, nil
}

// Generate implements ProviderClient.Generate
func (p *AnthropicProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	// Build Anthropic request
	anthReq := p.buildRequest(req, false)

	// Marshal request
	reqBody, err := json.Marshal(anthReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Check for HTTP errors
	if httpResp.StatusCode != http.StatusOK {
		var errResp anthropicResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != nil {
			return nil, fmt.Errorf("anthropic error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("http error %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Parse response
	var anthResp anthropicResponse
	if err := json.Unmarshal(respBody, &anthResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Extract content
	content := ""
	if len(anthResp.Content) > 0 {
		content = anthResp.Content[0].Text
	}

	totalTokens := anthResp.Usage.InputTokens + anthResp.Usage.OutputTokens

	return &GenerateResponse{
		Content:      content,
		TokensUsed:   totalTokens,
		InputTokens:  anthResp.Usage.InputTokens,
		OutputTokens: anthResp.Usage.OutputTokens,
		Model:        anthResp.Model,
		Latency:      time.Since(startTime),
		FinishReason: anthResp.StopReason,
		Provider:     p.config.Name,
	}, nil
}

// Stream implements ProviderClient.Stream
func (p *AnthropicProvider) Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error) {
	chunkChan := make(chan StreamChunk, 10)

	// Build Anthropic request with streaming enabled
	anthReq := p.buildRequest(req, true)

	// Marshal request
	reqBody, err := json.Marshal(anthReq)
	if err != nil {
		close(chunkChan)
		return chunkChan, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(reqBody))
	if err != nil {
		close(chunkChan)
		return chunkChan, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Send request
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		close(chunkChan)
		return chunkChan, fmt.Errorf("send request: %w", err)
	}

	// Start goroutine to read stream
	go p.readStream(httpResp, chunkChan)

	return chunkChan, nil
}

// readStream reads the SSE stream from Anthropic
func (p *AnthropicProvider) readStream(resp *http.Response, chunkChan chan StreamChunk) {
	defer close(chunkChan)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	fullContent := ""

	for scanner.Scan() {
		line := scanner.Text()

		// SSE format: "data: {...}" or "event: <type>"
		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")
			if eventType == "message_stop" {
				// End of stream
				chunkChan <- StreamChunk{
					Content:   fullContent,
					Delta:     "",
					Done:      true,
					Timestamp: time.Now(),
				}
				return
			}
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Parse chunk
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			chunkChan <- StreamChunk{
				Error: fmt.Errorf("unmarshal chunk: %w", err),
				Done:  true,
			}
			return
		}

		// Extract delta from content_block_delta events
		if eventType, ok := event["type"].(string); ok && eventType == "content_block_delta" {
			if delta, ok := event["delta"].(map[string]interface{}); ok {
				if text, ok := delta["text"].(string); ok {
					fullContent += text

					chunkChan <- StreamChunk{
						Content:   fullContent,
						Delta:     text,
						Done:      false,
						Timestamp: time.Now(),
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		chunkChan <- StreamChunk{
			Error: fmt.Errorf("read stream: %w", err),
			Done:  true,
		}
	}
}

// buildRequest constructs an Anthropic API request from our GenerateRequest
func (p *AnthropicProvider) buildRequest(req *GenerateRequest, stream bool) *anthropicRequest {
	// Determine model
	model := p.model
	if reqModel, ok := req.Config["model"].(string); ok && reqModel != "" {
		model = reqModel
	}

	// Build messages (Anthropic doesn't include system in messages)
	messages := []anthropicMessage{}

	// Add context messages
	for _, msg := range req.Context {
		messages = append(messages, anthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add user prompt
	messages = append(messages, anthropicMessage{
		Role:    "user",
		Content: req.Prompt,
	})

	// Determine max tokens
	maxTokens := p.maxTokens
	if req.MaxTokens > 0 {
		maxTokens = req.MaxTokens
	}

	// Temperature
	temperature := 0.7
	if req.Temperature > 0 {
		temperature = req.Temperature
	}

	return &anthropicRequest{
		Model:       model,
		Messages:    messages,
		System:      req.SystemPrompt, // System prompt is separate in Anthropic
		MaxTokens:   maxTokens,
		Temperature: temperature,
		TopP:        req.TopP,
		Stream:      stream,
	}
}

// GetCapabilities implements ProviderClient.GetCapabilities
func (p *AnthropicProvider) GetCapabilities() *ProviderCapabilities {
	// Extract capabilities from config
	caps := &ProviderCapabilities{
		SupportsStreaming: true,
		SupportsTools:     true,
		SupportsMultiTurn: true,
		SupportsVision:    true,
		MaxContextTokens:  200000, // Claude 3.5 default
		CostPer1KTokens:   0.0,    // Will be set by router based on model
	}

	if configCaps, ok := p.config.Config["capabilities"].(map[string]interface{}); ok {
		if streaming, ok := configCaps["streaming"].(bool); ok {
			caps.SupportsStreaming = streaming
		}
		if tools, ok := configCaps["tools"].(bool); ok {
			caps.SupportsTools = tools
		}
		if multiTurn, ok := configCaps["multi_turn"].(bool); ok {
			caps.SupportsMultiTurn = multiTurn
		}
		if maxCtx, ok := configCaps["max_context_tokens"].(int); ok {
			caps.MaxContextTokens = maxCtx
		}
	}

	return caps
}

// GetInfo implements ProviderClient.GetInfo
func (p *AnthropicProvider) GetInfo() *ProviderInfo {
	return &ProviderInfo{
		Name:        p.config.Name,
		Version:     p.config.Version,
		Author:      "ai-dev",
		Type:        ProviderTypeAPI,
		TrustLevel:  p.trustLevel,
		Description: fmt.Sprintf("Anthropic Claude API provider: %s", p.baseURL),
	}
}

// IsAvailable implements ProviderClient.IsAvailable
func (p *AnthropicProvider) IsAvailable() bool {
	return p.apiKey != ""
}

// Health implements ProviderClient.Health
func (p *AnthropicProvider) Health(ctx context.Context) error {
	// Simple health check: try a minimal request
	req := &anthropicRequest{
		Model:     p.model,
		Messages:  []anthropicMessage{{Role: "user", Content: "ping"}},
		MaxTokens: 1,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal health check: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("create health check request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Close implements ProviderClient.Close
func (p *AnthropicProvider) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}
