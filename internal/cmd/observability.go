package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/log"
	"github.com/felixgeelhaar/specular/internal/metrics"
	"github.com/felixgeelhaar/specular/internal/telemetry"
	"github.com/felixgeelhaar/specular/internal/version"
)

// setupObservability configures logging, metrics, and optional telemetry.
// It returns a cleanup function that should be deferred by the caller.
func setupObservability(ctx context.Context) func() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Unable to load config for observability: %v\n", err)
		cfg = defaultGlobalConfig()
	}

	logCleanup := setupLogging(cfg)
	metrics.InitDefault()
	telemetryCleanup := setupTelemetry(ctx, cfg)

	return func() {
		telemetryCleanup()
		logCleanup()
	}
}

func setupLogging(cfg *GlobalConfig) func() {
	info := version.GetInfo()

	level := getLogLevel(cfg)
	format := getLogFormat()
	loggerOutput, _, fileCleanup := configureLogOutput(cfg)

	logger := log.New(log.Config{
		Level:          log.ParseLevel(level),
		Format:         log.ParseFormat(format),
		Output:         loggerOutput,
		AddSource:      false,
		ServiceName:    "specular",
		ServiceVersion: info.Version,
	})

	log.SetDefaultLogger(logger)

	// Note: We intentionally don't log initialization to stdout
	// as it clutters the user experience. File logging captures this if enabled.

	return fileCleanup
}

func setupTelemetry(ctx context.Context, cfg *GlobalConfig) func() {
	if !telemetryRequested(cfg) {
		return func() {}
	}

	info := version.GetInfo()
	telemCfg := telemetry.Config{
		ServiceName:    "specular",
		ServiceVersion: info.Version,
		Environment:    telemetryEnvironment(),
		Enabled:        true,
		Endpoint:       telemetryEndpoint(cfg),
		SampleRate:     telemetrySampleRate(cfg),
	}

	shutdown, err := telemetry.InitProvider(ctx, telemCfg)
	if err != nil {
		log.DefaultLogger().Warn("Failed to initialize telemetry", "error", err)
		return func() {}
	}

	log.DefaultLogger().Info("Telemetry enabled",
		"endpoint", telemCfg.Endpoint,
		"sample_rate", telemCfg.SampleRate,
	)

	return func() {
		if shutdown == nil {
			return
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := shutdown(shutdownCtx); err != nil {
			log.DefaultLogger().Warn("Failed to flush telemetry", "error", err)
		}
	}
}

func telemetryRequested(cfg *GlobalConfig) bool {
	if val := strings.ToLower(os.Getenv("SPECULAR_TELEMETRY")); val != "" {
		return val == "on" || val == "true" || val == "1" || val == "enabled"
	}

	if cfg == nil {
		return false
	}

	return cfg.Telemetry.Enabled
}

func telemetryEndpoint(cfg *GlobalConfig) string {
	if env := os.Getenv("SPECULAR_TELEMETRY_ENDPOINT"); env != "" {
		return env
	}
	if cfg != nil {
		return cfg.Telemetry.Endpoint
	}
	return ""
}

func telemetrySampleRate(cfg *GlobalConfig) float64 {
	if env := os.Getenv("SPECULAR_TELEMETRY_SAMPLE_RATE"); env != "" {
		if v, err := strconv.ParseFloat(env, 64); err == nil {
			return clampSampleRate(v)
		}
	}
	if cfg != nil && cfg.Telemetry.SampleRate > 0 {
		return clampSampleRate(cfg.Telemetry.SampleRate)
	}
	return 1.0
}

func telemetryEnvironment() string {
	if env := os.Getenv("SPECULAR_ENV"); env != "" {
		return env
	}
	return "cli"
}

func clampSampleRate(value float64) float64 {
	switch {
	case value <= 0:
		return 0.0
	case value >= 1:
		return 1.0
	default:
		return value
	}
}

func getLogLevel(cfg *GlobalConfig) string {
	if env := os.Getenv("SPECULAR_LOG_LEVEL"); env != "" {
		return env
	}
	if cfg != nil && cfg.Logging.Level != "" {
		return cfg.Logging.Level
	}
	return "info"
}

func getLogFormat() string {
	if env := os.Getenv("SPECULAR_LOG_FORMAT"); env != "" {
		return env
	}
	return "json"
}

func configureLogOutput(cfg *GlobalConfig) (log.Output, string, func()) {
	var writers []io.Writer

	// Only write to stdout if explicitly requested via environment variable
	// This keeps the CLI output clean by default
	if os.Getenv("SPECULAR_LOG_STDOUT") == "true" {
		writers = append(writers, os.Stdout)
	}

	var file *os.File
	var filePath string
	if cfg != nil && cfg.Logging.EnableFile {
		dir := cfg.Logging.LogDir
		if dir == "" {
			dir = defaultLogDir()
		}
		dir = expandPath(dir)
		if err := os.MkdirAll(dir, 0o750); err == nil {
			path := filepath.Join(dir, "specular.log")
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
			if err == nil {
				file = f
				filePath = path
				writers = append(writers, f)
			}
		}
	}

	// If no writers configured (no stdout, no file), default to discard
	if len(writers) == 0 {
		writers = append(writers, io.Discard)
	}

	output := log.NewOutput(io.MultiWriter(writers...))
	cleanup := func() {
		if file != nil {
			_ = file.Close()
		}
	}
	return output, filePath, cleanup
}

func defaultLogDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./.specular/logs"
	}
	return filepath.Join(home, ".specular", "logs")
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}
