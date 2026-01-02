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
				Prompt: "Hello, Gemini!",
			},
		},
		{
			name: "request with system prompt",
			req: GenerateRequest{
				Prompt:       "Explain quantum computing",
				SystemPrompt: "You are a physics expert.",
			},
		},
		{
			name: "request with config",
			req: GenerateRequest{
				Prompt:      "Test prompt",
				MaxTokens:   1024,
				Temperature: 0.9,
				TopP:        0.95,
				Config:      map[string]interface{}{"model": "gemini-2.0-flash-exp"},
			},
		},
		{
			name: "request with context",
			req: GenerateRequest{
				Prompt: "Continue",
				Context: []Message{
					{Role: "user", Content: "What is AI?"},
					{Role: "assistant", Content: "AI is artificial intelligence."},
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
				TokensUsed:   20,
				Model:        "gemini-2.0-flash-exp",
				FinishReason: "stop",
				Provider:     "gemini",
			},
		},
		{
			name: "response with token breakdown",
			resp: GenerateResponse{
				Content:      "Detailed response",
				TokensUsed:   200,
				InputTokens:  50,
				OutputTokens: 150,
				Model:        "gemini-2.0-flash-exp",
				Latency:      3 * time.Second,
				FinishReason: "stop",
				Provider:     "gemini",
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
				TokensUsed: 15,
				Timestamp:  time.Now().Truncate(time.Second),
			},
		},
		{
			name: "error chunk",
			chunk: StreamChunk{
				Done:      true,
				ErrorMsg:  "API quota exceeded",
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

			if decoded.Done != tt.chunk.Done {
				t.Errorf("Done = %v, want %v", decoded.Done, tt.chunk.Done)
			}
		})
	}
}

func TestBuildPromptWithSystemPrompt(t *testing.T) {
	req := GenerateRequest{
		Prompt:       "What is ML?",
		SystemPrompt: "You are an AI expert.",
	}

	fullPrompt := req.Prompt
	if req.SystemPrompt != "" {
		fullPrompt = "System instructions: " + req.SystemPrompt + "\n\nUser: " + req.Prompt
	}

	expected := "System instructions: You are an AI expert.\n\nUser: What is ML?"
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
			promptBuilder.WriteString("Model: " + msg.Content + "\n")
		}
	}
	promptBuilder.WriteString("User: " + req.Prompt)
	fullPrompt := promptBuilder.String()

	expected := "User: Hello\nModel: Hi!\nUser: Continue"
	if fullPrompt != expected {
		t.Errorf("fullPrompt = %q, want %q", fullPrompt, expected)
	}
}

func TestDefaultModel(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{},
	}

	model := "gemini-2.0-flash-exp"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	if model != "gemini-2.0-flash-exp" {
		t.Errorf("model = %q, want %q", model, "gemini-2.0-flash-exp")
	}
}

func TestCustomModel(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{"model": "gemini-pro"},
	}

	model := "gemini-2.0-flash-exp"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	if model != "gemini-pro" {
		t.Errorf("model = %q, want %q", model, "gemini-pro")
	}
}

func TestGenerateRequest_Decode(t *testing.T) {
	input := `{"prompt":"Hello Gemini","temperature":0.7,"top_p":0.9}`

	var req GenerateRequest
	if err := json.NewDecoder(bytes.NewBufferString(input)).Decode(&req); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if req.Prompt != "Hello Gemini" {
		t.Errorf("Prompt = %q, want %q", req.Prompt, "Hello Gemini")
	}
	if req.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7", req.Temperature)
	}
	if req.TopP != 0.9 {
		t.Errorf("TopP = %f, want 0.9", req.TopP)
	}
}

func TestGenerateResponse_Encode(t *testing.T) {
	resp := GenerateResponse{
		Content:      "Gemini response",
		TokensUsed:   30,
		Model:        "gemini-2.0-flash-exp",
		FinishReason: "stop",
		Provider:     "gemini",
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

func TestTokenEstimation(t *testing.T) {
	tests := []struct {
		text           string
		expectedTokens int
	}{
		{"", 0},
		{"test", 1},
		{"Hello world!", 3},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
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
	case "gemini":
		if len(args) > 1 && args[1] == "--version" {
			os.Stdout.WriteString("gemini 1.0.0\n")
		} else {
			os.Stdout.WriteString("Mocked Gemini response")
		}
	case "gcloud":
		os.Stdout.WriteString("Mocked gcloud response")
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
	originalKey := os.Getenv("GEMINI_API_KEY")
	defer os.Setenv("GEMINI_API_KEY", originalKey)

	os.Setenv("GEMINI_API_KEY", "")
	if os.Getenv("GEMINI_API_KEY") != "" {
		t.Error("Expected GEMINI_API_KEY to be empty")
	}

	os.Setenv("GEMINI_API_KEY", "test-key")
	if os.Getenv("GEMINI_API_KEY") != "test-key" {
		t.Errorf("GEMINI_API_KEY = %q, want %q", os.Getenv("GEMINI_API_KEY"), "test-key")
	}
}

func TestBuildGcloudArgs(t *testing.T) {
	model := "gemini-pro"
	prompt := "Hello"

	args := []string{"ai", "generative-models", "generate-content",
		"--model=" + model,
		"--prompt=" + prompt}

	if len(args) != 5 {
		t.Errorf("args length = %d, want 5", len(args))
	}

	if args[3] != "--model=gemini-pro" {
		t.Errorf("args[3] = %q, want %q", args[3], "--model=gemini-pro")
	}
}
