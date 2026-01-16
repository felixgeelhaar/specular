package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDomainEventLifecycle(t *testing.T) {
	domainEventLog = nil

	RecordProviderDetected("alpha", "ALPHA_ENV", "1.0", true)
	RecordProviderRegistered("alpha", "builtin", "high")
	RecordProviderHealth("alpha", true, "healthy")
	RecordSpecRequested("spec.yaml", "alpha")

	events := Events()
	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}

	// Ensure Events returns a copy of the slice
	events[0] = DomainEvent{Type: EventProviderRegistered}
	if domainEventLog[0].Type != EventProviderDetected {
		t.Fatalf("expected original event to stay ProviderDetected, got %s", domainEventLog[0].Type)
	}

	for _, evt := range events {
		text := FormatEvent(evt)
		if strings.TrimSpace(text) == "" {
			t.Errorf("FormatEvent returned empty string for %+v", evt)
		}
	}

	// Persist only last two events
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "events.log")
	if err := PersistEvents(target, 2); err != nil {
		t.Fatalf("PersistEvents() error = %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed to read persisted events: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 persisted lines, got %d", len(lines))
	}
}
