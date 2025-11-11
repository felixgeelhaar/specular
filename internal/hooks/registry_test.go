package hooks

import (
	"context"
	"testing"
)

// MockHook for testing
type MockHook struct {
	name       string
	eventTypes []EventType
	enabled    bool
	executed   bool
	shouldFail bool
}

func (m *MockHook) Name() string            { return m.name }
func (m *MockHook) EventTypes() []EventType { return m.eventTypes }
func (m *MockHook) Enabled() bool           { return m.enabled }
func (m *MockHook) Execute(ctx context.Context, event *Event) error {
	m.executed = true
	if m.shouldFail {
		return context.DeadlineExceeded
	}
	return nil
}

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	hook := &MockHook{
		name:       "test-hook",
		eventTypes: []EventType{EventWorkflowStart, EventWorkflowComplete},
		enabled:    true,
	}

	err := registry.Register(hook)
	if err != nil {
		t.Errorf("Register failed: %v", err)
	}

	// Check hook was registered for both events
	if !registry.HasHooksFor(EventWorkflowStart) {
		t.Error("Hook not registered for EventWorkflowStart")
	}
	if !registry.HasHooksFor(EventWorkflowComplete) {
		t.Error("Hook not registered for EventWorkflowComplete")
	}

	// Check count
	if registry.Count() != 1 {
		t.Errorf("Count mismatch: got %d, want 1", registry.Count())
	}
}

func TestRegistryRegisterDisabled(t *testing.T) {
	registry := NewRegistry()

	hook := &MockHook{
		name:       "disabled-hook",
		eventTypes: []EventType{EventWorkflowStart},
		enabled:    false,
	}

	err := registry.Register(hook)
	if err != nil {
		t.Errorf("Register failed: %v", err)
	}

	// Disabled hooks should not be registered
	if registry.HasHooksFor(EventWorkflowStart) {
		t.Error("Disabled hook should not be registered")
	}

	if registry.Count() != 0 {
		t.Errorf("Count mismatch: got %d, want 0", registry.Count())
	}
}

func TestRegistryRegisterNil(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register(nil)
	if err == nil {
		t.Error("Expected error when registering nil hook")
	}
}

func TestRegistryUnregister(t *testing.T) {
	registry := NewRegistry()

	hook1 := &MockHook{
		name:       "hook-1",
		eventTypes: []EventType{EventWorkflowStart},
		enabled:    true,
	}
	hook2 := &MockHook{
		name:       "hook-2",
		eventTypes: []EventType{EventWorkflowStart},
		enabled:    true,
	}

	registry.Register(hook1)
	registry.Register(hook2)

	if registry.Count() != 2 {
		t.Errorf("Count mismatch: got %d, want 2", registry.Count())
	}

	// Unregister hook1
	registry.Unregister("hook-1")

	if registry.Count() != 1 {
		t.Errorf("Count after unregister: got %d, want 1", registry.Count())
	}

	// Hook2 should still be registered
	hooks := registry.GetHooks(EventWorkflowStart)
	if len(hooks) != 1 {
		t.Errorf("Hooks count: got %d, want 1", len(hooks))
	}
	if hooks[0].Name() != "hook-2" {
		t.Errorf("Wrong hook remained: got %s, want hook-2", hooks[0].Name())
	}
}

func TestRegistryTrigger(t *testing.T) {
	registry := NewRegistry()

	hook := &MockHook{
		name:       "test-hook",
		eventTypes: []EventType{EventWorkflowStart},
		enabled:    true,
	}

	registry.Register(hook)

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)
	results := registry.Trigger(context.Background(), event)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if !hook.executed {
		t.Error("Hook was not executed")
	}

	if !results[0].Success {
		t.Error("Hook execution should have succeeded")
	}
}

func TestRegistryTriggerNoHooks(t *testing.T) {
	registry := NewRegistry()

	event := NewEvent(EventWorkflowStart, "test-workflow", nil)
	results := registry.Trigger(context.Background(), event)

	if results != nil {
		t.Errorf("Expected nil results for event with no hooks, got %d results", len(results))
	}
}

func TestRegistryGetHooks(t *testing.T) {
	registry := NewRegistry()

	hook1 := &MockHook{
		name:       "hook-1",
		eventTypes: []EventType{EventWorkflowStart},
		enabled:    true,
	}
	hook2 := &MockHook{
		name:       "hook-2",
		eventTypes: []EventType{EventWorkflowStart, EventStepBefore},
		enabled:    true,
	}

	registry.Register(hook1)
	registry.Register(hook2)

	// Check EventWorkflowStart has 2 hooks
	hooks := registry.GetHooks(EventWorkflowStart)
	if len(hooks) != 2 {
		t.Errorf("EventWorkflowStart hooks: got %d, want 2", len(hooks))
	}

	// Check EventStepBefore has 1 hook
	hooks = registry.GetHooks(EventStepBefore)
	if len(hooks) != 1 {
		t.Errorf("EventStepBefore hooks: got %d, want 1", len(hooks))
	}
	if hooks[0].Name() != "hook-2" {
		t.Errorf("Wrong hook: got %s, want hook-2", hooks[0].Name())
	}
}

func TestRegistryGetAllHooks(t *testing.T) {
	registry := NewRegistry()

	hook := &MockHook{
		name:       "test-hook",
		eventTypes: []EventType{EventWorkflowStart, EventWorkflowComplete},
		enabled:    true,
	}

	registry.Register(hook)

	allHooks := registry.GetAllHooks()

	if len(allHooks) != 2 {
		t.Errorf("Expected 2 event types, got %d", len(allHooks))
	}

	if len(allHooks[EventWorkflowStart]) != 1 {
		t.Errorf("EventWorkflowStart hooks: got %d, want 1", len(allHooks[EventWorkflowStart]))
	}

	if len(allHooks[EventWorkflowComplete]) != 1 {
		t.Errorf("EventWorkflowComplete hooks: got %d, want 1", len(allHooks[EventWorkflowComplete]))
	}
}

func TestRegistryClear(t *testing.T) {
	registry := NewRegistry()

	hook := &MockHook{
		name:       "test-hook",
		eventTypes: []EventType{EventWorkflowStart},
		enabled:    true,
	}

	registry.Register(hook)

	if registry.Count() != 1 {
		t.Errorf("Count before clear: got %d, want 1", registry.Count())
	}

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Count after clear: got %d, want 0", registry.Count())
	}

	if registry.HasHooksFor(EventWorkflowStart) {
		t.Error("Registry should have no hooks after clear")
	}
}

func TestRegistryRegisterFromConfig(t *testing.T) {
	registry := NewRegistry()

	// Register a factory
	factory := func(config *HookConfig) (Hook, error) {
		return &MockHook{
			name:       config.Name,
			eventTypes: config.Events,
			enabled:    config.Enabled,
		}, nil
	}
	registry.RegisterFactory("mock", factory)

	config := &HookConfig{
		Name:    "test-hook",
		Type:    "mock",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
	}

	err := registry.RegisterFromConfig(config)
	if err != nil {
		t.Errorf("RegisterFromConfig failed: %v", err)
	}

	if !registry.HasHooksFor(EventWorkflowStart) {
		t.Error("Hook not registered from config")
	}
}

func TestRegistryRegisterFromConfigUnknownType(t *testing.T) {
	registry := NewRegistry()

	config := &HookConfig{
		Name:    "test-hook",
		Type:    "unknown",
		Events:  []EventType{EventWorkflowStart},
		Enabled: true,
	}

	err := registry.RegisterFromConfig(config)
	if err == nil {
		t.Error("Expected error for unknown hook type")
	}
}

func TestRegistryRegisterFromConfigDisabled(t *testing.T) {
	registry := NewRegistry()

	factory := func(config *HookConfig) (Hook, error) {
		return &MockHook{
			name:       config.Name,
			eventTypes: config.Events,
			enabled:    config.Enabled,
		}, nil
	}
	registry.RegisterFactory("mock", factory)

	config := &HookConfig{
		Name:    "test-hook",
		Type:    "mock",
		Events:  []EventType{EventWorkflowStart},
		Enabled: false,
	}

	err := registry.RegisterFromConfig(config)
	if err != nil {
		t.Errorf("RegisterFromConfig failed: %v", err)
	}

	// Disabled hooks should not be registered
	if registry.HasHooksFor(EventWorkflowStart) {
		t.Error("Disabled hook should not be registered")
	}
}
