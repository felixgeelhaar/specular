package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/health"
	"github.com/felixgeelhaar/specular/internal/server"
	"github.com/felixgeelhaar/specular/internal/version"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server with health endpoints",
	Long: `Start an HTTP server with Kubernetes-style health endpoints for
zero-downtime deployments.

The server provides three health probe endpoints:
  /health/live    - Liveness probe (process alive and responsive)
  /health/ready   - Readiness probe (ready to accept traffic)
  /health/startup - Startup probe (finished initialization)
  /healthz        - Backward-compatible readiness endpoint

The server implements graceful shutdown with connection draining when
it receives SIGTERM or SIGINT signals. This ensures zero-downtime
deployments when used with Kubernetes rolling updates.

Example:
  # Start server on default port 8080
  specular serve

  # Start server on custom port
  specular serve --port 9090

  # Start server with custom shutdown timeout
  specular serve --shutdown-timeout 60s`,
	RunE: runServe,
}

var (
	servePort            string
	serveAddress         string
	serveShutdownTimeout time.Duration
	serveReadTimeout     time.Duration
	serveWriteTimeout    time.Duration
	serveIdleTimeout     time.Duration
)

func init() {
	serveCmd.Flags().StringVar(&servePort, "port", "8080", "Port to listen on")
	serveCmd.Flags().StringVar(&serveAddress, "address", "0.0.0.0", "Address to bind to")
	serveCmd.Flags().DurationVar(&serveShutdownTimeout, "shutdown-timeout", 30*time.Second, "Maximum time to wait for connections to drain during shutdown")
	serveCmd.Flags().DurationVar(&serveReadTimeout, "read-timeout", 10*time.Second, "Maximum duration for reading the entire request")
	serveCmd.Flags().DurationVar(&serveWriteTimeout, "write-timeout", 10*time.Second, "Maximum duration before timing out writes of the response")
	serveCmd.Flags().DurationVar(&serveIdleTimeout, "idle-timeout", 60*time.Second, "Maximum amount of time to wait for the next request")

	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Get version info
	info := version.GetInfo()

	// Create probe manager
	pm := health.NewProbeManager(info.Version)

	// Register health checkers
	// These check dependencies like Docker, Git, providers
	pm.AddChecker(health.NewDockerChecker())
	pm.AddChecker(health.NewGitChecker())

	// Create server
	listenAddr := fmt.Sprintf("%s:%s", serveAddress, servePort)
	srv := server.NewServer(pm, server.Config{
		Address:         listenAddr,
		ShutdownTimeout: serveShutdownTimeout,
		ReadTimeout:     serveReadTimeout,
		WriteTimeout:    serveWriteTimeout,
		IdleTimeout:     serveIdleTimeout,
	})

	// Print startup message
	fmt.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                      [ specular ]                            ║\n")
	fmt.Printf("║            AI-Native Spec and Build Assistant                ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")
	fmt.Printf("Version: %s\n", info.Version)
	fmt.Printf("Listening on: http://%s\n\n", listenAddr)
	fmt.Printf("Health Endpoints:\n")
	fmt.Printf("  Liveness:  http://%s/health/live\n", listenAddr)
	fmt.Printf("  Readiness: http://%s/health/ready\n", listenAddr)
	fmt.Printf("  Startup:   http://%s/health/startup\n", listenAddr)
	fmt.Printf("  Legacy:    http://%s/healthz\n\n", listenAddr)
	fmt.Printf("Press Ctrl+C to stop the server\n\n")

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		// Server failed to start or encountered an error
		return fmt.Errorf("server error: %w", err)

	case sig := <-sigChan:
		// Received shutdown signal
		fmt.Printf("\nReceived signal: %s\n", sig)
		fmt.Println("Initiating graceful shutdown...")

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(ctx, serveShutdownTimeout+5*time.Second)
		defer cancel()

		// Shutdown server
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}

		fmt.Println("Server stopped gracefully")
		return nil
	}
}
