// Package server provides HTTP server functionality with health endpoints.
//
// It implements zero-downtime deployments with:
//   - Kubernetes-style health probes (liveness, readiness, startup)
//   - Graceful shutdown with connection draining
//   - Configurable shutdown timeout
//
// This package supports M9.1.1 (Zero-Downtime Deployments) in the v2.0 roadmap.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/felixgeelhaar/specular/internal/health"
)

// Server provides HTTP server functionality with health endpoints.
type Server struct {
	httpServer      *http.Server
	probeManager    *health.ProbeManager
	inShutdown      atomic.Bool
	shutdownTimeout time.Duration
}

// Config holds server configuration.
type Config struct {
	// Address is the listen address (e.g., ":8080", "0.0.0.0:8080")
	Address string

	// ShutdownTimeout is the maximum time to wait for connections to drain during shutdown.
	// Defaults to 30 seconds if not specified.
	ShutdownTimeout time.Duration

	// ReadTimeout is the maximum duration for reading the entire request.
	// Defaults to 10 seconds if not specified.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of the response.
	// Defaults to 10 seconds if not specified.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum amount of time to wait for the next request.
	// Defaults to 60 seconds if not specified.
	IdleTimeout time.Duration
}

// NewServer creates a new HTTP server with health endpoints.
func NewServer(probeManager *health.ProbeManager, cfg Config) *Server {
	// Set defaults
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 30 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 10 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 10 * time.Second
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 60 * time.Second
	}

	s := &Server{
		probeManager:    probeManager,
		shutdownTimeout: cfg.ShutdownTimeout,
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// Register health endpoints
	mux.HandleFunc("/health/live", s.handleLiveness)
	mux.HandleFunc("/health/ready", s.handleReadiness)
	mux.HandleFunc("/health/startup", s.handleStartup)

	// Backward compatibility: /healthz endpoint (maps to readiness)
	mux.HandleFunc("/healthz", s.handleReadiness)

	s.httpServer = &http.Server{
		Addr:         cfg.Address,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return s
}

// Start starts the HTTP server.
// This is a blocking call that returns when the server is stopped or encounters an error.
// Returns http.ErrServerClosed when the server is shut down gracefully.
func (s *Server) Start() error {
	// Mark initialization complete (server is ready to start)
	s.probeManager.MarkInitialized()

	return s.httpServer.ListenAndServe()
}

// Shutdown performs graceful shutdown of the HTTP server.
//
// It:
//  1. Marks the server as shutting down (readiness probes will fail)
//  2. Disables HTTP keep-alives to stop accepting new requests
//  3. Waits for existing connections to drain (up to ShutdownTimeout)
//  4. Forces closure of any remaining connections after timeout
//
// This ensures zero-downtime deployments when used with Kubernetes rolling updates.
func (s *Server) Shutdown(ctx context.Context) error {
	// Mark server as shutting down
	s.inShutdown.Store(true)
	s.probeManager.MarkShutdown()

	// Disable keep-alives to stop accepting new requests on existing connections
	s.httpServer.SetKeepAlivesEnabled(false)

	// Create context with shutdown timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()

	// Gracefully shutdown the server
	return s.httpServer.Shutdown(shutdownCtx)
}

// IsShuttingDown returns whether the server is shutting down.
func (s *Server) IsShuttingDown() bool {
	return s.inShutdown.Load()
}

// writeProbeResponse is a helper function to write probe responses with consistent error handling.
func (s *Server) writeProbeResponse(w http.ResponseWriter, result *health.ProbeResult, unhealthyStatus int) {
	w.Header().Set("Content-Type", "application/json")

	// Determine HTTP status code based on health status
	if result.Status == health.StatusUnhealthy {
		w.WriteHeader(unhealthyStatus)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// handleLiveness handles liveness probe requests.
// GET /health/live
//
// Liveness probes determine if the application is alive and responsive.
// If this check fails, Kubernetes will restart the container.
//
// Returns:
//   - 200 OK with JSON: Application is running normally
//   - 200 OK with JSON (degraded status): Application is shutting down but still alive
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	result := s.probeManager.CheckLiveness(ctx)

	// Liveness should always return 200 (even during shutdown)
	s.writeProbeResponse(w, result, http.StatusOK)
}

// handleReadiness handles readiness probe requests.
// GET /health/ready
//
// Readiness probes determine if the application is ready to accept traffic.
// If this check fails, Kubernetes removes the pod from service endpoints.
//
// Returns:
//   - 200 OK with JSON: Application is ready to serve requests
//   - 503 Service Unavailable with JSON: Application is not ready (shutting down or dependencies unhealthy)
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	result := s.probeManager.CheckReadiness(ctx)

	// Return 503 if not ready (shutting down or dependencies unhealthy)
	s.writeProbeResponse(w, result, http.StatusServiceUnavailable)
}

// handleStartup handles startup probe requests.
// GET /health/startup
//
// Startup probes determine if the application has finished initialization.
// Kubernetes waits for this probe to pass before checking liveness/readiness.
// This allows slow-starting applications more time to initialize.
//
// Returns:
//   - 200 OK with JSON: Application has finished initialization
//   - 503 Service Unavailable with JSON: Application is still starting up
func (s *Server) handleStartup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	result := s.probeManager.CheckStartup(ctx)

	// Return 503 if not yet initialized
	s.writeProbeResponse(w, result, http.StatusServiceUnavailable)
}
