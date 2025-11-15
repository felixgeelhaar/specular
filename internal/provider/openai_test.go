package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewOpenAIProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ProviderConfig{
				Name:    "openai",
				Type:    ProviderTypeAPI,
				Enabled: true,
				Config: map[string]interface{}{
					"api_key":  "test-key",
					"base_url": "https://api.openai.com/v1",
				},
			},
			wantErr: false,
		},
		{
			name: "missing api_key",
			config: &ProviderConfig{
				Name:    "openai",
				Type:    ProviderTypeAPI,
				Enabled: true,
				Config:  map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "defaults to openai base_url",
			config: &ProviderConfig{
				Name:    "openai",
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
			provider, err := NewOpenAIProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenAIProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewOpenAIProvider() returned nil provider without error")
			}
		})
	}
}

func TestOpenAIProvider_Generate(t *testing.T) {
	// Create mock server
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		// Parse request
		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		// Verify request fields
		if req.Model != "gpt-4o-mini" {
			t.Errorf("unexpected model: %s", req.Model)
		}
		if len(req.Messages) == 0 {
			t.Error("no messages in request")
		}

		// Send mock response
		resp := openAIResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
			Choices: []openAIChoice{
				{
					Index: 0,
					Message: openAIMessage{
						Role:    "assistant",
						Content: "The answer is 4.",
					},
					FinishReason: "stop",
				},
			},
			Usage: openAIUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock server
	provider, err := NewOpenAIProvider(&ProviderConfig{
		Name:    "openai",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Config: map[string]interface{}{
			"api_key":  "test-key",
			"base_url": server.URL,
		},
	})
	if err != nil {
		t.Fatalf("NewOpenAIProvider() error = %v", err)
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
	if resp.Model != "gpt-4o-mini" {
		t.Errorf("unexpected model: %s", resp.Model)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("unexpected finish reason: %s", resp.FinishReason)
	}
	if resp.Provider != "openai" {
		t.Errorf("unexpected provider: %s", resp.Provider)
	}
}

func TestOpenAIProvider_Generate_WithSystemPrompt(t *testing.T) {
	// Create mock server
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req openAIRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		// Verify system message
		if len(req.Messages) < 2 {
			t.Error("expected at least 2 messages (system + user)")
		}
		if req.Messages[0].Role != "system" {
			t.Errorf("first message should be system, got: %s", req.Messages[0].Role)
		}
		if req.Messages[0].Content != "You are helpful." {
			t.Errorf("unexpected system prompt: %s", req.Messages[0].Content)
		}

		// Send response
		resp := openAIResponse{
			Model: "gpt-4o-mini",
			Choices: []openAIChoice{
				{Message: openAIMessage{Content: "OK"}},
			},
			Usage: openAIUsage{TotalTokens: 10},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider(&ProviderConfig{
		Name: "openai",
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

func TestOpenAIProvider_Generate_Error(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    string
	}{
		{
			name:       "http 401 unauthorized",
			statusCode: http.StatusUnauthorized,
			response: openAIResponse{
				Error: &openAIError{
					Message: "Invalid API key",
					Type:    "invalid_request_error",
				},
			},
			wantErr: "Invalid API key",
		},
		{
			name:       "http 429 rate limit",
			statusCode: http.StatusTooManyRequests,
			response: openAIResponse{
				Error: &openAIError{
					Message: "Rate limit exceeded",
					Type:    "rate_limit_error",
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
			server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			provider, _ := NewOpenAIProvider(&ProviderConfig{
				Name: "openai",
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

func TestOpenAIProvider_Stream(t *testing.T) {
	// Create mock SSE server
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming is requested
		var req openAIRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if !req.Stream {
			t.Error("stream not requested")
		}

		// Send SSE chunks
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Chunk 1
		chunk1 := openAIResponse{
			Choices: []openAIChoice{
				{Delta: openAIMessage{Content: "The "}},
			},
		}
		data, _ := json.Marshal(chunk1)
		_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()

		// Chunk 2
		chunk2 := openAIResponse{
			Choices: []openAIChoice{
				{Delta: openAIMessage{Content: "answer "}},
			},
		}
		data, _ = json.Marshal(chunk2)
		_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()

		// Chunk 3
		chunk3 := openAIResponse{
			Choices: []openAIChoice{
				{Delta: openAIMessage{Content: "is 4."}},
			},
		}
		data, _ = json.Marshal(chunk3)
		_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()

		// Done marker
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider(&ProviderConfig{
		Name: "openai",
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

func TestOpenAIProvider_GetCapabilities(t *testing.T) {
	provider, _ := NewOpenAIProvider(&ProviderConfig{
		Name: "openai",
		Config: map[string]interface{}{
			"api_key": "test-key",
			"capabilities": map[string]interface{}{
				"streaming":          true,
				"tools":              true,
				"multi_turn":         true,
				"max_context_tokens": 128000,
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
	if caps.MaxContextTokens != 128000 {
		t.Errorf("max context tokens = %d, want 128000", caps.MaxContextTokens)
	}
}

func TestOpenAIProvider_GetInfo(t *testing.T) {
	provider, _ := NewOpenAIProvider(&ProviderConfig{
		Name:    "openai",
		Version: "1.0.0",
		Config: map[string]interface{}{
			"api_key": "test-key",
		},
	})

	info := provider.GetInfo()

	if info.Name != "openai" {
		t.Errorf("name = %s, want 'openai'", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("version = %s, want '1.0.0'", info.Version)
	}
	if info.Type != ProviderTypeAPI {
		t.Errorf("type = %s, want %s", info.Type, ProviderTypeAPI)
	}
	if info.Author != "specular" {
		t.Errorf("author = %s, want 'specular'", info.Author)
	}
}

func TestOpenAIProvider_IsAvailable(t *testing.T) {
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
			provider := &OpenAIProvider{
				apiKey: tt.apiKey,
			}

			available := provider.IsAvailable()
			if available != tt.available {
				t.Errorf("IsAvailable() = %v, want %v", available, tt.available)
			}
		})
	}
}

func TestOpenAIProvider_Health(t *testing.T) {
	// Mock successful health check
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]string{
					{"id": "gpt-4o-mini"},
				},
			})
		}
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider(&ProviderConfig{
		Name: "openai",
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

func TestOpenAIProvider_Health_Error(t *testing.T) {
	// Mock failed health check
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider(&ProviderConfig{
		Name: "openai",
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
}

func TestOpenAIProvider_Close(t *testing.T) {
	provider, _ := NewOpenAIProvider(&ProviderConfig{
		Name: "openai",
		Config: map[string]interface{}{
			"api_key": "test-key",
		},
	})

	err := provider.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}
