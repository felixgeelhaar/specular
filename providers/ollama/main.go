package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// GenerateRequest matches internal/provider/types.go
type GenerateRequest struct {
	Prompt       string                 `json:"prompt"`
	SystemPrompt string                 `json:"system_prompt,omitempty"`
	MaxTokens    int                    `json:"max_tokens,omitempty"`
	Temperature  float64                `json:"temperature,omitempty"`
	TopP         float64                `json:"top_p,omitempty"`
	Tools        []interface{}          `json:"tools,omitempty"`
	Context      []Message              `json:"context,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
}

// GenerateResponse matches internal/provider/types.go
type GenerateResponse struct {
	Content      string        `json:"content"`
	TokensUsed   int           `json:"tokens_used"`
	InputTokens  int           `json:"input_tokens,omitempty"`
	OutputTokens int           `json:"output_tokens,omitempty"`
	Model        string        `json:"model"`
	Latency      time.Duration `json:"latency"`
	FinishReason string        `json:"finish_reason"`
	Error        string        `json:"error,omitempty"`
	Provider     string        `json:"provider"`
}

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamChunk matches internal/provider/interface.go
type StreamChunk struct {
	Content    string    `json:"content"`
	Delta      string    `json:"delta"`
	Done       bool      `json:"done"`
	TokensUsed int       `json:"tokens_used,omitempty"`
	ErrorMsg   string    `json:"error,omitempty"` // JSON-serializable error message
	Timestamp  time.Time `json:"timestamp"`
}

// OllamaGenerateRequest is the format ollama CLI expects
type OllamaGenerateRequest struct {
	Model   string   `json:"model"`
	Prompt  string   `json:"prompt"`
	System  string   `json:"system,omitempty"`
	Stream  bool     `json:"stream"`
	Options *Options `json:"options,omitempty"`
}

// Options for ollama generation
type Options struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"` // max tokens
}

// OllamaGenerateResponse is what ollama returns
type OllamaGenerateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  generate  - Generate text from prompt\n")
		fmt.Fprintf(os.Stderr, "  stream    - Stream text generation\n")
		fmt.Fprintf(os.Stderr, "  health    - Check if ollama is available\n")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "generate":
		if err := handleGenerate(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "stream":
		if err := handleStream(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "health":
		if err := handleHealth(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func handleGenerate() error {
	// Read request from stdin
	var req GenerateRequest
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request: %w", err)
	}

	startTime := time.Now()

	// Get model from config, default to llama3.2
	model := "llama3.2"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	// Build conversation prompt if context is provided
	fullPrompt := req.Prompt
	if len(req.Context) > 0 {
		// Build conversation history into the prompt
		var promptBuilder string
		for _, msg := range req.Context {
			if msg.Role == "user" {
				promptBuilder += "User: " + msg.Content + "\n"
			} else if msg.Role == "assistant" {
				promptBuilder += "Assistant: " + msg.Content + "\n"
			}
		}
		fullPrompt = promptBuilder + "User: " + req.Prompt + "\nAssistant:"
	}

	// Build ollama request
	ollamaReq := OllamaGenerateRequest{
		Model:  model,
		Prompt: fullPrompt,
		System: req.SystemPrompt,
		Stream: false,
	}

	// Add options if provided
	if req.Temperature > 0 || req.TopP > 0 || req.MaxTokens > 0 {
		ollamaReq.Options = &Options{
			Temperature: req.Temperature,
			TopP:        req.TopP,
			NumPredict:  req.MaxTokens,
		}
	}

	// Convert to JSON for ollama
	reqJSON, err := json.Marshal(ollamaReq)
	if err != nil {
		return fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	// Call ollama using the generate API for clean JSON output
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Use curl to call ollama API directly for clean JSON
	cmd := exec.CommandContext(ctx, "curl", "-s", "http://localhost:11434/api/generate",
		"-d", string(reqJSON))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ollama API call failed: %w\nOutput: %s", err, string(output))
	}

	// Parse ollama response
	var ollamaResp OllamaGenerateResponse
	if err := json.Unmarshal(output, &ollamaResp); err != nil {
		// If JSON parsing fails, try to extract plain text response
		resp := GenerateResponse{
			Content:      string(output),
			TokensUsed:   0,
			Model:        model,
			Latency:      time.Since(startTime),
			FinishReason: "stop",
			Provider:     "ollama",
		}
		return json.NewEncoder(os.Stdout).Encode(resp)
	}

	// Convert to our response format
	resp := GenerateResponse{
		Content:      ollamaResp.Response,
		TokensUsed:   ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		InputTokens:  ollamaResp.PromptEvalCount,
		OutputTokens: ollamaResp.EvalCount,
		Model:        ollamaResp.Model,
		Latency:      time.Since(startTime),
		FinishReason: "stop",
		Provider:     "ollama",
	}

	if !ollamaResp.Done {
		resp.FinishReason = "length"
	}

	// Write response to stdout
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(resp); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return nil
}

func handleStream() error {
	// Read request from stdin
	var req GenerateRequest
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request: %w", err)
	}

	// Get model from config, default to llama3.2
	model := "llama3.2"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	// Build conversation prompt if context is provided
	fullPrompt := req.Prompt
	if len(req.Context) > 0 {
		var promptBuilder string
		for _, msg := range req.Context {
			if msg.Role == "user" {
				promptBuilder += "User: " + msg.Content + "\n"
			} else if msg.Role == "assistant" {
				promptBuilder += "Assistant: " + msg.Content + "\n"
			}
		}
		fullPrompt = promptBuilder + "User: " + req.Prompt + "\nAssistant:"
	}

	// Build ollama request with streaming enabled
	ollamaReq := OllamaGenerateRequest{
		Model:  model,
		Prompt: fullPrompt,
		System: req.SystemPrompt,
		Stream: true, // Enable streaming
	}

	// Add options if provided
	if req.Temperature > 0 || req.TopP > 0 || req.MaxTokens > 0 {
		ollamaReq.Options = &Options{
			Temperature: req.Temperature,
			TopP:        req.TopP,
			NumPredict:  req.MaxTokens,
		}
	}

	// Convert to JSON for ollama
	reqJSON, err := json.Marshal(ollamaReq)
	if err != nil {
		return fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	// Call ollama using curl with streaming
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "curl", "-s", "-N", "http://localhost:11434/api/generate",
		"-d", string(reqJSON))

	// Get stdout pipe to read streaming response
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ollama API call: %w", err)
	}

	// Read streaming response line-by-line
	var fullContent string
	var totalTokens int
	encoder := json.NewEncoder(os.Stdout)
	scanner := json.NewDecoder(stdout)

	for scanner.More() {
		var ollamaResp OllamaGenerateResponse
		if err := scanner.Decode(&ollamaResp); err != nil {
			// Output error chunk
			chunk := StreamChunk{
				Content:   fullContent,
				Done:      true,
				ErrorMsg:  fmt.Sprintf("failed to parse ollama response: %v", err),
				Timestamp: time.Now(),
			}
			_ = encoder.Encode(chunk) // Best effort to send error chunk, ignore encoding errors
			return err
		}

		// Update full content with the delta
		delta := ollamaResp.Response
		fullContent += delta

		// Create stream chunk
		chunk := StreamChunk{
			Content:    fullContent,
			Delta:      delta,
			Done:       ollamaResp.Done,
			TokensUsed: 0,
			Timestamp:  time.Now(),
		}

		// Add token counts in final chunk
		if ollamaResp.Done {
			totalTokens = ollamaResp.PromptEvalCount + ollamaResp.EvalCount
			chunk.TokensUsed = totalTokens
		}

		// Output the chunk as newline-delimited JSON
		if err := encoder.Encode(chunk); err != nil {
			return fmt.Errorf("failed to encode chunk: %w", err)
		}

		// Exit if done
		if ollamaResp.Done {
			break
		}
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ollama API call failed: %w", err)
	}

	return nil
}

func handleHealth() error {
	// Check if ollama is available
	cmd := exec.Command("ollama", "list")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ollama not available: %w", err)
	}
	fmt.Println("OK")
	return nil
}
