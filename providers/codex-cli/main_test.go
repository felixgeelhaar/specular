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
				Prompt: "Write a function",
			},
		},
		{
			name: "request with system prompt",
			req: GenerateRequest{
				Prompt:       "Complete the code",
				SystemPrompt: "You are a code assistant.",
			},
		},
		{
			name: "request with config",
			req: GenerateRequest{
				Prompt:      "Test prompt",
				MaxTokens:   256,
				Temperature: 0.0,
				Config:      map[string]interface{}{"model": "o3-mini"},
			},
		},
		{
			name: "request with context",
			req: GenerateRequest{
				Prompt: "Continue the code",
				Context: []Message{
					{Role: "user", Content: "def hello():"},
					{Role: "assistant", Content: "    print('Hello')"},
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
				Content:      "def hello():\n    print('Hello')",
				TokensUsed:   0,
				Model:        "",
				FinishReason: "stop",
				Provider:     "codex-cli",
			},
		},
		{
			name: "response with model",
			resp: GenerateResponse{
				Content:      "Generated code",
				TokensUsed:   0,
				Model:        "o3-mini",
				Latency:      2 * time.Second,
				FinishReason: "stop",
				Provider:     "codex-cli",
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

func TestModelFromConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    map[string]interface{}
		wantModel string
	}{
		{
			name:      "empty config",
			config:    map[string]interface{}{},
			wantModel: "",
		},
		{
			name:      "nil config",
			config:    nil,
			wantModel: "",
		},
		{
			name:      "with model",
			config:    map[string]interface{}{"model": "o3-mini"},
			wantModel: "o3-mini",
		},
		{
			name:      "empty model string",
			config:    map[string]interface{}{"model": ""},
			wantModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var model string
			if tt.config != nil {
				if modelVal, ok := tt.config["model"].(string); ok && modelVal != "" {
					model = modelVal
				}
			}

			if model != tt.wantModel {
				t.Errorf("model = %q, want %q", model, tt.wantModel)
			}
		})
	}
}

func TestBuildCommandArgs(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		prompt   string
		wantArgs []string
	}{
		{
			name:     "without model",
			model:    "",
			prompt:   "Hello",
			wantArgs: []string{"exec", "Hello"},
		},
		{
			name:     "with model",
			model:    "o3-mini",
			prompt:   "Hello",
			wantArgs: []string{"exec", "--model", "o3-mini", "Hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"exec"}

			if tt.model != "" {
				args = append(args, "--model", tt.model)
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

func TestGenerateRequest_Decode(t *testing.T) {
	input := `{"prompt":"Write code","config":{"model":"o3-mini"}}`

	var req GenerateRequest
	if err := json.NewDecoder(bytes.NewBufferString(input)).Decode(&req); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if req.Prompt != "Write code" {
		t.Errorf("Prompt = %q, want %q", req.Prompt, "Write code")
	}

	if modelVal, ok := req.Config["model"].(string); !ok || modelVal != "o3-mini" {
		t.Errorf("Config model = %v, want %q", req.Config["model"], "o3-mini")
	}
}

func TestGenerateResponse_Encode(t *testing.T) {
	resp := GenerateResponse{
		Content:      "def foo(): pass",
		TokensUsed:   0,
		Model:        "",
		FinishReason: "stop",
		Provider:     "codex-cli",
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
	case "codex":
		if len(args) > 1 && args[1] == "--version" {
			os.Stdout.WriteString("codex 1.0.0\n")
		} else {
			os.Stdout.WriteString("Mocked codex response")
		}
	default:
		os.Stderr.WriteString("Unknown command: " + cmd)
		os.Exit(1)
	}
}

func TestResponseProvider(t *testing.T) {
	resp := GenerateResponse{
		Content:      "test",
		FinishReason: "stop",
		Provider:     "codex-cli",
	}

	if resp.Provider != "codex-cli" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "codex-cli")
	}
}

func TestResponseTokensZero(t *testing.T) {
	// Codex CLI doesn't report token usage
	resp := GenerateResponse{
		Content:      "test output",
		TokensUsed:   0,
		InputTokens:  0,
		OutputTokens: 0,
		Provider:     "codex-cli",
	}

	if resp.TokensUsed != 0 {
		t.Errorf("TokensUsed = %d, want 0", resp.TokensUsed)
	}
	if resp.InputTokens != 0 {
		t.Errorf("InputTokens = %d, want 0", resp.InputTokens)
	}
	if resp.OutputTokens != 0 {
		t.Errorf("OutputTokens = %d, want 0", resp.OutputTokens)
	}
}
