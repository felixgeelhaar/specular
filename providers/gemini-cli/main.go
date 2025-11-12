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

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  generate  - Generate text from prompt\n")
		fmt.Fprintf(os.Stderr, "  stream    - Stream text generation (not supported)\n")
		fmt.Fprintf(os.Stderr, "  health    - Check if gemini CLI is available\n")
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
		fmt.Fprintf(os.Stderr, "Streaming not supported by Gemini CLI\n")
		os.Exit(1)
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

	// Get model from config, default to gemini-2.5-pro
	model := "gemini-2.5-pro"
	if modelVal, ok := req.Config["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	// Build command: gemini --model <model> --prompt "<prompt>"
	args := []string{
		"--model", model,
		"--prompt", req.Prompt,
	}

	// Execute command
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gemini", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gemini command failed: %w: %s", err, string(output))
	}

	// Convert to our response format
	resp := GenerateResponse{
		Content:      string(output),
		TokensUsed:   0, // Gemini CLI doesn't report token usage
		InputTokens:  0,
		OutputTokens: 0,
		Model:        model,
		Latency:      time.Since(startTime),
		FinishReason: "stop",
		Provider:     "gemini-cli",
	}

	// Write response to stdout
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(resp); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return nil
}

func handleHealth() error {
	// Check if gemini CLI is available
	cmd := exec.Command("gemini", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gemini not available: %w", err)
	}
	fmt.Println("OK")
	return nil
}
