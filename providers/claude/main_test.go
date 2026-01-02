package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestGenerateRequest_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		req  GenerateRequest
	}{
		{
			name: "basic request",
			req: GenerateRequest{
				Prompt: "Hello, world!",
			},
		},
		{
			name: "request with system prompt",
			req: GenerateRequest{
				Prompt:       "What is Go?",
				SystemPrompt: "You are a helpful assistant.",
			},
		},
		{
			name: "request with config",
			req: GenerateRequest{
				Prompt:      "Test prompt",
				MaxTokens:   100,
				Temperature: 0.7,
				TopP:        0.9,
				Config:      map[string]interface{}{"model": "claude-sonnet-4-20250514"},
			},
		},
		{
			name: "request with context",
			req: GenerateRequest{
				Prompt: "Continue the conversation",
				Context: []Message{
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi there!"},
				},
			},
		},
		{
			name: "request with metadata",
			req: GenerateRequest{
				Prompt:   "Test",
				Metadata: map[string]string{"session_id": "abc123"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Unmarshal back
			var decoded GenerateRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Verify key fields
			if decoded.Prompt != tt.req.Prompt {
				t.Errorf("Prompt = %q, want %q", decoded.Prompt, tt.req.Prompt)
			}
			if decoded.SystemPrompt != tt.req.SystemPrompt {
				t.Errorf("SystemPrompt = %q, want %q", decoded.SystemPrompt, tt.req.SystemPrompt)
			}
			if decoded.MaxTokens != tt.req.MaxTokens {
				t.Errorf("MaxTokens = %d, want %d", decoded.MaxTokens, tt.req.MaxTokens)
			}
			if decoded.Temperature != tt.req.Temperature {
				t.Errorf("Temperature = %f, want %f", decoded.Temperature, tt.req.Temperature)
			}
		})
	}
}

func TestGenerateResponse_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		resp GenerateResponse
	}{
		{
			name: "basic response",
			resp: GenerateResponse{
				Content:      "Hello!",
				TokensUsed:   10,
				Model:        "claude-sonnet-4-20250514",
				FinishReason: "stop",
				Provider:     "claude",
			},
		},
		{
			name: "response with token breakdown",
			resp: GenerateResponse{
				Content:      "Generated text",
				TokensUsed:   150,
				InputTokens:  50,
				OutputTokens: 100,
				Model:        "claude-sonnet-4-20250514",
				Latency:      time.Second,
				FinishReason: "stop",
				Provider:     "claude",
			},
		},
		{
			name: "response with error",
			resp: GenerateResponse{
				Content:      "",
				Error:        "API rate limit exceeded",
				Model:        "claude-sonnet-4-20250514",
				FinishReason: "error",
				Provider:     "claude",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.resp)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded GenerateResponse
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if decoded.Content != tt.resp.Content {
				t.Errorf("Content = %q, want %q", decoded.Content, tt.resp.Content)
			}
			if decoded.TokensUsed != tt.resp.TokensUsed {
				t.Errorf("TokensUsed = %d, want %d", decoded.TokensUsed, tt.resp.TokensUsed)
			}
			if decoded.Provider != tt.resp.Provider {
				t.Errorf("Provider = %q, want %q", decoded.Provider, tt.resp.Provider)
			}
		})
	}
}

func TestMessage_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
	}{
		{
			name: "user message",
			msg:  Message{Role: "user", Content: "Hello"},
		},
		{
			name: "assistant message",
			msg:  Message{Role: "assistant", Content: "Hi there!"},
		},
		{
			name: "empty content",
			msg:  Message{Role: "user", Content: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.msg)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded Message
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if decoded.Role != tt.msg.Role {
				t.Errorf("Role = %q, want %q", decoded.Role, tt.msg.Role)
			}
			if decoded.Content != tt.msg.Content {
				t.Errorf("Content = %q, want %q", decoded.Content, tt.msg.Content)
			}
		})
	}
}

func TestStreamChunk_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name  string
		chunk StreamChunk
	}{
		{
			name: "partial chunk",
			chunk: StreamChunk{
				Content:   "Hello",
				Delta:     "Hello",
				Done:      false,
				Timestamp: time.Now().Truncate(time.Second),
			},
		},
		{
			name: "final chunk",
			chunk: StreamChunk{
				Content:    "Hello, world!",
				Delta:      ", world!",
				Done:       true,
				TokensUsed: 10,
				Timestamp:  time.Now().Truncate(time.Second),
			},
		},
		{
			name: "error chunk",
			chunk: StreamChunk{
				Content:   "",
				Done:      true,
				ErrorMsg:  "Connection timeout",
				Timestamp: time.Now().Truncate(time.Second),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.chunk)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded StreamChunk
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if decoded.Content != tt.chunk.Content {
				t.Errorf("Content = %q, want %q", decoded.Content, tt.chunk.Content)
			}
			if decoded.Delta != tt.chunk.Delta {
				t.Errorf("Delta = %q, want %q", decoded.Delta, tt.chunk.Delta)
			}
			if decoded.Done != tt.chunk.Done {
				t.Errorf("Done = %v, want %v", decoded.Done, tt.chunk.Done)
			}
			if decoded.ErrorMsg != tt.chunk.ErrorMsg {
				t.Errorf("ErrorMsg = %q, want %q", decoded.ErrorMsg, tt.chunk.ErrorMsg)
			}
		})
	}
}

func TestBuildPromptWithSystemPrompt(t *testing.T) {
	req := GenerateRequest{
		Prompt:       "What is Go?",
		SystemPrompt: "You are a helpful assistant.",
	}

	fullPrompt := req.Prompt
	if req.SystemPrompt != "" {
		fullPrompt = "System: " + req.SystemPrompt + "\n\nUser: " + req.Prompt
	}

	expected := "System: You are a helpful assistant.\n\nUser: What is Go?"
	if fullPrompt != expected {
		t.Errorf("fullPrompt = %q, want %q", fullPrompt, expected)
	}
}

func TestBuildPromptWithContext(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Continue",
		Context: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi!"},
		},
	}

	var promptBuilder strings.Builder
	for _, msg := range req.Context {
		if msg.Role == "user" {
			promptBuilder.WriteString("User: " + msg.Content + "\n")
		} else if msg.Role == "assistant" {
			promptBuilder.WriteString("Assistant: " + msg.Content + "\n")
		}
	}
	promptBuilder.WriteString("User: " + req.Prompt)
	fullPrompt := promptBuilder.String()

	expected := "User: Hello\nAssistant: Hi!\nUser: Continue"
	if fullPrompt != expected {
		t.Errorf("fullPrompt = %q, want %q", fullPrompt, expected)
	}
}

func TestTokenEstimation(t *testing.T) {
	tests := []struct {
		text           string
		expectedTokens int
	}{
		{"", 0},
		{"test", 1},
		{"Hello world!", 3},
		{"This is a longer piece of text that should have more tokens", 14},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			// Rough approximation: ~4 chars per token
			tokens := len(tt.text) / 4
			if tokens != tt.expectedTokens {
				t.Errorf("tokens = %d, want %d", tokens, tt.expectedTokens)
			}
		})
	}
}

// TestHelperProcess is used by exec tests to mock external commands
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}

	if len(args) == 0 {
		os.Exit(1)
	}

	cmd := args[0]
	switch cmd {
	case "claude":
		if len(args) > 1 && args[1] == "--version" {
			os.Stdout.WriteString("claude 1.0.0\n")
		} else {
			os.Stdout.WriteString("Mocked Claude response")
		}
	default:
		os.Stderr.WriteString("Unknown command: " + cmd)
		os.Exit(1)
	}
}

func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestGenerateRequest_Decode(t *testing.T) {
	input := `{"prompt":"Hello","system_prompt":"Be helpful","max_tokens":100}`

	var req GenerateRequest
	if err := json.NewDecoder(bytes.NewBufferString(input)).Decode(&req); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if req.Prompt != "Hello" {
		t.Errorf("Prompt = %q, want %q", req.Prompt, "Hello")
	}
	if req.SystemPrompt != "Be helpful" {
		t.Errorf("SystemPrompt = %q, want %q", req.SystemPrompt, "Be helpful")
	}
	if req.MaxTokens != 100 {
		t.Errorf("MaxTokens = %d, want %d", req.MaxTokens, 100)
	}
}

func TestGenerateResponse_Encode(t *testing.T) {
	resp := GenerateResponse{
		Content:      "Test response",
		TokensUsed:   25,
		Model:        "claude-sonnet-4-20250514",
		FinishReason: "stop",
		Provider:     "claude",
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(resp); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Verify it's valid JSON
	var decoded GenerateResponse
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if decoded.Content != resp.Content {
		t.Errorf("Content = %q, want %q", decoded.Content, resp.Content)
	}
}

func TestDefaultModel(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{},
	}

	// Get model from config or use default
	model := "claude-sonnet-4-20250514"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	if model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want %q", model, "claude-sonnet-4-20250514")
	}
}

func TestCustomModel(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{"model": "claude-opus-4-20250514"},
	}

	model := "claude-sonnet-4-20250514"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	if model != "claude-opus-4-20250514" {
		t.Errorf("model = %q, want %q", model, "claude-opus-4-20250514")
	}
}
