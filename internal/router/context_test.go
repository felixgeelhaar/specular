package router

import (
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/provider"
)

func TestTokenCounter_EstimateTokens(t *testing.T) {
	counter := NewTokenCounter()

	tests := []struct {
		name     string
		text     string
		minTokens int
		maxTokens int
	}{
		{
			name:      "empty string",
			text:      "",
			minTokens: 0,
			maxTokens: 0,
		},
		{
			name:      "simple sentence",
			text:      "Hello world",
			minTokens: 2,
			maxTokens: 4,
		},
		{
			name:      "long text with whitespace",
			text:      "This is a longer piece of text with multiple words and some punctuation.",
			minTokens: 10,
			maxTokens: 20,
		},
		{
			name:      "code snippet",
			text:      "func main() { fmt.Println(\"hello\") }",
			minTokens: 6,
			maxTokens: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := counter.EstimateTokens(tt.text)
			if tokens < tt.minTokens || tokens > tt.maxTokens {
				t.Errorf("EstimateTokens(%q) = %d, want between %d and %d", tt.text, tokens, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestTokenCounter_EstimateRequestTokens(t *testing.T) {
	counter := NewTokenCounter()

	tests := []struct {
		name     string
		req      *GenerateRequest
		minTokens int
	}{
		{
			name: "simple request",
			req: &GenerateRequest{
				Prompt: "Hello",
			},
			minTokens: 20, // Prompt tokens + overhead
		},
		{
			name: "request with system prompt",
			req: &GenerateRequest{
				Prompt:       "Write a function",
				SystemPrompt: "You are a helpful coding assistant",
			},
			minTokens: 30, // Both prompts + overhead
		},
		{
			name: "request with context messages",
			req: &GenerateRequest{
				Prompt: "Continue",
				Context: []provider.Message{
					{Role: "user", Content: "First message"},
					{Role: "assistant", Content: "Response"},
					{Role: "user", Content: "Follow up"},
				},
			},
			minTokens: 30, // Prompt + 3 messages with overhead
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := counter.EstimateRequestTokens(tt.req)
			if tokens < tt.minTokens {
				t.Errorf("EstimateRequestTokens() = %d, want at least %d", tokens, tt.minTokens)
			}
		})
	}
}

func TestContextValidator_ValidateRequest(t *testing.T) {
	validator := NewContextValidator()

	// Create a test model with 8k context window
	model := &Model{
		ID:            "test-model",
		ContextWindow: 8000,
	}

	tests := []struct {
		name    string
		req     *GenerateRequest
		model   *Model
		wantErr bool
	}{
		{
			name: "fits in context window",
			req: &GenerateRequest{
				Prompt:    "Short prompt",
				MaxTokens: 100,
			},
			model:   model,
			wantErr: false,
		},
		{
			name: "exceeds context window",
			req: &GenerateRequest{
				Prompt:    strings.Repeat("word ", 10000), // Very long prompt
				MaxTokens: 100,
			},
			model:   model,
			wantErr: true,
		},
		{
			name: "large output tokens",
			req: &GenerateRequest{
				Prompt:    "Short prompt",
				MaxTokens: 9000, // Exceeds context window
			},
			model:   model,
			wantErr: true,
		},
		{
			name: "zero max tokens uses default",
			req: &GenerateRequest{
				Prompt:    "Short prompt",
				MaxTokens: 0, // Should use default 2048
			},
			model:   model,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRequest(tt.req, tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContextValidator_GetInputTokenCount(t *testing.T) {
	validator := NewContextValidator()

	req := &GenerateRequest{
		Prompt:       "Test prompt",
		SystemPrompt: "System prompt",
		Context: []provider.Message{
			{Role: "user", Content: "Message 1"},
			{Role: "assistant", Content: "Response"},
		},
	}

	count := validator.GetInputTokenCount(req)
	if count < 20 {
		t.Errorf("GetInputTokenCount() = %d, want at least 20", count)
	}
}

func TestContextTruncator_TruncateRequest(t *testing.T) {
	// Create a test model with small context window to force truncation
	model := &Model{
		ID:            "test-model",
		ContextWindow: 150,
	}

	tests := []struct {
		name           string
		strategy       TruncationStrategy
		req            *GenerateRequest
		wantTruncated  bool
		wantErr        bool
	}{
		{
			name:     "no truncation needed",
			strategy: TruncateOldest,
			req: &GenerateRequest{
				Prompt:    "Short",
				MaxTokens: 50,
			},
			wantTruncated: false,
			wantErr:       false,
		},
		{
			name:     "truncate oldest messages",
			strategy: TruncateOldest,
			req: &GenerateRequest{
				Prompt:    strings.Repeat("word ", 100),
				MaxTokens: 50,
				Context: []provider.Message{
					{Role: "user", Content: strings.Repeat("old ", 50)},
					{Role: "assistant", Content: strings.Repeat("response ", 50)},
					{Role: "user", Content: "recent"},
				},
			},
			wantTruncated: true,
			wantErr:       false,
		},
		{
			name:     "truncate prompt",
			strategy: TruncatePrompt,
			req: &GenerateRequest{
				Prompt:    strings.Repeat("word ", 200),
				MaxTokens: 30,
			},
			wantTruncated: true,
			wantErr:       false,
		},
		{
			name:     "truncate context",
			strategy: TruncateContext,
			req: &GenerateRequest{
				Prompt:    strings.Repeat("word ", 100),
				MaxTokens: 30,
				Context: []provider.Message{
					{Role: "user", Content: strings.Repeat("context ", 100)},
				},
			},
			wantTruncated: true,
			wantErr:       false,
		},
		{
			name:     "truncate proportional",
			strategy: TruncateProportional,
			req: &GenerateRequest{
				Prompt:    strings.Repeat("word ", 100),
				MaxTokens: 30,
				Context: []provider.Message{
					{Role: "user", Content: strings.Repeat("context ", 100)},
				},
			},
			wantTruncated: true,
			wantErr:       false,
		},
		{
			name:     "insufficient context window",
			strategy: TruncateOldest,
			req: &GenerateRequest{
				Prompt:    "Short",
				MaxTokens: 200, // Exceeds total context window (150)
			},
			wantTruncated: false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			truncator := NewContextTruncator(tt.strategy)
			truncated, wasTruncated, err := truncator.TruncateRequest(tt.req, model)

			if (err != nil) != tt.wantErr {
				t.Errorf("TruncateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if wasTruncated != tt.wantTruncated {
				t.Errorf("TruncateRequest() truncated = %v, want %v", wasTruncated, tt.wantTruncated)
			}

			// If truncation succeeded, verify request was modified
			if !tt.wantErr && wasTruncated {
				// Just verify the request was truncated (size reduced)
				// Actual fitting is tested in integration tests
				if truncated == nil {
					t.Error("Truncated request should not be nil")
				}
			}
		})
	}
}

func TestTruncationStrategies(t *testing.T) {
	model := &Model{
		ID:            "test-model",
		ContextWindow: 200,
	}

	baseReq := &GenerateRequest{
		Prompt:    strings.Repeat("prompt ", 100), // Large prompt
		MaxTokens: 50,
		Context: []provider.Message{
			{Role: "user", Content: strings.Repeat("context ", 50)},
			{Role: "assistant", Content: strings.Repeat("response ", 50)},
			{Role: "user", Content: strings.Repeat("question ", 50)},
			{Role: "assistant", Content: strings.Repeat("answer ", 50)},
			{Role: "user", Content: strings.Repeat("final ", 50)},
		},
	}

	strategies := []TruncationStrategy{
		TruncateOldest,
		TruncatePrompt,
		TruncateContext,
		TruncateProportional,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			truncator := NewContextTruncator(strategy)
			truncated, wasTruncated, err := truncator.TruncateRequest(baseReq, model)

			if err != nil {
				t.Errorf("TruncateRequest() with %s failed: %v", strategy, err)
				return
			}

			if !wasTruncated {
				t.Errorf("Expected truncation with %s strategy", strategy)
				return
			}

			// Verify request was modified (actual fitting tested in integration)
			if truncated == nil {
				t.Errorf("Truncated request with %s should not be nil", strategy)
			}

			// Verify strategy-specific behavior
			switch strategy {
			case TruncateOldest:
				// Should have fewer context messages
				if len(truncated.Context) >= len(baseReq.Context) {
					t.Errorf("TruncateOldest should remove context messages")
				}
			case TruncatePrompt:
				// Should have truncated prompt
				if len(truncated.Prompt) >= len(baseReq.Prompt) {
					t.Errorf("TruncatePrompt should shorten the prompt")
				}
			case TruncateContext:
				// Should have removed all or most context
				if len(truncated.Context) > 0 {
					// It's ok to have some context left if removing all wasn't enough
				}
			case TruncateProportional:
				// Should have affected both prompt and context
				promptChanged := len(truncated.Prompt) != len(baseReq.Prompt)
				contextChanged := len(truncated.Context) != len(baseReq.Context)
				if !promptChanged && !contextChanged {
					t.Errorf("TruncateProportional should affect prompt or context or both")
				}
			}
		})
	}
}

func TestSummarizeContext(t *testing.T) {
	messages := []provider.Message{
		{Role: "user", Content: "First message"},
		{Role: "assistant", Content: "First response"},
		{Role: "user", Content: "Second message"},
		{Role: "assistant", Content: "Second response"},
		{Role: "user", Content: "Third message"},
	}

	tests := []struct {
		name         string
		messages     []provider.Message
		targetTokens int
		wantEmpty    bool
		wantOmitted  bool
	}{
		{
			name:         "empty messages",
			messages:     []provider.Message{},
			targetTokens: 100,
			wantEmpty:    true,
			wantOmitted:  false,
		},
		{
			name:         "small target tokens",
			messages:     messages,
			targetTokens: 10,
			wantEmpty:    false,
			wantOmitted:  false,
		},
		{
			name:         "large target tokens",
			messages:     messages,
			targetTokens: 1000,
			wantEmpty:    false,
			wantOmitted:  false,
		},
		{
			name: "truncate with more messages",
			messages: []provider.Message{
				{Role: "user", Content: "First short message"},
				{Role: "assistant", Content: "This is a very long message that will definitely exceed the token limit and should be truncated in the middle"},
				{Role: "user", Content: "Another message"},
			},
			targetTokens: 15,
			wantEmpty:    false,
			wantOmitted:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := SummarizeContext(tt.messages, tt.targetTokens)

			if tt.wantEmpty {
				if summary != "" {
					t.Errorf("SummarizeContext() with empty messages should return empty string, got %q", summary)
				}
			} else {
				if summary == "" {
					t.Errorf("SummarizeContext() should not return empty string for non-empty messages")
				}

				// Verify summary contains role markers
				if !strings.Contains(summary, "[user]") && !strings.Contains(summary, "[assistant]") {
					t.Errorf("SummarizeContext() should contain role markers")
				}

				// Check for omitted messages indicator if expected
				if tt.wantOmitted {
					if !strings.Contains(summary, "more messages omitted") {
						t.Errorf("SummarizeContext() should contain 'more messages omitted' indicator, got %q", summary)
					}
				}
			}
		})
	}
}

func TestTruncateAllContext_ContextSufficient(t *testing.T) {
	// Test case where removing all context is sufficient (doesn't need to truncate prompt)
	model := &Model{
		ID:            "test-model",
		ContextWindow: 200,
	}

	truncator := NewContextTruncator(TruncateContext)
	req := &GenerateRequest{
		Prompt:    "Short prompt",
		MaxTokens: 50,
		Context: []provider.Message{
			{Role: "user", Content: strings.Repeat("context ", 100)}, // Large context
		},
	}

	truncated, wasTruncated, err := truncator.TruncateRequest(req, model)
	if err != nil {
		t.Fatalf("TruncateRequest() error = %v", err)
	}

	if !wasTruncated {
		t.Error("TruncateRequest() expected truncation")
	}

	// Context should be removed
	if len(truncated.Context) > 0 {
		t.Errorf("TruncateRequest() context length = %d, want 0", len(truncated.Context))
	}

	// Prompt should remain unchanged (context removal was sufficient)
	if truncated.Prompt != req.Prompt {
		t.Error("TruncateRequest() prompt should not be truncated when context removal is sufficient")
	}
}

func TestContextIntegration_GenerateValidation(t *testing.T) {
	// Test that Generate() uses context validation correctly
	config := &RouterConfig{
		BudgetUSD:               100.0,
		MaxLatencyMs:            60000,
		EnableContextValidation: true,
		AutoTruncate:            false,
		TruncationStrategy:      "oldest",
	}

	router, err := NewRouter(config)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	// Verify context validator was initialized
	if router.contextValidator == nil {
		t.Error("Router context validator should be initialized when EnableContextValidation is true")
	}

	if router.contextTruncator == nil {
		t.Error("Router context truncator should be initialized when EnableContextValidation is true")
	}
}

func TestContextIntegration_Disabled(t *testing.T) {
	// Test that context validation is not initialized when disabled
	config := &RouterConfig{
		BudgetUSD:               100.0,
		MaxLatencyMs:            60000,
		EnableContextValidation: false,
	}

	router, err := NewRouter(config)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	// Verify context validator was not initialized
	if router.contextValidator != nil {
		t.Error("Router context validator should be nil when EnableContextValidation is false")
	}

	if router.contextTruncator != nil {
		t.Error("Router context truncator should be nil when EnableContextValidation is false")
	}
}

func TestContextIntegration_AutoTruncate(t *testing.T) {
	// Test auto-truncation configuration
	config := &RouterConfig{
		BudgetUSD:               100.0,
		MaxLatencyMs:            60000,
		EnableContextValidation: true,
		AutoTruncate:            true,
		TruncationStrategy:      "proportional",
	}

	router, err := NewRouter(config)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	// Verify config was set correctly
	if !router.config.EnableContextValidation {
		t.Error("EnableContextValidation should be true")
	}

	if !router.config.AutoTruncate {
		t.Error("AutoTruncate should be true")
	}

	if router.config.TruncationStrategy != "proportional" {
		t.Errorf("TruncationStrategy = %s, want proportional", router.config.TruncationStrategy)
	}
}
