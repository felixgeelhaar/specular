package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAnthropicProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ProviderConfig{
				Name:    "anthropic",
				Type:    ProviderTypeAPI,
				Enabled: true,
				Config: map[string]interface{}{
					"api_key":  "test-key",
					"base_url": "https://api.anthropic.com/v1",
				},
			},
			wantErr: false,
		},
		{
			name: "missing api_key",
			config: &ProviderConfig{
				Name:    "anthropic",
				Type:    ProviderTypeAPI,
				Enabled: true,
				Config:  map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "defaults to anthropic base_url",
			config: &ProviderConfig{
				Name:    "anthropic",
				Type:    ProviderTypeAPI,
				Enabled: true,
				Config: map[string]interface{}{
					"api_key": "test-key",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAnthropicProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAnthropicProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewAnthropicProvider() returned nil provider without error")
			}
		})
	}
}

func TestAnthropicProvider_Generate(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("unexpected api key header: %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("unexpected version header: %s", r.Header.Get("anthropic-version"))
		}

		// Parse request
		var req anthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		// Verify request fields
		if req.Model != "claude-sonnet-3.5" {
			t.Errorf("unexpected model: %s", req.Model)
		}
		if len(req.Messages) == 0 {
			t.Error("no messages in request")
		}
		if req.MaxTokens == 0 {
			t.Error("max_tokens not set")
		}

		// Send mock response
		resp := anthropicResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []anthropicContent{
				{
					Type: "text",
					Text: "The answer is 4.",
				},
			},
			Model:      "claude-sonnet-3.5",
			StopReason: "end_turn",
			Usage: anthropicUsage{
				InputTokens:  10,
				OutputTokens: 5,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock server
	provider, err := NewAnthropicProvider(&ProviderConfig{
		Name:    "anthropic",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Config: map[string]interface{}{
			"api_key":  "test-key",
			"base_url": server.URL,
		},
	})
	if err != nil {
		t.Fatalf("NewAnthropicProvider() error = %v", err)
	}

	// Test generation
	ctx := context.Background()
	resp, err := provider.Generate(ctx, &GenerateRequest{
		Prompt:      "What is 2 + 2?",
		Temperature: 0.7,
		MaxTokens:   100,
	})

	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify response
	if resp.Content != "The answer is 4." {
		t.Errorf("unexpected content: %s", resp.Content)
	}
	if resp.TokensUsed != 15 {
		t.Errorf("unexpected tokens used: %d", resp.TokensUsed)
	}
	if resp.InputTokens != 10 {
		t.Errorf("unexpected input tokens: %d", resp.InputTokens)
	}
	if resp.OutputTokens != 5 {
		t.Errorf("unexpected output tokens: %d", resp.OutputTokens)
	}
	if resp.Model != "claude-sonnet-3.5" {
		t.Errorf("unexpected model: %s", resp.Model)
	}
	if resp.FinishReason != "end_turn" {
		t.Errorf("unexpected finish reason: %s", resp.FinishReason)
	}
	if resp.Provider != "anthropic" {
		t.Errorf("unexpected provider: %s", resp.Provider)
	}
}

func TestAnthropicProvider_Generate_WithSystemPrompt(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req anthropicRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify system prompt is separate field
		if req.System != "You are helpful." {
			t.Errorf("unexpected system prompt: %s", req.System)
		}

		// Send response
		resp := anthropicResponse{
			Content: []anthropicContent{
				{Type: "text", Text: "OK"},
			},
			Usage: anthropicUsage{
				InputTokens:  5,
				OutputTokens: 2,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider(&ProviderConfig{
		Name: "anthropic",
		Config: map[string]interface{}{
			"api_key":  "test-key",
			"base_url": server.URL,
		},
	})

	ctx := context.Background()
	_, err := provider.Generate(ctx, &GenerateRequest{
		Prompt:       "Hello",
		SystemPrompt: "You are helpful.",
	})

	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestAnthropicProvider_Generate_Error(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    string
	}{
		{
			name:       "http 401 unauthorized",
			statusCode: http.StatusUnauthorized,
			response: anthropicResponse{
				Error: &anthropicError{
					Type:    "authentication_error",
					Message: "Invalid API key",
				},
			},
			wantErr: "Invalid API key",
		},
		{
			name:       "http 429 rate limit",
			statusCode: http.StatusTooManyRequests,
			response: anthropicResponse{
				Error: &anthropicError{
					Type:    "rate_limit_error",
					Message: "Rate limit exceeded",
				},
			},
			wantErr: "Rate limit exceeded",
		},
		{
			name:       "http 500 server error",
			statusCode: http.StatusInternalServerError,
			response:   "Internal server error",
			wantErr:    "http error 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			provider, _ := NewAnthropicProvider(&ProviderConfig{
				Name: "anthropic",
				Config: map[string]interface{}{
					"api_key":  "test-key",
					"base_url": server.URL,
				},
			})

			ctx := context.Background()
			_, err := provider.Generate(ctx, &GenerateRequest{
				Prompt: "test",
			})

			if err == nil {
				t.Fatal("Generate() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Generate() error = %v, want substring %s", err, tt.wantErr)
			}
		})
	}
}

func TestAnthropicProvider_Stream(t *testing.T) {
	// Create mock SSE server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming is requested
		var req anthropicRequest
		json.NewDecoder(r.Body).Decode(&req)
		if !req.Stream {
			t.Error("stream not requested")
		}

		// Send SSE chunks
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Message start event
		w.Write([]byte("event: message_start\n"))
		w.Write([]byte("data: {\"type\":\"message_start\"}\n\n"))
		flusher.Flush()

		// Content block delta 1
		delta1 := map[string]interface{}{
			"type": "content_block_delta",
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": "The ",
			},
		}
		data, _ := json.Marshal(delta1)
		w.Write([]byte("event: content_block_delta\n"))
		w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()

		// Content block delta 2
		delta2 := map[string]interface{}{
			"type": "content_block_delta",
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": "answer ",
			},
		}
		data, _ = json.Marshal(delta2)
		w.Write([]byte("event: content_block_delta\n"))
		w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()

		// Content block delta 3
		delta3 := map[string]interface{}{
			"type": "content_block_delta",
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": "is 4.",
			},
		}
		data, _ = json.Marshal(delta3)
		w.Write([]byte("event: content_block_delta\n"))
		w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()

		// Message stop event
		w.Write([]byte("event: message_stop\n"))
		w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider(&ProviderConfig{
		Name: "anthropic",
		Config: map[string]interface{}{
			"api_key":  "test-key",
			"base_url": server.URL,
		},
	})

	ctx := context.Background()
	chunkChan, err := provider.Stream(ctx, &GenerateRequest{
		Prompt: "What is 2 + 2?",
	})

	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	// Collect chunks
	var chunks []StreamChunk
	for chunk := range chunkChan {
		chunks = append(chunks, chunk)
	}

	// Verify chunks
	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(chunks))
	}

	// Verify content builds up
	if chunks[0].Delta != "The " {
		t.Errorf("chunk 0 delta = %s, want 'The '", chunks[0].Delta)
	}
	if chunks[1].Content != "The answer " {
		t.Errorf("chunk 1 content = %s, want 'The answer '", chunks[1].Content)
	}
	if chunks[2].Content != "The answer is 4." {
		t.Errorf("chunk 2 content = %s, want 'The answer is 4.'", chunks[2].Content)
	}

	// Verify final chunk is marked done
	if !chunks[3].Done {
		t.Error("final chunk not marked as done")
	}
	if chunks[3].Content != "The answer is 4." {
		t.Errorf("final chunk content = %s, want 'The answer is 4.'", chunks[3].Content)
	}
}

func TestAnthropicProvider_GetCapabilities(t *testing.T) {
	provider, _ := NewAnthropicProvider(&ProviderConfig{
		Name: "anthropic",
		Config: map[string]interface{}{
			"api_key": "test-key",
			"capabilities": map[string]interface{}{
				"streaming":          true,
				"tools":              true,
				"multi_turn":         true,
				"max_context_tokens": 200000,
			},
		},
	})

	caps := provider.GetCapabilities()

	if !caps.SupportsStreaming {
		t.Error("expected streaming support")
	}
	if !caps.SupportsTools {
		t.Error("expected tools support")
	}
	if !caps.SupportsMultiTurn {
		t.Error("expected multi-turn support")
	}
	if !caps.SupportsVision {
		t.Error("expected vision support")
	}
	if caps.MaxContextTokens != 200000 {
		t.Errorf("max context tokens = %d, want 200000", caps.MaxContextTokens)
	}
}

func TestAnthropicProvider_GetInfo(t *testing.T) {
	provider, _ := NewAnthropicProvider(&ProviderConfig{
		Name:    "anthropic",
		Version: "1.0.0",
		Config: map[string]interface{}{
			"api_key": "test-key",
		},
	})

	info := provider.GetInfo()

	if info.Name != "anthropic" {
		t.Errorf("name = %s, want 'anthropic'", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("version = %s, want '1.0.0'", info.Version)
	}
	if info.Type != ProviderTypeAPI {
		t.Errorf("type = %s, want %s", info.Type, ProviderTypeAPI)
	}
	if info.Author != "ai-dev" {
		t.Errorf("author = %s, want 'ai-dev'", info.Author)
	}
}

func TestAnthropicProvider_IsAvailable(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		available bool
	}{
		{
			name:      "with api key",
			apiKey:    "test-key",
			available: true,
		},
		{
			name:      "without api key",
			apiKey:    "",
			available: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &AnthropicProvider{
				apiKey: tt.apiKey,
			}

			available := provider.IsAvailable()
			if available != tt.available {
				t.Errorf("IsAvailable() = %v, want %v", available, tt.available)
			}
		})
	}
}

func TestAnthropicProvider_Health(t *testing.T) {
	// Mock successful health check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/messages" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(anthropicResponse{
				Content: []anthropicContent{
					{Type: "text", Text: "pong"},
				},
				Usage: anthropicUsage{
					InputTokens:  1,
					OutputTokens: 1,
				},
			})
		}
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider(&ProviderConfig{
		Name: "anthropic",
		Config: map[string]interface{}{
			"api_key":  "test-key",
			"base_url": server.URL,
		},
	})

	ctx := context.Background()
	err := provider.Health(ctx)
	if err != nil {
		t.Errorf("Health() error = %v, want nil", err)
	}
}

func TestAnthropicProvider_Health_Error(t *testing.T) {
	// Mock failed health check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(anthropicResponse{
			Error: &anthropicError{
				Type:    "authentication_error",
				Message: "Invalid API key",
			},
		})
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider(&ProviderConfig{
		Name: "anthropic",
		Config: map[string]interface{}{
			"api_key":  "invalid-key",
			"base_url": server.URL,
		},
	})

	ctx := context.Background()
	err := provider.Health(ctx)
	if err == nil {
		t.Error("Health() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Health() error = %v, want substring '401'", err)
	}
}

func TestAnthropicProvider_Close(t *testing.T) {
	provider, _ := NewAnthropicProvider(&ProviderConfig{
		Name: "anthropic",
		Config: map[string]interface{}{
			"api_key": "test-key",
		},
	})

	err := provider.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}
