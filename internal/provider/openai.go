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

// OpenAIProvider implements the ProviderClient interface for OpenAI API
type OpenAIProvider struct {
	apiKey     string
	baseURL    string
	client     *http.Client
	config     *ProviderConfig
	model      string
	maxTokens  int
	trustLevel TrustLevel
}

// OpenAI API request/response structures
type openAIRequest struct {
	Model       string           `json:"model"`
	Messages    []openAIMessage  `json:"messages"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	TopP        float64          `json:"top_p,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []openAIChoice  `json:"choices"`
	Usage   openAIUsage     `json:"usage"`
	Error   *openAIError    `json:"error,omitempty"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message,omitempty"`
	Delta        openAIMessage `json:"delta,omitempty"`
	FinishReason string        `json:"finish_reason,omitempty"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// NewOpenAIProvider creates a new OpenAI provider instance
func NewOpenAIProvider(config *ProviderConfig) (*OpenAIProvider, error) {
	// Extract API key from config
	apiKey, ok := config.Config["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("api_key not found in provider config")
	}

	// Get base URL (defaults to OpenAI)
	baseURL, ok := config.Config["base_url"].(string)
	if !ok || baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	// Get default model (if specified)
	model := "gpt-4o-mini" // default
	if modelVal, ok := config.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	// Get max tokens (if specified)
	maxTokens := 0
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

	return &OpenAIProvider{
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
func (p *OpenAIProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	// Build OpenAI request
	oaiReq := p.buildRequest(req, false)

	// Marshal request
	reqBody, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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
		var errResp openAIResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != nil {
			return nil, fmt.Errorf("openai error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("http error %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Parse response
	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Extract content
	content := ""
	finishReason := ""
	if len(oaiResp.Choices) > 0 {
		content = oaiResp.Choices[0].Message.Content
		finishReason = oaiResp.Choices[0].FinishReason
	}

	return &GenerateResponse{
		Content:      content,
		TokensUsed:   oaiResp.Usage.TotalTokens,
		InputTokens:  oaiResp.Usage.PromptTokens,
		OutputTokens: oaiResp.Usage.CompletionTokens,
		Model:        oaiResp.Model,
		Latency:      time.Since(startTime),
		FinishReason: finishReason,
		Provider:     p.config.Name,
	}, nil
}

// Stream implements ProviderClient.Stream
func (p *OpenAIProvider) Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error) {
	chunkChan := make(chan StreamChunk, 10)

	// Build OpenAI request with streaming enabled
	oaiReq := p.buildRequest(req, true)

	// Marshal request
	reqBody, err := json.Marshal(oaiReq)
	if err != nil {
		close(chunkChan)
		return chunkChan, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		close(chunkChan)
		return chunkChan, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
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

// readStream reads the SSE stream from OpenAI
func (p *OpenAIProvider) readStream(resp *http.Response, chunkChan chan StreamChunk) {
	defer close(chunkChan)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	fullContent := ""

	for scanner.Scan() {
		line := scanner.Text()

		// SSE format: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for end marker
		if data == "[DONE]" {
			chunkChan <- StreamChunk{
				Content: fullContent,
				Delta:   "",
				Done:    true,
				Timestamp: time.Now(),
			}
			return
		}

		// Parse chunk
		var oaiResp openAIResponse
		if err := json.Unmarshal([]byte(data), &oaiResp); err != nil {
			chunkChan <- StreamChunk{
				Error: fmt.Errorf("unmarshal chunk: %w", err),
				Done:  true,
			}
			return
		}

		// Extract delta
		if len(oaiResp.Choices) > 0 {
			delta := oaiResp.Choices[0].Delta.Content
			fullContent += delta

			chunkChan <- StreamChunk{
				Content:   fullContent,
				Delta:     delta,
				Done:      false,
				Timestamp: time.Now(),
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

// buildRequest constructs an OpenAI API request from our GenerateRequest
func (p *OpenAIProvider) buildRequest(req *GenerateRequest, stream bool) *openAIRequest {
	// Determine model
	model := p.model
	if reqModel, ok := req.Config["model"].(string); ok && reqModel != "" {
		model = reqModel
	}

	// Build messages
	messages := []openAIMessage{}

	// Add system prompt if present
	if req.SystemPrompt != "" {
		messages = append(messages, openAIMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// Add context messages
	for _, msg := range req.Context {
		messages = append(messages, openAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add user prompt
	messages = append(messages, openAIMessage{
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

	return &openAIRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		TopP:        req.TopP,
		Stream:      stream,
	}
}

// GetCapabilities implements ProviderClient.GetCapabilities
func (p *OpenAIProvider) GetCapabilities() *ProviderCapabilities {
	// Extract capabilities from config
	caps := &ProviderCapabilities{
		SupportsStreaming: true,
		SupportsTools:     true,
		SupportsMultiTurn: true,
		SupportsVision:    false,
		MaxContextTokens:  128000, // gpt-4o default
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
func (p *OpenAIProvider) GetInfo() *ProviderInfo {
	return &ProviderInfo{
		Name:        p.config.Name,
		Version:     p.config.Version,
		Author:      "ai-dev",
		Type:        ProviderTypeAPI,
		TrustLevel:  p.trustLevel,
		Description: fmt.Sprintf("OpenAI API provider: %s", p.baseURL),
	}
}

// IsAvailable implements ProviderClient.IsAvailable
func (p *OpenAIProvider) IsAvailable() bool {
	return p.apiKey != ""
}

// Health implements ProviderClient.Health
func (p *OpenAIProvider) Health(ctx context.Context) error {
	// Simple health check: try to list models
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("create health check request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// Close implements ProviderClient.Close
func (p *OpenAIProvider) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}
