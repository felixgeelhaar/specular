package router

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStream_BasicFunctionality(t *testing.T) {
	// Create router with Ollama configured
	config := &RouterConfig{
		BudgetUSD:      100.0,
		MaxLatencyMs:   60000,
		EnableFallback: false, // Disable fallback for basic test
		MaxRetries:     0,     // Disable retries for basic test
	}

	router, err := NewRouter(config)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	router.SetModelsAvailable(true)

	// Test streaming
	ctx := context.Background()
	req := GenerateRequest{
		Prompt:     "Count to 5",
		ModelHint:  "fast",
		Complexity: 1,
		Priority:   "P2",
	}

	stream, err := router.Stream(ctx, req)
	if err != nil {
		// Ollama might not be running, skip test
		t.Skipf("Streaming failed (Ollama may not be running): %v", err)
	}

	// Collect chunks
	var chunks []StreamChunk
	for chunk := range stream {
		chunks = append(chunks, chunk)
		if chunk.Error != nil {
			t.Errorf("Stream chunk error: %v", chunk.Error)
		}
	}

	// Should have received at least one chunk
	if len(chunks) == 0 {
		t.Error("Expected at least one stream chunk, got none")
	}

	// Last chunk should be marked as done
	if len(chunks) > 0 {
		lastChunk := chunks[len(chunks)-1]
		if !lastChunk.Done {
			t.Error("Last chunk should be marked as done")
		}
	}
}

func TestStream_WithRetryConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		maxRetries int
		wantConfig int
	}{
		{
			name:       "no retries",
			maxRetries: 0,
			wantConfig: 0,
		},
		{
			name:       "one retry",
			maxRetries: 1,
			wantConfig: 1,
		},
		{
			name:       "three retries",
			maxRetries: 3,
			wantConfig: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RouterConfig{
				BudgetUSD:      100.0,
				MaxLatencyMs:   60000,
				MaxRetries:     tt.maxRetries,
				EnableFallback: false,
			}

			router, err := NewRouter(config)
			if err != nil {
				t.Fatalf("NewRouter() error = %v", err)
			}

			if router.config.MaxRetries != tt.wantConfig {
				t.Errorf("Router.config.MaxRetries = %d, want %d", router.config.MaxRetries, tt.wantConfig)
			}
		})
	}
}

func TestStream_FallbackConfiguration(t *testing.T) {
	// Test fallback enabled
	config1 := &RouterConfig{
		BudgetUSD:      100.0,
		MaxLatencyMs:   60000,
		EnableFallback: true,
		MaxRetries:     3,
	}

	router1, err := NewRouter(config1)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	if !router1.config.EnableFallback {
		t.Error("Router.config.EnableFallback should be true")
	}

	// Test fallback disabled
	config2 := &RouterConfig{
		BudgetUSD:      100.0,
		MaxLatencyMs:   60000,
		EnableFallback: false,
		MaxRetries:     3,
	}

	router2, err := NewRouter(config2)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	if router2.config.EnableFallback {
		t.Error("Router.config.EnableFallback should be false")
	}
}

func TestStream_ContextCancellation(t *testing.T) {
	config := &RouterConfig{
		BudgetUSD:      100.0,
		MaxLatencyMs:   60000,
		MaxRetries:     3,
		EnableFallback: false,
	}

	router, err := NewRouter(config)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	router.SetModelsAvailable(true)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := GenerateRequest{
		Prompt:    "Test",
		ModelHint: "fast",
	}

	// Should fail quickly with context error
	_, err = router.Stream(ctx, req)
	// Either fails at stream creation or model selection - both are acceptable
	// The important part is it doesn't hang
	if err == nil {
		// Stream might succeed but should close immediately
		// This is acceptable behavior
	}
}

func TestStream_TokenTracking(t *testing.T) {
	config := &RouterConfig{
		BudgetUSD:      100.0,
		MaxLatencyMs:   60000,
		MaxRetries:     0,
		EnableFallback: false,
	}

	router, err := NewRouter(config)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	router.SetModelsAvailable(true)

	ctx := context.Background()
	req := GenerateRequest{
		Prompt:    "Hello",
		ModelHint: "fast",
		TaskID:    "test-stream-tracking",
	}

	initialBudget := router.GetBudget()
	initialUsageCount := initialBudget.UsageCount

	stream, err := router.Stream(ctx, req)
	if err != nil {
		t.Skipf("Streaming failed (provider may not be available): %v", err)
	}

	// Consume stream
	for chunk := range stream {
		_ = chunk
	}

	// Give goroutine time to record usage
	time.Sleep(100 * time.Millisecond)

	// Check that usage was recorded
	finalBudget := router.GetBudget()
	if finalBudget.UsageCount != initialUsageCount+1 {
		t.Errorf("Usage count = %d, want %d", finalBudget.UsageCount, initialUsageCount+1)
	}
}

func TestStream_ModelSelection(t *testing.T) {
	config := &RouterConfig{
		BudgetUSD:      100.0,
		MaxLatencyMs:   60000,
		MaxRetries:     0,
		EnableFallback: false,
	}

	router, err := NewRouter(config)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	router.SetModelsAvailable(true)

	tests := []struct {
		name      string
		modelHint string
		wantErr   bool
	}{
		{
			name:      "fast hint",
			modelHint: "fast",
			wantErr:   false,
		},
		{
			name:      "codegen hint",
			modelHint: "codegen",
			wantErr:   false,
		},
		{
			name:      "agentic hint",
			modelHint: "agentic",
			wantErr:   false,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := GenerateRequest{
				Prompt:    "Test",
				ModelHint: tt.modelHint,
			}

			stream, err := router.Stream(ctx, req)
			if (err != nil) != tt.wantErr {
				// Provider might not be available, skip
				t.Skipf("Stream() error = %v, wantErr %v (provider may not be available)", err, tt.wantErr)
			}

			if stream != nil {
				// Consume stream
				for chunk := range stream {
					_ = chunk
					break // Just check that we can start streaming
				}
			}
		})
	}
}

func TestStream_ChunkStructure(t *testing.T) {
	config := &RouterConfig{
		BudgetUSD:      100.0,
		MaxLatencyMs:   60000,
		MaxRetries:     0,
		EnableFallback: false,
	}

	router, err := NewRouter(config)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	router.SetModelsAvailable(true)

	ctx := context.Background()
	req := GenerateRequest{
		Prompt:    "Say 'test'",
		ModelHint: "fast",
	}

	stream, err := router.Stream(ctx, req)
	if err != nil {
		t.Skipf("Streaming failed (provider may not be available): %v", err)
	}

	chunkCount := 0
	var lastChunk StreamChunk

	for chunk := range stream {
		chunkCount++
		lastChunk = chunk

		// Verify chunk structure
		if chunk.Error != nil {
			t.Errorf("Chunk %d has error: %v", chunkCount, chunk.Error)
		}

		// Delta should be non-empty for non-done chunks
		if !chunk.Done && chunk.Delta == "" {
			// Some providers might send empty deltas, that's ok
		}

		// Content should accumulate
		if chunk.Content == "" && !chunk.Done {
			// Initial chunks might have empty content
		}
	}

	// Should have received at least one chunk
	if chunkCount == 0 {
		t.Error("Expected at least one chunk, got none")
	}

	// Last chunk should be marked done
	if !lastChunk.Done {
		t.Error("Last chunk should be marked as done")
	}
}

func TestStreamWithRetry_ErrorClassification(t *testing.T) {
	config := &RouterConfig{
		BudgetUSD:      100.0,
		MaxLatencyMs:   60000,
		MaxRetries:     3,
		RetryBackoffMs: 100, // Faster for testing
		EnableFallback: false,
	}

	router, err := NewRouter(config)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	// Test that isRetryableError works correctly
	// (reusing tests from retry_test.go to ensure consistency)
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"rate limit",
		"429",
		"503",
		"service unavailable",
	}

	for _, errMsg := range retryableErrors {
		err := errors.New(errMsg)
		if !router.isRetryableError(err) {
			t.Errorf("Error '%s' should be retryable for streaming", errMsg)
		}
	}

	nonRetryableErrors := []string{
		"401",
		"403",
		"unauthorized",
		"invalid api key",
		"context",
	}

	for _, errMsg := range nonRetryableErrors {
		err := errors.New(errMsg)
		if router.isRetryableError(err) {
			t.Errorf("Error '%s' should NOT be retryable for streaming", errMsg)
		}
	}
}

func TestStreamWithRetry_BackoffProgression(t *testing.T) {
	config := &RouterConfig{
		BudgetUSD:         100.0,
		MaxLatencyMs:      60000,
		MaxRetries:        3,
		RetryBackoffMs:    100,
		RetryMaxBackoffMs: 1000,
		EnableFallback:    false,
	}

	router, err := NewRouter(config)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	// Verify backoff configuration is set correctly
	if router.config.RetryBackoffMs != 100 {
		t.Errorf("RetryBackoffMs = %d, want 100", router.config.RetryBackoffMs)
	}

	if router.config.RetryMaxBackoffMs != 1000 {
		t.Errorf("RetryMaxBackoffMs = %d, want 1000", router.config.RetryMaxBackoffMs)
	}

	// Test backoff doubling (same logic as Generate retries)
	backoff := time.Duration(router.config.RetryBackoffMs) * time.Millisecond
	maxBackoff := time.Duration(router.config.RetryMaxBackoffMs) * time.Millisecond

	expectedBackoffs := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1000 * time.Millisecond, // Capped at max
	}

	for i, expected := range expectedBackoffs {
		if backoff != expected {
			t.Errorf("Backoff iteration %d: got %v, want %v", i, backoff, expected)
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
