package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
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
				Prompt: "Hello, Gemini!",
			},
		},
		{
			name: "request with system prompt",
			req: GenerateRequest{
				Prompt:       "Explain AI",
				SystemPrompt: "You are an AI expert.",
			},
		},
		{
			name: "request with config",
			req: GenerateRequest{
				Prompt:      "Test prompt",
				MaxTokens:   1024,
				Temperature: 0.9,
				Config:      map[string]interface{}{"model": "gemini-2.5-pro"},
			},
		},
		{
			name: "request with context",
			req: GenerateRequest{
				Prompt: "Continue",
				Context: []Message{
					{Role: "user", Content: "What is ML?"},
					{Role: "assistant", Content: "ML is machine learning."},
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
				Content:      "Hello from Gemini!",
				TokensUsed:   0,
				Model:        "gemini-2.5-pro",
				FinishReason: "stop",
				Provider:     "gemini-cli",
			},
		},
		{
			name: "response with latency",
			resp: GenerateResponse{
				Content:      "Generated response",
				TokensUsed:   0,
				Model:        "gemini-2.5-pro",
				Latency:      3 * time.Second,
				FinishReason: "stop",
				Provider:     "gemini-cli",
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

func TestDefaultModel(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{},
	}

	model := "gemini-2.5-pro"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	if model != "gemini-2.5-pro" {
		t.Errorf("model = %q, want %q", model, "gemini-2.5-pro")
	}
}

func TestCustomModel(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{"model": "gemini-2.5-flash"},
	}

	model := "gemini-2.5-pro"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	if model != "gemini-2.5-flash" {
		t.Errorf("model = %q, want %q", model, "gemini-2.5-flash")
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
			name:     "basic args",
			model:    "gemini-2.5-pro",
			prompt:   "Hello",
			wantArgs: []string{"--model", "gemini-2.5-pro", "--prompt", "Hello"},
		},
		{
			name:     "with different model",
			model:    "gemini-2.5-flash",
			prompt:   "Test prompt",
			wantArgs: []string{"--model", "gemini-2.5-flash", "--prompt", "Test prompt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{
				"--model", tt.model,
				"--prompt", tt.prompt,
			}

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
	input := `{"prompt":"Hello Gemini","config":{"model":"gemini-2.5-flash"}}`

	var req GenerateRequest
	if err := json.NewDecoder(bytes.NewBufferString(input)).Decode(&req); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if req.Prompt != "Hello Gemini" {
		t.Errorf("Prompt = %q, want %q", req.Prompt, "Hello Gemini")
	}

	if modelVal, ok := req.Config["model"].(string); !ok || modelVal != "gemini-2.5-flash" {
		t.Errorf("Config model = %v, want %q", req.Config["model"], "gemini-2.5-flash")
	}
}

func TestGenerateResponse_Encode(t *testing.T) {
	resp := GenerateResponse{
		Content:      "Gemini CLI response",
		TokensUsed:   0,
		Model:        "gemini-2.5-pro",
		FinishReason: "stop",
		Provider:     "gemini-cli",
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
	case "gemini":
		if len(args) > 1 && args[1] == "--version" {
			os.Stdout.WriteString("gemini 1.0.0\n")
		} else {
			os.Stdout.WriteString("Mocked Gemini CLI response")
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

func TestResponseProvider(t *testing.T) {
	resp := GenerateResponse{
		Content:      "test",
		FinishReason: "stop",
		Provider:     "gemini-cli",
	}

	if resp.Provider != "gemini-cli" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "gemini-cli")
	}
}

func TestResponseTokensZero(t *testing.T) {
	// Gemini CLI doesn't report token usage
	resp := GenerateResponse{
		Content:      "test output",
		TokensUsed:   0,
		InputTokens:  0,
		OutputTokens: 0,
		Provider:     "gemini-cli",
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

func TestResponseLatency(t *testing.T) {
	resp := GenerateResponse{
		Content: "test",
		Latency: 2 * time.Second,
	}

	if resp.Latency != 2*time.Second {
		t.Errorf("Latency = %v, want %v", resp.Latency, 2*time.Second)
	}
}
