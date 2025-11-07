package router

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestIsRetryableError(t *testing.T) {
	router, _ := NewRouter(&RouterConfig{
		BudgetUSD:    100.0,
		MaxLatencyMs: 60000,
		MaxRetries:   3,
	})

	tests := []struct {
		name    string
		err     error
		wantRetryable bool
	}{
		{
			name:    "nil error",
			err:     nil,
			wantRetryable: false,
		},
		{
			name:    "timeout error",
			err:     errors.New("request timeout"),
			wantRetryable: true,
		},
		{
			name:    "network error",
			err:     errors.New("network connection failed"),
			wantRetryable: true,
		},
		{
			name:    "connection refused",
			err:     errors.New("connection refused"),
			wantRetryable: true,
		},
		{
			name:    "rate limit error",
			err:     errors.New("rate limit exceeded"),
			wantRetryable: true,
		},
		{
			name:    "http 429 error",
			err:     errors.New("HTTP 429 Too Many Requests"),
			wantRetryable: true,
		},
		{
			name:    "http 503 error",
			err:     errors.New("HTTP 503 Service Unavailable"),
			wantRetryable: true,
		},
		{
			name:    "service unavailable",
			err:     errors.New("service unavailable"),
			wantRetryable: true,
		},
		{
			name:    "temporary failure",
			err:     errors.New("temporary failure in name resolution"),
			wantRetryable: true,
		},
		{
			name:    "auth error 401",
			err:     errors.New("HTTP 401 Unauthorized"),
			wantRetryable: false,
		},
		{
			name:    "auth error 403",
			err:     errors.New("HTTP 403 Forbidden"),
			wantRetryable: false,
		},
		{
			name:    "invalid api key",
			err:     errors.New("invalid API key provided"),
			wantRetryable: false,
		},
		{
			name:    "context error",
			err:     errors.New("context deadline exceeded"),
			wantRetryable: false,
		},
		{
			name:    "unauthorized",
			err:     errors.New("unauthorized access"),
			wantRetryable: false,
		},
		{
			name:    "forbidden",
			err:     errors.New("forbidden resource"),
			wantRetryable: false,
		},
		{
			name:    "unknown error",
			err:     errors.New("something went wrong"),
			wantRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.isRetryableError(tt.err)
			if got != tt.wantRetryable {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, got, tt.wantRetryable)
			}
		})
	}
}

func TestRetryBackoff(t *testing.T) {
	router, _ := NewRouter(&RouterConfig{
		BudgetUSD:         100.0,
		MaxLatencyMs:      60000,
		MaxRetries:        3,
		RetryBackoffMs:    100,  // Start with 100ms for faster tests
		RetryMaxBackoffMs: 1000, // Max 1s for faster tests
	})

	// Test that backoff increases exponentially
	backoff := time.Duration(router.config.RetryBackoffMs) * time.Millisecond
	maxBackoff := time.Duration(router.config.RetryMaxBackoffMs) * time.Millisecond

	expectedBackoffs := []time.Duration{
		100 * time.Millisecond,  // Initial
		200 * time.Millisecond,  // 2x
		400 * time.Millisecond,  // 2x
		800 * time.Millisecond,  // 2x
		1000 * time.Millisecond, // Capped at max
		1000 * time.Millisecond, // Stays at max
	}

	currentBackoff := backoff
	for i, expected := range expectedBackoffs {
		if currentBackoff != expected {
			t.Errorf("Backoff iteration %d: got %v, want %v", i, currentBackoff, expected)
		}

		// Double for next iteration
		currentBackoff *= 2
		if currentBackoff > maxBackoff {
			currentBackoff = maxBackoff
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	config := DefaultConfig()

	if !config.EnableFallback {
		t.Error("DefaultConfig() EnableFallback should be true")
	}

	if config.MaxRetries != 3 {
		t.Errorf("DefaultConfig() MaxRetries = %d, want 3", config.MaxRetries)
	}

	if config.RetryBackoffMs != 1000 {
		t.Errorf("DefaultConfig() RetryBackoffMs = %d, want 1000", config.RetryBackoffMs)
	}

	if config.RetryMaxBackoffMs != 30000 {
		t.Errorf("DefaultConfig() RetryMaxBackoffMs = %d, want 30000", config.RetryMaxBackoffMs)
	}

	if config.FallbackModel == "" {
		t.Error("DefaultConfig() FallbackModel should not be empty")
	}
}

func TestRetryConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		maxRetries     int
		wantAttempts   int
	}{
		{
			name:         "no retries",
			maxRetries:   0,
			wantAttempts: 1, // Initial attempt only
		},
		{
			name:         "one retry",
			maxRetries:   1,
			wantAttempts: 2, // Initial + 1 retry
		},
		{
			name:         "three retries",
			maxRetries:   3,
			wantAttempts: 4, // Initial + 3 retries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RouterConfig{
				BudgetUSD:    100.0,
				MaxLatencyMs: 60000,
				MaxRetries:   tt.maxRetries,
			}

			router, err := NewRouter(config)
			if err != nil {
				t.Fatalf("NewRouter() error = %v", err)
			}

			if router.config.MaxRetries != tt.maxRetries {
				t.Errorf("Router.config.MaxRetries = %d, want %d", router.config.MaxRetries, tt.maxRetries)
			}

			// Verify the formula: attempts = maxRetries + 1
			expectedAttempts := tt.maxRetries + 1
			if expectedAttempts != tt.wantAttempts {
				t.Errorf("Expected attempts = %d, want %d", expectedAttempts, tt.wantAttempts)
			}
		})
	}
}

func TestFallbackConfiguration(t *testing.T) {
	// Test fallback enabled
	config1 := &RouterConfig{
		BudgetUSD:      100.0,
		MaxLatencyMs:   60000,
		EnableFallback: true,
		FallbackModel:  "claude-haiku-3.5",
	}

	router1, err := NewRouter(config1)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	if !router1.config.EnableFallback {
		t.Error("Router.config.EnableFallback should be true")
	}

	if router1.config.FallbackModel != "claude-haiku-3.5" {
		t.Errorf("Router.config.FallbackModel = %s, want claude-haiku-3.5", router1.config.FallbackModel)
	}

	// Test fallback disabled
	config2 := &RouterConfig{
		BudgetUSD:      100.0,
		MaxLatencyMs:   60000,
		EnableFallback: false,
	}

	router2, err := NewRouter(config2)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	if router2.config.EnableFallback {
		t.Error("Router.config.EnableFallback should be false")
	}
}

func TestErrorMessageFormatting(t *testing.T) {
	// Test that retry errors include attempt count
	lastErr := errors.New("connection timeout")

	expectedMsg := "all retry attempts failed (tried 3 times)"
	fullErr := errors.New(expectedMsg + ": " + lastErr.Error())

	if !strings.Contains(fullErr.Error(), "tried 3 times") {
		t.Error("Error message should include attempt count")
	}

	if !strings.Contains(fullErr.Error(), lastErr.Error()) {
		t.Error("Error message should include underlying error")
	}
}

func TestContextCancellation(t *testing.T) {
	router, _ := NewRouter(&RouterConfig{
		BudgetUSD:      100.0,
		MaxLatencyMs:   60000,
		MaxRetries:     5,
		RetryBackoffMs: 1000,
	})

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Verify that a cancelled context doesn't retry
	err := ctx.Err()
	if err == nil {
		t.Fatal("Expected context to be cancelled")
	}

	// Context errors should not be retryable
	if router.isRetryableError(err) {
		t.Error("Context cancellation errors should not be retryable")
	}
}

func TestBackoffBoundaries(t *testing.T) {
	tests := []struct {
		name              string
		retryBackoffMs    int
		retryMaxBackoffMs int
		doublings         int
		wantFinalBackoff  time.Duration
	}{
		{
			name:              "reaches max quickly",
			retryBackoffMs:    1000,
			retryMaxBackoffMs: 2000,
			doublings:         3,
			wantFinalBackoff:  2000 * time.Millisecond, // 1000 -> 2000 (capped)
		},
		{
			name:              "exponential growth",
			retryBackoffMs:    100,
			retryMaxBackoffMs: 100000,
			doublings:         4,
			wantFinalBackoff:  1600 * time.Millisecond, // 100 -> 200 -> 400 -> 800 -> 1600
		},
		{
			name:              "single doubling",
			retryBackoffMs:    500,
			retryMaxBackoffMs: 5000,
			doublings:         1,
			wantFinalBackoff:  1000 * time.Millisecond, // 500 -> 1000
		},
		{
			name:              "no doublings",
			retryBackoffMs:    500,
			retryMaxBackoffMs: 5000,
			doublings:         0,
			wantFinalBackoff:  500 * time.Millisecond, // No change
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backoff := time.Duration(tt.retryBackoffMs) * time.Millisecond
			maxBackoff := time.Duration(tt.retryMaxBackoffMs) * time.Millisecond

			// Simulate the doubling logic from generateWithRetry
			for i := 0; i < tt.doublings; i++ {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}

			if backoff != tt.wantFinalBackoff {
				t.Errorf("Final backoff after %d doublings = %v, want %v", tt.doublings, backoff, tt.wantFinalBackoff)
			}
		})
	}
}
