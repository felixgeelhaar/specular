package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
	ErrorMsg   string    `json:"error,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  generate  - Generate text/code from prompt\n")
		fmt.Fprintf(os.Stderr, "  stream    - Stream text generation\n")
		fmt.Fprintf(os.Stderr, "  health    - Check if codex CLI is available\n")
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

	// Build prompt for codex CLI
	fullPrompt := req.Prompt
	if req.SystemPrompt != "" {
		fullPrompt = fmt.Sprintf("%s\n\n%s", req.SystemPrompt, req.Prompt)
	}

	// Add conversation context if provided
	if len(req.Context) > 0 {
		var promptBuilder strings.Builder
		for _, msg := range req.Context {
			promptBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}
		promptBuilder.WriteString(fmt.Sprintf("user: %s", req.Prompt))
		fullPrompt = promptBuilder.String()
	}

	// Call codex CLI
	// Usage: codex [options] <prompt>
	// or: openai api completions.create -m gpt-3.5-turbo-instruct -p "prompt"
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	args := []string{"api", "completions.create"}

	// Get model from config or use default
	model := "gpt-3.5-turbo-instruct"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}
	args = append(args, "-m", model)

	// Add max tokens if specified
	if req.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxTokens))
	}

	// Add temperature if specified
	if req.Temperature > 0 {
		args = append(args, "-t", fmt.Sprintf("%.2f", req.Temperature))
	}

	// Add top_p if specified
	if req.TopP > 0 {
		args = append(args, "--top-p", fmt.Sprintf("%.2f", req.TopP))
	}

	// Add the prompt
	args = append(args, "-p", fullPrompt)

	// Use openai CLI (codex is accessed via openai CLI)
	cmd := exec.CommandContext(ctx, "openai", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("codex CLI call failed: %w\nOutput: %s", err, string(output))
	}

	content := strings.TrimSpace(string(output))

	// Estimate tokens (rough approximation: ~4 chars per token)
	inputTokens := len(fullPrompt) / 4
	outputTokens := len(content) / 4

	// Convert to our response format
	resp := GenerateResponse{
		Content:      content,
		TokensUsed:   inputTokens + outputTokens,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Model:        model,
		Latency:      time.Since(startTime),
		FinishReason: "stop",
		Provider:     "codex",
	}

	// Write response to stdout
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(resp); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return nil
}

func handleStream() error {
	// For codex CLI, streaming might not be directly supported
	// Fall back to non-streaming generation and emit as single chunk

	// Read request from stdin
	var req GenerateRequest
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request: %w", err)
	}

	startTime := time.Now()

	// Build prompt
	fullPrompt := req.Prompt
	if req.SystemPrompt != "" {
		fullPrompt = fmt.Sprintf("%s\n\n%s", req.SystemPrompt, req.Prompt)
	}

	// Call codex CLI (same as generate)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	model := "gpt-3.5-turbo-instruct"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	args := []string{"api", "completions.create", "-m", model, "-p", fullPrompt}

	cmd := exec.CommandContext(ctx, "openai", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Output error chunk
		chunk := StreamChunk{
			Content:   "",
			Done:      true,
			ErrorMsg:  fmt.Sprintf("codex CLI call failed: %v", err),
			Timestamp: time.Now(),
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.Encode(chunk)
		return err
	}

	content := strings.TrimSpace(string(output))

	// Estimate tokens
	inputTokens := len(fullPrompt) / 4
	outputTokens := len(content) / 4

	// Emit single chunk with full response
	chunk := StreamChunk{
		Content:    content,
		Delta:      content,
		Done:       true,
		TokensUsed: inputTokens + outputTokens,
		Timestamp:  time.Now(),
	}

	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(chunk); err != nil {
		return fmt.Errorf("failed to encode chunk: %w", err)
	}

	_ = startTime // Suppress unused warning
	return nil
}

func handleHealth() error {
	// Check if openai CLI is available (codex accessed via openai CLI)
	cmd := exec.Command("openai", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("codex/openai CLI not available: %w", err)
	}

	// Check if OPENAI_API_KEY is set
	if os.Getenv("OPENAI_API_KEY") == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	fmt.Println("OK")
	return nil
}
