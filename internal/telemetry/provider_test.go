package telemetry

import (
	"context"
	"testing"
)

func TestInitProviderDisabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false

	ctx := context.Background()
	shutdown, err := InitProvider(ctx, config)
	if err != nil {
		t.Fatalf("InitProvider failed: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected shutdown function, got nil")
	}

	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown returned error: %v", err)
	}
}

func TestInitProviderEnabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.Endpoint = "https://collector.example.com"
	config.SampleRate = 0.5

	ctx := context.Background()
	shutdown, err := InitProvider(ctx, config)
	if err != nil {
		t.Fatalf("InitProvider failed: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected shutdown function, got nil")
	}
}

func TestShutdownForceFlush(t *testing.T) {
	ctx := context.Background()
	if err := Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
	if err := ForceFlush(ctx); err != nil {
		t.Fatalf("ForceFlush failed: %v", err)
	}
}
