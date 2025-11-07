package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// ExecutableProvider wraps any executable that speaks JSON over stdin/stdout
// This is the simplest provider type - any program can be a provider
type ExecutableProvider struct {
	// path is the path to the executable
	path string

	// args are additional arguments to pass to the executable
	args []string

	// info contains provider metadata
	info *ProviderInfo

	// capabilities describes what this provider can do
	capabilities *ProviderCapabilities

	// config contains provider-specific configuration
	config map[string]interface{}
}

// NewExecutableProvider creates a new executable-based provider
func NewExecutableProvider(path string, config *ProviderConfig) (*ExecutableProvider, error) {
	// Verify executable exists
	if _, err := exec.LookPath(path); err != nil {
		return nil, fmt.Errorf("executable not found: %s: %w", path, err)
	}

	// Extract args from config if provided
	args := []string{}
	if argsVal, ok := config.Config["args"]; ok {
		if argsList, ok := argsVal.([]interface{}); ok {
			for _, arg := range argsList {
				if str, ok := arg.(string); ok {
					args = append(args, str)
				}
			}
		}
	}

	// Create provider info
	info := &ProviderInfo{
		Name:        config.Name,
		Version:     config.Version,
		Type:        ProviderTypeCLI,
		TrustLevel:  TrustLevelCommunity, // Default to community
		Description: fmt.Sprintf("Executable provider: %s", path),
	}

	// Set trust level from config if provided
	if trustLevel, ok := config.Config["trust_level"].(string); ok {
		info.TrustLevel = TrustLevel(trustLevel)
	}

	// Get capabilities (or use defaults)
	capabilities := &ProviderCapabilities{
		SupportsStreaming:    false, // Most executables don't support streaming
		SupportsTools:        false,
		SupportsMultiTurn:    false,
		SupportsVision:       false,
		MaxContextTokens:     4096, // Conservative default
		CostPer1KTokens:      0.0,  // Assume free for executable providers
	}

	// Override capabilities from config if provided
	if caps, ok := config.Config["capabilities"].(map[string]interface{}); ok {
		if streaming, ok := caps["streaming"].(bool); ok {
			capabilities.SupportsStreaming = streaming
		}
		if tools, ok := caps["tools"].(bool); ok {
			capabilities.SupportsTools = tools
		}
		if multiTurn, ok := caps["multi_turn"].(bool); ok {
			capabilities.SupportsMultiTurn = multiTurn
		}
		if maxTokens, ok := caps["max_context_tokens"].(float64); ok {
			capabilities.MaxContextTokens = int(maxTokens)
		}
	}

	return &ExecutableProvider{
		path:         path,
		args:         args,
		info:         info,
		capabilities: capabilities,
		config:       config.Config,
	}, nil
}

// Generate sends a prompt to the executable and returns the response
func (e *ExecutableProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	// Build command with args
	cmdArgs := append(e.args, "generate")
	cmd := exec.CommandContext(ctx, e.path, cmdArgs...)

	// Prepare request as JSON
	requestJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Write request to stdin
	go func() {
		defer stdin.Close()
		stdin.Write(requestJSON)
	}()

	// Execute and capture output
	output, err := cmd.Output()
	if err != nil {
		// Try to extract error message from stderr
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("provider failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute provider: %w", err)
	}

	// Parse response
	var resp GenerateResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse provider response: %w", err)
	}

	// Fill in metadata
	resp.Latency = time.Since(startTime)
	// Use "ollama" as provider name if not already set
	if resp.Provider == "" {
		resp.Provider = e.info.Name
	}

	return &resp, nil
}

// Stream implements ProviderClient.Stream for executable providers
func (e *ExecutableProvider) Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error) {
	if !e.capabilities.SupportsStreaming {
		return nil, fmt.Errorf("streaming not supported by this provider")
	}

	chunkChan := make(chan StreamChunk, 10)

	// Marshal request to JSON
	reqJSON, err := json.Marshal(req)
	if err != nil {
		close(chunkChan)
		return chunkChan, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Prepare command with "stream" argument
	cmdArgs := append([]string{"stream"}, e.args...)
	cmd := exec.CommandContext(ctx, e.path, cmdArgs...)
	cmd.Stdin = bytes.NewReader(reqJSON)

	// Get stdout pipe for line-by-line reading
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		close(chunkChan)
		return chunkChan, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		close(chunkChan)
		return chunkChan, fmt.Errorf("failed to start command: %w", err)
	}

	// Start goroutine to read stream chunks
	go func() {
		defer close(chunkChan)
		defer stdout.Close()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			// Parse stream chunk from JSON
			var chunk StreamChunk
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				chunkChan <- StreamChunk{
					Error: fmt.Errorf("failed to parse stream chunk: %w", err),
					Done:  true,
				}
				return
			}

			chunkChan <- chunk

			if chunk.Done {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			chunkChan <- StreamChunk{
				Error: fmt.Errorf("stream read error: %w", err),
				Done:  true,
			}
		}

		// Wait for command to finish
		cmd.Wait()
	}()

	return chunkChan, nil
}

// GetCapabilities returns what this provider supports
func (e *ExecutableProvider) GetCapabilities() *ProviderCapabilities {
	return e.capabilities
}

// GetInfo returns provider metadata
func (e *ExecutableProvider) GetInfo() *ProviderInfo {
	return e.info
}

// IsAvailable checks if the executable exists and is accessible
func (e *ExecutableProvider) IsAvailable() bool {
	_, err := exec.LookPath(e.path)
	return err == nil
}

// Health performs a health check by calling the provider with a simple request
func (e *ExecutableProvider) Health(ctx context.Context) error {
	// Build health check command
	cmdArgs := append(e.args, "health")
	cmd := exec.CommandContext(ctx, e.path, cmdArgs...)

	// Set a timeout for health check
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd = exec.CommandContext(healthCtx, e.path, cmdArgs...)

	// Run health check
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Close cleans up resources (executable providers typically don't need cleanup)
func (e *ExecutableProvider) Close() error {
	// Nothing to clean up for executable providers
	return nil
}
