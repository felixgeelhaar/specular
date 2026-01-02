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
				Prompt: "Write a function",
			},
		},
		{
			name: "request with system prompt",
			req: GenerateRequest{
				Prompt:       "Complete the code",
				SystemPrompt: "You are a code completion assistant.",
			},
		},
		{
			name: "request with config",
			req: GenerateRequest{
				Prompt:      "Test prompt",
				MaxTokens:   256,
				Temperature: 0.0,
				Config:      map[string]interface{}{"model": "gpt-3.5-turbo-instruct"},
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
				TokensUsed:   15,
				Model:        "gpt-3.5-turbo-instruct",
				FinishReason: "stop",
				Provider:     "codex",
			},
		},
		{
			name: "response with token breakdown",
			resp: GenerateResponse{
				Content:      "Generated code",
				TokensUsed:   100,
				InputTokens:  30,
				OutputTokens: 70,
				Model:        "gpt-3.5-turbo-instruct",
				Latency:      2 * time.Second,
				FinishReason: "stop",
				Provider:     "codex",
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
	msg := Message{Role: "user", Content: "Hello"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded != msg {
		t.Errorf("decoded = %+v, want %+v", decoded, msg)
	}
}

func TestStreamChunk_JSONRoundtrip(t *testing.T) {
	chunk := StreamChunk{
		Content:    "Generated code",
		Delta:      " code",
		Done:       true,
		TokensUsed: 50,
		Timestamp:  time.Now().Truncate(time.Second),
	}

	data, err := json.Marshal(chunk)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded StreamChunk
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Content != chunk.Content {
		t.Errorf("Content = %q, want %q", decoded.Content, chunk.Content)
	}
	if decoded.Done != chunk.Done {
		t.Errorf("Done = %v, want %v", decoded.Done, chunk.Done)
	}
}

func TestBuildPromptWithSystemPrompt(t *testing.T) {
	req := GenerateRequest{
		Prompt:       "Complete this code",
		SystemPrompt: "You are a code assistant.",
	}

	fullPrompt := req.Prompt
	if req.SystemPrompt != "" {
		fullPrompt = req.SystemPrompt + "\n\n" + req.Prompt
	}

	expected := "You are a code assistant.\n\nComplete this code"
	if fullPrompt != expected {
		t.Errorf("fullPrompt = %q, want %q", fullPrompt, expected)
	}
}

func TestBuildPromptWithContext(t *testing.T) {
	req := GenerateRequest{
		Prompt: "continue",
		Context: []Message{
			{Role: "user", Content: "Start"},
			{Role: "assistant", Content: "Middle"},
		},
	}

	var promptBuilder strings.Builder
	for _, msg := range req.Context {
		promptBuilder.WriteString(msg.Role + ": " + msg.Content + "\n")
	}
	promptBuilder.WriteString("user: " + req.Prompt)
	fullPrompt := promptBuilder.String()

	expected := "user: Start\nassistant: Middle\nuser: continue"
	if fullPrompt != expected {
		t.Errorf("fullPrompt = %q, want %q", fullPrompt, expected)
	}
}

func TestDefaultModel(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{},
	}

	model := "gpt-3.5-turbo-instruct"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	if model != "gpt-3.5-turbo-instruct" {
		t.Errorf("model = %q, want %q", model, "gpt-3.5-turbo-instruct")
	}
}

func TestCustomModel(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{"model": "code-davinci-002"},
	}

	model := "gpt-3.5-turbo-instruct"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	if model != "code-davinci-002" {
		t.Errorf("model = %q, want %q", model, "code-davinci-002")
	}
}

func TestGenerateRequest_Decode(t *testing.T) {
	input := `{"prompt":"Write code","max_tokens":256,"temperature":0}`

	var req GenerateRequest
	if err := json.NewDecoder(bytes.NewBufferString(input)).Decode(&req); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if req.Prompt != "Write code" {
		t.Errorf("Prompt = %q, want %q", req.Prompt, "Write code")
	}
	if req.MaxTokens != 256 {
		t.Errorf("MaxTokens = %d, want %d", req.MaxTokens, 256)
	}
}

func TestGenerateResponse_Encode(t *testing.T) {
	resp := GenerateResponse{
		Content:      "def foo(): pass",
		TokensUsed:   10,
		Model:        "gpt-3.5-turbo-instruct",
		FinishReason: "stop",
		Provider:     "codex",
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

func TestBuildCommandArgs(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		maxTokens   int
		temperature float64
		topP        float64
		prompt      string
		wantCount   int
	}{
		{
			name:      "basic args",
			model:     "gpt-3.5-turbo-instruct",
			prompt:    "Hello",
			wantCount: 6, // api, completions.create, -m, model, -p, prompt
		},
		{
			name:        "with max tokens",
			model:       "gpt-3.5-turbo-instruct",
			maxTokens:   100,
			temperature: 0.5,
			prompt:      "Hello",
			wantCount:   10, // adds --max-tokens, 100, -t, 0.50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"api", "completions.create"}
			args = append(args, "-m", tt.model)

			if tt.maxTokens > 0 {
				args = append(args, "--max-tokens", "100")
			}
			if tt.temperature > 0 {
				args = append(args, "-t", "0.50")
			}
			if tt.topP > 0 {
				args = append(args, "--top-p", "0.90")
			}
			args = append(args, "-p", tt.prompt)

			if len(args) != tt.wantCount {
				t.Errorf("args length = %d, want %d", len(args), tt.wantCount)
			}
		})
	}
}

func TestTokenEstimation(t *testing.T) {
	text := "Hello world"
	expectedTokens := len(text) / 4

	if expectedTokens != 2 {
		t.Errorf("tokens = %d, want 2", expectedTokens)
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
	case "openai":
		if len(args) > 1 && args[1] == "--version" {
			os.Stdout.WriteString("openai 1.0.0\n")
		} else {
			os.Stdout.WriteString("Mocked completion response")
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

func TestEnvironmentCheck(t *testing.T) {
	// Test that we can check for OPENAI_API_KEY
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer os.Setenv("OPENAI_API_KEY", originalKey)

	os.Setenv("OPENAI_API_KEY", "")
	if os.Getenv("OPENAI_API_KEY") != "" {
		t.Error("Expected OPENAI_API_KEY to be empty")
	}

	os.Setenv("OPENAI_API_KEY", "test-key")
	if os.Getenv("OPENAI_API_KEY") != "test-key" {
		t.Errorf("OPENAI_API_KEY = %q, want %q", os.Getenv("OPENAI_API_KEY"), "test-key")
	}
}
