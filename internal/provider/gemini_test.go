package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewGeminiProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ProviderConfig{
				Name:    "gemini",
				Type:    ProviderTypeAPI,
				Enabled: true,
				Config: map[string]interface{}{
					"api_key":  "test-key",
					"base_url": "https://generativelanguage.googleapis.com/v1beta",
				},
			},
			wantErr: false,
		},
		{
			name: "missing api_key",
			config: &ProviderConfig{
				Name:    "gemini",
				Type:    ProviderTypeAPI,
				Enabled: true,
				Config:  map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "defaults to gemini base_url",
			config: &ProviderConfig{
				Name:    "gemini",
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
			provider, err := NewGeminiProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGeminiProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewGeminiProvider() returned nil provider without error")
			}
		})
	}
}

func TestGeminiProvider_Generate(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL contains model and API key
		if !strings.Contains(r.URL.Path, "/models/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.Path, ":generateContent") {
			t.Errorf("path missing :generateContent: %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "key=test-key") {
			t.Errorf("missing API key in query: %s", r.URL.RawQuery)
		}

		// Parse request
		var req geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		// Verify request fields
		if len(req.Contents) == 0 {
			t.Error("no contents in request")
		}
		if req.SystemInstruction != nil && len(req.SystemInstruction.Parts) == 0 {
			t.Error("system instruction has no parts")
		}

		// Send mock response
		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role: "model",
						Parts: []geminiPart{
							{Text: "Hello! This is a test response from Gemini."},
						},
					},
					FinishReason: "STOP",
					Index:        0,
				},
			},
			UsageMetadata: &geminiUsage{
				PromptTokenCount:     10,
				CandidatesTokenCount: 20,
				TotalTokenCount:      30,
			},
			ModelVersion: "gemini-2.0-flash-exp",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock server
	provider, err := NewGeminiProvider(&ProviderConfig{
		Name:    "gemini",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Config: map[string]interface{}{
			"api_key":  "test-key",
			"base_url": server.URL,
		},
	})
	if err != nil {
		t.Fatalf("NewGeminiProvider() failed: %v", err)
	}

	// Test generate
	ctx := context.Background()
	resp, err := provider.Generate(ctx, &GenerateRequest{
		Prompt:       "Say hello",
		SystemPrompt: "You are a helpful assistant",
		Temperature:  0.7,
		MaxTokens:    100,
	})

	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify response
	if resp.Content == "" {
		t.Error("response content is empty")
	}
	if !strings.Contains(resp.Content, "Hello") {
		t.Errorf("unexpected content: %s", resp.Content)
	}
	if resp.TokensUsed != 30 {
		t.Errorf("unexpected token count: %d", resp.TokensUsed)
	}
	if resp.InputTokens != 10 {
		t.Errorf("unexpected input tokens: %d", resp.InputTokens)
	}
	if resp.OutputTokens != 20 {
		t.Errorf("unexpected output tokens: %d", resp.OutputTokens)
	}
	if resp.Provider != "gemini" {
		t.Errorf("unexpected provider: %s", resp.Provider)
	}
}

func TestGeminiProvider_Generate_WithContext(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		// Verify multi-turn conversation
		if len(req.Contents) < 3 { // context (2 messages) + prompt
			t.Errorf("expected at least 3 contents, got %d", len(req.Contents))
		}

		// Verify roles are converted correctly
		foundUserMsg := false
		foundModelMsg := false
		for _, content := range req.Contents {
			if content.Role == "user" {
				foundUserMsg = true
			}
			if content.Role == "model" {
				foundModelMsg = true
			}
		}
		if !foundUserMsg {
			t.Error("no user message found in contents")
		}
		if !foundModelMsg {
			t.Error("no model message found (assistant should be converted to model)")
		}

		// Send response
		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Parts: []geminiPart{{Text: "Context received"}},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: &geminiUsage{TotalTokenCount: 30},
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewGeminiProvider(&ProviderConfig{
		Name:    "gemini",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Config: map[string]interface{}{
			"api_key":  "test-key",
			"base_url": server.URL,
		},
	})

	// Test with context
	ctx := context.Background()
	_, err := provider.Generate(ctx, &GenerateRequest{
		Prompt: "Continue the conversation",
		Context: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
	})

	if err != nil {
		t.Fatalf("Generate() with context failed: %v", err)
	}
}

func TestGeminiProvider_Generate_ModelOverride(t *testing.T) {
	// Test that model can be overridden via request config
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the custom model name is in the URL
		if !strings.Contains(r.URL.Path, "custom-model") {
			t.Errorf("Expected custom-model in URL path, got: %s", r.URL.Path)
		}

		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Parts: []geminiPart{
							{Text: "Response"},
						},
						Role: "model",
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: &geminiUsage{
				PromptTokenCount:     5,
				CandidatesTokenCount: 5,
				TotalTokenCount:      10,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewGeminiProvider(&ProviderConfig{
		Name:    "gemini",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Config: map[string]interface{}{
			"api_key":  "test-key",
			"model":    "default-model",
			"base_url": server.URL,
		},
	})

	ctx := context.Background()
	resp, err := provider.Generate(ctx, &GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{
			"model": "custom-model",
		},
	})

	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if resp == nil {
		t.Fatal("Generate() returned nil response")
	}
}

func TestGeminiProvider_Stream(t *testing.T) {
	// Create mock SSE server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming endpoint
		if !strings.Contains(r.URL.Path, ":streamGenerateContent") {
			t.Errorf("path missing :streamGenerateContent: %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "alt=sse") {
			t.Errorf("missing alt=sse in query: %s", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter doesn't support flushing")
		}

		// Send streaming chunks
		chunks := []string{
			`{"candidates":[{"content":{"parts":[{"text":"Hello"}]}}]}`,
			`{"candidates":[{"content":{"parts":[{"text":" world"}]}}]}`,
			`{"candidates":[{"content":{"parts":[{"text":"!"}]},"finishReason":"STOP"}],"usageMetadata":{"totalTokenCount":15}}`,
		}

		for _, chunk := range chunks {
			w.Write([]byte("data: " + chunk + "\n\n"))
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer server.Close()

	provider, _ := NewGeminiProvider(&ProviderConfig{
		Name:    "gemini",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Config: map[string]interface{}{
			"api_key":  "test-key",
			"base_url": server.URL,
		},
	})

	// Test streaming
	ctx := context.Background()
	chunkChan, err := provider.Stream(ctx, &GenerateRequest{
		Prompt: "Say hello world",
	})

	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}

	// Collect chunks
	var chunks []StreamChunk
	for chunk := range chunkChan {
		chunks = append(chunks, chunk)
		if chunk.Error != nil {
			t.Errorf("chunk error: %v", chunk.Error)
		}
		if chunk.Done {
			break
		}
	}

	// Verify chunks
	if len(chunks) == 0 {
		t.Fatal("no chunks received")
	}

	lastChunk := chunks[len(chunks)-1]
	if !lastChunk.Done {
		t.Error("last chunk should be marked as done")
	}
	if !strings.Contains(lastChunk.Content, "Hello") {
		t.Errorf("unexpected content: %s", lastChunk.Content)
	}
	if lastChunk.TokensUsed != 15 {
		t.Errorf("unexpected token count: %d", lastChunk.TokensUsed)
	}
}

func TestGeminiProvider_GetCapabilities(t *testing.T) {
	tests := []struct {
		name               string
		config             *ProviderConfig
		wantMaxTokens      int
		wantCostPer1KToken float64
	}{
		{
			name: "default capabilities",
			config: &ProviderConfig{
				Name:    "gemini",
				Type:    ProviderTypeAPI,
				Enabled: true,
				Config: map[string]interface{}{
					"api_key": "test-key",
				},
			},
			wantMaxTokens:      1000000,
			wantCostPer1KToken: 0.0,
		},
		{
			name: "with config overrides",
			config: &ProviderConfig{
				Name:    "gemini",
				Type:    ProviderTypeAPI,
				Enabled: true,
				Config: map[string]interface{}{
					"api_key": "test-key",
					"capabilities": map[string]interface{}{
						"max_context_tokens":  float64(128000),
						"cost_per_1k_tokens": float64(0.01),
					},
				},
			},
			wantMaxTokens:      128000,
			wantCostPer1KToken: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGeminiProvider(tt.config)
			if err != nil {
				t.Fatalf("NewGeminiProvider() error = %v", err)
			}

			caps := provider.GetCapabilities()

			if !caps.SupportsStreaming {
				t.Error("Gemini should support streaming")
			}
			if !caps.SupportsTools {
				t.Error("Gemini should support tools")
			}
			if !caps.SupportsMultiTurn {
				t.Error("Gemini should support multi-turn")
			}
			if !caps.SupportsVision {
				t.Error("Gemini should support vision")
			}
			if caps.MaxContextTokens != tt.wantMaxTokens {
				t.Errorf("MaxContextTokens = %d, want %d", caps.MaxContextTokens, tt.wantMaxTokens)
			}
			if caps.CostPer1KTokens != tt.wantCostPer1KToken {
				t.Errorf("CostPer1KTokens = %.2f, want %.2f", caps.CostPer1KTokens, tt.wantCostPer1KToken)
			}
		})
	}
}

func TestGeminiProvider_GetInfo(t *testing.T) {
	provider, _ := NewGeminiProvider(&ProviderConfig{
		Name:    "gemini",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Version: "1.0.0",
		Config: map[string]interface{}{
			"api_key": "test-key",
		},
	})

	info := provider.GetInfo()

	if info.Name != "gemini" {
		t.Errorf("unexpected name: %s", info.Name)
	}
	if info.Type != ProviderTypeAPI {
		t.Errorf("unexpected type: %s", info.Type)
	}
	if info.Version != "1.0.0" {
		t.Errorf("unexpected version: %s", info.Version)
	}
}

func TestGeminiProvider_IsAvailable(t *testing.T) {
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
			provider := &GeminiProvider{apiKey: tt.apiKey}
			if provider.IsAvailable() != tt.available {
				t.Errorf("IsAvailable() = %v, want %v", provider.IsAvailable(), tt.available)
			}
		})
	}
}

func TestGeminiProvider_Generate_APIError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{
			Error: &geminiError{
				Code:    400,
				Message: "Invalid API key",
				Status:  "INVALID_ARGUMENT",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewGeminiProvider(&ProviderConfig{
		Name:    "gemini",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Config: map[string]interface{}{
			"api_key":  "invalid-key",
			"base_url": server.URL,
		},
	})

	ctx := context.Background()
	_, err := provider.Generate(ctx, &GenerateRequest{
		Prompt: "Test",
	})

	if err == nil {
		t.Error("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGeminiProvider_Health(t *testing.T) {
	// Mock successful health check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's using query parameter auth
		if !strings.Contains(r.URL.RawQuery, "key=test-key") {
			t.Error("expected query parameter with api key")
		}

		// Return successful response
		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Parts: []geminiPart{
							{Text: "Hi"},
						},
						Role: "model",
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: &geminiUsage{
				PromptTokenCount:     1,
				CandidatesTokenCount: 1,
				TotalTokenCount:      2,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewGeminiProvider(&ProviderConfig{
		Name:    "gemini",
		Type:    ProviderTypeAPI,
		Enabled: true,
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

func TestGeminiProvider_Health_Error(t *testing.T) {
	// Mock failed health check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{
			Error: &geminiError{
				Code:    401,
				Message: "API key not valid",
				Status:  "UNAUTHENTICATED",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewGeminiProvider(&ProviderConfig{
		Name:    "gemini",
		Type:    ProviderTypeAPI,
		Enabled: true,
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
	if !strings.Contains(err.Error(), "API key not valid") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGeminiProvider_Close(t *testing.T) {
	provider, err := NewGeminiProvider(&ProviderConfig{
		Name:    "gemini",
		Type:    ProviderTypeAPI,
		Enabled: true,
		Config: map[string]interface{}{
			"api_key": "test-key",
		},
	})
	if err != nil {
		t.Fatalf("NewGeminiProvider() error = %v", err)
	}

	// Close should not return an error (HTTP client doesn't need cleanup)
	err = provider.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}
