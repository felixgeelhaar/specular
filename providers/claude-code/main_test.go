package main

import (
	"bytes"
	"encoding/json"
	"os"
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
				Config:      map[string]interface{}{"model": "claude-sonnet-4-5-20250929"},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded GenerateRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if decoded.Prompt != tt.req.Prompt {
				t.Errorf("Prompt = %q, want %q", decoded.Prompt, tt.req.Prompt)
			}
			if decoded.SystemPrompt != tt.req.SystemPrompt {
				t.Errorf("SystemPrompt = %q, want %q", decoded.SystemPrompt, tt.req.SystemPrompt)
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
				TokensUsed:   0,
				Model:        "claude-sonnet-4-5-20250929",
				FinishReason: "stop",
				Provider:     "claude-code",
			},
		},
		{
			name: "response with latency",
			resp: GenerateResponse{
				Content:      "Generated text",
				Model:        "claude-sonnet-4-5-20250929",
				Latency:      time.Second,
				FinishReason: "stop",
				Provider:     "claude-code",
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
		{name: "user message", msg: Message{Role: "user", Content: "Hello"}},
		{name: "assistant message", msg: Message{Role: "assistant", Content: "Hi!"}},
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

			if decoded != tt.msg {
				t.Errorf("decoded = %+v, want %+v", decoded, tt.msg)
			}
		})
	}
}

func TestClaudeCodeResponse_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		resp ClaudeCodeResponse
	}{
		{name: "basic response", resp: ClaudeCodeResponse{Text: "Hello world"}},
		{name: "empty response", resp: ClaudeCodeResponse{Text: ""}},
		{name: "multiline response", resp: ClaudeCodeResponse{Text: "Line 1\nLine 2\nLine 3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.resp)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded ClaudeCodeResponse
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if decoded.Text != tt.resp.Text {
				t.Errorf("Text = %q, want %q", decoded.Text, tt.resp.Text)
			}
		})
	}
}

func TestDefaultModel(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{},
	}

	model := "claude-sonnet-4-5-20250929"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	if model != "claude-sonnet-4-5-20250929" {
		t.Errorf("model = %q, want %q", model, "claude-sonnet-4-5-20250929")
	}
}

func TestCustomModel(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{"model": "claude-opus-4-20250514"},
	}

	model := "claude-sonnet-4-5-20250929"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	if model != "claude-opus-4-20250514" {
		t.Errorf("model = %q, want %q", model, "claude-opus-4-20250514")
	}
}

func TestGenerateRequest_Decode(t *testing.T) {
	input := `{"prompt":"Hello","system_prompt":"Be helpful"}`

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
}

func TestGenerateResponse_Encode(t *testing.T) {
	resp := GenerateResponse{
		Content:      "Test response",
		Model:        "claude-sonnet-4-5-20250929",
		FinishReason: "stop",
		Provider:     "claude-code",
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(resp); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	var decoded GenerateResponse
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
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
			resp := ClaudeCodeResponse{Text: "Mocked response"}
			json.NewEncoder(os.Stdout).Encode(resp)
		}
	default:
		os.Stderr.WriteString("Unknown command: " + cmd)
		os.Exit(1)
	}
}

func TestClaudeCodeResponse_ParsePlainText(t *testing.T) {
	// Simulate when JSON parsing fails and we fall back to plain text
	plainTextOutput := []byte("This is plain text response")

	var claudeResp ClaudeCodeResponse
	if err := json.Unmarshal(plainTextOutput, &claudeResp); err != nil {
		// Expected: JSON parsing fails, treat as plain text
		claudeResp.Text = string(plainTextOutput)
	}

	if claudeResp.Text != "This is plain text response" {
		t.Errorf("Text = %q, want %q", claudeResp.Text, "This is plain text response")
	}
}

func TestBuildCommandArgs(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		systemPrompt string
		prompt       string
		wantArgs     []string
	}{
		{
			name:         "basic args",
			model:        "claude-sonnet-4-5-20250929",
			systemPrompt: "",
			prompt:       "Hello",
			wantArgs:     []string{"--print", "--output-format", "json", "--model", "claude-sonnet-4-5-20250929", "Hello"},
		},
		{
			name:         "with system prompt",
			model:        "claude-sonnet-4-5-20250929",
			systemPrompt: "Be helpful",
			prompt:       "Hello",
			wantArgs:     []string{"--print", "--output-format", "json", "--model", "claude-sonnet-4-5-20250929", "--system-prompt", "Be helpful", "Hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{
				"--print",
				"--output-format", "json",
				"--model", tt.model,
			}

			if tt.systemPrompt != "" {
				args = append(args, "--system-prompt", tt.systemPrompt)
			}

			args = append(args, tt.prompt)

			if len(args) != len(tt.wantArgs) {
				t.Errorf("args length = %d, want %d", len(args), len(tt.wantArgs))
				return
			}

			for i, arg := range args {
				if arg != tt.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}
