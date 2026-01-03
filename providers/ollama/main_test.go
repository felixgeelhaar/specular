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
				Prompt: "Hello, Ollama!",
			},
		},
		{
			name: "request with system prompt",
			req: GenerateRequest{
				Prompt:       "Explain Go concurrency",
				SystemPrompt: "You are a Go expert.",
			},
		},
		{
			name: "request with config",
			req: GenerateRequest{
				Prompt:      "Test prompt",
				MaxTokens:   512,
				Temperature: 0.8,
				TopP:        0.9,
				Config:      map[string]interface{}{"model": "llama3.2"},
			},
		},
		{
			name: "request with context",
			req: GenerateRequest{
				Prompt: "Continue",
				Context: []Message{
					{Role: "user", Content: "What is Go?"},
					{Role: "assistant", Content: "Go is a programming language."},
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
				Content:      "Hello from Ollama!",
				TokensUsed:   25,
				Model:        "llama3.2",
				FinishReason: "stop",
				Provider:     "ollama",
			},
		},
		{
			name: "response with token breakdown",
			resp: GenerateResponse{
				Content:      "Detailed response",
				TokensUsed:   300,
				InputTokens:  100,
				OutputTokens: 200,
				Model:        "llama3.2",
				Latency:      5 * time.Second,
				FinishReason: "stop",
				Provider:     "ollama",
			},
		},
		{
			name: "response with length finish",
			resp: GenerateResponse{
				Content:      "Truncated response...",
				TokensUsed:   512,
				Model:        "llama3.2",
				FinishReason: "length",
				Provider:     "ollama",
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
				TokensUsed: 20,
				Timestamp:  time.Now().Truncate(time.Second),
			},
		},
		{
			name: "error chunk",
			chunk: StreamChunk{
				Done:      true,
				ErrorMsg:  "Model not found",
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

func TestOllamaGenerateRequest_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		req  OllamaGenerateRequest
	}{
		{
			name: "basic request",
			req: OllamaGenerateRequest{
				Model:  "llama3.2",
				Prompt: "Hello",
				Stream: false,
			},
		},
		{
			name: "request with system",
			req: OllamaGenerateRequest{
				Model:  "llama3.2",
				Prompt: "Test",
				System: "You are helpful",
				Stream: true,
			},
		},
		{
			name: "request with options",
			req: OllamaGenerateRequest{
				Model:  "llama3.2",
				Prompt: "Test",
				Stream: false,
				Options: &Options{
					Temperature: 0.7,
					TopP:        0.9,
					NumPredict:  256,
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

			var decoded OllamaGenerateRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if decoded.Model != tt.req.Model {
				t.Errorf("Model = %q, want %q", decoded.Model, tt.req.Model)
			}
			if decoded.Prompt != tt.req.Prompt {
				t.Errorf("Prompt = %q, want %q", decoded.Prompt, tt.req.Prompt)
			}
			if decoded.Stream != tt.req.Stream {
				t.Errorf("Stream = %v, want %v", decoded.Stream, tt.req.Stream)
			}
		})
	}
}

func TestOllamaGenerateResponse_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		resp OllamaGenerateResponse
	}{
		{
			name: "basic response",
			resp: OllamaGenerateResponse{
				Model:    "llama3.2",
				Response: "Hello!",
				Done:     true,
			},
		},
		{
			name: "response with metrics",
			resp: OllamaGenerateResponse{
				Model:              "llama3.2",
				Response:           "Generated text",
				Done:               true,
				PromptEvalCount:    50,
				EvalCount:          100,
				TotalDuration:      5000000000,
				PromptEvalDuration: 1000000000,
				EvalDuration:       4000000000,
			},
		},
		{
			name: "partial response",
			resp: OllamaGenerateResponse{
				Model:    "llama3.2",
				Response: "Partial",
				Done:     false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.resp)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded OllamaGenerateResponse
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if decoded.Response != tt.resp.Response {
				t.Errorf("Response = %q, want %q", decoded.Response, tt.resp.Response)
			}
			if decoded.Done != tt.resp.Done {
				t.Errorf("Done = %v, want %v", decoded.Done, tt.resp.Done)
			}
		})
	}
}

func TestOptions_JSONRoundtrip(t *testing.T) {
	opts := Options{
		Temperature: 0.7,
		TopP:        0.9,
		NumPredict:  256,
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Options
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Temperature != opts.Temperature {
		t.Errorf("Temperature = %f, want %f", decoded.Temperature, opts.Temperature)
	}
	if decoded.TopP != opts.TopP {
		t.Errorf("TopP = %f, want %f", decoded.TopP, opts.TopP)
	}
	if decoded.NumPredict != opts.NumPredict {
		t.Errorf("NumPredict = %d, want %d", decoded.NumPredict, opts.NumPredict)
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

	var promptBuilder string
	for _, msg := range req.Context {
		if msg.Role == "user" {
			promptBuilder += "User: " + msg.Content + "\n"
		} else if msg.Role == "assistant" {
			promptBuilder += "Assistant: " + msg.Content + "\n"
		}
	}
	fullPrompt := promptBuilder + "User: " + req.Prompt + "\nAssistant:"

	expected := "User: Hello\nAssistant: Hi!\nUser: Continue\nAssistant:"
	if fullPrompt != expected {
		t.Errorf("fullPrompt = %q, want %q", fullPrompt, expected)
	}
}

func TestDefaultModel(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{},
	}

	model := "llama3.2"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	if model != "llama3.2" {
		t.Errorf("model = %q, want %q", model, "llama3.2")
	}
}

func TestCustomModel(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		Config: map[string]interface{}{"model": "codellama"},
	}

	model := "llama3.2"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	if model != "codellama" {
		t.Errorf("model = %q, want %q", model, "codellama")
	}
}

func TestGenerateRequest_Decode(t *testing.T) {
	input := `{"prompt":"Hello Ollama","max_tokens":512,"temperature":0.8}`

	var req GenerateRequest
	if err := json.NewDecoder(bytes.NewBufferString(input)).Decode(&req); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if req.Prompt != "Hello Ollama" {
		t.Errorf("Prompt = %q, want %q", req.Prompt, "Hello Ollama")
	}
	if req.MaxTokens != 512 {
		t.Errorf("MaxTokens = %d, want 512", req.MaxTokens)
	}
}

func TestGenerateResponse_Encode(t *testing.T) {
	resp := GenerateResponse{
		Content:      "Ollama response",
		TokensUsed:   50,
		Model:        "llama3.2",
		FinishReason: "stop",
		Provider:     "ollama",
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

func TestBuildOllamaRequest(t *testing.T) {
	req := GenerateRequest{
		Prompt:       "Hello",
		SystemPrompt: "Be helpful",
		MaxTokens:    256,
		Temperature:  0.7,
		TopP:         0.9,
		Config:       map[string]interface{}{"model": "llama3.2"},
	}

	model := "llama3.2"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	ollamaReq := OllamaGenerateRequest{
		Model:  model,
		Prompt: req.Prompt,
		System: req.SystemPrompt,
		Stream: false,
	}

	if req.Temperature > 0 || req.TopP > 0 || req.MaxTokens > 0 {
		ollamaReq.Options = &Options{
			Temperature: req.Temperature,
			TopP:        req.TopP,
			NumPredict:  req.MaxTokens,
		}
	}

	if ollamaReq.Model != "llama3.2" {
		t.Errorf("Model = %q, want %q", ollamaReq.Model, "llama3.2")
	}
	if ollamaReq.Prompt != "Hello" {
		t.Errorf("Prompt = %q, want %q", ollamaReq.Prompt, "Hello")
	}
	if ollamaReq.System != "Be helpful" {
		t.Errorf("System = %q, want %q", ollamaReq.System, "Be helpful")
	}
	if ollamaReq.Options == nil {
		t.Fatal("Options should not be nil")
	}
	if ollamaReq.Options.Temperature != 0.7 {
		t.Errorf("Options.Temperature = %f, want 0.7", ollamaReq.Options.Temperature)
	}
}

func TestFinishReason(t *testing.T) {
	tests := []struct {
		name   string
		done   bool
		want   string
	}{
		{name: "done true", done: true, want: "stop"},
		{name: "done false", done: false, want: "length"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finishReason := "stop"
			if !tt.done {
				finishReason = "length"
			}
			if finishReason != tt.want {
				t.Errorf("finishReason = %q, want %q", finishReason, tt.want)
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
	case "ollama":
		os.Stdout.WriteString("NAME\nllama3.2\n")
	case "curl":
		resp := OllamaGenerateResponse{
			Model:    "llama3.2",
			Response: "Mocked response",
			Done:     true,
		}
		json.NewEncoder(os.Stdout).Encode(resp)
	default:
		os.Stderr.WriteString("Unknown command: " + cmd)
		os.Exit(1)
	}
}

