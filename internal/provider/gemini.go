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

// GeminiProvider implements the ProviderClient interface for Google Gemini API
type GeminiProvider struct {
	apiKey     string
	baseURL    string
	client     *http.Client
	config     *ProviderConfig
	model      string
	maxTokens  int
	trustLevel TrustLevel
}

// Gemini API request/response structures
type geminiRequest struct {
	Contents          []geminiContent         `json:"contents"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
}

type geminiResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata *geminiUsage      `json:"usageMetadata,omitempty"`
	ModelVersion  string            `json:"modelVersion,omitempty"`
	Error         *geminiError      `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
	Index        int           `json:"index"`
}

type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// NewGeminiProvider creates a new Gemini provider instance
func NewGeminiProvider(config *ProviderConfig) (*GeminiProvider, error) {
	// Extract API key from config
	apiKey, ok := config.Config["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("api_key not found in provider config")
	}

	// Get base URL (defaults to Gemini)
	baseURL, ok := config.Config["base_url"].(string)
	if !ok || baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	// Get default model (if specified)
	model := "gemini-2.0-flash-exp" // default
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

	return &GeminiProvider{
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
func (p *GeminiProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	// Build Gemini request
	geminiReq := p.buildRequest(req)

	// Marshal request
	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Determine model to use (from request config or default)
	modelName := p.model
	if req.Config != nil {
		if model, ok := req.Config["model"].(string); ok && model != "" {
			modelName = model
		}
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, modelName, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	// Parse response
	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Check for API errors
	if geminiResp.Error != nil {
		return nil, fmt.Errorf("Gemini API error: %s (code: %d)", geminiResp.Error.Message, geminiResp.Error.Code)
	}

	// Convert to our response format
	return p.convertResponse(&geminiResp, time.Since(startTime), modelName)
}

// Stream implements ProviderClient.Stream
func (p *GeminiProvider) Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error) {
	chunkChan := make(chan StreamChunk, 10)

	// Build Gemini request
	geminiReq := p.buildRequest(req)

	// Marshal request
	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		close(chunkChan)
		return chunkChan, fmt.Errorf("marshal request: %w", err)
	}

	// Determine model to use
	modelName := p.model
	if req.Config != nil {
		if model, ok := req.Config["model"].(string); ok && model != "" {
			modelName = model
		}
	}

	// Create HTTP request for streaming
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse", p.baseURL, modelName, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		close(chunkChan)
		return chunkChan, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		close(chunkChan)
		return chunkChan, fmt.Errorf("send request: %w", err)
	}

	// Start goroutine to read streaming response
	go func() {
		defer close(chunkChan)
		defer httpResp.Body.Close()

		scanner := bufio.NewScanner(httpResp.Body)
		fullContent := ""
		var lastUsage *geminiUsage

		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines and non-data lines
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			// Extract JSON data
			data := strings.TrimPrefix(line, "data: ")
			if data == "" {
				continue
			}

			// Parse chunk
			var geminiResp geminiResponse
			if err := json.Unmarshal([]byte(data), &geminiResp); err != nil {
				chunkChan <- StreamChunk{
					Error: fmt.Errorf("parse chunk: %w", err),
					Done:  true,
				}
				return
			}

			// Check for errors
			if geminiResp.Error != nil {
				chunkChan <- StreamChunk{
					Error: fmt.Errorf("Gemini API error: %s", geminiResp.Error.Message),
					Done:  true,
				}
				return
			}

			// Extract content from candidates
			if len(geminiResp.Candidates) > 0 {
				candidate := geminiResp.Candidates[0]
				if len(candidate.Content.Parts) > 0 {
					delta := candidate.Content.Parts[0].Text
					fullContent += delta

					// Save usage metadata
					if geminiResp.UsageMetadata != nil {
						lastUsage = geminiResp.UsageMetadata
					}

					// Send chunk
					chunk := StreamChunk{
						Content:   fullContent,
						Delta:     delta,
						Done:      candidate.FinishReason != "",
						Timestamp: time.Now(),
					}

					// Include token count on final chunk
					if chunk.Done && lastUsage != nil {
						chunk.TokensUsed = lastUsage.TotalTokenCount
					}

					chunkChan <- chunk

					if chunk.Done {
						return
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			chunkChan <- StreamChunk{
				Error: fmt.Errorf("stream read error: %w", err),
				Done:  true,
			}
		}
	}()

	return chunkChan, nil
}

// buildRequest converts our GenerateRequest to Gemini format
func (p *GeminiProvider) buildRequest(req *GenerateRequest) *geminiRequest {
	geminiReq := &geminiRequest{
		Contents: []geminiContent{},
	}

	// Add system instruction if provided
	if req.SystemPrompt != "" {
		geminiReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		}
	}

	// Add context messages (multi-turn conversation)
	for _, msg := range req.Context {
		role := "user"
		if msg.Role == "assistant" {
			role = "model" // Gemini uses "model" instead of "assistant"
		}
		geminiReq.Contents = append(geminiReq.Contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: msg.Content}},
		})
	}

	// Add the main prompt
	geminiReq.Contents = append(geminiReq.Contents, geminiContent{
		Role:  "user",
		Parts: []geminiPart{{Text: req.Prompt}},
	})

	// Add generation config
	genConfig := &geminiGenerationConfig{}

	if req.Temperature > 0 {
		temp := req.Temperature
		genConfig.Temperature = &temp
	}

	if req.MaxTokens > 0 {
		genConfig.MaxOutputTokens = req.MaxTokens
	} else if p.maxTokens > 0 {
		genConfig.MaxOutputTokens = p.maxTokens
	}

	if req.TopP > 0 {
		topP := req.TopP
		genConfig.TopP = &topP
	}

	geminiReq.GenerationConfig = genConfig

	return geminiReq
}

// convertResponse converts Gemini response to our format
func (p *GeminiProvider) convertResponse(resp *geminiResponse, latency time.Duration, model string) (*GenerateResponse, error) {
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no content parts in response")
	}

	// Extract text content
	content := candidate.Content.Parts[0].Text

	// Build response
	result := &GenerateResponse{
		Content:      content,
		Model:        model,
		Provider:     "gemini",
		FinishReason: candidate.FinishReason,
		Latency:      latency,
	}

	// Add token usage
	if resp.UsageMetadata != nil {
		result.InputTokens = resp.UsageMetadata.PromptTokenCount
		result.OutputTokens = resp.UsageMetadata.CandidatesTokenCount
		result.TokensUsed = resp.UsageMetadata.TotalTokenCount
	}

	return result, nil
}

// GetCapabilities returns provider capabilities
func (p *GeminiProvider) GetCapabilities() *ProviderCapabilities {
	caps := &ProviderCapabilities{
		SupportsStreaming: true,
		SupportsTools:     true,
		SupportsMultiTurn: true,
		SupportsVision:    true,
		MaxContextTokens:  1000000, // Gemini 2.0 supports 1M tokens
		CostPer1KTokens:   0.0,     // Pricing varies by model
	}

	// Override with config if provided
	if configCaps, ok := p.config.Config["capabilities"].(map[string]interface{}); ok {
		if maxTokens, ok := configCaps["max_context_tokens"].(float64); ok {
			caps.MaxContextTokens = int(maxTokens)
		}
		if cost, ok := configCaps["cost_per_1k_tokens"].(float64); ok {
			caps.CostPer1KTokens = cost
		}
	}

	return caps
}

// GetInfo returns provider metadata
func (p *GeminiProvider) GetInfo() *ProviderInfo {
	return &ProviderInfo{
		Name:        p.config.Name,
		Version:     p.config.Version,
		Type:        ProviderTypeAPI,
		TrustLevel:  p.trustLevel,
		Description: "Google Gemini API provider with 1M context window",
	}
}

// IsAvailable checks if the provider is configured and accessible
func (p *GeminiProvider) IsAvailable() bool {
	return p.apiKey != ""
}

// Health performs a health check
func (p *GeminiProvider) Health(ctx context.Context) error {
	// Try a simple generation request
	req := &GenerateRequest{
		Prompt:      "Hello",
		MaxTokens:   10,
		Temperature: 0.1,
	}

	_, err := p.Generate(ctx, req)
	return err
}

// Close cleans up resources
func (p *GeminiProvider) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}
