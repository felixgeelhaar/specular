package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProviderStatus represents the runtime state of a provider aggregate.
type ProviderStatus struct {
	Enabled       bool
	Available     bool
	Healthy       bool
	LastError     string
	VisibleReason string
	LastUpdated   time.Time
}

// ProviderAggregate encapsulates descriptor metadata, configuration, and runtime status.
type ProviderAggregate struct {
	Descriptor ProviderDescriptor
	Config     ProviderConfig
	Status     ProviderStatus
}

// NewProviderAggregate builds an aggregate from a descriptor.
func NewProviderAggregate(desc ProviderDescriptor) *ProviderAggregate {
	config := desc.ToProviderConfig()
	status := ProviderStatus{
		Enabled:       config.Enabled,
		Available:     false,
		Healthy:       false,
		VisibleReason: fmt.Sprintf("Descriptor trust=%s source=%s", desc.TrustLevel, desc.Source),
		LastUpdated:   time.Now(),
	}

	return &ProviderAggregate{
		Descriptor: desc,
		Config:     config,
		Status:     status,
	}
}

// DomainEventType enumerates provider-related events.
type DomainEventType string

const (
	EventProviderDetected        DomainEventType = "ProviderDetected"
	EventProviderRegistered      DomainEventType = "ProviderRegistered"
	EventProviderEnabled         DomainEventType = "ProviderEnabled"
	EventProviderHealthChanged   DomainEventType = "ProviderHealthChanged"
	EventSpecGenerationRequested DomainEventType = "SpecGenerationRequested"
)

// DomainEvent captures metadata for auditing/tracing.
type DomainEvent struct {
	Type      DomainEventType
	Timestamp time.Time
	Payload   map[string]interface{}
}

var domainEventLog []DomainEvent

// RecordEvent logs an event for later inspection.
func RecordEvent(evt DomainEvent) {
	evt.Timestamp = time.Now()
	domainEventLog = append(domainEventLog, evt)
}

// Events returns all recorded domain events.
func Events() []DomainEvent {
	return append([]DomainEvent(nil), domainEventLog...)
}

// NewEvent creates a DomainEvent with the given type and payload.
func NewEvent(eventType DomainEventType, payload map[string]interface{}) DomainEvent {
	return DomainEvent{
		Type:    eventType,
		Payload: payload,
	}
}

// RecordProviderDetected logs provider detection details.
func RecordProviderDetected(name, envVar, version string, available bool) {
	payload := map[string]interface{}{
		"name":      name,
		"available": available,
		"env_var":   envVar,
		"version":   version,
	}
	RecordEvent(NewEvent(EventProviderDetected, payload))
}

// RecordProviderRegistered logs when a provider is registered.
func RecordProviderRegistered(name, source, trust string) {
	payload := map[string]interface{}{
		"name":   name,
		"source": source,
		"trust":  trust,
	}
	RecordEvent(NewEvent(EventProviderRegistered, payload))
}

// RecordProviderHealth logs provider health results.
func RecordProviderHealth(name string, healthy bool, reason string) {
	payload := map[string]interface{}{
		"name":    name,
		"healthy": healthy,
		"reason":  reason,
	}
	RecordEvent(NewEvent(EventProviderHealthChanged, payload))
}

// RecordSpecRequested logs that a spec generation was requested.
func RecordSpecRequested(prdFile, providerHint string) {
	payload := map[string]interface{}{
		"prd_file":      prdFile,
		"provider_hint": providerHint,
	}
	RecordEvent(NewEvent(EventSpecGenerationRequested, payload))
}

// FormatEvent returns a human-readable string for a domain event.
func FormatEvent(evt DomainEvent) string {
	timestamp := evt.Timestamp.Format("15:04:05")

	switch evt.Type {
	case EventProviderDetected:
		name, _ := evt.Payload["name"].(string)
		available, _ := evt.Payload["available"].(bool)
		envVar, _ := evt.Payload["env_var"].(string)
		version, _ := evt.Payload["version"].(string)
		return fmt.Sprintf("%s Detected %s (available=%t, env=%s, version=%s)", timestamp, name, available, envVar, version)
	case EventProviderRegistered:
		name, _ := evt.Payload["name"].(string)
		source, _ := evt.Payload["source"].(string)
		trust, _ := evt.Payload["trust"].(string)
		return fmt.Sprintf("%s Registered %s (source=%s, trust=%s)", timestamp, name, source, trust)
	case EventProviderHealthChanged:
		name, _ := evt.Payload["name"].(string)
		healthy, _ := evt.Payload["healthy"].(bool)
		reason, _ := evt.Payload["reason"].(string)
		return fmt.Sprintf("%s Health %s: healthy=%t (%s)", timestamp, name, healthy, reason)
	case EventSpecGenerationRequested:
		prdFile, _ := evt.Payload["prd_file"].(string)
		providerHint, _ := evt.Payload["provider_hint"].(string)
		return fmt.Sprintf("%s Spec requested for %s (providers=%s)", timestamp, prdFile, providerHint)
	default:
		return fmt.Sprintf("%s %s %v", timestamp, evt.Type, evt.Payload)
	}
}

// PersistEvents writes the most recent domain events to a file.
func PersistEvents(path string, maxEvents int) error {
	events := Events()
	if len(events) == 0 {
		return nil
	}

	if maxEvents > 0 && len(events) > maxEvents {
		events = events[len(events)-maxEvents:]
	}

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("create telemetry directory: %w", err)
	}

	var builder strings.Builder
	for _, evt := range events {
		builder.WriteString(FormatEvent(evt))
		builder.WriteByte('\n')
	}

	if err := os.WriteFile(path, []byte(builder.String()), 0600); err != nil {
		return fmt.Errorf("write telemetry log: %w", err)
	}

	return nil
}
