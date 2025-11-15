package telemetry

// Config holds configuration for the tracer
type Config struct {
	// ServiceName is the name of the service
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string

	// Environment is the deployment environment (dev, staging, production)
	Environment string

	// Enabled determines whether tracing is enabled
	// When false, a noop tracer is used
	Enabled bool

	// Endpoint is the OTLP collector endpoint (optional)
	// If empty, traces are not exported
	Endpoint string

	// SampleRate is the fraction of traces to sample (0.0 to 1.0)
	// 1.0 means all traces are sampled
	SampleRate float64
}

// DefaultConfig returns a sensible default configuration
// Tracing disabled by default for CLI tool
func DefaultConfig() Config {
	return Config{
		ServiceName:    "specular",
		ServiceVersion: "dev",
		Environment:    "development",
		Enabled:        false,
		Endpoint:       "",
		SampleRate:     1.0,
	}
}

// DevelopmentConfig returns a configuration suitable for development
// Tracing enabled but not exported
func DevelopmentConfig() Config {
	return Config{
		ServiceName:    "specular",
		ServiceVersion: "dev",
		Environment:    "development",
		Enabled:        true,
		Endpoint:       "",
		SampleRate:     1.0,
	}
}

// ProductionConfig returns a configuration suitable for production
// Tracing enabled with sampling
func ProductionConfig(endpoint string) Config {
	return Config{
		ServiceName:    "specular",
		ServiceVersion: "unknown",
		Environment:    "production",
		Enabled:        true,
		Endpoint:       endpoint,
		SampleRate:     0.1, // Sample 10% of traces in production
	}
}
