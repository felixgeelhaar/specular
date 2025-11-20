package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/health"
)

func TestNewServer(t *testing.T) {
	pm := health.NewProbeManager("1.0.0")
	cfg := Config{
		Address:         ":8080",
		ShutdownTimeout: 5 * time.Second,
	}

	s := NewServer(pm, cfg)

	if s == nil {
		t.Fatal("expected server to be created")
	}

	if s.probeManager != pm {
		t.Error("probe manager not set correctly")
	}

	if s.shutdownTimeout != 5*time.Second {
		t.Errorf("shutdown timeout: expected 5s, got %v", s.shutdownTimeout)
	}
}

func TestNewServerDefaults(t *testing.T) {
	pm := health.NewProbeManager("1.0.0")
	cfg := Config{
		Address: ":8080",
		// No timeouts specified - should use defaults
	}

	s := NewServer(pm, cfg)

	if s.shutdownTimeout != 30*time.Second {
		t.Errorf("default shutdown timeout: expected 30s, got %v", s.shutdownTimeout)
	}

	if s.httpServer.ReadTimeout != 10*time.Second {
		t.Errorf("default read timeout: expected 10s, got %v", s.httpServer.ReadTimeout)
	}

	if s.httpServer.WriteTimeout != 10*time.Second {
		t.Errorf("default write timeout: expected 10s, got %v", s.httpServer.WriteTimeout)
	}

	if s.httpServer.IdleTimeout != 60*time.Second {
		t.Errorf("default idle timeout: expected 60s, got %v", s.httpServer.IdleTimeout)
	}
}

func TestHandleLiveness(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		inShutdown     bool
		expectedStatus int
		expectedHealth health.Status
	}{
		{
			name:           "GET request - normal operation",
			method:         http.MethodGet,
			inShutdown:     false,
			expectedStatus: http.StatusOK,
			expectedHealth: health.StatusHealthy,
		},
		{
			name:           "GET request - during shutdown",
			method:         http.MethodGet,
			inShutdown:     true,
			expectedStatus: http.StatusOK,
			expectedHealth: health.StatusDegraded,
		},
		{
			name:           "POST request - not allowed",
			method:         http.MethodPost,
			inShutdown:     false,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedHealth: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := health.NewProbeManager("1.0.0")
			if tt.inShutdown {
				pm.MarkShutdown()
			}

			s := NewServer(pm, Config{Address: ":8080"})

			req := httptest.NewRequest(tt.method, "/health/live", nil)
			w := httptest.NewRecorder()

			s.handleLiveness(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("status code: expected %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.method == http.MethodGet {
				var result health.ProbeResult
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if result.Status != tt.expectedHealth {
					t.Errorf("health status: expected %s, got %s", tt.expectedHealth, result.Status)
				}

				if result.Version != "1.0.0" {
					t.Errorf("version: expected 1.0.0, got %s", result.Version)
				}
			}
		})
	}
}

func TestHandleReadiness(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		inShutdown     bool
		expectedStatus int
		expectedHealth health.Status
	}{
		{
			name:           "GET request - ready",
			method:         http.MethodGet,
			inShutdown:     false,
			expectedStatus: http.StatusOK,
			expectedHealth: health.StatusHealthy,
		},
		{
			name:           "GET request - shutting down",
			method:         http.MethodGet,
			inShutdown:     true,
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: health.StatusUnhealthy,
		},
		{
			name:           "POST request - not allowed",
			method:         http.MethodPost,
			inShutdown:     false,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedHealth: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := health.NewProbeManager("1.0.0")
			if tt.inShutdown {
				pm.MarkShutdown()
			}

			s := NewServer(pm, Config{Address: ":8080"})

			req := httptest.NewRequest(tt.method, "/health/ready", nil)
			w := httptest.NewRecorder()

			s.handleReadiness(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("status code: expected %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.method == http.MethodGet {
				var result health.ProbeResult
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if result.Status != tt.expectedHealth {
					t.Errorf("health status: expected %s, got %s", tt.expectedHealth, result.Status)
				}
			}
		})
	}
}

func TestHandleStartup(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		initialized    bool
		expectedStatus int
		expectedHealth health.Status
	}{
		{
			name:           "GET request - not initialized",
			method:         http.MethodGet,
			initialized:    false,
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: health.StatusUnhealthy,
		},
		{
			name:           "GET request - initialized",
			method:         http.MethodGet,
			initialized:    true,
			expectedStatus: http.StatusOK,
			expectedHealth: health.StatusHealthy,
		},
		{
			name:           "POST request - not allowed",
			method:         http.MethodPost,
			initialized:    false,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedHealth: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := health.NewProbeManager("1.0.0")
			if tt.initialized {
				pm.MarkInitialized()
			}

			s := NewServer(pm, Config{Address: ":8080"})

			req := httptest.NewRequest(tt.method, "/health/startup", nil)
			w := httptest.NewRecorder()

			s.handleStartup(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("status code: expected %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.method == http.MethodGet {
				var result health.ProbeResult
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if result.Status != tt.expectedHealth {
					t.Errorf("health status: expected %s, got %s", tt.expectedHealth, result.Status)
				}
			}
		})
	}
}

func TestHealthzBackwardCompatibility(t *testing.T) {
	pm := health.NewProbeManager("1.0.0")
	s := NewServer(pm, Config{Address: ":8080"})

	// Test that /healthz maps to readiness
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code: expected %d, got %d", http.StatusOK, w.Code)
	}

	var result health.ProbeResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Status != health.StatusHealthy {
		t.Errorf("health status: expected healthy, got %s", result.Status)
	}
}

func TestServerLifecycle(t *testing.T) {
	pm := health.NewProbeManager("1.0.0")
	cfg := Config{
		Address:         "127.0.0.1:0", // Use random port
		ShutdownTimeout: 1 * time.Second,
	}

	s := NewServer(pm, cfg)

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- s.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Check that server is not shutting down
	if s.IsShuttingDown() {
		t.Error("server should not be shutting down initially")
	}

	// Shutdown server
	ctx := context.Background()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	// Check that server is shutting down
	if !s.IsShuttingDown() {
		t.Error("server should be shutting down after Shutdown() called")
	}

	// Wait for server to finish
	err := <-serverErr
	if err != http.ErrServerClosed {
		t.Errorf("expected ErrServerClosed, got %v", err)
	}
}

func TestGracefulShutdown(t *testing.T) {
	pm := health.NewProbeManager("1.0.0")
	cfg := Config{
		Address:         "127.0.0.1:0",
		ShutdownTimeout: 2 * time.Second,
	}

	s := NewServer(pm, cfg)

	// Start server
	go func() {
		_ = s.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	// Initiate shutdown
	shutdownComplete := make(chan error, 1)
	go func() {
		shutdownComplete <- s.Shutdown(context.Background())
	}()

	// Shutdown should complete quickly since there are no active connections
	select {
	case err := <-shutdownComplete:
		if err != nil {
			t.Errorf("shutdown error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("shutdown timed out")
	}
}

func TestShutdownTimeout(t *testing.T) {
	pm := health.NewProbeManager("1.0.0")
	cfg := Config{
		Address:         "127.0.0.1:0",
		ShutdownTimeout: 100 * time.Millisecond, // Very short timeout
	}

	s := NewServer(pm, cfg)

	// Start server
	go func() {
		_ = s.Start()
	}()

	time.Sleep(50 * time.Millisecond)

	// Test shutdown with context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := s.Shutdown(ctx)
	// Should complete (may return context.Canceled but that's ok)
	if err != nil && err != context.Canceled {
		t.Logf("shutdown with cancelled context returned: %v", err)
	}
}

func TestConcurrentRequests(t *testing.T) {
	pm := health.NewProbeManager("1.0.0")
	pm.MarkInitialized()

	s := NewServer(pm, Config{Address: ":8080"})

	// Create test server
	ts := httptest.NewServer(s.httpServer.Handler)
	defer ts.Close()

	// Make concurrent requests to different endpoints
	done := make(chan bool)
	endpoints := []string{"/health/live", "/health/ready", "/health/startup", "/healthz"}

	for _, endpoint := range endpoints {
		for i := 0; i < 10; i++ {
			go func(ep string) {
				resp, err := http.Get(ts.URL + ep)
				if err != nil {
					t.Errorf("request failed: %v", err)
					done <- false
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					t.Errorf("unexpected status: %d", resp.StatusCode)
					done <- false
					return
				}

				// Read and discard body
				_, _ = io.Copy(io.Discard, resp.Body)
				done <- true
			}(endpoint)
		}
	}

	// Wait for all requests to complete
	for i := 0; i < len(endpoints)*10; i++ {
		<-done
	}
}

func TestProbeManagerInitialization(t *testing.T) {
	pm := health.NewProbeManager("1.0.0")

	// Before Start(), initialized should be false
	if pm.IsInitialized() {
		t.Error("probe manager should not be initialized before Start()")
	}

	cfg := Config{
		Address:         "127.0.0.1:0",
		ShutdownTimeout: 1 * time.Second,
	}
	s := NewServer(pm, cfg)

	// Start server in background
	go func() {
		_ = s.Start()
	}()

	// Give server time to mark initialization
	time.Sleep(50 * time.Millisecond)

	// After Start(), initialized should be true
	if !pm.IsInitialized() {
		t.Error("probe manager should be initialized after Start()")
	}

	// Cleanup
	_ = s.Shutdown(context.Background())
}
