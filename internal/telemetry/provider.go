package telemetry

import (
	"context"
)

// InitProvider is currently a noop placeholder. When telemetry is enabled via
// configuration, this hook allows enterprise builds to initialize their own
// exporters (e.g., OTLP) by replacing this implementation.
func InitProvider(context.Context, Config) (func(context.Context) error, error) {
	return func(context.Context) error { return nil }, nil
}

// Shutdown is a noop placeholder.
func Shutdown(ctx context.Context) error {
	return nil
}

// ForceFlush is a noop placeholder.
func ForceFlush(ctx context.Context) error {
	return nil
}
