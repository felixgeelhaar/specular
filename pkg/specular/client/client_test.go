package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

func TestNew(t *testing.T) {
	client := New("https://api.example.com", "test-key")

	if client.baseURL != "https://api.example.com" {
		t.Errorf("Expected baseURL 'https://api.example.com', got %s", client.baseURL)
	}

	if client.apiKey != "test-key" {
		t.Errorf("Expected apiKey 'test-key', got %s", client.apiKey)
	}

	if client.maxRetries != 3 {
		t.Errorf("Expected maxRetries 3, got %d", client.maxRetries)
	}

	if client.retryDelay != time.Second {
		t.Errorf("Expected retryDelay 1s, got %v", client.retryDelay)
	}

	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
}

func TestNewWithConfig(t *testing.T) {
	cfg := &Config{
		MaxRetries: 5,
		RetryDelay: 2 * time.Second,
		Timeout:    60 * time.Second,
	}

	client := NewWithConfig("https://api.example.com", "test-key", cfg)

	if client.maxRetries != 5 {
		t.Errorf("Expected maxRetries 5, got %d", client.maxRetries)
	}

	if client.retryDelay != 2*time.Second {
		t.Errorf("Expected retryDelay 2s, got %v", client.retryDelay)
	}

	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", client.httpClient.Timeout)
	}
}

func TestNewWithConfig_NilConfig(t *testing.T) {
	client := NewWithConfig("https://api.example.com", "test-key", nil)

	// Should use defaults
	if client.maxRetries != 3 {
		t.Errorf("Expected default maxRetries 3, got %d", client.maxRetries)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", cfg.MaxRetries)
	}

	if cfg.RetryDelay != time.Second {
		t.Errorf("Expected RetryDelay 1s, got %v", cfg.RetryDelay)
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout 30s, got %v", cfg.Timeout)
	}
}

func TestAPIError_Error(t *testing.T) {
	t.Run("with request ID", func(t *testing.T) {
		err := &APIError{
			StatusCode: 500,
			Message:    "Internal server error",
			RequestID:  "req-123",
		}

		expected := "platform API error (status 500, request_id req-123): Internal server error"
		if err.Error() != expected {
			t.Errorf("Expected error message %q, got %q", expected, err.Error())
		}
	})

	t.Run("without request ID", func(t *testing.T) {
		err := &APIError{
			StatusCode: 404,
			Message:    "Not found",
		}

		expected := "platform API error (status 404): Not found"
		if err.Error() != expected {
			t.Errorf("Expected error message %q, got %q", expected, err.Error())
		}
	})
}

func TestHealth_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/health" {
			t.Errorf("Expected path /health, got %s", r.URL.Path)
		}

		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("Expected X-API-Key header 'test-key', got %s", r.Header.Get("X-API-Key"))
		}

		// Return healthy response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{
			Status:  "ok",
			Version: "1.0.0",
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	err := client.Health(context.Background())

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestHealth_Unhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{
			Status: "degraded",
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	err := client.Health(context.Background())

	if err == nil {
		t.Error("Expected error for unhealthy status, got nil")
	}

	if !strings.Contains(err.Error(), "unhealthy status") {
		t.Errorf("Expected 'unhealthy status' error, got %v", err)
	}
}

func TestHealth_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", "req-456")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Database connection failed",
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	err := client.Health(context.Background())

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// The error is wrapped by Health method, so we need to check the message
	if !strings.Contains(err.Error(), "health check failed") {
		t.Errorf("Expected error to contain 'health check failed', got %v", err)
	}

	if !strings.Contains(err.Error(), "Database connection failed") {
		t.Errorf("Expected error message to contain 'Database connection failed', got %v", err)
	}

	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("Expected error to contain 'status 500', got %v", err)
	}
}

func TestGenerateSpec_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/v1/spec/generate" {
			t.Errorf("Expected path /v1/spec/generate, got %s", r.URL.Path)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse request body
		var req GenerateSpecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Prompt != "Build a todo app" {
			t.Errorf("Expected prompt 'Build a todo app', got %s", req.Prompt)
		}

		// Return spec
		spec := &types.ProductSpec{
			Product: "Todo App",
			Goals:   []string{"Manage tasks efficiently"},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(spec)
	}))
	defer server.Close()

	client := New(server.URL, "test-key")

	req := &GenerateSpecRequest{
		Prompt: "Build a todo app",
	}

	spec, err := client.GenerateSpec(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if spec.Product != "Todo App" {
		t.Errorf("Expected product 'Todo App', got %s", spec.Product)
	}

	if len(spec.Goals) != 1 || spec.Goals[0] != "Manage tasks efficiently" {
		t.Errorf("Expected goals ['Manage tasks efficiently'], got %v", spec.Goals)
	}
}

func TestGeneratePlan_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/v1/plan/generate" {
			t.Errorf("Expected path /v1/plan/generate, got %s", r.URL.Path)
		}

		// Parse request body
		var req GeneratePlanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Spec.Product != "Todo App" {
			t.Errorf("Expected spec product 'Todo App', got %s", req.Spec.Product)
		}

		// Return plan
		plan := &types.Plan{
			Tasks: []types.Task{
				{
					ID:       "task-1",
					Skill:    "go-backend",
					Priority: "P0",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(plan)
	}))
	defer server.Close()

	client := New(server.URL, "test-key")

	req := &GeneratePlanRequest{
		Spec: &types.ProductSpec{
			Product: "Todo App",
		},
	}

	plan, err := client.GeneratePlan(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(plan.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(plan.Tasks))
	}

	if plan.Tasks[0].ID != "task-1" {
		t.Errorf("Expected task ID 'task-1', got %s", plan.Tasks[0].ID)
	}
}

func TestRetryLogic_Success(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		// Fail first 2 attempts, succeed on 3rd
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Service temporarily unavailable",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	}))
	defer server.Close()

	cfg := &Config{
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond, // Fast retries for testing
		Timeout:    5 * time.Second,
	}

	client := NewWithConfig(server.URL, "test-key", cfg)
	err := client.Health(context.Background())

	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryLogic_ExhaustedRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Service unavailable",
		})
	}))
	defer server.Close()

	cfg := &Config{
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
		Timeout:    5 * time.Second,
	}

	client := NewWithConfig(server.URL, "test-key", cfg)
	err := client.Health(context.Background())

	if err == nil {
		t.Error("Expected error after exhausted retries, got nil")
	}

	if !strings.Contains(err.Error(), "max retries exceeded") {
		t.Errorf("Expected 'max retries exceeded' error, got %v", err)
	}

	// Should attempt: initial + 2 retries = 3 total
	if attempts != 3 {
		t.Errorf("Expected 3 attempts (1 initial + 2 retries), got %d", attempts)
	}
}

func TestRetryLogic_NoRetryOn4xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request",
		})
	}))
	defer server.Close()

	cfg := &Config{
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
		Timeout:    5 * time.Second,
	}

	client := NewWithConfig(server.URL, "test-key", cfg)
	err := client.Health(context.Background())

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Check error contains the 400 status and error message
	if !strings.Contains(err.Error(), "status 400") {
		t.Errorf("Expected error to contain 'status 400', got %v", err)
	}

	if !strings.Contains(err.Error(), "Invalid request") {
		t.Errorf("Expected error to contain 'Invalid request', got %v", err)
	}

	// Should not retry on 4xx errors
	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retries on 4xx), got %d", attempts)
	}
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := client.Health(ctx)

	if err == nil {
		t.Error("Expected context deadline error, got nil")
	}

	// The error is wrapped, so check if it contains context deadline exceeded
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected error to contain 'context deadline exceeded', got %v", err)
	}
}
